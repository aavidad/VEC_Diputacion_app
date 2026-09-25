package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

const registrarAuditoriaFronteraRectificacionSQL = `SELECT vec_personal.registrar_auditoria_frontera_rectificacion_dietas_v1($1::text,$2::text,$3::text,$4::text,$5::text,NULLIF($6::text,''),NULLIF($7::text,''),$8::integer)`
const preflightAuditoriaFronteraRectificacionSQL = `SELECT pg_catalog.has_function_privilege(current_user,'vec_personal.registrar_auditoria_frontera_rectificacion_dietas_v1(text,text,text,text,text,text,text,integer)','EXECUTE')`

type consultorAuditoriaRectificacion interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type RegistradorAuditoriaFronteraRectificacionPostgreSQL struct {
	consultor consultorAuditoriaRectificacion
}

var _ personalports.RegistradorAuditoriaFronteraRectificacionDietas = (*RegistradorAuditoriaFronteraRectificacionPostgreSQL)(nil)

func NuevoRegistradorAuditoriaFronteraRectificacionPostgreSQL(pool *pgxpool.Pool) (*RegistradorAuditoriaFronteraRectificacionPostgreSQL, error) {
	if nuloRelacion(pool) {
		return nil, personalports.ErrRectificacionDietasNoDisponible
	}
	return &RegistradorAuditoriaFronteraRectificacionPostgreSQL{consultor: pool}, nil
}

func (r *RegistradorAuditoriaFronteraRectificacionPostgreSQL) Preflight(ctx context.Context) error {
	if r == nil || ctx == nil || nuloRelacion(r.consultor) || ctx.Err() != nil {
		return personalports.ErrRectificacionDietasNoDisponible
	}
	var permitido bool
	if err := r.consultor.QueryRow(ctx, preflightAuditoriaFronteraRectificacionSQL).Scan(&permitido); err != nil || !permitido {
		return personalports.ErrRectificacionDietasNoDisponible
	}
	return nil
}

func (r *RegistradorAuditoriaFronteraRectificacionPostgreSQL) RegistrarAuditoriaFronteraRectificacionDietas(ctx context.Context, o personalports.OrdenAuditoriaFronteraRectificacionDietas) error {
	if r == nil || ctx == nil || nuloRelacion(r.consultor) || ctx.Err() != nil || o.Validar() != nil {
		return personalports.ErrRectificacionDietasNoDisponible
	}
	var registrada bool
	if err := r.consultor.QueryRow(ctx, registrarAuditoriaFronteraRectificacionSQL,
		o.CorrelacionRef, o.Motivo, "api.personal.solicitudes_rectificacion_dietas", o.Ruta, o.Accion, o.ActorRef, o.RecursoRef, o.EstadoHTTP,
	).Scan(&registrada); err != nil || !registrada {
		return personalports.ErrRectificacionDietasNoDisponible
	}
	return nil
}
