package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var (
	ErrServicioComunicacionLlamamientoInvalido = errors.New(
		"contratacion temporal: servicio de comunicacion de llamamiento invalido",
	)
	ErrSolicitudComunicacionLlamamientoInvalida = errors.New(
		"contratacion temporal: solicitud de comunicacion de llamamiento invalida",
	)
	ErrComunicacionLlamamientoDenegada = errors.New(
		"contratacion temporal: comunicacion de llamamiento denegada",
	)
	ErrVersionComunicacionLlamamientoEnConflicto = errors.New(
		"contratacion temporal: version de comunicacion de llamamiento en conflicto",
	)
	ErrClaveComunicacionLlamamientoEnColision = errors.New(
		"contratacion temporal: clave de comunicacion de llamamiento usada con otros datos",
	)
	ErrComunicacionLlamamientoNoDisponible = errors.New(
		"contratacion temporal: comunicacion de llamamiento no disponible",
	)
	ErrResultadoComunicacionLlamamientoNoConfiable = errors.New(
		"contratacion temporal: resultado de comunicacion de llamamiento no confiable",
	)
)

type ServicioComunicacionLlamamiento struct {
	transaccion ports.TransaccionComunicacionLlamamiento
}

func NuevoServicioComunicacionLlamamiento(
	transaccion ports.TransaccionComunicacionLlamamiento,
) (*ServicioComunicacionLlamamiento, error) {
	if dependenciaNula(transaccion) {
		return nil, ErrServicioComunicacionLlamamientoInvalido
	}
	return &ServicioComunicacionLlamamiento{
		transaccion: transaccion,
	}, nil
}

func (s *ServicioComunicacionLlamamiento) Registrar(
	ctx context.Context,
	solicitud ports.SolicitudRegistrarComunicacionLlamamiento,
) (ports.ComunicacionProbatoria, error) {
	if s == nil || ctx == nil || dependenciaNula(s.transaccion) {
		return ports.ComunicacionProbatoria{},
			ErrServicioComunicacionLlamamientoInvalido
	}
	if solicitud.Validar() != nil {
		return ports.ComunicacionProbatoria{},
			ErrSolicitudComunicacionLlamamientoInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ComunicacionProbatoria{}, err
	}
	resultado, err := s.transaccion.RegistrarComunicacion(ctx, solicitud)
	if errContexto := ctx.Err(); errContexto != nil {
		return ports.ComunicacionProbatoria{}, errContexto
	}
	if err != nil {
		if errContexto := ctx.Err(); errContexto != nil {
			return ports.ComunicacionProbatoria{}, errContexto
		}
		if resultado != (ports.ComunicacionProbatoria{}) {
			return ports.ComunicacionProbatoria{},
				ErrResultadoComunicacionLlamamientoNoConfiable
		}
		return ports.ComunicacionProbatoria{},
			clasificarErrorComunicacionLlamamiento(ctx, err)
	}
	if resultado.ValidarPara(solicitud) == nil {
		return resultado, nil
	}
	return ports.ComunicacionProbatoria{},
		ErrResultadoComunicacionLlamamientoNoConfiable
}

func (s *ServicioComunicacionLlamamiento) Resolver(
	ctx context.Context,
	solicitud ports.SolicitudResolverLlamamiento,
) (ports.ResultadoResolucionLlamamiento, error) {
	if s == nil || ctx == nil || dependenciaNula(s.transaccion) {
		return ports.ResultadoResolucionLlamamiento{},
			ErrServicioComunicacionLlamamientoInvalido
	}
	if solicitud.Validar() != nil {
		return ports.ResultadoResolucionLlamamiento{},
			ErrSolicitudComunicacionLlamamientoInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoResolucionLlamamiento{}, err
	}
	resultado, err := s.transaccion.ResolverLlamamiento(ctx, solicitud)
	if errContexto := ctx.Err(); errContexto != nil {
		return ports.ResultadoResolucionLlamamiento{}, errContexto
	}
	if err != nil {
		if errContexto := ctx.Err(); errContexto != nil {
			return ports.ResultadoResolucionLlamamiento{}, errContexto
		}
		if resultado != (ports.ResultadoResolucionLlamamiento{}) {
			return ports.ResultadoResolucionLlamamiento{},
				ErrResultadoComunicacionLlamamientoNoConfiable
		}
		return ports.ResultadoResolucionLlamamiento{},
			clasificarErrorComunicacionLlamamiento(ctx, err)
	}
	if resultado.ValidarPara(solicitud) == nil {
		return resultado, nil
	}
	return ports.ResultadoResolucionLlamamiento{},
		ErrResultadoComunicacionLlamamientoNoConfiable
}

func clasificarErrorComunicacionLlamamiento(ctx context.Context, err error) error {
	if ctx != nil {
		if errContexto := ctx.Err(); errContexto != nil {
			return errContexto
		}
	}
	switch {
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return context.DeadlineExceeded
	case errors.Is(err, ports.ErrOperacionComunicacionLlamamientoDenegada):
		return ErrComunicacionLlamamientoDenegada
	case errors.Is(err, ports.ErrVersionComunicacionLlamamientoEnConflicto):
		return ErrVersionComunicacionLlamamientoEnConflicto
	case errors.Is(err, ports.ErrClaveComunicacionLlamamientoUsada):
		return ErrClaveComunicacionLlamamientoEnColision
	default:
		return ErrComunicacionLlamamientoNoDisponible
	}
}
