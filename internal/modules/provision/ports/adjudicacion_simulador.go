package ports

import "vec-diputacion-granada/internal/modules/provision/domain"

type PeticionAdjudicacion struct {
	Configuracion domain.ConfiguracionAdjudicacion `json:"configuracion"`
	Entrada       domain.EntradaAdjudicacion       `json:"entrada"`
}

type SimuladorAdjudicacion interface {
	SimularAdjudicacion(domain.ConfiguracionAdjudicacion, domain.EntradaAdjudicacion) (domain.ResultadoAdjudicacion, error)
}
