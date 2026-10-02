package config

import (
	"errors"
	"testing"
)

func TestOrganizacionHistoricaGobiernoSelectorCerrado(t *testing.T) {
	for _, valor := range []string{"", "false", " false "} {
		activo, err := (Config{OrganizacionHistoricaGobiernoEnabled: valor}).OrganizacionHistoricaGobiernoDesarrolloActivo()
		if activo || err != nil {
			t.Fatalf("%q: %v %v", valor, activo, err)
		}
	}
	for _, valor := range []string{"1", "yes", "TRUE", "true false"} {
		activo, err := (Config{OrganizacionHistoricaGobiernoEnabled: valor}).OrganizacionHistoricaGobiernoDesarrolloActivo()
		if activo || !errors.Is(err, ErrConfiguracionOrganizacionHistoricaGobiernoSelector) {
			t.Fatalf("%q admitido: %v %v", valor, activo, err)
		}
	}
}

func TestOrganizacionHistoricaGobiernoExigeLlavesYNoSeleccionaB2(t *testing.T) {
	cfg := Config{
		OrganizacionHistoricaGobiernoEnabled: " true ",
		ExecutionProfile:                     ExecutionProfileDevelopment,
		AuthMode:                             AuthModeDevelopment,
		DevelopmentGuard:                     DevelopmentGuardAcknowledgement,
	}
	if activo, err := cfg.OrganizacionHistoricaGobiernoDesarrolloActivo(); !activo || err != nil {
		t.Fatalf("selección válida rechazada: %v", err)
	}
	if activo, err := cfg.PersonalB2GobiernoDesarrolloActivo(); activo || err != nil {
		t.Fatal("OH selecciona B2", err)
	}
	for _, falta := range []string{"perfil", "modo", "llave"} {
		c := cfg
		switch falta {
		case "perfil":
			c.ExecutionProfile = ExecutionProfileProduction
		case "modo":
			c.AuthMode = AuthModeDisabled
		case "llave":
			c.DevelopmentGuard = ""
		}
		if activo, err := c.OrganizacionHistoricaGobiernoDesarrolloActivo(); activo || !errors.Is(err, ErrConfiguracionOrganizacionHistoricaGobiernoActivacion) {
			t.Fatalf("abre sin %s: %v", falta, err)
		}
	}
}

func TestOrganizacionHistoricaGobiernoCargaConfiguracionCanonica(t *testing.T) {
	t.Setenv(EnvOrganizacionHistoricaGobiernoEnabled, " true ")
	if cfg := Load().Normalize(); cfg.OrganizacionHistoricaGobiernoEnabled != "true" {
		t.Fatal("selector no cargado en configuración canónica")
	}
}
