package ports

import (
	"context"
	"errors"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

var (
	ErrMaterialBasesInvalido       = errors.New("seleccion.preparacion.material_invalido")
	ErrPreparadorBasesNoDisponible = errors.New("seleccion.preparacion.no_disponible")
)

// Las referencias aportadas son propuestas, no evidencias resueltas o aprobadas.
type ReferenciasPreparacionBases struct {
	FuenteBases      bolsa.ReferenciaConfiguracionConvocatoria `json:"fuente_bases"`
	Catalogos        bolsa.ReferenciaConfiguracionConvocatoria `json:"catalogos"`
	Calendario       bolsa.ReferenciaConfiguracionConvocatoria `json:"calendario"`
	ReglasBaremacion bolsa.ReferenciaConfiguracionConvocatoria `json:"reglas_baremacion"`
	FlujoProceso     bolsa.ReferenciaConfiguracionConvocatoria `json:"flujo_proceso"`
	FlujoSolicitud   bolsa.ReferenciaConfiguracionConvocatoria `json:"flujo_solicitud"`
	Plantilla        bolsa.ReferenciaConfiguracionConvocatoria `json:"plantilla"`
	Plaza            bolsa.ReferenciaConfiguracionConvocatoria `json:"plaza"`
	OEP              bolsa.ReferenciaConfiguracionConvocatoria `json:"oep"`
	RPT              bolsa.ReferenciaConfiguracionConvocatoria `json:"rpt"`
}

// VersionMaterial identifica una revisión local; no es la versión gobernada de Bolsa.
type MaterialBasesPropuesto struct {
	Alcance           string                                `json:"alcance"`
	IdentidadMaterial string                                `json:"identidad_material"`
	VersionMaterial   int                                   `json:"version_material"`
	Contenido         bolsa.ContenidoPublicableConvocatoria `json:"contenido"`
	Referencias       ReferenciasPreparacionBases           `json:"referencias"`
}

type PendientePreparacionBases struct {
	Campo  string `json:"campo"`
	Codigo string `json:"codigo"`
}

type PreparacionBases struct {
	Estado                 string                                 `json:"estado"`
	MaterialPropuesto      MaterialBasesPropuesto                 `json:"material_propuesto"`
	ContenidoCanonicoBolsa *bolsa.ContenidoPublicableConvocatoria `json:"contenido_canonico_bolsa,omitempty"`
	Pendientes             []PendientePreparacionBases            `json:"pendientes"`
}

// CanonizadorBases reutiliza las comprobaciones estructurales del dueño Bolsa.
// No resuelve las referencias ni consume autorización, firma, custodia o gobierno.
type CanonizadorBases interface {
	EvaluarMaterialBases(context.Context, MaterialBasesPropuesto) (EvaluacionMaterialBases, error)
	ComprobarReferencia(bolsa.ReferenciaConfiguracionConvocatoria) error
	CanonizarContenido(context.Context, bolsa.ContenidoPublicableConvocatoria) (bolsa.ContenidoPublicableConvocatoria, error)
}

type EvaluacionMaterialBases struct {
	ContenidoCanonicoBolsa *bolsa.ContenidoPublicableConvocatoria
	Pendientes             []PendientePreparacionBases
}
