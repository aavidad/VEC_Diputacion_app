package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
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
}

// MovimientoSaldo is the traceable book entry. The application never writes
// a mutable balance and does not infer approved adjustments from a punch.
type MovimientoSaldo struct {
	Fecha              string
	Tipo               string
	DeltaMicrosegundos int64
	Fuentes            []string
}

// OrdenConsultaSaldo is built by the trusted identity boundary from the
// registered actor context. A raw employee reference is never authority.
type OrdenConsultaSaldo struct{ contexto vecdomain.ContextoActor }

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

func (o OrdenConsultaSaldo) ContextoActor() (vecdomain.ContextoActor, error) {
	if o.contexto.Validar() != nil {
		return vecdomain.ContextoActor{}, ErrConsultaSaldoInvalida
	}
	return o.contexto.Clonar()
}

type JornadaPrevista struct {
	Fecha              string
	TurnoRef           string
	PoliticaVersionRef string
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
	PrevistosMinutos  *int64 `json:"previstos_minutos"`
	TrabajadosMinutos int64  `json:"trabajados_minutos"`
	SaldoMinutos      *int64 `json:"saldo_minutos"`
	Estado            string `json:"estado"`
}

type DetalleSaldoDia struct {
	Fecha              string       `json:"fecha"`
	TurnoRef           string       `json:"turno_ref,omitempty"`
	PoliticaVersionRef string       `json:"politica_version_ref,omitempty"`
	PrevistosMinutos   *int64       `json:"previstos_minutos"`
	TrabajadosMinutos  int64        `json:"trabajados_minutos"`
	PausasMinutos      int64        `json:"pausas_minutos"`
	SaldoMinutos       *int64       `json:"saldo_minutos"`
	Estado             string       `json:"estado"`
	Marcajes           []MarcajeDia `json:"marcajes"`
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
