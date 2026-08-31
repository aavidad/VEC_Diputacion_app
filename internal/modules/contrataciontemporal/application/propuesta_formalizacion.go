package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var (
	ErrServicioPropuestaFormalizacionInvalido = errors.New(
		"contratacion temporal: servicio de propuesta de formalizacion invalido",
	)
	ErrSolicitudPropuestaFormalizacionInvalida = errors.New(
		"contratacion temporal: solicitud de propuesta de formalizacion invalida",
	)
	ErrPropuestaFormalizacionDenegada = errors.New(
		"contratacion temporal: propuesta de formalizacion denegada",
	)
	ErrVersionPropuestaFormalizacionEnConflicto = errors.New(
		"contratacion temporal: version de propuesta de formalizacion en conflicto",
	)
	ErrClavePropuestaFormalizacionEnColision = errors.New(
		"contratacion temporal: clave de propuesta de formalizacion usada con otros datos",
	)
	ErrResolucionFormalizacionNoAceptada = errors.New(
		"contratacion temporal: resolucion de llamamiento no aceptada para formalizacion",
	)
	ErrPropuestaFormalizacionNoDisponible = errors.New(
		"contratacion temporal: propuesta de formalizacion no disponible",
	)
	ErrResultadoPropuestaFormalizacionNoConfiable = errors.New(
		"contratacion temporal: resultado de propuesta de formalizacion no confiable",
	)
)

type ServicioPropuestaFormalizacion struct {
	transaccion ports.TransaccionPropuestaFormalizacion
}

func NuevoServicioPropuestaFormalizacion(
	transaccion ports.TransaccionPropuestaFormalizacion,
) (*ServicioPropuestaFormalizacion, error) {
	if dependenciaNula(transaccion) {
		return nil, ErrServicioPropuestaFormalizacionInvalido
	}
	return &ServicioPropuestaFormalizacion{transaccion: transaccion}, nil
}

// PrepararYConfirmar normaliza una intencion minimizada y delega un unico
// commit local. La cancelacion observada tras el puerto prevalece incluso si
// este devuelve resultado valido y error nil.
func (s *ServicioPropuestaFormalizacion) PrepararYConfirmar(
	ctx context.Context,
	solicitud ports.SolicitudPropuestaFormalizacion,
) (ports.ResultadoPropuestaFormalizacion, error) {
	if s == nil || ctx == nil || dependenciaNula(s.transaccion) {
		return ports.ResultadoPropuestaFormalizacion{},
			ErrServicioPropuestaFormalizacionInvalido
	}
	if errContexto := ctx.Err(); errContexto != nil {
		return ports.ResultadoPropuestaFormalizacion{}, errContexto
	}
	normalizada, err := solicitud.Normalizar()
	if err != nil || normalizada.Validar() != nil {
		return ports.ResultadoPropuestaFormalizacion{},
			ErrSolicitudPropuestaFormalizacionInvalida
	}
	if errContexto := ctx.Err(); errContexto != nil {
		return ports.ResultadoPropuestaFormalizacion{}, errContexto
	}

	resultado, err := s.transaccion.ConfirmarPropuesta(ctx, normalizada.Clonar())
	if errContexto := ctx.Err(); errContexto != nil {
		return ports.ResultadoPropuestaFormalizacion{}, errContexto
	}
	if err != nil {
		if errContexto := ctx.Err(); errContexto != nil {
			return ports.ResultadoPropuestaFormalizacion{}, errContexto
		}
		if !resultado.EsCero() {
			return ports.ResultadoPropuestaFormalizacion{},
				ErrResultadoPropuestaFormalizacionNoConfiable
		}
		return ports.ResultadoPropuestaFormalizacion{},
			clasificarErrorPropuestaFormalizacion(ctx, err)
	}
	if resultado.ValidarPara(normalizada) != nil {
		return ports.ResultadoPropuestaFormalizacion{},
			ErrResultadoPropuestaFormalizacionNoConfiable
	}
	return resultado.Clonar(), nil
}

func clasificarErrorPropuestaFormalizacion(ctx context.Context, err error) error {
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
	case errors.Is(err, ports.ErrOperacionPropuestaFormalizacionDenegada):
		return ErrPropuestaFormalizacionDenegada
	case errors.Is(err, ports.ErrVersionPropuestaFormalizacionEnConflicto):
		return ErrVersionPropuestaFormalizacionEnConflicto
	case errors.Is(err, ports.ErrClavePropuestaFormalizacionUsada):
		return ErrClavePropuestaFormalizacionEnColision
	case errors.Is(err, ports.ErrResolucionLlamamientoNoAceptada):
		return ErrResolucionFormalizacionNoAceptada
	default:
		return ErrPropuestaFormalizacionNoDisponible
	}
}
