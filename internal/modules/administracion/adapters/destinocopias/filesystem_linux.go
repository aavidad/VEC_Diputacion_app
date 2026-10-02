//go:build linux

package destinocopias

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"

	app "vec-diputacion-granada/internal/modules/administracion/application/destinocopias"
	ports "vec-diputacion-granada/internal/modules/administracion/ports/destinocopias"
)

type ConfiguracionFilesystem struct {
	Raiz             string
	MaximoClaroBytes int64
}
type Filesystem struct {
	raiz      *os.Root
	protector ports.Protector
	max       int64
}
type paquete struct {
	Version    int    `json:"version"`
	Manifiesto []byte `json:"manifiesto"`
	Contenido  []byte `json:"contenido"`
}

func NuevoFilesystem(c ConfiguracionFilesystem, p ports.Protector) (*Filesystem, error) {
	if p == nil || (reflect.ValueOf(p).Kind() == reflect.Ptr && reflect.ValueOf(p).IsNil()) || c.MaximoClaroBytes < 1 || c.MaximoClaroBytes > (1<<30) {
		return nil, app.ErrConfiguracion
	}
	raiz, err := abrirRaiz(c.Raiz)
	if err != nil {
		return nil, app.ErrConfiguracion
	}
	return &Filesystem{raiz, p, c.MaximoClaroBytes}, nil
}
func (f *Filesystem) Close() error {
	if f == nil || f.raiz == nil {
		return nil
	}
	return f.raiz.Close()
}
func (f *Filesystem) Publicar(ctx context.Context, s ports.Solicitud) (ports.Referencia, error) {
	if !f.disponible(ctx) || len(s.Manifiesto) == 0 || len(s.Contenido) == 0 || int64(len(s.Manifiesto)) > f.max || int64(len(s.Contenido)) > f.max || app.Huella(s.Manifiesto) != s.Vinculo.ManifiestoSHA256 {
		return ports.Referencia{}, app.ErrMaterial
	}
	id, err := app.Objeto(s.Vinculo)
	if err != nil {
		return ports.Referencia{}, err
	}
	claro, err := json.Marshal(paquete{1, s.Manifiesto, s.Contenido})
	if err != nil || int64(len(claro)) > f.max {
		clear(claro)
		return ports.Referencia{}, app.ErrMaterial
	}
	defer clear(claro)
	cifrado, err := f.protector.Proteger(ctx, s.Vinculo, claro)
	if err != nil {
		return ports.Referencia{}, app.ErrMaterial
	}
	defer clear(cifrado)
	r := ports.Referencia{Vinculo: s.Vinculo, ObjetoRef: id, CifradoSHA256: app.Huella(cifrado), TamanoCifrado: int64(len(cifrado))}
	if app.ValidarReferencia(r, f.max*2+8192) != nil {
		return ports.Referencia{}, app.ErrMaterial
	}
	var random [16]byte
	if _, err = rand.Read(random[:]); err != nil {
		return ports.Referencia{}, app.ErrNoDisponible
	}
	temporal := "tmp-" + hex.EncodeToString(random[:])
	archivo, err := f.raiz.OpenFile(temporal, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return ports.Referencia{}, app.ErrNoDisponible
	}
	defer func() { _ = f.raiz.Remove(temporal) }()
	_, err = io.Copy(archivo, bytes.NewReader(cifrado))
	if err == nil {
		err = archivo.Sync()
	}
	cerrar := archivo.Close()
	if err != nil || cerrar != nil || ctx.Err() != nil {
		return ports.Referencia{}, app.ErrNoDisponible
	}
	// Link publishes complete bytes atomically and exclusively: no overwrite.
	if err = f.raiz.Link(temporal, id); err != nil {
		if os.IsExist(err) {
			return ports.Referencia{}, app.ErrExiste
		}
		return ports.Referencia{}, app.ErrNoDisponible
	}
	if err = f.raiz.Remove(temporal); err != nil {
		return ports.Referencia{}, app.ErrNoDisponible
	}
	if err = f.sync(); err != nil {
		return ports.Referencia{}, app.ErrNoDisponible
	}
	return r, nil
}
func (f *Filesystem) Recuperar(ctx context.Context, r ports.Referencia) (ports.Recuperado, error) {
	if !f.disponible(ctx) || app.ValidarReferencia(r, f.max*2+8192) != nil {
		return ports.Recuperado{}, app.ErrMaterial
	}
	archivo, err := f.raiz.OpenFile(r.ObjetoRef, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return ports.Recuperado{}, app.ErrMaterial
	}
	defer archivo.Close()
	st, err := archivo.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Size() != r.TamanoCifrado || st.Mode().Perm() != 0600 {
		return ports.Recuperado{}, app.ErrMaterial
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok || sys.Nlink != 1 || int64(sys.Uid) != int64(os.Geteuid()) {
		return ports.Recuperado{}, app.ErrMaterial
	}
	cifrado, err := io.ReadAll(io.LimitReader(archivo, r.TamanoCifrado+1))
	if err != nil || int64(len(cifrado)) != r.TamanoCifrado || app.Huella(cifrado) != r.CifradoSHA256 {
		return ports.Recuperado{}, app.ErrMaterial
	}
	defer clear(cifrado)
	claro, err := f.protector.Recuperar(ctx, r.Vinculo, cifrado)
	if err != nil || int64(len(claro)) > f.max {
		clear(claro)
		return ports.Recuperado{}, app.ErrMaterial
	}
	defer clear(claro)
	var p paquete
	d := json.NewDecoder(bytes.NewReader(claro))
	d.DisallowUnknownFields()
	if err = d.Decode(&p); err != nil || d.Decode(new(any)) != io.EOF || p.Version != 1 || len(p.Manifiesto) == 0 || len(p.Contenido) == 0 || app.Huella(p.Manifiesto) != r.Vinculo.ManifiestoSHA256 || ctx.Err() != nil {
		clear(p.Manifiesto)
		clear(p.Contenido)
		return ports.Recuperado{}, app.ErrMaterial
	}
	return ports.Recuperado{Manifiesto: p.Manifiesto, Contenido: p.Contenido}, nil
}
func (f *Filesystem) Comprobar(ctx context.Context, r ports.Referencia) error {
	p, e := f.Recuperar(ctx, r)
	clear(p.Manifiesto)
	clear(p.Contenido)
	return e
}

// Borrar is a storage capability, not a retention policy or ADMIN permission.
// It authenticates the exact reference and object before deleting its leaf.
func (f *Filesystem) Borrar(ctx context.Context, r ports.Referencia) error {
	if err := f.Comprobar(ctx, r); err != nil {
		return err
	}
	if ctx.Err() != nil {
		return app.ErrNoDisponible
	}
	if err := f.raiz.Remove(r.ObjetoRef); err != nil {
		return app.ErrNoDisponible
	}
	return f.sync()
}
func (f *Filesystem) disponible(ctx context.Context) bool {
	return f != nil && f.raiz != nil && f.protector != nil && ctx != nil && ctx.Err() == nil
}
func (f *Filesystem) sync() error {
	d, e := f.raiz.Open(".")
	if e != nil {
		return app.ErrNoDisponible
	}
	defer d.Close()
	if d.Sync() != nil {
		return app.ErrNoDisponible
	}
	return nil
}

// Walk the configured absolute path using no-follow directory descriptors.
// /proc/self/fd opens the pinned directory, never a manifest/client path.
func abrirRaiz(path string) (*os.Root, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == "/" {
		return nil, app.ErrConfiguracion
	}
	fd, err := syscall.Open("/", syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, app.ErrConfiguracion
	}
	for _, part := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		next, e := syscall.Openat(fd, part, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		_ = syscall.Close(fd)
		if e != nil {
			return nil, app.ErrConfiguracion
		}
		fd = next
	}
	d := ficheroDesdeDescriptor(fd, "copias-root")
	if d == nil {
		_ = syscall.Close(fd)
		return nil, app.ErrConfiguracion
	}
	defer d.Close()
	st, err := d.Stat()
	if err != nil || st.Mode().Perm() != 0700 {
		return nil, app.ErrConfiguracion
	}
	s, ok := st.Sys().(*syscall.Stat_t)
	if !ok || int64(s.Uid) != int64(os.Geteuid()) {
		return nil, app.ErrConfiguracion
	}
	// Name is exclusively an integer descriptor created above.
	return os.OpenRoot("/proc/self/fd/" + descriptor(fd))
}
func descriptor(fd int) string {
	const digits = "0123456789"
	if fd == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for fd > 0 {
		i--
		b[i] = digits[fd%10]
		fd /= 10
	}
	return string(b[i:])
}

var _ ports.Destino = (*Filesystem)(nil)

// LeerMaterialLocal reads an operator-selected file without symlink traversal.
// secreto additionally requires private mode; no underlying path reaches errors.
func LeerMaterialLocal(path string, limite int64, secreto bool) ([]byte, error) {
	if limite < 1 || limite > (1<<30) {
		return nil, app.ErrConfiguracion
	}
	absoluto, err := filepath.Abs(path)
	if err != nil {
		return nil, app.ErrMaterial
	}
	parts := strings.Split(strings.TrimPrefix(filepath.Clean(absoluto), "/"), "/")
	fd, err := syscall.Open("/", syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, app.ErrMaterial
	}
	for _, part := range parts[:len(parts)-1] {
		next, e := syscall.Openat(fd, part, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		_ = syscall.Close(fd)
		if e != nil {
			return nil, app.ErrMaterial
		}
		fd = next
	}
	leaf, err := syscall.Openat(fd, parts[len(parts)-1], syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	_ = syscall.Close(fd)
	if err != nil {
		return nil, app.ErrMaterial
	}
	f := ficheroDesdeDescriptor(leaf, "copias-material")
	if f == nil {
		_ = syscall.Close(leaf)
		return nil, app.ErrMaterial
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Size() < 1 || st.Size() > limite || (secreto && st.Mode().Perm() != 0600) {
		return nil, app.ErrMaterial
	}
	s, ok := st.Sys().(*syscall.Stat_t)
	if !ok || s.Nlink != 1 || (secreto && int64(s.Uid) != int64(os.Geteuid())) {
		return nil, app.ErrMaterial
	}
	b, err := io.ReadAll(io.LimitReader(f, limite+1))
	if err != nil || int64(len(b)) != st.Size() {
		clear(b)
		return nil, app.ErrMaterial
	}
	return b, nil
}

func ficheroDesdeDescriptor(fd int, nombre string) *os.File {
	if fd < 0 {
		return nil
	}
	return os.NewFile(uintptr(fd), nombre)
}
