package application

import (
	"context"
	"errors"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type contextoSituacionPrueba struct{ err error }

func (c contextoSituacionPrueba) ResolverContextoSituacionParticipacion(context.Context, dominiovec.ContextoActor, string, string) (puertosbolsa.ContextoSituacionParticipacionResuelto, error) {
	if c.err != nil {
		return puertosbolsa.ContextoSituacionParticipacionResuelto{}, c.err
	}
	return puertosbolsa.ContextoSituacionParticipacionResuelto{UnidadRef: "unidad:seleccion", AmbitoRef: "ambito:bolsa"}, nil
}

type repositorioSituacionPrueba struct {
	vigente      puertosbolsa.SituacionParticipacion
	pertenece    bool
	consultas    int
	pertenencias int
	lecturas     int
	llamadas     int
	reutilizado  *puertosbolsa.RegistroSituacionParticipacion
	ultimo       puertosbolsa.ComandoCambiarSituacionParticipacion
}

func (r *repositorioSituacionPrueba) ParticipacionPerteneceABolsa(context.Context, string, string) (bool, error) {
	r.consultas++
	r.pertenencias++
	return r.pertenece, nil
}

func (r *repositorioSituacionPrueba) SituacionVigente(context.Context, string) (puertosbolsa.SituacionParticipacion, error) {
	r.consultas++
	r.lecturas++
	return r.vigente, nil
}
func (r *repositorioSituacionPrueba) BuscarRegistroSituacion(context.Context, string, string) (puertosbolsa.RegistroSituacionParticipacion, error) {
	r.consultas++
	r.lecturas++
	if r.reutilizado == nil {
		return puertosbolsa.RegistroSituacionParticipacion{}, puertosbolsa.ErrSituacionParticipacionNoEncontrada
	}
	return *r.reutilizado, nil
}
func (r *repositorioSituacionPrueba) RegistrarSituacion(_ context.Context, comando puertosbolsa.ComandoCambiarSituacionParticipacion) (puertosbolsa.RegistroSituacionParticipacion, error) {
	r.llamadas++
	r.ultimo = comando
	if r.reutilizado != nil {
		return *r.reutilizado, nil
	}
	return puertosbolsa.RegistroSituacionParticipacion{ReciboRef: comando.ReciboRef, SituacionParticipacion: puertosbolsa.SituacionParticipacion{ParticipacionRef: comando.Cambio.ParticipacionRef, Situacion: comando.Cambio.Destino, Desde: comando.Cambio.Desde, FechaDisponible: comando.Cambio.FechaDisponible}}, nil
}

func solicitudSituacionPrueba(t *testing.T, ahora time.Time) puertosbolsa.SolicitudCambiarSituacionParticipacion {
	t.Helper()
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	return puertosbolsa.SolicitudCambiarSituacionParticipacion{Vinculo: vinculo, ResultadoContexto: resultado, BolsaRef: "bolsa:b2", ParticipacionRef: "participacion:b2", Destino: "no_disponible", Motivo: "Pausa comunicada", ClaveIdempotencia: "b2-cambio-0001", Correlacion: correlacionBorradorPrueba(t), MotivoAutorizacion: motivoBorradorPrueba()}
}

func TestServicioSituacionRecuperaElMismoReciboSinNuevaFila(t *testing.T) {
	ahora := time.Date(2026, 9, 21, 16, 0, 0, 0, time.UTC)
	recibo := puertosbolsa.RegistroSituacionParticipacion{Reutilizada: true, ReciboRef: "recibo:situacion:existente", Motivo: "Pausa comunicada", SituacionParticipacion: puertosbolsa.SituacionParticipacion{ParticipacionRef: "participacion:b2", Situacion: "no_disponible", Desde: ahora}}
	repo := &repositorioSituacionPrueba{pertenece: true, reutilizado: &recibo}
	autorizador := &autorizadorBorradorPrueba{t: t, instante: ahora}
	servicio, _ := NuevoServicioSituacionParticipacion(contextoSituacionPrueba{}, autorizador, repo, func() time.Time { return ahora })
	resultado, err := servicio.Cambiar(context.Background(), solicitudSituacionPrueba(t, ahora))
	if err != nil || !resultado.Reutilizada || resultado.ReciboRef != recibo.ReciboRef || repo.llamadas != 1 || autorizador.llamadas != 1 {
		t.Fatalf("resultado=%+v err=%v llamadas=%d", resultado, err, repo.llamadas)
	}
}

func TestServicioSituacionCambiaConMotivoYReciboDeterminista(t *testing.T) {
	ahora := time.Date(2026, 9, 21, 16, 0, 0, 0, time.UTC)
	repo := &repositorioSituacionPrueba{pertenece: true, vigente: puertosbolsa.SituacionParticipacion{ParticipacionRef: "participacion:b2", Situacion: "disponible", Desde: ahora}}
	autorizador := &autorizadorBorradorPrueba{t: t, instante: ahora}
	servicio, err := NuevoServicioSituacionParticipacion(contextoSituacionPrueba{}, autorizador, repo, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	s := solicitudSituacionPrueba(t, ahora)
	primero, err := servicio.Cambiar(context.Background(), s)
	if err != nil || primero.ReciboRef == "" || repo.llamadas != 1 || repo.ultimo.Material.ValidarEstructura() != nil {
		t.Fatalf("resultado=%+v err=%v llamadas=%d", primero, err, repo.llamadas)
	}
}

func TestServicioSituacionRechazaTransicionNoAdmitida(t *testing.T) {
	ahora := time.Date(2026, 9, 21, 16, 0, 0, 0, time.UTC)
	repo := &repositorioSituacionPrueba{pertenece: true, vigente: puertosbolsa.SituacionParticipacion{ParticipacionRef: "participacion:b2", Situacion: "disponible", Desde: ahora}}
	servicio, _ := NuevoServicioSituacionParticipacion(contextoSituacionPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, func() time.Time { return ahora.Add(-time.Microsecond) })
	s := solicitudSituacionPrueba(t, ahora)
	s.Destino = "trabajando"
	s.ClaveIdempotencia = "b2-cambio-0002"
	_, err := servicio.Cambiar(context.Background(), s)
	if err == nil || repo.llamadas != 0 {
		t.Fatalf("err=%v llamadas=%d", err, repo.llamadas)
	}
}

func TestServicioSituacionDeniegaAntesDeConsultarLaParticipacion(t *testing.T) {
	ahora := time.Date(2026, 9, 21, 16, 0, 0, 0, time.UTC)
	repo := &repositorioSituacionPrueba{pertenece: true}
	autorizador := &autorizadorBorradorPrueba{t: t, instante: ahora, err: dominiovec.ErrAutorizacionDenegada}
	servicio, _ := NuevoServicioSituacionParticipacion(contextoSituacionPrueba{}, autorizador, repo, func() time.Time { return ahora })
	_, err := servicio.Cambiar(context.Background(), solicitudSituacionPrueba(t, ahora))
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || repo.pertenencias != 1 || repo.lecturas != 0 || repo.llamadas != 0 {
		t.Fatalf("err=%v pertenencias=%d lecturas=%d escrituras=%d", err, repo.pertenencias, repo.lecturas, repo.llamadas)
	}
}

func TestServicioSituacionConservaDenegacionDeAmbitoSinConsultarDatos(t *testing.T) {
	ahora := time.Date(2026, 9, 21, 16, 0, 0, 0, time.UTC)
	repo := &repositorioSituacionPrueba{}
	autorizador := &autorizadorBorradorPrueba{t: t, instante: ahora}
	servicio, _ := NuevoServicioSituacionParticipacion(contextoSituacionPrueba{err: dominiovec.ErrAutorizacionDenegada}, autorizador, repo, func() time.Time { return ahora })
	_, err := servicio.Cambiar(context.Background(), solicitudSituacionPrueba(t, ahora))
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || repo.consultas != 0 || autorizador.llamadas != 0 {
		t.Fatalf("err=%v consultas=%d autorizaciones=%d", err, repo.consultas, autorizador.llamadas)
	}
}

func TestServicioSituacionDeniegaParticipacionAjenaAntesDeAutorizarOLeerEstado(t *testing.T) {
	ahora := time.Date(2026, 9, 21, 16, 0, 0, 0, time.UTC)
	repo := &repositorioSituacionPrueba{}
	autorizador := &autorizadorBorradorPrueba{t: t, instante: ahora}
	servicio, _ := NuevoServicioSituacionParticipacion(contextoSituacionPrueba{}, autorizador, repo, func() time.Time { return ahora })
	_, err := servicio.Cambiar(context.Background(), solicitudSituacionPrueba(t, ahora))
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || repo.pertenencias != 1 || repo.lecturas != 0 || autorizador.llamadas != 0 {
		t.Fatalf("err=%v pertenencias=%d lecturas=%d autorizaciones=%d", err, repo.pertenencias, repo.lecturas, autorizador.llamadas)
	}
}
