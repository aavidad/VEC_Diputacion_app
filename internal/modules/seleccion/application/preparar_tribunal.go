package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

var ErrPreparacionTribunalNoDisponible = errors.New("seleccion.tribunal.no_disponible")

// PrepararMaterialTribunal produce material revisable sin efectos externos.
func PrepararMaterialTribunal(ctx context.Context, material domain.MaterialTribunalPropuesto) (domain.PreparacionTribunal, error) {
	if ctx == nil {
		return domain.PreparacionTribunal{}, ErrPreparacionTribunalNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return domain.PreparacionTribunal{}, err
	}
	resultado, err := domain.PrepararTribunal(material)
	if cancelacion := ctx.Err(); cancelacion != nil {
		return domain.PreparacionTribunal{}, cancelacion
	}
	return resultado, err
}
