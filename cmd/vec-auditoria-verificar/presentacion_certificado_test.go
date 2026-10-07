package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/auditoria"
)

func TestCLIPresentacionCertificadoCamposYNullCerrados(t *testing.T) {
	b, err := os.ReadFile("../../internal/vec/auditoria/testdata/presentacion_certificado_ad221_sintetico.json")
	if err != nil {
		t.Fatal(err)
	}
	var objeto map[string]json.RawMessage
	if err := json.Unmarshal(b, &objeto); err != nil {
		t.Fatal(err)
	}
	checkpoint := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := os.WriteFile(checkpoint, objeto["manifiesto"], 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"--checkpoint", checkpoint, "--max-bytes", "32768", "--max-registros", "5"}
	casos := map[string]struct {
		buscar, cambiar string
		codigo          int
	}{
		"valido":                       {codigo: 0},
		"fecha_jsonb_offset":           {`"presentacion_valida_hasta": "2026-10-07T09:00:01.000001Z"`, `"presentacion_valida_hasta": "2026-10-07T10:00:01.000001+01:00"`, 0},
		"perfil_extra":                 {`"fase":`, `"perfil_activo_ref":"SECRET","fase":`, 2},
		"decision_extra":               {`"fase":`, `"decision_ref":"SECRET","fase":`, 2},
		"nulo_no_admitido":             {`"recibo_sha256": "1afcc20efc560b2e3807a9f3f6d9cd6db3528d96cb52007e399925bba086fafd"`, `"recibo_sha256": null`, 2},
		"nulo_actual_antes_revocacion": {`"canal_sha256": "baf3b3ab0f476dc34c2e68977ac0e58cd725a8f4943b92592c241c7ca6d55ab9"`, `"canal_sha256": null`, 2},
		"actual_en_revocacion":         {`"acr_actual": null`, `"acr_actual": "urn:vec:acr:certificado-desarrollo-protegido"`, 2},
		"actor_humano_en_control_is2":  {`"actor_cuenta_ref": null`, `"actor_cuenta_ref": "cta_abcdefghijklmnopqrstuv"`, 2},
		"control_en_nominal":           {`"control_sesion_ref": null`, `"control_sesion_ref": "cse_abcdefghijklmnopqrstuv"`, 2},
		"control_ausente_en_is2":       {`"control_sesion_ref": "cse_0273677fe6605dbb433b5cdffb9d7aa2"`, `"control_sesion_ref": null`, 2},
		"fase_is2_divergente":          {`"fase": "sin_actor_humano_nominal"`, `"fase": "pre_f1_sin_perfil_activo"`, 1},
		"revision_is2_divergente":      {`"control_sesion_revision": "7"`, `"control_sesion_revision": "07"`, 1},
		"campo_ausente":                {`"presentacion_ref": "prs_83a7f4da2b3c0475ab975539040f95ae",`, ``, 2},
		"clave_duplicada":              {`"tipo": "apertura"`, `"tipo": "reanudacion", "tipo": "apertura"`, 2},
		"recibo_divergente":            {`"recibo_sha256": "1afcc20efc560b2e3807a9f3f6d9cd6db3528d96cb52007e399925bba086fafd"`, `"recibo_sha256": "` + strings.Repeat("a", 64) + `"`, 1},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			entrada := string(b)
			if caso.buscar != "" {
				entrada = strings.Replace(entrada, caso.buscar, caso.cambiar, 1)
				if entrada == string(b) {
					t.Fatal("el caso no modificó el fixture")
				}
			}
			var salida bytes.Buffer
			if codigo := ejecutar(args, strings.NewReader(entrada), &salida); codigo != caso.codigo || strings.Contains(salida.String(), "SECRET") {
				t.Fatalf("caso %s: código=%d esperado=%d salida=%s", nombre, codigo, caso.codigo, salida.String())
			}
		})
	}
}

func TestCLIPresentacionCertificadoConservaAD219(t *testing.T) {
	b, err := os.ReadFile("../../internal/vec/auditoria/testdata/catalogo_acciones_ad219_sintetico.json")
	if err != nil {
		t.Fatal(err)
	}
	var objeto map[string]json.RawMessage
	if err := json.Unmarshal(b, &objeto); err != nil {
		t.Fatal(err)
	}
	objeto["esquema"], err = json.Marshal(auditoria.EsquemaVerificacionPresentacionCertificado)
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := json.Marshal(objeto)
	if err != nil {
		t.Fatal(err)
	}
	checkpoint := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := os.WriteFile(checkpoint, objeto["manifiesto"], 0600); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	args := []string{"--checkpoint", checkpoint, "--max-bytes", "32768", "--max-registros", "2"}
	if codigo := ejecutar(args, bytes.NewReader(contenido), &salida); codigo != 0 {
		t.Fatalf("AD219 anterior rechazado: código=%d salida=%s", codigo, salida.String())
	}
}
