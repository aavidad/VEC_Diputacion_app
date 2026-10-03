package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

func TestCLIPreparaMaterialConCatalogosReales(t *testing.T) {
	material, err := os.ReadFile("testdata/material.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			var salida, errores bytes.Buffer
			args := []string{"-catalogos-dir", "../../web/static/textos", "-idioma", idioma}
			if codigo := ejecutar(context.Background(), args, bytes.NewReader(material), &salida, &errores); codigo != 0 {
				t.Fatalf("codigo %d: %s", codigo, errores.String())
			}
			var resultado struct {
				Preparacion domain.PreparacionTribunal `json:"preparacion"`
				Limite      string                     `json:"limite"`
				Mensajes    []pendienteVisible         `json:"mensajes"`
			}
			if err := json.Unmarshal(salida.Bytes(), &resultado); err != nil {
				t.Fatal(err)
			}
			if resultado.Preparacion.Estado != "pendiente" || len(resultado.Preparacion.MaterialPropuesto.Miembros) != 2 ||
				len(resultado.Preparacion.MaterialPropuesto.Incidencias) != 2 || resultado.Preparacion.MaterialPropuesto.BasesS2.Revision != 2 ||
				resultado.Limite == "" || len(resultado.Mensajes) != len(resultado.Preparacion.Pendientes) {
				t.Fatalf("salida incompleta: %#v", resultado)
			}
			for _, mensaje := range resultado.Mensajes {
				if mensaje.Mensaje == "" || strings.Contains(mensaje.Mensaje, "campo_") {
					t.Fatalf("mensaje sin traducir: %#v", mensaje)
				}
			}
			var repetida bytes.Buffer
			if codigo := ejecutar(context.Background(), args, bytes.NewReader(material), &repetida, &errores); codigo != 0 || !bytes.Equal(repetida.Bytes(), salida.Bytes()) {
				t.Fatal("la misma propuesta produce material distinto")
			}
			errores.Reset()
			if codigo := ejecutar(context.Background(), args, bytes.NewReader(material), salidaFallida{}, &errores); codigo != 1 {
				t.Fatal("fallo de escritura no comunicado")
			}
			catalogo, err := cargarCatalogo("../../web/static/textos", idioma)
			if err != nil {
				t.Fatal(err)
			}
			var fallo map[string]string
			if json.Unmarshal(errores.Bytes(), &fallo) != nil || fallo["mensaje"] != catalogo.T(idioma, "error_salida") {
				t.Fatal("mensaje de escritura incorrecto")
			}
			if codigo := ejecutar(context.Background(), args, bytes.NewReader(material), salidaFallida{}, salidaFallida{}); codigo != MotivoDiagnosticoNoDisponible {
				t.Fatal("fallo de diagnóstico sin código propio")
			}
		})
	}
}

type salidaFallida struct{}

func (salidaFallida) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestCLIRechazaJSONAmbiguoSinSalida(t *testing.T) {
	casos := []string{
		`{"alcance":"preparacion_sintetica","alcance":"real"}`,
		`{"Alcance":"preparacion_sintetica"}`,
		`{"acto_aprobado":true}`,
		`{} {}`,
		strings.Repeat(" ", maximoEntradaJSON+1),
	}
	for _, material := range casos {
		var salida, errores bytes.Buffer
		codigo := ejecutar(context.Background(), []string{"-catalogos-dir", "../../web/static/textos"}, strings.NewReader(material), &salida, &errores)
		if codigo == 0 || salida.Len() != 0 || !strings.Contains(errores.String(), "error_clave") {
			t.Fatalf("entrada ambigua aceptada: codigo=%d salida=%s error=%s", codigo, salida.String(), errores.String())
		}
	}
}

func TestCatalogoNoSaleDeRaiz(t *testing.T) {
	raiz := t.TempDir()
	if err := os.Mkdir(filepath.Join(raiz, "es"), 0700); err != nil {
		t.Fatal(err)
	}
	exterior := filepath.Join(t.TempDir(), "catalogo.json")
	if err := os.WriteFile(exterior, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(exterior, filepath.Join(raiz, "es", "selectivos-tribunal.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := cargarCatalogo(raiz, "es"); err == nil {
		t.Fatal("catalogo externo aceptado")
	}
	if _, err := cargarCatalogo(raiz, "../es"); err == nil {
		t.Fatal("idioma con ruta aceptado")
	}
}
