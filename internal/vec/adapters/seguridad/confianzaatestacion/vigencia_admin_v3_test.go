package confianzaatestacion

import (
	"crypto/ed25519"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
)

func TestVigenciaConfianzaV3SoloCTConDobleLlave(t *testing.T) {
	ahora := time.Date(2026, 10, 15, 14, 30, 0, 0, time.UTC)
	privada := ed25519.NewKeyFromSeed([]byte(strings.Repeat("v", ed25519.SeedSize)))
	raiz := func(id, audiencia string, clave ed25519.PublicKey) RaizPublicaAtestacionAutorizacionV3 {
		t.Helper()
		r, err := NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(id, 1, clave, audiencia,
			EstadoClaveAtestacionAutorizacionV3Activa, ahora.Add(-time.Hour), ahora.Add(91*24*time.Hour), time.Time{})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	ct := raiz("clave:atestacion:v3:ct", audienciaConfianzaCTDesarrolloV3, privada.Public().(ed25519.PublicKey))
	preferencias := raiz("clave:atestacion:v3:otra", "vec:desarrollo:otra:atestacion:v3",
		ed25519.NewKeyFromSeed([]byte(strings.Repeat("w", ed25519.SeedSize))).Public().(ed25519.PublicKey))
	crear := func(vida time.Duration, raices ...RaizPublicaAtestacionAutorizacionV3) error {
		_, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV3("confianza:atestacion:ct:desarrollo:2026-10-15:r1", 1,
			ahora, ahora.Add(vida), raices...)
		return err
	}
	t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileProduction)
	t.Setenv(config.EnvAuthMode, config.AuthModeDisabled)
	t.Setenv(config.EnvDevelopmentGuard, "")
	if !errors.Is(crear(25*time.Hour, ct), ErrConfiguracionConfianzaAtestacionV3Invalida) || crear(24*time.Hour, ct) != nil {
		t.Fatal("producción no conserva el límite de 24 horas")
	}
	t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileDevelopment)
	t.Setenv(config.EnvAuthMode, config.AuthModeDevelopment)
	if !errors.Is(crear(25*time.Hour, ct), ErrConfiguracionConfianzaAtestacionV3Invalida) {
		t.Fatal("doble llave incompleta aceptada")
	}
	t.Setenv(config.EnvDevelopmentGuard, config.DevelopmentGuardAcknowledgement)
	if crear(90*24*time.Hour, ct) != nil ||
		!errors.Is(crear(90*24*time.Hour+time.Microsecond, ct), ErrConfiguracionConfianzaAtestacionV3Invalida) ||
		!errors.Is(crear(25*time.Hour, preferencias), ErrConfiguracionConfianzaAtestacionV3Invalida) ||
		!errors.Is(crear(25*time.Hour, ct, preferencias), ErrConfiguracionConfianzaAtestacionV3Invalida) {
		t.Fatal("límite o audiencia de confianza V3 incorrectos")
	}
}

func TestDatosPruebaConfianzaV3MismoLimiteDeAudiencia(t *testing.T) {
	t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileDevelopment)
	t.Setenv(config.EnvAuthMode, config.AuthModeDevelopment)
	t.Setenv(config.EnvDevelopmentGuard, config.DevelopmentGuardAcknowledgement)
	e := nuevoEscenarioConfianzaAtestacionV3Prueba(t)
	prueba, err := e.servicio.Verificar(t.Context(), e.solicitud, e.decision, e.motivo, e.resultado, e.atestacion)
	if err != nil {
		t.Fatal(err)
	}
	d, err := prueba.Datos()
	if err != nil {
		t.Fatal(err)
	}
	d.AudienciaDespliegue = audienciaConfianzaCTDesarrolloV3
	d.ConfiguracionPublicadaEn = e.ahora
	d.ConfiguracionExpiraEn = e.ahora.Add(90 * 24 * time.Hour)
	d.RaizValidaHasta = e.ahora.Add(91 * 24 * time.Hour)
	d.HuellaPruebaSHA256 = calcularHuellaPruebaConfianzaAtestacionV3(d)
	if d.Validar() != nil {
		t.Fatal("prueba CT de 90 días rechazada")
	}
	d.ConfiguracionExpiraEn = d.ConfiguracionExpiraEn.Add(time.Microsecond)
	d.HuellaPruebaSHA256 = calcularHuellaPruebaConfianzaAtestacionV3(d)
	if !errors.Is(d.Validar(), ErrPruebaConfianzaAtestacionV3Invalida) {
		t.Fatal("prueba de más de 90 días aceptada")
	}
	d.ConfiguracionExpiraEn = e.ahora.Add(25 * time.Hour)
	d.AudienciaDespliegue = audienciaConfianzaAtestacionV3Prueba
	d.HuellaPruebaSHA256 = calcularHuellaPruebaConfianzaAtestacionV3(d)
	if !errors.Is(d.Validar(), ErrPruebaConfianzaAtestacionV3Invalida) {
		t.Fatal("otra audiencia aceptó más de 24 horas")
	}
}
