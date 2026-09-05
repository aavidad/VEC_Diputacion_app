package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Los mismos sentinelas sirven a persistencia, aplicación y transporte.
var (
	ErrServicioRespuestasRecibidasInvalido   = errors.New("ct_respuesta_recibida_servicio_invalido")
	ErrSolicitudRespuestaRecibidaInvalida    = ports.ErrSolicitudRespuestaRecibidaInvalida
	ErrClaveRespuestaRecibidaEnColision      = ports.ErrClaveRespuestaRecibidaUsada
	ErrVersionRespuestaRecibidaEnConflicto   = ports.ErrVersionRespuestaRecibidaEnConflicto
	ErrRespuestaRecibidaDenegada             = ports.ErrOperacionRespuestaRecibidaDenegada
	ErrRespuestaRecibidaNoDisponible         = ports.ErrRespuestaRecibidaNoDisponible
	ErrResultadoRespuestaRecibidaNoConfiable = ports.ErrResultadoRespuestaRecibidaNoConfiable
)

// ServicioRespuestasRecibidas registra declaraciones, no resuelve llamamientos.
// No mantiene caché de recibos: cada llamada, incluido un replay, pasa por
// el registro que consume autorización fresca junto con su efecto.
type ServicioRespuestasRecibidas struct {
	registro ports.RegistroRespuestasRecibidas
}

func NuevoServicioRespuestasRecibidas(registro ports.RegistroRespuestasRecibidas) (*ServicioRespuestasRecibidas, error) {
	if dependenciaNula(registro) {
		return nil, ErrServicioRespuestasRecibidasInvalido
	}
	return &ServicioRespuestasRecibidas{registro: registro}, nil
}

func (s *ServicioRespuestasRecibidas) Registrar(
	ctx context.Context,
	solicitud ports.SolicitudRegistrarRespuestaRecibida,
) (ports.RespuestaRecibidaRegistrada, error) {
	if s == nil || ctx == nil || dependenciaNula(s.registro) {
		return ports.RespuestaRecibidaRegistrada{}, ErrServicioRespuestasRecibidasInvalido
	}
	if solicitud.Validar() != nil {
		return ports.RespuestaRecibidaRegistrada{}, ErrSolicitudRespuestaRecibidaInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.RespuestaRecibidaRegistrada{}, err
	}
	resultado, err := s.registro.RegistrarRespuestaRecibida(ctx, solicitud)
	if errContexto := ctx.Err(); errContexto != nil {
		return ports.RespuestaRecibidaRegistrada{}, errContexto
	}
	if err != nil {
		if resultado != (ports.RespuestaRecibidaRegistrada{}) {
			return ports.RespuestaRecibidaRegistrada{}, ErrResultadoRespuestaRecibidaNoConfiable
		}
		return ports.RespuestaRecibidaRegistrada{}, clasificarErrorRespuestaRecibida(err)
	}
	if resultado.ValidarPara(solicitud) != nil {
		return ports.RespuestaRecibidaRegistrada{}, ErrResultadoRespuestaRecibidaNoConfiable
	}
	return resultado, nil
}

// No propaga detalles internos ni datos del proveedor.
func clasificarErrorRespuestaRecibida(err error) error {
	for _, conocido := range []error{
		context.Canceled, context.DeadlineExceeded,
		ErrSolicitudRespuestaRecibidaInvalida,
		ErrRespuestaRecibidaDenegada,
		ErrClaveRespuestaRecibidaEnColision,
		ErrVersionRespuestaRecibidaEnConflicto,
		ErrResultadoRespuestaRecibidaNoConfiable,
	} {
		if errors.Is(err, conocido) {
			return conocido
		}
	}
	return ErrRespuestaRecibidaNoDisponible
}
