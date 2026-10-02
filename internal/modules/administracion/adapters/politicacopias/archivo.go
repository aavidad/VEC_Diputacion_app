//go:build linux

package politicacopias

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"
	"time"
	d "vec-diputacion-granada/internal/modules/administracion/domain/politicacopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/politicacopias"
)

const maxVersiones = 4096

// Archivo stores one policy and its journal in an operator-selected external
// control directory. It contains no backup payload. SHA256 checks accidental
// modification only; OS permissions, central audit and external authenticity
// remain production infrastructure obligations. The directory must survive rollback.
type Archivo struct{ root *os.Root }

func Abrir(ruta string) (*Archivo, error) {
	if !filepath.IsAbs(ruta) {
		return nil, d.ErrEntrada
	}
	// Only the leaf may be created. Do not create arbitrary parent trees.
	if err := os.Mkdir(ruta, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, d.ErrDependencia
	}
	info, err := os.Lstat(ruta)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return nil, d.ErrDependencia
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int64(owner.Uid) != int64(os.Geteuid()) {
		return nil, d.ErrDependencia
	}
	root, err := os.OpenRoot(ruta)
	if err != nil {
		return nil, d.ErrDependencia
	}
	return &Archivo{root}, nil
}
func (a *Archivo) Cerrar() error { return a.root.Close() }
func (a *Archivo) bloqueo(ctx context.Context) (func(), error) {
	f, e := a.root.OpenFile("control.lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if e != nil {
		return nil, d.ErrDependencia
	}
	info, e := f.Stat()
	if e != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || !privado(info) {
		_ = f.Close()
		return nil, d.ErrDependencia
	}
	fd := f.Fd()
	if fd > 1<<31-1 {
		_ = f.Close()
		return nil, d.ErrDependencia
	}
	descriptor := int(fd)
	for {
		if e = ctx.Err(); e != nil {
			_ = f.Close()
			return nil, e
		}
		e = syscall.Flock(descriptor, syscall.LOCK_EX|syscall.LOCK_NB)
		if e == nil {
			break
		}
		if !errors.Is(e, syscall.EWOULDBLOCK) {
			_ = f.Close()
			return nil, d.ErrDependencia
		}
		select {
		case <-ctx.Done():
			_ = f.Close()
			return nil, ctx.Err()
		case <-time.After(5 * time.Millisecond):
		}
	}
	return func() { _ = syscall.Flock(descriptor, syscall.LOCK_UN); _ = f.Close() }, nil
}
func (a *Archivo) leer() ([]p.Registro, error) {
	dir, e := a.root.Open(".")
	if e != nil {
		return nil, d.ErrDependencia
	}
	entries, e := dir.ReadDir(-1)
	_ = dir.Close()
	if e != nil {
		return nil, d.ErrDependencia
	}
	names := make([]string, 0, len(entries))
	for _, f := range entries {
		if strings.HasSuffix(f.Name(), ".json") {
			names = append(names, f.Name())
		}
	}
	if len(names) > maxVersiones {
		return nil, d.ErrHistoria
	}
	sort.Strings(names)
	history := make([]p.Registro, 0, len(names))
	previous := ""
	var old d.Politica
	for i, name := range names {
		if name != fmt.Sprintf("%020d.json", i+1) {
			return nil, d.ErrHistoria
		}
		f, e := a.root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
		if e != nil {
			return nil, d.ErrHistoria
		}
		stat, e := f.Stat()
		if e != nil || !stat.Mode().IsRegular() || stat.Mode().Perm()&0077 != 0 || !privado(stat) {
			_ = f.Close()
			return nil, d.ErrHistoria
		}
		var r p.Registro
		e = Decodificar(f, &r)
		_ = f.Close()
		if e != nil || r.Version != uint64(i+1) || r.Politica.Validar() != nil || r.SHA256 != r.Politica.SHA256() || r.AnteriorSHA256 != previous || r.RegistroSHA256 != sello(r) || !d.Referencia(r.Actor) || !d.Referencia(r.Correlacion) || r.Instante.IsZero() || r.Instante.Location() != time.UTC || (i > 0 && (old.Referencia != r.Politica.Referencia || r.Instante.Before(history[i-1].Instante))) {
			return nil, d.ErrHistoria
		}
		if r.Revisor != "" && (!d.Referencia(r.Revisor) || r.Revisor == r.Actor) {
			return nil, d.ErrHistoria
		}
		needs := (i == 0 || !reflect.DeepEqual(old.Normalizar().Retencion, r.Politica.Normalizar().Retencion)) && (r.Politica.Retencion.ExigeDobleControl() || i > 0 && old.Retencion.ExigeDobleControl())
		if needs && r.Revisor == "" {
			return nil, d.ErrHistoria
		}
		r.Politica = r.Politica.Normalizar()
		history = append(history, r)
		previous = r.RegistroSHA256
		old = r.Politica
	}
	if e = a.comprobarConfirmacion(history); e != nil {
		return nil, e
	}
	return history, nil
}
func sello(r p.Registro) string {
	r.RegistroSHA256 = ""
	b, _ := json.Marshal(r)
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}
func (a *Archivo) Historia(ctx context.Context) ([]p.Registro, error) {
	unlock, e := a.bloqueo(ctx)
	if e != nil {
		return nil, e
	}
	defer unlock()
	return a.leer()
}
func (a *Archivo) Actual(ctx context.Context) (p.Registro, error) {
	h, e := a.Historia(ctx)
	if e != nil {
		return p.Registro{}, e
	}
	if len(h) == 0 {
		return p.Registro{}, nil
	}
	return h[len(h)-1], nil
}
func (a *Archivo) Guardar(ctx context.Context, expected uint64, policy d.Politica, actor p.Atribucion, at time.Time, recheck p.Revalidacion) (p.Registro, error) {
	if policy.Validar() != nil || !d.Referencia(actor.Actor) || !d.Referencia(actor.Correlacion) || at.IsZero() || recheck == nil {
		return p.Registro{}, d.ErrEntrada
	}
	unlock, e := a.bloqueo(ctx)
	if e != nil {
		return p.Registro{}, e
	}
	defer unlock()
	history, e := a.leer()
	if e != nil {
		return p.Registro{}, e
	}
	if uint64(len(history)) != expected {
		return p.Registro{}, d.ErrVersion
	}
	if len(history) >= maxVersiones {
		return p.Registro{}, d.ErrHistoria
	}
	policy = policy.Normalizar()
	previous := ""
	var old d.Politica
	if len(history) > 0 {
		last := history[len(history)-1]
		if last.Politica.Referencia != policy.Referencia || at.Before(last.Instante) {
			return p.Registro{}, d.ErrEntrada
		}
		previous = last.RegistroSHA256
		old = last.Politica
	}
	needs := (len(history) == 0 || !reflect.DeepEqual(old.Normalizar().Retencion, policy.Retencion)) && (policy.Retencion.ExigeDobleControl() || len(history) > 0 && old.Retencion.ExigeDobleControl())
	if (actor.Revisor != "" && (!d.Referencia(actor.Revisor) || actor.Revisor == actor.Actor)) || (needs && actor.Revisor == "") {
		return p.Registro{}, d.ErrDenegado
	}
	r := p.Registro{Version: expected + 1, Politica: policy, SHA256: policy.SHA256(), Actor: actor.Actor, Revisor: actor.Revisor, Correlacion: actor.Correlacion, Instante: at.UTC(), AnteriorSHA256: previous}
	r.RegistroSHA256 = sello(r)
	if e = recheck(ctx); e != nil {
		return p.Registro{}, e
	}
	if ctx.Err() != nil {
		return p.Registro{}, ctx.Err()
	}
	b, e := json.Marshal(r)
	if e != nil {
		return p.Registro{}, d.ErrEntrada
	}
	// Private pending entry is never mistaken for committed history. File sync,
	// atomic rename and directory sync establish the durable append boundary.
	pending := "pendiente"
	f, e := a.root.OpenFile(pending, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0600)
	if errors.Is(e, os.ErrExist) {
		if e = a.root.Remove(pending); e != nil {
			return p.Registro{}, d.ErrDependencia
		}
		f, e = a.root.OpenFile(pending, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0600)
	}
	if e != nil {
		return p.Registro{}, d.ErrDependencia
	}
	defer func() { _ = a.root.Remove(pending) }()
	_, e = f.Write(b)
	if e == nil {
		e = f.Sync()
	}
	closeErr := f.Close()
	if e != nil || closeErr != nil {
		return p.Registro{}, d.ErrDependencia
	}
	if e = a.root.Rename(pending, fmt.Sprintf("%020d.json", r.Version)); e != nil {
		return p.Registro{}, d.ErrDependencia
	}
	dir, e := a.root.Open(".")
	if e != nil {
		return p.Registro{}, d.ErrDependencia
	}
	e = dir.Sync()
	_ = dir.Close()
	if e != nil {
		return p.Registro{}, d.ErrDependencia
	}
	if e = a.confirmar(r); e != nil {
		return p.Registro{}, e
	}
	return r, nil
}

// ConVersion serializes configuration changes with scheduling. Real platform
// work must be bounded; this implementation holds the control lock until return.
func (a *Archivo) ConVersion(ctx context.Context, v uint64, fn func(context.Context) error) error {
	if fn == nil {
		return d.ErrEntrada
	}
	unlock, e := a.bloqueo(ctx)
	if e != nil {
		return e
	}
	defer unlock()
	h, e := a.leer()
	if e != nil {
		return e
	}
	if len(h) == 0 || uint64(len(h)) != v {
		return d.ErrVersion
	}
	return fn(ctx)
}

func privado(info os.FileInfo) bool {
	s, ok := info.Sys().(*syscall.Stat_t)
	return ok && s.Nlink == 1 && int64(s.Uid) == int64(os.Geteuid())
}
