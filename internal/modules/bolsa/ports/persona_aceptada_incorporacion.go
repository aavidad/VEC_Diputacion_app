package ports

import (
	"context"
	"errors"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionConsultaPersonaAceptacionCT    = "bolsa.aceptacion_ct.persona.consultar"
	AudienciaConsultaPersonaAceptacionCT = "vec_bolsa_llamamientos.aceptacion_ct.persona.v1"
	FinalidadConsultaPersonaAceptacionCT = "preparar_incorporacion_ct"
	TipoRecursoPersonaAceptacionCT       = "persona_aceptacion_ct"
)

var (
	ErrConsultaPersonaAceptacionCTInvalida     = errors.New("bolsa: consulta de persona aceptada invalida")
	ErrConsultaPersonaAceptacionCTDenegada     = errors.New("bolsa: consulta de persona aceptada denegada")
	ErrConsultaPersonaAceptacionCTNoDisponible = errors.New("bolsa: consulta de persona aceptada no disponible")
)

// El selector procede de una preparación CT confiable y conserva aceptación
// y apertura originales. Nunca acepta persona, candidato ni datos identificativos.
type SelectorPersonaAceptacionCT struct {
	UnidadRef                string `json:"unidad_ref"`
	CategoriaRef             string `json:"categoria_ref"`
	NecesidadRef             string `json:"necesidad_ref"`
	AceptacionOperacionRef   string `json:"aceptacion_operacion_ref"`
	AceptacionRegistroSHA256 string `json:"aceptacion_registro_sha256"`
	AperturaOperacionRef     string `json:"apertura_operacion_ref"`
	AperturaRegistroSHA256   string `json:"apertura_registro_sha256"`
	LlamamientoRef           string `json:"llamamiento_ref"`
	PropuestaRef             string `json:"propuesta_ref"`
}

type SolicitudConsultaPersonaAceptacionCT struct {
	Selector       SelectorPersonaAceptacionCT
	ActorConfiable ActorConfiablePersonaAceptacionCT
}

type ActorConfiablePersonaAceptacionCT struct {
	Vinculo   core.VinculoAutenticacionActorV2
	Resultado core.ResultadoContextoActorRegistradoV2
}

// El proveedor cerrado posee el vínculo autenticado, motivo y correlación del
// montaje. Solicita concesión nominal nueva; no crea autoridad desde el selector.
type PreparacionConsultaPersonaAceptacionCT struct {
	Solicitud        SolicitudConsultaPersonaAceptacionCT
	MaterialCanonico []byte
	Recurso          core.RecursoAutorizable
}
type ProveedorAutorizacionConsultaPersonaAceptacionCT interface {
	AutorizarConsultaPersonaAceptacionCT(context.Context, PreparacionConsultaPersonaAceptacionCT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
type OrdenConsultaPersonaAceptacionCT struct {
	Solicitud SolicitudConsultaPersonaAceptacionCT
	Material  vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type AceptacionPersonaCT struct {
	OperacionRef           string `json:"operacion_ref"`
	ReciboRef              string `json:"recibo_ref"`
	RegistroSHA256         string `json:"registro_sha256"`
	AperturaOperacionRef   string `json:"apertura_operacion_ref"`
	AperturaRegistroSHA256 string `json:"apertura_registro_sha256"`
	LlamamientoRef         string `json:"llamamiento_ref"`
}
type PersonaAceptadaCT struct {
	Ref     string `json:"ref"`
	Version int64  `json:"version"`
}
type VinculoPersonaAceptadaCT struct {
	Ref                string    `json:"ref"`
	Version            int64     `json:"version"`
	ProcedenciaRef     string    `json:"procedencia_ref"`
	ProcedenciaVersion int64     `json:"procedencia_version"`
	ProcedenciaSHA256  string    `json:"procedencia_sha256"`
	Poblacion          string    `json:"poblacion"`
	VigenteHasta       time.Time `json:"vigente_hasta"`
}
type EvidenciaConsultaPersonaAceptacionCT struct {
	DecisionRef         string    `json:"decision_ref"`
	ConsumoHuellaSHA256 string    `json:"consumo_huella_sha256"`
	AuditoriaRef        string    `json:"auditoria_ref"`
	ConsultadaEn        time.Time `json:"consultada_en"`
}
type ResultadoConsultaPersonaAceptacionCT struct {
	Estado     string                               `json:"estado"`
	Aceptacion *AceptacionPersonaCT                 `json:"aceptacion"`
	Persona    *PersonaAceptadaCT                   `json:"persona"`
	Vinculo    *VinculoPersonaAceptadaCT            `json:"vinculo"`
	Evidencia  EvidenciaConsultaPersonaAceptacionCT `json:"evidencia"`
}
type ConsultaPersonaAceptacionCT interface {
	ConsultarPersonaAceptacionCT(context.Context, SolicitudConsultaPersonaAceptacionCT) (ResultadoConsultaPersonaAceptacionCT, error)
}
type RepositorioConsultaPersonaAceptacionCT interface {
	ConsultarPersonaAceptacionCT(context.Context, OrdenConsultaPersonaAceptacionCT) (ResultadoConsultaPersonaAceptacionCT, error)
}
