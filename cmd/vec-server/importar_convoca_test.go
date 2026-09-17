package main

import (
	"bytes"
	"testing"
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
