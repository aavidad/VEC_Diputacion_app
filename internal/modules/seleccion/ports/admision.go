package ports

import (
	meritos "vec-diputacion-granada/internal/modules/meritos/ports"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

// MaterialAdmisionPreparacion solo recibe datos sintéticos locales. Hechos
// reutiliza el contrato de preparación RUM; no es el lector nominal RUM05.
type MaterialAdmisionPreparacion struct {
	PreparacionRef    string                         `json:"preparacion_ref"`
	Revision          int                            `json:"revision"`
	Alcance           string                         `json:"alcance"`
	Bases             domain.BasesAdmision           `json:"bases"`
	SolicitudContexto *domain.ContextoSolicitudLocal `json:"solicitud_contexto,omitempty"`
	Requisitos        []domain.RequisitoAdmision     `json:"requisitos"`
	Hechos            *meritos.HechosPreparados      `json:"hechos,omitempty"`
}
