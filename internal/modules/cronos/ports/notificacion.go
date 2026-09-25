package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrComunicacionNoAcreditada = errors.New("cronos comunicacion no acreditada")

// Los proveedores son nominales: la composición debe conectar V3 real.
type ProveedorAutorizacionComunicaciones interface {
	ProveerEnvioNotificacion(context.Context, domain.MaterialAutorizacionNotificacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	ProveerArchivoMensaje(context.Context, MaterialArchivoMensaje) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	ProveerMensajeResolucion(context.Context, MaterialMensajeResolucion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// OrdenComunicaciones solo se forma en el servidor tras resolver identidad y
// empleado propio. El proveedor y repositorio revalidan concesión en el efecto.
type OrdenComunicaciones struct {
	actor     vecdomain.ContextoActor
	proveedor ProveedorAutorizacionComunicaciones
}

func NuevaOrdenComunicaciones(actor vecdomain.ContextoActor, proveedor ProveedorAutorizacionComunicaciones) (OrdenComunicaciones, error) {
	if actor.Validar() != nil || proveedor == nil {
		return OrdenComunicaciones{}, ErrComunicacionNoAcreditada
	}
	copia, err := actor.Clonar()
	if err != nil {
		return OrdenComunicaciones{}, ErrComunicacionNoAcreditada
	}
	return OrdenComunicaciones{actor: copia, proveedor: proveedor}, nil
}
func (o OrdenComunicaciones) Actor() (vecdomain.ContextoActor, error) {
	if o.proveedor == nil || o.actor.Validar() != nil {
		return vecdomain.ContextoActor{}, ErrComunicacionNoAcreditada
	}
	return o.actor.Clonar()
}
func (o OrdenComunicaciones) Proveedor() ProveedorAutorizacionComunicaciones { return o.proveedor }

type SolicitudNotificacion struct {
	TipoRef, FechaReferida, Texto, AdjuntoRef, ClaveOperacion string
	TipoVersion                                               int64
}

// El catálogo es gobernado y versionado; seleccionar un tipo visible no
// concede permiso de envío ni sustituye su revalidación transaccional.
type TipoNotificacionDisponible struct {
	Referencia      string `json:"referencia"`
	Version         int64  `json:"version"`
	NombreClaveI18n string `json:"nombre_clave_i18n"`
}

// Los estados distinguen el registro local del resultado del transporte y de
// una eventual prueba de recepción. Solo el repositorio autorizado puede avanzar.
type EstadoNotificacion string

const (
	NotificacionRegistrada     EstadoNotificacion = "registrada"
	NotificacionPendienteEnvio EstadoNotificacion = "pendiente_envio"
	NotificacionEnviada        EstadoNotificacion = "enviada"
	NotificacionRecibida       EstadoNotificacion = "recibida"
)

type NotificacionGuardada struct {
	Referencia, TipoRef, FechaReferida, Texto, AdjuntoRef string
	TipoVersion                                           int64
	Estado                                                EstadoNotificacion
	Version                                               int64
	RegistradaUTC                                         time.Time
	EnviadaUTC, RecibidaUTC                               time.Time
}

type ReciboNotificacion struct {
	Referencia      string             `json:"referencia"`
	NotificacionRef string             `json:"notificacion_ref"`
	RegistradaUTC   time.Time          `json:"registrada_utc"`
	Estado          EstadoNotificacion `json:"estado"`
	Replay          bool               `json:"replay"`
}

// La implementación debe consumir V3, escribir notificación, historia,
// auditoría, outbox y recibo en una transacción; mismo material/clave recupera
// el recibo y otra huella bajo esa clave devuelve conflicto.
type RepositorioNotificaciones interface {
	RegistrarAutorizada(context.Context, domain.MaterialAutorizacionNotificacion, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboNotificacion, error)
	TiposDisponibles(context.Context, vecdomain.ContextoActor, string) ([]TipoNotificacionDisponible, error)
	// Consultas: concesión positiva exacta, ámbito propio y auditoría también
	// ante denegación. Sin autoridad disponible, error; nunca lista vacía.
	ListarPropias(context.Context, vecdomain.ContextoActor, string, bool) ([]NotificacionGuardada, error)
	ConsultarPropia(context.Context, vecdomain.ContextoActor, string, string) (NotificacionGuardada, error)
}

type CasoUsoNotificaciones interface {
	ListarTiposNotificacion(context.Context, OrdenComunicaciones) ([]TipoNotificacionDisponible, error)
	EnviarNotificacion(context.Context, OrdenComunicaciones, SolicitudNotificacion) (ReciboNotificacion, error)
	ListarNotificaciones(context.Context, OrdenComunicaciones, bool) ([]NotificacionGuardada, error)
	ConsultarNotificacion(context.Context, OrdenComunicaciones, string) (NotificacionGuardada, error)
}
