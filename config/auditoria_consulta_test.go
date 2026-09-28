package config

import (
	"errors"
	"testing"
)

func TestRutaCatalogoAuditoriaConsultaDesarrollo(t *testing.T) {
	activo := Config{ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment, DevelopmentGuard: DevelopmentGuardAcknowledgement}
	if ruta, err := activo.RutaCatalogoAuditoriaConsultaDesarrollo(); err != nil || ruta != RutaCatalogoAuditoriaConsultaEjemplo {
		t.Fatalf("catalogo predeterminado: %q, %v", ruta, err)
	}
	t.Setenv(EnvAuditoriaConsultaCatalogoPath, " /tmp/catalogo-auditoria.json ")
	if ruta, err := activo.RutaCatalogoAuditoriaConsultaDesarrollo(); err != nil || ruta != "/tmp/catalogo-auditoria.json" {
		t.Fatalf("catalogo editable: %q, %v", ruta, err)
	}
	if _, err := (Config{ExecutionProfile: ExecutionProfileProduction}).RutaCatalogoAuditoriaConsultaDesarrollo(); !errors.Is(err, ErrCatalogoAuditoriaConsultaFueraDesarrollo) {
		t.Fatalf("catalogo de ejemplo aceptado fuera de desarrollo: %v", err)
	}
	t.Setenv(EnvAuditoriaConsultaCatalogoPath, "  ")
	if _, err := activo.RutaCatalogoAuditoriaConsultaDesarrollo(); !errors.Is(err, ErrCatalogoAuditoriaConsultaFueraDesarrollo) {
		t.Fatalf("ruta vacia aceptada: %v", err)
	}
}

func TestExpedientesAuditoriaConsultaDesarrolloExigeDosReferenciasExactas(t *testing.T) {
	activo := Config{ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment, DevelopmentGuard: DevelopmentGuardAcknowledgement}
	if _, _, err := activo.ExpedientesAuditoriaConsultaDesarrollo(); !errors.Is(err, ErrExpedientesAuditoriaConsultaInvalidos) {
		t.Fatalf("se admitieron referencias ausentes: %v", err)
	}
	t.Setenv(EnvAuditoriaConsultaExpedienteCT, "expediente:ct:sintetico:001")
	t.Setenv(EnvAuditoriaConsultaExpedienteBolsa, "participacion:bolsa:sintetica:001")
	ct, bolsa, err := activo.ExpedientesAuditoriaConsultaDesarrollo()
	if err != nil || ct != "expediente:ct:sintetico:001" || bolsa != "participacion:bolsa:sintetica:001" {
		t.Fatalf("referencias explícitas rechazadas: %q %q %v", ct, bolsa, err)
	}
	for _, invalida := range []string{"*", "participacion:bolsa:*", " participacion:bolsa:sintetica:001", "expediente:ct:sintetico:001"} {
		t.Setenv(EnvAuditoriaConsultaExpedienteBolsa, invalida)
		if _, _, err := activo.ExpedientesAuditoriaConsultaDesarrollo(); !errors.Is(err, ErrExpedientesAuditoriaConsultaInvalidos) || err.Error() != ErrExpedientesAuditoriaConsultaInvalidos.Error() {
			t.Fatalf("referencia ajena o abierta admitida o expuesta: %q, %v", invalida, err)
		}
	}
	if _, _, err := (Config{ExecutionProfile: ExecutionProfileProduction}).ExpedientesAuditoriaConsultaDesarrollo(); !errors.Is(err, ErrExpedientesAuditoriaConsultaInvalidos) {
		t.Fatalf("referencias DEMO disponibles en producción: %v", err)
	}
}
