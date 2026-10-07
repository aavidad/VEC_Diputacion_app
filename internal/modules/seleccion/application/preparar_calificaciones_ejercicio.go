package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

var ErrPreparacionCalificacionesNoDisponible = errors.New("seleccion.calificaciones_ejercicio.no_disponible")

func PrepararMaterialCalificacionesEjercicio(ctx context.Context, material domain.MaterialCalificacionesEjercicio) (domain.RegistroCalificacionesPreparado, error) {
	if ctx == nil {
		return domain.RegistroCalificacionesPreparado{}, ErrPreparacionCalificacionesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return domain.RegistroCalificacionesPreparado{}, err
	}
	resultado, err := domain.PrepararCalificacionesEjercicio(material)
	if cancelacion := ctx.Err(); cancelacion != nil {
		return domain.RegistroCalificacionesPreparado{}, cancelacion
	}
	return resultado, err
}
