package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ServicioSelectorIdentidadDocumentalRRHH struct {
	autoridad ports.AutoridadContextoConsultaRRHH
	emisor    *ports.EmisorMaterialConsultaRRHH
	sesion    ports.SesionSelectorIdentidadDocumentalRRHH
	reloj     ports.Reloj
}

func NuevoServicioSelectorIdentidadDocumentalRRHH(a ports.AutoridadContextoConsultaRRHH,
	e *ports.EmisorMaterialConsultaRRHH, s ports.SesionSelectorIdentidadDocumentalRRHH,
	r ports.Reloj) (*ServicioSelectorIdentidadDocumentalRRHH, error) {
	if dependenciaNula(a) || e == nil || dependenciaNula(s) || dependenciaNula(r) {
		return nil, ErrServicioConsultaRRHHInvalido
	}
	return &ServicioSelectorIdentidadDocumentalRRHH{a, e, s, r}, nil
}

// Reutiliza la decisión nominal de detalle. El captor existe solo durante
// esta llamada y nunca entrega el selector a la respuesta HTTP del detalle.
func (s *ServicioSelectorIdentidadDocumentalRRHH) Consultar(ctx context.Context, expediente string) (domain.SelectorBolsaPlanB2Nativo, error) {
	z := domain.SelectorBolsaPlanB2Nativo{}
	if s == nil || ctx == nil || dependenciaNula(s.sesion) {
		return z, ports.ErrConsultaRRHHNoDisponible
	}
	q, err := ports.NuevaSolicitudDetalleRRHH(expediente, 0)
	if err != nil {
		return z, ErrSolicitudConsultaRRHHInvalida
	}
	captor := &captorSelectorIdentidadDocumentalRRHH{sesion: s.sesion}
	servicio, err := NuevoServicioConsultaDetalleRRHH(s.autoridad, s.emisor, captor, s.reloj)
	if err != nil {
		return z, err
	}
	d, err := servicio.Consultar(ctx, q)
	if ctx.Err() != nil {
		return z, ctx.Err()
	}
	if err != nil {
		return z, err
	}
	selector := captor.selector
	if !selector.Valido() || selector.ExpedienteRef != expediente ||
		selector.OrganizacionRef != d.Resumen.OrganizacionRef ||
		d.Analisis == nil || selector.CategoriaRef != d.Analisis.CategoriaRef ||
		d.Asignacion == nil || selector.UnidadRef != d.Asignacion.UnidadRef ||
		d.Resumen.Version < 7 || uint64(len(d.Hitos)) != d.Resumen.Version ||
		d.Hitos[6].AccionClave != "registrar_propuesta_formalizacion" {
		return z, ports.ErrResultadoConsultaRRHHNoConfiable
	}
	return selector, nil
}

type captorSelectorIdentidadDocumentalRRHH struct {
	sesion   ports.SesionSelectorIdentidadDocumentalRRHH
	selector domain.SelectorBolsaPlanB2Nativo
}

func (c *captorSelectorIdentidadDocumentalRRHH) ConsultarCuadroYRegistrar(context.Context, ports.OrdenConsultaCuadroRRHH) (ports.PaginaCuadroRRHH, error) {
	return ports.PaginaCuadroRRHH{}, ports.ErrConsultaRRHHNoDisponible
}
func (c *captorSelectorIdentidadDocumentalRRHH) ConsultarDetalleYRegistrar(ctx context.Context, o ports.OrdenConsultaDetalleRRHH) (ports.DetalleExpedienteRRHH, error) {
	d, selector, err := c.sesion.ConsultarSelectorYRegistrarAcceso(ctx, o)
	if err != nil {
		return ports.DetalleExpedienteRRHH{}, err
	}
	if !selector.Valido() || selector.ExpedienteRef != o.Solicitud().ExpedienteRef() {
		return ports.DetalleExpedienteRRHH{}, ports.ErrResultadoConsultaRRHHNoConfiable
	}
	c.selector = selector
	return d, nil
}
