package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ProveedorMaterialRecuperacionCorreccionPropia es una autoridad distinta de
// la solicitud y de la recuperación de marcajes remotos.
type ProveedorMaterialRecuperacionCorreccionPropia interface {
	ProveerMaterialRecuperacionCorreccionPropia(context.Context, domain.MaterialRecuperacionCorreccionPropia) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
