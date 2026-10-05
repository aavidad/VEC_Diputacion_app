package ports

import (
	"context"
	"vec-diputacion-granada/internal/vec/domain"
)

// FuentePreservacionAuditoria publica y consulta con su autoridad técnica
// propia. Conserva el efecto y la auditoría común en una misma transacción;
// no entrega datos ni acuses mientras COMMIT no esté confirmado.
type FuentePreservacionAuditoria interface {
	PublicarPreservacionAuditoria(context.Context, domain.SolicitudPreservacionAuditoria) (domain.ResultadoPreservacionAuditoria, error)
	ConsultarPreservacionAuditoria(context.Context, uint64) (domain.ResultadoPreservacionAuditoria, error)
}
