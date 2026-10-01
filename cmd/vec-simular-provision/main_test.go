package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"vec-diputacion-granada/internal/modules/provision/domain"
)

func fixture(t *testing.T) []byte {
	t.Helper()
	datos, err := os.ReadFile("testdata/proceso.sintetico.json")
	if err != nil {
		t.Fatal(err)
	}
	return datos
}
func TestCLIEjecutaStdinYArchivo(t *testing.T) {
	for _, args := range [][]string{nil, {"-entrada", "testdata/proceso.sintetico.json"}} {
		var out, errout bytes.Buffer
		if code := ejecutar(args, bytes.NewReader(fixture(t)), &out, &errout); code != 0 {
			t.Fatal(code, errout.String())
		}
		var r domain.ResultadoProceso
		if err := json.Unmarshal(out.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		if r.Alcance != "simulacion" || r.Estado != "borrador" || len(r.Valoraciones) != 2 || r.Valoraciones[0].Resultado.Total == nil {
			t.Fatal("salida incompleta")
		}
	}
}
func TestCLIContratoEstricto(t *testing.T) {
	datos := string(fixture(t))
	casos := []string{
		strings.Replace(datos, `"proceso":`, `"proceso":null,"proceso":`, 1),
		strings.Replace(datos, `"proceso":`, `"Proceso":`, 1),
		strings.Replace(datos, `"proceso":`, `"nombre": "persona", "proceso":`, 1),
		datos + "{}",
		strings.Replace(datos, `"condicion_interna": "cumple"`, `"condicion_interna": null`, 1),
		strings.Repeat(" ", (2<<20)+1),
	}
	for i, entrada := range casos {
		var out, errout bytes.Buffer
		if code := ejecutar(nil, strings.NewReader(entrada), &out, &errout); code != 1 || out.Len() != 0 || !strings.Contains(errout.String(), `"codigo"`) {
			t.Fatalf("caso %d: %d %s %s", i, code, out.String(), errout.String())
		}
	}
}
func TestCLIErroresNoRevelanRuta(t *testing.T) {
	var out, errout bytes.Buffer
	if code := ejecutar([]string{"-entrada", "/ruta-privada/no-existe.json"}, bytes.NewReader(nil), &out, &errout); code != 1 || strings.Contains(errout.String(), "ruta-privada") {
		t.Fatal(code, errout.String())
	}
	if code := ejecutar([]string{"-entrada", "testdata"}, bytes.NewReader(nil), &out, &errout); code != 1 {
		t.Fatal("acepta directorio")
	}
}
