package simulacion

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
)

// LectorHechos mantiene una copia de un paquete comprobado por Preparar.
// No usa PostgreSQL, transporte ni autoridad de identidad o permisos.
type LectorHechos struct{ preparacion domain.Preparacion }

var _ ports.LectorHechosPreparacion = (*LectorHechos)(nil)

func NuevoLectorHechos(p domain.Paquete) (*LectorHechos, error) {
	preparacion, err := domain.Preparar(p)
	if err != nil {
		return nil, ports.ErrHechosPreparacion
	}
	return &LectorHechos{preparacion: preparacion}, nil
}

func (l *LectorHechos) LeerHechosSinteticos(ctx context.Context, s ports.SelectorHechosPreparacion) (ports.HechosPreparados, error) {
	fallo := ports.HechosPreparados{}
	if l == nil || ctx == nil {
		return fallo, ports.ErrHechosPreparacion
	}
	if err := ctx.Err(); err != nil {
		return fallo, err
	}
	fecha, err := time.Parse("2006-01-02", s.FechaCorte)
	corte, corteErr := time.Parse("2006-01-02", l.preparacion.Paquete.FechaCorte)
	if err != nil || corteErr != nil || !fecha.Equal(corte) || !domain.ReferenciaValida(s.PersonaRef) || len(s.Hechos) == 0 || len(s.Hechos) > 1000 {
		return fallo, ports.ErrHechosPreparacion
	}
	actuales := make(map[string]int)
	for i, h := range l.preparacion.Paquete.Hechos {
		actuales[h.Referencia] = i
	}
	out := ports.HechosPreparados{Alcance: domain.AlcanceSintetico, VersionPaquete: l.preparacion.Paquete.Version, Hechos: make([]ports.HechoPreparado, 0, len(s.Hechos))}
	vistos := make(map[string]bool)
	for _, ref := range s.Hechos {
		i, ok := actuales[ref.Referencia]
		if !ok || vistos[ref.Referencia] || !domain.ReferenciaValida(ref.Referencia) || ref.VersionEsperada < 1 {
			return fallo, ports.ErrHechosPreparacion
		}
		vistos[ref.Referencia] = true
		h := l.preparacion.Paquete.Hechos[i]
		if h.PersonaRef != s.PersonaRef || h.Version != ref.VersionEsperada {
			return fallo, ports.ErrHechosPreparacion
		}
		rev := l.preparacion.Revisiones[i]
		p := ports.HechoPreparado{Referencia: h.Referencia, Version: h.Version, Tipo: h.Tipo, ConceptoRef: h.ConceptoRef,
			Procedencia: h.Procedencia, Vigencia: h.Vigencia, Estado: h.Estado,
			Evidencias: append(h.Evidencias[:0:0], h.Evidencias...), VigenciaEnCorte: rev.VigenciaEnCorte,
			Pendientes: append([]string{}, rev.Pendientes...)}
		if h.Horas != nil {
			horas := *h.Horas
			p.Horas = &horas
		}
		out.Hechos = append(out.Hechos, p)
	}
	return out, nil
}
