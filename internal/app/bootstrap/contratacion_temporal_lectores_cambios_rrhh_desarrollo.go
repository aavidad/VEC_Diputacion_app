package bootstrap

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var _ httpinterno.ConsultorCambiosRRHH = multiplexorDetalleLectoresRRHHDesarrollo{}

// ConsultarCambios (petición RRHH p.4) selecciona el lector igual que el
// detalle: por la capacidad mTLS ya fijada para la ruta de detalle.
func (m multiplexorDetalleLectoresRRHHDesarrollo) ConsultarCambios(ctx context.Context, solicitud ports.SolicitudDetalleRRHH) (ports.ResultadoConsultaCambiosRRHH, error) {
	var cero ports.ResultadoConsultaCambiosRRHH
	if m.multiplexoresLectoresRRHHDesarrollo == nil || ctx == nil || ctx.Err() != nil {
		return cero, ports.ErrAutorizacionDenegada
	}
	c, ok := capacidadLectorRRHHDesarrollo(ctx, httpinterno.RutaConsultaDetalleRRHH)
	if !ok {
		return cero, ports.ErrAutorizacionDenegada
	}
	servicio, soporte := m.detalles[c.principal.ID], m.soportes[c.principal.ID]
	if c.sello != m.sello || servicio == nil || soporte == nil || soporte.sello != m.sello || soporte.principalID != c.principal.ID || soporte.certificadoSHA256 != c.principal.Attributes["certificate_sha256"] {
		return cero, ports.ErrAutorizacionDenegada
	}
	consultor, disponible := servicio.(httpinterno.ConsultorCambiosRRHH)
	if !disponible {
		return cero, ports.ErrConsultaRRHHNoDisponible
	}
	return consultor.ConsultarCambios(ctx, solicitud)
}
