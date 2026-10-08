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

func TestCLICatalogoAccionesExigeCamposExactosYHuella(t *testing.T) {
	b, err := os.ReadFile("../../internal/vec/auditoria/testdata/catalogo_acciones_ad219_sintetico.json")
	if err != nil {
		t.Fatal(err)
	}
	var o map[string]json.RawMessage
	if err := json.Unmarshal(b, &o); err != nil {
		t.Fatal(err)
	}
	checkpoint := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := os.WriteFile(checkpoint, o["manifiesto"], 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"--checkpoint", checkpoint, "--max-bytes", "16384", "--max-registros", "2"}
	casos := map[string]struct {
		buscar, cambiar string
		codigo          int
	}{
		"valido":         {codigo: 0},
		"extra":          {`"catalogo_version":`, `"dato_privado":"SECRET","catalogo_version":`, 2},
		"nulo":           {`"catalogo_version": "2"`, `"catalogo_version": null`, 2},
		"falta":          {`"catalogo_version": "2",`, ``, 2},
		"doble":          {`"catalogo_version":`, `"catalogo_version":"3","catalogo_version":`, 2},
		"aprobacion":     {`"aprobacion_ref": "aprobacion:catalogo:uno"`, `"aprobacion_ref": "aprobacion:catalogo:otra"`, 1},
		"huella_eslabon": {`"eslabon_sha256": "e3e75e70a7600cdf04e6a8cdd8a0b47ab88ad488bb14ceb6d41a2093024be69b"`, `"eslabon_sha256": "` + strings.Repeat("a", 64) + `"`, 1},
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

func TestCLICatalogoAccionesConservaAD196(t *testing.T) {
	b, err := os.ReadFile("../../internal/vec/auditoria/testdata/perfiles_asignables_ad196.json")
	if err != nil {
		t.Fatal(err)
	}
	var o map[string]json.RawMessage
	if err := json.Unmarshal(b, &o); err != nil {
		t.Fatal(err)
	}
	o["esquema"], err = json.Marshal(auditoria.EsquemaVerificacionCatalogoAcciones)
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	checkpoint := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := os.WriteFile(checkpoint, o["manifiesto"], 0600); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	args := []string{"--checkpoint", checkpoint, "--max-bytes", "32768", "--max-registros", "4"}
	if codigo := ejecutar(args, bytes.NewReader(contenido), &salida); codigo != 0 {
		t.Fatalf("AD196 anterior rechazado: código=%d salida=%s", codigo, salida.String())
	}
}
