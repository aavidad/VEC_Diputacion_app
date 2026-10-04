package ports

import vecports "vec-diputacion-granada/internal/vec/ports"

// FalloLecturaAuditado conserva la causa nominal mediante Unwrap y expone
// exclusivamente el acuse confirmado por la autoridad común de auditoría.
type FalloLecturaAuditado interface {
	error
	Unwrap() error
	AcuseLecturaAuditada() vecports.AcuseIntentoAuditoria
}
