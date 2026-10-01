package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestCLIExportaPreparacionDeterminista(t *testing.T) {
	in, err := os.ReadFile("testdata/entrada.json")
	if err != nil {
		t.Fatal(err)
	}
	var a, b, errs bytes.Buffer
	if run(bytes.NewReader(in), &a, &errs) != 0 || run(bytes.NewReader(in), &b, &errs) != 0 {
		t.Fatal(errs.String())
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) || errs.Len() != 0 || !strings.Contains(a.String(), `"estado_global": "pendiente"`) || strings.Contains(a.String(), `"elegible"`) {
		t.Fatal("exportacion incoherente")
	}
	var salida struct {
		Casos []struct {
			Antecedentes struct {
				Convocatoria struct{ Referencia, Fuente, Version string }
				Periodos     []struct{ Inicio, Fin string }
			}
		}
	}
	if err := json.Unmarshal(a.Bytes(), &salida); err != nil {
		t.Fatal(err)
	}
	if len(salida.Casos) != 3 || salida.Casos[2].Antecedentes.Convocatoria.Referencia != "convocatoria-sintetica-modulo-selectivos-A" ||
		salida.Casos[2].Antecedentes.Convocatoria.Fuente != "fuente-sintetica-carrera" || salida.Casos[2].Antecedentes.Convocatoria.Version != "ensayo-1" ||
		len(salida.Casos[1].Antecedentes.Periodos) != 2 {
		t.Fatal("exportacion pierde antecedentes")
	}
}
func TestCLIFallaSinSalidaParcialNiContenidoDeEntrada(t *testing.T) {
	for _, in := range []string{`{"alcance":"preparacion_sintetica","alcance":"produccion"}`, `{"secreto":"dato-no-publicable"}`, `{} {}`, `null`, strings.Repeat(" ", 1024*1024+1)} {
		var out, errs bytes.Buffer
		if run(strings.NewReader(in), &out, &errs) != 1 || out.Len() != 0 || strings.Contains(errs.String(), "dato-no-publicable") {
			t.Fatalf("rechazo incorrecto: %s", errs.String())
		}
	}
}
func TestCLIRechazaMiembrosConMayusculas(t *testing.T) {
	in, err := os.ReadFile("testdata/entrada.json")
	if err != nil {
		t.Fatal(err)
	}
	in = bytes.Replace(in, []byte(`"alcance": "preparacion_sintetica"`),
		[]byte(`"alcance": "preparacion_sintetica", "ALCANCE": "preparacion_sintetica"`), 1)
	var out, errs bytes.Buffer
	if run(bytes.NewReader(in), &out, &errs) != 1 || out.Len() != 0 || !strings.Contains(errs.String(), "carrera.error.json_invalido") {
		t.Fatal("permite claves equivalentes por plegado de mayusculas")
	}
}
