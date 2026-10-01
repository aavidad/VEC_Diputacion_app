package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	fisica "vec-diputacion-granada/internal/modules/administracion/adapters/capturafisica"
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

func TestDiagnosticoNominalOcultaPathErrorsYErroresUnidos(t *testing.T) {
	privado := &os.PathError{Op: "sync", Path: "/synthetic-private-destination/component.tar", Err: syscall.ENOSPC}
	for _, caso := range []struct {
		err    error
		codigo string
	}{
		{privado, "captura_fisica_origen"},
		{errors.Join(privado, fisica.ErrControl), "captura_fisica_control"},
		{errors.Join(privado, context.Canceled), "captura_fisica_cancelada"},
		{errors.Join(privado, context.DeadlineExceeded), "captura_fisica_tiempo_agotado"},
	} {
		var salida bytes.Buffer
		informarError(&salida, caso.err)
		if salida.String() != caso.codigo+"\n" {
			t.Fatalf("diagnostico inesperado %q", salida.String())
		}
	}
}
