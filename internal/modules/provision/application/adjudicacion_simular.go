package application

import "vec-diputacion-granada/internal/modules/provision/domain"

// SimularAdjudicacion coordina el ensayo puro; no persiste, admite ni resuelve.
func SimularAdjudicacion(c domain.ConfiguracionAdjudicacion, e domain.EntradaAdjudicacion) (domain.ResultadoAdjudicacion, error) {
	return domain.SimularAdjudicacion(c, e)
}
