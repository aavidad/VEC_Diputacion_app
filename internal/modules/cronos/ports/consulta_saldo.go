package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrConsultaSaldoInvalida = errors.New("cronos consulta de saldo invalida")

type PeriodoSaldo string

const (
	PeriodoSaldoHoy    PeriodoSaldo = "hoy"
	PeriodoSaldoSemana PeriodoSaldo = "semana"
	PeriodoSaldoMes    PeriodoSaldo = "mes"
	PeriodoSaldoAnio   PeriodoSaldo = "anio"
	PeriodoSaldoRango  PeriodoSaldo = "rango"
)

const (
	EstadoSaldoDisponible   = "disponible"
	EstadoSaldoNoDisponible = "no_disponible"
	EstadoSaldoIncompleto   = "incompleto"
)

// FuenteSaldo is an authorized, employee-scoped snapshot. Dates are civil dates
// in the zone supplied to the application; Desde and Hasta are inclusive.
type FuenteSaldo struct {
	EmpleadoRef      string
	Desde            string
	Hasta            string
	ZonaHoraria      string
	Completo         bool
	Jornadas         []JornadaPrevista
	Marcajes         []MarcajeSaldo
	MovimientosSaldo []MovimientoSaldo
	EfectosC3        *FuenteSaldoEfectosC3
}

type FuenteSaldoEfectosC3 struct {
	SnapshotRef       string
	FuenteSHA256      string
	CorteUTC          time.Time
	Politicas         []domain.PoliticaSaldoEfectosC3
	Programaciones    []domain.ProgramacionSaldoEfectosC3
	Permisos          []domain.PermisoConcedidoSaldoEfectosC3
	CoberturaPermisos bool
}

// MovimientoSaldo is the traceable book entry. The application never writes
// a mutable balance and does not infer approved adjustments from a punch.
type MovimientoSaldo struct {
	Fecha              string
	Tipo               string
	DeltaMicrosegundos int64
	Fuentes            []string
}

// ProveedorMaterialConsultaSaldoPropio es la autoridad nominal V3 de la
// lectura del saldo propio. Recibe material construido por el servidor.
type ProveedorMaterialConsultaSaldoPropio interface {
	ProveerMaterialConsultaSaldoPropio(context.Context, domain.MaterialConsultaSaldoPropio) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// OrdenConsultaSaldo is built by the trusted identity boundary from the
// registered actor context. A raw employee reference is never authority.
// The durable reader requires the nominal V3 provider; an order without it
// can only be served by in-memory test adapters.
type OrdenConsultaSaldo struct {
	contexto  vecdomain.ContextoActor
	proveedor ProveedorMaterialConsultaSaldoPropio
}

func NuevaOrdenConsultaSaldo(contexto vecdomain.ContextoActor) (OrdenConsultaSaldo, error) {
	if contexto.Validar() != nil {
		return OrdenConsultaSaldo{}, ErrConsultaSaldoInvalida
	}
	copia, err := contexto.Clonar()
	if err != nil {
		return OrdenConsultaSaldo{}, ErrConsultaSaldoInvalida
	}
	return OrdenConsultaSaldo{contexto: copia}, nil
}

// NuevaOrdenConsultaSaldoAutorizada liga el contexto registrado al proveedor
// V3 de la misma petición. Sin proveedor no hay orden.
func NuevaOrdenConsultaSaldoAutorizada(contexto vecdomain.ContextoActor, proveedor ProveedorMaterialConsultaSaldoPropio) (OrdenConsultaSaldo, error) {
	if dependenciaRemotaNula(proveedor) {
		return OrdenConsultaSaldo{}, ErrDependenciaNoDisponible
	}
	orden, err := NuevaOrdenConsultaSaldo(contexto)
	if err != nil {
		return OrdenConsultaSaldo{}, err
	}
	orden.proveedor = proveedor
	return orden, nil
}

func (o OrdenConsultaSaldo) ProveedorMaterial() ProveedorMaterialConsultaSaldoPropio {
	if dependenciaRemotaNula(o.proveedor) {
		return nil
	}
	return o.proveedor
}

func (o OrdenConsultaSaldo) ContextoActor() (vecdomain.ContextoActor, error) {
	if o.contexto.Validar() != nil {
		return vecdomain.ContextoActor{}, ErrConsultaSaldoInvalida
	}
	return o.contexto.Clonar()
}

type JornadaPrevista struct {
	Fecha              string
	ProgramacionRef    string
	TurnoRef           string
	PoliticaVersionRef string
	FuenteRef          string
	Version            int64
	MinutosPrevistos   int64
}

type MarcajeSaldo struct {
	MarcajeRef  string
	Movimiento  domain.PunchKind
	InstanteUTC time.Time
	Canal       CanalSaldo
	OrigenRef   string
	TipoOrigen  *string
}

type CanalSaldo struct {
	PoliticaVersionRef string
	CanalRef           string
	OrigenRef          string
	CalidadRef         string
}

type RepositorioConsultaSaldo interface {
	// ConsultarFuenteSaldo must consume nominal read authorization, audit and
	// read in one durable boundary. It may include bordering night-shift facts.
	ConsultarFuenteSaldo(context.Context, OrdenConsultaSaldo, string, string, string, string) (FuenteSaldo, error)
}

type PeriodoConsultaSaldo struct {
	Tipo  PeriodoSaldo `json:"tipo"`
	Desde string       `json:"desde"`
	Hasta string       `json:"hasta"`
}

type ResumenConsultaSaldo struct {
	PrevistosMinutos      *int64 `json:"previstos_minutos"`
	TrabajadosMinutos     int64  `json:"trabajados_minutos"`
	SaldoMinutos          *int64 `json:"saldo_minutos"`
	Estado                string `json:"estado"`
	SaldoBaseMinutos      *int64 `json:"saldo_base_minutos,omitempty"`
	CreditoPermisoMinutos *int64 `json:"credito_permiso_minutos,omitempty"`
	SnapshotRef           string `json:"snapshot_ref,omitempty"`
}

type DetalleSaldoDia struct {
	Fecha                    string               `json:"fecha"`
	ProgramacionRef          string               `json:"programacion_ref,omitempty"`
	TurnoRef                 string               `json:"turno_ref,omitempty"`
	PoliticaVersionRef       string               `json:"politica_version_ref,omitempty"`
	FuenteRef                string               `json:"fuente_ref,omitempty"`
	ProgramacionVersion      int64                `json:"programacion_version,omitempty"`
	PrevistosMinutos         *int64               `json:"previstos_minutos"`
	TrabajadosMinutos        int64                `json:"trabajados_minutos"`
	PausasMinutos            int64                `json:"pausas_minutos"`
	SaldoMinutos             *int64               `json:"saldo_minutos"`
	Estado                   string               `json:"estado"`
	Marcajes                 []MarcajeDia         `json:"marcajes"`
	SaldoBaseMinutos         *int64               `json:"saldo_base_minutos,omitempty"`
	CreditoPermisoMinutos    *int64               `json:"credito_permiso_minutos,omitempty"`
	CausaEfecto              string               `json:"causa_efecto,omitempty"`
	PoliticaEfectosRef       string               `json:"politica_efectos_ref,omitempty"`
	PoliticaEfectosVersion   int64                `json:"politica_efectos_version,omitempty"`
	PoliticaEfectosSHA256    string               `json:"politica_efectos_sha256,omitempty"`
	ReglaEfectoRef           string               `json:"regla_efecto_ref,omitempty"`
	ResolucionPermisoRef     string               `json:"resolucion_permiso_ref,omitempty"`
	ResolucionPermisoVersion int64                `json:"resolucion_permiso_version,omitempty"`
	FuenteEfectoRef          string               `json:"fuente_efecto_ref,omitempty"`
	MovimientosEfecto        []MovimientoEfectoC3 `json:"movimientos_efecto,omitempty"`
}

type MovimientoEfectoC3 struct {
	Tipo                   string    `json:"tipo"`
	DeltaMinutos           int64     `json:"delta_minutos"`
	SolicitudRef           string    `json:"solicitud_ref"`
	ResolucionRef          string    `json:"resolucion_ref"`
	ResolucionVersion      int64     `json:"resolucion_version"`
	PoliticaRef            string    `json:"politica_ref"`
	PoliticaVersion        int64     `json:"politica_version"`
	PoliticaSHA256         string    `json:"politica_sha256"`
	ReglaRef               string    `json:"regla_ref"`
	FuenteRef              string    `json:"fuente_ref"`
	CatalogoVersionRef     string    `json:"catalogo_version_ref"`
	CatalogoFuenteRef      string    `json:"catalogo_fuente_ref"`
	ResolucionDecisionRef  string    `json:"resolucion_decision_ref"`
	ResolucionAuditoriaRef string    `json:"resolucion_auditoria_ref"`
	ResueltaEn             time.Time `json:"resuelta_en"`
	PendienteJustificar    bool      `json:"pendiente_justificar"`
}

type MarcajeDia struct {
	InstanteUTC time.Time        `json:"instante_utc"`
	Movimiento  domain.PunchKind `json:"movimiento"`
	Origen      *string          `json:"origen"`
}

type ConsultaSaldo struct {
	Periodo PeriodoConsultaSaldo `json:"periodo"`
	Resumen ResumenConsultaSaldo `json:"resumen"`
	Detalle []DetalleSaldoDia    `json:"detalle"`
}

type CasoUsoConsultarSaldo interface {
	// desde and hasta are required only for rango, formatted YYYY-MM-DD.
	ConsultarSaldo(context.Context, OrdenConsultaSaldo, PeriodoSaldo, string, string) (ConsultaSaldo, error)
}
