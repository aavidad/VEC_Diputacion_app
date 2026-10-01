package contrastecopias

import (
	"context"
	"vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
)

type Lector interface {
	Capturar(context.Context) (contrastecopias.Snapshot, error)
}
