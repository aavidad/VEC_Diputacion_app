package bootstrap

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
)

func TestConfianzaCTAdoptaVigenciaAdminSinAcortarEnMedianoche(t *testing.T) {
	t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileDevelopment)
	t.Setenv(config.EnvAuthMode, config.AuthModeDevelopment)
	t.Setenv(config.EnvDevelopmentGuard, config.DevelopmentGuardAcknowledgement)
	ahora := time.Date(2026, 10, 15, 14, 30, 0, 0, time.UTC)
	previa := materialRenovableCTPrueba(t, ahora)
	admin := previa
	admin.publicadaEn, admin.expiraEn = ahora, ahora.Add(60*24*time.Hour)
	admin.configuracionOrden++
	admin.configuracionRef = "confianza:atestacion:ct:desarrollo:2026-10-15:r90"
	var err error
	admin.configuracion, err = confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
		admin.configuracionRef, admin.configuracionOrden, admin.publicadaEn, admin.expiraEn, admin.raiz)
	if err != nil {
		t.Fatal(err)
	}
	admin.configuracionHuella, err = admin.configuracion.HuellaSHA256ParaGobierno()
	if err != nil {
		t.Fatal(err)
	}
	reloj := &relojRenovableCTPrueba{}
	reloj.fijar(ahora)
	f := fuenteRenovableCTPrueba(t, previa, reloj)
	var renovaciones atomic.Int32
	f.leer = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		return admin, nil
	}
	f.renovar = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		renovaciones.Add(1)
		return admin, nil
	}
	if _, err := f.instantanea(t.Context()); err != nil || f.material.configuracionRef != admin.configuracionRef {
		t.Fatalf("publicación ADMIN no adoptada: %v", err)
	}
	reloj.fijar(previa.expiraEn.Add(time.Minute))
	if _, err := f.instantanea(t.Context()); err != nil || renovaciones.Load() != 0 || f.material.expiraEn != admin.expiraEn {
		t.Fatalf("confianza recortada al cambiar el día: %v, renovaciones=%d", err, renovaciones.Load())
	}
	reinicio := fuenteRenovableCTPrueba(t, admin, reloj)
	reinicio.leer = f.leer
	reinicio.renovar = f.renovar
	if _, err := reinicio.instantanea(t.Context()); err != nil || reinicio.material.configuracionRef != admin.configuracionRef || renovaciones.Load() != 0 {
		t.Fatalf("reinicio cambió publicación: %v", err)
	}
}

func TestConfianzaCTValidaVigenciaPublicadaSinPerderRaiz(t *testing.T) {
	ahora := time.Date(2026, 10, 15, 14, 30, 0, 0, time.UTC)
	raizHasta := ahora.Add(91 * 24 * time.Hour)
	t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileDevelopment)
	t.Setenv(config.EnvAuthMode, config.AuthModeDevelopment)
	t.Setenv(config.EnvDevelopmentGuard, config.DevelopmentGuardAcknowledgement)
	for _, tc := range []struct {
		nombre string
		vida   time.Duration
		raiz   time.Time
		ok     bool
	}{
		{"una_hora", time.Hour, raizHasta, true},
		{"24_horas", 24 * time.Hour, raizHasta, true},
		{"90_dias", 90 * 24 * time.Hour, raizHasta, true},
		{"mas_de_90_dias", 90*24*time.Hour + time.Microsecond, raizHasta, false},
		{"raiz_corta", 90 * 24 * time.Hour, ahora.Add(90*24*time.Hour - time.Microsecond), false},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			if got := vigenciaPublicadaCTDesarrolloValida(ahora, ahora.Add(tc.vida), tc.raiz, ahora); got != tc.ok {
				t.Fatalf("vigencia aceptada=%v, esperada=%v", got, tc.ok)
			}
		})
	}
	t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileProduction)
	if vigenciaPublicadaCTDesarrolloValida(ahora, ahora.Add(60*24*time.Hour), raizHasta, ahora) {
		t.Fatal("producción aceptó 60 días")
	}
	if !vigenciaPublicadaCTDesarrolloValida(ahora.Truncate(24*time.Hour), ahora.Truncate(24*time.Hour).Add(24*time.Hour), raizHasta, ahora) {
		t.Fatal("publicación diaria histórica rechazada")
	}
	if publicacionDiariaCTDesarrollo(ahora, ahora.Add(24*time.Hour)) ||
		publicacionDiariaCTDesarrollo(ahora, ahora.Add(60*24*time.Hour)) ||
		!publicacionDiariaCTDesarrollo(ahora.Truncate(24*time.Hour), ahora.Truncate(24*time.Hour).Add(24*time.Hour)) {
		t.Fatal("publicación ADMIN vencida entraría en renovación automática")
	}
}
