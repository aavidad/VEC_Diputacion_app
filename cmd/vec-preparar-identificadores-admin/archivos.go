package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

const noSeguirEnlaces = syscall.O_NOFOLLOW | syscall.O_CLOEXEC | syscall.O_NONBLOCK

// Mismo protocolo que los demás comandos de preparación administrativa y que
// el cargador del runtime: directorio propio 0700, sin enlaces ni repositorio.
func abrirRaizPrivada(ruta string) (*os.Root, error) {
	if !filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta {
		return nil, errors.New("ruta")
	}
	padre := filepath.Dir(ruta)
	resuelta, err := filepath.EvalSymlinks(padre)
	if err != nil || resuelta != padre {
		return nil, errors.New("enlace")
	}
	dir, err := os.Lstat(padre)
	if err != nil || !dir.IsDir() || dir.Mode().Perm() != 0700 || !propio(dir) {
		return nil, errors.New("directorio")
	}
	for actual := padre; actual != "/"; actual = filepath.Dir(actual) {
		marca := filepath.Join(actual, ".git")
		if info, err := os.Lstat(marca); err == nil {
			// Un fichero .git enlaza un worktree; un directorio con HEAD es un
			// repositorio. Un directorio .git vacío no contiene ninguno.
			if !info.IsDir() {
				return nil, errors.New("repositorio")
			}
			if _, err := os.Lstat(filepath.Join(marca, "HEAD")); err == nil {
				return nil, errors.New("repositorio")
			}
		}
	}
	raiz, err := os.OpenRoot(padre)
	if err != nil {
		return nil, err
	}
	abierto, err := raiz.Stat(".")
	if err != nil || !os.SameFile(dir, abierto) || abierto.Mode().Perm() != 0700 || !propio(abierto) {
		_ = raiz.Close()
		return nil, errors.New("directorio_cambiado")
	}
	return raiz, nil
}

func propio(i os.FileInfo) bool {
	s, ok := i.Sys().(*syscall.Stat_t)
	return ok && int64(s.Uid) == int64(os.Getuid())
}

func leerPrivado(ruta string) ([]byte, error) { return leerPrivadoHasta(ruta, limiteDocumento) }

func leerPrivadoHasta(ruta string, limite int64) ([]byte, error) {
	raiz, err := abrirRaizPrivada(ruta)
	if err != nil {
		return nil, err
	}
	defer raiz.Close()
	nombre := filepath.Base(ruta)
	i, err := raiz.Lstat(nombre)
	if err != nil || !i.Mode().IsRegular() || i.Mode().Perm() != 0600 || !propio(i) || i.Size() == 0 || i.Size() > limite {
		return nil, errors.New("fichero")
	}
	f, err := raiz.OpenFile(nombre, os.O_RDONLY|noSeguirEnlaces, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil || !os.SameFile(i, actual) || !actual.Mode().IsRegular() || actual.Mode().Perm() != 0600 || !propio(actual) {
		return nil, errors.New("cambio")
	}
	b, err := io.ReadAll(io.LimitReader(f, limite+1))
	if err != nil || int64(len(b)) != i.Size() {
		clear(b)
		return nil, errors.New("lectura")
	}
	return b, nil
}

// crearExclusivo nunca sobrescribe: una salida existente es un rechazo. Si la
// escritura falla, retira el archivo incompleto que acaba de crear.
func crearExclusivo(ruta string, b []byte) error {
	raiz, err := abrirRaizPrivada(ruta)
	if err != nil {
		return err
	}
	defer raiz.Close()
	nombre := filepath.Base(ruta)
	f, err := raiz.OpenFile(nombre, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Chmod(0600)
	}
	if err == nil {
		err = f.Sync()
	}
	if cerrar := f.Close(); err == nil {
		err = cerrar
	}
	if err == nil {
		// La entrada del directorio también debe quedar en disco.
		var dir *os.File
		if dir, err = raiz.Open("."); err == nil {
			err = dir.Sync()
			if cerrar := dir.Close(); err == nil {
				err = cerrar
			}
		}
	}
	if err != nil {
		_ = raiz.Remove(nombre)
	}
	return err
}

// retirar informa si el archivo rechazado sigue en disco.
func retirar(ruta string) error {
	raiz, err := abrirRaizPrivada(ruta)
	if err != nil {
		return err
	}
	defer raiz.Close()
	return raiz.Remove(filepath.Base(ruta))
}
