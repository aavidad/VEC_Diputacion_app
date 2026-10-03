package config

import (
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/plannominal"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

// PrepararPlanCompetenciaFirmaV2 exige un resolutor compuesto y un cotejo de
// publicación autorizado. No abre una fuente DEMO ni fabrica una aprobación.
func PrepararPlanCompetenciaFirmaV2(r *reglas.Resolutor, fuente plannominal.PublicacionAutorizada, version ct.VersionPlanFirmaV2) (*plannominal.Fuente, error) {
	return plannominal.NuevaFuente(r, fuente, version)
}
