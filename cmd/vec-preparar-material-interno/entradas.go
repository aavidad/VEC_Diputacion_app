package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	core "vec-diputacion-granada/internal/vec/domain"
)

const (
	errDSN           = errorPropio("DSN del gobierno: indique exactamente uno, la variable " + variableDSN + " o -dsn-archivo")
	errDSNFichero    = errorPropio("DSN del gobierno: fichero inseguro o ilegible")
	errClaveBase     = errorPropio("clave base: fichero inseguro, ilegible o de tamaño no admitido")
	errMotivos       = errorPropio("motivos: fichero inseguro, ilegible o con formato no admitido")
	errInventarioCT  = errorPropio("inventario CT: ruta insegura o ct_v3.json rechazado por el cargador real")
	errEmisorCT      = errorPropio("inventario CT: las capacidades no comparten un único emisor")
	maximoClaveBase  = 256
	maximoMotivos    = 16 << 10
	maximoDSNFichero = 4 << 10
)

type opciones struct {
	inventarioCT, claveBase, motivos, salida, dsnArchivo string
}

// motivosB2 es el formato del fichero de motivos: exactamente las ocho
// referencias del catálogo gobernado, con los mismos nombres que el
// inventario de formato 4.
type motivosB2 struct {
	Ficha             core.ReferenciaEntradaCatalogo `json:"ficha"`
	Vacantes          core.ReferenciaEntradaCatalogo `json:"vacantes"`
	Alta              core.ReferenciaEntradaCatalogo `json:"alta"`
	Hecho             core.ReferenciaEntradaCatalogo `json:"hecho"`
	CatalogoConsultar core.ReferenciaEntradaCatalogo `json:"catalogo_consultar"`
	CatalogoPublicar  core.ReferenciaEntradaCatalogo `json:"catalogo_publicar"`
	CatalogoRetirar   core.ReferenciaEntradaCatalogo `json:"catalogo_retirar"`
	Empleados         core.ReferenciaEntradaCatalogo `json:"empleados"`
}

func (m motivosB2) lista() [8]core.ReferenciaEntradaCatalogo {
	return [8]core.ReferenciaEntradaCatalogo{m.Ficha, m.Vacantes, m.Alta, m.Hecho, m.CatalogoConsultar, m.CatalogoPublicar, m.CatalogoRetirar, m.Empleados}
}

// rutaCanonica exige una ruta absoluta, limpia y sin enlaces simbólicos en
// ninguno de sus componentes.
func rutaCanonica(ruta string) bool {
	if ruta == "" || !filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta || strings.TrimSpace(ruta) != ruta {
		return false
	}
	resuelta, err := filepath.EvalSymlinks(ruta)
	return err == nil && resuelta == ruta
}

// leerFicheroPrivado abre un fichero regular 0600 del usuario sin seguir
// enlaces (O_NOFOLLOW), comprueba que el descriptor abierto es el mismo
// inodo inspeccionado y lee como máximo `limite` bytes. El llamante borra
// el resultado.
func leerFicheroPrivado(ruta string, limite int64) ([]byte, bool) {
	if !rutaCanonica(ruta) {
		return nil, false
	}
	info, err := os.Lstat(ruta)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || !delUsuario(info) || info.Size() < 1 || info.Size() > limite {
		return nil, false
	}
	fd, err := syscall.Open(ruta, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, false
	}
	f := os.NewFile(uintptr(fd), "privado")
	defer f.Close()
	actual, err := f.Stat()
	if err != nil || !os.SameFile(info, actual) {
		return nil, false
	}
	b, err := io.ReadAll(io.LimitReader(f, limite+1))
	if err != nil || int64(len(b)) != info.Size() {
		clear(b)
		return nil, false
	}
	return b, true
}

func delUsuario(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(st.Uid) == os.Getuid()
}

func leerClaveBase(ruta string) ([]byte, error) {
	b, ok := leerFicheroPrivado(ruta, maximoClaveBase)
	if !ok || len(b) < 32 {
		clear(b)
		return nil, errClaveBase
	}
	return b, nil
}

func leerMotivos(ruta, catalogo string) (motivosB2, error) {
	var m motivosB2
	b, ok := leerFicheroPrivado(ruta, maximoMotivos)
	if !ok {
		return m, errMotivos
	}
	defer clear(b)
	if clavesDuplicadas(b) {
		return m, errMotivos
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&m) != nil || d.Decode(new(any)) != io.EOF {
		return motivosB2{}, errMotivos
	}
	for _, r := range m.lista() {
		if !core.ReferenciaMotivoAutorizacionV2Valida(r) || r.CatalogoID != catalogo {
			return motivosB2{}, errMotivos
		}
	}
	return m, nil
}

// leerDSN toma el DSN del fichero 0600 o de la variable de entorno. Nunca se
// acepta como argumento del proceso.
func leerDSN(archivo, entorno string) (string, error) {
	if archivo == "" {
		if strings.TrimSpace(entorno) == "" || strings.ContainsAny(entorno, "\r\n") {
			return "", errDSN
		}
		return entorno, nil
	}
	b, ok := leerFicheroPrivado(archivo, maximoDSNFichero)
	if !ok {
		return "", errDSNFichero
	}
	defer clear(b)
	texto := strings.TrimSuffix(string(b), "\n")
	if strings.TrimSpace(texto) == "" || strings.ContainsAny(texto, "\r\n") {
		return "", errDSNFichero
	}
	return texto, nil
}

// clavesDuplicadas rechaza objetos JSON con nombres repetidos, que el
// decodificador estándar aceptaría quedándose con el último.
func clavesDuplicadas(b []byte) bool {
	d := json.NewDecoder(bytes.NewReader(b))
	var leer func() bool
	leer = func() bool {
		t, err := d.Token()
		if err != nil {
			return false
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return true
		}
		switch delim {
		case '{':
			vistas := map[string]bool{}
			for d.More() {
				k, err := d.Token()
				s, ok := k.(string)
				if err != nil || !ok || vistas[s] {
					return false
				}
				vistas[s] = true
				if !leer() {
					return false
				}
			}
		case '[':
			for d.More() {
				if !leer() {
					return false
				}
			}
		default:
			return false
		}
		_, err = d.Token()
		return err == nil
	}
	if !leer() {
		return true
	}
	_, err := d.Token()
	return err != io.EOF
}

// dentroDeGit replica la frontera del cargador real: ningún material privado
// vive bajo un árbol Git.
func dentroDeGit(ruta string) bool {
	for actual := ruta; ; {
		if _, err := os.Lstat(filepath.Join(actual, ".git")); err == nil {
			return true
		}
		padre := filepath.Dir(actual)
		if padre == actual {
			return false
		}
		actual = padre
	}
}
