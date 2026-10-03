package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/personal/domain"
)

// LectorCargoCompetencial entrega evidencia nominal dentro de la misma
// transaccion que registra o recupera la firma. El consumidor proporciona los
// bytes exactos del canon CT; Personal coteja solo sus campos propietarios.
type LectorCargoCompetencial interface {
	LeerRevalidarCargoOcupanteCT(context.Context, []byte) (EvidenciaCargoCompetencial, error)
}

type EvidenciaCargoCompetencial struct {
	Cargo             domain.ReferenciaCargoCompetencial
	EnlaceOcupante    domain.EnlaceCargoCompetencial
	OcupantePersona   string
	PersonaEjerciente string
	TipoEjercicio     domain.ClaseEjercicio
	OrganizacionRef   string
	UnidadRef         string
	PuestoRef         string
	ProcedenciaRef    string
	ReciboRef         string
}
