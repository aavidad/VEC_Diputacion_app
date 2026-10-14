package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/dietas/domain"
)

// PreparadorLiquidacionEconomicaHTTP lee nominalmente la comisión y el
// catálogo publicados, con autorización de lectura, antes de reconstruir la
// preparación. Una lista nil pide la propuesta base sin acto de escritura.
type PreparadorLiquidacionEconomicaHTTP interface {
	PrepararLiquidacionEconomica(context.Context, string, uint64, []domain.RevisionLineaLiquidacion) (*domain.PreparacionLiquidacionPublicada, error)
}

// ResolutorRegistroLiquidacionEconomica liga la decisión V3 a la preparación
// opaca y fija la unidad competente desde el perfil central.
type ResolutorRegistroLiquidacionEconomica interface {
	ResolverRegistroLiquidacionEconomica(context.Context, SolicitudLiquidacionEconomica) (IdentidadEfectivaCircuito, error)
}

// CapturadorLecturaLiquidacionEconomica toma la sesión actual y la evaluación
// nominal de permisos antes de que el lector registre la auditoría de lectura.
type CapturadorLecturaLiquidacionEconomica interface {
	CapturarLecturaLiquidacionEconomica(context.Context, string) (CapturaLecturaLiquidacionEconomica, error)
}
