package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	rolLectorResultadoCoberturaO405 = "" +
		"vec_contratacion_temporal_lector_resultado_cobertura"
	esquemaFuncionRecuperacionResultadoCoberturaO405 = "" +
		"vec_contratacion_temporal"
	nombreFuncionRecuperacionResultadoCoberturaO405 = "" +
		"recuperar_resultado_propio_decision_cobertura_o405_v1"
	propietarioFuncionRecuperacionResultadoCoberturaO405 = "" +
		"vec_contratacion_temporal_propietario"
	firmaFuncionRecuperacionResultadoCoberturaO405 = "" +
		"vec_contratacion_temporal." +
		"recuperar_resultado_propio_decision_cobertura_o405_v1(jsonb)"
	argumentosFuncionRecuperacionResultadoCoberturaO405 = "p_consulta jsonb"
	retornoFuncionRecuperacionResultadoCoberturaO405    = "" +
		"TABLE(resultado_json jsonb)"
	lenguajeFuncionRecuperacionResultadoCoberturaO405     = "plpgsql"
	huellaProsrcFuncionRecuperacionResultadoCoberturaO405 = "" +
		"8e44dc20eef5d66b4acc3c6b89801ea85618b6f451151f2d50dd4be2ed06c0d3"
	huellaDefinicionFuncionRecuperacionResultadoCoberturaO405 = "" +
		"81dcbffeb598368bd11ee9100351c649b37308c6a7a0a0876835b86d82e7d555"
)

func configuracionFuncionRecuperacionResultadoCoberturaO405() []string {
	return []string{
		"TimeZone=UTC",
		"lock_timeout=2s",
		"row_security=on",
		"search_path=pg_catalog",
	}
}

type modoTLSAcreditacionPoolO405 uint8

const (
	modoTLSAcreditacionPoolO405Produccion modoTLSAcreditacionPoolO405 = iota + 1
	modoTLSAcreditacionPoolO405SocketUnixPrueba
)

type conexionAcreditacionPoolO405 interface {
	Configuracion() *pgx.ConnConfig
	Sello() *selloFabricaPoolO405
	QueryRow(context.Context, string, ...any) pgx.Row
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
	Liberar()
}

type origenAcreditacionPoolO405 interface {
	Configuracion() *pgxpool.Config
	Sello() *selloFabricaPoolO405
	Adquirir(context.Context) (conexionAcreditacionPoolO405, error)
}

// La consulta solo observa catálogos y la conexión física adquirida. No
// ejecuta la función de recuperación ni ninguna función de efecto.
const consultaAcreditacionPoolRecuperacionCoberturaO405 = `
	WITH funcion_lectura AS MATERIALIZED (
		SELECT pg_catalog.to_regprocedure($1::text) AS oid
	),
	base_actual AS MATERIALIZED (
		SELECT base.oid,base.datacl,base.datdba
		  FROM pg_catalog.pg_database AS base
		 WHERE base.datname=pg_catalog.current_database()
	),
	esquema_ct AS MATERIALIZED (
		SELECT esquema.oid,esquema.nspname,
		       esquema.nspacl,esquema.nspowner
		  FROM pg_catalog.pg_namespace AS esquema
		 WHERE esquema.nspname=$3
	)
	SELECT COALESCE(funcion.oid::oid,0::oid),
	       session_user::text,current_user::text,
	       COALESCE((
	         SELECT ssl AND (
	                  (
	                    version='TLSv1.2'
	                    AND cipher IN (
	                      'ECDHE-ECDSA-AES128-GCM-SHA256',
	                      'ECDHE-ECDSA-AES256-GCM-SHA384',
	                      'ECDHE-RSA-AES128-GCM-SHA256',
	                      'ECDHE-RSA-AES256-GCM-SHA384',
	                      'ECDHE-ECDSA-CHACHA20-POLY1305',
	                      'ECDHE-RSA-CHACHA20-POLY1305'
	                    )
	                  )
	                  OR (
	                    version='TLSv1.3'
	                    AND cipher IN (
	                      'TLS_AES_128_GCM_SHA256',
	                      'TLS_AES_256_GCM_SHA384',
	                      'TLS_CHACHA20_POLY1305_SHA256'
	                    )
	                  )
	                )
	           FROM pg_catalog.pg_stat_ssl
	          WHERE pid=pg_catalog.pg_backend_pid()
	       ),false),
	       NOT pg_catalog.pg_is_in_recovery(),
	       COALESCE(
	         login.rolcanlogin AND NOT login.rolsuper
	         AND NOT login.rolcreatedb AND NOT login.rolcreaterole
	         AND login.rolinherit AND NOT login.rolreplication
	         AND NOT login.rolbypassrls,
	         false
	       ),
	       COALESCE(
	         NOT grupo.rolcanlogin AND NOT grupo.rolsuper
	         AND NOT grupo.rolcreatedb AND NOT grupo.rolcreaterole
	         AND grupo.rolinherit AND NOT grupo.rolreplication
	         AND NOT grupo.rolbypassrls,
	         false
	       ),
	       COALESCE((
	         SELECT count(*)=1
	                AND bool_and(m.roleid=grupo.oid)
	                AND bool_and(m.inherit_option)
	                AND bool_and(NOT m.set_option)
	                AND bool_and(NOT m.admin_option)
	           FROM pg_catalog.pg_auth_members AS m
	          WHERE m.member=login.oid
	       ),false),
	       COALESCE((
	         SELECT count(*)=1 AND bool_and(alcanzable.oid=grupo.oid)
	           FROM pg_catalog.pg_roles AS alcanzable
	          WHERE alcanzable.oid<>login.oid
	            AND pg_catalog.pg_has_role(
	                  login.oid,alcanzable.oid,'MEMBER')
	       ),false),
	       COALESCE(
	         login.rolconfig IS NULL
	         AND NOT EXISTS(
	           SELECT 1 FROM pg_catalog.pg_db_role_setting AS ajuste
	            WHERE ajuste.setrole=login.oid)
	         AND NOT EXISTS(
	           SELECT 1
	             FROM pg_catalog.pg_default_acl AS predeterminada
	             LEFT JOIN LATERAL pg_catalog.aclexplode(
	               COALESCE(predeterminada.defaclacl,'{}'::aclitem[])
	             ) AS acl ON true
	            WHERE predeterminada.defaclrole=login.oid
	               OR acl.grantee=login.oid OR acl.grantor=login.oid)
	         AND NOT EXISTS(
	           SELECT 1 FROM pg_catalog.pg_policy AS politica
	            WHERE login.oid=ANY(politica.polroles))
	         AND NOT EXISTS(
	           SELECT 1 FROM pg_catalog.pg_shdepend AS dependencia
	            WHERE dependencia.refclassid=
	                  'pg_catalog.pg_authid'::pg_catalog.regclass
	              AND dependencia.refobjid=login.oid),
	         false
	       ),
	       COALESCE(
	         grupo.rolconfig IS NULL
	         AND NOT EXISTS(
	           SELECT 1 FROM pg_catalog.pg_db_role_setting AS ajuste
	            WHERE ajuste.setrole=grupo.oid)
	         AND NOT EXISTS(
	           SELECT 1
	             FROM pg_catalog.pg_default_acl AS predeterminada
	             LEFT JOIN LATERAL pg_catalog.aclexplode(
	               COALESCE(predeterminada.defaclacl,'{}'::aclitem[])
	             ) AS acl ON true
	            WHERE predeterminada.defaclrole=grupo.oid
	               OR acl.grantee=grupo.oid OR acl.grantor=grupo.oid)
	         AND NOT EXISTS(
	           SELECT 1 FROM pg_catalog.pg_policy AS politica
	            WHERE grupo.oid=ANY(politica.polroles))
	         AND (
	           SELECT count(*)=3
	                  AND bool_and(
	                    dependencia.deptype='a'
	                    AND dependencia.objsubid=0
	                    AND (
	                      (dependencia.classid=
	                         'pg_catalog.pg_database'::pg_catalog.regclass
	                       AND dependencia.objid=base.oid)
	                      OR (dependencia.classid=
	                            'pg_catalog.pg_namespace'::pg_catalog.regclass
	                          AND dependencia.objid=esquema.oid)
	                      OR (dependencia.classid=
	                            'pg_catalog.pg_proc'::pg_catalog.regclass
	                          AND dependencia.objid=funcion.oid)
	                    ))
	             FROM pg_catalog.pg_shdepend AS dependencia
	             CROSS JOIN base_actual AS base
	             CROSS JOIN esquema_ct AS esquema
	             CROSS JOIN funcion_lectura AS funcion
	            WHERE dependencia.refclassid=
	                  'pg_catalog.pg_authid'::pg_catalog.regclass
	              AND dependencia.refobjid=grupo.oid)
	         AND (
	           SELECT count(*)=1
	                  AND bool_and(acl.privilege_type='CONNECT')
	                  AND bool_and(NOT acl.is_grantable)
	             FROM base_actual AS base
	             CROSS JOIN LATERAL pg_catalog.aclexplode(
	               COALESCE(
	                 base.datacl,
	                 pg_catalog.acldefault('d',base.datdba))
	             ) AS acl
	            WHERE acl.grantee=grupo.oid)
	         AND (
	           SELECT count(*)=1
	                  AND bool_and(acl.privilege_type='USAGE')
	                  AND bool_and(NOT acl.is_grantable)
	             FROM esquema_ct AS esquema
	             CROSS JOIN LATERAL pg_catalog.aclexplode(
	               COALESCE(
	                 esquema.nspacl,
	                 pg_catalog.acldefault('n',esquema.nspowner))
	             ) AS acl
	            WHERE acl.grantee=grupo.oid)
	         AND (
	           SELECT count(*)=1
	                  AND bool_and(acl.privilege_type='EXECUTE')
	                  AND bool_and(NOT acl.is_grantable)
	             FROM pg_catalog.pg_proc AS procedimiento
	             CROSS JOIN funcion_lectura AS funcion
	             CROSS JOIN LATERAL pg_catalog.aclexplode(
	               COALESCE(
	                 procedimiento.proacl,
	                 pg_catalog.acldefault('f',procedimiento.proowner))
	             ) AS acl
	            WHERE procedimiento.oid=funcion.oid
	              AND acl.grantee=grupo.oid),
	         false
	       ),
	       COALESCE(
	         pg_catalog.has_database_privilege(
	           login.oid,base.oid,'CONNECT')
	         AND NOT pg_catalog.has_database_privilege(
	           login.oid,base.oid,'CREATE')
	         AND NOT pg_catalog.has_database_privilege(
	           login.oid,base.oid,'TEMP')
	         AND pg_catalog.has_schema_privilege(
	           login.oid,esquema.oid,'USAGE')
	         AND NOT pg_catalog.has_schema_privilege(
	           login.oid,esquema.oid,'CREATE')
	         AND pg_catalog.has_function_privilege(
	           login.oid,funcion.oid,'EXECUTE')
	         AND (
	           SELECT count(*)=1
	                  AND bool_and(procedimiento.oid=funcion.oid)
	             FROM pg_catalog.pg_proc AS procedimiento
	            WHERE procedimiento.pronamespace=esquema.oid
	              AND pg_catalog.has_function_privilege(
	                    login.oid,procedimiento.oid,'EXECUTE'))
	         AND NOT EXISTS(
	           SELECT 1 FROM pg_catalog.pg_class AS objeto
	            WHERE objeto.relnamespace=esquema.oid
	              AND objeto.relkind IN ('r','p','v','m','f')
	              AND pg_catalog.has_table_privilege(
	                    login.oid,objeto.oid,
	                    'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,'||
	                    'REFERENCES,TRIGGER,MAINTAIN'))
	         AND NOT EXISTS(
	           SELECT 1 FROM pg_catalog.pg_class AS secuencia
	            WHERE secuencia.relnamespace=esquema.oid
	              AND CASE WHEN secuencia.relkind='S' THEN
	                    pg_catalog.has_sequence_privilege(
	                      login.oid,secuencia.oid,'USAGE,SELECT,UPDATE')
	                  ELSE false END),
	         false
	       ),
	       COALESCE(
	         funcion.oid IS NOT NULL
	         AND procedimiento.oid=funcion.oid
	         AND procedimiento.pronamespace=esquema.oid
	         AND esquema.nspname=$3
	         AND procedimiento.proname=$4
	         AND procedimiento.prokind='f',
	         false
	       ),
	       COALESCE(
	         procedimiento.proowner=propietario.oid,
	         false
	       ),
	       COALESCE(
	         procedimiento.prosecdef
	         AND procedimiento.provolatile='s',
	         false
	       ),
	       COALESCE(
	         (
	           SELECT pg_catalog.array_agg(
	                    valor ORDER BY valor COLLATE "C")
	             FROM pg_catalog.unnest(
	               COALESCE(
	                 procedimiento.proconfig,
	                 ARRAY[]::text[])
	             ) AS configuracion(valor)
	         ) IS NOT DISTINCT FROM $6::text[],
	         false
	       ),
	       COALESCE(
	         procedimiento.pronargs=1
	         AND procedimiento.pronargdefaults=0
	         AND pg_catalog.pg_get_function_identity_arguments(
	               procedimiento.oid)=$7
	         AND pg_catalog.pg_get_function_arguments(
	               procedimiento.oid)=$7
	         AND pg_catalog.pg_get_function_result(
	               procedimiento.oid)=$8
	         AND pg_catalog.cardinality(
	               procedimiento.proargtypes::oid[])=1
	         AND procedimiento.proargtypes[0]=
	               'pg_catalog.jsonb'::pg_catalog.regtype::oid
	         AND procedimiento.proallargtypes IS NOT DISTINCT FROM
	               ARRAY[
	                 'pg_catalog.jsonb'::pg_catalog.regtype::oid,
	                 'pg_catalog.jsonb'::pg_catalog.regtype::oid]
	         AND procedimiento.proargmodes IS NOT DISTINCT FROM
	               ARRAY['i','t']::"char"[]
	         AND procedimiento.proargnames IS NOT DISTINCT FROM
	               ARRAY['p_consulta','resultado_json']::text[]
	         AND procedimiento.prorettype=
	               'pg_catalog.jsonb'::pg_catalog.regtype::oid
	         AND procedimiento.proretset,
	         false
	       ),
	       COALESCE(
	         lenguaje.lanname=$9
	         AND procedimiento.probin IS NULL,
	         false
	       ),
	       COALESCE(
	         pg_catalog.encode(
	           pg_catalog.sha256(
	             pg_catalog.convert_to(
	               procedimiento.prosrc,'UTF8')),
	           'hex')=$10,
	         false
	       ),
	       COALESCE(
	         pg_catalog.encode(
	           pg_catalog.sha256(
	             pg_catalog.convert_to(
	               pg_catalog.rtrim(
	                 pg_catalog.regexp_replace(
	                   pg_catalog.replace(
	                     pg_catalog.pg_get_functiondef(
	                       procedimiento.oid),
	                     E'\r\n',E'\n'),
	                   E'[ \t]+\n',E'\n','g'),
	                 E' \t\n\r')||E'\n',
	               'UTF8')),
	           'hex')=$11,
	         false
	       )
	  FROM pg_catalog.pg_roles AS login
	  JOIN pg_catalog.pg_roles AS grupo
	    ON grupo.rolname=$2
	  LEFT JOIN pg_catalog.pg_roles AS propietario
	    ON propietario.rolname=$5
	  CROSS JOIN base_actual AS base
	  CROSS JOIN esquema_ct AS esquema
	  CROSS JOIN funcion_lectura AS funcion
	  LEFT JOIN pg_catalog.pg_proc AS procedimiento
	    ON procedimiento.oid=funcion.oid
	  LEFT JOIN pg_catalog.pg_language AS lenguaje
	    ON lenguaje.oid=procedimiento.prolang
	 WHERE login.rolname=session_user`
