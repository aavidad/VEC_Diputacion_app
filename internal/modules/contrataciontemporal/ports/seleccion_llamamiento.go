package ports

import (
	"context"
	"errors"
	"time"
)

var ErrEjecucionSeleccionLlamamientoInvalida = errors.New(
	"contratacion temporal: ejecucion de seleccion y llamamiento invalida",
)

type SituacionEjecucionSeleccionLlamamiento string

const (
	EjecucionSeleccionLlamamientoPropietaria   SituacionEjecucionSeleccionLlamamiento = "propietaria"
	EjecucionSeleccionLlamamientoConfirmada    SituacionEjecucionSeleccionLlamamiento = "confirmada"
	EjecucionSeleccionLlamamientoOcupada       SituacionEjecucionSeleccionLlamamiento = "ocupada"
	EjecucionSeleccionLlamamientoColision      SituacionEjecucionSeleccionLlamamiento = "colision"
	EjecucionSeleccionLlamamientoIndeterminada SituacionEjecucionSeleccionLlamamiento = "indeterminada"
)

type EfectoSeleccionLlamamiento string

const (
	EfectoPrepararOrdenSeleccionLlamamiento EfectoSeleccionLlamamiento = "preparar_orden"
	EfectoSolicitarSeleccionLlamamiento     EfectoSeleccionLlamamiento = "solicitar_llamamiento"
)

// SolicitudReservaEjecucionSeleccionLlamamiento liga la UUID de intención a
// la orden gobernada y a la cantidad exacta observada antes del primer efecto.
// La huella no concede autoridad ni contiene posiciones o datos personales.
type SolicitudReservaEjecucionSeleccionLlamamiento struct {
	ClaveIdempotencia string
	HuellaSemantica   string
}

func NuevaSolicitudReservaEjecucionSeleccionLlamamiento(
	clave string,
	comando ComandoPrepararOrdenBolsa,
	cantidadDisponible uint32,
	instante time.Time,
) (SolicitudReservaEjecucionSeleccionLlamamiento, error) {
	if !ClaveIdempotenciaValida(clave) || cantidadDisponible == 0 ||
		comando.ValidarEn(instante) != nil ||
		comando.MaximoPosiciones < cantidadDisponible {
		return SolicitudReservaEjecucionSeleccionLlamamiento{},
			ErrEjecucionSeleccionLlamamientoInvalida
	}
	contexto, err := comando.Contexto.DatosEn(instante)
	if err != nil {
		return SolicitudReservaEjecucionSeleccionLlamamiento{},
			ErrEjecucionSeleccionLlamamientoInvalida
	}
	canon := nuevoCanonicoBolsa("ejecucion-seleccion-llamamiento")
	canon.campo("clave_idempotencia", clave)
	canon.campo("organizacion_ref", contexto.OrganizacionRef)
	canon.campo("expediente_ref", contexto.ExpedienteRef)
	canon.entero("version_expediente", contexto.VersionExpediente)
	canon.campo("correlacion_ref", contexto.CorrelacionRef)
	canon.referencia("accion", contexto.Accion)
	canon.referencia("recurso", contexto.Recurso)
	canon.referencia("finalidad", contexto.Finalidad)
	canon.referencia("necesidad", comando.Necesidad)
	canon.referencia("bolsa", comando.Bolsa)
	canon.referencia("politica", comando.Politica)
	canon.entero("maximo_posiciones", uint64(comando.MaximoPosiciones))
	canon.entero("cantidad_disponible", uint64(cantidadDisponible))
	return SolicitudReservaEjecucionSeleccionLlamamiento{
		ClaveIdempotencia: clave,
		HuellaSemantica:   huellaBytesBolsa(canon.bytes()),
	}, nil
}

type ReservaEjecucionSeleccionLlamamiento struct {
	Solicitud  SolicitudReservaEjecucionSeleccionLlamamiento
	ReservaRef string
}

type EstadoEjecucionSeleccionLlamamiento struct {
	Solicitud        SolicitudReservaEjecucionSeleccionLlamamiento
	Situacion        SituacionEjecucionSeleccionLlamamiento
	ReservaRef       string
	EfectoPosible    EfectoSeleccionLlamamiento
	ReciboConfirmado ReciboSolicitudLlamamientoBolsa
}

// EjecucionesSeleccionLlamamiento es el límite atómico de idempotencia. Una
// implementación durable debe comparar UUID y huella en la misma operación de
// reserva, mantener una reserva viva como ocupada y convertir una reserva
// abandonada después de AbrirVentanaEfecto en indeterminada, nunca liberarla
// para repetir a ciegas. LiberarAntesDeEfectos solo admite una ejecución que
// todavía no cruzó ninguna ventana. Confirmar conserva el recibo exacto.
// Este contrato no aporta persistencia: la composición real requiere un
// adaptador durable y recuperación explícita de estados indeterminados.
type EjecucionesSeleccionLlamamiento interface {
	Reservar(
		context.Context,
		SolicitudReservaEjecucionSeleccionLlamamiento,
	) (EstadoEjecucionSeleccionLlamamiento, error)
	AbrirVentanaEfecto(
		context.Context,
		ReservaEjecucionSeleccionLlamamiento,
		EfectoSeleccionLlamamiento,
	) error
	MarcarIndeterminada(
		context.Context,
		ReservaEjecucionSeleccionLlamamiento,
		EfectoSeleccionLlamamiento,
	) error
	LiberarAntesDeEfectos(
		context.Context,
		ReservaEjecucionSeleccionLlamamiento,
	) error
	Confirmar(
		context.Context,
		ReservaEjecucionSeleccionLlamamiento,
		ReciboSolicitudLlamamientoBolsa,
	) error
	ConsultarEstado(
		context.Context,
		SolicitudReservaEjecucionSeleccionLlamamiento,
	) (EstadoEjecucionSeleccionLlamamiento, error)
}

// PreparadorSeleccionLlamamiento resuelve desde una referencia de intención
// no autoritativa los contextos, la política y los límites gobernados de cada
// paso. No selecciona una posición ni ejecuta efectos en Bolsa.
type PreparadorSeleccionLlamamiento interface {
	PrepararConsultaDisponibilidad(
		context.Context,
		string,
	) (SolicitudDisponibilidadBolsa, error)
	PrepararOrdenCompleto(
		context.Context,
		string,
		ResultadoDisponibilidadBolsa,
	) (ComandoPrepararOrdenBolsa, error)
	PrepararContextoLlamamiento(
		context.Context,
		string,
		ReciboOrdenBolsa,
	) (ContextoPeticionIntegracionBolsa, error)
}
