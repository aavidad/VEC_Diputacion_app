package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Rechazos nominales de la resolución de permisos (cronos_v1 000009).
// Ninguno se interpreta como autorización ni éxito.
var (
	// ErrResolucionNoCompetente: sin asignación vigente en el circuito para
	// esa persona y paso, solicitud propia, inexistente o separación de
	// funciones. No distingue los casos para no revelar solicitudes ajenas.
	ErrResolucionNoCompetente = errors.New("cronos resolucion no competente")
	// ErrResolucionEstadoCambiado: la versión o el estado ya no son los que
	// vio quien resuelve (otra resolución o un aviso ya archivado).
	ErrResolucionEstadoCambiado = errors.New("cronos resolucion con estado cambiado")
	// ErrResolucionPendienteAsignacion: la persona no tiene jefatura
	// asignada ni marca de circuito directo (cronos_v1 000010). RRHH la ve,
	// pero nadie puede resolverla hasta que se publique la asignación.
	ErrResolucionPendienteAsignacion = errors.New("cronos resolucion pendiente de asignacion")
)

// ---- Quien resuelve: bandeja y resolución ----

type ProveedorMaterialResolucionPermisos interface {
	ProveerMaterialBandejaPermisos(context.Context, domain.MaterialBandejaPermisos) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	ProveerMaterialResolucionPermiso(context.Context, domain.MaterialResolucionPermiso) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// OrdenResolucionPermisos liga el contexto registrado de la petición al
// proveedor V3 nominal de la misma petición. Sin proveedor no hay orden.
type OrdenResolucionPermisos struct {
	contexto  vecdomain.ContextoActor
	proveedor ProveedorMaterialResolucionPermisos
}

func NuevaOrdenResolucionPermisos(contexto vecdomain.ContextoActor, proveedor ProveedorMaterialResolucionPermisos) (OrdenResolucionPermisos, error) {
	if dependenciaRemotaNula(proveedor) || contexto.Validar() != nil {
		return OrdenResolucionPermisos{}, ErrDependenciaNoDisponible
	}
	copia, err := contexto.Clonar()
	if err != nil {
		return OrdenResolucionPermisos{}, ErrDependenciaNoDisponible
	}
	return OrdenResolucionPermisos{contexto: copia, proveedor: proveedor}, nil
}

func (o OrdenResolucionPermisos) ContextoActor() (vecdomain.ContextoActor, error) {
	if dependenciaRemotaNula(o.proveedor) || o.contexto.Validar() != nil {
		return vecdomain.ContextoActor{}, ErrDependenciaNoDisponible
	}
	return o.contexto.Clonar()
}

func (o OrdenResolucionPermisos) ProveedorMaterial() ProveedorMaterialResolucionPermisos {
	if dependenciaRemotaNula(o.proveedor) {
		return nil
	}
	return o.proveedor
}

// SolicitudPendiente es una fila de la bandeja. EmpleadoEtiqueta es el
// nombre visible publicado con el circuito; puede faltar. Circuito es el
// aplicado (J-A salvo marca directa), no el del catálogo. PendienteAsignacion
// marca en la bandeja de RRHH lo que espera una jefatura y no se resuelve.
type SolicitudPendiente struct {
	SolicitudRef        string                        `json:"solicitud_ref"`
	EmpleadoRef         string                        `json:"empleado_ref"`
	EmpleadoEtiqueta    string                        `json:"empleado_etiqueta"`
	PermisoRef          string                        `json:"permiso_ref"`
	Nombre              string                        `json:"nombre"`
	Circuito            domain.CircuitoPermiso        `json:"circuito"`
	PendienteAsignacion bool                          `json:"pendiente_asignacion"`
	JustificanteExigido bool                          `json:"justificante_exigido"`
	Desde               string                        `json:"desde"`
	Hasta               string                        `json:"hasta"`
	HoraInicio          string                        `json:"hora_inicio,omitempty"`
	HoraFin             string                        `json:"hora_fin,omitempty"`
	Cantidad            int64                         `json:"cantidad"`
	Unidad              domain.LeaveUnit              `json:"unidad"`
	Estado              domain.EstadoSolicitudPermiso `json:"estado"`
	Version             int                           `json:"version"`
	SolicitadaEnUTC     time.Time                     `json:"solicitada_en"`
}

type BandejaPermisos struct {
	Paso       domain.PasoPermiso   `json:"paso"`
	Pendientes []SolicitudPendiente `json:"pendientes"`
}

type PeticionResolucionPermiso struct {
	ClaveOperacion, SolicitudRef string
	Paso                         domain.PasoPermiso
	Decision                     domain.DecisionPermiso
	Motivo                       string
	VersionEsperada              int
}

type ReciboResolucionPermiso struct {
	ResolucionRef string                        `json:"resolucion_ref"`
	SolicitudRef  string                        `json:"solicitud_ref"`
	ReciboRef     string                        `json:"recibo_ref"`
	Estado        domain.EstadoSolicitudPermiso `json:"estado"`
	Version       int                           `json:"version"`
	InstanteUTC   time.Time                     `json:"instante_utc"`
	Replay        bool                          `json:"replay"`
}

type RepositorioResolucionPermisos interface {
	// Cada llamada consume la decisión V3 nominal, audita y lee o escribe en
	// una única frontera durable, filtrada por el circuito publicado.
	ConsultarBandeja(context.Context, OrdenResolucionPermisos, domain.MaterialBandejaPermisos) (BandejaPermisos, error)
	ResolverPermiso(context.Context, OrdenResolucionPermisos, domain.MaterialResolucionPermiso) (ReciboResolucionPermiso, error)
}

type CasoUsoResolucionPermisos interface {
	ConsultarBandeja(context.Context, OrdenResolucionPermisos, domain.PasoPermiso) (BandejaPermisos, error)
	ResolverPermiso(context.Context, OrdenResolucionPermisos, PeticionResolucionPermiso) (ReciboResolucionPermiso, error)
}

// ---- La persona empleada: avisos de resolución ----

type ProveedorMaterialAvisosPropios interface {
	ProveerMaterialConsultaAvisosPropios(context.Context, domain.MaterialConsultaAvisosPropios) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	ProveerMaterialArchivoAvisoPropio(context.Context, domain.MaterialArchivoAvisoPropio) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type OrdenAvisosPropios struct {
	contexto  vecdomain.ContextoActor
	proveedor ProveedorMaterialAvisosPropios
}

func NuevaOrdenAvisosPropios(contexto vecdomain.ContextoActor, proveedor ProveedorMaterialAvisosPropios) (OrdenAvisosPropios, error) {
	if dependenciaRemotaNula(proveedor) || contexto.Validar() != nil {
		return OrdenAvisosPropios{}, ErrDependenciaNoDisponible
	}
	copia, err := contexto.Clonar()
	if err != nil {
		return OrdenAvisosPropios{}, ErrDependenciaNoDisponible
	}
	return OrdenAvisosPropios{contexto: copia, proveedor: proveedor}, nil
}

func (o OrdenAvisosPropios) ContextoActor() (vecdomain.ContextoActor, error) {
	if dependenciaRemotaNula(o.proveedor) || o.contexto.Validar() != nil {
		return vecdomain.ContextoActor{}, ErrDependenciaNoDisponible
	}
	return o.contexto.Clonar()
}

func (o OrdenAvisosPropios) ProveedorMaterial() ProveedorMaterialAvisosPropios {
	if dependenciaRemotaNula(o.proveedor) {
		return nil
	}
	return o.proveedor
}

// AvisoPropio: resolución final de una solicitud propia, con su motivo.
type AvisoPropio struct {
	AvisoRef       string                        `json:"aviso_ref"`
	SolicitudRef   string                        `json:"solicitud_ref"`
	Estado         domain.EstadoSolicitudPermiso `json:"estado"`
	Motivo         string                        `json:"motivo,omitempty"`
	ResueltoEnUTC  time.Time                     `json:"resuelto_en"`
	PermisoRef     string                        `json:"permiso_ref"`
	Nombre         string                        `json:"nombre"`
	Desde          string                        `json:"desde"`
	Hasta          string                        `json:"hasta"`
	HoraInicio     string                        `json:"hora_inicio,omitempty"`
	HoraFin        string                        `json:"hora_fin,omitempty"`
	Cantidad       int64                         `json:"cantidad"`
	Unidad         domain.LeaveUnit              `json:"unidad"`
	Archivado      bool                          `json:"archivado"`
	ArchivadoEnUTC *time.Time                    `json:"archivado_en,omitempty"`
}

type ConsultaAvisosPropios struct {
	Avisos []AvisoPropio `json:"avisos"`
}

type PeticionArchivoAviso struct {
	ClaveOperacion, AvisoRef string
}

type ReciboArchivoAviso struct {
	ArchivoRef  string    `json:"archivo_ref"`
	AvisoRef    string    `json:"aviso_ref"`
	ReciboRef   string    `json:"recibo_ref"`
	InstanteUTC time.Time `json:"instante_utc"`
	Replay      bool      `json:"replay"`
}

type RepositorioAvisosPropios interface {
	ConsultarAvisos(context.Context, OrdenAvisosPropios, domain.MaterialConsultaAvisosPropios) (ConsultaAvisosPropios, error)
	ArchivarAviso(context.Context, OrdenAvisosPropios, domain.MaterialArchivoAvisoPropio) (ReciboArchivoAviso, error)
}

type CasoUsoAvisosPropios interface {
	ConsultarAvisos(context.Context, OrdenAvisosPropios) (ConsultaAvisosPropios, error)
	ArchivarAviso(context.Context, OrdenAvisosPropios, PeticionArchivoAviso) (ReciboArchivoAviso, error)
}
