package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type repositorioPoliticaPrueba struct {
	*repositorioOperacionPrueba
	vigente ports.PoliticaSegregacionVigente
	err     error
}

func (r *repositorioPoliticaPrueba) PoliticaSegregacion(context.Context) (ports.PoliticaSegregacionVigente, error) {
	return r.vigente, r.err
}

func solicitudOperacionPoliticaPrueba(t *testing.T, ahora time.Time) ports.SolicitudOperacionSituacion {
	t.Helper()
	q := ports.SolicitudOperacionSituacion{SolicitudCambiarSituacionParticipacion: solicitudSituacionPrueba(t, ahora), Operacion: domain.OperacionPausar, Justificante: domain.JustificanteOperacionSituacion{Tipo: domain.JustificanteSolicitudCandidato, Referencia: "justificante:01", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}
	q.Validador = q.ResultadoContexto.Contexto.PersonaRef
	return q
}

func TestOperacionB8AplicaLaPoliticaConfiguradaDeSegundaPersona(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	politica, err := domain.NuevaPoliticaSegregacion([]string{domain.OperacionPausar, domain.OperacionExcluir})
	if err != nil {
		t.Fatal(err)
	}
	repo := &repositorioPoliticaPrueba{repositorioOperacionPrueba: &repositorioOperacionPrueba{repositorioSituacionPrueba: &repositorioSituacionPrueba{pertenece: true, vigente: ports.SituacionParticipacion{ParticipacionRef: "participacion:b2", Situacion: domain.SituacionDisponible, Desde: ahora}}}, vigente: ports.PoliticaSegregacionVigente{Version: 2, Politica: politica}}
	s, _ := NuevoServicioSituacionParticipacion(contextoSituacionPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, func() time.Time { return ahora })
	q := solicitudOperacionPoliticaPrueba(t, ahora)
	if _, err := s.Operar(context.Background(), q); !errors.Is(err, domain.ErrOperacionSituacionParticipacionInvalida) || repo.escrituras != 0 {
		t.Fatalf("la pausa configurada exige segunda persona: %v escrituras=%d", err, repo.escrituras)
	}
	q.Validador = "persona:validadora"
	if _, err := s.Operar(context.Background(), q); err != nil || repo.escrituras != 1 {
		t.Fatalf("con segunda persona debe registrarse: %v escrituras=%d", err, repo.escrituras)
	}
}

func TestOperacionB8SinPoliticaConservaLaConductaDeHoy(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	repo := &repositorioOperacionPrueba{repositorioSituacionPrueba: &repositorioSituacionPrueba{pertenece: true, vigente: ports.SituacionParticipacion{ParticipacionRef: "participacion:b2", Situacion: domain.SituacionDisponible, Desde: ahora}}}
	s, _ := NuevoServicioSituacionParticipacion(contextoSituacionPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, func() time.Time { return ahora })
	if _, err := s.Operar(context.Background(), solicitudOperacionPoliticaPrueba(t, ahora)); err != nil || repo.escrituras != 1 {
		t.Fatalf("sin política la pausa admite a la misma persona: %v escrituras=%d", err, repo.escrituras)
	}
}

func TestOperacionB8FallaCerradaSiLaPoliticaNoSePuedeLeer(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	repo := &repositorioPoliticaPrueba{repositorioOperacionPrueba: &repositorioOperacionPrueba{repositorioSituacionPrueba: &repositorioSituacionPrueba{pertenece: true, vigente: ports.SituacionParticipacion{ParticipacionRef: "participacion:b2", Situacion: domain.SituacionDisponible, Desde: ahora}}}, err: errors.New("caida")}
	s, _ := NuevoServicioSituacionParticipacion(contextoSituacionPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, func() time.Time { return ahora })
	q := solicitudOperacionPoliticaPrueba(t, ahora)
	q.Validador = "persona:validadora"
	if _, err := s.Operar(context.Background(), q); !errors.Is(err, ErrCambioSituacionParticipacionNoDisponible) || repo.escrituras != 0 {
		t.Fatalf("una política ilegible no autoriza: %v escrituras=%d", err, repo.escrituras)
	}
}
