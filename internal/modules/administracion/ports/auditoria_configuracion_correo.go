package ports

import (
	"context"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// PreparadorAuditoriaConfiguracionCorreo es la frontera de auditoría ya
// acreditada. Recibe el principal verificado y la versión que se pretende
// producir; el adaptador completa la identidad seudonimizada y la correlación
// antes de que se prepare el payload SMTP.
type PreparadorAuditoriaConfiguracionCorreo interface {
	PrepararAuditoriaConfiguracionCorreo(context.Context, vecdomain.Principal, uint64) (vecdomain.AuditEntry, error)
}
