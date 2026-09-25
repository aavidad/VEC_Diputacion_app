package config

import (
	"errors"
	"testing"
)

func TestPersonalEmpleadoApagadoPorDefectoYSelectorCerrado(t *testing.T) {
	for _, valor := range []string{"", "false", " false "} {
		if activo, err := (Config{PersonalEmpleadoEnabled: valor}).PersonalEmpleadoDesarrolloActivo(); activo || err != nil {
			t.Fatalf("%q: %v %v", valor, activo, err)
		}
	}
	for _, valor := range []string{"1", "yes", "TRUE"} {
		if activo, err := (Config{PersonalEmpleadoEnabled: valor}).PersonalEmpleadoDesarrolloActivo(); activo || !errors.Is(err, ErrConfiguracionPersonalEmpleadoSelector) {
			t.Fatalf("%q abre: %v %v", valor, activo, err)
		}
	}
	if activo, err := (Config{PersonalEmpleadoEnabled: "true"}).PersonalEmpleadoDesarrolloActivo(); activo || !errors.Is(err, ErrConfiguracionPersonalEmpleadoActivacion) {
		t.Fatal("se activa sin la doble llave de desarrollo", err)
	}
}
