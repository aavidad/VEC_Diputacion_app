package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIUsaMismoEjemploQuePDF(t *testing.T) {
	var salida bytes.Buffer
	args := []string{"../../web/static/textos/es/cronos-informe-movimientos-csv.json", "../vec-cronos-informe-saldo/testdata/movimientos.json"}
	if err := ejecutar(context.Background(), args, &salida); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(salida.String(), "Europe/Madrid") || !strings.Contains(salida.String(), "+02:00") || !strings.Contains(salida.String(), "+01:00") || !strings.Contains(salida.String(), "Ejemplo sintético") {
		t.Fatal("CSV incompleto")
	}
	if strings.Contains(salida.String(), "Carmen Molina Ortega") {
		t.Fatal("se exportó un nombre innecesario")
	}
}

func TestCLINoEscribeAnteEntradaAjena(t *testing.T) {
	raw, err := os.ReadFile("../vec-cronos-informe-saldo/testdata/movimientos.json")
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "entrada.json")
	raw = bytes.Replace(raw, []byte(`"demo": true`), []byte(`"demo": false`), 1)
	if err := os.WriteFile(ruta, raw, 0600); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	if err := ejecutar(context.Background(), []string{"../../web/static/textos/es/cronos-informe-movimientos-csv.json", ruta}, &salida); err == nil || salida.Len() != 0 {
		t.Fatal("produjo CSV con entrada ajena", err)
	}
}
