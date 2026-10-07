package ports

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrAutoridadAdministracionPerfilesNoDisponible = errors.New("vec: autoridad de administracion de perfiles no disponible")

// Los alias conservan el contrato de datos del puerto. La validacion de
// negocio y de recibos pertenece al dominio.
type RolAdministrable = domain.RolAdministrable
type PropuestaAdministracionPerfiles = domain.PropuestaAdministracionPerfiles
type CierrePropuestaAdministracionPerfiles = domain.CierrePropuestaAdministracionPerfiles
type PreimagenBootstrapAdministracionPerfiles = domain.PreimagenBootstrapAdministracionPerfiles
type ReciboBootstrapAdministracionPerfiles = domain.ReciboBootstrapAdministracionPerfiles

type CatalogoRolesAdministrables interface {
	// ResolverRolAdministrable consulta el catalogo autorizado. Nunca fabrica
	// un rol a partir de una peticion ni publica permisos nuevos.
	ResolverRolAdministrable(context.Context, string) (RolAdministrable, error)
}

// AutoridadActosAdministracionPerfiles es un puerto indivisible. Ningun
// metodo admite una decision de permiso construida por HTTP. Su adaptador
// debe resolver el ContextoActor acreditado, revalidar la instantanea de
// autorizacion mediante el PDP V3 y consumir la autorizacion en la misma
// transaccion SERIALIZABLE que el CAS CA20/AUT24, la auditoria, historia y
// recibo. Debe comprobar que el aprobador sigue siendo administrador activo,
// distinto por persona del proponente y del afectado, y que revocar deja al
// menos una persona administradora efectiva. El proponente puede ser el
// afectado solo al revocar su propio perfil administrador. Si queda una sola
// persona, nuevos actos sensibles quedan cerrados por falta de doble control
// hasta la recuperacion autorizada. Intervencion usa el mismo doble control;
// su rol exacto sigue cerrado hasta publicarlo en catalogo.
// Una revocacion no puede reactivar el mismo perfil/vinculo historico.
// El adaptador rechaza EvidenciaSesionAdministracionPerfiles vacía, cruzada o
// ajena al actor también en replay. Entrega al PDP V3 el vínculo V2 y su
// resultado registrado exactos, sin serializar contexto V2 como contexto V3.
//
// Las referencias de operacion son idempotentes: mismo contenido recupera el
// mismo recibo, contenido distinto falla; incluso al recuperar se revalida
// autorizacion vigente. Las propuestas sensibles no mutan el perfil.
// Clase y operacion del cierre se reconstruyen de la propuesta almacenada;
// ningun campo del cliente puede alterar esa clasificacion.
type AutoridadActosAdministracionPerfiles interface {
	AplicarActoOrdinario(context.Context, domain.SolicitudActoAdministracionPerfiles) (domain.ReciboAdministracionPerfiles, error)
	ProponerActoSensible(context.Context, domain.SolicitudActoAdministracionPerfiles) (PropuestaAdministracionPerfiles, error)
	CerrarPropuestaSensible(context.Context, domain.SolicitudCierrePropuestaAdministracionPerfiles) (CierrePropuestaAdministracionPerfiles, error)
}

// ProvisionadorBootstrapAdministracionPerfiles pertenece exclusivamente al
// canal de operador privado. La implementacion debe verificar aprobacion del
// operador contra la huella exacta, CAS de las dos personas acreditadas y
// contador inicial vacio, todo en una transaccion con auditoria y recibo.
// El origen de la aprobacion es configuracion privada, no un campo HTTP.
// No hay adaptador de HTTP ni implementacion en este corte.
type ProvisionadorBootstrapAdministracionPerfiles interface {
	ProvisionarDosAdministradoresIniciales(context.Context, PreimagenBootstrapAdministracionPerfiles) (ReciboBootstrapAdministracionPerfiles, error)
}
