package separacionportales

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"vec-diputacion-granada/config"
)

// FicheroMarcaPortal declara, dentro del directorio de material, a qué
// portal pertenece. Un proceso separado exige que coincida con el suyo y un
// proceso combinado rechaza un material que ya se haya separado.
const FicheroMarcaPortal = "portal-proceso.json"

const (
	tamanoMaximoMarca         = 1 << 10
	tamanoMaximoEntornoDecl   = 64 << 10
	entradasMaximasMaterial   = 4096
	profundidadMaximaMaterial = 8
)

type pertenencia int

const (
	pertenenciaInterna pertenencia = iota
	pertenenciaExterna
	// pertenenciaComun: cada proceso tiene su propia copia, con contenido
	// distinto (claves, TLS, idempotencia). Lo comprueba ComprobarSeparacion.
	pertenenciaComun
	// pertenenciaProhibida: ningún proceso separado la necesita. Son claves
	// privadas de la autoridad certificadora o de clientes, con las que se
	// podrían emitir o suplantar certificados de acceso.
	pertenenciaProhibida
)

var ficherosExternos = map[string]struct{}{
	config.DevelopmentExternalMailSeedRelativePath: {},
	"identidad/bolsa-candidato.json":               {},
	"identidad/candidato.json":                     {},
	"identidad/usuarios-preferencias-externa.json": {},
	"mtls/candidato.crt":                           {},
}

var ficherosComunes = map[string]struct{}{
	FicheroMarcaPortal: {},
	"manifiesto.json":  {},
	"desarrollo.env":   {},
	"ca/ca.crt":        {},
	"tls/servidor.crt": {},
	"tls/servidor.key": {},
}

// El material KMS y TSA nominal pertenece exclusivamente al interno.
var ficherosInternos = map[string]struct{}{
	"kms/clave-maestra.bin":        {},
	"kms/atestacion-ed25519.key":   {},
	"kms/atestacion-ed25519.pub":   {},
	"kms/revalidacion-ed25519.key": {},
	"kms/revalidacion-ed25519.pub": {},
	"tsa/clave-hmac.bin":           {},
}

var directoriosComunes = []string{"idempotencia/"}

// clasificar asigna una ruta relativa (con barras) a su portal. Lo que no está
// en ninguna lista es interno: el portal externo trabaja con lista positiva.
func clasificar(relativa string) pertenencia {
	switch {
	case strings.HasPrefix(relativa, "ca/") && relativa != "ca/ca.crt":
		return pertenenciaProhibida
	case strings.HasPrefix(relativa, "tls/") || strings.HasPrefix(relativa, "kms/") || strings.HasPrefix(relativa, "tsa/"):
		if _, comun := ficherosComunes[relativa]; !comun {
			if _, interno := ficherosInternos[relativa]; !interno {
				return pertenenciaProhibida
			}
		}
	case strings.HasSuffix(relativa, ".pem") || strings.HasSuffix(relativa, ".p12") || strings.HasSuffix(relativa, ".password"):
		return pertenenciaProhibida
	case strings.HasSuffix(relativa, ".key") && !strings.HasPrefix(relativa, "tls/") && !strings.HasPrefix(relativa, "kms/"):
		return pertenenciaProhibida
	}
	if _, ok := ficherosExternos[relativa]; ok || strings.HasPrefix(relativa, "externo/") {
		return pertenenciaExterna
	}
	if _, ok := ficherosComunes[relativa]; ok {
		return pertenenciaComun
	}
	for _, prefijo := range directoriosComunes {
		if strings.HasPrefix(relativa, prefijo) {
			return pertenenciaComun
		}
	}
	return pertenenciaInterna
}

type marcaPortal struct {
	Version int    `json:"version"`
	Portal  string `json:"portal"`
}

// ComprobarMaterial revisa el directorio de material de un proceso sin abrir
// ningún secreto: solo lee la marca de portal y la configuración declarada.
// En el portal combinado únicamente rechaza un material ya separado.
func ComprobarMaterial(p Portal, directorio string) error {
	if !p.Separado() {
		if p != PortalCombinado {
			return ErrPortalDesconocido
		}
		if directorio == "" {
			return nil
		}
		if _, err := os.Lstat(filepath.Join(directorio, FicheroMarcaPortal)); err == nil {
			return rechazo("material separado usado por un proceso combinado", FicheroMarcaPortal)
		}
		return nil
	}
	if directorio == "" || !filepath.IsAbs(directorio) {
		return rechazo("el proceso separado necesita su propio directorio de material", "")
	}
	marca, err := leerMarca(directorio)
	if err != nil {
		return err
	}
	if marca != p {
		return rechazo("el material pertenece a otro portal", FicheroMarcaPortal)
	}
	return recorrerMaterial(directorio, func(relativa string) error {
		switch clasificar(relativa) {
		case pertenenciaProhibida:
			return rechazo("clave privada de emision o de cliente en el material", relativa)
		case pertenenciaExterna:
			if p == PortalInterno {
				return rechazo("material del portal externo", relativa)
			}
		case pertenenciaInterna:
			if p == PortalExterno {
				return rechazo("material del portal interno o sin clasificar", relativa)
			}
		}
		if relativa == "desarrollo.env" {
			return comprobarEntornoDeclarado(p, filepath.Join(directorio, relativa))
		}
		return nil
	})
}

// leerMarca devuelve el portal declarado. Un fichero ausente, enlazado,
// legible por otros o mal formado impide arrancar.
func leerMarca(directorio string) (Portal, error) {
	contenido, err := leerFicheroAcotado(filepath.Join(directorio, FicheroMarcaPortal), tamanoMaximoMarca)
	if err != nil {
		return PortalCombinado, rechazo("marca de portal ausente o no valida", FicheroMarcaPortal)
	}
	var marca marcaPortal
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var sobra any
	if decodificador.Decode(&marca) != nil || !errors.Is(decodificador.Decode(&sobra), io.EOF) || marca.Version != 1 {
		return PortalCombinado, rechazo("marca de portal ausente o no valida", FicheroMarcaPortal)
	}
	portal, err := Parsear(marca.Portal)
	if err != nil || !portal.Separado() {
		return PortalCombinado, rechazo("marca de portal ausente o no valida", FicheroMarcaPortal)
	}
	return portal, nil
}

// recorrerMaterial visita cada fichero regular con su ruta relativa. No sigue
// enlaces: cualquier enlace, dispositivo, tubería o socket se rechaza, porque
// podría apuntar al material del otro proceso.
func recorrerMaterial(directorio string, visitar func(relativa string) error) error {
	info, err := os.Lstat(directorio)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return rechazo("directorio de material no valido", "")
	}
	entradas := 0
	return filepath.WalkDir(directorio, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return rechazo("no se puede recorrer el material", "")
		}
		relativa, errRel := filepath.Rel(directorio, ruta)
		if errRel != nil {
			return rechazo("no se puede recorrer el material", "")
		}
		relativa = filepath.ToSlash(relativa)
		if relativa == "." {
			return nil
		}
		entradas++
		if entradas > entradasMaximasMaterial || strings.Count(relativa, "/") >= profundidadMaximaMaterial {
			return rechazo("material demasiado grande o profundo", "")
		}
		tipo := d.Type()
		switch {
		case tipo&fs.ModeSymlink != 0:
			return rechazo("enlace en el material", relativa)
		case d.IsDir():
			return nil
		case !tipo.IsRegular():
			return rechazo("entrada que no es un fichero regular en el material", relativa)
		}
		return visitar(path.Clean(relativa))
	})
}

var lineaDeclaracion = regexp.MustCompile(`^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)

// comprobarEntornoDeclarado aplica a desarrollo.env las mismas reglas que al
// entorno del proceso: tampoco puede traer conexiones ni secretos ajenos.
func comprobarEntornoDeclarado(p Portal, ruta string) error {
	contenido, err := leerFicheroAcotado(ruta, tamanoMaximoEntornoDecl)
	if err != nil {
		return rechazo("configuracion declarada no legible", "desarrollo.env")
	}
	for _, v := range leerDeclaraciones(contenido) {
		if err := comprobarVariable(p, v.nombre, v.valor); err != nil {
			return err
		}
	}
	return nil
}

// leerDeclaraciones interpreta líneas NOMBRE=valor, con «export» opcional y
// comillas simples o dobles alrededor del valor. No expande variables.
func leerDeclaraciones(contenido []byte) []variable {
	var resultado []variable
	lector := bufio.NewScanner(bytes.NewReader(contenido))
	lector.Buffer(make([]byte, 0, 4096), tamanoMaximoEntornoDecl)
	for lector.Scan() {
		partes := lineaDeclaracion.FindStringSubmatch(lector.Text())
		if partes == nil {
			continue
		}
		valor := strings.TrimSpace(partes[2])
		if len(valor) >= 2 && (valor[0] == '"' || valor[0] == '\'') && valor[len(valor)-1] == valor[0] {
			valor = valor[1 : len(valor)-1]
		}
		resultado = append(resultado, variable{nombre: partes[1], valor: valor})
	}
	return resultado
}

// leerFicheroAcotado lee un fichero regular, no enlazado, sin permisos para
// grupo ni otros y de tamaño acotado, comprobando que no cambia al abrirlo.
func leerFicheroAcotado(ruta string, limite int64) ([]byte, error) {
	inicial, err := os.Lstat(ruta)
	if err != nil || !inicial.Mode().IsRegular() || inicial.Mode().Perm()&0o077 != 0 ||
		inicial.Size() < 1 || inicial.Size() > limite {
		return nil, errors.New("fichero no valido")
	}
	fichero, err := os.Open(ruta)
	if err != nil {
		return nil, err
	}
	defer fichero.Close()
	abierto, err := fichero.Stat()
	if err != nil || !os.SameFile(inicial, abierto) {
		return nil, errors.New("fichero cambiado")
	}
	contenido, err := io.ReadAll(io.LimitReader(fichero, limite+1))
	if err != nil || int64(len(contenido)) > limite {
		return nil, errors.New("fichero no valido")
	}
	return contenido, nil
}
