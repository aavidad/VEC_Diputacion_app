package observabilidad

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const archivoActivoRecolector = "incidencias-activo.jsonl"

type archivoRecolector struct {
	nombre     string
	secuencia  uint64
	modificado time.Time
}

type almacenRecolector struct {
	cfg       ConfiguracionRecolector
	raiz      *os.Root
	lock      *os.File
	activo    *os.File
	tamano    int64
	secuencia uint64
	retirados uint64
	reloj     func() time.Time
	desde     time.Time
}

func abrirAlmacenRecolector(cfg ConfiguracionRecolector, reloj func() time.Time) (_ *almacenRecolector, err error) {
	if !filepath.IsAbs(cfg.Directorio) {
		return nil, os.ErrInvalid
	}
	if err := os.Mkdir(cfg.Directorio, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, err
	}
	info, err := os.Lstat(cfg.Directorio)
	stat, segura := infoSysSeguro(info)
	if err != nil || !segura || !propietarioRecolectorValido(stat) || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return nil, os.ErrInvalid
	}
	raiz, err := os.OpenRoot(cfg.Directorio)
	if err != nil {
		return nil, err
	}
	info, err = raiz.Stat(".")
	stat, segura = infoSysSeguro(info)
	if err != nil || !segura || !propietarioRecolectorValido(stat) || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		_ = raiz.Close()
		return nil, os.ErrInvalid
	}
	a := &almacenRecolector{cfg: cfg, raiz: raiz, reloj: reloj}
	defer func() {
		if err != nil {
			_ = a.cerrar()
		}
	}()
	// El bloqueo se conserva durante las rotaciones y se libera al terminar
	// el proceso. El archivo vacío puede sobrevivir a una caída sin bloquear
	// el siguiente arranque; no se usa una marca exclusiva que quede obsoleta.
	a.lock, err = a.abrirPrivado(".recolector.lock", os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, err
	}
	fd := a.lock.Fd()
	if fd > math.MaxInt {
		return nil, os.ErrInvalid
	}
	if err = syscall.Flock(int(fd), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return nil, err
	}
	archivos, err := a.archivos()
	if err != nil {
		return nil, err
	}
	for _, f := range archivos {
		if f.secuencia > a.secuencia {
			a.secuencia = f.secuencia
		}
	}
	if err = a.aplicarRetencion(archivos); err != nil {
		return nil, err
	}
	a.activo, err = a.abrirPrivado(archivoActivoRecolector, os.O_CREATE|os.O_RDWR|os.O_APPEND)
	if err != nil {
		return nil, err
	}
	info, err = a.activo.Stat()
	if err != nil {
		return nil, err
	}
	a.tamano = info.Size()
	a.desde = reloj()
	if a.tamano > 0 {
		a.desde = info.ModTime()
		var fin [1]byte
		if _, err = a.activo.ReadAt(fin[:], a.tamano-1); err != nil || fin[0] != '\n' {
			return nil, os.ErrInvalid
		}
		if a.tamano >= cfg.MaxArchivoBytes || reloj().Sub(info.ModTime()) >= time.Duration(cfg.RetencionSegundos)*time.Second {
			if err = a.rotar(); err != nil {
				return nil, err
			}
		}
	}
	return a, nil
}

func (a *almacenRecolector) abrirPrivado(nombre string, flags int) (*os.File, error) {
	// os.Root limita toda resolución al directorio privado, y NOFOLLOW y
	// NONBLOCK impiden seguir un enlace de hoja o bloquearse en una FIFO.
	f, err := a.raiz.OpenFile(nombre, flags|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	stat, segura := infoSysSeguro(info)
	if err != nil || !segura || !propietarioRecolectorValido(stat) || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || stat.Nlink != 1 {
		_ = f.Close()
		return nil, os.ErrInvalid
	}
	return f, nil
}

func (a *almacenRecolector) archivos() ([]archivoRecolector, error) {
	dir, err := a.raiz.Open(".")
	if err != nil {
		return nil, err
	}
	entradas, err := dir.ReadDir(-1)
	if e := dir.Close(); err == nil {
		err = e
	}
	if err != nil {
		return nil, err
	}
	var archivos []archivoRecolector
	for _, entrada := range entradas {
		nombre := entrada.Name()
		if !strings.HasPrefix(nombre, "incidencias-") || !strings.HasSuffix(nombre, ".jsonl") || nombre == archivoActivoRecolector {
			continue
		}
		numero := strings.TrimSuffix(strings.TrimPrefix(nombre, "incidencias-"), ".jsonl")
		secuencia, err := strconv.ParseUint(numero, 10, 64)
		if err != nil {
			return nil, os.ErrInvalid
		}
		if secuencia == 0 || fmt.Sprintf("%020d", secuencia) != numero {
			continue
		}
		info, err := a.raiz.Lstat(nombre)
		stat, segura := infoSysSeguro(info)
		if err != nil || !segura || !propietarioRecolectorValido(stat) || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
			return nil, os.ErrInvalid
		}
		archivos = append(archivos, archivoRecolector{nombre, secuencia, info.ModTime()})
	}
	sort.Slice(archivos, func(i, j int) bool { return archivos[i].secuencia < archivos[j].secuencia })
	return archivos, nil
}

func (a *almacenRecolector) aplicarRetencion(archivos []archivoRecolector) error {
	for i, f := range archivos {
		if len(archivos)-i >= a.cfg.MaxArchivos || a.reloj().Sub(f.modificado) >= time.Duration(a.cfg.RetencionSegundos)*time.Second {
			if err := a.raiz.Remove(f.nombre); err != nil {
				return err
			}
			a.retirados++
		}
	}
	return nil
}

func (a *almacenRecolector) rotar() error {
	if a.secuencia == ^uint64(0) {
		return os.ErrInvalid
	}
	if a.activo.Sync() != nil || a.activo.Close() != nil {
		return os.ErrInvalid
	}
	a.activo = nil
	a.secuencia++
	nombre := fmt.Sprintf("incidencias-%020d.jsonl", a.secuencia)
	// Un nombre ya existente nunca se sustituye, aunque no pertenezca a la
	// lista administrada. El directorio es exclusivo del usuario del proceso.
	if _, err := a.raiz.Lstat(nombre); !errors.Is(err, os.ErrNotExist) {
		return os.ErrInvalid
	}
	if err := a.raiz.Rename(archivoActivoRecolector, nombre); err != nil {
		return err
	}
	archivos, err := a.archivos()
	if err != nil {
		return err
	}
	if err := a.aplicarRetencion(archivos); err != nil {
		return err
	}
	a.activo, err = a.abrirPrivado(archivoActivoRecolector, os.O_CREATE|os.O_EXCL|os.O_RDWR|os.O_APPEND)
	a.tamano = 0
	a.desde = a.reloj()
	return err
}

func (a *almacenRecolector) escribir(linea []byte) error {
	if int64(len(linea)) > a.cfg.MaxArchivoBytes {
		return os.ErrInvalid
	}
	if a.tamano > 0 && (a.tamano > a.cfg.MaxArchivoBytes-int64(len(linea)) || a.reloj().Sub(a.desde) >= time.Duration(a.cfg.RetencionSegundos)*time.Second) {
		if err := a.rotar(); err != nil {
			return err
		}
	}
	n, err := a.activo.Write(linea)
	if err != nil || n != len(linea) {
		return os.ErrInvalid
	}
	a.tamano += int64(n)
	return nil
}

func infoSysSeguro(info os.FileInfo) (*syscall.Stat_t, bool) {
	if info == nil {
		return nil, false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return stat, ok
}

func propietarioRecolectorValido(stat *syscall.Stat_t) bool {
	uid := os.Geteuid()
	if uid < 0 || uid > math.MaxUint32 {
		return false
	}
	return stat.Uid == uint32(uid)
}

func (a *almacenRecolector) cerrar() error {
	var err error
	if a.activo != nil {
		err = a.activo.Sync()
		err = errors.Join(err, a.activo.Close())
	}
	if a.lock != nil {
		err = errors.Join(err, a.lock.Close())
	}
	if a.raiz != nil {
		err = errors.Join(err, a.raiz.Close())
	}
	return err
}
