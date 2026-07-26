package application

import (
	"context"
	"errors"
	"log/slog"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var (
	ErrServicioOperacionAnalisisInvalido = errors.New(
		"contratacion temporal: servicio de operacion de analisis invalido",
	)
	ErrSolicitudOperacionAnalisisInvalida = errors.New(
		"contratacion temporal: solicitud de operacion de analisis invalida",
	)
	ErrOperacionAnalisisDenegada = errors.New(
		"contratacion temporal: operacion de analisis denegada",
	)
	ErrDependenciaOperacionAnalisisNoDisponible = errors.New(
		"contratacion temporal: dependencia de operacion de analisis no disponible",
	)
	ErrOperacionAnalisisEnConflicto = errors.New(
		"contratacion temporal: operacion de analisis en conflicto",
	)
	ErrResultadoOperacionAnalisisNoConfiable = errors.New(
		"contratacion temporal: resultado de operacion de analisis no confiable",
	)
)

type tipoErrorOperacionAnalisis uint8

const (
	tipoErrorSolicitud tipoErrorOperacionAnalisis = iota + 1
	tipoErrorDenegacion
	tipoErrorDependencia
	tipoErrorConflicto
	tipoErrorResultado
)

type errorOperacionAnalisis struct {
	tipo     tipoErrorOperacionAnalisis
	contexto error
}

func nuevoErrorOperacionAnalisis(
	tipo tipoErrorOperacionAnalisis,
	contexto error,
) error {
	if !errors.Is(contexto, context.Canceled) &&
		!errors.Is(contexto, context.DeadlineExceeded) {
		contexto = nil
	}
	return errorOperacionAnalisis{tipo: tipo, contexto: contexto}
}

func (e errorOperacionAnalisis) Error() string {
	switch e.tipo {
	case tipoErrorSolicitud:
		return ErrSolicitudOperacionAnalisisInvalida.Error()
	case tipoErrorDenegacion:
		return ErrOperacionAnalisisDenegada.Error()
	case tipoErrorConflicto:
		return ErrOperacionAnalisisEnConflicto.Error()
	case tipoErrorResultado:
		return ErrResultadoOperacionAnalisisNoConfiable.Error()
	default:
		return ErrDependenciaOperacionAnalisisNoDisponible.Error()
	}
}

func (e errorOperacionAnalisis) Unwrap() error {
	return e.contexto
}

func (e errorOperacionAnalisis) Is(objetivo error) bool {
	switch e.tipo {
	case tipoErrorSolicitud:
		return objetivo == ErrSolicitudOperacionAnalisisInvalida
	case tipoErrorDenegacion:
		return objetivo == ErrOperacionAnalisisDenegada ||
			objetivo == ports.ErrAutorizacionDenegada
	case tipoErrorConflicto:
		return objetivo == ErrOperacionAnalisisEnConflicto ||
			objetivo == domain.ErrVersionEnConflicto ||
			objetivo ==
				ports.ErrConjuntoFuentesAnalisisYaConsumido ||
			objetivo ==
				ports.ErrClaveIdempotenciaOperacionAnalisisUsada
	case tipoErrorResultado:
		return objetivo == ErrResultadoOperacionAnalisisNoConfiable ||
			objetivo == ports.ErrResultadoOperacionAnalisisNoConfiable
	default:
		return objetivo == ErrDependenciaOperacionAnalisisNoDisponible
	}
}

func (e errorOperacionAnalisis) LogValue() slog.Value {
	return slog.StringValue(e.Error())
}
