package ports

import (
	"context"

	vec "vec-diputacion-granada/internal/vec/domain"
)

// AuditoriaIntentos es la capacidad AppendAudit de la autoridad común AuditStore.
// Registra fallos sin payload, documentos ni material V3. Los éxitos forman parte
// de ConfirmarCambio/RecuperarCambio, nunca de otra escritura fuera del commit.
type AuditoriaIntentos interface {
	AppendAudit(context.Context, vec.AuditEntry) (vec.AuditEntry, error)
}
