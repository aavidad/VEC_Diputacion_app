package informejuridico

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ProveedorPlantillasBorrador resuelve una instantánea publicada y vigente en
// cada petición. Su implementación durable gobierna selección, autorización
// administrativa y recuperación; este adaptador sólo consume el resultado.
type ProveedorPlantillasBorrador interface {
	ObtenerPlantillas(context.Context, time.Time) (*PlantillasBorrador, error)
}

// BorradorCatalogado liga los bytes a la misma versión de catálogo usada para
// rellenarlos. No acredita firma, aprobación ni custodia documental.
type BorradorCatalogado struct {
	Contenido            []byte
	Formato              vecdomain.FormatoDocumento
	Tipo                 ports.TipoBorradorRRHH
	PlantillaRef         string
	CatalogoRef          string
	CatalogoHuellaSHA256 string
}

// RenderizadorBorradorCatalogoVivo conserva los puertos PDF y DOCX existentes
// y ofrece una salida con referencia y huella para el recibo HTTP. Cada llamada
// obtiene una sola instantánea: un cambio de versión no mezcla texto y recibo.
type RenderizadorBorradorCatalogoVivo struct {
	Proveedor ProveedorPlantillasBorrador
	PDF       vecports.RenderizadorDocumento
	DOCX      vecports.RenderizadorDocumento
	Etiquetas EtiquetadorReferencias
	Ahora     func() time.Time
}

func (r RenderizadorBorradorCatalogoVivo) GenerarPDF(
	ctx context.Context, tipo ports.TipoBorradorRRHH, detalle ports.DetalleExpedienteRRHH,
) (BorradorCatalogado, error) {
	return r.generar(ctx, tipo, detalle, vecdomain.FormatoDocumentoPDF)
}

func (r RenderizadorBorradorCatalogoVivo) GenerarDOCX(
	ctx context.Context, tipo ports.TipoBorradorRRHH, detalle ports.DetalleExpedienteRRHH,
) (BorradorCatalogado, error) {
	return r.generar(ctx, tipo, detalle, vecdomain.FormatoDocumentoDOCX)
}

func (r RenderizadorBorradorCatalogoVivo) RenderizarBorrador(
	ctx context.Context, tipo ports.TipoBorradorRRHH, detalle ports.DetalleExpedienteRRHH,
) ([]byte, error) {
	resultado, err := r.GenerarPDF(ctx, tipo, detalle)
	return resultado.Contenido, err
}

func (r RenderizadorBorradorCatalogoVivo) RenderizarBorradorDOCX(
	ctx context.Context, tipo ports.TipoBorradorRRHH, detalle ports.DetalleExpedienteRRHH,
) ([]byte, error) {
	resultado, err := r.GenerarDOCX(ctx, tipo, detalle)
	return resultado.Contenido, err
}

func (r RenderizadorBorradorCatalogoVivo) generar(
	ctx context.Context, tipo ports.TipoBorradorRRHH, detalle ports.DetalleExpedienteRRHH,
	formato vecdomain.FormatoDocumento,
) (BorradorCatalogado, error) {
	if ctx == nil || r.Proveedor == nil {
		return BorradorCatalogado{}, ports.ErrConsultaRRHHNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return BorradorCatalogado{}, err
	}
	if formato == vecdomain.FormatoDocumentoPDF && (r.PDF == nil || r.PDF.Formato() != formato) ||
		formato == vecdomain.FormatoDocumentoDOCX && (r.DOCX == nil || r.DOCX.Formato() != formato) {
		return BorradorCatalogado{}, ports.ErrConsultaRRHHNoDisponible
	}
	ahora := time.Now
	if r.Ahora != nil {
		ahora = r.Ahora
	}
	plantillas, err := r.Proveedor.ObtenerPlantillas(ctx, ahora().UTC())
	if err != nil || plantillas == nil {
		if ctx.Err() != nil {
			return BorradorCatalogado{}, ctx.Err()
		}
		return BorradorCatalogado{}, ports.ErrConsultaRRHHNoDisponible
	}
	plantilla, existe := plantillas.Plantilla(tipo)
	if !existe {
		return BorradorCatalogado{}, ports.ErrBorradorRRHHNoDisponible
	}
	var contenido []byte
	if formato == vecdomain.FormatoDocumentoPDF {
		contenido, err = (RenderizadorBorradorDesarrollo{PDF: r.PDF, Etiquetas: r.Etiquetas, Plantillas: plantillas}).RenderizarBorrador(ctx, tipo, detalle)
	} else {
		contenido, err = (RenderizadorBorradorDOCXDesarrollo{DOCX: r.DOCX, Etiquetas: r.Etiquetas, Plantillas: plantillas}).RenderizarBorradorDOCX(ctx, tipo, detalle)
	}
	if err != nil {
		return BorradorCatalogado{}, err
	}
	return BorradorCatalogado{
		Contenido: contenido, Formato: formato, Tipo: tipo,
		PlantillaRef: plantilla.Referencia, CatalogoRef: plantillas.Referencia(),
		CatalogoHuellaSHA256: plantillas.Huella(),
	}, nil
}
