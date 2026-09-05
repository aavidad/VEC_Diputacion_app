package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// ExpedienteParaSeleccion conserva la versión fiscalizada y la versión actual.
// Una versión histórica sirve para recuperar un recibo; no autoriza otro efecto.
type ExpedienteParaSeleccion struct {
	Fiscalizado   domain.Expediente
	VersionActual uint64
}

// LectorExpedienteSeleccionLlamamiento solo lee el agregado propio. El llamador
// debe exigir primero identidad y ámbito, y autorizar después cada efecto.
type LectorExpedienteSeleccionLlamamiento interface {
	LeerExpedienteParaSeleccion(context.Context, string, string, uint64) (ExpedienteParaSeleccion, error)
}
