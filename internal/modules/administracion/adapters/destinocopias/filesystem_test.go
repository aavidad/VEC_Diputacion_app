package destinocopias_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	adapter "vec-diputacion-granada/internal/modules/administracion/adapters/destinocopias"
	app "vec-diputacion-granada/internal/modules/administracion/application/destinocopias"
	ports "vec-diputacion-granada/internal/modules/administracion/ports/destinocopias"
)

const max int64 = 65536

func nuevo(t *testing.T, dir string, k byte) *adapter.Filesystem {
	t.Helper()
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	var clave [32]byte
	for i := range clave {
		clave[i] = k
	}
	p, e := adapter.NuevoProtectorJWE(clave, "clave:fixture", "v1", max)
	if e != nil {
		t.Fatal(e)
	}
	f, e := adapter.NuevoFilesystem(adapter.ConfiguracionFilesystem{Raiz: dir, MaximoClaroBytes: max}, p)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}
func solicitud() ports.Solicitud {
	m := []byte(`{"componentes":[{"id":"fixture","tipo":"sintetico"}]}`)
	return ports.Solicitud{Vinculo: ports.Vinculo{ConjuntoRef: "conjunto:fixture", ComponenteRef: "componente:fixture", Posicion: 1, ManifiestoSHA256: app.Huella(m)}, Manifiesto: m, Contenido: []byte("contenido sintetico CS03")}
}
func TestRoundtripReinicioBorradoYNoSobrescritura(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s := solicitud()
	f := nuevo(t, dir, 1)
	r, e := f.Publicar(ctx, s)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.Publicar(ctx, s); e != app.ErrExiste {
		t.Fatalf("overwrite: %v", e)
	}
	_ = f.Close()
	f = nuevo(t, dir, 1)
	p, e := f.Recuperar(ctx, r)
	if e != nil || !bytes.Equal(p.Contenido, s.Contenido) || !bytes.Equal(p.Manifiesto, s.Manifiesto) {
		t.Fatalf("roundtrip: %v", e)
	}
	onDisk, e := os.ReadFile(filepath.Join(dir, r.ObjetoRef))
	if e != nil || bytes.Contains(onDisk, s.Contenido) || bytes.Contains(onDisk, s.Manifiesto) {
		t.Fatal("plaintext")
	}
	if e = f.Borrar(ctx, r); e != nil {
		t.Fatal(e)
	}
	if e = f.Comprobar(ctx, r); e == nil {
		t.Fatal("deleted object accepted")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatal("temporary residue")
	}
}
func TestManipulacionTruncadoClaveYVinculos(t *testing.T) {
	for _, modo := range []string{"tamper", "truncado", "clave", "manifiesto", "componente", "posicion", "referencia", "digest"} {
		t.Run(modo, func(t *testing.T) {
			dir := t.TempDir()
			ctx := context.Background()
			f := nuevo(t, dir, 1)
			r, e := f.Publicar(ctx, solicitud())
			if e != nil {
				t.Fatal(e)
			}
			path := filepath.Join(dir, r.ObjetoRef)
			b, _ := os.ReadFile(path)
			switch modo {
			case "tamper":
				var j map[string]any
				if e = json.Unmarshal(b, &j); e != nil {
					t.Fatal(e)
				}
				c, e := base64.RawURLEncoding.DecodeString(j["ciphertext"].(string))
				if e != nil {
					t.Fatal(e)
				}
				c[len(c)/2] ^= 1
				j["ciphertext"] = base64.RawURLEncoding.EncodeToString(c)
				b, e = json.Marshal(j)
				if e != nil {
					t.Fatal(e)
				}
				_ = os.WriteFile(path, b, 0600)
				r.TamanoCifrado = int64(len(b))
				r.CifradoSHA256 = app.Huella(b)
			case "truncado":
				b = b[:len(b)-1]
				_ = os.WriteFile(path, b, 0600)
				r.CifradoSHA256 = app.Huella(b)
				r.TamanoCifrado = int64(len(b))
			case "clave":
				f = nuevo(t, dir, 2)
			case "manifiesto":
				r.Vinculo.ManifiestoSHA256 = app.Huella([]byte("otro"))
			case "componente":
				r.Vinculo.ComponenteRef = "componente:otro"
			case "posicion":
				r.Vinculo.Posicion = 2
			case "referencia":
				r.ObjetoRef = "../fuera"
			case "digest":
				r.CifradoSHA256 = app.Huella([]byte("otro"))
			}
			if modo == "manifiesto" || modo == "componente" || modo == "posicion" {
				r.ObjetoRef, e = app.Objeto(r.Vinculo)
				if e != nil {
					t.Fatal(e)
				}
				if e = os.WriteFile(filepath.Join(dir, r.ObjetoRef), b, 0600); e != nil {
					t.Fatal(e)
				}
			}
			if e = f.Comprobar(ctx, r); e == nil {
				t.Fatal("altered object accepted")
			}
			if e = f.Borrar(ctx, r); e == nil {
				t.Fatal("altered object deleted")
			}
			if _, e = os.Stat(path); e != nil {
				t.Fatal("original removed")
			}
		})
	}
}
func TestSymlinksHardlinksFIFOYLimites(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	f := nuevo(t, dir, 1)
	s := solicitud()
	r, e := f.Publicar(ctx, s)
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, r.ObjetoRef)
	outside := filepath.Join(t.TempDir(), "outside")
	b, _ := os.ReadFile(path)
	_ = os.WriteFile(outside, b, 0600)
	_ = os.Remove(path)
	if e = os.Symlink(outside, path); e != nil {
		t.Fatal(e)
	}
	if f.Comprobar(ctx, r) == nil || f.Borrar(ctx, r) == nil {
		t.Fatal("symlink accepted")
	}
	_ = os.Remove(path)
	_ = os.Link(outside, path)
	if f.Comprobar(ctx, r) == nil {
		t.Fatal("hardlink accepted")
	}
	_ = os.Remove(path)
	if e = syscall.Mkfifo(path, 0600); e != nil {
		t.Fatal(e)
	}
	if f.Comprobar(ctx, r) == nil {
		t.Fatal("FIFO accepted")
	}
	_ = os.Remove(path)
	s.Contenido = bytes.Repeat([]byte{1}, int(max))
	if _, e = f.Publicar(ctx, s); e == nil {
		t.Fatal("oversize accepted")
	}
	s = solicitud()
	s.Vinculo.ConjuntoRef = "../../outside"
	if _, e = f.Publicar(ctx, s); e == nil {
		t.Fatal("traversal accepted")
	}
	linked := filepath.Join(t.TempDir(), "linked")
	_ = os.Symlink(dir, linked)
	p, _ := adapter.NuevoProtectorJWE([32]byte{1}, "clave:fixture", "v1", max)
	if _, e = adapter.NuevoFilesystem(adapter.ConfiguracionFilesystem{Raiz: linked, MaximoClaroBytes: max}, p); e == nil {
		t.Fatal("root symlink accepted")
	}
}
func TestConcurrentPublishOneWinner(t *testing.T) {
	f := nuevo(t, t.TempDir(), 1)
	ctx := context.Background()
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := f.Publicar(ctx, solicitud()); results <- e }()
	}
	wg.Wait()
	close(results)
	success := 0
	for e := range results {
		if e == nil {
			success++
		} else if e != app.ErrExiste {
			t.Fatal(e)
		}
	}
	if success != 1 {
		t.Fatalf("successes %d", success)
	}
}
func TestCancellationNoEffects(t *testing.T) {
	dir := t.TempDir()
	f := nuevo(t, dir, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := f.Publicar(ctx, solicitud()); e == nil {
		t.Fatal("cancel accepted")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatal("cancel wrote")
	}
}

func TestPublicacionInterrumpidaRechazaHardlinkYRecuperaTrasRetiradaTemporal(t *testing.T) {
	dir := t.TempDir()
	f := nuevo(t, dir, 1)
	r, e := f.Publicar(context.Background(), solicitud())
	if e != nil {
		t.Fatal(e)
	}
	objeto := filepath.Join(dir, r.ObjetoRef)
	temporal := filepath.Join(dir, "tmp-interrupcion-fixture")
	if e = os.Link(objeto, temporal); e != nil {
		t.Fatal(e)
	}
	if f.Comprobar(context.Background(), r) == nil {
		t.Fatal("interrupted publication accepted")
	}
	a, e := os.Lstat(objeto)
	if e != nil {
		t.Fatal(e)
	}
	b, e := os.Lstat(temporal)
	if e != nil || !os.SameFile(a, b) {
		t.Fatal("temporary does not match object")
	}
	if e = os.Remove(temporal); e != nil {
		t.Fatal(e)
	}
	if e = f.Comprobar(context.Background(), r); e != nil {
		t.Fatal(e)
	}
}
