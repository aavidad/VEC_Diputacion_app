package pdf

import (
	"context"
	vecpdf "vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type Renderizador struct{}

func (Renderizador) Renderizar(ctx context.Context, c vecdomain.ContenidoDocumento) ([]byte, error) {
	return (vecpdf.Renderizador{}).Renderizar(ctx, c)
}
