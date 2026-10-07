package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestCLIProyeccionDeterminista(t *testing.T) {
	b, err := os.ReadFile("ejemplo.json")
	if err != nil {
		t.Fatal(err)
	}
	var anterior []byte
	for i := 0; i < 2; i++ {
		var out, diagnostico bytes.Buffer
		if run(bytes.NewReader(b), &out, &diagnostico) != 0 || diagnostico.Len() != 0 {
			t.Fatalf("falló: %s", diagnostico.String())
		}
		if i == 1 && !bytes.Equal(anterior, out.Bytes()) {
			t.Fatal("bytes no deterministas")
		}
		anterior = bytes.Clone(out.Bytes())
	}
	var salida struct {
		Alcance   string `json:"alcance"`
		Ediciones []struct {
			Referencia string `json:"referencia"`
		} `json:"ediciones"`
		Checklist []struct {
			Estado string `json:"estado"`
		} `json:"checklist"`
	}
	if err := json.Unmarshal(anterior, &salida); err != nil {
		t.Fatal(err)
	}
	if salida.Alcance != "preparacion_sintetica" || len(salida.Ediciones) != 3 || len(salida.Checklist) != 12 {
		t.Fatal("salida inesperada")
	}
	for _, c := range salida.Checklist {
		if c.Estado != "pendiente" {
			t.Fatal("afirmó efecto")
		}
	}
}
func TestCLIFalloNoFiltraEntrada(t *testing.T) {
	var out, diagnostico bytes.Buffer
	if run(strings.NewReader(`{"dato_personal":"no_imprimir"}`), &out, &diagnostico) == 0 || out.Len() != 0 || strings.Contains(diagnostico.String(), "no_imprimir") {
		t.Fatal("fallo filtra entrada")
	}
}
