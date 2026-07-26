package importacionconvoca

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

const (
	EstadoConciliacionPendiente   EstadoConciliacion  = "pendiente"
	EstadoConciliacionConfirmada  EstadoConciliacion  = "confirmada"
	EstadoConciliacionDescartada  EstadoConciliacion  = "descartada"
	EstadoStagingDisponible       EstadoStaging       = "disponible"
	EstadoStagingExpurgado        EstadoStaging       = "expurgado"
	ResultadoConciliadoConfirmado ResultadoConciliado = "confirmada"
	ResultadoConciliadoDescartado ResultadoConciliado = "descartada"

	MaximoLotesPorExpurgo   = 1_000
	maximaVersionPostgreSQL = uint64(1<<63 - 1)
)

var (
	ErrGestionDurableInvalida  = errors.New("bolsa: gestion durable Convoca invalida")
	ErrImportacionNoEncontrada = errors.New("bolsa: importacion Convoca no encontrada")
	ErrStagingExpurgado        = errors.New("bolsa: staging Convoca expurgado")
	ErrConciliacionEnConflicto = errors.New("bolsa: conciliacion Convoca en conflicto")
	ErrRetencionEnConflicto    = errors.New("bolsa: retencion Convoca en conflicto")

	referenciaOpacaDurable = regexp.MustCompile(`^[a-z][a-z0-9_.:/-]{2,511}$`)
	codigoGobernado        = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{2,127}$`)
)

type EstadoConciliacion string
type EstadoStaging string
type ResultadoConciliado string

type EstadoImportacion struct {
	Acta                     dominio.ActaImportacion
	EstadoConciliacion       EstadoConciliacion
	EstadoStaging            EstadoStaging
	PoliticaRetencionRef     string
	PoliticaRetencionVersion uint64
	ConservarStagingHasta    time.Time
	BloqueoRetencion         bool
	Version                  uint64
}

func (e EstadoImportacion) Validar() error {
	if e.Acta.Validar() != nil || !e.EstadoConciliacion.valido() ||
		!e.EstadoStaging.valido() || !referenciaValida(e.PoliticaRetencionRef) ||
		!versionPostgreSQLValida(e.PoliticaRetencionVersion) ||
		!versionPostgreSQLValida(e.Version) ||
		e.ConservarStagingHasta.Location() != time.UTC ||
		e.ConservarStagingHasta.Nanosecond()%1_000 != 0 ||
		!e.ConservarStagingHasta.After(e.Acta.RegistradaEn) {
		return ErrGestionDurableInvalida
	}
	return nil
}

type ConsultaImportacionesDurables interface {
	ConsultarEstado(context.Context, string) (EstadoImportacion, bool, error)
	RecuperarLote(context.Context, string) (dominio.LoteValidado, EstadoImportacion, bool, error)
}

type SolicitudConciliacion struct {
	ImportacionRef         string
	ConciliacionRef        string
	RegistroCorporativoRef string
	Resultado              ResultadoConciliado
	ActorRef               string
	MotivoCodigo           string
}

func (s SolicitudConciliacion) Validar() error {
	if !referenciaValida(s.ImportacionRef) || !referenciaValida(s.ConciliacionRef) ||
		!referenciaValida(s.RegistroCorporativoRef) || !codigoGobernado.MatchString(s.ActorRef) ||
		!codigoGobernado.MatchString(s.MotivoCodigo) || !s.Resultado.valido() {
		return ErrGestionDurableInvalida
	}
	return nil
}

type ConfirmacionConciliacion struct {
	ImportacionRef  string
	ConciliacionRef string
	Resultado       ResultadoConciliado
	RegistradaEn    time.Time
	Reutilizada     bool
}

func (c ConfirmacionConciliacion) ValidarPara(s SolicitudConciliacion) error {
	if s.Validar() != nil || c.ImportacionRef != s.ImportacionRef ||
		c.ConciliacionRef != s.ConciliacionRef || c.Resultado != s.Resultado ||
		c.RegistradaEn.IsZero() || c.RegistradaEn.Location() != time.UTC ||
		c.RegistradaEn.Nanosecond()%1_000 != 0 {
		return ErrGestionDurableInvalida
	}
	return nil
}

type RepositorioConciliaciones interface {
	Conciliar(context.Context, SolicitudConciliacion) (ConfirmacionConciliacion, error)
}

type SolicitudCambioBloqueoRetencion struct {
	ImportacionRef string
	DecisionRef    string
	ActorRef       string
	MotivoCodigo   string
	Bloqueado      bool
}

func (s SolicitudCambioBloqueoRetencion) Validar() error {
	if !referenciaValida(s.ImportacionRef) || !referenciaValida(s.DecisionRef) ||
		!codigoGobernado.MatchString(s.ActorRef) || !codigoGobernado.MatchString(s.MotivoCodigo) {
		return ErrGestionDurableInvalida
	}
	return nil
}

type ConfirmacionCambioBloqueo struct {
	ImportacionRef string
	DecisionRef    string
	Bloqueado      bool
	RegistradaEn   time.Time
	Reutilizada    bool
}

func (c ConfirmacionCambioBloqueo) ValidarPara(s SolicitudCambioBloqueoRetencion) error {
	if s.Validar() != nil || c.ImportacionRef != s.ImportacionRef ||
		c.DecisionRef != s.DecisionRef || c.Bloqueado != s.Bloqueado ||
		c.RegistradaEn.IsZero() || c.RegistradaEn.Location() != time.UTC ||
		c.RegistradaEn.Nanosecond()%1_000 != 0 {
		return ErrGestionDurableInvalida
	}
	return nil
}

type SolicitudExpurgoStaging struct {
	EjecucionRef    string
	ActorRef        string
	PoliticaRef     string
	PoliticaVersion uint64
	Limite          int
}

func (s SolicitudExpurgoStaging) Validar() error {
	if !referenciaValida(s.EjecucionRef) || !codigoGobernado.MatchString(s.ActorRef) ||
		!referenciaValida(s.PoliticaRef) || !versionPostgreSQLValida(s.PoliticaVersion) ||
		s.Limite < 1 || s.Limite > MaximoLotesPorExpurgo {
		return ErrGestionDurableInvalida
	}
	return nil
}

type ResultadoExpurgoStaging struct {
	EjecucionRef string
	Lotes        int
	Filas        int
	EjecutadaEn  time.Time
	Reutilizada  bool
}

func (r ResultadoExpurgoStaging) ValidarPara(s SolicitudExpurgoStaging) error {
	if s.Validar() != nil || r.EjecucionRef != s.EjecucionRef ||
		r.Lotes < 0 || r.Lotes > s.Limite || r.Filas < 0 ||
		r.EjecutadaEn.IsZero() || r.EjecutadaEn.Location() != time.UTC ||
		r.EjecutadaEn.Nanosecond()%1_000 != 0 {
		return ErrGestionDurableInvalida
	}
	return nil
}

type RepositorioRetencion interface {
	CambiarBloqueo(context.Context, SolicitudCambioBloqueoRetencion) (ConfirmacionCambioBloqueo, error)
	ExpurgarVencidos(context.Context, SolicitudExpurgoStaging) (ResultadoExpurgoStaging, error)
}

func (e EstadoConciliacion) valido() bool {
	return e == EstadoConciliacionPendiente ||
		e == EstadoConciliacionConfirmada ||
		e == EstadoConciliacionDescartada
}

func (e EstadoStaging) valido() bool {
	return e == EstadoStagingDisponible || e == EstadoStagingExpurgado
}

func (r ResultadoConciliado) valido() bool {
	return r == ResultadoConciliadoConfirmado || r == ResultadoConciliadoDescartado
}

func referenciaValida(valor string) bool {
	return referenciaOpacaDurable.MatchString(valor) && strings.TrimSpace(valor) == valor
}

func versionPostgreSQLValida(version uint64) bool {
	return version > 0 && version <= maximaVersionPostgreSQL
}
