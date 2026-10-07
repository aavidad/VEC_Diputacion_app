package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCLIUnionConsumosAD173YFuentesAD174(t *testing.T) {
	b, err := os.ReadFile("testdata/union_consumos_fuentes_ad173_ad174.json")
	if err != nil {
		t.Fatal(err)
	}
	var d map[string]json.RawMessage
	if json.Unmarshal(b, &d) != nil {
		t.Fatal("fixture")
	}
	ruta := filepath.Join(t.TempDir(), "checkpoint.json")
	if os.WriteFile(ruta, d["manifiesto"], 0600) != nil {
		t.Fatal("checkpoint")
	}
	var salida bytes.Buffer
	if codigo := ejecutar([]string{"-checkpoint", ruta, "-max-bytes", "65536", "-max-registros", "9"}, bytes.NewReader(b), &salida); codigo != 0 ||
		!bytes.Contains(salida.Bytes(), []byte(`"consumos_historicos_sin_fecha_ligada":true`)) || !bytes.Contains(salida.Bytes(), []byte(`"fecha_consumo_ligada_cotejada":true`)) {
		t.Fatalf("CLI unión: %d %s", codigo, salida.String())
	}
}
