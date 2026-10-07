package main

import (
	"bytes"
	"strings"
	"testing"
)

const declaracionSintetica = `{"tipo":"privada","actividad_ref":"ensayo:00000000000000000000000000000001","funciones_ref":"ensayo:00000000000000000000000000000002","titular_ref":"ensayo:00000000000000000000000000000003","jornada_ref":"ensayo:00000000000000000000000000000004","horario_ref":"ensayo:00000000000000000000000000000005","relacion_con_puesto":"desconocida"}`

func TestCLIValidaDeclaracionSinteticaSinRepetirDatos(t *testing.T) {
	var salida bytes.Buffer
	if codigo := ejecutar(strings.NewReader(declaracionSintetica), &salida); codigo != 0 {
		t.Fatalf("código %d: %s", codigo, salida.String())
	}
	if got, want := salida.String(), "{\"status\":\"structurally_complete\"}\n"; got != want {
		t.Fatalf("respuesta %q, esperada %q", got, want)
	}
}

func TestCLIRechazaCamposAusentesYEntradaAmbigua(t *testing.T) {
	for _, caso := range []struct {
		entrada string
		codigo  int
		status  string
	}{
		{`{"tipo":"privada"}`, 1, `"status":"incomplete"`},
		{declaracionSintetica + declaracionSintetica, 2, `"status":"invalid_input"`},
		{`{"tipo":"privada","desconocido":"dato"}`, 2, `"status":"invalid_input"`},
		{`{"tipo":"privada","tipo":"segunda_publica"}`, 2, `"status":"invalid_input"`},
		{`{"tipo":"privada","actividad_ref":"personal:00000000000000000000000000000001"}`, 2, `"status":"invalid_input"`},
		{strings.Repeat("x", maximoEntrada+1), 2, `"status":"invalid_input"`},
	} {
		var salida bytes.Buffer
		if got := ejecutar(strings.NewReader(caso.entrada), &salida); got != caso.codigo || !strings.Contains(salida.String(), caso.status) {
			t.Fatalf("código %d, respuesta %q", got, salida.String())
		}
		if strings.Contains(salida.String(), "ensayo:00000000000000000000000000000001") {
			t.Fatal("la respuesta repite contenido de la declaración")
		}
	}
}
