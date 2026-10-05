package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCLIUnidadInicialYCamposCruzados(t *testing.T) {
	b, err := os.ReadFile("testdata/unidad_inicial_ad176.json")
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
	args := []string{"-checkpoint", ruta, "-max-bytes", "65536", "-max-registros", "6"}
	var salida bytes.Buffer
	if codigo := ejecutar(args, bytes.NewReader(b), &salida); codigo != 0 || !bytes.Contains(salida.Bytes(), []byte(`"material_unidad_recalculado":true`)) || !bytes.Contains(salida.Bytes(), []byte(`"material_intentos_unidad_recalculado":true`)) {
		t.Fatalf("CLI: %d %s", codigo, salida.String())
	}
	var registros []map[string]json.RawMessage
	json.Unmarshal(d["registros"], &registros)
	var campos map[string]json.RawMessage
	json.Unmarshal(registros[2]["intento_unidad_inicial"], &campos)
	for _, clave := range []string{"fuente_ref", "recibo_ref", "plan_ref", "aprobacion_ref", "perfil_activo_ref", "actor_ref"} {
		t.Run(clave, func(t *testing.T) {
			campos[clave] = json.RawMessage(`"inventado"`)
			registros[2]["intento_unidad_inicial"], _ = json.Marshal(campos)
			d["registros"], _ = json.Marshal(registros)
			alterado, _ := json.Marshal(d)
			salida.Reset()
			if codigo := ejecutar(args, bytes.NewReader(alterado), &salida); codigo != 2 {
				t.Fatalf("material ficticio: %d %s", codigo, salida.String())
			}
			delete(campos, clave)
		})
	}
}
