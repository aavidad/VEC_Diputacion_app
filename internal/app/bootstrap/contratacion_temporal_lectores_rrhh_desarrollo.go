package bootstrap

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// multiplexoresLectoresRRHHDesarrollo selecciona únicamente con la capacidad
// mTLS ya fijada por el revalidador. No recibe ni inspecciona la petición HTTP.
type multiplexoresLectoresRRHHDesarrollo struct {
	cuadros             map[string]httpinterno.ConsultorCuadroRRHH
	detalles            map[string]httpinterno.ConsultorDetalleRRHH
	originalesPropuesta map[string]httpinterno.ConsultorDetalleRRHH
	sello               *selloConsultasContratacionTemporalDesarrollo
	soportes            map[string]*soporteAltaContratacionTemporalDesarrollo
}

func nuevoMultiplexoresLectoresRRHHDesarrollo(sello *selloConsultasContratacionTemporalDesarrollo) *multiplexoresLectoresRRHHDesarrollo {
	return &multiplexoresLectoresRRHHDesarrollo{cuadros: make(map[string]httpinterno.ConsultorCuadroRRHH), detalles: make(map[string]httpinterno.ConsultorDetalleRRHH), originalesPropuesta: make(map[string]httpinterno.ConsultorDetalleRRHH), soportes: make(map[string]*soporteAltaContratacionTemporalDesarrollo), sello: sello}
}

func (m *multiplexoresLectoresRRHHDesarrollo) registrar(principal string, soporte *soporteAltaContratacionTemporalDesarrollo, cuadro httpinterno.ConsultorCuadroRRHH, detalle httpinterno.ConsultorDetalleRRHH, originalesPropuesta ...httpinterno.ConsultorDetalleRRHH) bool {
	if m == nil || principal == "" || soporte == nil || soporte.sello != m.sello || cuadro == nil || detalle == nil || len(originalesPropuesta) > 1 || m.cuadros[principal] != nil {
		return false
	}
	var originalPropuesta httpinterno.ConsultorDetalleRRHH
	if len(originalesPropuesta) == 1 {
		originalPropuesta = originalesPropuesta[0]
		if originalPropuesta == nil {
			return false
		}
	}
	m.cuadros[principal], m.detalles[principal], m.originalesPropuesta[principal], m.soportes[principal] = cuadro, detalle, originalPropuesta, soporte
	return true
}

func (m *multiplexoresLectoresRRHHDesarrollo) Consultar(ctx context.Context, solicitud ports.SolicitudCuadroRRHH) (ports.PaginaCuadroRRHH, error) {
	if m == nil || ctx == nil || ctx.Err() != nil {
		return ports.PaginaCuadroRRHH{}, ports.ErrAutorizacionDenegada
	}
	c, ok := capacidadLectorRRHHDesarrollo(ctx, httpinterno.RutaConsultaCuadroRRHH)
	if !ok {
		return ports.PaginaCuadroRRHH{}, ports.ErrAutorizacionDenegada
	}
	servicio, soporte := m.cuadros[c.principal.ID], m.soportes[c.principal.ID]
	if c.sello != m.sello || servicio == nil || soporte == nil || soporte.sello != m.sello || soporte.principalID != c.principal.ID || soporte.certificadoSHA256 != c.principal.Attributes["certificate_sha256"] {
		return ports.PaginaCuadroRRHH{}, ports.ErrAutorizacionDenegada
	}
	return servicio.Consultar(ctx, solicitud)
}

func (m *multiplexoresLectoresRRHHDesarrollo) ConsultarDetalle(ctx context.Context, solicitud ports.SolicitudDetalleRRHH) (ports.DetalleExpedienteRRHH, error) {
	if m == nil || ctx == nil || ctx.Err() != nil {
		return ports.DetalleExpedienteRRHH{}, ports.ErrAutorizacionDenegada
	}
	c, ok := capacidadLectorRRHHDesarrollo(ctx, httpinterno.RutaConsultaDetalleRRHH)
	if !ok {
		return ports.DetalleExpedienteRRHH{}, ports.ErrAutorizacionDenegada
	}
	servicio, soporte := m.detalles[c.principal.ID], m.soportes[c.principal.ID]
	if c.sello != m.sello || servicio == nil || soporte == nil || soporte.sello != m.sello || soporte.principalID != c.principal.ID || soporte.certificadoSHA256 != c.principal.Attributes["certificate_sha256"] {
		return ports.DetalleExpedienteRRHH{}, ports.ErrAutorizacionDenegada
	}
	return servicio.Consultar(ctx, solicitud)
}

func (m *multiplexoresLectoresRRHHDesarrollo) ConsultarOriginalPropuesta(ctx context.Context, solicitud ports.SolicitudDetalleRRHH) (ports.DetalleExpedienteRRHH, error) {
	if m == nil || ctx == nil || ctx.Err() != nil || solicitud.VersionObservada() != 7 {
		return ports.DetalleExpedienteRRHH{}, ports.ErrAutorizacionDenegada
	}
	c, ok := capacidadLectorRRHHDesarrollo(ctx, httpinterno.RutaConsultaDetalleRRHH)
	if !ok {
		return ports.DetalleExpedienteRRHH{}, ports.ErrAutorizacionDenegada
	}
	servicio, soporte := m.originalesPropuesta[c.principal.ID], m.soportes[c.principal.ID]
	if c.sello != m.sello || servicio == nil || soporte == nil || soporte.sello != m.sello || soporte.principalID != c.principal.ID || soporte.certificadoSHA256 != c.principal.Attributes["certificate_sha256"] {
		return ports.DetalleExpedienteRRHH{}, ports.ErrAutorizacionDenegada
	}
	return servicio.Consultar(ctx, solicitud)
}

type multiplexorDetalleLectoresRRHHDesarrollo struct {
	*multiplexoresLectoresRRHHDesarrollo
}

type multiplexorOriginalPropuestaLectoresRRHHDesarrollo struct {
	*multiplexoresLectoresRRHHDesarrollo
}

func (m multiplexorOriginalPropuestaLectoresRRHHDesarrollo) Consultar(ctx context.Context, s ports.SolicitudDetalleRRHH) (ports.DetalleExpedienteRRHH, error) {
	return m.ConsultarOriginalPropuesta(ctx, s)
}

func (m multiplexorDetalleLectoresRRHHDesarrollo) Consultar(ctx context.Context, s ports.SolicitudDetalleRRHH) (ports.DetalleExpedienteRRHH, error) {
	return m.ConsultarDetalle(ctx, s)
}

func capacidadLectorRRHHDesarrollo(ctx context.Context, ruta string) (capacidadConsultaContratacionTemporalDesarrollo, bool) {
	c, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	if !ok || c.sello == nil || c.ruta != ruta || !rutaConsultaRRHHContratacionTemporalDesarrollo(ruta) || c.principal.ID == "" {
		return capacidadConsultaContratacionTemporalDesarrollo{}, false
	}
	return c, true
}
