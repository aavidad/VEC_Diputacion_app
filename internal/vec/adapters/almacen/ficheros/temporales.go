package ficheros

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"vec-diputacion-granada/internal/vec/ports"
)

// prepararCerrojoYLimpiar toma ambos cerrojos en orden fijo: primero el de
// vida de los temporales y después el de operación. Un escritor mantiene el
// primero hasta renombrar o retirar su temporal.
func prepararCerrojoYLimpiar(raiz string) (*os.File, error) {
	cerrojo, err := os.OpenFile(filepath.Join(raiz, ficheroCerrojo), os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, ErrDirectorioInseguro
	}
	if info, err := cerrojo.Stat(); err != nil || !ficheroPrivado(info) {
		_ = cerrojo.Close()
		return nil, ErrDirectorioInseguro
	}
	volcados, err := os.OpenFile(filepath.Join(raiz, ficheroVolcados), os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		_ = cerrojo.Close()
		return nil, ErrDirectorioInseguro
	}
	if info, err := volcados.Stat(); err != nil || !ficheroPrivado(info) {
		_ = volcados.Close()
		_ = cerrojo.Close()
		return nil, ErrDirectorioInseguro
	}
	if err := syscall.Flock(int(volcados.Fd()), syscall.LOCK_EX); err != nil {
		_ = volcados.Close()
		_ = cerrojo.Close()
		return nil, ErrDirectorioInseguro
	}
	if err := syscall.Flock(int(cerrojo.Fd()), syscall.LOCK_EX); err != nil {
		_ = syscall.Flock(int(volcados.Fd()), syscall.LOCK_UN)
		_ = volcados.Close()
		_ = cerrojo.Close()
		return nil, ErrDirectorioInseguro
	}
	borrados, err := limpiarTemporales(filepath.Join(raiz, dirTemporal))
	_ = syscall.Flock(int(cerrojo.Fd()), syscall.LOCK_UN)
	_ = syscall.Flock(int(volcados.Fd()), syscall.LOCK_UN)
	_ = volcados.Close()
	if err != nil {
		_ = cerrojo.Close()
		return nil, err
	}
	if borrados > 0 {
		log.Printf("vec: temporales huérfanos borrados: %d", borrados)
	}
	return cerrojo, nil
}

func ficheroPrivado(info fs.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return info.Mode().IsRegular() && info.Mode().Perm()&0o077 == 0 && ok && int(st.Uid) == os.Geteuid()
}

// limpiarTemporales solo elimina ficheros regulares tmp_* y nunca sigue
// enlaces. Se llama con ambos flock exclusivos.
func limpiarTemporales(ruta string) (int, error) {
	entradas, err := os.ReadDir(ruta)
	if err != nil {
		return 0, ErrDirectorioInseguro
	}
	borrados := 0
	for _, entrada := range entradas {
		nombre := entrada.Name()
		if !strings.HasPrefix(nombre, "tmp_") {
			continue
		}
		ubicacion := filepath.Join(ruta, nombre)
		info, err := os.Lstat(ubicacion)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return borrados, ErrDirectorioInseguro
		}
		if !info.Mode().IsRegular() || info.Mode()&fs.ModeSymlink != 0 {
			continue
		}
		if err := os.Remove(ubicacion); err != nil {
			return borrados, ErrDirectorioInseguro
		}
		borrados++
	}
	if borrados > 0 && sincronizarDirectorio(ruta) != nil {
		return borrados, ErrDirectorioInseguro
	}
	return borrados, nil
}

// cerrojoVolcado impide que otro proceso limpie un temporal todavía activo.
// Se mantiene hasta terminar la operación, también al esperar el otro cerrojo.
func (a *Almacen) cerrojoVolcado() (func(), error) {
	f, err := os.OpenFile(filepath.Join(a.raiz, ficheroVolcados), os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, ports.ErrCapacidadAlmacenNoDisponible
	}
	info, err := f.Stat()
	if err != nil || !ficheroPrivado(info) {
		_ = f.Close()
		return nil, ports.ErrCapacidadAlmacenNoDisponible
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_SH); err != nil {
		_ = f.Close()
		return nil, ports.ErrCapacidadAlmacenNoDisponible
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}

// volcarContenido copia exactamente tamano bytes a un temporal, calcula la
// huella y sincroniza. Devuelve la ruta temporal que el llamante debe
// renombrar o borrar.
func (a *Almacen) volcarContenido(ctx context.Context, origen io.Reader, tamano int64, huella string) (string, error) {
	tmp, err := a.temporal()
	if err != nil {
		return "", err
	}
	nombre := tmp.Name()
	fallo := func(e error) (string, error) {
		_ = tmp.Close()
		_ = os.Remove(nombre)
		return "", e
	}
	h := sha256.New()
	copiados, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(lectorCancelable{ctx: ctx, r: origen}, tamano+1))
	if ctx.Err() != nil {
		return fallo(ctx.Err())
	}
	if err != nil {
		return fallo(ports.ErrCapacidadAlmacenNoDisponible)
	}
	if copiados != tamano || hex.EncodeToString(h.Sum(nil)) != huella {
		return fallo(ports.ErrIntegridadObjetoAlmacen)
	}
	if tmp.Sync() != nil || tmp.Close() != nil {
		_ = os.Remove(nombre)
		return "", ports.ErrCapacidadAlmacenNoDisponible
	}
	return nombre, nil
}

type lectorCancelable struct {
	ctx context.Context
	r   io.Reader
}

func (l lectorCancelable) Read(p []byte) (int, error) {
	if err := l.ctx.Err(); err != nil {
		return 0, err
	}
	return l.r.Read(p)
}
