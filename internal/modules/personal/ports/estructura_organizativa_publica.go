package ports

import (
	"context"
	"vec-diputacion-granada/internal/modules/personal/domain"
)

type ConsultaEstructuraOrganizativaPublica interface {
	ObtenerEstructuraOrganizativaPublica(context.Context) (domain.EstructuraOrganizativaPublica, error)
}
