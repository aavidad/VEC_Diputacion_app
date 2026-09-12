package application

import (
	"context"
	"errors"
	"sync"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/i18n"
)

var (
	ErrServicioCorreoLlamamientoInvalido     = errors.New("contratacion temporal: servicio de correo de llamamiento invalido")
	ErrCorreoLlamamientoDenegado             = errors.New("contratacion temporal: correo de llamamiento denegado")
	ErrCorreoLlamamientoNoDisponible         = errors.New("contratacion temporal: correo de llamamiento no disponible")
	ErrResultadoCorreoLlamamientoNoConfiable = errors.New("contratacion temporal: resultado de correo de llamamiento no confiable")
)

const tiempoMaximoRegistroResultadoCorreoLlamamiento = 2 * time.Second

type ServicioCorreoLlamamiento struct {
	autorizador ports.AutorizadorDespachoCorreoLlamamiento
	resultado   ports.AutorizadorResultadoCorreoLlamamiento
	auditoria   ports.PreparadorAuditoriaResultadoCorreoLlamamiento
	resolutor   ports.ResolutorDestinoCorreoLlamamiento
	registro    ports.RegistroIntentosCorreoLlamamiento
	transporte  ports.TransportadorCorreoLlamamiento
	reloj       ports.Reloj
	traductor   interfazTraductorCorreo
}

type interfazTraductorCorreo interface {
	Message(string, string) (string, bool)
}

const (
	claveAsuntoCorreoLlamamiento = "contratacion_temporal.correo_llamamiento.asunto"
	claveCuerpoCorreoLlamamiento = "contratacion_temporal.correo_llamamiento.cuerpo"
)

func NuevoServicioCorreoLlamamiento(a ports.AutorizadorDespachoCorreoLlamamiento, resultado ports.AutorizadorResultadoCorreoLlamamiento, auditoria ports.PreparadorAuditoriaResultadoCorreoLlamamiento, r ports.ResolutorDestinoCorreoLlamamiento, registro ports.RegistroIntentosCorreoLlamamiento, t ports.TransportadorCorreoLlamamiento, reloj ports.Reloj, traductor interfazTraductorCorreo) (*ServicioCorreoLlamamiento, error) {
	if dependenciaNula(a) || dependenciaNula(resultado) || dependenciaNula(auditoria) || dependenciaNula(r) || dependenciaNula(registro) || dependenciaNula(t) || dependenciaNula(reloj) || dependenciaNula(traductor) {
		return nil, ErrServicioCorreoLlamamientoInvalido
	}
	return &ServicioCorreoLlamamiento{autorizador: a, resultado: resultado, auditoria: auditoria, resolutor: r, registro: registro, transporte: t, reloj: reloj, traductor: traductor}, nil
}

// Despachar reserva una vez y nunca repite SMTP si esa reserva ya existe. La
// aceptación del relay no afirma entrega, plazo ni efecto en Bolsa.
func (s *ServicioCorreoLlamamiento) Despachar(ctx context.Context, solicitud ports.SolicitudDespacharCorreoLlamamiento) (ports.ReservaIntentoCorreoLlamamiento, error) {
	cero := ports.ReservaIntentoCorreoLlamamiento{}
	if s == nil || ctx == nil || dependenciaNula(s.autorizador) || dependenciaNula(s.resultado) || dependenciaNula(s.auditoria) || dependenciaNula(s.resolutor) || dependenciaNula(s.registro) || dependenciaNula(s.transporte) || dependenciaNula(s.reloj) || dependenciaNula(s.traductor) {
		return cero, ErrServicioCorreoLlamamientoInvalido
	}
	if solicitud.Validar() != nil {
		return cero, ErrCorreoLlamamientoDenegado
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	capacidad, err := s.autorizador.AutorizarDespachoCorreoLlamamiento(ctx, solicitud)
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if err != nil || ValidarCapacidadDespachoCorreoLlamamiento(capacidad, solicitud, s.reloj.Ahora()) != nil {
		return cero, ErrCorreoLlamamientoDenegado
	}
	reserva, finalizacion, err := s.registro.ReservarIntentoCorreoLlamamiento(ctx, solicitud, capacidad)
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if err != nil {
		return cero, ErrCorreoLlamamientoNoDisponible
	}
	if reserva.ValidarPara(solicitud) != nil {
		return cero, ErrResultadoCorreoLlamamientoNoConfiable
	}
	if reserva.YaReservado {
		if !finalizacion.EsCero() {
			return cero, ErrResultadoCorreoLlamamientoNoConfiable
		}
		return reserva, nil
	}
	if finalizacion.ValidarPara(reserva) != nil {
		return cero, ErrResultadoCorreoLlamamientoNoConfiable
	}
	estado := ports.CorreoLlamamientoNoAceptadoTransitorio
	callbacks := 0
	var callbackMu sync.Mutex
	callbackCerrado, callbackInvalido := false, false
	err = s.resolutor.ConDestinoCorreoLlamamiento(ctx, solicitud, capacidad, func(destino ports.DestinoCorreoLlamamiento) error {
		callbackMu.Lock()
		defer callbackMu.Unlock()
		if callbackCerrado {
			return ErrResultadoCorreoLlamamientoNoConfiable
		}
		callbacks++
		if callbacks > 1 {
			callbackInvalido = true
			return ErrResultadoCorreoLlamamientoNoConfiable
		}
		mensaje, mensajeErr := destinoMensajeCorreo(destino, reserva, s.traductor)
		if mensajeErr != nil {
			callbackInvalido = true
			return mensajeErr
		}
		estado = s.transporte.Enviar(ctx, mensaje)
		if !estado.EsResultado() {
			estado = ports.CorreoLlamamientoIndeterminado
			callbackInvalido = true
			return ErrResultadoCorreoLlamamientoNoConfiable
		}
		return nil
	})
	callbackMu.Lock()
	callbackCerrado = true
	if callbacks != 1 || callbackInvalido {
		err = ErrResultadoCorreoLlamamientoNoConfiable
	}
	callbackMu.Unlock()
	// Conserva el vínculo del request sin su cancelación; el límite evita una
	// escritura huérfana y deja la reserva como barrera frente al reenvío ciego.
	registroCtx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), tiempoMaximoRegistroResultadoCorreoLlamamiento)
	solicitudResultado, errSolicitud := ports.NuevaSolicitudRegistrarResultadoCorreoLlamamiento(solicitud, reserva, estado, ports.PlantillaCorreoLlamamientoV1)
	if errSolicitud != nil {
		cancelar()
		return reserva, ErrResultadoCorreoLlamamientoNoConfiable
	}
	auditoria, errAuditoria := s.auditoria.PrepararAuditoriaResultadoCorreoLlamamiento(registroCtx, solicitudResultado, capacidad, s.reloj.Ahora())
	if errAuditoria != nil {
		cancelar()
		return reserva, ErrCorreoLlamamientoNoDisponible
	}
	capacidadResultado, errAutorizacion := s.resultado.AutorizarResultadoCorreoLlamamiento(registroCtx, solicitudResultado, auditoria)
	if errAutorizacion != nil || ValidarCapacidadResultadoCorreoLlamamiento(capacidadResultado, solicitudResultado, auditoria, s.reloj.Ahora()) != nil {
		cancelar()
		return reserva, ErrCorreoLlamamientoNoDisponible
	}
	errRegistro := s.registro.RegistrarResultadoIntentoCorreoLlamamiento(registroCtx, solicitudResultado, finalizacion, auditoria, capacidadResultado)
	cancelar()
	if errRegistro != nil {
		return reserva, ErrCorreoLlamamientoNoDisponible
	}
	reserva.Estado = estado
	if err != nil {
		return reserva, ErrCorreoLlamamientoDenegado
	}
	if err := ctx.Err(); err != nil {
		return reserva, err
	}
	return reserva, nil
}
func destinoMensajeCorreo(d ports.DestinoCorreoLlamamiento, r ports.ReservaIntentoCorreoLlamamiento, traductor interfazTraductorCorreo) (ports.MensajeCorreoLlamamiento, error) {
	asunto, asuntoOk := traductor.Message(i18n.DefaultLocale, claveAsuntoCorreoLlamamiento)
	cuerpo, cuerpoOk := traductor.Message(i18n.DefaultLocale, claveCuerpoCorreoLlamamiento)
	if !asuntoOk || !cuerpoOk || asunto == "" || cuerpo == "" {
		return ports.MensajeCorreoLlamamiento{}, ErrServicioCorreoLlamamientoInvalido
	}
	m, err := d.Mensaje(asunto, cuerpo)
	if err != nil {
		return ports.MensajeCorreoLlamamiento{}, ErrResultadoCorreoLlamamientoNoConfiable
	}
	m.MessageID = r.MessageID
	m.FechaOrigen = r.FechaOrigen
	return m, nil
}
