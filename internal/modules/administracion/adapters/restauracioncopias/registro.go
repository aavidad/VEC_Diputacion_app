// Package restauracioncopias conserva declaraciones offline fuera del destino.
// Este adaptador no es autorización V3 ni sirve para sustituciones operativas.
package restauracioncopias

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"sync"
	"syscall"

	d "vec-diputacion-granada/internal/modules/administracion/domain/restauracioncopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/restauracioncopias"
)

var ErrRegistro = errors.New("copias_restauracion_registro_no_valido")
var ErrConflicto = errors.New("copias_restauracion_cas_conflicto")

// Revalidar se invoca dentro de la exclusión antes de cada lectura/escritura.
// La CLI declara offline y no proporciona una concesión de producción.
type Revalidar func(context.Context, p.Concesion) error

type RegistroArchivo struct {
	raiz    *os.Root
	validar Revalidar
	mu      sync.Mutex
}
type evento struct {
	Anterior string     `json:"anterior"`
	Registro p.Registro `json:"registro"`
	SHA256   string     `json:"sha256"`
}

func Abrir(directorio string, revalidar Revalidar) (*RegistroArchivo, error) {
	if revalidar == nil {
		return nil, ErrRegistro
	}
	// Ruta configurada exclusivamente por operador CLI, nunca por HTTP.
	raiz, err := os.OpenRoot(directorio)
	if err != nil {
		return nil, ErrRegistro
	}
	info, err := raiz.Stat(".")
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		_ = raiz.Close()
		return nil, ErrRegistro
	}
	return &RegistroArchivo{raiz: raiz, validar: revalidar}, nil
}
func (r *RegistroArchivo) Close() error { r.mu.Lock(); defer r.mu.Unlock(); return r.raiz.Close() }
func (r *RegistroArchivo) transaccion(ctx context.Context, c p.Concesion, fn func(map[string]p.Registro) (*p.Registro, error)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, err := r.raiz.OpenFile("propuestas.v1.jsonl", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return ErrRegistro
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 8<<20 || info.Sys().(*syscall.Stat_t).Nlink != 1 {
		return ErrRegistro
	}
	fd := f.Fd()
	if fd > math.MaxInt32 {
		return ErrRegistro
	}
	lockFD := int(fd)
	if err = syscall.Flock(lockFD, syscall.LOCK_EX); err != nil {
		return ErrRegistro
	}
	defer func() { _ = syscall.Flock(lockFD, syscall.LOCK_UN) }()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	// Re-stat after locking: another process may have appended while waiting.
	info, err = f.Stat()
	if err != nil || info.Size() > 8<<20 {
		return ErrRegistro
	}
	registros := map[string]p.Registro{}
	anterior := ""
	scanner := bufio.NewScanner(io.LimitReader(f, 8<<20+1))
	scanner.Buffer(make([]byte, 4096), 128<<10)
	for scanner.Scan() {
		var e evento
		if json.Unmarshal(scanner.Bytes(), &e) != nil || !d.SHA256Valida(e.SHA256) || !d.SHA256Valida(e.Registro.Propuesta.SHA256) || e.Anterior != anterior || e.SHA256 != d.Huella("vec-restauracion-journal-v1", struct {
			Anterior string
			Registro p.Registro
		}{e.Anterior, e.Registro}) || !d.Valida(e.Registro.Propuesta.Propuesta) || e.Registro.Propuesta.SHA256 != d.Huella("vec-restauracion-propuesta-v1", e.Registro.Propuesta.Propuesta) || e.Registro.Propuesta.Propuesta.Entorno != "sintetico_offline" {
			return ErrRegistro
		}
		previo, exists := registros[e.Registro.Propuesta.Propuesta.Ref]
		if !exists && e.Registro.Version != 1 || exists && (e.Registro.Version != previo.Version+1 || e.Registro.Propuesta != previo.Propuesta || previo.Revision != nil || e.Registro.Revision == nil) || e.Registro.Version == 1 && e.Registro.Revision != nil {
			return ErrRegistro
		}
		if e.Registro.Revision != nil && e.Registro.Propuesta.ComprobarRevision(*e.Registro.Revision, e.Registro.Revision.Fecha) != nil {
			return ErrRegistro
		}
		registros[e.Registro.Propuesta.Propuesta.Ref] = e.Registro
		anterior = e.SHA256
	}
	if scanner.Err() != nil {
		return ErrRegistro
	}
	// Unterminated append is a torn record, even if JSON itself happens to parse.
	if info.Size() > 0 {
		var last [1]byte
		if _, err = f.ReadAt(last[:], info.Size()-1); err != nil || last[0] != '\n' {
			return ErrRegistro
		}
	}
	if err = r.validar(ctx, c); err != nil {
		return err
	}
	nuevo, err := fn(registros)
	if err != nil || nuevo == nil {
		return err
	}
	e := evento{Anterior: anterior, Registro: *nuevo}
	e.SHA256 = d.Huella("vec-restauracion-journal-v1", struct {
		Anterior string
		Registro p.Registro
	}{e.Anterior, e.Registro})
	if !d.SHA256Valida(e.SHA256) {
		return ErrRegistro
	}
	b, err := json.Marshal(e)
	if err != nil {
		return ErrRegistro
	}
	b = append(b, '\n')
	if info.Size()+int64(len(b)) > 8<<20 {
		return ErrRegistro
	}
	if _, err = f.Seek(0, io.SeekEnd); err != nil {
		return ErrRegistro
	}
	n, err := f.Write(b)
	if err != nil || n != len(b) {
		return ErrRegistro
	}
	if f.Sync() != nil {
		return ErrRegistro
	}
	// fsync directory also makes first journal creation durable.
	dir, err := r.raiz.Open(".")
	if err != nil {
		return ErrRegistro
	}
	err = errors.Join(dir.Sync(), dir.Close())
	if err != nil {
		return ErrRegistro
	}
	return nil
}
func acceso(c p.Concesion, accion p.Accion, s d.Sellada) bool {
	return c.Ref != "" && c.PersonaRef != "" && c.Acceso.Accion == accion && c.Acceso.DestinoRef == s.Propuesta.DestinoRef && c.Acceso.PropuestaSHA256 == s.SHA256
}
func (r *RegistroArchivo) Crear(ctx context.Context, s d.Sellada, c p.Concesion) (p.Registro, error) {
	var out p.Registro
	if s.Propuesta.Entorno != "sintetico_offline" || !acceso(c, p.Proponer, s) || c.PersonaRef != s.Propuesta.ProponentePersonaRef || s.Comprobar(s.Propuesta.Creada) != nil {
		return out, ErrRegistro
	}
	err := r.transaccion(ctx, c, func(rs map[string]p.Registro) (*p.Registro, error) {
		if previo, ok := rs[s.Propuesta.Ref]; ok {
			if previo.Propuesta != s {
				return nil, ErrConflicto
			}
			out = previo
			return nil, nil
		}
		out = p.Registro{Propuesta: s, Version: 1}
		return &out, nil
	})
	return out, err
}
func (r *RegistroArchivo) Leer(ctx context.Context, ref string, c p.Concesion) (p.Registro, error) {
	var out p.Registro
	err := r.transaccion(ctx, c, func(rs map[string]p.Registro) (*p.Registro, error) {
		var ok bool
		out, ok = rs[ref]
		if !ok || !acceso(c, c.Acceso.Accion, out.Propuesta) || (c.Acceso.Accion != p.Proponer && c.Acceso.Accion != p.Revisar) {
			return nil, ErrRegistro
		}
		return nil, nil
	})
	return out, err
}
func (r *RegistroArchivo) RevisarCAS(ctx context.Context, ref string, version uint64, sello string, rev d.Revision, c p.Concesion) (p.Registro, error) {
	var out p.Registro
	err := r.transaccion(ctx, c, func(rs map[string]p.Registro) (*p.Registro, error) {
		previo, ok := rs[ref]
		if !ok || previo.Version != version || previo.Propuesta.SHA256 != sello || previo.Revision != nil {
			return nil, ErrConflicto
		}
		if !acceso(c, p.Revisar, previo.Propuesta) || c.PersonaRef != rev.PersonaRef || previo.Propuesta.ComprobarRevision(rev, rev.Fecha) != nil {
			return nil, ErrRegistro
		}
		out = p.Registro{Propuesta: previo.Propuesta, Version: previo.Version + 1, Revision: &rev}
		return &out, nil
	})
	return out, err
}
func (r *RegistroArchivo) Cercar(context.Context, string, uint64, string, string, p.Concesion) error {
	return ErrRegistro
}
