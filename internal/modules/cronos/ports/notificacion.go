package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Notificaciones de la persona empleada a RRHH (C9, cronos_v1 000010).

// ErrTipoNotificacionNoVigente: el tipo elegido ya no está en el catálogo
// vigente. Nunca se interpreta como registro.
var ErrTipoNotificacionNoVigente = errors.New("cronos tipo de notificacion no vigente")

// ErrBandejaNotificacionesDemasiadoGrande: la bandeja de RRHH supera 500
// notificaciones; se rechaza entera en vez de recortarla en silencio.
var ErrBandejaNotificacionesDemasiadoGrande = errors.New("cronos bandeja de notificaciones demasiado grande")

// Estados que ve la persona. "atendida" sólo indica que RRHH la marcó
// atendida; no es una resolución ni una respuesta.
const (
	EstadoNotificacionRegistrada = "registrada"
	EstadoNotificacionAtendida   = "atendida"
)

// ---- La persona: registro y consulta de las propias ----

type ProveedorMaterialNotificacionesPropias interface {
	ProveerMaterialRegistroNotificacion(context.Context, domain.MaterialRegistroNotificacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	ProveerMaterialConsultaNotificacionesPropias(context.Context, domain.MaterialConsultaNotificaciones) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// OrdenNotificacionesPropias liga el contexto registrado de la petición al
// proveedor V3 nominal de la misma petición. Sin proveedor no hay orden.
type OrdenNotificacionesPropias struct {
	contexto  vecdomain.ContextoActor
	proveedor ProveedorMaterialNotificacionesPropias
}

func NuevaOrdenNotificacionesPropias(contexto vecdomain.ContextoActor, proveedor ProveedorMaterialNotificacionesPropias) (OrdenNotificacionesPropias, error) {
	if dependenciaRemotaNula(proveedor) || contexto.Validar() != nil {
		return OrdenNotificacionesPropias{}, ErrDependenciaNoDisponible
	}
	copia, err := contexto.Clonar()
	if err != nil {
		return OrdenNotificacionesPropias{}, ErrDependenciaNoDisponible
	}
	return OrdenNotificacionesPropias{contexto: copia, proveedor: proveedor}, nil
}

func (o OrdenNotificacionesPropias) ContextoActor() (vecdomain.ContextoActor, error) {
	if dependenciaRemotaNula(o.proveedor) || o.contexto.Validar() != nil {
		return vecdomain.ContextoActor{}, ErrDependenciaNoDisponible
	}
	return o.contexto.Clonar()
}

func (o OrdenNotificacionesPropias) ProveedorMaterial() ProveedorMaterialNotificacionesPropias {
	if dependenciaRemotaNula(o.proveedor) {
		return nil
	}
	return o.proveedor
}

// TipoNotificacion es una versión vigente del catálogo publicado.
type TipoNotificacion struct {
	TipoVersionRef string `json:"tipo_version_ref"`
	TipoRef        string `json:"tipo_ref"`
	Nombre         string `json:"nombre"`
}

// NotificacionPropia: lo que la persona comunicó y su estado. El adjunto es
// sólo la referencia de custodia y la huella del documento.
type NotificacionPropia struct {
	NotificacionRef string     `json:"notificacion_ref"`
	TipoRef         string     `json:"tipo_ref"`
	TipoNombre      string     `json:"tipo_nombre"`
	FechaReferida   string     `json:"fecha_referida"`
	Texto           string     `json:"texto"`
	AdjuntoRef      string     `json:"adjunto_ref,omitempty"`
	AdjuntoSHA256   string     `json:"adjunto_sha256,omitempty"`
	RegistradaEnUTC time.Time  `json:"registrada_en"`
	Estado          string     `json:"estado"`
	AtendidaEnUTC   *time.Time `json:"atendida_en,omitempty"`
}

type ConsultaNotificacionesPropias struct {
	Tipos          []TipoNotificacion   `json:"tipos"`
	Notificaciones []NotificacionPropia `json:"notificaciones"`
}

type PeticionRegistroNotificacion struct {
	ClaveOperacion, TipoVersionRef, FechaReferida, Texto string
	AdjuntoRef, AdjuntoSHA256                            string
}

type ReciboNotificacion struct {
	NotificacionRef string    `json:"notificacion_ref"`
	ReciboRef       string    `json:"recibo_ref"`
	InstanteUTC     time.Time `json:"instante_utc"`
	Replay          bool      `json:"replay"`
}

// La implementación consume la decisión V3 nominal y escribe notificación,
// auditoría, outbox y recibo en una transacción; la misma clave y material
// recupera el recibo y otro material con esa clave devuelve conflicto.
type RepositorioNotificacionesPropias interface {
	ConsultarPropias(context.Context, OrdenNotificacionesPropias, domain.MaterialConsultaNotificaciones) (ConsultaNotificacionesPropias, error)
	RegistrarNotificacion(context.Context, OrdenNotificacionesPropias, domain.MaterialRegistroNotificacion) (ReciboNotificacion, error)
}

type CasoUsoNotificacionesPropias interface {
	ConsultarPropias(context.Context, OrdenNotificacionesPropias) (ConsultaNotificacionesPropias, error)
	RegistrarNotificacion(context.Context, OrdenNotificacionesPropias, PeticionRegistroNotificacion) (ReciboNotificacion, error)
}

// ---- RRHH: bandeja y atención ----

type ProveedorMaterialBandejaNotificaciones interface {
	ProveerMaterialBandejaNotificaciones(context.Context, domain.MaterialConsultaNotificaciones) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	ProveerMaterialAtencionNotificacion(context.Context, domain.MaterialAtencionNotificacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type OrdenBandejaNotificaciones struct {
	contexto  vecdomain.ContextoActor
	proveedor ProveedorMaterialBandejaNotificaciones
}

func NuevaOrdenBandejaNotificaciones(contexto vecdomain.ContextoActor, proveedor ProveedorMaterialBandejaNotificaciones) (OrdenBandejaNotificaciones, error) {
	if dependenciaRemotaNula(proveedor) || contexto.Validar() != nil {
		return OrdenBandejaNotificaciones{}, ErrDependenciaNoDisponible
	}
	copia, err := contexto.Clonar()
	if err != nil {
		return OrdenBandejaNotificaciones{}, ErrDependenciaNoDisponible
	}
	return OrdenBandejaNotificaciones{contexto: copia, proveedor: proveedor}, nil
}

func (o OrdenBandejaNotificaciones) ContextoActor() (vecdomain.ContextoActor, error) {
	if dependenciaRemotaNula(o.proveedor) || o.contexto.Validar() != nil {
		return vecdomain.ContextoActor{}, ErrDependenciaNoDisponible
	}
	return o.contexto.Clonar()
}

func (o OrdenBandejaNotificaciones) ProveedorMaterial() ProveedorMaterialBandejaNotificaciones {
	if dependenciaRemotaNula(o.proveedor) {
		return nil
	}
	return o.proveedor
}

// NotificacionRecibida es una fila de la bandeja de RRHH. EmpleadoEtiqueta
// es el nombre publicado con el circuito; puede faltar.
type NotificacionRecibida struct {
	NotificacionRef  string     `json:"notificacion_ref"`
	EmpleadoRef      string     `json:"empleado_ref"`
	EmpleadoEtiqueta string     `json:"empleado_etiqueta"`
	TipoRef          string     `json:"tipo_ref"`
	TipoNombre       string     `json:"tipo_nombre"`
	FechaReferida    string     `json:"fecha_referida"`
	Texto            string     `json:"texto"`
	AdjuntoRef       string     `json:"adjunto_ref,omitempty"`
	AdjuntoSHA256    string     `json:"adjunto_sha256,omitempty"`
	RegistradaEnUTC  time.Time  `json:"registrada_en"`
	Atendida         bool       `json:"atendida"`
	AtendidaEnUTC    *time.Time `json:"atendida_en,omitempty"`
}

type BandejaNotificaciones struct {
	Notificaciones []NotificacionRecibida `json:"notificaciones"`
}

type PeticionAtencionNotificacion struct {
	ClaveOperacion, NotificacionRef string
}

type ReciboAtencionNotificacion struct {
	AtencionRef     string    `json:"atencion_ref"`
	NotificacionRef string    `json:"notificacion_ref"`
	ReciboRef       string    `json:"recibo_ref"`
	InstanteUTC     time.Time `json:"instante_utc"`
	Replay          bool      `json:"replay"`
}

// Cada llamada consume la decisión V3 nominal, audita y lee o escribe en una
// única frontera durable, filtrada por el circuito publicado (paso de
// administración).
type RepositorioBandejaNotificaciones interface {
	ConsultarBandeja(context.Context, OrdenBandejaNotificaciones, domain.MaterialConsultaNotificaciones) (BandejaNotificaciones, error)
	AtenderNotificacion(context.Context, OrdenBandejaNotificaciones, domain.MaterialAtencionNotificacion) (ReciboAtencionNotificacion, error)
}

type CasoUsoBandejaNotificaciones interface {
	ConsultarBandeja(context.Context, OrdenBandejaNotificaciones) (BandejaNotificaciones, error)
	AtenderNotificacion(context.Context, OrdenBandejaNotificaciones, PeticionAtencionNotificacion) (ReciboAtencionNotificacion, error)
}
