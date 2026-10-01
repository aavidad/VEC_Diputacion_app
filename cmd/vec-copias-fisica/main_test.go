package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestConfiguracionPrivadaNoAceptaTextoExtraClavesDesconocidasONoRegular(t *testing.T) {
	for _, contenido := range []string{"{} {}", "{\"inventado\":1}", "[1]"} {
		ruta := filepath.Join(t.TempDir(), "control.json")
		if e := os.WriteFile(ruta, []byte(contenido), 0600); e != nil {
			t.Fatal(e)
		}
		if _, e := leer(ruta); e == nil {
			t.Fatal("accepted", contenido)
		}
	}
	ruta := filepath.Join(t.TempDir(), "control.json")
	if e := os.WriteFile(ruta, []byte("{}"), 0644); e != nil {
		t.Fatal(e)
	}
	if _, e := leer(ruta); e == nil {
		t.Fatal("non-private config accepted")
	}
	if _, e := leer(t.TempDir()); e == nil {
		t.Fatal("directory accepted")
	}
}
func TestPrecondicionNoDevuelveRutasPrivadas(t *testing.T) {
	var out, err bytes.Buffer
	code := run([]string{"-config", "/synthetic-secret-unavailable.json"}, &out, &err)
	if code != 2 || out.Len() != 0 || err.String() != "captura_fisica_configuracion\n" {
		t.Fatalf("%d %q %q", code, out.String(), err.String())
	}
}
