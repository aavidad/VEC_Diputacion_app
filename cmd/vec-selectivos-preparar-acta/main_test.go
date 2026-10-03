package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

type salidaFallida struct{}

func (salidaFallida) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestCLIActaConservaPropuestasConCatalogosReales(t *testing.T) {
	raw, err := os.ReadFile("testdata/material.json")
	if err != nil {
		t.Fatal(err)
	}
	var original domain.MaterialActaPropuesto
	if err := json.Unmarshal(raw, &original); err != nil {
		t.Fatal(err)
	}
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			args := []string{"-catalogos-dir", "../../web/static/textos", "-idioma", idioma}
			var salida, errores bytes.Buffer
			if codigo := ejecutar(context.Background(), args, bytes.NewReader(raw), &salida, &errores); codigo != 0 {
				t.Fatalf("codigo %d: %s", codigo, errores.String())
			}
			var r struct {
				Titulo      string                 `json:"titulo"`
				Preparacion domain.PreparacionActa `json:"preparacion"`
				Limite      string                 `json:"limite"`
				Mensajes    []pendienteVisible     `json:"mensajes"`
			}
			if json.Unmarshal(salida.Bytes(), &r) != nil || r.Preparacion.Estado != "borrador_propuesto" ||
				!reflect.DeepEqual(r.Preparacion.MaterialPropuesto, original) || r.Titulo == "" || r.Limite == "" ||
				len(r.Mensajes) != len(r.Preparacion.Pendientes) || bytes.Contains(salida.Bytes(), []byte(`"fecha_propuesta":`)) {
				t.Fatalf("borrador distinto o incompleto: %s", salida.String())
			}
			for _, mensaje := range r.Mensajes {
				if mensaje.Mensaje == "" || strings.Contains(mensaje.Mensaje, "campo_") {
					t.Fatalf("mensaje sin traducir: %#v", mensaje)
				}
			}
			var repetida bytes.Buffer
			if codigo := ejecutar(context.Background(), args, bytes.NewReader(raw), &repetida, &errores); codigo != 0 || !bytes.Equal(salida.Bytes(), repetida.Bytes()) {
				t.Fatal("misma entrada produce otro borrador")
			}
			errores.Reset()
			if codigo := ejecutar(context.Background(), args, bytes.NewReader(raw), salidaFallida{}, &errores); codigo != 1 {
				t.Fatal("fallo de salida no comunicado")
			}
			catalogo, err := cargarCatalogo("../../web/static/textos", idioma)
			if err != nil {
				t.Fatal(err)
			}
			var fallo map[string]string
			if json.Unmarshal(errores.Bytes(), &fallo) != nil || fallo["mensaje"] != catalogo.T(idioma, "error_salida") {
				t.Fatal("mensaje de salida incorrecto")
			}
			if codigo := ejecutar(context.Background(), args, bytes.NewReader(raw), salidaFallida{}, salidaFallida{}); codigo != MotivoDiagnosticoNoDisponible {
				t.Fatal("fallo de diagnóstico sin código propio")
			}
		})
	}
}

func TestCLIActaRechazaDatosDeActuacionYJSONAmbiguo(t *testing.T) {
	for _, raw := range []string{
		`{"asistentes":[]}`, `{"votos":1}`, `{"aprobada":true}`, `{"bases_s2":{}}`,
		`{"alcance":"preparacion_sintetica","alcance":"real"}`, `{"Alcance":"preparacion_sintetica"}`,
		`{} {}`, strings.Repeat(" ", maximoEntradaJSON+1),
	} {
		var salida, errores bytes.Buffer
		if codigo := ejecutar(context.Background(), []string{"-catalogos-dir", "../../web/static/textos"}, strings.NewReader(raw), &salida, &errores); codigo != 1 || salida.Len() != 0 {
			t.Fatalf("entrada aceptada: %s", raw[:min(len(raw), 100)])
		}
	}
}

func TestCatalogoActaNoSaleDeRaiz(t *testing.T) {
	raiz := t.TempDir()
	if err := os.Mkdir(filepath.Join(raiz, "es"), 0700); err != nil {
		t.Fatal(err)
	}
	exterior := filepath.Join(t.TempDir(), "catalogo.json")
	if err := os.WriteFile(exterior, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(exterior, filepath.Join(raiz, "es", "selectivos-acta-preparacion.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := cargarCatalogo(raiz, "es"); err == nil {
		t.Fatal("catalogo externo aceptado")
	}
}

func TestCLIActaNoReemplazaSustitutosUnicodeAislados(t *testing.T) {
	raw, err := os.ReadFile("testdata/material.json")
	if err != nil {
		t.Fatal(err)
	}
	var original domain.MaterialActaPropuesto
	if err := json.Unmarshal(raw, &original); err != nil {
		t.Fatal(err)
	}
	textoOriginal, err := json.Marshal(original.OrdenDiaPropuesto[0].TextoPropuesto)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre, textoJSON, esperado string
		rechazar                    bool
	}{
		{"alto_aislado", `\ud800`, "", true},
		{"bajo_aislado", `\udc00`, "", true},
		{"par_valido", `\ud83d\ude00`, "😀", false},
		{"reemplazo_literal", "�", "�", false},
		{"barra_literal", `\\ud800`, `\ud800`, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			entrada := strings.Replace(string(raw), string(textoOriginal), `"`+caso.textoJSON+`"`, 1)
			var salida, errores bytes.Buffer
			codigo := ejecutar(context.Background(), []string{"-catalogos-dir", "../../web/static/textos"}, strings.NewReader(entrada), &salida, &errores)
			if caso.rechazar {
				if codigo != 1 || salida.Len() != 0 {
					t.Fatal("sustituto aislado aceptado")
				}
				return
			}
			var resultado struct {
				Preparacion domain.PreparacionActa `json:"preparacion"`
			}
			if codigo != 0 || json.Unmarshal(salida.Bytes(), &resultado) != nil ||
				resultado.Preparacion.MaterialPropuesto.OrdenDiaPropuesto[0].TextoPropuesto != caso.esperado {
				t.Fatalf("texto válido alterado: %d %s", codigo, errores.String())
			}
		})
	}
}
