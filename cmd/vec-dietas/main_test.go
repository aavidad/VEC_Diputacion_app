package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/dietas/application/simulaciondevengo"
)

func fixture(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/ensayo_nacional.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func TestCLIEnsayoJSON(t *testing.T) {
	var out bytes.Buffer
	if code := ejecutar(bytes.NewReader(fixture(t)), &out); code != 0 {
		t.Fatalf("%d %s", code, &out)
	}
	var s simulaciondevengo.Salida
	if err := json.Unmarshal(out.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.Liquidable || s.Resultado.TotalMaximoOrientativoCentimos != 1871 || len(s.Huellas.EntradaSHA256) != 64 {
		t.Fatalf("invalid output: %+v", s)
	}
	if s.Huellas.EntradaSHA256 != "b12c1f14fcabb7e532cdb0900bf4b84e84dd402abf2270b0228871ef396a186f" || s.Huellas.ResultadoSHA256 != "d7d068f85f3566581999fd245f3966448f5ee58691d9f65c5eb39e41b32b67f4" {
		t.Fatal("schema fingerprint changed")
	}
	// Orden de las claves y formato de entrada no cambian la huella tipada.
	var raw map[string]any
	if err := json.Unmarshal(fixture(t), &raw); err != nil {
		t.Fatal(err)
	}
	reordered, _ := json.Marshal(raw)
	out.Reset()
	if ejecutar(bytes.NewReader(reordered), &out) != 0 {
		t.Fatal(out.String())
	}
	var other simulaciondevengo.Salida
	if err := json.Unmarshal(out.Bytes(), &other); err != nil {
		t.Fatal(err)
	}
	if s.Huellas != other.Huellas {
		t.Fatal("format-sensitive hashes")
	}
}
func TestCLIRechazaJSONNoEstricto(t *testing.T) {
	valid := string(fixture(t))
	cases := []string{"", "null", "[]", valid + "{}", valid + "true", strings.Replace(valid, "\"grupo\": 2", "\"grupo\": 2, \"grupo\": 1", 1), strings.Replace(valid, "\"grupo\": 2", "\"grupo\": 2, \"Grupo\": 1", 1), strings.Replace(valid, "\"grupo\": 2", "\"grupo\": null", 1), strings.Replace(valid, "\"grupo\": 2", "\"grupo\": 2.5", 1), strings.Replace(valid, "\"grupo\": 2", "\"grupo\": 2, \"persona_ref\": \"persona:x\"", 1), strings.Replace(valid, "\"liquidable\": false,", "", 1), strings.Repeat(" ", limiteEntrada+1), "{\"esquema\":\"\xff\"}"}
	for n, v := range cases {
		var out bytes.Buffer
		if code := ejecutar(strings.NewReader(v), &out); code != 2 || out.String() != "{\"codigo\":\"entrada_json_invalida\"}\n" {
			t.Errorf("case %d: %d %s", n, code, &out)
		}
	}
}

type failingIO struct{}

func (failingIO) Read([]byte) (int, error)  { return 0, errors.New("test") }
func (failingIO) Write([]byte) (int, error) { return 0, errors.New("test") }
func TestCLIErrorIOYNegocio(t *testing.T) {
	if _, err := leerEntrada(failingIO{}); err == nil {
		t.Fatal("read error ignored")
	}
	if ejecutar(bytes.NewReader(fixture(t)), failingIO{}) != 1 {
		t.Fatal("write error ignored")
	}
	valid := strings.Replace(string(fixture(t)), "\"grupo\": 2", "\"grupo\": 0", 1)
	var out bytes.Buffer
	if ejecutar(strings.NewReader(valid), &out) != 2 || strings.Contains(out.String(), "resultado") {
		t.Fatal(out.String())
	}
}
func TestLimiteExacto(t *testing.T) {
	b := fixture(t)
	b = append(b, bytes.Repeat([]byte(" "), limiteEntrada-len(b))...)
	if ejecutar(bytes.NewReader(b), io.Discard) != 0 {
		t.Fatal("exact limit rejected")
	}
}

func TestArgumentosRechazadosYFalloEscritura(t *testing.T) {
	var out bytes.Buffer
	if ejecutarConArgumentos([]string{"extra"}, failingIO{}, &out) != 2 || out.String() != "{\"codigo\":\"argumentos_no_admitidos\"}\n" {
		t.Fatal(out.String())
	}
	if ejecutarConArgumentos([]string{"extra"}, failingIO{}, failingIO{}) != 1 {
		t.Fatal("write error ignored")
	}
	if ejecutarConArgumentos(nil, bytes.NewReader(fixture(t)), io.Discard) != 0 {
		t.Fatal("valid invocation rejected")
	}
}
