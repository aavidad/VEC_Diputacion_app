package contrastecopias

import (
	"context"
	"vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
)

type Lector interface {
	Capturar(context.Context) (contrastecopias.Snapshot, error)
}

// EjecutorPostgreSQL es un canal de infraestructura cerrado. Su implementación
// inspecciona imagen, aislamiento y propiedad del cluster desechable; no acepta
// herramientas ni argumentos procedentes de un manifiesto recibido.
type EjecutorPostgreSQL interface {
	EjecutarPostgreSQL(context.Context, string, []string, []byte, int) ([]byte, error)
}

// ExclusionObservada inspecciona la ventana exclusiva del ejecutor y devuelve
// el sello de cluster/runtime/lease observados. No es un booleano de configuración.
// El propietario del canal garantiza que no admite escritores durante la captura.
type ExclusionObservada interface {
	ComprobarExclusion(context.Context) (string, error)
}
