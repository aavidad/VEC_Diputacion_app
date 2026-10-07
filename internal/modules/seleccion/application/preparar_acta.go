package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

var ErrPreparacionActaNoDisponible = errors.New("seleccion.acta_preparacion.no_disponible")

// PrepararMaterialActa produce un borrador revisable sin efectos externos.
func PrepararMaterialActa(ctx context.Context, material domain.MaterialActaPropuesto) (domain.PreparacionActa, error) {
	if ctx == nil {
		return domain.PreparacionActa{}, ErrPreparacionActaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return domain.PreparacionActa{}, err
	}
	resultado, err := domain.PrepararActa(material)
	if cancelacion := ctx.Err(); cancelacion != nil {
		return domain.PreparacionActa{}, cancelacion
	}
	return resultado, err
}
