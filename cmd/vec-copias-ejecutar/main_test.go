package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClaveExternaRechazaArchivoCompartidoEnlaceYAlteracion(t *testing.T) {
	raiz := t.TempDir()
	ruta := filepath.Join(raiz, "clave.bin")
	b := make([]byte, 32)
	for i := range b {
		b[i] = byte(i + 1)
	}
	if err := os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
	k, err := clave(ruta)
	if err != nil || k[0] != 1 || k[31] != 32 {
		t.Fatalf("clave privada válida: %v", err)
	}
	clear(k[:])
	if err := os.Chmod(ruta, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := clave(ruta); err == nil {
		t.Fatal("se aceptó una clave legible por terceros")
	}
	if err := os.Chmod(ruta, 0600); err != nil {
		t.Fatal(err)
	}
	enlace := filepath.Join(raiz, "enlace")
	if err := os.Symlink(ruta, enlace); err != nil {
		t.Fatal(err)
	}
	if _, err := clave(enlace); err == nil {
		t.Fatal("se aceptó un enlace como clave")
	}
	if err := os.WriteFile(ruta, b[:31], 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := clave(ruta); err == nil {
		t.Fatal("se aceptó una clave truncada")
	}
}
