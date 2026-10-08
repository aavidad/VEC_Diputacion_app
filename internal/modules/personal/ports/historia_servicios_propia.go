package ports

import (
	"context"
	"vec-diputacion-granada/internal/modules/personal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ProveedorAutorizacionHistoriaServiciosPropia interface {
	AutorizarHistoriaServiciosPropia(context.Context, domain.MaterialHistoriaServiciosPropia) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
type OrdenHistoriaServiciosPropia struct {
	Material     domain.MaterialHistoriaServiciosPropia
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}
type ResultadoHistoriaServiciosPropia struct {
	Historia domain.HistoriaServiciosPropia `json:"historia"`
	// Estructura común; decisión, consumo, auditoría y recibo nuevos propios
	// de esta lectura. No aceptar evidencia de ficha, RRHH, CER o exportación.
	Evidencia EvidenciaRegistroEmpleadoB2 `json:"evidencia"`
}

// Contrato pendiente de adaptador durable. Revalida identidad, perfil y único
// empleado canónico actuales; consume V3 propio junto a lectura y auditoría
// común en una transacción. Valida material, respuesta y evidencia antes COMMIT.
// Comprueba que cada servicio y la revisión de su relación pertenecen al
// empleado propio; la referencia de relación en el DTO no prueba pertenencia.
// Todas las revisiones conocidas que solapan el periodo se conservan; no usa
// DISTINCT ON para reducirlas a la última. Más del máximo es error, sin truncar.
// Antes de devolver un error ya cerró/rollback su transacción. No crea outbox,
// documento, certificado ni otro registro de auditoría local. No lee SQL ajeno.
type RepositorioHistoriaServiciosPropia interface {
	ConsultarHistoriaServiciosPropia(context.Context, OrdenHistoriaServiciosPropia) (ResultadoHistoriaServiciosPropia, error)
}

type IntentoHistoriaServiciosPropia struct{ Motivo string }

// Destino #502 y contexto/vínculo/correlación originales capturados antes de
// validar entrada. El adaptador conserva orden/referencia en COMMIT ambiguo;
// no vuelve a leer la historia al reintentar el append. Sin acuse no hay datos.
type RegistroIntentosHistoriaServiciosPropia interface {
	VerificarRegistroHistoriaServiciosPropia(context.Context) error
	RegistrarIntentoHistoriaServiciosPropia(context.Context, IntentoHistoriaServiciosPropia) error
}
