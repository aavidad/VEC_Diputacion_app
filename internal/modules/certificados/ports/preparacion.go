package ports

import (
	"context"
	"vec-diputacion-granada/internal/modules/certificados/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// FuenteServicios must return an immutable snapshot from the data owner.
// The file adapter is a rehearsal consumer, never a replacement for Personal.
type FuenteServicios interface {
	Obtener(context.Context) (domain.FuenteServicios, error)
}
type CatalogoPlantillas interface {
	Obtener(context.Context, string, int, string) (domain.Plantilla, domain.Textos, error)
}
type Renderizador interface {
	Renderizar(context.Context, vecdomain.ContenidoDocumento) ([]byte, error)
}
