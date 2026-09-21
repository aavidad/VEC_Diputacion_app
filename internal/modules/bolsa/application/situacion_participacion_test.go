package application

import (
	"context"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type autorizadorSituacionPrueba struct{ actor string }

func (a autorizadorSituacionPrueba) AutorizarCambioSituacionParticipacion(context.Context, string, string) (string, error) {
	return a.actor, nil
}

type repositorioSituacionPrueba struct {
	vigente     puertosbolsa.SituacionParticipacion
	llamadas    int
	reutilizado *puertosbolsa.RegistroSituacionParticipacion
}

func (r *repositorioSituacionPrueba) SituacionVigente(context.Context, string) (puertosbolsa.SituacionParticipacion, error) {
	return r.vigente, nil
}
func (r *repositorioSituacionPrueba) BuscarRegistroSituacion(context.Context, string, string) (puertosbolsa.RegistroSituacionParticipacion, error) {
	if r.reutilizado == nil {
		return puertosbolsa.RegistroSituacionParticipacion{}, puertosbolsa.ErrSituacionParticipacionNoEncontrada
	}
	return *r.reutilizado, nil
}
func (r *repositorioSituacionPrueba) RegistrarSituacion(_ context.Context, ref, situacion string, desde time.Time, fecha *time.Time, motivo, actor, clave, recibo string, _ time.Time) (puertosbolsa.RegistroSituacionParticipacion, error) {
	r.llamadas++
	return puertosbolsa.RegistroSituacionParticipacion{ReciboRef: recibo, SituacionParticipacion: puertosbolsa.SituacionParticipacion{ParticipacionRef: ref, Situacion: situacion, Desde: desde, FechaDisponible: fecha}}, nil
}

func TestServicioSituacionRecuperaElMismoReciboSinNuevaFila(t *testing.T) {
	ahora := time.Date(2026, 9, 21, 16, 0, 0, 0, time.UTC)
	recibo := puertosbolsa.RegistroSituacionParticipacion{Reutilizada: true, ReciboRef: "recibo:situacion:existente", Motivo: "Pausa comunicada", SituacionParticipacion: puertosbolsa.SituacionParticipacion{ParticipacionRef: "participacion:b2", Situacion: "no_disponible", Desde: ahora}}
	repo := &repositorioSituacionPrueba{reutilizado: &recibo}
	servicio, _ := NuevoServicioSituacionParticipacion(autorizadorSituacionPrueba{actor: "per_servidor"}, repo, func() time.Time { return ahora })
	resultado, err := servicio.Cambiar(context.Background(), SolicitudCambioSituacionParticipacion{ParticipacionRef: "participacion:b2", Destino: "no_disponible", Motivo: "Pausa comunicada", ClaveIdempotencia: "b2-cambio-0001", Desde: ahora})
	if err != nil || !resultado.Reutilizada || resultado.ReciboRef != recibo.ReciboRef || repo.llamadas != 0 {
		t.Fatalf("resultado=%+v err=%v llamadas=%d", resultado, err, repo.llamadas)
	}
}

func TestServicioSituacionCambiaConMotivoYReciboDeterminista(t *testing.T) {
	ahora := time.Date(2026, 9, 21, 16, 0, 0, 0, time.UTC)
	repo := &repositorioSituacionPrueba{vigente: puertosbolsa.SituacionParticipacion{ParticipacionRef: "participacion:b2", Situacion: "disponible", Desde: ahora}}
	servicio, err := NuevoServicioSituacionParticipacion(autorizadorSituacionPrueba{actor: "per_servidor"}, repo, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	s := SolicitudCambioSituacionParticipacion{ParticipacionRef: "participacion:b2", Destino: "no_disponible", Motivo: "Pausa comunicada", ClaveIdempotencia: "b2-cambio-0001", Desde: ahora}
	primero, err := servicio.Cambiar(context.Background(), s)
	if err != nil || primero.ReciboRef == "" || repo.llamadas != 1 {
		t.Fatalf("resultado=%+v err=%v llamadas=%d", primero, err, repo.llamadas)
	}
}

func TestServicioSituacionRechazaDesdeAnterior(t *testing.T) {
	ahora := time.Date(2026, 9, 21, 16, 0, 0, 0, time.UTC)
	repo := &repositorioSituacionPrueba{vigente: puertosbolsa.SituacionParticipacion{ParticipacionRef: "participacion:b2", Situacion: "disponible", Desde: ahora}}
	servicio, _ := NuevoServicioSituacionParticipacion(autorizadorSituacionPrueba{actor: "per_servidor"}, repo, func() time.Time { return ahora })
	_, err := servicio.Cambiar(context.Background(), SolicitudCambioSituacionParticipacion{ParticipacionRef: "participacion:b2", Destino: "no_disponible", Motivo: "Pausa comunicada", ClaveIdempotencia: "b2-cambio-0002", Desde: ahora.Add(-time.Microsecond)})
	if err == nil || repo.llamadas != 0 {
		t.Fatalf("err=%v llamadas=%d", err, repo.llamadas)
	}
}
