package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func TestFuenteConstituidaRRHHSelectorInvalidoSeDeniegaYRegistra(t *testing.T) {
	t.Setenv(envBolsaCeseCTEnabled, "valor_invalido")
	var registro bytes.Buffer
	salidaAnterior := log.Writer()
	log.SetOutput(&registro)
	t.Cleanup(func() { log.SetOutput(salidaAnterior) })

	cfg := config.Config{
		ExecutionProfile: config.ExecutionProfileDevelopment,
		AuthMode:         config.AuthModeDevelopment,
		DevelopmentGuard: config.DevelopmentGuardAcknowledgement,
	}
	if fuente := nuevaFuenteConstituidaRRHHDesarrollo(context.Background(), cfg); fuente != nil {
		t.Fatal("el selector invalido no debe componer la fuente")
	}
	if !strings.Contains(registro.String(), "etapa=selector_cese causa=") {
		t.Fatalf("falta diagnostico de la denegacion: %q", registro.String())
	}
}

func TestFalloFuenteConstituidaRRHHNoRegistraTextoSensible(t *testing.T) {
	var registro bytes.Buffer
	salidaAnterior := log.Writer()
	log.SetOutput(&registro)
	t.Cleanup(func() { log.SetOutput(salidaAnterior) })

	registrarFalloFuenteConstituidaRRHHDesarrollo("pool_bolsa", errors.New("postgres://usuario:secreto@localhost/base"))
	salida := registro.String()
	if !strings.Contains(salida, "etapa=pool_bolsa causa=") || strings.Contains(salida, "secreto") || strings.Contains(salida, "usuario") {
		t.Fatalf("diagnostico incorrecto o sensible: %q", salida)
	}
}
