package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLICSVPermisosCompletoYReducido(t *testing.T) {
	fixture := "testdata/ejemplo.json"
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			catalogo := "../../web/static/textos/" + idioma + "/cronos-informe-permisos-csv.json"
			var salida bytes.Buffer
			if err := ejecutar(context.Background(), []string{"--formato=csv", catalogo, fixture}, &salida); err != nil {
				t.Fatal(err)
			}
			filas, err := csv.NewReader(&salida).ReadAll()
			if err != nil || len(filas) != 4 || len(filas[0]) != 12 {
				t.Fatal("CSV completo", err, filas)
			}
			if filas[1][2] != "2026" || filas[2][9] != "12" || filas[3][9] != "120" {
				t.Fatal("contexto o enteros", filas)
			}
			base, err := os.ReadFile(fixture)
			if err != nil {
				t.Fatal(err)
			}
			completa := []byte(`"campos_permitidos": ["etiqueta", "unidad", "computo", "pendiente_resolver", "concedido", "restante", "conciliacion"]`)
			ruta := filepath.Join(t.TempDir(), "reducido.json")
			if err := os.WriteFile(ruta, bytes.Replace(base, completa, []byte(`"campos_permitidos": ["etiqueta", "unidad", "concedido"]`), 1), 0600); err != nil {
				t.Fatal(err)
			}
			salida.Reset()
			if err := ejecutar(context.Background(), []string{"--formato=csv", catalogo, ruta}, &salida); err != nil {
				t.Fatal(err)
			}
			filas, err = csv.NewReader(&salida).ReadAll()
			if err != nil || len(filas) != 4 || len(filas[0]) != 8 || filas[3][7] != "120" {
				t.Fatal("CSV reducido", err, filas)
			}
			for _, fila := range filas {
				if len(fila) != 8 {
					t.Fatal("fila irregular", fila)
				}
			}
			for _, excluido := range []string{"restante", "conciliacion", "pendiente_resolver"} {
				if strings.Contains(strings.ToLower(salida.String()), excluido) {
					t.Fatal("campo excluido", excluido)
				}
			}
		})
	}
}

func TestCLICSVPermisosOpcionesYDemoCerrados(t *testing.T) {
	catalogo := "../../web/static/textos/es/cronos-informe-permisos-csv.json"
	fixture := "testdata/ejemplo.json"
	for _, args := range [][]string{
		{"--formato=pdf", catalogo, fixture},
		{"--formato=csv", "--vista=permisos", catalogo, fixture},
		{"--formato=csv", "../../web/static/textos/es/cronos-informe-permisos.json", fixture},
		{"--formato=csv", catalogo},
		{catalogo, fixture},
	} {
		var salida bytes.Buffer
		if err := ejecutar(context.Background(), args, &salida); err == nil || salida.Len() != 0 {
			t.Fatal("opción inválida", args, err)
		}
	}
	base, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "sin-demo.json")
	if err := os.WriteFile(ruta, bytes.Replace(base, []byte(`"demo": true`), []byte(`"demo": false`), 1), 0600); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	if err := ejecutar(context.Background(), []string{"--formato=csv", catalogo, ruta}, &salida); err == nil || salida.Len() != 0 {
		t.Fatal("demo ausente produjo salida", err)
	}
}
