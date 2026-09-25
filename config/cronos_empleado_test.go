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

func TestCronosResolucionApagadaPorDefectoYExigeCronos(t *testing.T) {
	for _, valor := range []string{"", "false", " false "} {
		if activo, err := (Config{CronosResolucionEnabled: valor}).CronosResolucionDesarrolloActiva(); activo || err != nil {
			t.Fatalf("%q: %v %v", valor, activo, err)
		}
	}
	for _, valor := range []string{"1", "yes", "TRUE"} {
		if activo, err := (Config{CronosResolucionEnabled: valor, CronosEmpleadoEnabled: "true"}).CronosResolucionDesarrolloActiva(); activo || !errors.Is(err, ErrConfiguracionCronosResolucionSelector) {
			t.Fatalf("%q abre: %v %v", valor, activo, err)
		}
	}
	if activo, err := (Config{CronosResolucionEnabled: "true"}).CronosResolucionDesarrolloActiva(); activo || !errors.Is(err, ErrConfiguracionCronosResolucionSelector) {
		t.Fatal("se activa sin Cronos de la persona empleada", err)
	}
	if activo, err := (Config{CronosResolucionEnabled: "true", CronosEmpleadoEnabled: "true"}).CronosResolucionDesarrolloActiva(); activo || !errors.Is(err, ErrConfiguracionCronosEmpleadoActivacion) {
		t.Fatal("se activa sin la doble llave de desarrollo", err)
	}
}

func TestCronosNotificacionesApagadasPorDefectoYExigenCronos(t *testing.T) {
	for _, valor := range []string{"", "false", " false "} {
		if activo, err := (Config{CronosNotificacionesEnabled: valor}).CronosNotificacionesDesarrolloActivas(); activo || err != nil {
			t.Fatalf("%q: %v %v", valor, activo, err)
		}
	}
	for _, valor := range []string{"1", "si", "TRUE"} {
		if activo, err := (Config{CronosNotificacionesEnabled: valor, CronosEmpleadoEnabled: "true"}).CronosNotificacionesDesarrolloActivas(); activo || !errors.Is(err, ErrConfiguracionCronosNotificacionesSelector) {
			t.Fatalf("%q abre: %v %v", valor, activo, err)
		}
	}
	if activo, err := (Config{CronosNotificacionesEnabled: "true"}).CronosNotificacionesDesarrolloActivas(); activo || !errors.Is(err, ErrConfiguracionCronosNotificacionesSelector) {
		t.Fatal("se activa sin Cronos de la persona empleada", err)
	}
	if activo, err := (Config{CronosNotificacionesEnabled: "true", CronosEmpleadoEnabled: "true"}).CronosNotificacionesDesarrolloActivas(); activo || !errors.Is(err, ErrConfiguracionCronosEmpleadoActivacion) {
		t.Fatal("se activa sin la doble llave de desarrollo", err)
	}
}
