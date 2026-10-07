package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"vec-diputacion-granada/internal/modules/dietas/domain"
)

func fixtureLiquidacion(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/preparacion_liquidacion.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func TestCLIPrepararLiquidacion(t *testing.T) {
	var out bytes.Buffer
	b := fixtureLiquidacion(t)
	if ejecutarConArgumentos([]string{"--preparar-liquidacion"}, bytes.NewReader(b), &out) != 0 {
		t.Fatal(out.String())
	}
	var s domain.InstantaneaLiquidacionPropuesta
	if json.Unmarshal(out.Bytes(), &s) != nil || s.Totales.ReconocidoPropuestoCentimos != 4470 || s.Liquidable || s.Procedencia != "propuesta_sin_registrar" || strings.Contains(out.String(), "recibo") {
		t.Fatal(out.String())
	}
	var raw map[string]any
	if json.Unmarshal(b, &raw) != nil {
		t.Fatal("fixture")
	}
	reordenado, _ := json.Marshal(raw)
	out.Reset()
	if ejecutarConArgumentos([]string{"--preparar-liquidacion"}, bytes.NewReader(reordenado), &out) != 0 {
		t.Fatal(out.String())
	}
	var s2 domain.InstantaneaLiquidacionPropuesta
	_ = json.Unmarshal(out.Bytes(), &s2)
	if s2.SnapshotSHA256 != s.SnapshotSHA256 {
		t.Fatal("huella sensible a formato")
	}
	b = append(b, bytes.Repeat([]byte(" "), limiteEntrada-len(b))...)
	if ejecutarPreparacion(bytes.NewReader(b), io.Discard) != 0 {
		t.Fatal("limite exacto")
	}
	b = append(b, ' ')
	if ejecutarPreparacion(bytes.NewReader(b), io.Discard) != 2 {
		t.Fatal("exceso limite")
	}
	if ejecutarPreparacion(bytes.NewReader(fixtureLiquidacion(t)), failingIO{}) != 1 {
		t.Fatal("fallo escritura")
	}
	if ejecutarPreparacion(failingIO{}, io.Discard) != 2 {
		t.Fatal("fallo lectura")
	}
}
func TestCLILiquidacionJSONEstricto(t *testing.T) {
	v := string(fixtureLiquidacion(t))
	casos := []string{"null", "", v + "{}", strings.Replace(v, `"comision_version": 2`, `"comision_version": 2, "comision_version": 2`, 1), strings.Replace(v, `"comision_version": 2`, `"Comision_version": 2`, 1), strings.Replace(v, `"comision_version": 2`, `"comision_version": null`, 1), strings.Replace(v, `"comision_version": 2`, `"comision_version": 2.5`, 1), strings.Replace(v, `"motivo_codigo": ""`, `"motivo_codigo": null`, 1), strings.Replace(v, `"indice_tramo": 0`, `"indice_tramo": null`, 1), strings.Replace(v, `"indice_tramo": 0`, `"indice_tramo": 0, "indice_tramo": 1`, 1), strings.Replace(v, `"comision_version": 2`, `"comision_version": 2, "recibo": "inventado"`, 1), "{\"esquema\":\"\xff\"}"}
	for i, b := range casos {
		var out bytes.Buffer
		if ejecutarPreparacion(strings.NewReader(b), &out) != 2 || out.String() != "{\"codigo\":\"entrada_json_invalida\"}\n" {
			t.Errorf("%d: %s", i, out.String())
		}
	}
}
