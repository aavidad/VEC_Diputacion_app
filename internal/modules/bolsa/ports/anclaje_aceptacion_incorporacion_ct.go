package ports

import (
	"context"

	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionConsultaAnclajeAceptacionCT    = "bolsa.aceptacion_ct.anclaje.consultar"
	AudienciaConsultaAnclajeAceptacionCT = "vec_bolsa_llamamientos.aceptacion_ct.anclaje.v1"
	FinalidadConsultaAnclajeAceptacionCT = "preparar_incorporacion_ct"
	TipoRecursoAnclajeAceptacionCT       = "anclaje_aceptacion_ct"
)

// Las clases de fallo siguen siendo las de la lectura nominal propietaria.
var (
	ErrConsultaAnclajeAceptacionCTInvalida     = ErrConsultaPersonaAceptacionCTInvalida
	ErrConsultaAnclajeAceptacionCTDenegada     = ErrConsultaPersonaAceptacionCTDenegada
	ErrConsultaAnclajeAceptacionCTNoDisponible = ErrConsultaPersonaAceptacionCTNoDisponible
)

// SelectorAnclajeAceptacionCT procede de CT confiable. El solicitante no dispone
// de la huella completa de apertura; sólo Bolsa puede derivarla y acreditarla.
type SelectorAnclajeAceptacionCT struct {
	UnidadRef                string `json:"unidad_ref"`
	CategoriaRef             string `json:"categoria_ref"`
	NecesidadRef             string `json:"necesidad_ref"`
	AceptacionOperacionRef   string `json:"aceptacion_operacion_ref"`
	AceptacionRegistroSHA256 string `json:"aceptacion_registro_sha256"`
	AperturaOperacionRef     string `json:"apertura_operacion_ref"`
	LlamamientoRef           string `json:"llamamiento_ref"`
	PropuestaRef             string `json:"propuesta_ref"`
}
type SolicitudConsultaAnclajeAceptacionCT struct {
	Selector       SelectorAnclajeAceptacionCT
	ActorConfiable ActorConfiablePersonaAceptacionCT
}
type PreparacionConsultaAnclajeAceptacionCT struct {
	Solicitud        SolicitudConsultaAnclajeAceptacionCT
	MaterialCanonico []byte
	Recurso          core.RecursoAutorizable
}
type ProveedorAutorizacionConsultaAnclajeAceptacionCT interface {
	AutorizarConsultaAnclajeAceptacionCT(context.Context, PreparacionConsultaAnclajeAceptacionCT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
type OrdenConsultaAnclajeAceptacionCT struct {
	Solicitud SolicitudConsultaAnclajeAceptacionCT
	Material  vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// El recibo de aceptación es ORIGINAL; la evidencia representa únicamente el
// acceso actual. El selector completo habilita la posterior consulta B67.
type AnclajeAceptacionIncorporacionCT struct {
	SelectorPersonaAceptacionCT
	AceptacionReciboRef string `json:"aceptacion_recibo_ref"`
}
type ResultadoConsultaAnclajeAceptacionCT struct {
	Estado    string                               `json:"estado"`
	Anclaje   *AnclajeAceptacionIncorporacionCT    `json:"anclaje"`
	Evidencia EvidenciaConsultaPersonaAceptacionCT `json:"evidencia"`
}
type ConsultaAnclajeAceptacionCT interface {
	ConsultarAnclajeAceptacionCT(context.Context, SolicitudConsultaAnclajeAceptacionCT) (ResultadoConsultaAnclajeAceptacionCT, error)
}
type RepositorioConsultaAnclajeAceptacionCT interface {
	ConsultarAnclajeAceptacionCT(context.Context, OrdenConsultaAnclajeAceptacionCT) (ResultadoConsultaAnclajeAceptacionCT, error)
}
