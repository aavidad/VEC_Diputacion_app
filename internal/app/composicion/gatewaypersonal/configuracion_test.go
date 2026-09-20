package gatewaypersonal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOrigenSeguroYFicheroPrivado(t *testing.T) {
	if !origenSeguro("https://auth.example.test") {
		t.Fatal("origen https exacto rechazado")
	}
	for _, s := range []string{"http://auth.example.test", "https://user@auth.example.test", "https://auth.example.test/ruta", "https://auth.example.test?x=1"} {
		if origenSeguro(s) {
			t.Fatalf("origen inseguro admitido: %s", s)
		}
	}
	r := t.TempDir()
	f := filepath.Join(r, "secreto")
	if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !ficheroSeguro(f, true) {
		t.Fatal("secreto 0600 rechazado")
	}
	if err := os.Chmod(f, 0o644); err != nil {
		t.Fatal(err)
	}
	if ficheroSeguro(f, true) {
		t.Fatal("secreto publico admitido")
	}
}
