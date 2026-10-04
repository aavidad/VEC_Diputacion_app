package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

const noSeguirEnlaces = syscall.O_NOFOLLOW | syscall.O_CLOEXEC | syscall.O_NONBLOCK

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
			// Un directorio vacío llamado .git no contiene un repositorio.
			// Un fichero .git sí puede enlazar un worktree real.
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
	if err != nil || abierto.Mode().Perm() != 0700 || !propio(abierto) {
		_ = raiz.Close()
		return nil, errors.New("directorio_cambiado")
	}
	return raiz, nil
}

func propio(i os.FileInfo) bool {
	s, ok := i.Sys().(*syscall.Stat_t)
	return ok && int64(s.Uid) == int64(os.Getuid())
}

func leerPrivado(ruta string) ([]byte, error) {
	raiz, err := abrirRaizPrivada(ruta)
	if err != nil {
		return nil, err
	}
	defer raiz.Close()
	return leerEnRaiz(raiz, filepath.Base(ruta))
}

func leerEnRaiz(raiz *os.Root, nombre string) ([]byte, error) {
	i, err := raiz.Lstat(nombre)
	if err != nil || !i.Mode().IsRegular() || i.Mode().Perm() != 0600 || !propio(i) || i.Size() == 0 || i.Size() > limiteDocumento {
		return nil, errors.New("fichero")
	}
	f, err := raiz.OpenFile(nombre, os.O_RDONLY|noSeguirEnlaces, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil || !actual.Mode().IsRegular() || actual.Mode().Perm() != 0600 || !propio(actual) {
		return nil, errors.New("cambio")
	}
	b, err := io.ReadAll(io.LimitReader(f, limiteDocumento+1))
	if err != nil || int64(len(b)) != i.Size() {
		clear(b)
		return nil, errors.New("lectura")
	}
	return b, nil
}

func crearOComparar(ruta string, b []byte) error {
	raiz, err := abrirRaizPrivada(ruta)
	if err != nil {
		return err
	}
	defer raiz.Close()
	f, err := raiz.OpenFile(filepath.Base(ruta), os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if errors.Is(err, syscall.EEXIST) {
		previo, e := leerEnRaiz(raiz, filepath.Base(ruta))
		if e != nil {
			return e
		}
		defer clear(previo)
		if !bytes.Equal(previo, b) {
			return errors.New("divergencia")
		}
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Chmod(0600); err != nil {
		_ = raiz.Remove(filepath.Base(ruta))
		return err
	}
	if _, err := f.Write(b); err != nil {
		_ = raiz.Remove(filepath.Base(ruta))
		return err
	}
	if err := f.Sync(); err != nil {
		_ = raiz.Remove(filepath.Base(ruta))
		return err
	}
	return nil
}
