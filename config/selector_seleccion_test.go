package config

import (
	"errors"
	"testing"
)

func TestSelectorSolicitudesSeleccion(t *testing.T) {
	desarrollo := func() Config {
		return Config{ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment, DevelopmentGuard: DevelopmentGuardAcknowledgement}
	}
	c := desarrollo()
	if activo, err := c.SeleccionSolicitudesDesarrolloActivo(); activo || err != nil {
		t.Fatal("apagado por defecto")
	}
	c.SeleccionSolicitudesEnabled = "si"
	if _, err := c.SeleccionSolicitudesDesarrolloActivo(); !errors.Is(err, ErrConfiguracionSeleccionSolicitudesSelector) {
		t.Fatal("un valor distinto de true/false debe rechazarse")
	}
	c.SeleccionSolicitudesEnabled = "true"
	if _, err := c.SeleccionSolicitudesDesarrolloActivo(); !errors.Is(err, ErrConfiguracionSeleccionSolicitudesActivacion) {
		t.Fatal("encendido sin catálogo debe rechazarse")
	}
	c.SeleccionConvocatoriasSourcePath = "catalogo.json"
	if activo, err := c.SeleccionSolicitudesDesarrolloActivo(); !activo || err != nil {
		t.Fatalf("encendido con catálogo y doble llave: %v", err)
	}
	c.ExecutionProfile = ExecutionProfileProduction
	if _, err := c.SeleccionSolicitudesDesarrolloActivo(); !errors.Is(err, ErrConfiguracionSeleccionSolicitudesActivacion) {
		t.Fatal("fuera del perfil de desarrollo debe rechazarse")
	}
}
