package ports

import (
	"context"
	"errors"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionConsultaAnclajeAceptacionNativaCT    = AccionConsultaAnclajeAceptacionCT
	AudienciaConsultaAnclajeAceptacionNativaCT = AudienciaConsultaAnclajeAceptacionCT
	AccionConsultaPersonaAceptacionNativaCT    = AccionConsultaPersonaAceptacionCT
	AudienciaConsultaPersonaAceptacionNativaCT = AudienciaConsultaPersonaAceptacionCT
)

var (
	ErrAceptacionNativaCTInvalida     = errors.New("bolsa: selector de aceptación nativa inválido")
	ErrAceptacionNativaCTDenegada     = errors.New("bolsa: consulta de aceptación nativa denegada")
	ErrAceptacionNativaCTNoDisponible = errors.New("bolsa: aceptación nativa no disponible")
)

// El selector incluye sólo referencias públicas del hecho ya emitido y su
// respuesta. Bolsa deriva la participación, el candidato y el orden de B110.
type SelectorAceptacionNativaCT struct {
	OrganizacionRef    string `json:"organizacion_ref"`
	ExpedienteRef      string `json:"expediente_ref"`
	ClaveCT            string `json:"clave_ct"`
	UnidadRef          string `json:"unidad_ref"`
	CategoriaRef       string `json:"categoria_ref"`
	NecesidadRef       string `json:"necesidad_ref"`
	BolsaRef           string `json:"bolsa_ref"`
	LlamamientoRef     string `json:"llamamiento_ref"`
	ReciboEmisionRef   string `json:"recibo_emision_ref"`
	ReciboRespuestaRef string `json:"recibo_respuesta_ref"`
}

type SolicitudConsultaAceptacionNativaCT struct {
	Selector       SelectorAceptacionNativaCT
	ActorConfiable ActorConfiablePersonaAceptacionCT
}

type PreparacionConsultaAceptacionNativaCT struct {
	Solicitud        SolicitudConsultaAceptacionNativaCT
	MaterialCanonico []byte
	Recurso          core.RecursoAutorizable
}

type OrdenConsultaAceptacionNativaCT struct {
	Solicitud SolicitudConsultaAceptacionNativaCT
	Material  vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ProveedorAutorizacionConsultaAceptacionNativaCT interface {
	AutorizarConsultaAceptacionNativaCT(context.Context, PreparacionConsultaAceptacionNativaCT, bool) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type AnclajeAceptacionNativaCT struct {
	SelectorAceptacionNativaCT
	VersionExpediente       uint64    `json:"version_expediente"`
	ActaRef                 string    `json:"acta_ref"`
	VersionBolsa            uint64    `json:"version_bolsa"`
	HuellaBolsaSHA256       string    `json:"huella_bolsa_sha256"`
	InstantaneaRef          string    `json:"instantanea_ref"`
	VersionInstantanea      uint64    `json:"version_instantanea"`
	HuellaInstantaneaSHA256 string    `json:"huella_instantanea_sha256"`
	PoliticaRef             string    `json:"politica_ref"`
	VersionPolitica         uint64    `json:"version_politica"`
	HuellaPoliticaSHA256    string    `json:"huella_politica_sha256"`
	ParticipacionRef        string    `json:"participacion_ref"`
	CandidatoRef            string    `json:"candidato_ref"`
	OrdenSeleccionado       uint64    `json:"orden_seleccionado"`
	RespuestaRef            string    `json:"respuesta_ref"`
	RespondidaEn            time.Time `json:"respondida_en"`
}

type ResultadoAnclajeAceptacionNativaCT struct {
	Estado    string                               `json:"estado"`
	Anclaje   *AnclajeAceptacionNativaCT           `json:"anclaje"`
	Evidencia EvidenciaConsultaPersonaAceptacionCT `json:"evidencia"`
}

type ResultadoPersonaAceptacionNativaCT struct {
	Estado     string                               `json:"estado"`
	Aceptacion *AnclajeAceptacionNativaCT           `json:"aceptacion"`
	Persona    *PersonaAceptadaCT                   `json:"persona"`
	Vinculo    *VinculoPersonaAceptadaCT            `json:"vinculo"`
	Evidencia  EvidenciaConsultaPersonaAceptacionCT `json:"evidencia"`
}

type ConsultaAceptacionNativaCT interface {
	ConsultarAnclajeAceptacionNativaCT(context.Context, SolicitudConsultaAceptacionNativaCT) (ResultadoAnclajeAceptacionNativaCT, error)
	ConsultarPersonaAceptacionNativaCT(context.Context, SolicitudConsultaAceptacionNativaCT) (ResultadoPersonaAceptacionNativaCT, error)
}

type RepositorioConsultaAceptacionNativaCT interface {
	ConsultarAnclajeAceptacionNativaCT(context.Context, OrdenConsultaAceptacionNativaCT) (ResultadoAnclajeAceptacionNativaCT, error)
	ConsultarPersonaAceptacionNativaCT(context.Context, OrdenConsultaAceptacionNativaCT) (ResultadoPersonaAceptacionNativaCT, error)
}
