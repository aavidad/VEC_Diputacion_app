package ports

import (
	"context"
	"errors"

	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrCeseNativoCTInvalido     = errors.New("bolsa: cese nativo CT inválido")
	ErrCeseNativoCTDenegado     = errors.New("bolsa: cese nativo CT denegado")
	ErrCeseNativoCTConflicto    = errors.New("bolsa: cese nativo CT divergente")
	ErrCeseNativoCTNoDisponible = errors.New("bolsa: cese nativo CT no disponible")
	ErrAltaNativaCTInvalida     = errors.New("bolsa: alta nativa CT inválida")
	ErrAltaNativaCTDenegada     = errors.New("bolsa: alta nativa CT denegada")
	ErrAltaNativaCTConflicto    = errors.New("bolsa: alta nativa CT divergente")
	ErrAltaNativaCTNoDisponible = errors.New("bolsa: alta nativa CT no disponible")
)

// PruebaCierrePersonalCeseCT es el contrato cerrado de P48. Su contenido
// procede de ambos recibos originales de Personal y del cese CT confirmado.
type PruebaCierrePersonalCeseCT struct {
	Esquema            string `json:"esquema"`
	ReciboCTRef        string `json:"recibo_ct_ref"`
	VersionCT          uint64 `json:"version_ct"`
	ActorRef           string `json:"actor_ref"`
	EmpleadoRef        string `json:"empleado_ref"`
	RelacionRef        string `json:"relacion_ref"`
	OcupacionRef       string `json:"ocupacion_ref"`
	OrganismoRef       string `json:"organismo_ref"`
	UnidadRef          string `json:"unidad_ref"`
	RelacionReciboRef  string `json:"relacion_recibo_ref"`
	RelacionVersion    uint64 `json:"relacion_version"`
	OcupacionReciboRef string `json:"ocupacion_recibo_ref"`
	OcupacionVersion   uint64 `json:"ocupacion_version"`
	VigenteHasta       string `json:"vigente_hasta"`
	FuenteHuellaSHA256 string `json:"fuente_huella_sha256"`
}

const EsquemaPruebaCierrePersonalCeseCT = "vec.personal.cierre-cese-ct.bolsa.v1"
const EsquemaActoCeseNativoCT = "vec.bolsa.cese-nativo-ct.acto.v1"
const EsquemaAltaNativaCT = "vec.bolsa.alta-nativa-ct.acto.v1"

// PruebaAltaPersonalCT liga la confirmación CT155 con los recibos P23
// originales. El propietario CT vuelve a comprobar P23 antes del primer alta.
type PruebaAltaPersonalCT struct {
	ReciboCTRef        string `json:"recibo_ct_ref"`
	VersionExpediente  uint64 `json:"version_expediente"`
	ActorRef           string `json:"actor_ref"`
	EmpleadoRef        string `json:"empleado_ref"`
	RelacionRef        string `json:"relacion_ref"`
	RelacionVersion    uint64 `json:"relacion_version"`
	RelacionReciboRef  string `json:"relacion_recibo_ref"`
	OcupacionRef       string `json:"ocupacion_ref"`
	OcupacionVersion   uint64 `json:"ocupacion_version"`
	OcupacionReciboRef string `json:"ocupacion_recibo_ref"`
	PlanRef            string `json:"plan_ref"`
	PlanSHA256         string `json:"plan_sha256"`
}

type MaterialActoAltaNativaCT struct {
	Esquema               string                     `json:"esquema"`
	Selector              SelectorAceptacionNativaCT `json:"selector"`
	ParticipacionRef      string                     `json:"participacion_ref"`
	UnidadAutorizacionRef string                     `json:"unidad_autorizacion_ref"`
	AmbitoRef             string                     `json:"ambito_ref"`
	Personal              PruebaAltaPersonalCT       `json:"personal"`
}

// MaterialActoCeseNativoCT es la intención completa ligada por SHA-256 al ACT
// V3 de la participación. P48 recibe el nodo Personal sin reserialización.
type MaterialActoCeseNativoCT struct {
	Esquema               string                     `json:"esquema"`
	Selector              SelectorAceptacionNativaCT `json:"selector"`
	ParticipacionRef      string                     `json:"participacion_ref"`
	UnidadAutorizacionRef string                     `json:"unidad_autorizacion_ref"`
	AmbitoRef             string                     `json:"ambito_ref"`
	Personal              PruebaCierrePersonalCeseCT `json:"personal"`
}

// Todas las referencias son hechos opacos ya confirmados por CT, Personal y
// Bolsa. La participación procede del anclaje B113 leído por composición;
// B115 la resuelve de nuevo desde el origen B110 antes de escribir.
type SolicitudCeseNativoCT struct {
	Selector           SelectorAceptacionNativaCT
	ParticipacionRef   string
	Personal           PruebaCierrePersonalCeseCT
	Vinculo            core.VinculoAutenticacionActorV2
	ResultadoContexto  core.ResultadoContextoActorRegistradoV2
	Correlacion        core.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion core.ReferenciaEntradaCatalogo
}

type OrdenCeseNativoCT struct {
	Solicitud        SolicitudCeseNativoCT
	ActorRef         string
	MaterialCanonico []byte
	Material         vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// El acuse usa las referencias B13/B45 originales. DisponibleDesde resulta
// de la política de cese propia de Bolsa; B90 decide elegibilidad al corte.
type ReciboCeseNativoCT struct {
	Reutilizada      bool   `json:"reutilizada"`
	ReciboRef        string `json:"recibo_ref"`
	AuditoriaRef     string `json:"auditoria_ref"`
	EventoRef        string `json:"evento_ref"`
	ParticipacionRef string `json:"participacion_ref"`
	BolsaRef         string `json:"bolsa_ref"`
	FechaEfecto      string `json:"fecha_efecto"`
	DisponibleDesde  string `json:"disponible_desde"`
	PoliticaVersion  uint64 `json:"politica_version"`
}

type RepositorioCeseNativoCT interface {
	AplicarCeseNativoCT(context.Context, OrdenCeseNativoCT) (ReciboCeseNativoCT, error)
}

type AplicadorCeseNativoCT interface {
	AplicarCeseNativoCT(context.Context, SolicitudCeseNativoCT) (ReciboCeseNativoCT, error)
}

type SolicitudAltaNativaCT struct {
	Selector           SelectorAceptacionNativaCT
	ParticipacionRef   string
	Personal           PruebaAltaPersonalCT
	Vinculo            core.VinculoAutenticacionActorV2
	ResultadoContexto  core.ResultadoContextoActorRegistradoV2
	Correlacion        core.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion core.ReferenciaEntradaCatalogo
}

type OrdenAltaNativaCT struct {
	Solicitud        SolicitudAltaNativaCT
	ActorRef         string
	MaterialCanonico []byte
	Material         vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ReciboAltaNativaCT struct {
	Reutilizada      bool   `json:"reutilizada"`
	EventoRef        string `json:"evento_ref"`
	ReciboSituacion  string `json:"recibo_situacion_ref"`
	AuditoriaRef     string `json:"auditoria_ref"`
	OutboxRef        string `json:"outbox_ref"`
	BolsaRef         string `json:"bolsa_ref"`
	ParticipacionRef string `json:"participacion_ref"`
	TrabajandoDesde  string `json:"trabajando_desde"`
}

type RepositorioAltaNativaCT interface {
	AplicarAltaNativaCT(context.Context, OrdenAltaNativaCT) (ReciboAltaNativaCT, error)
}

type AplicadorAltaNativaCT interface {
	AplicarAltaNativaCT(context.Context, SolicitudAltaNativaCT) (ReciboAltaNativaCT, error)
}
