package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// FuentePlanCoberturaFirmaNominalRRHH relee el plan publicado y sus vigencias.
// La respuesta de ficha sólo orienta la consulta; no concede acceso a firmas.
type FuentePlanCoberturaFirmaNominalRRHH interface {
	Plan(context.Context) (domain.PlanCompetenciaFirmaV2, error)
}

type EstadoCoberturaFirmaNominalRRHH string

const (
	CoberturaFirmaNominalDisponible   EstadoCoberturaFirmaNominalRRHH = "disponible"
	CoberturaFirmaNominalNoDisponible EstadoCoberturaFirmaNominalRRHH = "no_disponible"
)

type CoberturaFirmaNominalRRHH struct {
	Estado     EstadoCoberturaFirmaNominalRRHH
	UnidadRef  string
	Plan       domain.VersionPlanFirmaV2
	Documentos []string
}
