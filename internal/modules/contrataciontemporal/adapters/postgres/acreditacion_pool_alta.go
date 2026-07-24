package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type consultorAcreditacionPoolAlta interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

const totalComprobacionesAcreditacionPoolAlta = 20

// AcreditarPoolAltasPostgreSQL verifica en la conexión real que la identidad
// de ejecución solo posee las dos funciones O2-06 necesarias. No acepta un
// indicador de configuración como sustituto de la comprobación nominal.
func AcreditarPoolAltasPostgreSQL(
	ctx context.Context,
	pool *pgxpool.Pool,
) error {
	return acreditarPoolAltasPostgreSQL(ctx, pool)
}

func acreditarPoolAltasPostgreSQL(
	ctx context.Context,
	consultor consultorAcreditacionPoolAlta,
) error {
	if ctx == nil || dependenciaNula(consultor) {
		return ports.ErrPersistenciaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var comprobaciones [totalComprobacionesAcreditacionPoolAlta]bool
	destinos := make([]any, len(comprobaciones))
	for indice := range comprobaciones {
		destinos[indice] = &comprobaciones[indice]
	}
	err := consultor.QueryRow(ctx, `
		SELECT current_user = session_user,
		       r.rolcanlogin,
		       NOT r.rolsuper,
		       NOT r.rolbypassrls,
		       NOT r.rolcreaterole,
		       NOT r.rolcreatedb,
		       NOT r.rolreplication,
		       pg_has_role(
		           current_user,
		           'vec_contratacion_temporal_ejecutor',
		           'MEMBER'
		       ),
		       NOT pg_has_role(
		           current_user,
		           'vec_contratacion_temporal_migrador',
		           'MEMBER'
		       ),
		       NOT pg_has_role(
		           current_user,
		           'vec_contratacion_temporal_propietario',
		           'MEMBER'
		       ),
		       has_schema_privilege(
		           current_user,
		           'vec_contratacion_temporal',
		           'USAGE'
		       ),
		       NOT has_schema_privilege(
		           current_user,
		           'vec_contratacion_temporal',
		           'CREATE'
		       ),
		       has_function_privilege(
		           current_user,
		           'vec_contratacion_temporal.resolver_candidatura_alta_tecnica_v1(text[],text[],text,text,text,text,text,text,text,text)',
		           'EXECUTE'
		       ),
		       has_function_privilege(
		           current_user,
		           'vec_contratacion_temporal.confirmar_alta_atestada_v2(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)',
		           'EXECUTE'
		       ),
		       NOT has_function_privilege(
		           current_user,
		           'vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)',
		           'EXECUTE'
		       ),
		       NOT has_function_privilege(
		           current_user,
		           'vec_contratacion_temporal.preparar_alta_v2(jsonb)',
		           'EXECUTE'
		       ),
		       NOT has_function_privilege(
		           current_user,
		           'vec_contratacion_temporal.reconciliar_agregado_alta_v1(bytea,text,text,text,text,text,text)',
		           'EXECUTE'
		       ),
		       NOT has_table_privilege(
		           current_user,
		           'vec_contratacion_temporal.identidad_reserva_alta',
		           'SELECT,INSERT,UPDATE,DELETE'
		       ),
		       NOT has_table_privilege(
		           current_user,
		           'vec_contratacion_temporal.expediente_alta',
		           'SELECT,INSERT,UPDATE,DELETE'
		       ),
		       NOT has_table_privilege(
		           current_user,
		           'vec_contratacion_temporal.candidatura_alta_tecnica',
		           'SELECT,INSERT,UPDATE,DELETE'
		       )
		  FROM pg_catalog.pg_roles r
		 WHERE r.rolname = current_user`,
	).Scan(destinos...)
	if err != nil {
		if errContexto := ctx.Err(); errContexto != nil {
			return errContexto
		}
		return ports.ErrPersistenciaNoDisponible
	}
	for _, acreditada := range comprobaciones {
		if !acreditada {
			return ports.ErrPersistenciaNoDisponible
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}
