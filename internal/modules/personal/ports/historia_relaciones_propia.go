package ports

import (
	"context"
	"vec-diputacion-granada/internal/modules/personal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ProveedorAutorizacionHistoriaRelacionesPropia interface {
	AutorizarHistoriaRelacionesPropia(context.Context, domain.MaterialHistoriaRelacionesPropia) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
type OrdenHistoriaRelacionesPropia struct {
	Material     domain.MaterialHistoriaRelacionesPropia
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}
type ResultadoHistoriaRelacionesPropia struct {
	Historia domain.HistoriaRelacionesPropia `json:"historia"`
	// Estructura común; decisión, consumo, auditoría y recibo nuevos propios
	// de esta lectura. No aceptar evidencia de ficha, RRHH, CER o exportación.
	// El adaptador HTTP no expone EmpleadoRef ni la evidencia V3 interna.
	Evidencia EvidenciaRegistroEmpleadoB2 `json:"evidencia"`
}

// Contrato pendiente de adaptador durable. Revalida identidad, perfil y único
// empleado canónico actuales; consume V3 propio junto a lectura y auditoría
// común en una transacción. Valida material, respuesta y evidencia antes COMMIT.
// Comprueba con la relación histórica durable que cada revisión pertenece al
// empleado propio; el prefijo y la referencia del DTO no prueban pertenencia.
// Denominaciones de puesto/situación sólo vienen de sus fuentes autorizadas;
// la relación no acredita por sí sola ocupación ni situación.
// Todas las revisiones conocidas que solapan el periodo se conservan; no usa
// DISTINCT ON para reducirlas a la última. Más del máximo es error, sin truncar.
// Antes de devolver un error ya cerró/rollback su transacción. No crea outbox,
// documento, certificado ni otro registro de auditoría local. No lee SQL ajeno.
type RepositorioHistoriaRelacionesPropia interface {
	ConsultarHistoriaRelacionesPropia(context.Context, OrdenHistoriaRelacionesPropia) (ResultadoHistoriaRelacionesPropia, error)
}

type IntentoHistoriaRelacionesPropia struct{ Motivo string }

// Destino #502 y contexto/vínculo/correlación originales capturados antes de
// validar entrada. El adaptador conserva orden/referencia en COMMIT ambiguo;
// no vuelve a leer la historia al reintentar el append. Sin acuse no hay datos.
type RegistroIntentosHistoriaRelacionesPropia interface {
	VerificarRegistroHistoriaRelacionesPropia(context.Context) error
	RegistrarIntentoHistoriaRelacionesPropia(context.Context, IntentoHistoriaRelacionesPropia) error
}
