package docx

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

// Renderizador adapta la generacion DOCX al puerto documental del nucleo.
type Renderizador struct{}

func (Renderizador) Formato() domain.FormatoDocumento {
	return domain.FormatoDocumentoDOCX
}

func (Renderizador) Renderizar(ctx context.Context, contenido domain.ContenidoDocumento) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return Renderizar(contenido.Titulo, contenido.Parrafos)
}
