package application

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var (
	ErrServicioComposicionVisualRRHHInvalido = errors.New(
		"contratacion temporal: servicio de composicion visual RRHH invalido",
	)
	ErrSolicitudComposicionVisualRRHHInvalida = errors.New(
		"contratacion temporal: solicitud de composicion visual RRHH invalida",
	)
	ErrComposicionVisualRRHHNoObservable = errors.New(
		"contratacion temporal: composicion visual RRHH no observable",
	)
	ErrComposicionVisualRRHHNoDisponible = errors.New(
		"contratacion temporal: composicion visual RRHH no disponible",
	)
	ErrResultadoComposicionVisualRRHHNoConfiable = errors.New(
		"contratacion temporal: composicion visual RRHH no confiable",
	)
)

type ServicioConsultaComposicionVisualRRHH struct {
	autoridad     ports.AutoridadContextoConsultaRRHH
	autorizador   ports.AutorizadorComposicionVisualRRHH
	sesion        ports.SesionComposicionVisualRRHH
	publicaciones ports.AutoridadPublicacionesVisualesRRHH
	reloj         ports.Reloj
	vocabulario   ports.VocabularioComposicionVisualRRHH
}

func NuevoServicioConsultaComposicionVisualRRHH(
	autoridad ports.AutoridadContextoConsultaRRHH,
	autorizador ports.AutorizadorComposicionVisualRRHH,
	sesion ports.SesionComposicionVisualRRHH,
	publicaciones ports.AutoridadPublicacionesVisualesRRHH,
	reloj ports.Reloj,
	vocabulario ports.VocabularioComposicionVisualRRHH,
) (*ServicioConsultaComposicionVisualRRHH, error) {
	if dependenciaNula(autoridad) || dependenciaNula(autorizador) ||
		dependenciaNula(sesion) || dependenciaNula(publicaciones) ||
		dependenciaNula(reloj) ||
		!vocabularioComposicionVisualValido(vocabulario) {
		return nil, ErrServicioComposicionVisualRRHHInvalido
	}
	return &ServicioConsultaComposicionVisualRRHH{
		autoridad: autoridad, autorizador: autorizador,
		sesion: sesion, publicaciones: publicaciones,
		reloj: reloj, vocabulario: vocabulario,
	}, nil
}

func (s *ServicioConsultaComposicionVisualRRHH) Consultar(
	ctx context.Context,
	solicitud ports.SolicitudComposicionVisualRRHH,
) (ports.ComposicionVisualRRHH, error) {
	if ctx == nil {
		return ports.ComposicionVisualRRHH{},
			ErrSolicitudComposicionVisualRRHHInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ComposicionVisualRRHH{}, err
	}
	if s == nil || dependenciaNula(s.autoridad) ||
		dependenciaNula(s.autorizador) || dependenciaNula(s.sesion) ||
		dependenciaNula(s.publicaciones) || dependenciaNula(s.reloj) ||
		!vocabularioComposicionVisualValido(s.vocabulario) ||
		!solicitudComposicionVisualValida(solicitud) {
		return ports.ComposicionVisualRRHH{},
			ErrSolicitudComposicionVisualRRHHInvalida
	}
	instante := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if instante.IsZero() {
		return ports.ComposicionVisualRRHH{},
			ErrComposicionVisualRRHHNoDisponible
	}
	contexto, err := s.autoridad.ResolverContextoConsultaRRHH(ctx)
	if errContexto := ctx.Err(); errContexto != nil {
		return ports.ComposicionVisualRRHH{}, errContexto
	}
	if err != nil {
		return ports.ComposicionVisualRRHH{},
			normalizarFalloComposicionVisualRRHH(err)
	}
	capacidad, err := s.autorizador.AutorizarComposicionVisualRRHH(
		ctx, contexto, s.vocabulario, solicitud, instante,
	)
	if errContexto := ctx.Err(); errContexto != nil {
		return ports.ComposicionVisualRRHH{}, errContexto
	}
	if err != nil {
		return ports.ComposicionVisualRRHH{},
			normalizarFalloComposicionVisualRRHH(err)
	}
	orden, err := ports.NuevaOrdenConsultaComposicionVisualRRHH(
		contexto, capacidad, s.vocabulario, solicitud, instante,
	)
	if err != nil {
		return ports.ComposicionVisualRRHH{},
			ErrResultadoComposicionVisualRRHHNoConfiable
	}
	resultado, err := s.sesion.ConsultarComposicionVisualYRegistrar(ctx, orden)
	if errContexto := ctx.Err(); errContexto != nil {
		return ports.ComposicionVisualRRHH{}, errContexto
	}
	if err != nil {
		return ports.ComposicionVisualRRHH{},
			normalizarFalloComposicionVisualRRHH(err)
	}
	copia, err := resultado.Clonar()
	if err != nil || copia.ValidarPara(orden) != nil {
		return ports.ComposicionVisualRRHH{},
			ErrResultadoComposicionVisualRRHHNoConfiable
	}
	atestacion, err := ports.NuevaSolicitudAtestacionPublicacionesVisualesRRHH(
		orden, copia,
	)
	if err != nil {
		return ports.ComposicionVisualRRHH{},
			ErrResultadoComposicionVisualRRHHNoConfiable
	}
	err = s.publicaciones.AtestarPublicacionesVisualesYRegistrar(
		ctx, atestacion,
	)
	if errContexto := ctx.Err(); errContexto != nil {
		return ports.ComposicionVisualRRHH{}, errContexto
	}
	if err != nil {
		return ports.ComposicionVisualRRHH{},
			normalizarFalloComposicionVisualRRHH(err)
	}
	return copia, nil
}

func vocabularioComposicionVisualValido(
	v ports.VocabularioComposicionVisualRRHH,
) bool {
	_, err := ports.NuevoVocabularioComposicionVisualRRHH(
		v.Accion(), v.Finalidad(),
	)
	return err == nil
}

func solicitudComposicionVisualValida(
	s ports.SolicitudComposicionVisualRRHH,
) bool {
	_, err := ports.NuevaSolicitudComposicionVisualRRHH(
		s.FlujoRef(), s.FlujoVersion(),
	)
	return err == nil
}

func normalizarFalloComposicionVisualRRHH(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return context.DeadlineExceeded
	case errors.Is(err, ports.ErrComposicionVisualRRHHNoObservable):
		return ErrComposicionVisualRRHHNoObservable
	case errors.Is(err, ports.ErrPublicacionesVisualesRRHHNoAtestadas):
		return ErrResultadoComposicionVisualRRHHNoConfiable
	case errors.Is(err, ports.ErrContextoConsultaRRHHInvalido),
		errors.Is(err, ports.ErrCapacidadComposicionVisualRRHHInvalida),
		errors.Is(err, ports.ErrOrdenComposicionVisualRRHHInvalida),
		errors.Is(err, ports.ErrResultadoComposicionVisualRRHHNoConfiable):
		return ErrResultadoComposicionVisualRRHHNoConfiable
	default:
		return ErrComposicionVisualRRHHNoDisponible
	}
}
