package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ServicioConsultaDetalleRRHH struct {
	autoridad              ports.AutoridadContextoConsultaRRHH
	emisor                 *ports.EmisorMaterialConsultaRRHH
	sesion                 ports.SesionConsultaRRHH
	reloj                  ports.Reloj
	filtroUnidad           ports.FuenteUnidadSelectoresRecuperacionRRHH
	historicoSinSelectores bool
}

// El original de propuesta v7 conserva la lectura autorizada de detalle,
// incluso si la decisión del perfil vigente permite el complemento de firma.
// Esta fachada histórica nunca publica selectores.
func NuevoServicioConsultaOriginalPropuestaRRHH(
	autoridad ports.AutoridadContextoConsultaRRHH,
	emisor *ports.EmisorMaterialConsultaRRHH,
	sesion ports.SesionConsultaRRHH,
	reloj ports.Reloj,
) (*ServicioConsultaDetalleRRHH, error) {
	s, err := NuevoServicioConsultaDetalleRRHH(autoridad, emisor, sesion, reloj)
	if err != nil {
		return nil, err
	}
	s.historicoSinSelectores = true
	return s, nil
}

// La fuente opcional sólo reduce los selectores de una decisión que ya los
// concedió. La lectura ordinaria continúa usando el contrato anterior.
func NuevoServicioConsultaDetalleRRHHConSelectores(
	autoridad ports.AutoridadContextoConsultaRRHH,
	emisor *ports.EmisorMaterialConsultaRRHH,
	sesion ports.SesionConsultaRRHH,
	reloj ports.Reloj,
	filtro ports.FuenteUnidadSelectoresRecuperacionRRHH,
) (*ServicioConsultaDetalleRRHH, error) {
	s, err := NuevoServicioConsultaDetalleRRHH(autoridad, emisor, sesion, reloj)
	if err != nil || dependenciaNula(filtro) {
		return nil, ErrServicioConsultaRRHHInvalido
	}
	s.filtroUnidad = filtro
	return s, nil
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
	if s.historicoSinSelectores && solicitud.VersionObservada() != 7 {
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
	if capacidad.AutorizaCamposSeguimiento() {
		return ports.DetalleExpedienteRRHH{}, ErrConsultaRRHHNoObservable
	}
	conSelectores, err := capacidad.ContratoSelectoresRecuperacionFirmas()
	if err != nil {
		return ports.DetalleExpedienteRRHH{}, ErrConsultaRRHHNoObservable
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
	var detalle ports.DetalleExpedienteRRHH
	if conSelectores && !s.historicoSinSelectores {
		lector, disponible := s.sesion.(ports.SesionConsultaSelectoresRecuperacionRRHH)
		if !disponible || dependenciaNula(lector) {
			return ports.DetalleExpedienteRRHH{}, ErrConsultaRRHHNoDisponible
		}
		var filtro ports.FiltroUnidadSelectoresRecuperacionRRHH
		if !dependenciaNula(s.filtroUnidad) {
			filtro, err = s.filtroUnidad.ResolverUnidadSelectoresRecuperacionRRHH(ctx, contexto)
			if err != nil {
				return ports.DetalleExpedienteRRHH{}, normalizarFalloConsultaRRHH(err)
			}
			if !filtro.Vacio() && !filtro.Valido() {
				return ports.DetalleExpedienteRRHH{}, ErrResultadoConsultaRRHHNoConfiable
			}
		}
		detalle, err = lector.ConsultarDetalleConSelectoresYRegistrar(ctx, orden, filtro)
	} else {
		detalle, err = s.sesion.ConsultarDetalleYRegistrar(ctx, orden)
	}
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return ports.DetalleExpedienteRRHH{}, errContexto
	}
	if err != nil {
		return ports.DetalleExpedienteRRHH{},
			normalizarFalloConsultaRRHH(err)
	}
	_, tieneSelectores := detalle.SelectoresRecuperacionFirmas()
	esperarSelectores := conSelectores && !s.historicoSinSelectores
	if detalle.ValidarPara(orden) != nil || tieneSelectores != esperarSelectores ||
		(esperarSelectores && detalle.ValidarReciboSelectoresRecuperacionFirmas() != nil) {
		return ports.DetalleExpedienteRRHH{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	return detalle.Clonar(), nil
}

// ResumenConsultaSeguimientoRRHH limita la salida del modulo a los dos campos
// que necesita la consulta de seguimiento. La organizacion y unidad se
// comprueban dentro de Contratacion temporal y nunca cruzan al consumidor.
type ResumenConsultaSeguimientoRRHH struct {
	ExpedienteRef     string `json:"expediente_ref"`
	VersionExpediente uint64 `json:"version_expediente"`
}

func (s *ServicioConsultaDetalleRRHH) ConsultarResumenSeguimiento(
	ctx context.Context, solicitud ports.SolicitudDetalleRRHH, organizacionRef, unidadRef string,
) (ResumenConsultaSeguimientoRRHH, error) {
	var cero ResumenConsultaSeguimientoRRHH
	if err := errorContextoConsultaRRHH(ctx); err != nil {
		return cero, err
	}
	if s == nil || dependenciaNula(s.autoridad) || dependenciaNula(s.emisor) ||
		dependenciaNula(s.sesion) || dependenciaNula(s.reloj) ||
		solicitud.ExpedienteRef() == "" ||
		!domain.ReferenciaOpacaValida(organizacionRef) || !domain.ReferenciaOpacaValida(unidadRef) {
		return cero, ErrSolicitudConsultaRRHHInvalida
	}
	lector, disponible := s.sesion.(ports.SesionConsultaResumenSeguimientoRRHH)
	if !disponible || dependenciaNula(lector) {
		return cero, ErrConsultaRRHHNoDisponible
	}
	contexto, err := s.autoridad.ResolverContextoConsultaRRHH(ctx)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return cero, errContexto
	}
	if err != nil {
		return cero, normalizarFalloConsultaRRHH(err)
	}
	if contexto.OrganizacionRef() != organizacionRef {
		return cero, ErrConsultaRRHHNoObservable
	}
	material, err := s.emisor.EmitirMaterialDetalleRRHH(ctx, contexto, solicitud)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return cero, errContexto
	}
	if err != nil {
		return cero, normalizarFalloConsultaRRHH(err)
	}
	instanteCapacidad := s.reloj.Ahora()
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return cero, errContexto
	}
	if !domain.InstanteUTCCanonico(instanteCapacidad) {
		return cero, ErrResultadoConsultaRRHHNoConfiable
	}
	capacidad, err := ports.NuevaCapacidadConsultaDetalleRRHH(contexto, material, solicitud, instanteCapacidad)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return cero, errContexto
	}
	if err != nil || !capacidad.AutorizaCamposSeguimiento() {
		return cero, ErrConsultaRRHHNoObservable
	}
	instanteOrden := s.reloj.Ahora()
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return cero, errContexto
	}
	if !domain.InstanteUTCCanonico(instanteOrden) || instanteOrden.Before(instanteCapacidad) ||
		instanteOrden.Before(capacidad.ValidaDesde()) || !instanteOrden.Before(capacidad.ValidaHasta()) {
		return cero, ErrResultadoConsultaRRHHNoConfiable
	}
	orden, err := ports.NuevaOrdenConsultaDetalleRRHH(contexto, capacidad, solicitud, instanteOrden)
	if err != nil {
		return cero, ErrResultadoConsultaRRHHNoConfiable
	}
	resultado, err := lector.ConsultarResumenSeguimientoYRegistrar(ctx, orden, unidadRef)
	if errContexto := errorContextoConsultaRRHH(ctx); errContexto != nil {
		return cero, errContexto
	}
	if err != nil {
		return cero, normalizarFalloConsultaRRHH(err)
	}
	if resultado.ValidarPara(orden, unidadRef) != nil {
		return cero, ErrResultadoConsultaRRHHNoConfiable
	}
	return ResumenConsultaSeguimientoRRHH{
		ExpedienteRef:     resultado.ExpedienteRef,
		VersionExpediente: resultado.VersionExpediente,
	}, nil
}
