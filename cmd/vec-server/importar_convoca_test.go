package main

import (
	"bytes"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestLeerArgumentosImportarConvoca(t *testing.T) {
	a, e := leerArgumentosImportarConvoca([]string{"--fichero", "entrada.xls", "--categoria", "administrativo", "--bolsa-ref", "bolsa:administrativo:2026-09-18", "--admitir-rechazos"}, &bytes.Buffer{})
	if e != nil || a.fichero != "entrada.xls" || a.categoria != "administrativo" || a.bolsaRef == "" || !a.admitirRechazos {
		t.Fatalf("%#v %v", a, e)
	}
}
func TestLeerArgumentosImportarConvocaRechazaObligatorios(t *testing.T) {
	if _, e := leerArgumentosImportarConvoca([]string{"--fichero", "entrada.xls"}, &bytes.Buffer{}); e == nil {
		t.Fatal("acepto categoria ausente")
	}
}

func TestDescribirSustituidas(t *testing.T) {
	if got := describirSustituidas(nil); got != "ninguna" {
		t.Fatalf("sin sustituidas: %q", got)
	}
	got := describirSustituidas([]ports.BolsaSustituida{{BolsaRef: "bolsa:a", VersionBolsa: 1}, {BolsaRef: "bolsa:b", VersionBolsa: 3}})
	if got != "bolsa:a@1,bolsa:b@3" {
		t.Fatalf("sustituidas: %q", got)
	}
}
