package postgres

import (
	"context"
	"time"
	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

var _ dietasports.FuenteTarifaLiquidacionPublicada = (*RepositorioBorradorComisionPostgreSQL)(nil)

// Se reutiliza la misma selección de publicación que en el alta y la edición.
func (r *RepositorioBorradorComisionPostgreSQL) ConsultarTarifaPublicada(ctx context.Context, version string, grupo int, vehiculo string, fecha time.Time) (domain.CatalogoLiquidacionPublicado, error) {
	if err := r.valido(ctx); err != nil {
		return domain.CatalogoLiquidacionPublicado{}, err
	}
	fuente := &RepositorioTarifasPublicadas{pool: r.pool}
	return fuente.Consultar(ctx, version, grupo, vehiculo, fecha)
}
