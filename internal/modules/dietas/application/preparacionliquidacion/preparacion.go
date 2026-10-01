// Package preparacionliquidacion prepara propuestas locales sin efectos ni autoridad.
package preparacionliquidacion

import "vec-diputacion-granada/internal/modules/dietas/domain"

type Entrada struct {
	Esquema         string                              `json:"esquema"`
	ComisionRef     string                              `json:"comision_ref"`
	ComisionVersion int64                               `json:"comision_version"`
	DocumentoSHA256 string                              `json:"documento_sha256"`
	Documento       domain.DocumentoComision            `json:"documento"`
	Catalogo        domain.CatalogoLiquidacionPropuesta `json:"catalogo"`
	Revisiones      []domain.RevisionLineaLiquidacion   `json:"revisiones"`
}

func Preparar(e Entrada) (*domain.PreparacionLiquidacion, error) {
	if e.Esquema != domain.EsquemaPreparacionLiquidacion {
		return nil, domain.ErrPreparacionLiquidacion
	}
	return domain.PrepararLiquidacion(e.ComisionRef, e.ComisionVersion, e.DocumentoSHA256, e.Documento, e.Catalogo, e.Revisiones)
}
