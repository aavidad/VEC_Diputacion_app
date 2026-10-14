package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionUnidadTrabajoPropiaCronos    = "personal.unidad_trabajo.propia.consultar"
	AudienciaUnidadTrabajoPropiaCronos = "vec_personal.unidad_trabajo.propia.cronos.v1"
)

// ConsultaUnidadTrabajoPropiaCronos nace del contexto de identidad del
// servidor. Ni empleado ni unidad proceden de la petición del navegador.
type ConsultaUnidadTrabajoPropiaCronos struct {
	Actor      vecdomain.ContextoActor
	Fecha      domain.FechaCivil
	ConocidoEn time.Time
}

// UnidadTrabajoCronos conserva únicamente la unidad de la relación de empleo.
// Cronos resuelve el centro en Organización/Personal, nunca lo infiere aquí.
type UnidadTrabajoCronos struct {
	EmpleadoRef      string            `json:"empleado_ref"`
	RelacionRef      string            `json:"relacion_ref"`
	OrganismoRef     string            `json:"organismo_ref"`
	UnidadClave      string            `json:"unidad_clave"`
	RegimenRef       string            `json:"regimen_ref"`
	Desde            domain.FechaCivil `json:"desde"`
	HastaExclusivo   domain.FechaCivil `json:"hasta_exclusivo"`
	FuenteRef        string            `json:"fuente_ref"`
	FuenteVersionRef string            `json:"fuente_version_ref"`
	Version          int64             `json:"version"`
}

type ResultadoUnidadTrabajoPropiaCronos struct {
	Unidad    *UnidadTrabajoCronos        `json:"unidad"`
	Cobertura string                      `json:"cobertura"` // completa, sin_datos o no_acreditada
	Evidencia EvidenciaRegistroEmpleadoB2 `json:"evidencia"`
}

type ProveedorAutorizacionUnidadTrabajoCronos interface {
	AutorizarUnidadTrabajoPropiaCronos(context.Context, domain.MaterialUnidadTrabajoCronos) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
type OrdenUnidadTrabajoCronos struct {
	Material     domain.MaterialUnidadTrabajoCronos
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}
type RepositorioUnidadTrabajoCronos interface {
	ConsultarUnidadTrabajoPropiaCronos(context.Context, OrdenUnidadTrabajoCronos) (ResultadoUnidadTrabajoPropiaCronos, error)
}

// La implementación de Personal obtiene una decisión V3 positiva para esta
// audiencia y consume esa decisión, la lectura y la auditoría en una
// transacción. La ausencia o ambigüedad nunca elige un centro arbitrario.
type LectorUnidadTrabajoPropiaCronos interface {
	ConsultarUnidadTrabajoPropiaCronos(context.Context, ConsultaUnidadTrabajoPropiaCronos) (ResultadoUnidadTrabajoPropiaCronos, error)
}
