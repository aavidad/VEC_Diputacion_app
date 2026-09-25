package application

import (
	"context"
	"errors"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/diagnostico"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
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
	autoridad ports.AutoridadContextoConsultaRRHH
	emisor    *ports.EmisorMaterialConsultaRRHH
	sesion    ports.SesionConsultaRRHH
	reloj     ports.Reloj
	// plazos es opcional: sin catálogo de reglas la página sale sin plazos.
	plazos ports.CalculadoraPlazoFaseRRHH
}

func NuevoServicioConsultaCuadroRRHH(
	autoridad ports.AutoridadContextoConsultaRRHH,
	emisor *ports.EmisorMaterialConsultaRRHH,
	sesion ports.SesionConsultaRRHH,
	reloj ports.Reloj,
) (*ServicioConsultaCuadroRRHH, error) {
	if dependenciaNula(autoridad) || dependenciaNula(emisor) ||
		dependenciaNula(sesion) || dependenciaNula(reloj) {
		return nil, ErrServicioConsultaRRHHInvalido
	}
	return &ServicioConsultaCuadroRRHH{
		autoridad: autoridad, emisor: emisor,
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
		dependenciaNula(s.emisor) || dependenciaNula(s.sesion) ||
		dependenciaNula(s.reloj) || solicitud.Limite() < 1 ||
		solicitud.Limite() > ports.LimiteMaximoCuadroRRHH {
		return ports.PaginaCuadroRRHH{}, ErrSolicitudConsultaRRHHInvalida
	}
	contexto, err := s.autoridad.ResolverContextoConsultaRRHH(ctx)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.PaginaCuadroRRHH{}, errContexto
	}
	if err != nil {
		return ports.PaginaCuadroRRHH{}, normalizarFalloConsultaRRHH(err)
	}
	material, err := s.emisor.EmitirMaterialCuadroRRHH(
		ctx, contexto, solicitud,
	)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.PaginaCuadroRRHH{}, errContexto
	}
	if err != nil {
		return ports.PaginaCuadroRRHH{}, normalizarFalloConsultaRRHH(err)
	}
	instanteCapacidad := s.reloj.Ahora()
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.PaginaCuadroRRHH{}, errContexto
	}
	if !domain.InstanteUTCCanonico(instanteCapacidad) {
		return ports.PaginaCuadroRRHH{}, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaReloj, Sentinela: ErrResultadoConsultaRRHHNoConfiable, Causa: nil}
	}
	capacidad, err := ports.NuevaCapacidadConsultaCuadroRRHH(
		contexto, material, solicitud, instanteCapacidad,
	)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.PaginaCuadroRRHH{}, errContexto
	}
	if err != nil {
		return ports.PaginaCuadroRRHH{}, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaCapacidad, Sentinela: ErrResultadoConsultaRRHHNoConfiable, Causa: err}
	}
	instanteOrden := s.reloj.Ahora()
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.PaginaCuadroRRHH{}, errContexto
	}
	if !domain.InstanteUTCCanonico(instanteOrden) ||
		instanteOrden.Before(instanteCapacidad) ||
		instanteOrden.Before(capacidad.ValidaDesde()) ||
		!instanteOrden.Before(capacidad.ValidaHasta()) {
		return ports.PaginaCuadroRRHH{}, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaReloj, Sentinela: ErrResultadoConsultaRRHHNoConfiable, Causa: nil}
	}
	orden, err := ports.NuevaOrdenConsultaCuadroRRHH(
		contexto, capacidad, solicitud, instanteOrden,
	)
	if err != nil {
		return ports.PaginaCuadroRRHH{}, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaOrden, Sentinela: ErrResultadoConsultaRRHHNoConfiable, Causa: err}
	}
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.PaginaCuadroRRHH{}, errContexto
	}
	pagina, err := s.sesion.ConsultarCuadroYRegistrar(ctx, orden)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.PaginaCuadroRRHH{}, errContexto
	}
	if err != nil {
		return ports.PaginaCuadroRRHH{}, normalizarFalloConsultaRRHH(err)
	}
	if err := pagina.ValidarPara(orden); err != nil {
		return ports.PaginaCuadroRRHH{}, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaPagina, Sentinela: ErrResultadoConsultaRRHHNoConfiable, Causa: err}
	}
	salida := clonarPaginaCuadroRRHH(pagina)
	salida.Plazos = s.calcularPlazosFase(ctx, salida)
	return salida, nil
}

func errorContextoConsultaRRHH(ctx context.Context) error {
	if ctx == nil {
		return ErrSolicitudConsultaRRHHInvalida
	}
	return ctx.Err()
}

func normalizarFalloConsultaRRHH(err error) error {
	var sentinela error
	switch {
	case errors.Is(err, context.Canceled):
		sentinela = context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		sentinela = context.DeadlineExceeded
	case errors.Is(err, ports.ErrConsultaRRHHNoObservable):
		sentinela = ErrConsultaRRHHNoObservable
	case errors.Is(err, ports.ErrContextoConsultaRRHHInvalido),
		errors.Is(err, ports.ErrCapacidadConsultaRRHHInvalida),
		errors.Is(err, ports.ErrOrdenConsultaRRHHInvalida),
		errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable):
		sentinela = ErrResultadoConsultaRRHHNoConfiable
	default:
		sentinela = ErrConsultaRRHHNoDisponible
	}
	fallo := &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaAplicacion, Sentinela: sentinela, Causa: err}
	var previo *diagnostico.FalloConsultaRRHH
	if errors.As(err, &previo) && previo != nil {
		// Conserva también el centinela del puerto, usado por consumidores existentes.
		fallo.Etapa, fallo.CodigoSQL = previo.Etapa, previo.CodigoSQL
	}
	return fallo
}

func clonarPaginaCuadroRRHH(
	pagina ports.PaginaCuadroRRHH,
) ports.PaginaCuadroRRHH {
	pagina.Expedientes = append(
		[]ports.ResumenExpedienteRRHH(nil),
		pagina.Expedientes...,
	)
	pagina.FasesDesde = append([]time.Time(nil), pagina.FasesDesde...)
	pagina.Plazos = nil
	return pagina
}
