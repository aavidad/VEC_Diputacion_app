package ports

import (
	"context"
	"errors"

	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionIdentidadDocumentoAceptacionCT    = "bolsa.aceptacion_ct.identidad_documento.consultar"
	AudienciaIdentidadDocumentoAceptacionCT = "vec_bolsa_llamamientos.aceptacion_ct.identidad_documento.v1"
	FinalidadIdentidadDocumentoAceptacionCT = "redactar_documento_ct"
	TipoIdentidadDocumentoAceptacionCT      = "identidad_documento_aceptacion_ct"
)

var (
	ErrIdentidadDocumentoAceptacionCTInvalida     = errors.New("bolsa: consulta documental inválida")
	ErrIdentidadDocumentoAceptacionCTDenegada     = errors.New("bolsa: consulta documental denegada")
	ErrIdentidadDocumentoAceptacionCTNoDisponible = errors.New("bolsa: identidad documental no disponible")
)

// El selector y el actor los construye la preparación CT confiable. Este
// contrato no concede el permiso de incorporación ni admite datos del navegador.
type SolicitudIdentidadDocumentoAceptacionCT struct {
	Selector       SelectorAceptacionNativaCT
	ActorConfiable ActorConfiablePersonaAceptacionCT
}

type PreparacionIdentidadDocumentoAceptacionCT struct {
	Solicitud        SolicitudIdentidadDocumentoAceptacionCT
	MaterialCanonico []byte
	Recurso          core.RecursoAutorizable
}

type OrdenIdentidadDocumentoAceptacionCT struct {
	Solicitud SolicitudIdentidadDocumentoAceptacionCT
	Material  vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ProveedorAutorizacionIdentidadDocumentoAceptacionCT interface {
	AutorizarIdentidadDocumentoAceptacionCT(context.Context, PreparacionIdentidadDocumentoAceptacionCT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// Fuente técnica acreditada por Bolsa SQL. Nunca contiene identidad clara.
type FuenteIdentidadDocumentoAceptacionCT struct {
	ActaRef                 string `json:"acta_ref"`
	FilaNumero              int    `json:"fila_numero"`
	InstantaneaRef          string `json:"instantanea_ref"`
	VersionInstantanea      uint64 `json:"version_instantanea"`
	HuellaInstantaneaSHA256 string `json:"huella_instantanea_sha256"`
	BolsaRef                string `json:"bolsa_ref"`
	HuellaListadoSHA256     string `json:"huella_listado_sha256"`
	ReciboEmisionRef        string `json:"recibo_emision_ref"`
	ReciboRespuestaRef      string `json:"recibo_respuesta_ref"`
}

type LecturaFuenteIdentidadDocumentoAceptacionCT struct {
	Estado    string                                `json:"estado"`
	Fuente    *FuenteIdentidadDocumentoAceptacionCT `json:"fuente"`
	Evidencia EvidenciaConsultaPersonaAceptacionCT  `json:"evidencia"`
}

type RepositorioIdentidadDocumentoAceptacionCT interface {
	ConsultarFuenteIdentidadDocumentoAceptacionCT(context.Context, OrdenIdentidadDocumentoAceptacionCT) (LecturaFuenteIdentidadDocumentoAceptacionCT, error)
}

type IdentidadDocumentoAceptacionCT struct {
	Nombre               string `json:"nombre"`
	Apellidos            string `json:"apellidos"`
	DocumentoEnmascarado string `json:"documento_enmascarado"`
	VersionFuente        uint64 `json:"version_fuente"`
}

type ResultadoIdentidadDocumentoAceptacionCT struct {
	Estado    string                               `json:"estado"`
	Identidad *IdentidadDocumentoAceptacionCT      `json:"identidad"`
	Evidencia EvidenciaConsultaPersonaAceptacionCT `json:"-"`
}

type ConsultaIdentidadDocumentoAceptacionCT interface {
	ConsultarIdentidadDocumentoAceptacionCT(context.Context, SolicitudIdentidadDocumentoAceptacionCT) (ResultadoIdentidadDocumentoAceptacionCT, error)
}
