package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

func entradaAportacion(t *testing.T) []byte {
	t.Helper()
	var material ports.MaterialAdmisionPreparacion
	if err := json.Unmarshal(fixture(t), &material); err != nil {
		t.Fatal(err)
	}
	antecedente, err := application.IdentificarAntecedenteAdmision(context.Background(), material)
	if err != nil {
		t.Fatal(err)
	}
	entrada, err := json.Marshal(application.MaterialAportacionAdmision{MaterialS4: material, Propuesta: domain.PropuestaAportacionAdmision{
		Alcance: domain.AlcanceAdmisionPreparacion, AportacionRef: "aportacion:propuesta", Revision: 1, Antecedente: antecedente,
		RequisitoRef: material.Requisitos[0].Referencia, RequisitoVersion: material.Requisitos[0].Version,
		SoportesPropuestos: []domain.SoporteAportacionAdmision{{Hecho: material.Requisitos[0].HechosEsperados[0], Documentos: []vec.ReferenciaDocumento{{ID: "documento:propuesto", Version: 1}}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return entrada
}

func TestCLIAntecedenteYAportacionDerivanMaterialConCatalogos(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		var salida, errores bytes.Buffer
		args := append(argumentos(idioma), "--salida", "antecedente")
		if ejecutar(context.Background(), args, bytes.NewReader(fixture(t)), &salida, &errores) != 0 {
			t.Fatal(errores.String())
		}
		var a domain.AntecedenteAdmision
		if json.Unmarshal(salida.Bytes(), &a) != nil || a.EsquemaMaterial != domain.EsquemaMaterialAdmisionLocal || len(a.HuellaMaterialSHA256) != 64 {
			t.Fatal(salida.String())
		}
		salida.Reset()
		args = append(argumentos(idioma), "--salida", "aportacion")
		if ejecutar(context.Background(), args, bytes.NewReader(entradaAportacion(t)), &salida, &errores) != 0 {
			t.Fatal(errores.String())
		}
		var out domain.AportacionAdmisionPreparada
		if json.Unmarshal(salida.Bytes(), &out) != nil || out.Resuelto || out.Presentado || out.RequisitoAnterior.Estado != "pendiente" || out.Propuesta.Antecedente != a {
			t.Fatal(salida.String())
		}
		esperado, err := os.ReadFile("testdata/aportacion-resultado.json")
		if err != nil || !bytes.Equal(esperado, salida.Bytes()) {
			t.Fatalf("consumer fixture differs: %v", err)
		}
		c, err := cargarCatalogo("../../web/static/textos", idioma)
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range out.Pendientes {
			if _, ok := mensajeCatalogo(c, idioma, key); !ok {
				t.Fatalf("missing %s %s", idioma, key)
			}
		}
	}
}

func TestCLIIdentificaIgualMaterialConEspaciosYOrdenDeClavesDistintos(t *testing.T) {
	var valor map[string]json.RawMessage
	if err := json.Unmarshal(fixture(t), &valor); err != nil {
		t.Fatal(err)
	}
	reordenado, err := json.Marshal(valor)
	if err != nil {
		t.Fatal(err)
	}
	var original, ordenado, errores bytes.Buffer
	args := append(argumentos("es"), "--salida", "antecedente")
	if ejecutar(context.Background(), args, bytes.NewReader(fixture(t)), &original, &errores) != 0 ||
		ejecutar(context.Background(), args, bytes.NewReader(reordenado), &ordenado, &errores) != 0 || !bytes.Equal(original.Bytes(), ordenado.Bytes()) {
		t.Fatalf("%s %s %s", original.String(), ordenado.String(), errores.String())
	}
}

func TestCLIPropuestaAmbiguaOVersionsustituidaNoDaResultado(t *testing.T) {
	raw := string(entradaAportacion(t))
	casos := []string{
		strings.Replace(raw, `"aportacion_ref":"aportacion:propuesta"`, `"aportacion_ref":"aportacion:propuesta","aportaci\u006fn_ref":"otra"`, 1),
		strings.Replace(raw, `"requisito_version":"1"`, `"requisito_version":"2"`, 1),
		strings.Replace(raw, `"soportes_propuestos":`, `"resuelto":true,"soportes_propuestos":`, 1),
		strings.Replace(raw, `"material_s4":`, `"Material_s4":`, 1),
	}
	for _, datos := range casos {
		var salida, errores bytes.Buffer
		if ejecutar(context.Background(), append(argumentos("es"), "--salida", "aportacion"), strings.NewReader(datos), &salida, &errores) == 0 || salida.Len() != 0 || !strings.Contains(errores.String(), "No se pudo preparar la aportación") {
			t.Fatalf("%s %s", salida.String(), errores.String())
		}
	}
}
