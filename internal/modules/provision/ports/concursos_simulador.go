package ports

import "vec-diputacion-granada/internal/modules/provision/domain"

type PeticionSimulacion struct {
	Configuracion domain.Configuracion `json:"configuracion"`
	Entrada       domain.Entrada       `json:"entrada"`
}

// Simulador permite conectar otro canal al mismo caso de uso sin trasladar
// reglas al transporte. Este corte no añade conectores de Personal/RUM.
type Simulador interface {
	Simular(domain.Configuracion, domain.Entrada) (domain.Resultado, error)
}
