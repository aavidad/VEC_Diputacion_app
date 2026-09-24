package ports

import (
	"context"
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
	EsquemaEfectoCircuitoV1    = "vec.dietas.circuito-operacion.v1"
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
	OperacionDecidirCircuito OperacionCircuito = "decidir"
	OperacionListarBandeja   OperacionCircuito = "listar_bandeja"
)

type SolicitudOperacionCircuito struct {
	Operacion OperacionCircuito
	Decision  SolicitudDecisionCircuito
	Consulta  ConsultaBandejaCircuito
}

// SelloAsignacionPersonal representa solo referencias de la asignación D7.
// La autoridad Personal debe revalidarlo al consumir el efecto; una copia
// recibida del navegador o una etiqueta de cargo no constituye el sello.
type SelloAsignacionPersonal struct {
	AsignacionRef            string `json:"asignacion_ref"`
	Version                  uint64 `json:"version"`
	RelacionRef              string `json:"relacion_ref"`
	PersonaRef               string `json:"persona_ref"`
	UnidadRef                string `json:"unidad_ref"`
	CentroRef                string `json:"centro_ref"`
	AdministrativoPersonaRef string `json:"administrativo_persona_ref"`
	ResponsablePersonaRef    string `json:"responsable_persona_ref"`
	VigenteDesde             string `json:"vigente_desde"`
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
	Asignacion           SelloAsignacionPersonal
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

type ResolutorIdentidadEfectivaCircuito interface {
	ResolverIdentidadEfectivaCircuito(context.Context, SolicitudOperacionCircuito) (IdentidadEfectivaCircuito, error)
}

type RepositorioCircuitoComision interface {
	Decidir(context.Context, IdentidadEfectivaCircuito, SolicitudDecisionCircuito) (ResultadoCircuitoComision, error)
	ListarPendientes(context.Context, IdentidadEfectivaCircuito, ConsultaBandejaCircuito) (PaginaBandejaCircuito, error)
}
