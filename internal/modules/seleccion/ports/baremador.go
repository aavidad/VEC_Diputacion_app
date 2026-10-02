package ports

import "vec-diputacion-granada/internal/modules/seleccion/domain"

// Baremador consume referencias a instantáneas; Selección no escribe méritos.
type Baremador interface {
	Valorar(convocatoriaRef string, basesVersion int, solicitudRef string) (domain.MeritosValorados, error)
}
