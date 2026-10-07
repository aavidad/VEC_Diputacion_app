package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestCLIProduceRegistroSinAprobacion(t *testing.T) {
	datos, err := os.ReadFile("testdata/material.json")
	if err != nil {
		t.Fatal(err)
	}
	var out, diagnostic bytes.Buffer
	if codigo := ejecutar(context.Background(), nil, bytes.NewReader(datos), &out, &diagnostic); codigo != 0 {
		t.Fatalf("código %d: %s", codigo, diagnostic.String())
	}
	var r struct {
		Estado string `json:"estado"`
		Huella string `json:"huella_material_sha256"`
		Notas  []struct {
			Estado string `json:"estado"`
		} `json:"notas"`
		Aprobada bool `json:"aprobada"`
	}
	if err := json.Unmarshal(out.Bytes(), &r); err != nil || r.Estado != "borrador_pendiente_validacion" || r.Notas[1].Estado != "pendiente" || r.Aprobada || len(r.Huella) != 64 {
		t.Fatalf("salida inesperada: %v %s", err, out.String())
	}
}

func TestCLIRechazaClavesDuplicadasYDesconocidas(t *testing.T) {
	for _, datos := range []string{
		`{"esquema":"a","esquema":"b"}`,
		`{"campo_desconocido":1}`,
		strings.Repeat(" ", maximoEntrada+1),
	} {
		var out, diagnostico bytes.Buffer
		if codigo := ejecutar(context.Background(), nil, strings.NewReader(datos), &out, &diagnostico); codigo != 1 || out.Len() != 0 {
			t.Fatalf("código %d, salida %q", codigo, out.String())
		}
	}
}
