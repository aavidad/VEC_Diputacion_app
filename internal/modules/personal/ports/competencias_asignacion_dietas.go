package ports

import (
	"context"
	"errors"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionConsultarCompetenciasAsignacionDietas    = "personal.asignacion_dietas.competencias_consultar"
	AudienciaConsultarCompetenciasAsignacionDietas = "vec_personal.asignacion_dietas.competencias.v1"
	FinalidadConsultarCompetenciasAsignacionDietas = "tramitar_dietas_asignadas"
)

var (
	ErrConsultaCompetenciasAsignacionDietasInvalida = errors.New("personal: consulta de competencias invalida")
	ErrCompetenciasAsignacionDietasNoDisponibles    = errors.New("personal: competencias de asignacion no disponibles")
	ErrCompetenciasAsignacionDietasDenegadas        = errors.New("personal: competencias de asignacion denegadas")
)

type ProveedorAutorizacionCompetenciasAsignacionDietas interface {
	AutorizarCompetenciasAsignacionDietas(context.Context, personaldomain.MaterialCompetenciasAsignacionDietas) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type OrdenConsultaCompetenciasAsignacionDietas struct {
	Material     personaldomain.MaterialCompetenciasAsignacionDietas
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type EvidenciaConsultaCompetenciasAsignacionDietas struct {
	ReciboRef           string    `json:"recibo_ref"`
	DecisionRef         string    `json:"decision_ref"`
	EfectoRef           string    `json:"efecto_ref"`
	ConsumoHuellaSHA256 string    `json:"consumo_huella_sha256"`
	AuditoriaRef        string    `json:"auditoria_ref"`
	ConsultadaEn        time.Time `json:"consultada_en"`
}

type ResultadoConsultaCompetenciasAsignacionDietas struct {
	Competencias []personaldomain.CompetenciaAsignacionDietas  `json:"competencias"`
	Evidencia    EvidenciaConsultaCompetenciasAsignacionDietas `json:"evidencia"`
}

type RepositorioCompetenciasAsignacionDietas interface {
	ConsultarCompetenciasAsignacionDietas(context.Context, OrdenConsultaCompetenciasAsignacionDietas) (ResultadoConsultaCompetenciasAsignacionDietas, error)
}
