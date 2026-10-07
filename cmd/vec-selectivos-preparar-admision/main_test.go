package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

func fixture(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/material.json")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func argumentos(idioma string) []string {
	return []string{"--catalogos-dir", "../../web/static/textos", "--idioma", idioma}
}

func TestCLIProduceContratoSinMensajesYTraductorReal(t *testing.T) {
	var anterior []byte
	for _, idioma := range []string{"es", "en"} {
		var salida, errores bytes.Buffer
		if codigo := ejecutar(context.Background(), argumentos(idioma), bytes.NewReader(fixture(t)), &salida, &errores); codigo != 0 || errores.Len() != 0 {
			t.Fatalf("%d %s", codigo, errores.String())
		}
		esperado, err := os.ReadFile("testdata/resultado.json")
		if err != nil || !bytes.Equal(esperado, salida.Bytes()) {
			t.Fatalf("consumer fixture differs: %v", err)
		}
		var resultado domain.PreparacionAdmision
		if json.Unmarshal(salida.Bytes(), &resultado) != nil || resultado.AdmisionOficial || resultado.Persistido || resultado.Requisitos[0].Estado != "pendiente" {
			t.Fatal(salida.String())
		}
		if anterior != nil && !bytes.Equal(anterior, salida.Bytes()) {
			t.Fatal("locale changed machine contract")
		}
		anterior = append([]byte{}, salida.Bytes()...)
		var ayuda bytes.Buffer
		if ejecutar(context.Background(), append(argumentos(idioma), "--help"), nil, &ayuda, &errores) != 0 || ayuda.Len() == 0 {
			t.Fatal("missing translated help")
		}
	}
}

func TestCLIRechazaJSONAmbiguoYActosSinResultado(t *testing.T) {
	raw := string(fixture(t))
	casos := []string{
		strings.Replace(raw, `"revision": 1`, `"revision": 1, "revi\u0073ion": 2`, 1),
		strings.Replace(raw, `"revision": 1`, `"Revision": 1`, 1),
		strings.Replace(raw, `"revision": 1`, `"revision": 1, "admision_oficial": true`, 1),
		strings.Replace(raw, `"estado": "sin_presentar"`, `"estado": "presentada"`, 1),
		raw + `{}`,
		strings.Repeat(" ", maximoEntradaJSON+1),
		string([]byte{0xff}),
	}
	for i, material := range casos {
		var salida, errores bytes.Buffer
		codigo := ejecutar(context.Background(), argumentos("en"), strings.NewReader(material), &salida, &errores)
		if codigo == 0 || salida.Len() != 0 || !strings.Contains(errores.String(), "The review could not be prepared") {
			t.Fatalf("case %d: %d %s %s", i, codigo, salida.String(), errores.String())
		}
	}
}

func TestCatalogosCubrenCausasYAccionesDelPreparador(t *testing.T) {
	var resultado bytes.Buffer
	if ejecutar(context.Background(), argumentos("es"), bytes.NewReader(fixture(t)), &resultado, &bytes.Buffer{}) != 0 {
		t.Fatal("fixture")
	}
	var r domain.PreparacionAdmision
	if err := json.Unmarshal(resultado.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	for _, idioma := range []string{"es", "en"} {
		c, err := cargarCatalogo("../../web/static/textos", idioma)
		if err != nil {
			t.Fatal(err)
		}
		claves := append([]string{}, r.Pendientes...)
		for _, requisito := range r.Requisitos {
			claves = append(claves, requisito.Causas...)
			claves = append(claves, requisito.AccionPropuesta)
		}
		for _, clave := range claves {
			if _, ok := mensajeCatalogo(c, idioma, clave); !ok {
				t.Fatalf("missing %s %s", idioma, clave)
			}
		}
	}
}
