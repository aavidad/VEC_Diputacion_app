package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type repositorioOperacionPrueba struct {
	*repositorioSituacionPrueba
	operacion  *ports.RegistroOperacionSituacion
	escrituras int
}

func (r *repositorioOperacionPrueba) RegistrarOperacion(_ context.Context, c ports.ComandoOperacionSituacion) (ports.RegistroSituacionParticipacion, error) {
	if r.operacion != nil {
		if r.operacion.Operacion != c.Operacion || r.operacion.Justificante != c.Justificante || r.operacion.Validador != c.Validador || r.operacion.Motivo != c.Cambio.Motivo {
			return ports.RegistroSituacionParticipacion{}, ports.ErrClaveOperacionReutilizada
		}
		previo := r.operacion.RegistroSituacionParticipacion
		previo.Reutilizada = true
		return previo, nil
	}
	r.escrituras++
	res := ports.RegistroSituacionParticipacion{ReciboRef: c.ReciboRef, Motivo: c.Cambio.Motivo, SituacionParticipacion: ports.SituacionParticipacion{ParticipacionRef: c.Cambio.ParticipacionRef, Situacion: c.Cambio.Destino, Desde: c.Cambio.Desde}}
	r.operacion = &ports.RegistroOperacionSituacion{RegistroSituacionParticipacion: res, Operacion: c.Operacion, Justificante: c.Justificante, Actor: c.Actor, Validador: c.Validador, ValidadaEn: c.ValidadaEn}
	return res, nil
}
func (r *repositorioOperacionPrueba) ListarOperaciones(context.Context, string, string, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]ports.RegistroOperacionSituacion, error) {
	if r.operacion == nil {
		return []ports.RegistroOperacionSituacion{}, nil
	}
	return []ports.RegistroOperacionSituacion{*r.operacion}, nil
}

func TestOperacionB8ReciboReplayYSeparacionDeExclusion(t *testing.T) {
	ahora := time.Date(2026, 9, 23, 9, 0, 0, 0, time.UTC)
	repo := &repositorioOperacionPrueba{repositorioSituacionPrueba: &repositorioSituacionPrueba{pertenece: true, vigente: ports.SituacionParticipacion{ParticipacionRef: "participacion:b2", Situacion: domain.SituacionDisponible, Desde: ahora}}}
	s, _ := NuevoServicioSituacionParticipacion(contextoSituacionPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, func() time.Time { return ahora })
	q := ports.SolicitudOperacionSituacion{SolicitudCambiarSituacionParticipacion: solicitudSituacionPrueba(t, ahora), Operacion: domain.OperacionPausar, Justificante: domain.JustificanteOperacionSituacion{Tipo: domain.JustificanteSolicitudCandidato, Referencia: "justificante:01", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, Validador: "persona:validadora"}
	primero, err := s.Operar(context.Background(), q)
	if err != nil || primero.ReciboRef == "" || primero.Reutilizada || repo.escrituras != 1 {
		t.Fatalf("alta=%+v error=%v escrituras=%d", primero, err, repo.escrituras)
	}
	segundo, err := s.Operar(context.Background(), q)
	if err != nil || !segundo.Reutilizada || segundo.ReciboRef != primero.ReciboRef || repo.escrituras != 1 {
		t.Fatalf("replay=%+v error=%v escrituras=%d", segundo, err, repo.escrituras)
	}
	q.Justificante.Referencia = "justificante:otro"
	if _, err := s.Operar(context.Background(), q); !errors.Is(err, ports.ErrClaveOperacionReutilizada) {
		t.Fatalf("clave alterada: %v", err)
	}
	q.Operacion = domain.OperacionExcluir
	q.Destino = domain.SituacionExcluido
	q.ClaveIdempotencia = "b8-exclusion"
	q.Validador = q.ResultadoContexto.Contexto.PersonaRef
	if _, err := s.Operar(context.Background(), q); !errors.Is(err, domain.ErrOperacionSituacionParticipacionInvalida) {
		t.Fatalf("autovalidación de exclusión: %v", err)
	}
}

func TestOperacionB8DeniegaFueraDeAmbitoAntesDeLeer(t *testing.T) {
	ahora := time.Date(2026, 9, 23, 9, 0, 0, 0, time.UTC)
	repo := &repositorioOperacionPrueba{repositorioSituacionPrueba: &repositorioSituacionPrueba{pertenece: false}}
	s, _ := NuevoServicioSituacionParticipacion(contextoSituacionPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, func() time.Time { return ahora })
	q := ports.SolicitudOperacionSituacion{SolicitudCambiarSituacionParticipacion: solicitudSituacionPrueba(t, ahora), Operacion: domain.OperacionPausar, Justificante: domain.JustificanteOperacionSituacion{Tipo: domain.JustificanteSolicitudCandidato, Referencia: "justificante:01", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, Validador: "persona:validadora"}
	if _, err := s.Operar(context.Background(), q); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || repo.lecturas != 0 || repo.escrituras != 0 {
		t.Fatalf("denegacion=%v lecturas=%d escrituras=%d", err, repo.lecturas, repo.escrituras)
	}
}
