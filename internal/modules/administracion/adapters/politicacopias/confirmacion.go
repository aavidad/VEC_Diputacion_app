//go:build linux

package politicacopias

import (
	"encoding/json"
	"errors"
	"os"
	"syscall"
	d "vec-diputacion-granada/internal/modules/administracion/domain/politicacopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/politicacopias"
)

type confirmacion struct {
	Version uint64 `json:"version"`
	SHA256  string `json:"sha256"`
}

// An independent high-water marker detects removal of a complete last version,
// which the remaining hash chain alone cannot detect. A split/partial commit
// blocks for reconciliation instead of silently accepting a shortened history.
func (a *Archivo) comprobarConfirmacion(history []p.Registro) error {
	f, e := a.root.OpenFile("confirmado", os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if errors.Is(e, os.ErrNotExist) && len(history) == 0 {
		return nil
	}
	if e != nil {
		return d.ErrHistoria
	}
	defer func() { _ = f.Close() }()
	info, e := f.Stat()
	if e != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || !privado(info) {
		return d.ErrHistoria
	}
	var mark confirmacion
	if Decodificar(f, &mark) != nil || len(history) == 0 {
		return d.ErrHistoria
	}
	last := history[len(history)-1]
	if mark.Version != last.Version || mark.SHA256 != last.RegistroSHA256 {
		return d.ErrHistoria
	}
	return nil
}
func (a *Archivo) confirmar(r p.Registro) error {
	pending := "confirmacion-pendiente"
	f, e := a.root.OpenFile(pending, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0600)
	if errors.Is(e, os.ErrExist) {
		if a.root.Remove(pending) != nil {
			return d.ErrDependencia
		}
		f, e = a.root.OpenFile(pending, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0600)
	}
	if e != nil {
		return d.ErrDependencia
	}
	defer func() { _ = a.root.Remove(pending) }()
	e = json.NewEncoder(f).Encode(confirmacion{r.Version, r.RegistroSHA256})
	if e == nil {
		e = f.Sync()
	}
	closed := f.Close()
	if e != nil || closed != nil {
		return d.ErrDependencia
	}
	if a.root.Rename(pending, "confirmado") != nil {
		return d.ErrDependencia
	}
	dir, e := a.root.Open(".")
	if e != nil {
		return d.ErrDependencia
	}
	e = dir.Sync()
	_ = dir.Close()
	if e != nil {
		return d.ErrDependencia
	}
	return nil
}
