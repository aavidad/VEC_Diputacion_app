package bootstrap

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	calendariospg "vec-diputacion-granada/internal/modules/calendarios/adapters/postgres"
	calendariosapp "vec-diputacion-granada/internal/modules/calendarios/application"
	calendariosports "vec-diputacion-granada/internal/modules/calendarios/ports"
)

// El calendario común contiene normas y festivos. El proceso externo usa un
// LOGIN propio y las dos funciones de lectura; no monta rutas de Calendarios.
func nuevaConsultaCalendariosMiBolsaPortalExterno(ctx context.Context, dsn string,
	topologia topologiaPostgreSQLPreferenciasUsuarios,
) (*pgxpool.Pool, calendariosports.ConsultaCalendarios, error) {
	pool, _, err := abrirPoolMiBolsaPortalExterno(ctx, dsn, rolLectorCalendariosDesarrollo)
	if err != nil {
		return nil, nil, errMiBolsaNoDisponible
	}
	valida := false
	defer func() {
		if !valida {
			pool.Close()
		}
	}()
	if cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, pool, topologia) != nil {
		return nil, nil, errMiBolsaNoDisponible
	}
	var minima bool
	err = pool.QueryRow(ctx, `SELECT
	 (SELECT count(*)=2 AND count(DISTINCT p.proname)=2 AND bool_and(p.prosecdef)
	  AND bool_and(p.proname=ANY(ARRAY['versiones_vigentes_v1','centros_con_calendario_v1']))
	  FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
	  WHERE n.nspname='vec_calendarios' AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE'))
	 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
	  WHERE n.nspname='vec_calendarios' AND c.relkind IN('r','p','v','m','f')
	  AND (pg_catalog.has_table_privilege(session_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
	   OR pg_catalog.has_any_column_privilege(session_user,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))
	 AND NOT pg_catalog.has_schema_privilege(session_user,'vec_calendarios','CREATE')
	 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n WHERE left(n.nspname,4)='vec_'
	  AND n.nspname<>'vec_calendarios' AND (pg_catalog.has_schema_privilege(session_user,n.oid,'USAGE')
	   OR pg_catalog.has_schema_privilege(session_user,n.oid,'CREATE')))`).Scan(&minima)
	if err != nil || !minima {
		return nil, nil, errMiBolsaNoDisponible
	}
	repositorio, err := calendariospg.NuevoRepositorio(pool)
	if err != nil {
		return nil, nil, errMiBolsaNoDisponible
	}
	servicio, err := calendariosapp.NuevoServicio(repositorio, relojCalendariosDesarrollo{})
	if err != nil {
		return nil, nil, errMiBolsaNoDisponible
	}
	valida = true
	return pool, servicio, nil
}
