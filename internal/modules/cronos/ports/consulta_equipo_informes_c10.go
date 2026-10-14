package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrConsultaEquipoInvalida = errors.New("cronos: consulta de equipo inválida")

type AlcanceConsultaEquipo string

const (
	AlcanceEquipoJefatura AlcanceConsultaEquipo = "jefatura"
	AlcanceEquipoRRHH     AlcanceConsultaEquipo = "rrhh"
)

type ProveedorConsultaEquipo interface {
	ProveerMaterialConsultaEquipo(context.Context, domain.MaterialConsultaEquipo) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// La frontera confiable fija el alcance; el cliente sólo indica equipo y fechas.
type OrdenConsultaEquipo struct {
	contexto  vecdomain.ContextoActor
	alcance   AlcanceConsultaEquipo
	proveedor ProveedorConsultaEquipo
}

func NuevaOrdenConsultaEquipo(ctx vecdomain.ContextoActor, alcance AlcanceConsultaEquipo, proveedor ProveedorConsultaEquipo) (OrdenConsultaEquipo, error) {
	if ctx.Validar() != nil || dependenciaRemotaNula(proveedor) || (alcance != AlcanceEquipoJefatura && alcance != AlcanceEquipoRRHH) {
		return OrdenConsultaEquipo{}, ErrDependenciaNoDisponible
	}
	copia, err := ctx.Clonar()
	if err != nil {
		return OrdenConsultaEquipo{}, ErrDependenciaNoDisponible
	}
	return OrdenConsultaEquipo{contexto: copia, alcance: alcance, proveedor: proveedor}, nil
}

func (o OrdenConsultaEquipo) ContextoActor() (vecdomain.ContextoActor, error) {
	if o.contexto.Validar() != nil {
		return vecdomain.ContextoActor{}, ErrDependenciaNoDisponible
	}
	return o.contexto.Clonar()
}
func (o OrdenConsultaEquipo) Alcance() AlcanceConsultaEquipo { return o.alcance }
func (o OrdenConsultaEquipo) ProveedorMaterial() ProveedorConsultaEquipo {
	if dependenciaRemotaNula(o.proveedor) {
		return nil
	}
	return o.proveedor
}

type FilaConsultaEquipo struct {
	EmpleadoRef       string `json:"empleado_ref"`
	EmpleadoEtiqueta  string `json:"empleado_etiqueta"`
	Fecha             string `json:"fecha"`
	Estado            string `json:"estado"`
	Causa             string `json:"causa"`
	Cobertura         string `json:"cobertura"`
	HechosRegistrados int64  `json:"hechos_registrados"`
}
type MarcajeEquipoCrudo struct {
	Ref         string           `json:"ref"`
	Version     int64            `json:"version"`
	Movimiento  domain.PunchKind `json:"movimiento"`
	InstanteUTC time.Time        `json:"instante_utc"`
}
type FilaEquipoCruda struct {
	EmpleadoRef           string               `json:"empleado_ref"`
	EmpleadoEtiqueta      string               `json:"empleado_etiqueta"`
	Fecha                 string               `json:"fecha"`
	CoberturaCompleta     bool                 `json:"cobertura_completa"`
	AntecedentesCompletos bool                 `json:"antecedentes_completos"`
	Marcajes              []MarcajeEquipoCrudo `json:"marcajes"`
}
type AgregadoConsultaEquipo struct {
	Total         int64 `json:"total"`
	Indeterminado int64 `json:"indeterminado"`
	Registrado    int64 `json:"registrado"`
	Incidencias   int64 `json:"incidencias"`
}
type ResultadoConsultaEquipo struct {
	EquipoRef       string                    `json:"equipo_ref"`
	Alcance         AlcanceConsultaEquipo     `json:"alcance"`
	Desde           string                    `json:"desde"`
	Hasta           string                    `json:"hasta"`
	Vista           domain.TipoConsultaEquipo `json:"vista"`
	VersionRef      string                    `json:"version_ref"`
	Filas           []FilaConsultaEquipo      `json:"filas"`
	Agregado        AgregadoConsultaEquipo    `json:"agregado"`
	SiguienteCursor string                    `json:"siguiente_cursor"`
}

type RepositorioConsultaEquipo interface {
	FotoEquipo(context.Context, string) (domain.FotoEquipoAutoritativa, error)
	ConsultarEquipo(context.Context, OrdenConsultaEquipo, domain.MaterialConsultaEquipo) (FuenteConsultaEquipo, error)
}
type FuenteConsultaEquipo struct {
	EquipoRef       string                    `json:"equipo_ref"`
	Alcance         AlcanceConsultaEquipo     `json:"alcance"`
	Desde           string                    `json:"desde"`
	Hasta           string                    `json:"hasta"`
	Vista           domain.TipoConsultaEquipo `json:"vista"`
	VersionRef      string                    `json:"version_ref"`
	ZonaHoraria     string                    `json:"zona_horaria"`
	CorteUTC        time.Time                 `json:"corte_utc"`
	Filas           []FilaEquipoCruda         `json:"filas"`
	SiguienteCursor string                    `json:"siguiente_cursor"`
}
type CasoUsoConsultaEquipo interface {
	ConsultarEquipo(context.Context, OrdenConsultaEquipo, domain.TipoConsultaEquipo, string, string, string, string) (ResultadoConsultaEquipo, error)
}
