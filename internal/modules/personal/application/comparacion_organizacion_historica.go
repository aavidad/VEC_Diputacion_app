package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

// Cada lectura conserva su selector exacto (incluido el cursor) y la evidencia
// emitida por la consulta V3 existente. No se fabrica un recibo de comparación.
type LecturaComparacionOrganizacion struct {
	Selector  domain.SelectorOrganizacionHistorica         `json:"selector"`
	Evidencia ports.EvidenciaConsultaOrganizacionHistorica `json:"evidencia"`
}
type ResultadoComparacionOrganizacionHistorica struct {
	Comparacion     domain.ComparacionOrganizacionHistorica `json:"comparacion"`
	LecturasAntes   []LecturaComparacionOrganizacion        `json:"lecturas_antes"`
	LecturasDespues []LecturaComparacionOrganizacion        `json:"lecturas_despues"`
}
type ServicioComparacionOrganizacionHistorica struct {
	consulta *ServicioConsultaOrganizacionHistorica
}

func NuevoServicioComparacionOrganizacionHistorica(c *ServicioConsultaOrganizacionHistorica) (*ServicioComparacionOrganizacionHistorica, error) {
	if c == nil || dependenciaOrganizacionHistoricaNula(c.autorizador) || dependenciaOrganizacionHistoricaNula(c.repositorio) {
		return nil, domain.ErrOrganizacionHistoricaNoDisponible
	}
	return &ServicioComparacionOrganizacionHistorica{consulta: c}, nil
}

func (s *ServicioComparacionOrganizacionHistorica) Comparar(ctx context.Context, antes, despues domain.SolicitudConsultaOrganizacionHistorica) (ResultadoComparacionOrganizacionHistorica, error) {
	var vacio ResultadoComparacionOrganizacionHistorica
	if ctx == nil || s == nil || s.consulta == nil {
		return vacio, domain.ErrOrganizacionHistoricaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if antes.Selector.Cursor != "" || despues.Selector.Cursor != "" || antes.Selector.OrganismoRef != despues.Selector.OrganismoRef || antes.Selector.UnidadClave != despues.Selector.UnidadClave {
		return vacio, domain.ErrComparacionOrganizacionHistoricaInvalida
	}
	// Validar ambos cortes antes de consumir la primera lectura autorizada.
	if _, err := domain.NuevoMaterialConsultaOrganizacionHistorica(antes); err != nil {
		return vacio, err
	}
	if _, err := domain.NuevoMaterialConsultaOrganizacionHistorica(despues); err != nil {
		return vacio, err
	}
	a, la, err := s.reunir(ctx, antes)
	if err != nil {
		return vacio, err
	}
	d, ld, err := s.reunir(ctx, despues)
	if err != nil {
		return vacio, err
	}
	comparacion, err := domain.CompararOrganizacionHistorica(a, d)
	if err != nil {
		return vacio, err
	}
	return ResultadoComparacionOrganizacionHistorica{Comparacion: comparacion, LecturasAntes: la, LecturasDespues: ld}, nil
}
func (s *ServicioComparacionOrganizacionHistorica) reunir(ctx context.Context, solicitud domain.SolicitudConsultaOrganizacionHistorica) (domain.InstantaneaComparacionOrganizacion, []LecturaComparacionOrganizacion, error) {
	var vacio domain.InstantaneaComparacionOrganizacion
	r, err := domain.NuevaReunionPaginasOrganizacion(solicitud.Selector)
	if err != nil {
		return vacio, nil, err
	}
	lecturas := []LecturaComparacionOrganizacion{}
	for n := 0; n < domain.LimitePaginasComparacionOrganizacion; n++ {
		resultado, err := s.consulta.Consultar(ctx, solicitud)
		if err != nil {
			return vacio, nil, err
		}
		p := resultado.Pagina
		i := domain.InstantaneaComparacionOrganizacion{Selector: p.Selector, Cobertura: domain.CoberturaComparacionOrganizacion(p.Cobertura), Unidades: p.Unidades, PuestosTipo: p.PuestosTipo, Dotaciones: p.Dotaciones, Plazas: p.Plazas, PuestosIndividuales: p.PuestosIndividuales, Vinculos: p.Vinculos}
		if err := r.Agregar(domain.PaginaInstantaneaOrganizacion{Instantanea: i, VersionRPTRef: p.VersionRPTRef, VersionPlantillaRef: p.VersionPlantillaRef, CursorSiguiente: p.CursorSiguiente}); err != nil {
			return vacio, nil, domain.ErrOrganizacionHistoricaNoDisponible
		}
		lecturas = append(lecturas, LecturaComparacionOrganizacion{Selector: p.Selector, Evidencia: resultado.Evidencia})
		if p.CursorSiguiente == "" {
			i, err := r.Finalizar()
			return i, lecturas, err
		}
		solicitud.Selector.Cursor = p.CursorSiguiente
	}
	return vacio, nil, domain.ErrOrganizacionHistoricaNoDisponible
}
