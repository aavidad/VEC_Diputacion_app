package ports

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

// AutoridadCierreGobiernoRolPorReferencia conserva el cierre en la autoridad
// central. Debe recuperar la propuesta original dentro de la misma transacción
// que consume V3, comprueba dos ADMIN distintos y publica el rol. Ninguna
// consulta previa ni campo del cliente sustituye esa recuperación.
type AutoridadCierreGobiernoRolPorReferencia interface {
	CerrarGobiernoRolPorReferencia(context.Context,
		domain.SolicitudCierreGobiernoRolPorReferencia) (domain.CierreGobiernoPerfil, error)
}
