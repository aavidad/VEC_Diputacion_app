package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrAsignacionPerfilNoEncontrada     = errors.New("vec: asignacion de perfil no encontrada")
	ErrVersionRolNoEncontrada           = errors.New("vec: version de rol no encontrada")
	ErrDecisionAutorizacionNoEncontrada = errors.New("vec: decision de autorizacion no encontrada")
	ErrVersionAutorizacionYaExiste      = errors.New("vec: version de autorizacion ya existe")
	ErrSecuenciaVersionInvalida         = errors.New("vec: secuencia de version de autorizacion invalida")
	ErrFuenteAutorizacionNoDisponible   = errors.New("vec: fuente de autorizacion no disponible")
	ErrRegistroDecisionNoDisponible     = errors.New("vec: registro de decisiones no disponible")
	ErrRegistroDenegacionNoDisponible   = errors.New("vec: registro de denegaciones no disponible")
	ErrInstantaneaAutorizacionObsoleta  = errors.New("vec: instantanea de autorizacion obsoleta")
)

// Autorizador es el unico puerto que deben consumir los casos de uso. Exigir
// devuelve ErrAutorizacionDenegada para cualquier resultado no concedido.
type Autorizador interface {
	Exigir(context.Context, domain.SolicitudAutorizacion) (domain.DecisionAutorizacion, error)
}

// AutorizadorSolicitudLigadaV2 es deliberadamente distinto de Autorizador.
// Impide que un flujo nuevo acepte por accidente una decision historica V1.
type AutorizadorSolicitudLigadaV2 interface {
	ExigirSolicitudLigadaV2(
		context.Context,
		domain.SolicitudAutorizacionLigadaV2,
	) (domain.DecisionAutorizacion, error)
}

// ValidadorReferenciaMotivoAutorizacionV2 resuelve positivamente una entrada
// contra el catalogo publicado. La validacion estructural de una referencia no
// demuestra su existencia ni evita que un llamador fabrique una huella.
type ValidadorReferenciaMotivoAutorizacionV2 interface {
	ValidarReferenciaMotivoAutorizacionV2(
		context.Context,
		domain.ReferenciaEntradaCatalogo,
		time.Time,
	) error
}

// FuenteAutorizacion aporta una unica instantanea coherente de todos los datos
// que pueden cambiar el resultado. El perfil se resuelve conjuntamente con el
// principal para impedir usar, o siquiera descubrir, el perfil de otra persona.
type FuenteAutorizacion interface {
	ObtenerInstantaneaAutorizacion(context.Context, string, string) (domain.InstantaneaAutorizacion, error)
}

// RegistroDecisionesAutorizacion conserva exclusivamente concesiones
// ejecutables. Debe ser duradero, de solo adicion y carecer de capacidades de
// consulta en produccion.
// RegistrarDecisionSiInstantaneaVigente debe comparar y cambiar de forma
// atomica: antes de insertar revalida que la asignacion actual y el
// control del catalogo de politicas coinciden exactamente con la evidencia de
// la decision. Cualquier diferencia devuelve ErrInstantaneaAutorizacionObsoleta
// y no inserta nada. Una concesion no es valida hasta completar este registro.
type RegistroDecisionesAutorizacion interface {
	RegistrarDecisionSiInstantaneaVigente(context.Context, domain.DecisionAutorizacion) error
}

// RegistroDecisionesAutorizacionSolicitudLigadaV2 nunca acepta V1. El nombre
// distinto obliga a seleccionar de forma visible el esquema durable.
type RegistroDecisionesAutorizacionSolicitudLigadaV2 interface {
	RegistrarDecisionSolicitudLigadaV2SiInstantaneaVigente(
		context.Context,
		OrdenRegistroDecisionAutorizacionSolicitudLigadaV2,
	) error
}

// RegistroDenegacionesAutorizacion conserva el resultado probatorio de una
// evaluacion negativa sin convertirlo en una capacidad consumible. Una
// denegacion sigue siendo efectiva si este registro falla, pero el fallo debe
// propagarse para que operacion y seguridad detecten la perdida de traza.
//
// Este puerto no revalida para conceder ni ofrece lectura al PDP. Su almacen
// productivo debe ser append-only y estar separado del registro de
// concesiones.
type RegistroDenegacionesAutorizacion interface {
	RegistrarDenegacionAutorizacion(context.Context, domain.DecisionAutorizacion) error
}

type RegistroDenegacionesAutorizacionSolicitudLigadaV2 interface {
	RegistrarDenegacionAutorizacionSolicitudLigadaV2(
		context.Context,
		OrdenRegistroDecisionAutorizacionSolicitudLigadaV2,
	) error
}

// GeneradorReferenciaDecisionAutorizacion evita que el caso de uso dependa de
// una biblioteca, formato de UUID o proveedor concreto.
type GeneradorReferenciaDecisionAutorizacion interface {
	NuevaReferenciaDecisionAutorizacion() (string, error)
}
