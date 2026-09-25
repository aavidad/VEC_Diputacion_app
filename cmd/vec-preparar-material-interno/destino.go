package main

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

const prefijoTemporal = ".vec-preparar-material-"

// destino fija el directorio padre de la salida por descriptor al empezar:
// el temporal se crea, se escribe, se retira y se activa siempre relativo a
// ese mismo inodo (os.Root y renameat(2)), nunca volviendo a resolver la
// ruta. Si la ruta del padre deja de apuntar a ese inodo o pierde su
// privacidad, la activación se rechaza.
type destino struct {
	padre, nombre string
	existia       bool
	dir           *os.File // O_DIRECTORY|O_NOFOLLOW sobre el padre
	raiz          *os.Root // mismo inodo que dir
}

// abrirDestino exige una salida inexistente o vacía (0700, del usuario, no
// enlace) bajo un padre canónico del usuario sin escritura de terceros y
// fuera de cualquier árbol Git.
func abrirDestino(salida string) (*destino, error) {
	if salida == "" || !filepath.IsAbs(salida) || filepath.Clean(salida) != salida || salida == "/" {
		return nil, errSalida
	}
	d := &destino{padre: filepath.Dir(salida), nombre: filepath.Base(salida)}
	if !rutaCanonica(d.padre) || dentroDeGit(d.padre) {
		return nil, errSalida
	}
	fd, err := syscall.Open(d.padre, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, errSalida
	}
	d.dir = os.NewFile(uintptr(fd), "padre")
	if d.raiz, err = os.OpenRoot(d.padre); err != nil {
		d.cerrar()
		return nil, errSalida
	}
	if !d.vigente() {
		d.cerrar()
		return nil, errSalida
	}
	info, err := d.raiz.Lstat(d.nombre)
	if os.IsNotExist(err) {
		return d, nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0700 || !delUsuario(info) || !d.vacia() {
		d.cerrar()
		return nil, errSalida
	}
	d.existia = true
	return d, nil
}

func (d *destino) cerrar() {
	if d == nil {
		return
	}
	if d.raiz != nil {
		_ = d.raiz.Close()
	}
	if d.dir != nil {
		_ = d.dir.Close()
	}
}

// vigente comprueba que el descriptor y el Root son el mismo inodo, que la
// ruta del padre sigue resolviéndose (sin enlaces) a ese inodo y que el
// padre sigue siendo del usuario y sin escritura de grupo ni otros.
func (d *destino) vigente() bool {
	fijado, err := d.dir.Stat()
	if err != nil || !fijado.IsDir() || !delUsuario(fijado) || fijado.Mode().Perm()&0022 != 0 {
		return false
	}
	porRoot, err := d.raiz.Stat(".")
	if err != nil || !os.SameFile(fijado, porRoot) || !rutaCanonica(d.padre) {
		return false
	}
	porRuta, err := os.Lstat(d.padre)
	return err == nil && os.SameFile(fijado, porRuta)
}

func (d *destino) vacia() bool {
	f, err := d.raiz.Open(d.nombre)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	return err == io.EOF
}

// crearTemporal crea un directorio 0700 de nombre aleatorio junto a la
// salida y lo devuelve abierto como Root.
func (d *destino) crearTemporal() (string, *os.Root, error) {
	var azar [12]byte
	if _, err := rand.Read(azar[:]); err != nil {
		return "", nil, errEscritura
	}
	nombre := prefijoTemporal + hex.EncodeToString(azar[:])
	if err := d.raiz.Mkdir(nombre, 0700); err != nil {
		return "", nil, errEscritura
	}
	raiz, err := d.raiz.OpenRoot(nombre)
	if err != nil || d.raiz.Chmod(nombre, 0700) != nil {
		if raiz != nil {
			_ = raiz.Close()
		}
		_ = d.raiz.RemoveAll(nombre)
		return "", nil, errEscritura
	}
	return nombre, raiz, nil
}

// rutaTemporal devuelve la ruta del temporal para los cargadores reales,
// que sólo aceptan rutas, tras comprobar que el padre sigue vigente y que la
// ruta resuelve exactamente al temporal abierto.
func (d *destino) rutaTemporal(nombre string, raiz *os.Root) (string, bool) {
	if !d.vigente() {
		return "", false
	}
	ruta := filepath.Join(d.padre, nombre)
	abierto, err := raiz.Stat(".")
	if err != nil {
		return "", false
	}
	porRuta, err := os.Lstat(ruta)
	return ruta, err == nil && porRuta.IsDir() && os.SameFile(abierto, porRuta)
}

func (d *destino) retirar(nombre string) { _ = d.raiz.RemoveAll(nombre) }

// activar sustituye la salida por el temporal con un único renameat(2)
// relativo al descriptor del padre. Si la salida no existía se crea vacía
// justo antes y se retira si el rename falla; si existía vacía, renameat(2)
// sólo la sustituye mientras siga vacía y siga siendo un directorio. Tras el
// rename el material ya está activado: un fallo de fsync del padre se
// devuelve como `sincronizado=false`, no como error.
func (d *destino) activar(temporal string, sincronizar func(*os.File) error) (sincronizado bool, err error) {
	if !d.vigente() {
		return false, errActivacion
	}
	creada := false
	if !d.existia {
		if err := d.raiz.Mkdir(d.nombre, 0700); err != nil {
			return false, errActivacion
		}
		creada = true
	}
	fd := int(d.dir.Fd())
	if err := syscall.Renameat(fd, temporal, fd, d.nombre); err != nil {
		if creada {
			_ = d.raiz.Remove(d.nombre)
		}
		return false, errActivacion
	}
	if sincronizar == nil {
		sincronizar = (*os.File).Sync
	}
	return sincronizar(d.dir) == nil, nil
}
