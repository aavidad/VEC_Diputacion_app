package application

import (
	"context"
	"errors"
	"sync"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type resolutorCandidaturaDoble struct {
	candidatura ports.CandidaturaAlta
	err         error
	llamadas    int
	solicitud   ports.SolicitudResolverCandidaturaAlta
	antes       func()
}

func (d *resolutorCandidaturaDoble) ResolverCandidaturaAlta(
	_ context.Context,
	solicitud ports.SolicitudResolverCandidaturaAlta,
) (ports.CandidaturaAlta, error) {
	d.llamadas++
	d.solicitud = solicitud
	if d.antes != nil {
		d.antes()
	}
	if d.candidatura == (ports.CandidaturaAlta{}) {
		return solicitud.Propuesta, d.err
	}
	return d.candidatura, d.err
}

type resolutorCandidaturaDurablePrueba struct {
	mu          sync.Mutex
	candidatura *ports.CandidaturaAlta
	llamadas    int
}

func (r *resolutorCandidaturaDurablePrueba) ResolverCandidaturaAlta(
	_ context.Context,
	solicitud ports.SolicitudResolverCandidaturaAlta,
) (ports.CandidaturaAlta, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llamadas++
	if r.candidatura == nil {
		candidatura := solicitud.Propuesta
		r.candidatura = &candidatura
		return candidatura, nil
	}
	if solicitud.ValidarResultado(*r.candidatura) != nil {
		return ports.CandidaturaAlta{}, ports.ErrClaveIdempotenciaUsada
	}
	return *r.candidatura, nil
}

func escenarioConReferenciasNuevas(
	t *testing.T,
	base escenarioRegistro,
) escenarioRegistro {
	t.Helper()
	nuevo := base
	nuevo.candidatura.ReservaRef = "reserva:alta-reinicio-002"
	nuevo.candidatura.Referencias = ports.ReferenciasAlta{
		ExpedienteRef: "expediente:ct-2026-0099",
		NumeroVisible: "2026/CT-0099",
		ReciboRef:     "recibo:alta-reinicio-002",
	}
	nuevo.recibo.ExpedienteRef = nuevo.candidatura.Referencias.ExpedienteRef
	nuevo.recibo.NumeroVisible = nuevo.candidatura.Referencias.NumeroVisible
	nuevo.recibo.ReciboRef = nuevo.candidatura.Referencias.ReciboRef
	huella, err := ports.CalcularHuellaReciboAlta(nuevo.recibo)
	if err != nil {
		t.Fatal(err)
	}
	nuevo.recibo.ReciboHuellaSHA256 = huella
	return nuevo
}

func TestRegistroSolicitudRecuperaCandidaturaEntreServiciosNuevos(t *testing.T) {
	primero := nuevoEscenarioRegistro(t)
	segundo := escenarioConReferenciasNuevas(t, primero)
	durable := &resolutorCandidaturaDurablePrueba{}

	servicioPrimero, dependenciasPrimero := construirServicioRegistro(t, primero)
	servicioPrimero.candidaturas = durable
	reciboPrimero, err := servicioPrimero.Registrar(
		context.Background(),
		primero.solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}

	servicioSegundo, dependenciasSegundo := construirServicioRegistro(t, segundo)
	servicioSegundo.candidaturas = durable
	dependenciasSegundo.transaccion.recibo = primero.recibo
	reciboSegundo, err := servicioSegundo.Registrar(
		context.Background(),
		segundo.solicitud,
	)
	if err != nil || reciboSegundo != reciboPrimero ||
		durable.llamadas != 2 ||
		dependenciasPrimero.referencias.llamadasReferencias != 1 ||
		dependenciasSegundo.referencias.llamadasReferencias != 1 {
		t.Fatalf(
			"replay tras reinicio divergente: %#v / %#v / %v / llamadas=%d",
			reciboPrimero,
			reciboSegundo,
			err,
			durable.llamadas,
		)
	}
	datosOrden, err := dependenciasSegundo.transaccion.orden.Datos()
	if err != nil || datosOrden.Candidatura.Referencias !=
		primero.candidatura.Referencias {
		t.Fatalf("la segunda instancia no recuperó la candidatura: %#v / %v",
			datosOrden.Candidatura, err)
	}
}

func TestRegistroSolicitudRechazaMismoAmbitoConOtraHuella(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	durable := &resolutorCandidaturaDurablePrueba{}
	servicio, _ := construirServicioRegistro(t, escenario)
	servicio.candidaturas = durable
	if _, err := servicio.Registrar(
		context.Background(),
		escenario.solicitud,
	); err != nil {
		t.Fatal(err)
	}

	segundo, dependencias := construirServicioRegistro(t, escenario)
	segundo.candidaturas = durable
	huellaActivaDistinta := selloHMACRegistroPrueba(
		"vec.contratacion-temporal.huella-peticion/v2",
		"e",
	)
	huellaRetenidaDistinta := selloHMACRegistroPrueba(
		clavePeticionRegistroPrueba,
		"f",
	)
	var err error
	dependencias.huellas.coleccion, err = ports.NuevaColeccionSellosHMAC(
		huellaActivaDistinta,
		[]string{huellaRetenidaDistinta},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = segundo.Registrar(context.Background(), escenario.solicitud)
	if !errors.Is(err, ports.ErrClaveIdempotenciaUsada) ||
		dependencias.autorizador.llamadas != 0 ||
		dependencias.transaccion.llamadas != 0 {
		t.Fatalf("ámbito reutilizado con otra huella aceptado: %v", err)
	}
}
