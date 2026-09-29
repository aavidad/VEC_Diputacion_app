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

// LectorExpedienteAvisoLlamamiento recupera la fiscalización que originó una
// selección ya confirmada. La versión se deriva de la ejecución durable ligada
// al llamamiento; nunca procede de HTTP ni de la cabeza actual del expediente.
// El llamador conserva la autorización fresca de cada efecto posterior.
type LectorExpedienteAvisoLlamamiento interface {
	LeerExpedienteParaAvisoConfirmado(context.Context, string, string, string) (ExpedienteParaSeleccion, error)
}

// LectorNumeroVisibleAvisoLlamamiento devuelve el número que se muestra hoy
// del expediente ligado a un llamamiento confirmado: el anual asignado después
// del alta si existe; si no, el original. No altera el snapshot ni los recibos.
type LectorNumeroVisibleAvisoLlamamiento interface {
	LeerNumeroVisibleVigenteAviso(context.Context, string, string, string) (string, error)
}

// LectorExpedienteLlamamiento reúne las dos lecturas internas que necesita la
// composición del llamamiento. Mantiene separado el contrato de selección
// inicial del que deriva el snapshot de un aviso ya confirmado.
type LectorExpedienteLlamamiento interface {
	LectorExpedienteSeleccionLlamamiento
	LectorExpedienteAvisoLlamamiento
}
