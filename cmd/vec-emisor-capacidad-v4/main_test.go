package main

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestCerrarYRetirarSocketRetiraSoloElPropio(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "emisor.sock")
	escucha, err := net.Listen("unix", ruta)
	if err != nil {
		t.Fatalf("abrir socket Unix: %v", err)
	}
	propio, err := os.Lstat(ruta)
	if err != nil {
		t.Fatalf("leer socket Unix: %v", err)
	}

	cerrarYRetirarSocket(escucha, ruta, propio)
	if _, err = os.Lstat(ruta); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("el socket propio no fue retirado: %v", err)
	}
}

func TestCerrarYRetirarSocketConservaUnReemplazo(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "emisor.sock")
	escucha, err := net.Listen("unix", ruta)
	if err != nil {
		t.Fatalf("abrir socket Unix: %v", err)
	}
	propio, err := os.Lstat(ruta)
	if err != nil {
		t.Fatalf("leer socket Unix: %v", err)
	}
	if err = os.Remove(ruta); err != nil {
		t.Fatalf("retirar socket para la prueba: %v", err)
	}
	if err = os.WriteFile(ruta, []byte("reemplazo"), 0o600); err != nil {
		t.Fatalf("crear reemplazo: %v", err)
	}

	cerrarYRetirarSocket(escucha, ruta, propio)
	contenido, err := os.ReadFile(ruta)
	if err != nil || string(contenido) != "reemplazo" {
		t.Fatalf("se retiro o altero un fichero ajeno: contenido=%q error=%v", contenido, err)
	}
}
