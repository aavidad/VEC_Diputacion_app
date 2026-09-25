package ports

import (
	"context"
	"encoding/json"
	"errors"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrAccesoCircuitoDenegado    = errors.New("dietas: acceso al circuito denegado")
	ErrCircuitoNoDisponible      = errors.New("dietas: circuito no disponible")
	ErrEstadoCircuitoConflicto   = errors.New("dietas: estado de circuito en conflicto")
	ErrResultadoCircuitoIncierto = errors.New("dietas: resultado de circuito incierto")
)

const (
	EsquemaEfectoCircuitoV2    = "vec.dietas.circuito-operacion.v2"
	TipoRecursoDocumentoDietas = "documento_dietas"
	TipoRecursoBandejaDietas   = "bandeja_dietas"
)

type SolicitudDecisionCircuito struct {
	Referencia        string                  `json:"referencia"`
	UnidadRef         string                  `json:"unidad_ref"`
	Etapa             domain.EtapaCircuito    `json:"etapa"`
	Decision          domain.DecisionCircuito `json:"decision"`
	Motivo            string                  `json:"motivo"`
	ClaveIdempotencia string                  `json:"clave_idempotencia"`
	VersionEsperada   uint64                  `json:"version_esperada"`
}

type ConsultaBandejaCircuito struct {
	Etapa      domain.EtapaCircuito `json:"etapa"`
	UnidadRef  string               `json:"unidad_ref"`
	FechaDesde string               `json:"fecha_desde,omitempty"`
	FechaHasta string               `json:"fecha_hasta,omitempty"`
	Limite     int                  `json:"limit"`
	Cursor     string               `json:"cursor,omitempty"`
}

type OperacionCircuito string

const (
	OperacionDecidirCircuito            OperacionCircuito = "decidir"
	OperacionListarBandeja              OperacionCircuito = "listar_bandeja"
	OperacionConsultarDocumentoCircuito OperacionCircuito = "consultar_documento"
)

// SolicitudDocumentoCircuito pide el documento que el actor tiene pendiente en
// una etapa. La unidad nunca procede del cliente: la fija la competencia.
type SolicitudDocumentoCircuito struct {
	Referencia string               `json:"referencia"`
	Etapa      domain.EtapaCircuito `json:"etapa"`
	UnidadRef  string               `json:"unidad_ref"`
}

type SolicitudOperacionCircuito struct {
	Operacion OperacionCircuito
	Decision  SolicitudDecisionCircuito
	Consulta  ConsultaBandejaCircuito
	Documento SolicitudDocumentoCircuito
}

// Etapa devuelve la etapa del circuito sobre la que actúa la operación.
func (s SolicitudOperacionCircuito) Etapa() domain.EtapaCircuito {
	switch s.Operacion {
	case OperacionDecidirCircuito:
		return s.Decision.Etapa
	case OperacionListarBandeja:
		return s.Consulta.Etapa
	case OperacionConsultarDocumentoCircuito:
		return s.Documento.Etapa
	default:
		return ""
	}
}

type AutorizacionCircuitoDurable struct {
	Material   vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	Accion     string
	RecursoRef string
	Finalidad  string
}

type IdentidadEfectivaCircuito struct {
	Vinculo              vecdomain.VinculoAutenticacionActorV2
	ContextoRegistrado   vecdomain.ResultadoContextoActorRegistradoV2
	Autorizacion         AutorizacionCircuitoDurable
	UnidadCompetenciaRef string
}

type EfectoAutorizacionCircuito struct {
	Material []byte
	Recurso  vecdomain.RecursoAutorizable
}

type VistaComisionCircuito struct {
	Referencia  string `json:"referencia"`
	Estado      string `json:"estado"`
	Version     uint64 `json:"version"`
	FechaInicio string `json:"fecha_inicio,omitempty"`
	FechaFin    string `json:"fecha_fin,omitempty"`
}

type ResultadoCircuitoComision struct {
	Comision VistaComisionCircuito  `json:"comision"`
	Recibo   ReciboBorradorComision `json:"recibo"`
}

type PaginaBandejaCircuito struct {
	Items           []VistaComisionCircuito `json:"items"`
	SiguienteCursor string                  `json:"siguiente_cursor,omitempty"`
}

// DocumentoCircuito es la vista del documento para quien lo revisa: fechas,
// motivo, itinerario, cálculo y líneas con sus justificantes. No incluye
// relación jurídica, unidad ni validadores de la persona titular.
type DocumentoCircuito struct {
	Referencia      string          `json:"referencia"`
	NumeroDocumento string          `json:"numero_documento"`
	FechaApertura   string          `json:"fecha_apertura"`
	Estado          string          `json:"estado"`
	Version         uint64          `json:"version"`
	FechaInicio     string          `json:"fecha_inicio"`
	FechaFin        string          `json:"fecha_fin"`
	HoraInicio      string          `json:"hora_inicio"`
	HoraFin         string          `json:"hora_fin"`
	Motivo          string          `json:"motivo"`
	CodigosRuta     []string        `json:"codigos_ruta"`
	VehiculoPropio  *bool           `json:"vehiculo_propio,omitempty"`
	Rutas           json.RawMessage `json:"rutas,omitempty"`
	Calculo         json.RawMessage `json:"calculo"`
	Documento       json.RawMessage `json:"documento"`
}

// Competencia de un revisor: unidad y etapa acreditadas por una fuente
// gobernada. Nunca se deduce de un cargo, de un perfil ni de una referencia
// libre de la asignación D7.
const (
	FuenteCompetenciaSinFuente  = "sin_fuente"
	FuenteCompetenciaAcreditada = "acreditada"
)

// ErrCompetenciaCircuitoSinFuente indica que no existe todavía la fuente
// gobernada que acredita quién revisa cada unidad. No es una denegación de
// permiso: la bandeja lo dice y no ofrece acciones.
var ErrCompetenciaCircuitoSinFuente = errors.New("dietas: competencia del circuito sin fuente gobernada")

type EstadoCompetenciasCircuito struct {
	Fuente string                 `json:"fuente"`
	Etapas []domain.EtapaCircuito `json:"etapas"`
}

// FuenteCompetenciaCircuito es la única autoridad que fija la unidad sobre
// la que el actor revisa en una etapa. Responde ErrCompetenciaCircuitoSinFuente
// mientras no exista el catálogo de validadores competentes.
type FuenteCompetenciaCircuito interface {
	EstadoCompetencias(context.Context, vecdomain.ResultadoContextoActorRegistradoV2) (EstadoCompetenciasCircuito, error)
	UnidadCompetente(context.Context, vecdomain.ResultadoContextoActorRegistradoV2, domain.EtapaCircuito) (string, error)
}

type ResolutorIdentidadEfectivaCircuito interface {
	ResolverIdentidadEfectivaCircuito(context.Context, SolicitudOperacionCircuito) (IdentidadEfectivaCircuito, error)
	EstadoCompetenciasCircuito(context.Context) (EstadoCompetenciasCircuito, error)
}

type RepositorioCircuitoComision interface {
	Decidir(context.Context, IdentidadEfectivaCircuito, SolicitudDecisionCircuito) (ResultadoCircuitoComision, error)
	ListarPendientes(context.Context, IdentidadEfectivaCircuito, ConsultaBandejaCircuito) (PaginaBandejaCircuito, error)
	ConsultarDocumento(context.Context, IdentidadEfectivaCircuito, SolicitudDocumentoCircuito) (DocumentoCircuito, error)
}
