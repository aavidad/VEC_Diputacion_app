package docx

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

// Renderizador adapta la generacion DOCX al puerto documental del nucleo.
type Renderizador struct {
	// Membrete pone el logotipo institucional en la cabecera. Falso conserva
	// byte a byte la salida anterior.
	Membrete bool
}

func (Renderizador) Formato() domain.FormatoDocumento {
	return domain.FormatoDocumentoDOCX
}

func (r Renderizador) Renderizar(ctx context.Context, contenido domain.ContenidoDocumento) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.Membrete {
		return RenderizarConMembrete(contenido.Titulo, contenido.Parrafos)
	}
	return Renderizar(contenido.Titulo, contenido.Parrafos)
}
