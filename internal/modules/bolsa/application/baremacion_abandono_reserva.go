package application

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

var ErrAbandonoReservaBaremacionNoAcreditado = errors.New(
	"bolsa: abandono de reserva de baremacion no acreditado",
)

const duracionMaximaAbandonoReservaBaremacion = 2 * time.Second

// abandonarReservaAntesDeConfirmar solo actua sobre una reserva nueva cuyo
// COMMIT aun no se ha invocado. Conserva los valores de enrutamiento del
// contexto, pero obtiene una sesion y una autorizacion nuevas dentro de un
// plazo independiente y acotado; no reutiliza capacidades previas.
func (s *ServicioBaremacion) abandonarReservaAntesDeConfirmar(
	ctx context.Context,
	actor ActorBaremacion,
	revision RevisionBaremacionIniciada,
	contenido dominiobolsa.ContenidoDecisionTecnica,
	solicitudReserva puertosbolsa.SolicitudReservarCambioBaremacion,
	reserva puertosbolsa.ReservaCambioBaremacion,
	referenciasAutorizacion map[string]struct{},
) error {
	if ctx == nil || s == nil || solicitudReserva.Validar() != nil ||
		reserva.ValidarPara(solicitudReserva) != nil || reserva.Repetida ||
		solicitudReserva.Clase != puertosbolsa.ClaseCambioIncorporarDecision ||
		solicitudReserva.BaremacionMeritoRef != contenido.BaremacionMeritoRef {
		return ErrAbandonoReservaBaremacionNoAcreditado
	}
	ahora, err := s.ahora()
	if err != nil {
		return ErrAbandonoReservaBaremacionNoAcreditado
	}
	restante := solicitudReserva.ExpiraEn.UTC().Sub(ahora)
	if restante <= 0 {
		return ErrAbandonoReservaBaremacionNoAcreditado
	}
	duracion := duracionMaximaAbandonoReservaBaremacion
	if restante < duracion {
		duracion = restante
	}
	ctxAbandono, cancelar := context.WithTimeout(context.WithoutCancel(ctx), duracion)
	defer cancelar()
	autorizacion, err := s.autorizarRevision(
		ctxAbandono, actor, revision,
		puertosbolsa.AccionAbandonarDecisionBaremacion,
		puertosbolsa.ClaseRecursoBaremacion,
		contenido.BaremacionMeritoRef,
		contenido.SujetoRef,
		contenido.FinalidadClave,
		contenido.CorrelacionRef,
	)
	if err != nil {
		return ErrAbandonoReservaBaremacionNoAcreditado
	}
	if _, repetida := referenciasAutorizacion[autorizacion.Proyeccion().AutorizacionRef]; repetida {
		return ErrAbandonoReservaBaremacionNoAcreditado
	}
	solicitud := puertosbolsa.SolicitudAbandonarReservaBaremacion{
		Contexto:            autorizacion,
		Token:               reserva.Token,
		Clase:               solicitudReserva.Clase,
		BaremacionMeritoRef: solicitudReserva.BaremacionMeritoRef,
	}
	if solicitud.Validar() != nil {
		return ErrAbandonoReservaBaremacionNoAcreditado
	}
	if err := s.repositorio.AbandonarReserva(ctxAbandono, solicitud); err == nil {
		return nil
	}
	if ctxAbandono.Err() != nil {
		return ErrAbandonoReservaBaremacionNoAcreditado
	}
	// La causa tecnica debe registrarse y expurgarse en el limite del conector;
	// esta capa solo conserva una clasificacion fija apta para propagacion.
	// El puerto es idempotente para la misma autorizacion y el mismo efecto.
	// Un unico reintento resuelve una respuesta perdida sin ampliar autoridad.
	if err := s.repositorio.AbandonarReserva(ctxAbandono, solicitud); err != nil {
		return ErrAbandonoReservaBaremacionNoAcreditado
	}
	return nil
}
