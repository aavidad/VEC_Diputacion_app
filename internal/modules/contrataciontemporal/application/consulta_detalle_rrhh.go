package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ServicioConsultaDetalleRRHH struct {
	autoridad ports.AutoridadContextoConsultaRRHH
	emisor    *ports.EmisorMaterialConsultaRRHH
	sesion    ports.SesionConsultaRRHH
	reloj     ports.Reloj
}

func NuevoServicioConsultaDetalleRRHH(
	autoridad ports.AutoridadContextoConsultaRRHH,
	emisor *ports.EmisorMaterialConsultaRRHH,
	sesion ports.SesionConsultaRRHH,
	reloj ports.Reloj,
) (*ServicioConsultaDetalleRRHH, error) {
	if dependenciaNula(autoridad) || dependenciaNula(emisor) ||
		dependenciaNula(sesion) || dependenciaNula(reloj) {
		return nil, ErrServicioConsultaRRHHInvalido
	}
	return &ServicioConsultaDetalleRRHH{
		autoridad: autoridad, emisor: emisor,
		sesion: sesion, reloj: reloj,
	}, nil
}

func (s *ServicioConsultaDetalleRRHH) Consultar(
	ctx context.Context,
	solicitud ports.SolicitudDetalleRRHH,
) (ports.DetalleExpedienteRRHH, error) {
	if err := errorContextoConsultaRRHH(ctx); err != nil {
		return ports.DetalleExpedienteRRHH{}, err
	}
	if s == nil || dependenciaNula(s.autoridad) ||
		dependenciaNula(s.emisor) || dependenciaNula(s.sesion) ||
		dependenciaNula(s.reloj) || solicitud.ExpedienteRef() == "" {
		return ports.DetalleExpedienteRRHH{}, ErrSolicitudConsultaRRHHInvalida
	}
	contexto, err := s.autoridad.ResolverContextoConsultaRRHH(ctx)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.DetalleExpedienteRRHH{}, errContexto
	}
	if err != nil {
		return ports.DetalleExpedienteRRHH{}, normalizarFalloConsultaRRHH(err)
	}
	material, err := s.emisor.EmitirMaterialDetalleRRHH(
		ctx, contexto, solicitud,
	)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.DetalleExpedienteRRHH{}, errContexto
	}
	if err != nil {
		return ports.DetalleExpedienteRRHH{},
			normalizarFalloConsultaRRHH(err)
	}
	instanteCapacidad := s.reloj.Ahora()
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.DetalleExpedienteRRHH{}, errContexto
	}
	if !domain.InstanteUTCCanonico(instanteCapacidad) {
		return ports.DetalleExpedienteRRHH{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	capacidad, err := ports.NuevaCapacidadConsultaDetalleRRHH(
		contexto, material, solicitud, instanteCapacidad,
	)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.DetalleExpedienteRRHH{}, errContexto
	}
	if err != nil {
		return ports.DetalleExpedienteRRHH{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	instanteOrden := s.reloj.Ahora()
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.DetalleExpedienteRRHH{}, errContexto
	}
	if !domain.InstanteUTCCanonico(instanteOrden) ||
		instanteOrden.Before(instanteCapacidad) ||
		instanteOrden.Before(capacidad.ValidaDesde()) ||
		!instanteOrden.Before(capacidad.ValidaHasta()) {
		return ports.DetalleExpedienteRRHH{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	orden, err := ports.NuevaOrdenConsultaDetalleRRHH(
		contexto, capacidad, solicitud, instanteOrden,
	)
	if err != nil {
		return ports.DetalleExpedienteRRHH{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.DetalleExpedienteRRHH{}, errContexto
	}
	detalle, err := s.sesion.ConsultarDetalleYRegistrar(ctx, orden)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.DetalleExpedienteRRHH{}, errContexto
	}
	if err != nil {
		return ports.DetalleExpedienteRRHH{},
			normalizarFalloConsultaRRHH(err)
	}
	if detalle.ValidarPara(orden) != nil {
		return ports.DetalleExpedienteRRHH{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	return detalle.Clonar(), nil
}
