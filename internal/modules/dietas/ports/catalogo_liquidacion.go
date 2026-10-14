package ports

import (
	"context"
	"time"
	"vec-diputacion-granada/internal/modules/dietas/domain"
)

// FuenteTarifaLiquidacionPublicada selecciona por la versión conservada en la
// comisión y su fecha; publicar otra versión no modifica ese expediente.
type FuenteTarifaLiquidacionPublicada interface {
	ConsultarTarifaPublicada(context.Context, string, int, string, time.Time) (domain.CatalogoLiquidacionPublicado, error)
}
