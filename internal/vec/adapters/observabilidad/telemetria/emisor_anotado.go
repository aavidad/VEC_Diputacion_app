package telemetria

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// EmisorAnotado envuelve el emisor de incidencias técnicas: cada incidencia
// declarada durante una petición queda también en su línea de acceso
// (código, componente y etapa del catálogo), y la correlación común enlaza
// ambas líneas. Se compone una vez en la raíz; los adaptadores no cambian.
type EmisorAnotado struct {
	interno ports.EmisorIncidenciasTecnicas
}

// NuevoEmisorAnotado devuelve el envoltorio. Un emisor nil se sustituye por
// el nulo.
func NuevoEmisorAnotado(interno ports.EmisorIncidenciasTecnicas) *EmisorAnotado {
	if interno == nil {
		interno = ports.EmisorIncidenciasTecnicasNulo{}
	}
	return &EmisorAnotado{interno: interno}
}

// Emitir delega sin contexto: no hay petición que anotar.
func (e *EmisorAnotado) Emitir(s domain.SolicitudIncidenciaTecnica) {
	e.interno.Emitir(s)
}

// EmitirConContexto anota la incidencia saneada en la ficha y delega.
func (e *EmisorAnotado) EmitirConContexto(ctx context.Context, s domain.SolicitudIncidenciaTecnica) {
	if f := FichaDe(ctx); f != nil {
		if c, _ := domain.ClasificarIncidenciaTecnica(s); c.Codigo != "" {
			f.anotarIncidencia(string(c.Codigo), string(c.Componente), string(c.Etapa))
		}
	}
	if contextual, ok := e.interno.(ports.EmisorIncidenciasTecnicasConContexto); ok {
		contextual.EmitirConContexto(ctx, s)
		return
	}
	e.interno.Emitir(s)
}

// MetricasEmision conserva la consulta de contadores del emisor envuelto.
func (e *EmisorAnotado) MetricasEmision() ports.MetricasEmisionIncidencias {
	if consulta, ok := e.interno.(ports.ConsultaMetricasEmisionIncidencias); ok {
		return consulta.MetricasEmision()
	}
	return ports.MetricasEmisionIncidencias{}
}

var _ ports.EmisorIncidenciasTecnicasConContexto = (*EmisorAnotado)(nil)
