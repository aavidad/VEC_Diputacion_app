package application

import (
	"context"
	"strings"
	"time"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func referenciaDemoPresencia(ref string) bool {
	return strings.HasPrefix(ref, "demo:") && domain.ReferenciaEfectosValida(ref)
}

// EnsayarPresencia is isolated from runtime identity, persistence and HTTP. It
// requires an exact nonempty selection, not a wildcard or an authorization.
func EnsayarPresencia(ctx context.Context, lector ports.LectorSnapshotEnsayoPresencia) (ports.ResultadoEnsayoPresencia, error) {
	vacio := ports.ResultadoEnsayoPresencia{}
	if ctx == nil || ctx.Err() != nil || dependenciaRemotaNula(lector) {
		return vacio, domain.ErrPresenciaInvalida
	}
	s, err := lector.LeerSnapshotEnsayoPresencia(ctx)
	if err != nil || ctx.Err() != nil || s.VersionEsquema != 1 || !s.Demostracion || !referenciaDemoPresencia(s.Referencia) || s.Version < 1 || !referenciaDemoPresencia(s.FuenteRef) || s.FuenteVersion < 1 || !referenciaDemoPresencia(s.OrganizacionRef) || !domain.HuellaEfectosValida(s.SHA256) || strings.TrimSpace(s.ZonaHoraria) != s.ZonaHoraria || (s.ZonaHoraria == "" || s.ZonaHoraria == "Local") || len(s.ZonaHoraria) > 128 || len(s.PersonasRef) < 1 || len(s.PersonasRef) > 100 || len(s.Personas) != len(s.PersonasRef) {
		return vacio, domain.ErrPresenciaInvalida
	}
	zona, err := time.LoadLocation(s.ZonaHoraria)
	if err != nil {
		return vacio, domain.ErrPresenciaInvalida
	}
	dia, err := time.ParseInLocation("2006-01-02", s.Fecha, zona)
	if err != nil || dia.Format("2006-01-02") != s.Fecha || s.InstanteCorteUTC.IsZero() || s.InstanteCorteUTC.In(zona).Format("2006-01-02") != s.Fecha {
		return vacio, domain.ErrPresenciaInvalida
	}
	seleccion := map[string]bool{}
	for _, ref := range s.PersonasRef {
		if !referenciaDemoPresencia(ref) || seleccion[ref] {
			return vacio, domain.ErrPresenciaInvalida
		}
		seleccion[ref] = true
	}
	porPersona := map[string]ports.EstadoPersonaEnsayoPresencia{}
	marcas := map[string]bool{}
	r := ports.ResultadoEnsayoPresencia{Demostracion: true, SnapshotRef: s.Referencia, SnapshotVersion: s.Version, SnapshotSHA256: s.SHA256, FuenteRef: s.FuenteRef, FuenteVersion: s.FuenteVersion, OrganizacionRef: s.OrganizacionRef, Fecha: s.Fecha, ZonaHoraria: s.ZonaHoraria, InstanteCorteUTC: s.InstanteCorteUTC, Personas: []ports.EstadoPersonaEnsayoPresencia{}}
	for _, p := range s.Personas {
		if ctx.Err() != nil || !seleccion[p.PersonaRef] || p.CompletaHastaCorte == nil || p.Marcajes == nil || len(p.Marcajes) > 100 {
			return vacio, domain.ErrPresenciaInvalida
		}
		if _, existe := porPersona[p.PersonaRef]; existe {
			return vacio, domain.ErrPresenciaInvalida
		}
		hechos := make([]domain.HechoSaldo, len(p.Marcajes))
		for i, m := range p.Marcajes {
			if !referenciaDemoPresencia(m.Referencia) || marcas[m.Referencia] || m.Version < 1 || m.FuenteRef != s.FuenteRef {
				return vacio, domain.ErrPresenciaInvalida
			}
			marcas[m.Referencia] = true
			hechos[i] = domain.HechoSaldo{Movimiento: m.Movimiento, InstanteUTC: m.InstanteUTC}
		}
		resultado, err := domain.EvaluarPresenciaAlCorte(hechos, dia.UTC(), s.InstanteCorteUTC, *p.CompletaHastaCorte)
		if err != nil {
			return vacio, err
		}
		porPersona[p.PersonaRef] = ports.EstadoPersonaEnsayoPresencia{PersonaRef: p.PersonaRef, Estado: resultado.Estado, Motivo: resultado.Causa, CoberturaCompleta: *p.CompletaHastaCorte}
		switch resultado.Estado {
		case domain.PresenciaRegistrada:
			r.Agregado.EntradasRegistradas++
		case domain.PausaRegistrada:
			r.Agregado.PausasRegistradas++
		case domain.SalidaRegistrada:
			r.Agregado.SalidasRegistradas++
		case domain.PresenciaIndeterminada:
			r.Agregado.Indeterminado++
		}
	}
	for _, ref := range s.PersonasRef {
		r.Personas = append(r.Personas, porPersona[ref])
	}
	r.Agregado.Total = len(r.Personas)
	return r, nil
}
