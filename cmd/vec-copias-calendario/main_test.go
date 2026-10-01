package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIReopensExternalJournalAndPlansOnly(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "control")
	args := []string{"--registro", dir, "--raiz-restaurada", t.TempDir(), "--textos", "../../web/static/textos/es/copias_calendario.json"}
	var out, diag bytes.Buffer
	input, e := os.ReadFile("testdata/configurar.json")
	if e != nil {
		t.Fatal(e)
	}
	if n := ejecutar(args, bytes.NewReader(input), &out, &diag); n != 0 {
		t.Fatal(n, diag.String())
	}
	var result struct {
		HabilitaCopia   bool `json:"habilita_copia"`
		HabilitaBorrado bool `json:"habilita_borrado"`
		Autorizacion    bool `json:"autorizacion_admin"`
		Resultado       struct {
			Version uint64 `json:"version"`
		} `json:"resultado"`
	}
	if json.Unmarshal(out.Bytes(), &result) != nil || result.HabilitaCopia || result.HabilitaBorrado || result.Autorizacion || result.Resultado.Version != 1 {
		t.Fatal(out.String())
	}
	out.Reset()
	diag.Reset()
	if n := ejecutar(args, strings.NewReader(`{"sintetica":true,"accion":"consultar"}`), &out, &diag); n != 0 || !strings.Contains(out.String(), `"version":1`) {
		t.Fatal(n, out.String(), diag.String())
	}
	for _, name := range []string{"agenda", "retencion"} {
		out.Reset()
		diag.Reset()
		input, e = os.ReadFile("testdata/" + name + ".json")
		if e != nil {
			t.Fatal(e)
		}
		if n := ejecutar(args, bytes.NewReader(input), &out, &diag); n != 0 {
			t.Fatal(n, diag.String())
		}
		if name == "retencion" && !strings.Contains(out.String(), `"candidata":false`) {
			t.Fatal("only copy candidate", out.String())
		}
	}
	out.Reset()
	diag.Reset()
	input, e = os.ReadFile("testdata/configurar.json")
	if e != nil {
		t.Fatal(e)
	}
	if n := ejecutar(args, bytes.NewReader(input), &out, &diag); n != 2 || !strings.Contains(diag.String(), "politica_version_distinta") {
		t.Fatal("CAS expected", n, diag.String())
	}
}
func TestCLIDeniesOperationalActionsAndAmbiguousInput(t *testing.T) {
	args := []string{"--registro", filepath.Join(t.TempDir(), "control"), "--raiz-restaurada", t.TempDir(), "--textos", "../../web/static/textos/en/copias_calendario.json"}
	for _, input := range []string{`{"sintetica":false,"accion":"consultar"}`, `{"sintetica":true,"accion":"borrar"}`, `{"sintetica":true,"accion":"capturar"}`, `{"sintetica":true,"sintetica":true,"accion":"consultar"}`, `{"Sintetica":true,"accion":"consultar"}`} {
		var out, diag bytes.Buffer
		if n := ejecutar(args, strings.NewReader(input), &out, &diag); n != 2 {
			t.Fatal(n, input)
		}
		if strings.Contains(diag.String(), args[1]) {
			t.Fatal("path disclosed")
		}
	}
}
