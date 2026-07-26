package application

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ServicioConsultaDetalleRRHH struct {
	autoridad   ports.AutoridadContextoConsultaRRHH
	autorizador ports.AutorizadorConsultaRRHH
	sesion      ports.SesionConsultaRRHH
	reloj       ports.Reloj
}

func NuevoServicioConsultaDetalleRRHH(
	autoridad ports.AutoridadContextoConsultaRRHH,
	autorizador ports.AutorizadorConsultaRRHH,
	sesion ports.SesionConsultaRRHH,
	reloj ports.Reloj,
) (*ServicioConsultaDetalleRRHH, error) {
	if dependenciaNula(autoridad) || dependenciaNula(autorizador) ||
		dependenciaNula(sesion) || dependenciaNula(reloj) {
		return nil, ErrServicioConsultaRRHHInvalido
	}
	return &ServicioConsultaDetalleRRHH{
		autoridad: autoridad, autorizador: autorizador,
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
		dependenciaNula(s.autorizador) || dependenciaNula(s.sesion) ||
		dependenciaNula(s.reloj) || solicitud.ExpedienteRef() == "" {
		return ports.DetalleExpedienteRRHH{}, ErrSolicitudConsultaRRHHInvalida
	}
	instante := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if err := errorContextoConsultaRRHH(ctx); err != nil {
		return ports.DetalleExpedienteRRHH{}, err
	}
	contexto, err := s.autoridad.ResolverContextoConsultaRRHH(ctx)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.DetalleExpedienteRRHH{}, errContexto
	}
	if err != nil {
		return ports.DetalleExpedienteRRHH{}, normalizarFalloConsultaRRHH(err)
	}
	capacidad, err := s.autorizador.AutorizarDetalleRRHH(
		ctx, contexto, solicitud, instante,
	)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.DetalleExpedienteRRHH{}, errContexto
	}
	if err != nil {
		return ports.DetalleExpedienteRRHH{}, normalizarFalloConsultaRRHH(err)
	}
	orden, err := ports.NuevaOrdenConsultaDetalleRRHH(
		contexto, capacidad, solicitud, instante,
	)
	if err != nil {
		return ports.DetalleExpedienteRRHH{},
			ErrResultadoConsultaRRHHNoConfiable
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
