package ports

import (
	"context"
	"errors"
	"time"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrPermisoNoDisponible = errors.New("cronos permiso no disponible")
var ErrPermisoEnConflicto = errors.New("cronos permiso en conflicto")

// Solo la composicion nominal puede construir una orden con contexto resuelto
// y proveedor V3. El repositorio debe consumirla en la misma transaccion del
// efecto, historia, recibo, auditoria y outbox; una comprobacion previa no basta.
type ProveedorMaterialPermiso interface {
	ProveerMaterialPermiso(context.Context, MaterialPermiso) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
type OrdenPermiso struct {
	actor     vecdomain.ContextoActor
	proveedor ProveedorMaterialPermiso
}

func NuevaOrdenPermiso(actor vecdomain.ContextoActor, p ProveedorMaterialPermiso) (OrdenPermiso, error) {
	if actor.Validar() != nil || p == nil {
		return OrdenPermiso{}, ErrPermisoNoDisponible
	}
	copia, err := actor.Clonar()
	if err != nil {
		return OrdenPermiso{}, ErrPermisoNoDisponible
	}
	return OrdenPermiso{actor: copia, proveedor: p}, nil
}
func (o OrdenPermiso) ContextoActor() (vecdomain.ContextoActor, error) {
	if o.proveedor == nil || o.actor.Validar() != nil {
		return vecdomain.ContextoActor{}, ErrPermisoNoDisponible
	}
	return o.actor.Clonar()
}
func (o OrdenPermiso) Proveedor() ProveedorMaterialPermiso { return o.proveedor }

type MaterialPermiso struct {
	ActorRef, PerfilRef, EmpleadoRef, SolicitudRef, CatalogoVersionRef string
	ClaveOperacion, HuellaMaterial, Accion                             string
	VersionEsperada                                                    int64
}

type PeticionPermiso struct {
	PermisoRef, Desde, Hasta, TramoHorarioRef, ClaveOperacion string
}
type DecisionSolicitudPermiso struct {
	SolicitudRef, ClaveOperacion, MotivoRef string
	VersionEsperada                         int64
	Paso                                    domain.PasoPermiso
	Decision                                domain.DecisionPermiso
}
type ReciboSolicitudPermiso struct {
	SolicitudRef, ReciboRef, CatalogoVersionRef string
	Version                                     int64
	Estado                                      domain.EstadoSolicitudPermiso
	FechaUTC                                    time.Time
	Replay                                      bool
}

// Registrar y Decidir deben revalidar catalogo, cuota, version optimista y
// autorizacion nominal antes de confirmar, con clave+huella semanticas.
type RepositorioPermisos interface {
	FuenteCatalogoPermisos
	ConsultarSolicitud(context.Context, OrdenPermiso, string) (domain.SolicitudPermiso, error)
	// RecuperarOperacion exige autorizacion nominal de lectura y devuelve conflicto
	// si la misma clave pertenece a otra huella. Permite replay tras cambio de catalogo.
	RecuperarOperacion(context.Context, OrdenPermiso, string, string) (ReciboSolicitudPermiso, bool, error)
	RegistrarSolicitud(context.Context, domain.SolicitudPermiso, MaterialPermiso, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboSolicitudPermiso, error)
	DecidirSolicitud(context.Context, DecisionSolicitudPermiso, MaterialPermiso, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboSolicitudPermiso, error)
}

type CasoUsoPermisos interface {
	ConsultarResumenAnual(context.Context, OrdenPermiso, string, int) (domain.ResumenAnualPermiso, error)
	SolicitarPermiso(context.Context, OrdenPermiso, PeticionPermiso) (ReciboSolicitudPermiso, error)
	DecidirPermiso(context.Context, OrdenPermiso, DecisionSolicitudPermiso) (ReciboSolicitudPermiso, error)
}
