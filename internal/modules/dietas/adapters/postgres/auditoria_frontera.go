package postgres

import (
	"context"
	"errors"
	"reflect"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/dietas/ports"
)

var ErrAuditoriaFronteraComisionNoDisponible = errors.New("dietas postgres: auditoria de frontera no disponible")

const consultaRegistrarAuditoriaFronteraComision = `SELECT vec_dietas.registrar_auditoria_frontera_comision_v2($1::text,$2::text,$3::text,$4::text,$5::text,NULLIF($6::text,''))`
const consultaPreflightAuditoriaFronteraComision = `SELECT pg_catalog.has_function_privilege(current_user,'vec_dietas.registrar_auditoria_frontera_comision_v2(text,text,text,text,text,text)','EXECUTE')`

type consultorAuditoriaFronteraComision interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type RegistradorAuditoriaFronteraComisionPostgreSQL struct {
	consultor consultorAuditoriaFronteraComision
}

var _ ports.RegistradorAuditoriaFronteraComision = (*RegistradorAuditoriaFronteraComisionPostgreSQL)(nil)

func NuevoRegistradorAuditoriaFronteraComisionPostgreSQL(pool *pgxpool.Pool) (*RegistradorAuditoriaFronteraComisionPostgreSQL, error) {
	if nuloAuditoriaFronteraComision(pool) {
		return nil, ErrAuditoriaFronteraComisionNoDisponible
	}
	return &RegistradorAuditoriaFronteraComisionPostgreSQL{consultor: pool}, nil
}

func nuloAuditoriaFronteraComision(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	return (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) && v.IsNil()
}

func (r *RegistradorAuditoriaFronteraComisionPostgreSQL) Preflight(ctx context.Context) error {
	if ctx == nil || r == nil || nuloAuditoriaFronteraComision(r.consultor) || ctx.Err() != nil {
		return ErrAuditoriaFronteraComisionNoDisponible
	}
	var permitido bool
	if err := r.consultor.QueryRow(ctx, consultaPreflightAuditoriaFronteraComision).Scan(&permitido); err != nil || !permitido {
		return ErrAuditoriaFronteraComisionNoDisponible
	}
	return nil
}

func (r *RegistradorAuditoriaFronteraComisionPostgreSQL) RegistrarAuditoriaFronteraComision(ctx context.Context, orden ports.OrdenAuditoriaFronteraComision) error {
	if ctx == nil || r == nil || nuloAuditoriaFronteraComision(r.consultor) || ctx.Err() != nil {
		return ErrAuditoriaFronteraComisionNoDisponible
	}
	if err := orden.Validar(); err != nil {
		return err
	}
	var registrada bool
	if err := r.consultor.QueryRow(ctx, consultaRegistrarAuditoriaFronteraComision,
		orden.CorrelacionRef, orden.Motivo, ports.SuperficieAuditoriaFronteraComision, orden.Ruta, orden.Accion, orden.ActorRef,
	).Scan(&registrada); err != nil || !registrada {
		return ErrAuditoriaFronteraComisionNoDisponible
	}
	return nil
}
