package registropropio

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// La autoridad institucional real (CA, revocación y equivalencia entre
// credenciales) sigue pendiente. Esta composición conserva la denegación
// explícita hasta que se inyecte una fuente acreditada.
type FuenteInstitucionalNoDisponible struct{}

func (FuenteInstitucionalNoDisponible) AcreditarRegistroPropio(context.Context, string) (domain.AcreditacionInstitucionalRegistroPropioV1, error) {
	return domain.AcreditacionInstitucionalRegistroPropioV1{}, ports.ErrRegistroPropioNoDisponible
}
func (FuenteInstitucionalNoDisponible) RevalidarRegistroPropio(context.Context, domain.AcreditacionInstitucionalRegistroPropioV1) error {
	return ports.ErrRegistroPropioNoDisponible
}
func (FuenteInstitucionalNoDisponible) ResolverEquivalenciaPersona(context.Context, string, domain.AcreditacionInstitucionalRegistroPropioV1) (ports.ResultadoEquivalenciaPersonaRegistroPropioV1, error) {
	return ports.ResultadoEquivalenciaPersonaRegistroPropioV1{}, ports.ErrRegistroPropioNoDisponible
}
