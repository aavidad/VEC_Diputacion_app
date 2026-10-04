package ejecucioncopias

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"

	cs03 "vec-diputacion-granada/internal/modules/administracion/application/destinocopias"
	puerto "vec-diputacion-granada/internal/modules/administracion/ports/destinocopias"
)

// CatalogoFichero guarda exclusivamente referencias de índices. El contenido
// siempre se abre y autentica mediante CS03 antes de aceptar un conjunto.
type CatalogoFichero struct{ raiz *os.Root }

func AbrirCatalogo(dir string, raicesRestauradas []string) (*CatalogoFichero, error) {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir || dir == "/" || len(raicesRestauradas) == 0 {
		return nil, ErrConjunto
	}
	real, err := filepath.EvalSymlinks(dir)
	if err != nil || real != dir {
		return nil, ErrConjunto
	}
	for _, rr := range raicesRestauradas {
		if !filepath.IsAbs(rr) {
			return nil, ErrConjunto
		}
		r, e := filepath.EvalSymlinks(rr)
		if e != nil {
			return nil, ErrConjunto
		}
		rel, e := filepath.Rel(r, real)
		if e != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return nil, ErrConjunto
		}
	}
	s, err := os.Stat(dir)
	if err != nil || !s.IsDir() || s.Mode().Perm() != 0700 {
		return nil, ErrConjunto
	}
	ss, ok := s.Sys().(*syscall.Stat_t)
	if !ok || ss.Uid != uint32(os.Geteuid()) { // #nosec G115 -- Linux: UID procede del kernel de 32 bits.
		return nil, ErrConjunto
	}
	r, err := abrirCatalogoSinEnlaces(dir)
	if err != nil {
		return nil, ErrConjunto
	}
	pinned, err := r.Open(".")
	if err != nil {
		_ = r.Close()
		return nil, ErrConjunto
	}
	actual, err := pinned.Stat()
	_ = pinned.Close()
	if err != nil || !os.SameFile(s, actual) {
		_ = r.Close()
		return nil, ErrConjunto
	}
	return &CatalogoFichero{raiz: r}, nil
}

func abrirCatalogoSinEnlaces(path string) (*os.Root, error) {
	fd, err := syscall.Open("/", syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, ErrConjunto
	}
	for _, parte := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		next, e := syscall.Openat(fd, parte, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		_ = syscall.Close(fd)
		if e != nil {
			return nil, ErrConjunto
		}
		fd = next
	}
	r, err := os.OpenRoot("/proc/self/fd/" + strconv.Itoa(fd))
	_ = syscall.Close(fd)
	if err != nil {
		return nil, ErrConjunto
	}
	return r, nil
}

func (c *CatalogoFichero) Close() error {
	if c == nil || c.raiz == nil {
		return nil
	}
	return c.raiz.Close()
}
func nombreCatalogo(ref string) string {
	h := sha256.Sum256([]byte(ref))
	return hex.EncodeToString(h[:]) + ".json"
}

type entradaCatalogo struct {
	Version     int               `json:"version"`
	ConjuntoRef string            `json:"conjunto_ref"`
	Indice      puerto.Referencia `json:"indice"`
}

func (c *CatalogoFichero) Crear(ctx context.Context, ref string, indice puerto.Referencia) error {
	if !c.valido(ctx, ref, indice) {
		return ErrConjunto
	}
	return c.bloqueado(ctx, func() error {
		name := nombreCatalogo(ref)
		f, err := c.raiz.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
		if err != nil {
			return ErrConjunto
		}
		b, _ := json.Marshal(entradaCatalogo{1, ref, indice})
		n, writeErr := f.Write(b)
		err = writeErr
		if n != len(b) {
			err = ErrConjunto
		}
		if err == nil {
			err = f.Sync()
		}
		ce := f.Close()
		if err != nil || ce != nil {
			_ = c.raiz.Remove(name)
			return ErrConjunto
		}
		return c.sync()
	})
}

func (c *CatalogoFichero) CAS(ctx context.Context, ref string, anterior, siguiente puerto.Referencia) error {
	if !c.valido(ctx, ref, anterior) || !c.valido(ctx, ref, siguiente) || reflect.DeepEqual(anterior, siguiente) {
		return ErrConjunto
	}
	return c.bloqueado(ctx, func() error {
		actual, err := c.leer(ref)
		if err != nil || !reflect.DeepEqual(actual, anterior) {
			return ErrConjunto
		}
		name := nombreCatalogo(ref)
		var nonce [16]byte
		if _, err := rand.Read(nonce[:]); err != nil {
			return ErrConjunto
		}
		temporal := "tmp-" + hex.EncodeToString(nonce[:])
		f, err := c.raiz.OpenFile(temporal, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
		if err != nil {
			return ErrConjunto
		}
		defer c.raiz.Remove(temporal)
		b, _ := json.Marshal(entradaCatalogo{1, ref, siguiente})
		n, writeErr := f.Write(b)
		err = writeErr
		if n != len(b) {
			err = ErrConjunto
		}
		if err == nil {
			err = f.Sync()
		}
		ce := f.Close()
		if err != nil || ce != nil {
			return ErrConjunto
		}
		if c.raiz.Rename(temporal, name) != nil {
			return ErrConjunto
		}
		return c.sync()
	})
}

func (c *CatalogoFichero) Leer(ctx context.Context, ref string) (puerto.Referencia, error) {
	if c == nil || c.raiz == nil || ctx == nil || ctx.Err() != nil || !cs03.ReferenciaOpaca(ref) {
		return puerto.Referencia{}, ErrConjunto
	}
	var r puerto.Referencia
	err := c.bloqueado(ctx, func() error { var e error; r, e = c.leer(ref); return e })
	return r, err
}

func (c *CatalogoFichero) leer(ref string) (puerto.Referencia, error) {
	f, err := c.raiz.OpenFile(nombreCatalogo(ref), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return puerto.Referencia{}, ErrConjunto
	}
	defer f.Close()
	s, err := f.Stat()
	if err != nil || !s.Mode().IsRegular() || s.Mode().Perm() != 0600 || s.Size() < 1 || s.Size() > 4096 {
		return puerto.Referencia{}, ErrConjunto
	}
	ss, ok := s.Sys().(*syscall.Stat_t)
	if !ok || ss.Nlink != 1 || ss.Uid != uint32(os.Geteuid()) { // #nosec G115 -- Linux: UID procede del kernel de 32 bits.
		return puerto.Referencia{}, ErrConjunto
	}
	b, err := io.ReadAll(io.LimitReader(f, 4097))
	if err != nil || int64(len(b)) != s.Size() {
		return puerto.Referencia{}, ErrConjunto
	}
	var e entradaCatalogo
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(&e) != nil || dec.Decode(new(any)) != io.EOF || e.Version != 1 || e.ConjuntoRef != ref || !c.valido(context.Background(), ref, e.Indice) {
		return puerto.Referencia{}, ErrConjunto
	}
	canon, _ := json.Marshal(e)
	if !bytes.Equal(canon, b) {
		return puerto.Referencia{}, ErrConjunto
	}
	return e.Indice, nil
}

func (c *CatalogoFichero) valido(ctx context.Context, ref string, r puerto.Referencia) bool {
	return c != nil && c.raiz != nil && ctx != nil && ctx.Err() == nil && cs03.ReferenciaOpaca(ref) && r.Vinculo.ConjuntoRef == ref && r.Vinculo.ComponenteRef == "indice" && cs03.ValidarReferencia(r, 2<<30) == nil
}
func (c *CatalogoFichero) bloqueado(ctx context.Context, fn func() error) error {
	if c == nil || c.raiz == nil || ctx == nil || ctx.Err() != nil {
		return ErrConjunto
	}
	f, err := c.raiz.OpenFile("catalogo.lock", os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return ErrConjunto
	}
	defer f.Close()
	s, err := f.Stat()
	if err != nil || !s.Mode().IsRegular() || s.Mode().Perm() != 0600 {
		return ErrConjunto
	}
	ss, ok := s.Sys().(*syscall.Stat_t)
	if !ok || ss.Nlink != 1 || ss.Uid != uint32(os.Geteuid()) { // #nosec G115 -- Linux: UID procede del kernel de 32 bits.
		return ErrConjunto
	}
	for {
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) // #nosec G115 -- Linux: descriptor válido obtenido de Open/Openat; el kernel usa int.
		if err == nil {
			break
		}
		if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN {
			return ErrConjunto
		}
		t := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN) // #nosec G115 -- Linux: descriptor válido obtenido de Open/Openat; el kernel usa int.
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return fn()
}
func (c *CatalogoFichero) sync() error {
	f, e := c.raiz.Open(".")
	if e != nil {
		return ErrConjunto
	}
	defer f.Close()
	if f.Sync() != nil {
		return ErrConjunto
	}
	return nil
}

var _ CatalogoIndice = (*CatalogoFichero)(nil)
