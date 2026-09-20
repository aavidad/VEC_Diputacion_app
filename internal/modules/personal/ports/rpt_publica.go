package ports

import (
	"context"
	"vec-diputacion-granada/internal/modules/personal/domain"
)

// ConsultaRPTPublica es el contrato mínimo de lectura de la proyección RPT.
type ConsultaRPTPublica interface {
	ObtenerRPTPublica(context.Context) (domain.CatalogoRPTPublica, error)
}
