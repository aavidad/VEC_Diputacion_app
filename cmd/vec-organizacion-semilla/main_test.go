package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrepararSemillaNoConectaNiSustituyeDatos(t *testing.T) {
	var salida bytes.Buffer
	if err := prepararSemilla(&salida, "../../data/catalogos/estructura-organizativa/v1.rpt-publica.json"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(salida.String(), "inicializar_organizacion_preparatoria_v1") || strings.Contains(salida.String(), "DELETE") || strings.Contains(salida.String(), "UPDATE") {
		t.Fatal("SQL de semilla inesperado")
	}
	if !strings.Contains(salida.String(), "SET LOCAL ROLE vec_contratacion_temporal_propietario;") {
		t.Fatal("la semilla debe usar el rol propietario como las migraciones")
	}
	salida.Reset()
	if err := prepararSemilla(&salida, "archivo-inexistente.json"); err == nil || salida.Len() != 0 {
		t.Fatal("no debe emitir SQL de una semilla inválida")
	}
}
