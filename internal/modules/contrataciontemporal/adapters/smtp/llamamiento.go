package smtp

import (
	"context"
	"errors"
	"reflect"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var ErrTransportadorCorreoLlamamientoInvalido = errors.New("contratacion temporal smtp: transportador de correo de llamamiento invalido")

type enviadorSMTP interface {
	Enviar(context.Context, Mensaje) Resultado
}

// TransportadorLlamamiento adapta el puerto de aplicación al transporte SMTP.
// No reserva intentos ni decide reintentos: esas responsabilidades quedan en
// el caso de uso que proporciona MessageID y FechaOrigen estables.
type TransportadorLlamamiento struct{ transporte enviadorSMTP }

func NuevoTransportadorLlamamiento(transporte *Adaptador) (*TransportadorLlamamiento, error) {
	return nuevoTransportadorLlamamiento(transporte)
}

func nuevoTransportadorLlamamiento(transporte enviadorSMTP) (*TransportadorLlamamiento, error) {
	if dependenciaNulaSMTP(transporte) {
		return nil, ErrTransportadorCorreoLlamamientoInvalido
	}
	return &TransportadorLlamamiento{transporte: transporte}, nil
}

func (t *TransportadorLlamamiento) Enviar(ctx context.Context, mensaje ports.MensajeCorreoLlamamiento) ports.EstadoCorreoLlamamiento {
	if t == nil || dependenciaNulaSMTP(t.transporte) || ctx == nil {
		return ports.CorreoLlamamientoIndeterminado
	}
	destino, asunto, cuerpo := mensaje.Datos()
	resultado := t.transporte.Enviar(ctx, Mensaje{Destino: destino, Asunto: asunto, Cuerpo: cuerpo, MessageID: mensaje.MessageID, FechaOrigen: mensaje.FechaOrigen})
	switch resultado.Estado {
	case NoAceptadoTransitorio:
		return ports.CorreoLlamamientoNoAceptadoTransitorio
	case NoAceptadoPermanente:
		return ports.CorreoLlamamientoNoAceptadoPermanente
	case AceptadoPorRelay:
		return ports.CorreoLlamamientoAceptadoPorRelay
	case Indeterminado:
		return ports.CorreoLlamamientoIndeterminado
	default:
		return ports.CorreoLlamamientoIndeterminado
	}
}

func dependenciaNulaSMTP(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return r.IsNil()
	}
	return false
}
