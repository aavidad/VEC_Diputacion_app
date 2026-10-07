package simulacion

import (
	"reflect"
	"strings"
	"testing"
	"vec-diputacion-granada/internal/modules/provision/application"
)

func TestDecodificarAdjudicacionNoAceptaContratoValoracion(t *testing.T) {
	// El DTO de adjudicación exige universos y política; no reutiliza el DTO
	// de valoración aceptando silenciosamente otro significado de entrada.
	if _, err := DecodificarAdjudicacion(strings.NewReader(`{"configuracion":{},"entrada":{}}`)); err == nil {
		t.Fatal("forma vacía aceptada")
	}
}

func TestEjemploAdjudicacionNoComparteMutacion(t *testing.T) {
	original, err := EjemploAdjudicacion()
	if err != nil {
		t.Fatal(err)
	}
	copia, err := EjemploAdjudicacion()
	if err != nil {
		t.Fatal(err)
	}
	copia.Configuracion.Desempates[0].Sentido = "menor"
	copia.Entrada.Solicitudes[0].Preferencias[0].Valoracion.Total = nil
	siguiente, err := EjemploAdjudicacion()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original, siguiente) {
		t.Fatal("fixture global mutado")
	}
	r, err := application.SimularAdjudicacion(siguiente.Configuracion, siguiente.Entrada)
	if err != nil || r.Estado != "propuesta_simulada" {
		t.Fatal(r, err)
	}
}
