package ports

import (
	"errors"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrRevalidacionAutenticacionActorNoDisponible = errors.New("vec: revalidacion de autenticacion y actor no disponible")

// RevalidadorAutenticacionActorV1 consulta registros autoritativos de sesion,
// cuenta, superficie, garantia y controles de revocacion. No debe copiar esos
// atributos de una peticion ni devolver una sesion almacenada sin revalidarla.
// La comprobacion de sesion y cuentas debe ser atomica en el adaptador.
type RevalidadorAutenticacionActorV1 = domain.RevalidadorAutenticacionActorV1
