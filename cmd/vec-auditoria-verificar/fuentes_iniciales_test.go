package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCLIFuentesInicialesYCamposCruzados(t *testing.T) {
	b, err := os.ReadFile("testdata/fuentes_iniciales_ad174.json")
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
	args := []string{"-checkpoint", ruta, "-max-bytes", "65536", "-max-registros", "2"}
	var salida bytes.Buffer
	if codigo := ejecutar(args, bytes.NewReader(b), &salida); codigo != 0 || !bytes.Contains(salida.Bytes(), []byte(`"material_fuentes_recalculado":true`)) {
		t.Fatalf("CLI: %d %s", codigo, salida.String())
	}
	var registros []map[string]json.RawMessage
	json.Unmarshal(d["registros"], &registros)
	var campos map[string]json.RawMessage
	json.Unmarshal(registros[0]["fuentes_iniciales"], &campos)
	for _, clave := range []string{"actor_ref", "perfil_activo_ref", "sesion_ref", "decision_ref"} {
		t.Run(clave, func(t *testing.T) {
			campos[clave] = json.RawMessage(`"inventado"`)
			registros[0]["fuentes_iniciales"], _ = json.Marshal(campos)
			d["registros"], _ = json.Marshal(registros)
			alterado, _ := json.Marshal(d)
			salida.Reset()
			if codigo := ejecutar(args, bytes.NewReader(alterado), &salida); codigo != 2 {
				t.Fatalf("campo cruzado: %d %s", codigo, salida.String())
			}
			delete(campos, clave)
		})
	}
}

func TestCLIIntentosFuentesSinMaterialFicticio(t *testing.T) {
	b, err := os.ReadFile("testdata/intentos_fuentes_ad174.json")
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
	if codigo := ejecutar(args, bytes.NewReader(b), &salida); codigo != 0 || !bytes.Contains(salida.Bytes(), []byte(`"material_intentos_fuentes_recalculado":true`)) {
		t.Fatalf("CLI: %d %s", codigo, salida.String())
	}
	var registros []map[string]json.RawMessage
	json.Unmarshal(d["registros"], &registros)
	var campos map[string]json.RawMessage
	json.Unmarshal(registros[0]["intento_fuentes_iniciales"], &campos)
	for _, clave := range []string{"fuente_ref", "fuente_sha256", "plan_ref", "preimagen_sha256", "aprobacion_ref", "actor_ref", "perfil_activo_ref"} {
		t.Run(clave, func(t *testing.T) {
			campos[clave] = json.RawMessage(`"inventado"`)
			registros[0]["intento_fuentes_iniciales"], _ = json.Marshal(campos)
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
