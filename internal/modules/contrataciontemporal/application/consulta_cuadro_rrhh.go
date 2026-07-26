package application

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var (
	ErrServicioConsultaRRHHInvalido = errors.New(
		"contratacion temporal: servicio de consulta RRHH invalido",
	)
	ErrSolicitudConsultaRRHHInvalida = errors.New(
		"contratacion temporal: solicitud de consulta RRHH invalida",
	)
	ErrConsultaRRHHNoObservable = errors.New(
		"contratacion temporal: consulta RRHH no observable",
	)
	ErrConsultaRRHHNoDisponible = errors.New(
		"contratacion temporal: consulta RRHH no disponible",
	)
	ErrResultadoConsultaRRHHNoConfiable = errors.New(
		"contratacion temporal: resultado de consulta RRHH no confiable",
	)
)

type ServicioConsultaCuadroRRHH struct {
	autoridad   ports.AutoridadContextoConsultaRRHH
	autorizador ports.AutorizadorConsultaRRHH
	sesion      ports.SesionConsultaRRHH
	reloj       ports.Reloj
}

func NuevoServicioConsultaCuadroRRHH(
	autoridad ports.AutoridadContextoConsultaRRHH,
	autorizador ports.AutorizadorConsultaRRHH,
	sesion ports.SesionConsultaRRHH,
	reloj ports.Reloj,
) (*ServicioConsultaCuadroRRHH, error) {
	if dependenciaNula(autoridad) || dependenciaNula(autorizador) ||
		dependenciaNula(sesion) || dependenciaNula(reloj) {
		return nil, ErrServicioConsultaRRHHInvalido
	}
	return &ServicioConsultaCuadroRRHH{
		autoridad: autoridad, autorizador: autorizador,
		sesion: sesion, reloj: reloj,
	}, nil
}

func (s *ServicioConsultaCuadroRRHH) Consultar(
	ctx context.Context,
	solicitud ports.SolicitudCuadroRRHH,
) (ports.PaginaCuadroRRHH, error) {
	if err := errorContextoConsultaRRHH(ctx); err != nil {
		return ports.PaginaCuadroRRHH{}, err
	}
	if s == nil || dependenciaNula(s.autoridad) ||
		dependenciaNula(s.autorizador) || dependenciaNula(s.sesion) ||
		dependenciaNula(s.reloj) || solicitud.Limite() < 1 ||
		solicitud.Limite() > ports.LimiteMaximoCuadroRRHH {
		return ports.PaginaCuadroRRHH{}, ErrSolicitudConsultaRRHHInvalida
	}
	instante := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if err := errorContextoConsultaRRHH(ctx); err != nil {
		return ports.PaginaCuadroRRHH{}, err
	}
	contexto, err := s.autoridad.ResolverContextoConsultaRRHH(ctx)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.PaginaCuadroRRHH{}, errContexto
	}
	if err != nil {
		return ports.PaginaCuadroRRHH{}, normalizarFalloConsultaRRHH(err)
	}
	capacidad, err := s.autorizador.AutorizarCuadroRRHH(
		ctx, contexto, solicitud, instante,
	)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.PaginaCuadroRRHH{}, errContexto
	}
	if err != nil {
		return ports.PaginaCuadroRRHH{}, normalizarFalloConsultaRRHH(err)
	}
	orden, err := ports.NuevaOrdenConsultaCuadroRRHH(
		contexto, capacidad, solicitud, instante,
	)
	if err != nil {
		return ports.PaginaCuadroRRHH{}, ErrResultadoConsultaRRHHNoConfiable
	}
	pagina, err := s.sesion.ConsultarCuadroYRegistrar(ctx, orden)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.PaginaCuadroRRHH{}, errContexto
	}
	if err != nil {
		return ports.PaginaCuadroRRHH{}, normalizarFalloConsultaRRHH(err)
	}
	if pagina.ValidarPara(orden) != nil {
		return ports.PaginaCuadroRRHH{}, ErrResultadoConsultaRRHHNoConfiable
	}
	return clonarPaginaCuadroRRHH(pagina), nil
}

func errorContextoConsultaRRHH(ctx context.Context) error {
	if ctx == nil {
		return ErrSolicitudConsultaRRHHInvalida
	}
	return ctx.Err()
}

func normalizarFalloConsultaRRHH(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return context.DeadlineExceeded
	case errors.Is(err, ports.ErrConsultaRRHHNoObservable):
		return ErrConsultaRRHHNoObservable
	case errors.Is(err, ports.ErrContextoConsultaRRHHInvalido),
		errors.Is(err, ports.ErrCapacidadConsultaRRHHInvalida),
		errors.Is(err, ports.ErrOrdenConsultaRRHHInvalida),
		errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable):
		return ErrResultadoConsultaRRHHNoConfiable
	default:
		return ErrConsultaRRHHNoDisponible
	}
}

func clonarPaginaCuadroRRHH(
	pagina ports.PaginaCuadroRRHH,
) ports.PaginaCuadroRRHH {
	pagina.Expedientes = append(
		[]ports.ResumenExpedienteRRHH(nil),
		pagina.Expedientes...,
	)
	return pagina
}
