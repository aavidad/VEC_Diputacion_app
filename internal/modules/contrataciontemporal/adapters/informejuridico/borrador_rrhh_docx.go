package informejuridico

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// RenderizadorBorradorDOCXDesarrollo representa el mismo detalle ya
// autorizado que PDF. No añade tipos documentales, autoridad ni persistencia.
type RenderizadorBorradorDOCXDesarrollo struct {
	DOCX vecports.RenderizadorDocumento
}

func (r RenderizadorBorradorDOCXDesarrollo) RenderizarBorradorDOCX(
	ctx context.Context, tipo ports.TipoBorradorRRHH, detalle ports.DetalleExpedienteRRHH,
) ([]byte, error) {
	if ctx == nil || r.DOCX == nil || r.DOCX.Formato() != vecdomain.FormatoDocumentoDOCX {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	documento, err := contenidoBorradorDesarrollo(tipo, detalle)
	if err != nil {
		return nil, err
	}
	contenido, err := r.DOCX.Renderizar(ctx, documento)
	if err != nil {
		return nil, err
	}
	if err := r.DOCX.ValidarSalida(ctx, contenido); err != nil {
		return nil, err
	}
	return contenido, ctx.Err()
}
