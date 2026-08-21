package application

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var (
	ErrServicioConfirmacionIncorporacionInvalido = errors.New(
		"contratacion temporal: servicio de confirmacion de incorporacion invalido",
	)
	ErrSolicitudConfirmacionIncorporacionInvalida = errors.New(
		"contratacion temporal: solicitud de confirmacion de incorporacion invalida",
	)
	ErrConfirmacionIncorporacionDenegada = errors.New(
		"contratacion temporal: confirmacion de incorporacion denegada",
	)
	ErrConfirmacionIncorporacionNoDisponible = errors.New(
		"contratacion temporal: confirmacion de incorporacion no disponible",
	)
	ErrResultadoConfirmacionIncorporacionNoConfiable = errors.New(
		"contratacion temporal: resultado de confirmacion de incorporacion no confiable",
	)
)

// SolicitudConfirmarIncorporacion recibe el resultado ya obtenido de Personal.
// No contiene actor, perfil, unidad ni una autoridad declarable por el canal.
type SolicitudConfirmarIncorporacion struct {
	SolicitudPersonal          ports.SolicitudAltaPersonalRPT
	ResultadoPersonal          ports.ResultadoAltaPersonalRPT
	VersionSeguimientoEsperada uint64
	FechaIncorporacion         time.Time
	FechaFinPrevista           time.Time
	MotivoClave                domain.ClaveCatalogo
	Documentos                 []domain.DocumentoSeguimiento
}

func (s SolicitudConfirmarIncorporacion) datos() ports.DatosConfirmacionIncorporacion {
	return ports.DatosConfirmacionIncorporacion{
		SolicitudPersonal:          s.SolicitudPersonal,
		ResultadoPersonal:          s.ResultadoPersonal,
		VersionSeguimientoEsperada: s.VersionSeguimientoEsperada,
		PeriodoIncorporacion: domain.IntervaloSeguimiento{
			Desde: s.FechaIncorporacion,
			Hasta: s.FechaFinPrevista,
		},
		MotivoClave: s.MotivoClave,
		Documentos:  append([]domain.DocumentoSeguimiento(nil), s.Documentos...),
	}
}

type ServicioConfirmacionIncorporacion struct {
	contextos   ports.ResolutorContextoConfirmacionIncorporacion
	reloj       ports.Reloj
	transaccion ports.TransaccionConfirmacionIncorporacion
}

func NuevoServicioConfirmacionIncorporacion(
	contextos ports.ResolutorContextoConfirmacionIncorporacion,
	reloj ports.Reloj,
	transaccion ports.TransaccionConfirmacionIncorporacion,
) (*ServicioConfirmacionIncorporacion, error) {
	if dependenciaNula(contextos) || dependenciaNula(reloj) || dependenciaNula(transaccion) {
		return nil, ErrServicioConfirmacionIncorporacionInvalido
	}
	return &ServicioConfirmacionIncorporacion{
		contextos: contextos, reloj: reloj, transaccion: transaccion,
	}, nil
}

func (s *ServicioConfirmacionIncorporacion) Confirmar(
	ctx context.Context,
	solicitud SolicitudConfirmarIncorporacion,
) (ports.ReciboConfirmacionIncorporacion, error) {
	if s == nil || ctx == nil || dependenciaNula(s.contextos) || dependenciaNula(s.reloj) ||
		dependenciaNula(s.transaccion) {
		return ports.ReciboConfirmacionIncorporacion{},
			ErrServicioConfirmacionIncorporacionInvalido
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboConfirmacionIncorporacion{}, err
	}
	datos := solicitud.datos()
	if datos.Validar() != nil {
		return ports.ReciboConfirmacionIncorporacion{},
			ErrSolicitudConfirmacionIncorporacionInvalida
	}
	contexto, err := s.contextos.ResolverContextoConfirmacionIncorporacion(ctx)
	if err != nil {
		if errContexto := ctx.Err(); errContexto != nil {
			return ports.ReciboConfirmacionIncorporacion{}, errContexto
		}
		return ports.ReciboConfirmacionIncorporacion{},
			ErrConfirmacionIncorporacionDenegada
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboConfirmacionIncorporacion{}, err
	}
	evaluadaEn := instanteCanonico(s.reloj.Ahora())
	if contexto.ValidarPara(datos, evaluadaEn) != nil {
		return ports.ReciboConfirmacionIncorporacion{},
			ErrConfirmacionIncorporacionDenegada
	}
	orden, err := ports.NuevaOrdenConfirmarIncorporacion(contexto, datos, evaluadaEn)
	if err != nil {
		return ports.ReciboConfirmacionIncorporacion{},
			ErrSolicitudConfirmacionIncorporacionInvalida
	}
	recibo, err := s.transaccion.ConfirmarIncorporacion(ctx, orden)
	if err != nil {
		return ports.ReciboConfirmacionIncorporacion{},
			normalizarFalloConfirmacionIncorporacion(ctx, err)
	}
	if recibo.ValidarPara(orden) != nil {
		return ports.ReciboConfirmacionIncorporacion{},
			ErrResultadoConfirmacionIncorporacionNoConfiable
	}
	recibo.Documentos = append(
		[]domain.DocumentoSeguimiento(nil),
		recibo.Documentos...,
	)
	return recibo, nil
}

func normalizarFalloConfirmacionIncorporacion(ctx context.Context, err error) error {
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
	default:
		return ErrConfirmacionIncorporacionNoDisponible
	}
}
