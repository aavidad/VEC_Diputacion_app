package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/ports"
)

const consultaRegistrarDenegacionFronteraIdentidadV1 = `
	SELECT vec_auditoria_frontera_v1.registrar_denegacion_frontera_identidad_v1(
		$1::text, $2::text, $3::text, $4::text, $5::text,
		NULLIF($6::text, ''), NULLIF($7::text, '')
	)`

const consultaPreflightDenegacionFronteraIdentidadV1 = `
	SELECT pg_catalog.has_function_privilege(
		current_user,
		'vec_auditoria_frontera_v1.registrar_denegacion_frontera_identidad_v1(text,text,text,text,text,text,text)',
		'EXECUTE'
	)`

type consultorFilaDenegacionFronteraIdentidadV1 interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// RegistradorDenegacionFronteraIdentidadV1PostgreSQL persiste exclusivamente
// las referencias minimizadas del puerto. No acepta material de sesión,
// certificados, cabeceras, IP, cuerpo ni errores de infraestructura.
type RegistradorDenegacionFronteraIdentidadV1PostgreSQL struct {
	consultor consultorFilaDenegacionFronteraIdentidadV1
}

var _ ports.RegistradorDenegacionFronteraIdentidadV1 = (*RegistradorDenegacionFronteraIdentidadV1PostgreSQL)(nil)

func NuevoRegistradorDenegacionFronteraIdentidadV1PostgreSQL(
	pool *pgxpool.Pool,
) (*RegistradorDenegacionFronteraIdentidadV1PostgreSQL, error) {
	if valorNuloPostgreSQL(pool) {
		return nil, ports.ErrRegistradorDenegacionFronteraIdentidadV1NoDisponible
	}
	return nuevoRegistradorDenegacionFronteraIdentidadV1PostgreSQL(pool)
}

func nuevoRegistradorDenegacionFronteraIdentidadV1PostgreSQL(
	consultor consultorFilaDenegacionFronteraIdentidadV1,
) (*RegistradorDenegacionFronteraIdentidadV1PostgreSQL, error) {
	if valorNuloPostgreSQL(consultor) {
		return nil, ports.ErrRegistradorDenegacionFronteraIdentidadV1NoDisponible
	}
	return &RegistradorDenegacionFronteraIdentidadV1PostgreSQL{consultor: consultor}, nil
}

func (r *RegistradorDenegacionFronteraIdentidadV1PostgreSQL) PreflightDenegacionFronteraIdentidadV1(
	ctx context.Context,
) error {
	if ctx == nil || r == nil || valorNuloPostgreSQL(r.consultor) {
		return ports.ErrRegistradorDenegacionFronteraIdentidadV1NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var permitido bool
	if err := r.consultor.QueryRow(ctx, consultaPreflightDenegacionFronteraIdentidadV1).Scan(&permitido); err != nil || !permitido {
		return ports.ErrRegistradorDenegacionFronteraIdentidadV1NoDisponible
	}
	return nil
}

func (r *RegistradorDenegacionFronteraIdentidadV1PostgreSQL) RegistrarDenegacionFronteraIdentidadV1(
	ctx context.Context,
	orden ports.OrdenDenegacionFronteraIdentidadV1,
) error {
	if ctx == nil || r == nil || valorNuloPostgreSQL(r.consultor) {
		return ports.ErrRegistradorDenegacionFronteraIdentidadV1NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := orden.Validar(); err != nil {
		return err
	}
	var registrada bool
	err := r.consultor.QueryRow(ctx, consultaRegistrarDenegacionFronteraIdentidadV1,
		orden.CorrelacionRef, orden.Superficie, orden.RutaExacta, orden.Accion,
		orden.Motivo, orden.CanalRef, orden.ActorRef,
	).Scan(&registrada)
	if err != nil || !registrada {
		return ports.ErrRegistradorDenegacionFronteraIdentidadV1NoDisponible
	}
	return nil
}
