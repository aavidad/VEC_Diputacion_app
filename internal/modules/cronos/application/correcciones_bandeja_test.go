package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type registroDecisionCorreccionPrueba struct {
	registroVinculoPrueba
	accion, finalidad string
}

func (r *registroDecisionCorreccionPrueba) RegistrarDenegacionCorreccion(_ context.Context, actor vecdomain.ContextoActor, accion, finalidad string) error {
	r.llamadas++
	r.actor = actor
	r.accion, r.finalidad = accion, finalidad
	return r.err
}

type repoDecisionCorreccionPrueba struct {
	bandeja  ports.BandejaCorrecciones
	recibo   ports.ReciboCorreccion
	material domain.MaterialActuacionCorreccion
	llamadas int
	err      error
}

func (r *repoDecisionCorreccionPrueba) ConsultarBandejaCorrecciones(_ context.Context, _ ports.OrdenConsumoCorreccion, _ domain.MaterialBandejaCorrecciones) (ports.BandejaCorrecciones, error) {
	r.llamadas++
	return r.bandeja, r.err
}
func (r *repoDecisionCorreccionPrueba) RegistrarDecisionCorreccion(_ context.Context, _ ports.OrdenConsumoCorreccion, m domain.MaterialActuacionCorreccion) (ports.ReciboCorreccion, error) {
	r.llamadas++
	r.material = m
	return r.recibo, r.err
}
func (r *repoDecisionCorreccionPrueba) ConsultarHistoriaCorreccion(context.Context, ports.OrdenConsumoCorreccion, domain.MaterialHistoriaCorreccion) (ports.HistoriaCorreccion, error) {
	return ports.HistoriaCorreccion{}, ports.ErrDependenciaNoDisponible
}

func TestDecisionCorreccionDerivaActorYCompruebaRecibo(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	orden := ordenCorreccionPrueba(t)
	repo := &repoDecisionCorreccionPrueba{recibo: ports.ReciboCorreccion{
		SolicitudRef: "correccion:cronos:solicitud-0001", ActuacionRef: "correccion:actuacion:00000000-0000-4000-8000-000000000000",
		ReciboRef: "recibo:cronos:00000000-0000-4000-8000-000000000000", Estado: domain.CorreccionPendienteRRHH,
		Version: 2, InstanteUTC: ahora}}
	s, err := NuevoServicioDecisionCorrecciones(repo, relojMarcajePrueba{ahora}, zona)
	if err != nil {
		t.Fatal(err)
	}
	p := ports.PeticionActuacionCorreccion{SolicitudRef: "correccion:cronos:solicitud-0001", ClaveOperacion: "actuacion-0001",
		Paso: domain.PasoDecisionResponsable, Resultado: domain.ResultadoFavorable, Motivo: "Olvido acreditado", VersionEsperada: 1}
	r, err := s.RegistrarDecisionCorreccion(context.Background(), orden, p)
	if err != nil || r.Version != 2 || repo.material.EmpleadoRef != "emp_0123456789abcdefghijkl" || repo.material.ActorRef == "" {
		t.Fatalf("actor o recibo divergentes: %+v %+v %v", r, repo.material, err)
	}
	q := p
	q.Resultado = domain.ResultadoDesfavorable
	q.Motivo = ""
	antes := repo.llamadas
	if _, err := s.RegistrarDecisionCorreccion(context.Background(), orden, q); !errors.Is(err, domain.ErrCorreccionInvalida) || repo.llamadas != antes {
		t.Fatalf("denegación sin motivo alcanzó repositorio: %v", err)
	}
	repo.recibo.Estado = domain.CorreccionAplicada
	if _, err := s.RegistrarDecisionCorreccion(context.Background(), orden, p); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatalf("recibo con otro estado aceptado: %v", err)
	}
}

func TestBandejaCorreccionesRechazaSolicitudPropia(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	orden := ordenCorreccionPrueba(t)
	repo := &repoDecisionCorreccionPrueba{bandeja: ports.BandejaCorrecciones{Paso: "responsable", Pendientes: []ports.CorreccionPendiente{{
		SolicitudRef: "correccion:cronos:solicitud-0001", EmpleadoRef: "emp_0123456789abcdefghijkl",
		Estado: domain.CorreccionPendienteResponsable, Version: 1, SolicitadaEnUTC: ahora}}}}
	s, _ := NuevoServicioDecisionCorrecciones(repo, relojMarcajePrueba{ahora}, zona)
	if _, err := s.ConsultarBandejaCorrecciones(context.Background(), orden, "responsable"); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatalf("bandeja propia aceptada: %v", err)
	}
}

func TestCorreccionRechazoCompetenciaExigeAuditoriaConfirmada(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	orden := ordenCorreccionPrueba(t)
	repo := &repoDecisionCorreccionPrueba{err: ports.ErrResolucionNoCompetente}
	s, _ := NuevoServicioDecisionCorrecciones(repo, relojMarcajePrueba{ahora}, zona)
	p := ports.PeticionActuacionCorreccion{SolicitudRef: "correccion:cronos:solicitud-0001", ClaveOperacion: "actuacion-0001",
		Paso: domain.PasoDecisionResponsable, Resultado: domain.ResultadoFavorable, Motivo: "Olvido acreditado", VersionEsperada: 1}
	auditoria := &registroDecisionCorreccionPrueba{}
	ctx := ConRegistroDenegacionVinculo(context.Background(), auditoria)
	if _, err := s.RegistrarDecisionCorreccion(ctx, orden, p); !errors.Is(err, ports.ErrResolucionNoCompetente) || auditoria.llamadas != 1 || auditoria.accion != AccionDecidirCorreccion || auditoria.finalidad != FinalidadDecisionResponsableCorreccion {
		t.Fatalf("rechazo sin auditoría: %v llamadas=%d", err, auditoria.llamadas)
	}
	auditoria.err = errors.New("almacén auditor no disponible")
	if _, err := s.RegistrarDecisionCorreccion(ctx, orden, p); !errors.Is(err, ports.ErrDependenciaNoDisponible) || errors.Is(err, ports.ErrResolucionNoCompetente) {
		t.Fatalf("auditoría fallida aún devuelve 403: %v", err)
	}
	if _, err := s.RegistrarDecisionCorreccion(context.Background(), orden, p); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatalf("sin registrador aún devuelve 403: %v", err)
	}
}
