package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func TestComprobarDietasCodigos(t *testing.T) {
	var llamadas int
	ok := func(context.Context, config.Config) error { llamadas++; return nil }
	var out, errOut bytes.Buffer
	if c := ejecutarComprobacionDietas(context.Background(), nil, &out, &errOut, config.Config{}, ok); c != 0 || llamadas != 1 || !strings.Contains(out.String(), "OK") {
		t.Fatalf("exito: codigo=%d llamadas=%d salida=%q", c, llamadas, out.String())
	}
	if c := ejecutarComprobacionDietas(context.Background(), []string{"--aplicar"}, &out, &errOut, config.Config{}, ok); c != 2 || llamadas != 1 {
		t.Fatalf("argumentos: codigo=%d llamadas=%d", c, llamadas)
	}
	if c := ejecutarComprobacionDietas(context.Background(), nil, &out, &errOut, config.Config{}, nil); c != 2 {
		t.Fatalf("sin comprobador: codigo=%d", c)
	}
	errOut.Reset()
	fallo := func(context.Context, config.Config) error { return errors.New("etapa X") }
	if c := ejecutarComprobacionDietas(context.Background(), nil, &out, &errOut, config.Config{}, fallo); c != 1 || !strings.Contains(errOut.String(), "NO: etapa X") {
		t.Fatalf("fallo: codigo=%d errores=%q", c, errOut.String())
	}
}
