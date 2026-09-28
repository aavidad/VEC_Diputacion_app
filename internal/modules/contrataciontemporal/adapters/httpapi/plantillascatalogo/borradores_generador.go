package plantillascatalogo

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/informejuridico"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// GeneradorBorradoresCatalogo representa la instantánea ya autorizada por
// CT133 con los renderizadores comunes. No vuelve a consultar el catálogo:
// bytes y referencias proceden de la misma publicación.
type GeneradorBorradoresCatalogo struct {
	PDF       vecports.RenderizadorDocumento
	DOCX      vecports.RenderizadorDocumento
	Etiquetas informejuridico.EtiquetadorReferencias
}

func (g GeneradorBorradoresCatalogo) Generar(
	ctx context.Context, plantillas *informejuridico.PlantillasBorrador, formato string,
	tipo ports.TipoBorradorRRHH, detalle ports.DetalleExpedienteRRHH,
) (DocumentoCatalogado, error) {
	if ctx == nil || plantillas == nil ||
		g.PDF == nil || g.PDF.Formato() != vecdomain.FormatoDocumentoPDF ||
		g.DOCX == nil || g.DOCX.Formato() != vecdomain.FormatoDocumentoDOCX {
		return DocumentoCatalogado{}, ports.ErrConsultaRRHHNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return DocumentoCatalogado{}, err
	}
	r := informejuridico.RenderizadorBorradorCatalogoVivo{
		Proveedor: PlantillasFijadas{Instantanea: plantillas},
		PDF:       g.PDF, DOCX: g.DOCX, Etiquetas: g.Etiquetas,
	}
	var resultado informejuridico.BorradorCatalogado
	var err error
	switch formato {
	case "pdf":
		resultado, err = r.GenerarPDF(ctx, tipo, detalle)
	case "docx":
		resultado, err = r.GenerarDOCX(ctx, tipo, detalle)
	default:
		return DocumentoCatalogado{}, ports.ErrConsultaRRHHNoDisponible
	}
	if err != nil {
		return DocumentoCatalogado{}, err
	}
	if resultado.Formato != vecdomain.FormatoDocumento(formato) || resultado.Tipo != tipo ||
		resultado.CatalogoRef != plantillas.Referencia() || resultado.CatalogoHuellaSHA256 != plantillas.Huella() {
		return DocumentoCatalogado{}, ports.ErrConsultaRRHHNoDisponible
	}
	return DocumentoCatalogado{
		Contenido: resultado.Contenido, Formato: formato, Tipo: resultado.Tipo,
		PlantillaRef: resultado.PlantillaRef, CatalogoRef: resultado.CatalogoRef,
		CatalogoHuellaSHA256: resultado.CatalogoHuellaSHA256,
	}, nil
}
