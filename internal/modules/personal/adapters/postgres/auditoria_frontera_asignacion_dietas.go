package postgres

import (
	"context"
	"errors"
	"reflect"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

var ErrAuditoriaFronteraAsignacionNoDisponible = errors.New("personal postgres: auditoria frontera de asignacion no disponible")

const registrarAuditoriaFronteraAsignacionSQL = `SELECT vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1($1::text,$2::text,$3::text,$4::text,$5::text,NULLIF($6::text,''),NULLIF($7::text,''),$8::smallint)`
const preflightAuditoriaFronteraAsignacionSQL = `SELECT pg_catalog.has_function_privilege(current_user,'vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint)','EXECUTE')`

type consultorAuditoriaFronteraAsignacion interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type RegistradorAuditoriaFronteraAsignacionPostgreSQL struct {
	consultor consultorAuditoriaFronteraAsignacion
}

var _ personalports.RegistradorAuditoriaFronteraAsignacionDietas = (*RegistradorAuditoriaFronteraAsignacionPostgreSQL)(nil)

func NuevoRegistradorAuditoriaFronteraAsignacionPostgreSQL(pool *pgxpool.Pool) (*RegistradorAuditoriaFronteraAsignacionPostgreSQL, error) {
	return nuevoRegistradorAuditoriaFronteraAsignacionPostgreSQL(pool)
}

func nuevoRegistradorAuditoriaFronteraAsignacionPostgreSQL(consultor consultorAuditoriaFronteraAsignacion) (*RegistradorAuditoriaFronteraAsignacionPostgreSQL, error) {
	if nuloAuditoriaAsignacion(consultor) {
		return nil, ErrAuditoriaFronteraAsignacionNoDisponible
	}
	return &RegistradorAuditoriaFronteraAsignacionPostgreSQL{consultor: consultor}, nil
}

func (r *RegistradorAuditoriaFronteraAsignacionPostgreSQL) Preflight(ctx context.Context) error {
	if r == nil || ctx == nil || nuloAuditoriaAsignacion(r.consultor) || ctx.Err() != nil {
		return ErrAuditoriaFronteraAsignacionNoDisponible
	}
	var permitido bool
	if err := r.consultor.QueryRow(ctx, preflightAuditoriaFronteraAsignacionSQL).Scan(&permitido); err != nil || !permitido {
		return ErrAuditoriaFronteraAsignacionNoDisponible
	}
	return nil
}

func (r *RegistradorAuditoriaFronteraAsignacionPostgreSQL) RegistrarAuditoriaFronteraAsignacionDietas(ctx context.Context, orden personalports.OrdenAuditoriaFronteraAsignacionDietas) error {
	if r == nil || ctx == nil || nuloAuditoriaAsignacion(r.consultor) || ctx.Err() != nil {
		return ErrAuditoriaFronteraAsignacionNoDisponible
	}
	if err := orden.Validar(); err != nil {
		return err
	}
	var registrada bool
	if err := r.consultor.QueryRow(ctx, registrarAuditoriaFronteraAsignacionSQL,
		orden.CorrelacionRef, orden.Motivo, personalports.SuperficieFronteraAsignacionDietas,
		orden.Ruta, orden.Accion, orden.ActorRef, orden.RecursoRef, int16(orden.EstadoHTTP),
	).Scan(&registrada); err != nil || !registrada {
		return ErrAuditoriaFronteraAsignacionNoDisponible
	}
	return nil
}

func nuloAuditoriaAsignacion(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	return (x.Kind() == reflect.Ptr || x.Kind() == reflect.Interface) && x.IsNil()
}
