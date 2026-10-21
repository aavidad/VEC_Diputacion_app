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
	FuenteIdentidad ports.FuenteIdentidadDocumentoRRHH
	DOCX            vecports.RenderizadorDocumento
	// OmitirReferencias usa la misma presentación de nombres que la descarga PDF.
	OmitirReferencias bool
	// Etiquetas: nombres de catálogo para las referencias; puede ser nil.
	Etiquetas EtiquetadorReferencias
	// Plantillas: el mismo catálogo que el PDF.
	Plantillas *PlantillasBorrador
}

func (r RenderizadorBorradorDOCXDesarrollo) RenderizarBorradorDOCX(
	ctx context.Context, tipo ports.TipoBorradorRRHH, detalle ports.DetalleExpedienteRRHH,
) ([]byte, error) {
	if ctx == nil || r.DOCX == nil || r.DOCX.Formato() != vecdomain.FormatoDocumentoDOCX || r.Plantillas == nil {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	identidad, err := identidadBorrador(ctx, tipo, detalle, r.FuenteIdentidad, r.Plantillas, false)
	if err != nil {
		return nil, err
	}
	documento, err := contenidoBorradorConIdentidad(tipo, detalle, r.Etiquetas, r.Plantillas, r.OmitirReferencias, identidad)
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
