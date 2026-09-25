package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Motivos nominales por los que la función durable rechaza una solicitud de
// la persona empleada. Ninguno se interpreta como autorización ni éxito.
var (
	ErrSolicitudCronosInvalida = errors.New("cronos solicitud invalida")
	ErrPermisoNoSolicitable    = errors.New("cronos permiso no solicitable")
	ErrCalendarioNoPublicado   = errors.New("cronos calendario laboral no publicado")
	ErrPermisoFueraDeLimites   = errors.New("cronos permiso fuera de minimo, maximo o cupo")
	ErrPermisoSolapado         = errors.New("cronos permiso solapado con otro vivo")
)

// ---- Movimientos: calendario, absentismos y correcciones propias ----

type ProveedorMaterialConsultaMovimientosPropios interface {
	ProveerMaterialConsultaMovimientosPropios(context.Context, domain.MaterialConsultaMovimientosPropios) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// OrdenConsultaMovimientos liga el contexto registrado de la petición al
// proveedor V3 nominal de la misma petición. Sin proveedor no hay orden.
type OrdenConsultaMovimientos struct {
	contexto  vecdomain.ContextoActor
	proveedor ProveedorMaterialConsultaMovimientosPropios
}

func NuevaOrdenConsultaMovimientos(contexto vecdomain.ContextoActor, proveedor ProveedorMaterialConsultaMovimientosPropios) (OrdenConsultaMovimientos, error) {
	if dependenciaRemotaNula(proveedor) || contexto.Validar() != nil {
		return OrdenConsultaMovimientos{}, ErrDependenciaNoDisponible
	}
	copia, err := contexto.Clonar()
	if err != nil {
		return OrdenConsultaMovimientos{}, ErrDependenciaNoDisponible
	}
	return OrdenConsultaMovimientos{contexto: copia, proveedor: proveedor}, nil
}

func (o OrdenConsultaMovimientos) ContextoActor() (vecdomain.ContextoActor, error) {
	if dependenciaRemotaNula(o.proveedor) || o.contexto.Validar() != nil {
		return vecdomain.ContextoActor{}, ErrDependenciaNoDisponible
	}
	return o.contexto.Clonar()
}

func (o OrdenConsultaMovimientos) ProveedorMaterial() ProveedorMaterialConsultaMovimientosPropios {
	if dependenciaRemotaNula(o.proveedor) {
		return nil
	}
	return o.proveedor
}

type DiaCalendario struct {
	Fecha  string `json:"fecha"`
	Tipo   string `json:"tipo"`
	Nombre string `json:"nombre"`
}

// CalendarioMovimientos: Disponible=false cuando falta el calendario de
// algún año del periodo; entonces Dias va vacío y nada se infiere.
type CalendarioMovimientos struct {
	Disponible bool            `json:"disponible"`
	Dias       []DiaCalendario `json:"dias"`
}

type MarcajesDia struct {
	Fecha    string `json:"fecha"`
	Marcajes int    `json:"marcajes"`
}

// Absentismo es un permiso concedido que ocupa días del periodo.
type Absentismo struct {
	SolicitudRef        string           `json:"solicitud_ref"`
	PermisoRef          string           `json:"permiso_ref"`
	Nombre              string           `json:"nombre"`
	Desde               string           `json:"desde"`
	Hasta               string           `json:"hasta"`
	Cantidad            int64            `json:"cantidad"`
	Unidad              domain.LeaveUnit `json:"unidad"`
	PendienteJustificar bool             `json:"pendiente_justificar"`
}

type CorreccionPropia struct {
	SolicitudRef    string                  `json:"solicitud_ref"`
	FechaCivil      string                  `json:"fecha_civil"`
	HoraPretendida  string                  `json:"hora_pretendida"`
	Movimiento      domain.PunchKind        `json:"movimiento"`
	Estado          domain.EstadoCorreccion `json:"estado"`
	Version         int                     `json:"version"`
	SolicitadaEnUTC time.Time               `json:"solicitada_en"`
}

type ConsultaMovimientos struct {
	Periodo        PeriodoConsultaSaldo  `json:"periodo"`
	Calendario     CalendarioMovimientos `json:"calendario"`
	MarcajesPorDia []MarcajesDia         `json:"marcajes_por_dia"`
	Absentismos    []Absentismo          `json:"absentismos"`
	Correcciones   []CorreccionPropia    `json:"correcciones"`
}

type RepositorioConsultaMovimientos interface {
	// ConsultarMovimientos consume la lectura nominal, audita y lee en una
	// única frontera durable. Devuelve el periodo exacto consultado.
	ConsultarMovimientos(ctx context.Context, orden OrdenConsultaMovimientos, empleado, desde, hasta, zona string) (ConsultaMovimientos, error)
}

type CasoUsoConsultarMovimientos interface {
	ConsultarMovimientos(context.Context, OrdenConsultaMovimientos, PeriodoSaldo, string, string) (ConsultaMovimientos, error)
}

// ---- Permisos propios: catálogo del año y solicitud ----

type ProveedorMaterialPermisosPropios interface {
	ProveerMaterialConsultaPermisosPropios(context.Context, domain.MaterialConsultaPermisosPropios) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	ProveerMaterialSolicitudPermisoPropio(context.Context, domain.MaterialSolicitudPermisoPropio) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type OrdenPermisosPropios struct {
	contexto  vecdomain.ContextoActor
	proveedor ProveedorMaterialPermisosPropios
}

func NuevaOrdenPermisosPropios(contexto vecdomain.ContextoActor, proveedor ProveedorMaterialPermisosPropios) (OrdenPermisosPropios, error) {
	if dependenciaRemotaNula(proveedor) || contexto.Validar() != nil {
		return OrdenPermisosPropios{}, ErrDependenciaNoDisponible
	}
	copia, err := contexto.Clonar()
	if err != nil {
		return OrdenPermisosPropios{}, ErrDependenciaNoDisponible
	}
	return OrdenPermisosPropios{contexto: copia, proveedor: proveedor}, nil
}

func (o OrdenPermisosPropios) ContextoActor() (vecdomain.ContextoActor, error) {
	if dependenciaRemotaNula(o.proveedor) || o.contexto.Validar() != nil {
		return vecdomain.ContextoActor{}, ErrDependenciaNoDisponible
	}
	return o.contexto.Clonar()
}

func (o OrdenPermisosPropios) ProveedorMaterial() ProveedorMaterialPermisosPropios {
	if dependenciaRemotaNula(o.proveedor) {
		return nil
	}
	return o.proveedor
}

// EntradaCatalogoPropio: versión vigente del catálogo para el año, con la
// marca de si se solicita desde el portal y de si sus valores son sintéticos.
type EntradaCatalogoPropio struct {
	Version     domain.CatalogoPermisoVersion
	Solicitable bool
	Sintetico   bool
}

type SolicitudPermisoPropia struct {
	SolicitudRef        string                        `json:"solicitud_ref"`
	CatalogoVersionRef  string                        `json:"catalogo_version_ref"`
	PermisoRef          string                        `json:"permiso_ref"`
	Desde               string                        `json:"desde"`
	Hasta               string                        `json:"hasta"`
	HoraInicio          string                        `json:"hora_inicio,omitempty"`
	HoraFin             string                        `json:"hora_fin,omitempty"`
	Cantidad            int64                         `json:"cantidad"`
	Unidad              domain.LeaveUnit              `json:"unidad"`
	Estado              domain.EstadoSolicitudPermiso `json:"estado"`
	Version             int                           `json:"version"`
	PendienteJustificar bool                          `json:"pendiente_justificar"`
	SolicitadaEnUTC     time.Time                     `json:"solicitada_en"`
}

type FuentePermisosPropios struct {
	EmpleadoRef string
	Anio        int
	Catalogo    []EntradaCatalogoPropio
	Solicitudes []SolicitudPermisoPropia
}

// PermisoAnualPropio es una fila del listado anual: catálogo más lo
// solicitado pendiente, lo concedido, lo pendiente de justificar y la resta.
type PermisoAnualPropio struct {
	PermisoRef          string                 `json:"permiso_ref"`
	VersionRef          string                 `json:"version_ref"`
	Nombre              string                 `json:"nombre"`
	Unidad              domain.LeaveUnit       `json:"unidad"`
	Computo             domain.ComputoPermiso  `json:"computo"`
	Circuito            domain.CircuitoPermiso `json:"circuito"`
	Minimo              int64                  `json:"minimo"`
	MaximoSolicitud     *int64                 `json:"maximo_solicitud"`
	MaximoMensual       *int64                 `json:"maximo_mensual"`
	MaximoAnual         *int64                 `json:"maximo_anual"`
	JustificanteExigido bool                   `json:"justificante_exigido"`
	Solicitable         bool                   `json:"solicitable"`
	Sintetico           bool                   `json:"sintetico"`
	Solicitado          int64                  `json:"solicitado"`
	Concedido           int64                  `json:"concedido"`
	PendienteJustificar int64                  `json:"pendiente_justificar"`
	Resta               *int64                 `json:"resta"`
	SinConciliar        bool                   `json:"sin_conciliar"`
}

type ConsultaPermisosPropios struct {
	Anio        int                      `json:"anio"`
	Permisos    []PermisoAnualPropio     `json:"permisos"`
	Solicitudes []SolicitudPermisoPropia `json:"solicitudes"`
}

type PeticionPermisoPropio struct {
	PermisoRef, Desde, Hasta, HoraInicio, HoraFin, ClaveOperacion string
}

type ReciboPermisoPropio struct {
	SolicitudRef       string                        `json:"solicitud_ref"`
	ReciboRef          string                        `json:"recibo_ref"`
	CatalogoVersionRef string                        `json:"catalogo_version_ref"`
	Version            int                           `json:"version"`
	Estado             domain.EstadoSolicitudPermiso `json:"estado"`
	Cantidad           int64                         `json:"cantidad"`
	Unidad             domain.LeaveUnit              `json:"unidad"`
	InstanteUTC        time.Time                     `json:"instante_utc"`
	Replay             bool                          `json:"replay"`
}

type RepositorioPermisosPropios interface {
	ConsultarPermisosPropios(ctx context.Context, orden OrdenPermisosPropios, empleado string, anio int, zona string) (FuentePermisosPropios, error)
	SolicitarPermisoPropio(ctx context.Context, orden OrdenPermisosPropios, material domain.MaterialSolicitudPermisoPropio) (ReciboPermisoPropio, error)
}

type CasoUsoPermisosPropios interface {
	ConsultarPermisosPropios(context.Context, OrdenPermisosPropios, int) (ConsultaPermisosPropios, error)
	SolicitarPermisoPropio(context.Context, OrdenPermisosPropios, PeticionPermisoPropio) (ReciboPermisoPropio, error)
}
