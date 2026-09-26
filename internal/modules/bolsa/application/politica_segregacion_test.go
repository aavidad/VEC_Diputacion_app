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

// La suspensión (efecto «pausar») aplica la misma política configurable que
// B8: si el catálogo pide segunda persona para pausar, quien anota no puede
// resolverla, y la base (000033) no llega a rechazarla con otro error.
func TestSancionSuspensionAplicaLaPoliticaConfiguradaDeSegundaPersona(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	politica, err := domain.NuevaPoliticaSegregacion([]string{domain.OperacionPausar, domain.OperacionReactivar, domain.OperacionExcluir})
	if err != nil {
		t.Fatal(err)
	}
	repoSituacion := &repositorioPoliticaPrueba{repositorioOperacionPrueba: &repositorioOperacionPrueba{repositorioSituacionPrueba: &repositorioSituacionPrueba{pertenece: true, vigente: ports.SituacionParticipacion{ParticipacionRef: "participacion:b2", Situacion: domain.SituacionDisponible, Desde: ahora.Add(-time.Hour)}}}, vigente: ports.PoliticaSegregacionVigente{Version: 2, Politica: politica}}
	situacion, err := NuevoServicioSituacionParticipacion(contextoSituacionPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repoSituacion, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	repo := &repositorioSancionesPrueba{}
	s, err := NuevoServicioSancionesParticipacion(situacion, &catalogoSancionesPrueba{}, repo)
	if err != nil {
		t.Fatal(err)
	}
	q := solicitudSancionPrueba(t, ahora, "b24.sancion.suspension")
	q.Datos.ResueltaPor = q.ResultadoContexto.Contexto.PersonaRef
	if _, err := s.Registrar(context.Background(), q); !errors.Is(err, domain.ErrSancionParticipacionInvalida) || repo.escrituras != 0 {
		t.Fatalf("suspensión autorresuelta con pausa en la política: %v escrituras=%d", err, repo.escrituras)
	}
	q.Datos.ResueltaPor = "persona:jefatura"
	if _, err := s.Registrar(context.Background(), q); err != nil || repo.escrituras != 1 {
		t.Fatalf("con segunda persona debe registrarse: %v escrituras=%d", err, repo.escrituras)
	}
	repoSituacion.err = errors.New("caida")
	if _, err := s.Registrar(context.Background(), q); err == nil || repo.escrituras != 1 {
		t.Fatalf("sin política legible no se registra: %v", err)
	}
}
