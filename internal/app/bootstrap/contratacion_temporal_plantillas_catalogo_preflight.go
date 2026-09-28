package bootstrap

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	plantillasapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
)

// preflightCatalogoPlantillasCT comprueba que CT131 y CT133 estén instaladas
// y ejecutables por el LOGIN CT nominal antes de publicar las rutas. No
// provisiona la preimagen ni toca datos: eso pertenece al migrador separado.
func preflightCatalogoPlantillasCT(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return plantillasapp.ErrNoDisponible
	}
	return comprobarPreflightCatalogoPlantillasCT(ctx, pool)
}

func comprobarPreflightCatalogoPlantillasCT(ctx context.Context, consulta interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) error {
	if ctx == nil || ctx.Err() != nil || consulta == nil {
		return plantillasapp.ErrNoDisponible
	}
	const sql = `SELECT
  current_user = session_user
  AND (SELECT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls FROM pg_catalog.pg_roles WHERE rolname = session_user)
  AND pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
  AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
  AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_gobernador','MEMBER')
  AND (SELECT coalesce(bool_and(f.oid IS NOT NULL AND pg_catalog.has_function_privilege(session_user,f.oid,'EXECUTE')),false)
       FROM pg_catalog.unnest(ARRAY[
         'vec_contratacion_temporal.operar_catalogo_plantillas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
         'vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
         'vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(text,bigint,text)'
       ]) AS nombre
       CROSS JOIN LATERAL pg_catalog.to_regprocedure(nombre) AS f(oid))
  AND NOT pg_catalog.has_function_privilege(session_user,
      'vec_contratacion_temporal.provisionar_catalogo_plantillas_base_v1(jsonb,text,text,text)','EXECUTE')`
	var valido bool
	if err := consulta.QueryRow(ctx, sql).Scan(&valido); err != nil || !valido {
		return plantillasapp.ErrNoDisponible
	}
	return nil
}
