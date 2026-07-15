package ports

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrFuenteContextoActorNoDisponible = errors.New("vec: fuente de contexto de actor no disponible")

// FuenteContextoActor devuelve todas las instantaneas que coincidan exactamente
// con cuenta y perfil. No debe usar LIMIT 1, precedencia ni perfil por defecto:
// el servicio de aplicacion es quien exige una coincidencia unica.
//
// La implementacion devuelve copias defensivas y nunca consulta por DNI, nombre,
// correo ni otro dato personal. Cuenta y referencias son identificadores opacos.
type FuenteContextoActor interface {
	BuscarInstantaneasContextoActor(
		context.Context,
		domain.SolicitudContextoActor,
	) ([]domain.InstantaneaContextoActor, error)
}
