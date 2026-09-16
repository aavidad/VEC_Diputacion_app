package ports

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

// RegistradorIntentoAutenticacionAdministracionV1 recibe el evento ya
// validado desde la frontera para entregarlo a la autoridad de auditoria T13.
// El puerto no autentica, autoriza ni persiste por su cuenta.
type RegistradorIntentoAutenticacionAdministracionV1 interface {
	RegistrarIntentoAutenticacionAdministracionV1(
		context.Context,
		domain.IntentoAutenticacionAdministracionV1,
	) error
}
