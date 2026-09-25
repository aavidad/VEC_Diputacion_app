package config

import (
	"errors"
	"testing"
)

func TestCronosEmpleadoApagadoPorDefectoYSelectorCerrado(t *testing.T) {
	for _, valor := range []string{"", "false", " false "} {
		if activo, err := (Config{CronosEmpleadoEnabled: valor}).CronosEmpleadoDesarrolloActivo(); activo || err != nil {
			t.Fatalf("%q: %v %v", valor, activo, err)
		}
	}
	for _, valor := range []string{"1", "yes", "TRUE"} {
		if activo, err := (Config{CronosEmpleadoEnabled: valor}).CronosEmpleadoDesarrolloActivo(); activo || !errors.Is(err, ErrConfiguracionCronosEmpleadoSelector) {
			t.Fatalf("%q abre: %v %v", valor, activo, err)
		}
	}
	if activo, err := (Config{CronosEmpleadoEnabled: "true"}).CronosEmpleadoDesarrolloActivo(); activo || !errors.Is(err, ErrConfiguracionCronosEmpleadoActivacion) {
		t.Fatal("se activa sin la doble llave de desarrollo", err)
	}
}
