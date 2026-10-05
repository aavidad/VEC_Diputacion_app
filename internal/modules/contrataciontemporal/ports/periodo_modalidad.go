package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// PreparadorPeriodoModalidad resuelve la publicación c12 confiable para un
// acto nuevo. El cliente no puede escoger referencia, versión ni huella.
type PreparadorPeriodoModalidad interface {
	PrepararPeriodoModalidad(context.Context, domain.ClaveCatalogo, domain.PeriodoPrevisto) (domain.PeriodoPrevisto, error)
}
