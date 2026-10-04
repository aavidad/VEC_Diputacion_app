package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

func leerEntradaPreparacion(ruta string, limite int) ([]byte, error) {
	// La carpeta local explícita queda fijada por descriptor. No se siguen
	// enlaces en la hoja y su apertura no bloquea ante un FIFO.
	dir, err := os.OpenRoot(filepath.Dir(ruta))
	if err != nil {
		return nil, err
	}
	defer func() { _ = dir.Close() }()
	f, err := dir.OpenFile(filepath.Base(ruta), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	antes, err := f.Stat()
	if err != nil || !antes.Mode().IsRegular() || antes.Size() < 2 || antes.Size() > int64(limite) {
		return nil, errors.Join(errMaterial, err)
	}
	b, err := io.ReadAll(io.LimitReader(f, int64(limite)+1))
	despues, errStat := f.Stat()
	errClose := f.Close()
	if err != nil || errStat != nil || errClose != nil {
		return nil, errors.Join(err, errStat, errClose)
	}
	if len(b) > limite || int64(len(b)) != despues.Size() || antes.Size() != despues.Size() ||
		!antes.ModTime().Equal(despues.ModTime()) || !despues.Mode().IsRegular() {
		return nil, errMaterial
	}
	return b, nil
}

// El descriptor de la carpeta fija el destino y su limpieza aunque se cambie
// el nombre de la carpeta durante la operación. O_EXCL nunca reemplaza una hoja.
func conservarPreparacion(ruta string, b []byte, r resumen, salida io.Writer) (err error) {
	// #nosec G703 -- carpeta local elegida por el operador; debe ser privada.
	dir, err := os.OpenRoot(filepath.Dir(ruta))
	if err != nil {
		return err
	}
	defer func() { _ = dir.Close() }()
	info, err := dir.Stat(".")
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return errors.Join(errMaterial, err)
	}
	nombre := filepath.Base(ruta)
	f, err := dir.OpenFile(nombre, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	conservado := false
	defer func() {
		_ = f.Close()
		if !conservado {
			err = errors.Join(err, dir.Remove(nombre))
		}
	}()
	if err = f.Chmod(0600); err != nil {
		return err
	}
	if _, err = io.Copy(f, bytes.NewReader(b)); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	// Relectura desde la misma carpeta y cotejo de todos los bytes antes de
	// informar de la preparación; ninguna transformación entra en el fichero.
	relectura, err := dir.OpenFile(nombre, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	actual, errStat := relectura.Stat()
	if errStat != nil || !actual.Mode().IsRegular() || actual.Mode().Perm() != 0600 || actual.Size() != int64(len(b)) {
		_ = relectura.Close()
		return errors.Join(errMaterial, errStat)
	}
	contenido, errRead := io.ReadAll(io.LimitReader(relectura, maxMaterial+1))
	errClose := relectura.Close()
	if errStat != nil || errRead != nil || errClose != nil {
		return errors.Join(errStat, errRead, errClose)
	}
	if !bytes.Equal(b, contenido) {
		return errMaterial
	}
	if err = json.NewEncoder(salida).Encode(r); err != nil {
		return err
	}
	conservado = true
	return nil
}
