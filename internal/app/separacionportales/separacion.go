package separacionportales

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	tamanoMaximoSecreto   = 256 << 10
	tamanoMaximoJSON      = 256 << 10
	usuarioSistemaAnonimo = "(usuario del sistema)"
)

// Proceso describe lo que recibe cada proceso: su directorio de material y,
// opcionalmente, el contenido del guion que exporta sus variables (líneas
// NOMBRE=valor o export NOMBRE="valor").
type Proceso struct {
	Material string
	Entorno  []byte
}

// InformeSeparacion resume la comparación sin revelar ningún valor.
type InformeSeparacion struct {
	SecretosInterno   int
	SecretosExterno   int
	UsuariosInterno   int
	UsuariosExterno   int
	UsuariosComunes   []string
	SecretosRepetidos []string
}

// ComprobarSeparacion la ejecuta quien despliega, que ve a la vez el material
// y el entorno de los dos procesos. Cada proceso por sí solo no puede saber
// si su clave coincide con la del otro; esta comprobación sí: exige que
// ningún secreto (KMS, sellado de tiempo, idempotencia, clave TLS) tenga el
// mismo contenido en los dos directorios y que ningún usuario de PostgreSQL
// aparezca en las conexiones de ambos.
func ComprobarSeparacion(interno, externo Proceso) (InformeSeparacion, error) {
	var informe InformeSeparacion
	if err := comprobarDirectoriosDistintos(interno.Material, externo.Material); err != nil {
		return informe, err
	}
	if err := ComprobarMaterial(PortalInterno, interno.Material); err != nil {
		return informe, err
	}
	if err := ComprobarMaterial(PortalExterno, externo.Material); err != nil {
		return informe, err
	}
	for _, par := range []struct {
		portal  Portal
		entorno []byte
	}{{PortalInterno, interno.Entorno}, {PortalExterno, externo.Entorno}} {
		for _, v := range leerDeclaraciones(par.entorno) {
			if err := comprobarVariable(par.portal, v.nombre, v.valor); err != nil {
				return informe, err
			}
		}
	}
	secretosInterno, err := huellasSecretos(interno.Material)
	if err != nil {
		return informe, err
	}
	secretosExterno, err := huellasSecretos(externo.Material)
	if err != nil {
		return informe, err
	}
	informe.SecretosInterno, informe.SecretosExterno = len(secretosInterno), len(secretosExterno)
	for huella, rutaInterna := range secretosInterno {
		if rutaExterna, ok := secretosExterno[huella]; ok {
			informe.SecretosRepetidos = append(informe.SecretosRepetidos, rutaInterna+" = "+rutaExterna)
		}
	}
	usuariosInterno, err := usuariosConexiones(interno)
	if err != nil {
		return informe, err
	}
	usuariosExterno, err := usuariosConexiones(externo)
	if err != nil {
		return informe, err
	}
	// Sin usuario explícito, pgx usa el del sistema o PGUSER: no se puede
	// asegurar que sea distinto del otro proceso.
	if _, ok := usuariosInterno[usuarioSistemaAnonimo]; ok {
		return informe, rechazo("conexion sin usuario explicito en el proceso interno", "")
	}
	if _, ok := usuariosExterno[usuarioSistemaAnonimo]; ok {
		return informe, rechazo("conexion sin usuario explicito en el proceso externo", "")
	}
	informe.UsuariosInterno, informe.UsuariosExterno = len(usuariosInterno), len(usuariosExterno)
	for usuario := range usuariosInterno {
		if _, ok := usuariosExterno[usuario]; ok {
			informe.UsuariosComunes = append(informe.UsuariosComunes, usuario)
		}
	}
	sort.Strings(informe.SecretosRepetidos)
	sort.Strings(informe.UsuariosComunes)
	if len(informe.SecretosRepetidos) > 0 {
		return informe, rechazo("el mismo secreto esta en los dos procesos", informe.SecretosRepetidos[0])
	}
	if len(informe.UsuariosComunes) > 0 {
		return informe, rechazo("el mismo usuario de PostgreSQL esta en los dos procesos", informe.UsuariosComunes[0])
	}
	return informe, nil
}

func comprobarDirectoriosDistintos(a, b string) error {
	if a == "" || b == "" || !filepath.IsAbs(a) || !filepath.IsAbs(b) {
		return rechazo("cada proceso necesita un directorio de material absoluto", "")
	}
	ra, errA := filepath.EvalSymlinks(a)
	rb, errB := filepath.EvalSymlinks(b)
	if errA != nil || errB != nil || ra != filepath.Clean(a) || rb != filepath.Clean(b) {
		return rechazo("directorio de material enlazado o inexistente", "")
	}
	if dentro(ra, rb) || dentro(rb, ra) {
		return rechazo("los dos procesos comparten directorio de material", "")
	}
	return nil
}

// dentro indica si hijo es padre o está bajo él.
func dentro(padre, hijo string) bool {
	rel, err := filepath.Rel(padre, hijo)
	return err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))))
}

// esSecreto señala los ficheros cuyo contenido no puede coincidir entre
// procesos. Las claves públicas y los certificados pueden repetirse.
func esSecreto(relativa string) bool {
	switch {
	case strings.HasPrefix(relativa, "kms/"):
		return strings.HasSuffix(relativa, ".bin") || strings.HasSuffix(relativa, ".key")
	case strings.HasPrefix(relativa, "tsa/"), strings.HasPrefix(relativa, "idempotencia/"),
		strings.HasPrefix(relativa, "externo/"):
		return strings.HasSuffix(relativa, ".bin")
	case strings.HasPrefix(relativa, "tls/"):
		return strings.HasSuffix(relativa, ".key")
	}
	return false
}

// huellasSecretos devuelve la huella SHA-256 de cada secreto con su ruta.
// Solo se conservan las huellas para comparar; el contenido no sale de aquí.
func huellasSecretos(directorio string) (map[[sha256.Size]byte]string, error) {
	resultado := map[[sha256.Size]byte]string{}
	err := recorrerMaterial(directorio, func(relativa string) error {
		if !esSecreto(relativa) {
			return nil
		}
		contenido, err := leerFicheroAcotado(filepath.Join(directorio, filepath.FromSlash(relativa)), tamanoMaximoSecreto)
		if err != nil {
			return rechazo("secreto no legible", relativa)
		}
		huella := sha256.Sum256(contenido)
		clear(contenido)
		if otra, ok := resultado[huella]; ok {
			return rechazo("el mismo secreto se usa para dos fines", otra+" = "+relativa)
		}
		resultado[huella] = relativa
		return nil
	})
	return resultado, err
}

// usuariosConexiones reúne los usuarios de PostgreSQL de las conexiones del
// entorno declarado y de los ficheros JSON del material.
func usuariosConexiones(p Proceso) (map[string]struct{}, error) {
	usuarios := map[string]struct{}{}
	for _, v := range leerDeclaraciones(p.Entorno) {
		if esConexion(v.nombre, v.valor) {
			usuarios[usuarioConexion(v.valor)] = struct{}{}
		}
	}
	err := recorrerMaterial(p.Material, func(relativa string) error {
		if !strings.HasSuffix(relativa, ".json") {
			return nil
		}
		contenido, err := leerFicheroAcotado(filepath.Join(p.Material, filepath.FromSlash(relativa)), tamanoMaximoJSON)
		if err != nil {
			return rechazo("fichero de material no legible", relativa)
		}
		defer clear(contenido)
		decodificador := json.NewDecoder(bytes.NewReader(contenido))
		for {
			simbolo, err := decodificador.Token()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return rechazo("fichero de material no es JSON valido", relativa)
			}
			if texto, ok := simbolo.(string); ok && esConexion("", texto) {
				usuarios[usuarioConexion(texto)] = struct{}{}
			}
		}
	})
	return usuarios, err
}

var (
	usuarioClaveValor = regexp.MustCompile(`(?:^|\s)user\s*=\s*'?([^'\s]+)`)
	usuarioURL        = regexp.MustCompile(`(?i)^postgres(?:ql)?://([^:@/?#]+)(?::[^@/]*)?@`)
	usuarioConsulta   = regexp.MustCompile(`[?&]user=([^&#]+)`)
)

// usuarioConexion extrae el usuario de una cadena de conexión en forma de URL
// o de pares clave=valor, sin interpretar el resto: los guiones de arranque
// dejan a veces variables de shell en el puerto. Como en pgx, el parámetro
// user de la consulta prevalece. Sin usuario explícito, pgx usa el del
// sistema.
func usuarioConexion(dsn string) string {
	dsn = strings.TrimSpace(dsn)
	minusculas := strings.ToLower(dsn)
	if strings.HasPrefix(minusculas, "postgres://") || strings.HasPrefix(minusculas, "postgresql://") {
		if partes := usuarioConsulta.FindStringSubmatch(dsn); partes != nil {
			return decodificarUsuario(partes[1])
		}
		if partes := usuarioURL.FindStringSubmatch(dsn); partes != nil {
			return decodificarUsuario(partes[1])
		}
		return usuarioSistemaAnonimo
	}
	if partes := usuarioClaveValor.FindStringSubmatch(dsn); partes != nil {
		return partes[1]
	}
	return usuarioSistemaAnonimo
}

func decodificarUsuario(codificado string) string {
	if usuario, err := url.PathUnescape(codificado); err == nil && usuario != "" {
		return usuario
	}
	return codificado
}
