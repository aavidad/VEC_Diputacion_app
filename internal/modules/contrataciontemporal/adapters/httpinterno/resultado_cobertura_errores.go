package httpinterno

import (
	"context"
	"errors"
	"net/http"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

var (
	errorResultadoConsultaCoberturaDenegado = nuevoErrorCobertura(
		http.StatusForbidden,
		"acceso_denegado",
	)
	errorResultadoConsultaCoberturaConflicto = nuevoErrorCobertura(
		http.StatusConflict,
		"conflicto",
	)
	errorResultadoConsultaCoberturaNoDisponible = nuevoErrorCobertura(
		http.StatusServiceUnavailable,
		"servicio_no_disponible",
	)
)

func clasificarErrorResultadoCobertura(err error) errorPublicoCobertura {
	switch {
	case errors.Is(err, application.ErrConsultaResultadoCoberturaDenegada):
		return errorResultadoConsultaCoberturaDenegado
	case errors.Is(err, application.ErrConsultaResultadoCoberturaConflicto):
		return errorResultadoConsultaCoberturaConflicto
	case errors.Is(err, application.ErrConsultaResultadoCoberturaNoConfiable),
		errors.Is(
			err,
			application.ErrConsultaResultadoCoberturaNoDisponible,
		),
		errors.Is(
			err,
			application.ErrServicioConsultaResultadoCoberturaInvalido,
		),
		errors.Is(err, context.Canceled),
		errors.Is(err, context.DeadlineExceeded):
		return errorResultadoConsultaCoberturaNoDisponible
	default:
		return errorResultadoConsultaCoberturaNoDisponible
	}
}
