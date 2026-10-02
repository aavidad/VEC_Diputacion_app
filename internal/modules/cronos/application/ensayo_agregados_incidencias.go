package application

import (
	"context"
	"time"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

// EnsayarAgregadosIncidencias consumes one synthetic snapshot. It cannot access
// Personal, runtime repositories, identity, authorization, SQL or HTTP.
func EnsayarAgregadosIncidencias(ctx context.Context, lector ports.LectorSnapshotEnsayoAgregadosIncidencias) (ports.ResultadoEnsayoAgregadosIncidencias, error) {
	vacio := ports.ResultadoEnsayoAgregadosIncidencias{}
	if ctx == nil || dependenciaRemotaNula(lector) {
		return vacio, domain.ErrAgregadosIncidenciasInvalidos
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	s, err := lector.LeerSnapshotEnsayoAgregadosIncidencias(ctx)
	if err != nil {
		return vacio, domain.ErrAgregadosIncidenciasInvalidos
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if s.VersionEsquema != 1 || !s.Demo || !domain.ReferenciaDemoAgregadosValida(s.Referencia) || s.Version < 1 || !domain.ReferenciaDemoAgregadosValida(s.OrganizacionRef) || !domain.HuellaEfectosValida(s.SHA256) || len(s.PersonasRef) < 1 || len(s.PersonasRef) > 100 || len(s.Fuentes) < 1 || len(s.Fuentes) > 4 || s.Cobertura == nil || len(s.Cobertura) > 10000 || s.Incidencias == nil || len(s.Incidencias) > 10000 {
		return vacio, domain.ErrAgregadosIncidenciasInvalidos
	}
	dias, err := domain.DiasPeriodoAgregados(s.Desde, s.HastaExclusiva, s.ZonaHoraria, s.InstanteCorteUTC)
	if err != nil || len(dias)*len(s.PersonasRef)*len(s.Fuentes) > 10000 {
		return vacio, domain.ErrAgregadosIncidenciasInvalidos
	}
	shaCatalogo, err := domain.HuellaCatalogoAgregados(s.Catalogo.Referencia, s.Catalogo.Version, s.Catalogo.Codigos, s.Catalogo.Estados)
	if err != nil {
		return vacio, err
	}
	personas, fechas := map[string]bool{}, map[string]bool{}
	for _, p := range s.PersonasRef {
		if !domain.ReferenciaDemoAgregadosValida(p) || personas[p] {
			return vacio, domain.ErrAgregadosIncidenciasInvalidos
		}
		personas[p] = true
	}
	for _, d := range dias {
		fechas[d] = true
	}
	fuentes := map[string]ports.FuenteEnsayoAgregadosIncidencias{}
	for _, f := range s.Fuentes {
		if _, existe := fuentes[f.Referencia]; existe || !domain.ReferenciaDemoAgregadosValida(f.Referencia) || f.Version < 1 || !domain.HuellaEfectosValida(f.SHA256) || (f.Estado != "disponible" && f.Estado != "no_disponible") {
			return vacio, domain.ErrAgregadosIncidenciasInvalidos
		}
		fuentes[f.Referencia] = f
	}
	coberturas := map[[3]string]domain.CoberturaIncidencias{}
	for _, c := range s.Cobertura {
		if err := ctx.Err(); err != nil {
			return vacio, err
		}
		k := [3]string{c.PersonaRef, c.Fecha, c.FuenteRef}
		f, existe := fuentes[c.FuenteRef]
		if _, duplicada := coberturas[k]; duplicada || !personas[c.PersonaRef] || !fechas[c.Fecha] || !existe || c.FuenteVersion != f.Version || (c.Estado != domain.CoberturaIncidenciasCompleta && c.Estado != domain.CoberturaIncidenciasIncompleta && c.Estado != domain.CoberturaIncidenciasDesconocida) {
			return vacio, domain.ErrAgregadosIncidenciasInvalidos
		}
		coberturas[k] = c.Estado
	}
	unidades := make([]domain.UnidadCoberturaIncidencias, 0, len(dias)*len(s.PersonasRef))
	for _, p := range s.PersonasRef {
		for _, d := range dias {
			estado := domain.CoberturaIncidenciasCompleta
			for _, f := range s.Fuentes {
				c, existe := coberturas[[3]string{p, d, f.Referencia}]
				if !existe || f.Estado != "disponible" || c == domain.CoberturaIncidenciasDesconocida {
					estado = domain.CoberturaIncidenciasDesconocida
					break
				}
				if c == domain.CoberturaIncidenciasIncompleta {
					estado = domain.CoberturaIncidenciasIncompleta
				}
			}
			unidades = append(unidades, domain.UnidadCoberturaIncidencias{PersonaRef: p, Fecha: d, Estado: estado})
		}
	}
	hechos := make([]domain.HechoIncidenciaAgregada, 0, len(s.Incidencias))
	loc, err := time.LoadLocation(s.ZonaHoraria)
	if err != nil {
		return vacio, domain.ErrAgregadosIncidenciasInvalidos
	}
	for _, h := range s.Incidencias {
		if err := ctx.Err(); err != nil {
			return vacio, err
		}
		f, existe := fuentes[h.FuenteRef]
		fecha, e := time.ParseInLocation("2006-01-02", h.Fecha, loc)
		if !existe || h.FuenteVersion != f.Version || h.Version < 1 || e != nil || h.RegistradaUTC.IsZero() || h.RegistradaUTC.Location() != time.UTC || h.RegistradaUTC.Nanosecond()%1000 != 0 || h.RegistradaUTC.Before(fecha) || h.RegistradaUTC.After(s.InstanteCorteUTC) {
			return vacio, domain.ErrAgregadosIncidenciasInvalidos
		}
		// Existing explicit facts remain observed even when their source is
		// unavailable at the cutoff; completeness never follows from them.
		hechos = append(hechos, domain.HechoIncidenciaAgregada{Referencia: h.Referencia, PersonaRef: h.PersonaRef, Fecha: h.Fecha, Codigo: h.Codigo, Estado: h.Estado})
	}
	agregado, err := domain.AgregarIncidenciasRegistradas(s.PersonasRef, dias, unidades, hechos)
	if err != nil {
		return vacio, err
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	return ports.ResultadoEnsayoAgregadosIncidencias{Demo: true, SnapshotRef: s.Referencia, SnapshotVersion: s.Version, SnapshotSHA256: s.SHA256, OrganizacionRef: s.OrganizacionRef, Desde: s.Desde, HastaExclusiva: s.HastaExclusiva, ZonaHoraria: s.ZonaHoraria, InstanteCorteUTC: s.InstanteCorteUTC, CatalogoRef: s.Catalogo.Referencia, CatalogoVersion: s.Catalogo.Version, CatalogoSHA256: shaCatalogo, Fuentes: append([]ports.FuenteEnsayoAgregadosIncidencias(nil), s.Fuentes...), Agregado: agregado}, nil
}
