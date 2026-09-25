package config

import (
	"errors"
	"testing"
)

func TestPersonalB2GobiernoApagadoPorDefectoYSelectorCerrado(t *testing.T) {
	for _, valor := range []string{"", "false", " false "} {
		if activo, err := (Config{PersonalB2GobiernoEnabled: valor}).PersonalB2GobiernoDesarrolloActivo(); activo || err != nil {
			t.Fatalf("%q: %v %v", valor, activo, err)
		}
	}
	for _, valor := range []string{"1", "yes", "TRUE"} {
		if activo, err := (Config{PersonalB2GobiernoEnabled: valor}).PersonalB2GobiernoDesarrolloActivo(); activo || !errors.Is(err, ErrConfiguracionPersonalB2GobiernoSelector) {
			t.Fatalf("%q abre: %v %v", valor, activo, err)
		}
	}
	if activo, err := (Config{PersonalB2GobiernoEnabled: "true"}).PersonalB2GobiernoDesarrolloActivo(); activo || !errors.Is(err, ErrConfiguracionPersonalB2GobiernoActivacion) {
		t.Fatal("se activa sin la doble llave de desarrollo", err)
	}
}
