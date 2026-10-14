package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const consultaEquipoC10 = `SELECT vec_cronos_v1.consultar_equipo_periodo_c10_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
const fotoEquipoC10 = `SELECT vec_cronos_v1.foto_equipo_c10_v1($1)`

type RepositorioEquipoInformesC10 struct{ db *pgxpool.Pool }

func NuevoRepositorioEquipoInformesC10(pool *pgxpool.Pool) (*RepositorioEquipoInformesC10, error) {
	if pool == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &RepositorioEquipoInformesC10{db: pool}, nil
}

func (r *RepositorioEquipoInformesC10) FotoEquipo(ctx context.Context, equipo string) (domain.FotoEquipoAutoritativa, error) {
	if r == nil || r.db == nil || ctx == nil || ctx.Err() != nil || equipo == "" || len(equipo) > 128 {
		return domain.FotoEquipoAutoritativa{}, ports.ErrDependenciaNoDisponible
	}
	var bruto []byte
	if err := r.db.QueryRow(ctx, fotoEquipoC10, equipo).Scan(&bruto); err != nil {
		return domain.FotoEquipoAutoritativa{}, errorSeguro(ctx, err)
	}
	defer clear(bruto)
	if len(bruto) == 0 || len(bruto) > 4096 {
		return domain.FotoEquipoAutoritativa{}, ports.ErrDependenciaNoDisponible
	}
	var foto struct {
		EquipoRef       string `json:"equipo_ref"`
		UnidadRef       string `json:"unidad_ref"`
		OrganismoRef    string `json:"organismo_ref"`
		VersionRef      string `json:"version_ref"`
		ReciboFuenteRef string `json:"recibo_fuente_ref"`
		FuenteRef       string `json:"fuente_ref"`
		Desde           string `json:"desde"`
		HastaExclusivo  string `json:"hasta_exclusivo"`
	}
	if decodificarEstricto(bruto, &foto) != nil || foto.EquipoRef != equipo || foto.UnidadRef == "" || foto.OrganismoRef == "" || foto.VersionRef == "" || foto.ReciboFuenteRef == "" || foto.FuenteRef == "" || foto.Desde == "" || foto.HastaExclusivo == "" {
		return domain.FotoEquipoAutoritativa{}, ports.ErrDependenciaNoDisponible
	}
	return domain.FotoEquipoAutoritativa{EquipoRef: foto.EquipoRef, UnidadRef: foto.UnidadRef, OrganismoRef: foto.OrganismoRef, VersionRef: foto.VersionRef, ReciboFuenteRef: foto.ReciboFuenteRef, FuenteRef: foto.FuenteRef, Desde: foto.Desde, HastaExclusivo: foto.HastaExclusivo}, nil
}

func (r *RepositorioEquipoInformesC10) ConsultarEquipo(ctx context.Context, o ports.OrdenConsultaEquipo, m domain.MaterialConsultaEquipo) (ports.FuenteConsultaEquipo, error) {
	if r == nil || r.db == nil || ctx == nil || ctx.Err() != nil || o.ProveedorMaterial() == nil || o.Alcance() != ports.AlcanceConsultaEquipo(m.Alcance) {
		return ports.FuenteConsultaEquipo{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := o.ContextoActor()
	if err != nil {
		return ports.FuenteConsultaEquipo{}, ports.ErrDependenciaNoDisponible
	}
	empleados, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 || actor.PersonaRef != m.ActorRef || actor.PerfilActivoRef != m.PerfilRef || empleados[0] != m.EmpleadoRef {
		return ports.FuenteConsultaEquipo{}, ports.ErrDependenciaNoDisponible
	}
	canon, err := m.Canonico()
	if err != nil {
		return ports.FuenteConsultaEquipo{}, ports.ErrConsultaEquipoInvalida
	}
	recurso, err := m.Recurso()
	if err != nil {
		return ports.FuenteConsultaEquipo{}, ports.ErrDependenciaNoDisponible
	}
	accion, _, audiencia, ok := application.ContratoConsultaEquipo(m.Tipo, o.Alcance())
	if !ok {
		return ports.FuenteConsultaEquipo{}, ports.ErrDependenciaNoDisponible
	}
	v3, err := o.ProveedorMaterial().ProveerMaterialConsultaEquipo(ctx, m)
	if err != nil {
		return ports.FuenteConsultaEquipo{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, audiencia, accion, recurso) {
		return ports.FuenteConsultaEquipo{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarLecturaV3(ctx, r.db, consultaEquipoC10, canon, v3)
	if err != nil {
		return ports.FuenteConsultaEquipo{}, err
	}
	defer clear(bruto)
	if len(bruto) == 0 || len(bruto) > 128*1024 {
		return ports.FuenteConsultaEquipo{}, ports.ErrDependenciaNoDisponible
	}
	var fuente ports.FuenteConsultaEquipo
	if decodificarEstricto(bruto, &fuente) != nil || fuente.Filas == nil || fuente.EquipoRef != m.Foto.EquipoRef || fuente.Alcance != o.Alcance() || fuente.Desde != m.Desde || fuente.Hasta != m.Hasta || fuente.Vista != m.Tipo || fuente.VersionRef != m.Foto.VersionRef {
		return ports.FuenteConsultaEquipo{}, ports.ErrDependenciaNoDisponible
	}
	return fuente, nil
}

var _ ports.RepositorioConsultaEquipo = (*RepositorioEquipoInformesC10)(nil)
