package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCLIBootstrapIntentosSinConfirmacionesFicticias(t *testing.T) {
	b, err := os.ReadFile("testdata/bootstrap_intentos_ad179.json")
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
	args := []string{"-checkpoint", ruta, "-max-bytes", "65536", "-max-registros", "4"}
	var salida bytes.Buffer
	if codigo := ejecutar(args, bytes.NewReader(b), &salida); codigo != 0 || !bytes.Contains(salida.Bytes(), []byte(`"material_intentos_bootstrap_recalculado":true`)) {
		t.Fatalf("CLI: %d %s", codigo, salida.String())
	}
	var registros []map[string]json.RawMessage
	json.Unmarshal(d["registros"], &registros)
	var campos map[string]json.RawMessage
	json.Unmarshal(registros[0]["intento_bootstrap_central"], &campos)
	for _, clave := range []string{"actor_ref", "perfil_activo_ref", "plan_sha256", "aprobacion_ref", "fuente_ref", "recibo_ref"} {
		t.Run(clave, func(t *testing.T) {
			campos[clave] = json.RawMessage(`"inventado"`)
			registros[0]["intento_bootstrap_central"], _ = json.Marshal(campos)
			d["registros"], _ = json.Marshal(registros)
			alterado, _ := json.Marshal(d)
			salida.Reset()
			if codigo := ejecutar(args, bytes.NewReader(alterado), &salida); codigo != 2 {
				t.Fatalf("campo ajeno: %d %s", codigo, salida.String())
			}
			delete(campos, clave)
		})
	}
}

func TestCLIUnionBootstrapPOST173(t *testing.T) {
	b, err := os.ReadFile("testdata/union_consumos_tecnicos_bootstrap_ad173_ad179.json")
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
	if codigo := ejecutar([]string{"-checkpoint", ruta, "-max-bytes", "65536", "-max-registros", "19"}, bytes.NewReader(b), &salida); codigo != 0 ||
		!bytes.Contains(salida.Bytes(), []byte(`"material_intentos_bootstrap_recalculado":true`)) || !bytes.Contains(salida.Bytes(), []byte(`"fecha_consumo_ligada_cotejada":true`)) {
		t.Fatalf("CLI unión: %d %s", codigo, salida.String())
	}
}
