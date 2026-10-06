package ports

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// ErrPoliticaCreditoOfertaNoDisponible: el catálogo de reglas está declarado
// pero no se puede leer. Nunca se interpreta como «no exige nada».
var ErrPoliticaCreditoOfertaNoDisponible = errors.New(
	"contratacion temporal: politica de credito para ofrecer no disponible",
)

// PoliticaCreditoOferta resuelve, en cada consulta, lo que el catálogo de
// reglas vigente exige al crédito antes de ofrecer el puesto. Así un cambio
// de RRHH en el catálogo se aplica sin reiniciar.
type PoliticaCreditoOferta interface {
	PoliticaCreditoOferta(ctx context.Context) (domain.PoliticaCreditoOferta, error)
}
