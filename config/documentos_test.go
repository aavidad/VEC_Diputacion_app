package config

import (
	"errors"
	"testing"
)

func TestDocumentosApagadoPorDefectoYSelectorCerrado(t *testing.T) {
	for _, valor := range []string{"", "false", " false "} {
		if activo, err := (Config{DocumentosEnabled: valor}).DocumentosDesarrolloActivo(); activo || err != nil {
			t.Fatalf("%q: %v %v", valor, activo, err)
		}
	}
	for _, valor := range []string{"1", "yes", "TRUE", "on"} {
		if activo, err := (Config{DocumentosEnabled: valor}).DocumentosDesarrolloActivo(); activo || !errors.Is(err, ErrConfiguracionDocumentosSelector) {
			t.Fatalf("%q abre: %v %v", valor, activo, err)
		}
	}
	if activo, err := (Config{DocumentosEnabled: "true"}).DocumentosDesarrolloActivo(); activo || !errors.Is(err, ErrConfiguracionDocumentosActivacion) {
		t.Fatal("se activa sin la doble llave de desarrollo", err)
	}
}
