package bootstrap

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
)

const (
	rolAuditoriaFronteraPostgreSQLBolsaDesarrollo         = "vec_bolsa_llamamientos_registrador_frontera"
	funcionAuditoriaFronteraBorradorLlamamientoPostgreSQL = "vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text)"
)

// nuevaAuditoriaFronteraBorradorLlamamientoPostgreSQLDesarrollo abre una
// conexion que solo sirve para la bitacora de frontera. Se valida antes de
// publicar contexto o instalar el PDP común para que B-BACK falle cerrado.
func nuevaAuditoriaFronteraBorradorLlamamientoPostgreSQLDesarrollo(ctx context.Context, cfg config.Config) (*postgresbolsa.AuditoriaIntentoBorradorLlamamientoPostgreSQL, func(), error) {
	if ctx == nil {
		return nil, nil, errBorradorNoDisponibleEn()
	}
	dsn, err := cfg.DSNBolsaAuditoriaFronteraSeparado()
	if err != nil || dsn == "" {
		return nil, nil, errBorradorNoDisponibleEn()
	}
	pool, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, dsn, "vec-bolsa-bback-auditoria-frontera", rolAuditoriaFronteraPostgreSQLBolsaDesarrollo)
	if err != nil {
		return nil, nil, errBorradorNoDisponibleEn()
	}
	cerrar := cerrarPoolIdempotenteBorradorLlamamientoDesarrollo(pool)
	if err := preflightAuditoriaFronteraBorradorLlamamientoPostgreSQLDesarrollo(ctx, pool); err != nil {
		cerrar()
		return nil, nil, errBorradorNoDisponibleEn()
	}
	auditoria, err := postgresbolsa.NuevaAuditoriaIntentoBorradorLlamamientoPostgreSQL(pool)
	if err != nil {
		cerrar()
		return nil, nil, errBorradorNoDisponibleEn()
	}
	return auditoria, cerrar, nil
}

func cerrarPoolIdempotenteBorradorLlamamientoDesarrollo(pool *pgxpool.Pool) func() {
	var unaVez sync.Once
	return func() {
		unaVez.Do(func() {
			if pool != nil {
				pool.Close()
			}
		})
	}
}

// configurarVerificacionPorConexionAuditoriaFronteraBolsaDesarrollo no
// modifica ningún pool CT. El registrador se verifica cuando nace cada conexión
// física y también antes de cada adquisición: un pool no acredita por sí solo
// que todas sus conexiones sigan conservando el mínimo privilegio.
func configurarVerificacionPorConexionAuditoriaFronteraBolsaDesarrollo(configuracion *pgxpool.Config, rolEsperado string) {
	if configuracion == nil || rolEsperado != rolAuditoriaFronteraPostgreSQLBolsaDesarrollo {
		return
	}
	validar := func(ctx context.Context, conexion *pgx.Conn) error {
		if conexion == nil {
			return falloPostgreSQLCTDesarrollo(nil)
		}
		if _, err := comprobarIdentidadAuditoriaFronteraPostgreSQLBolsaDesarrollo(ctx, conexion); err != nil {
			return err
		}
		return comprobarPreflightAuditoriaFronteraBorradorLlamamientoPostgreSQLDesarrollo(ctx, conexion)
	}
	configuracion.AfterConnect = validar
	configuracion.BeforeAcquire = func(ctx context.Context, conexion *pgx.Conn) bool {
		return validar(ctx, conexion) == nil
	}
}

// comprobarIdentidadAuditoriaFronteraPostgreSQLBolsaDesarrollo conserva el
// control CT intacto y exige para Bolsa el único grupo NOLOGIN permitido. El
// LOGIN no puede heredar ningún segundo rol, incluido el ejecutor de negocio.
func comprobarIdentidadAuditoriaFronteraPostgreSQLBolsaDesarrollo(ctx context.Context, consultador interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) (string, error) {
	if ctx == nil || consultador == nil {
		return "", falloPostgreSQLCTDesarrollo(nil)
	}
	var usuario string
	var valido bool
	err := consultador.QueryRow(ctx, `
		WITH RECURSIVE membresias_efectivas(rol_id, admin_option) AS (
			SELECT directa.roleid, directa.admin_option FROM pg_catalog.pg_auth_members AS directa WHERE directa.member = session_user::regrole
			UNION
			SELECT siguiente.roleid, previa.admin_option OR siguiente.admin_option FROM pg_catalog.pg_auth_members AS siguiente JOIN membresias_efectivas AS previa ON previa.rol_id = siguiente.member
		)
		SELECT session_user::text,
		       session_user = current_user AND identidad.rolcanlogin AND identidad.rolinherit
		       AND NOT identidad.rolsuper AND NOT identidad.rolcreatedb AND NOT identidad.rolcreaterole
		       AND NOT identidad.rolreplication AND NOT identidad.rolbypassrls
		       AND pg_catalog.pg_has_role(session_user, $1::regrole, 'MEMBER')
		       AND pg_catalog.pg_has_role(session_user, $1::regrole, 'USAGE')
		       AND NOT grupo.rolcanlogin AND NOT grupo.rolsuper AND NOT grupo.rolcreatedb
		       AND NOT grupo.rolcreaterole AND grupo.rolinherit AND NOT grupo.rolreplication
		       AND NOT grupo.rolbypassrls
		       AND NOT EXISTS (SELECT 1 FROM membresias_efectivas WHERE rol_id <> $1::regrole OR admin_option)
		  FROM pg_catalog.pg_roles AS identidad
		 CROSS JOIN pg_catalog.pg_roles AS grupo
		 WHERE identidad.rolname = session_user AND grupo.oid = $1::regrole`, rolAuditoriaFronteraPostgreSQLBolsaDesarrollo).Scan(&usuario, &valido)
	if err != nil || !valido || usuario == "" {
		return "", falloPostgreSQLCTDesarrollo(err)
	}
	return usuario, nil
}

func preflightAuditoriaFronteraBorradorLlamamientoPostgreSQLDesarrollo(ctx context.Context, pool *pgxpool.Pool) error {
	if ctx == nil || pool == nil {
		return errBorradorNoDisponibleEn()
	}
	return comprobarPreflightAuditoriaFronteraBorradorLlamamientoPostgreSQLDesarrollo(ctx, pool)
}

func comprobarPreflightAuditoriaFronteraBorradorLlamamientoPostgreSQLDesarrollo(ctx context.Context, consultador interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) error {
	if ctx == nil || consultador == nil {
		return errBorradorNoDisponibleEn()
	}
	var valido bool
	err := consultador.QueryRow(ctx, `
		WITH funcion_nominal AS (
			SELECT pg_catalog.to_regprocedure($1) AS oid
		), esquemas_usuario AS (
			SELECT espacio.oid, espacio.nspname, espacio.nspowner, espacio.nspacl
			  FROM pg_catalog.pg_namespace AS espacio
			 WHERE espacio.nspname NOT IN ('pg_catalog', 'information_schema')
			   AND espacio.nspname NOT LIKE 'pg_toast%'
			   AND espacio.nspname NOT LIKE 'pg_temp_%'
			   AND NOT pg_catalog.pg_is_other_temp_schema(espacio.oid)
		)
		SELECT (SELECT oid IS NOT NULL FROM funcion_nominal)
		   AND pg_catalog.has_function_privilege(session_user, (SELECT oid FROM funcion_nominal), 'EXECUTE')
		   AND NOT pg_catalog.has_database_privilege(session_user, pg_catalog.current_database(), 'CREATE')
		   AND NOT pg_catalog.has_database_privilege(session_user, pg_catalog.current_database(), 'TEMPORARY')
		   AND NOT EXISTS (
		       SELECT 1 FROM pg_catalog.pg_database AS base
		        WHERE base.datname = pg_catalog.current_database()
		          AND pg_catalog.pg_has_role(session_user, base.datdba, 'MEMBER')
		   )
		   AND NOT EXISTS (
		       SELECT 1 FROM pg_catalog.pg_database AS base
		         CROSS JOIN LATERAL pg_catalog.aclexplode(base.datacl) AS acl
		        WHERE base.datname = pg_catalog.current_database()
		          AND acl.privilege_type = 'CONNECT' AND acl.is_grantable
		          AND (acl.grantee = 0 OR pg_catalog.pg_has_role(session_user, acl.grantee, 'USAGE'))
		   )
		   AND NOT EXISTS (
		       SELECT 1 FROM esquemas_usuario AS espacio
		        WHERE (espacio.nspname = 'vec_bolsa_llamamientos' AND (
		                  pg_catalog.pg_has_role(session_user, espacio.nspowner, 'MEMBER')
		                  OR NOT pg_catalog.has_schema_privilege(session_user, espacio.oid, 'USAGE')
		                  OR pg_catalog.has_schema_privilege(session_user, espacio.oid, 'CREATE')
		              ))
		           OR (espacio.nspname <> 'vec_bolsa_llamamientos' AND (
		                  pg_catalog.pg_has_role(session_user, espacio.nspowner, 'MEMBER')
		                  OR pg_catalog.has_schema_privilege(session_user, espacio.oid, 'USAGE')
		                  OR pg_catalog.has_schema_privilege(session_user, espacio.oid, 'CREATE')
		              ))
		   )
		   AND NOT EXISTS (
		       SELECT 1 FROM esquemas_usuario AS espacio
		         CROSS JOIN LATERAL pg_catalog.aclexplode(espacio.nspacl) AS acl
		        WHERE espacio.nspname = 'vec_bolsa_llamamientos'
		          AND acl.privilege_type = 'USAGE' AND acl.is_grantable
		          AND (acl.grantee = 0 OR pg_catalog.pg_has_role(session_user, acl.grantee, 'USAGE'))
		   )
		   AND NOT EXISTS (
		       SELECT 1 FROM pg_catalog.pg_proc AS funcion JOIN esquemas_usuario AS espacio ON espacio.oid = funcion.pronamespace
		        WHERE pg_catalog.pg_has_role(session_user, funcion.proowner, 'MEMBER') OR (
		              funcion.oid <> (SELECT oid FROM funcion_nominal)
		              AND pg_catalog.has_function_privilege(session_user, funcion.oid, 'EXECUTE')
		          )
		   )
		   AND NOT EXISTS (
		       SELECT 1 FROM pg_catalog.pg_proc AS funcion
		         CROSS JOIN LATERAL pg_catalog.aclexplode(funcion.proacl) AS acl
		        WHERE funcion.oid = (SELECT oid FROM funcion_nominal)
		          AND acl.privilege_type = 'EXECUTE' AND acl.is_grantable
		          AND (acl.grantee = 0 OR pg_catalog.pg_has_role(session_user, acl.grantee, 'USAGE'))
		   )
		   AND NOT EXISTS (
		       SELECT 1 FROM pg_catalog.pg_class AS objeto JOIN esquemas_usuario AS espacio ON espacio.oid = objeto.relnamespace
		        WHERE
		             pg_catalog.pg_has_role(session_user, objeto.relowner, 'MEMBER')
		          OR (objeto.relkind IN ('r', 'p', 'v', 'm', 'f') AND (
		                   pg_catalog.has_table_privilege(session_user, objeto.oid, 'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN')
		                   OR pg_catalog.has_any_column_privilege(session_user, objeto.oid, 'SELECT,INSERT,UPDATE,REFERENCES')
		               )
		              )
		           OR (objeto.relkind = 'S' AND pg_catalog.has_sequence_privilege(session_user, objeto.oid, 'USAGE,SELECT,UPDATE'))
		   )
		   AND NOT EXISTS (
		       SELECT 1 FROM pg_catalog.pg_type AS tipo JOIN esquemas_usuario AS espacio ON espacio.oid = tipo.typnamespace
		          AND pg_catalog.pg_has_role(session_user, tipo.typowner, 'MEMBER')
		   )
		   AND NOT EXISTS (
		       SELECT 1
		         FROM pg_catalog.pg_proc AS funcion
		         JOIN pg_catalog.pg_namespace AS espacio ON espacio.oid = funcion.pronamespace
		         CROSS JOIN LATERAL pg_catalog.aclexplode(funcion.proacl) AS acl
		        WHERE espacio.nspname IN ('pg_catalog', 'information_schema')
		          AND acl.privilege_type = 'EXECUTE'
		          AND (acl.grantee = 0 OR pg_catalog.pg_has_role(session_user, acl.grantee, 'USAGE'))
		   )`, funcionAuditoriaFronteraBorradorLlamamientoPostgreSQL).Scan(&valido)
	if err != nil || !valido {
		return errBorradorNoDisponibleEn()
	}
	return nil
}
