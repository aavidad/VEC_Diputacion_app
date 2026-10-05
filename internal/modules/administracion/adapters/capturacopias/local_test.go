package capturacopias

import (
	"context"
	"os"
	"testing"
)

func TestExclusionSimultaneaPorOrigenYLiberacion(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	r, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	e := ExclusorLocal{Raiz: r}
	liberar, err := e.Adquirir(context.Background(), "origen:1")
	if err != nil {
		t.Fatal(err)
	}
	if l, err := e.Adquirir(context.Background(), "origen:1"); err == nil {
		_ = l()
		t.Fatal("admitió captura simultánea")
	}
	if err := liberar(); err != nil {
		t.Fatal(err)
	}
	l, err := e.Adquirir(context.Background(), "origen:1")
	if err != nil {
		t.Fatal(err)
	}
	_ = l()
}

func TestArchivoNoSobrescribeYLímiteNoDejaArchivoPublicable(t *testing.T) {
	r, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	f, err := r.OpenFile("existente", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	p := PostgreSQL{Raiz: r}
	if _, err := p.archivo(context.Background(), "existente", "x", "x", Comando{}, 1); err == nil {
		t.Fatal("sobrescribe")
	}
	f, err = r.OpenFile("limite", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := acotado{f: f, restantes: 2}
	if n, err := w.Write([]byte("123")); err == nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}
