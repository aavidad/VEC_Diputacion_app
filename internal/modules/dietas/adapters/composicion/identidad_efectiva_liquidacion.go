package composicion

import (
	"context"
	"errors"

	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

// ResolverRegistroLiquidacionEconomica usa la competencia nominal de la etapa
// y emite material para la acción económica exacta. La unidad del cliente se
// sustituye antes de construir el efecto autorizado.
func (r *ResolutorIdentidadEfectivaCircuito) ResolverRegistroLiquidacionEconomica(ctx context.Context, solicitud dietasports.SolicitudLiquidacionEconomica) (dietasports.IdentidadEfectivaCircuito, error) {
	var cero dietasports.IdentidadEfectivaCircuito
	base, err := r.base(ctx)
	if err != nil {
		return cero, err
	}
	unidad, err := r.fuente.UnidadCompetente(ctx, base.Contexto, domain.EtapaLiquidacion)
	if errors.Is(err, dietasports.ErrCompetenciaCircuitoSinFuente) {
		return cero, err
	}
	if err != nil || unidad == "" {
		return cero, opacoCircuito(ctx, err)
	}
	solicitud.Decision.UnidadRef = unidad
	efecto, err := dietasapp.ConstruirEfectoAutorizacionLiquidacionEconomica(base.Contexto, unidad, solicitud)
	if err != nil || efecto.Recurso.Referencia != solicitud.Decision.Referencia {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	return r.emitir(ctx, base, unidad, dietasapp.AccionRegistrarLiquidacionEconomica,
		solicitud.Decision.Referencia, dietasapp.FinalidadRegistrarLiquidacionEconomica,
		dietasapp.AudienciaRegistrarLiquidacionEconomica, efecto)
}

var _ dietasports.ResolutorRegistroLiquidacionEconomica = (*ResolutorIdentidadEfectivaCircuito)(nil)
