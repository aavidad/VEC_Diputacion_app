package config

import (
	"errors"
	"strings"
	"testing"
)

func TestSelectorIncorporacionAcreditada(t *testing.T) {
	base := Config{ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment, DevelopmentGuard: DevelopmentGuardAcknowledgement}
	if activo, err := base.CTIncorporacionAcreditadaDesarrolloActivo(); activo || err != nil {
		t.Fatalf("la ausencia equivale a apagado: %v %v", activo, err)
	}
	c := base
	c.CTIncorporacionAcreditadaEnabled = "TRUE"
	if _, err := c.CTIncorporacionAcreditadaDesarrolloActivo(); !errors.Is(err, ErrConfiguracionCTIncorporacionAcreditadaSelector) {
		t.Fatalf("valor mal escrito: %v", err)
	}
	c.CTIncorporacionAcreditadaEnabled = "true"
	if _, err := c.CTIncorporacionAcreditadaDesarrolloActivo(); !errors.Is(err, ErrConfiguracionCTIncorporacionAcreditadaActivacion) {
		t.Fatalf("sin catálogo de reglas: %v", err)
	}
	c.ReglasEjemplo.CTSourcePath = "reglas.json"
	if _, err := c.CTIncorporacionAcreditadaDesarrolloActivo(); !errors.Is(err, ErrConfiguracionCTIncorporacionAcreditadaActivacion) ||
		!strings.Contains(err.Error(), EnvCTSeguimientoCeseEnabled) {
		t.Fatalf("sin el seguimiento de cese: %v", err)
	}
	c.CTSeguimientoCeseEnabled = "true"
	c.ReglasEjemplo.CausasCeseSourcePath = "causas.json"
	c.CTAnalisisMotivosSourcePath = "motivos.json"
	if activo, err := c.CTIncorporacionAcreditadaDesarrolloActivo(); !activo || err != nil {
		t.Fatalf("encendido completo: %v %v", activo, err)
	}
	fuera := Config{CTIncorporacionAcreditadaEnabled: "true"}
	if _, err := fuera.CTIncorporacionAcreditadaDesarrolloActivo(); !errors.Is(err, ErrConfiguracionCTIncorporacionAcreditadaActivacion) {
		t.Fatalf("fuera de la doble llave: %v", err)
	}
}
