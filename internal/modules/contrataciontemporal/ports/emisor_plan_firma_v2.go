package ports

import (
	"context"

	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// EmisorMaterialPlanFirmaV2 emite la autorización exterior sobre el contexto
// preparado con el pin y la decisión interior. No confirma un efecto ni concede
// derechos por los datos del recurso; consume el mismo PDP nominal común.
type EmisorMaterialPlanFirmaV2 interface {
	AutorizarMaterialPlanFirmaV2(context.Context, MaterialFirmaVerificadaV2, vd.RecursoAutorizable) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
