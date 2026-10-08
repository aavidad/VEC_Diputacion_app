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

func TestCLIPreIdentidadTecnicaCamposMaterialYCadena(t *testing.T) {
	b, err := os.ReadFile("../../internal/vec/auditoria/testdata/pre_identidad_tecnica_ad222_sintetico.json")
	if err != nil {
		t.Fatal(err)
	}
	var documento map[string]json.RawMessage
	if err := json.Unmarshal(b, &documento); err != nil {
		t.Fatal(err)
	}
	checkpoint := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := os.WriteFile(checkpoint, documento["manifiesto"], 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"--checkpoint", checkpoint, "--max-bytes", "32768", "--max-registros", "8"}
	casos := map[string]struct {
		buscar, cambiar string
		codigo          int
	}{
		"valido":               {codigo: 0},
		"actor_extra":          {`"fase": "preacreditacion"`, `"actor_ref":"SECRET","fase": "preacreditacion"`, 2},
		"perfil_extra":         {`"fase": "preacreditacion"`, `"perfil_ref":"SECRET","fase": "preacreditacion"`, 2},
		"decision_extra":       {`"fase": "preacreditacion"`, `"decision_ref":"SECRET","fase": "preacreditacion"`, 2},
		"campo_extra":          {`"fase": "preacreditacion"`, `"dato_privado":"SECRET","fase": "preacreditacion"`, 2},
		"campo_ausente":        {`"fase": "preacreditacion",`, "", 2},
		"campo_nulo":           {`"fase": "preacreditacion"`, `"fase": null`, 2},
		"campo_duplicado":      {`"fase": "preacreditacion"`, `"fase":"otra","fase": "preacreditacion"`, 2},
		"metodo_ruta_cruzados": {`"metodo_esperado": "GET"`, `"metodo_esperado": "POST"`, 1},
		"motivo_cruzado":       {`"motivo_ref": "certificado_requerido"`, `"motivo_ref": "servicio_no_disponible"`, 1},
		"recurso":              {`"recurso_ref": "solicitud_sesion:81acc89b3ecbaa1f95dec17a2e3eb75e"`, `"recurso_ref": "solicitud_sesion:` + strings.Repeat("a", 32) + `"`, 1},
		"material":             {`"evento_material_sha256": "c31e42a013d90ce198f47884b2722ae13dc57f3220f6166c4e35ac3b6d1ee911"`, `"evento_material_sha256": "` + strings.Repeat("b", 64) + `"`, 1},
		"asiento":              {`"huella_sha256": "d8c7b323b40780df512fca25aec95b4bf78568cab65d347be5446574389ce2fd"`, `"huella_sha256": "` + strings.Repeat("c", 64) + `"`, 1},
		"eslabon":              {`"eslabon_sha256": "f05a20f53b4e1625efec6d53e03e43228184a895da45462a953dc7047a8529e4"`, `"eslabon_sha256": "` + strings.Repeat("d", 64) + `"`, 1},
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

func TestCLIPreIdentidadTecnicaConservaAD221(t *testing.T) {
	b, err := os.ReadFile("../../internal/vec/auditoria/testdata/presentacion_certificado_ad221_sintetico.json")
	if err != nil {
		t.Fatal(err)
	}
	var documento map[string]json.RawMessage
	if err := json.Unmarshal(b, &documento); err != nil {
		t.Fatal(err)
	}
	documento["esquema"], err = json.Marshal(auditoria.EsquemaVerificacionPreIdentidadTecnica)
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := json.Marshal(documento)
	if err != nil {
		t.Fatal(err)
	}
	checkpoint := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := os.WriteFile(checkpoint, documento["manifiesto"], 0600); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	args := []string{"--checkpoint", checkpoint, "--max-bytes", "32768", "--max-registros", "5"}
	if codigo := ejecutar(args, bytes.NewReader(contenido), &salida); codigo != 0 {
		t.Fatalf("AD221 anterior rechazado: código=%d salida=%s", codigo, salida.String())
	}
}
