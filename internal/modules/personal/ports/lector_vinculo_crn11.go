package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/personal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ProveedorAutorizacionVinculoCRN11 interface {
	AutorizarVinculoPropioCRN11(context.Context, domain.MaterialVinculoPropioCRN11) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
type OrdenVinculoPropioCRN11 struct {
	Material     domain.MaterialVinculoPropioCRN11
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}
type ResultadoVinculoPropioCRN11 struct {
	Vinculo   domain.VinculoHistoricoCRN11 `json:"vinculo"`
	Evidencia EvidenciaRegistroEmpleadoB2  `json:"evidencia"`
}

// RepositorioVinculoPropioCRN11 sólo devuelve la pareja mínima de Personal.
// En una misma transacción revalida contexto y permiso nominal actual, consume
// V3, comprueba la fuente histórica y registra recibo/auditoría. No hay outbox.
// Ausencia, ambigüedad o autoridad retirada devuelven cero y error nominal.
type RepositorioVinculoPropioCRN11 interface {
	ConsultarVinculoPropioCRN11(context.Context, OrdenVinculoPropioCRN11) (ResultadoVinculoPropioCRN11, error)
}
type LectorVinculoPropioCRN11 interface {
	ConsultarVinculoPropioCRN11(context.Context, domain.SolicitudVinculoPropioCRN11) (ResultadoVinculoPropioCRN11, error)
}
