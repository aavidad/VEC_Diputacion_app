package application

import "errors"

var (
	ErrServicioAsignacionInvalido = errors.New(
		"contratacion temporal: servicio de asignacion invalido",
	)
	ErrSolicitudAsignacionInvalida = errors.New(
		"contratacion temporal: solicitud de asignacion invalida",
	)
	ErrAsignacionDenegada = errors.New(
		"contratacion temporal: asignacion denegada",
	)
	ErrResultadoAsignacionNoConfiable = errors.New(
		"contratacion temporal: resultado de asignacion no confiable",
	)
)
