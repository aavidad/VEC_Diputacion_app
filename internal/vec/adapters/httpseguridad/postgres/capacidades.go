package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

const (
	capacidadProvisionar = "vec_identidad_sesiones_v1_provisionador"
	capacidadRegistrar   = "vec_identidad_sesiones_v1_registrador"
	capacidadRevalidar   = "vec_identidad_sesiones_v1_revalidador"
	capacidadRevocar     = "vec_identidad_sesiones_v1_revocador"
)

type manifiestoCapacidadIdentidad struct {
	grupo     string
	funciones [2]string
}

// manifiestoParaCapacidad es deliberadamente cerrado e inmutable. Un nuevo
// rol o una nueva funcion no obtiene autoridad por coincidencia de nombre: el
// binario debe declarar expresamente el perfil minimo que sabe acreditar.
func manifiestoParaCapacidad(
	capacidad string,
) (manifiestoCapacidadIdentidad, bool) {
	switch capacidad {
	case capacidadProvisionar:
		return manifiestoCapacidadIdentidad{
			grupo: capacidadProvisionar,
			funciones: [2]string{
				"vec_identidad_sesiones_v1.provisionar_cuenta_v1(text,text,text,text,bigint,bytea,bytea,boolean,bytea)",
				"vec_identidad_sesiones_v1.registrar_alias_hmac_cuenta_v1(text,text,text,text,text,bigint,bytea,bytea)",
			},
		}, true
	case capacidadRegistrar:
		return manifiestoCapacidadIdentidad{
			grupo: capacidadRegistrar,
			funciones: [2]string{
				"vec_identidad_sesiones_v1.registrar_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)",
				"vec_identidad_sesiones_v1.reconciliar_registro_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)",
			},
		}, true
	case capacidadRevalidar:
		return manifiestoCapacidadIdentidad{
			grupo: capacidadRevalidar,
			funciones: [2]string{
				"vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)",
				"vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)",
			},
		}, true
	case capacidadRevocar:
		return manifiestoCapacidadIdentidad{
			grupo: capacidadRevocar,
			funciones: [2]string{
				"vec_identidad_sesiones_v1.cambiar_estado_cuenta_v1(text,text,text,text)",
				"vec_identidad_sesiones_v1.revocar_sesion_v1(text,text,text,text)",
			},
		}, true
	default:
		return manifiestoCapacidadIdentidad{}, false
	}
}

func (m manifiestoCapacidadIdentidad) valido() bool {
	return m.grupo != "" && m.funciones[0] != "" &&
		m.funciones[1] != "" && m.funciones[0] != m.funciones[1]
}

func (m manifiestoCapacidadIdentidad) firmasFunciones() []string {
	return []string{m.funciones[0], m.funciones[1]}
}

// MEMBER incluye membresias directas e indirectas con independencia de que
// sus cadenas habiliten USAGE (INHERIT) o SET ROLE. Exigir que el cierre total
// contenga solo el grupo esperado es deliberadamente mas estricto: tambien
// rechaza una cadena hoy inerte que un cambio posterior de opciones pudiera
// convertir en autoridad efectiva.
//
// pg_shdepend completa las vistas ACL de esta base con dependencias de objetos
// compartidos y de otras bases del mismo cluster. Asi se rechazan tambien
// ownership, ACL, privilegios iniciales y politicas fuera de la base actual.
// Las vistas concretas verifican tanto el significado exacto de las cuatro ACL
// permitidas como estados que no deben existir (default ACL y rolconfig).
const consultaAcreditarCapacidad = `
	WITH funciones_manifestadas AS MATERIALIZED (
		SELECT entrada.firma,
		       pg_catalog.to_regprocedure(entrada.firma) AS oid
		  FROM pg_catalog.unnest($2::text[]) AS entrada(firma)
	),
	base_actual AS MATERIALIZED (
		SELECT base.oid, base.datacl, base.datdba
		  FROM pg_catalog.pg_database AS base
		 WHERE base.datname = current_database()
	),
	esquema_identidad AS MATERIALIZED (
		SELECT esquema.oid, esquema.nspacl, esquema.nspowner
		  FROM pg_catalog.pg_namespace AS esquema
		 WHERE esquema.nspname = 'vec_identidad_sesiones_v1'
	)
	SELECT session_user::text, current_user::text,
	       COALESCE(
	           login.rolcanlogin AND NOT login.rolsuper
	           AND NOT login.rolcreatedb AND NOT login.rolcreaterole
	           AND login.rolinherit
	           AND NOT login.rolreplication AND NOT login.rolbypassrls,
	           false
	       ),
	       COALESCE(
	           NOT grupo.rolcanlogin AND NOT grupo.rolsuper
	           AND NOT grupo.rolcreatedb AND NOT grupo.rolcreaterole
	           AND NOT grupo.rolinherit
	           AND NOT grupo.rolreplication AND NOT grupo.rolbypassrls,
	           false
	       ),
	       COALESCE((
	           SELECT count(*) = 1
	                  AND bool_and(membresia.roleid = grupo.oid)
	                  AND bool_and(membresia.inherit_option)
	                  AND bool_and(NOT membresia.set_option)
	                  AND bool_and(NOT membresia.admin_option)
	             FROM pg_catalog.pg_auth_members AS membresia
	            WHERE membresia.member = login.oid
	       ), false),
	       COALESCE((
	           SELECT count(*) = 1
	                  AND bool_and(rol_alcanzable.oid = grupo.oid)
	             FROM pg_catalog.pg_roles AS rol_alcanzable
	            WHERE rol_alcanzable.oid <> login.oid
	              AND pg_catalog.pg_has_role(
	                      login.oid, rol_alcanzable.oid, 'MEMBER'
	                  )
	       ), false),
	       COALESCE((
	           SELECT count(*) = 2
	                  AND count(DISTINCT funcion.firma) = 2
	                  AND count(DISTINCT funcion.oid) = 2
	                  AND bool_and(funcion.oid IS NOT NULL)
	                  AND bool_and(procedimiento.pronamespace = esquema.oid)
	             FROM funciones_manifestadas AS funcion
	             LEFT JOIN pg_catalog.pg_proc AS procedimiento
	               ON procedimiento.oid = funcion.oid
	             CROSS JOIN esquema_identidad AS esquema
	       ), false),
	       COALESCE(
	           login.rolconfig IS NULL
	           AND NOT EXISTS (
	               SELECT 1
	                 FROM pg_catalog.pg_db_role_setting AS ajuste
	                WHERE ajuste.setrole = login.oid
	           )
	           AND NOT EXISTS (
	               SELECT 1
	                 FROM pg_catalog.pg_default_acl AS predeterminada
	                 LEFT JOIN LATERAL pg_catalog.aclexplode(
	                     COALESCE(predeterminada.defaclacl, '{}'::aclitem[])
	                 ) AS acl ON true
	                WHERE predeterminada.defaclrole = login.oid
	                   OR acl.grantee = login.oid
	                   OR acl.grantor = login.oid
	           )
	           AND NOT EXISTS (
	               SELECT 1
	                 FROM pg_catalog.pg_policy AS politica
	                WHERE login.oid = ANY(politica.polroles)
	           )
	           AND NOT EXISTS (
	               SELECT 1
	                 FROM pg_catalog.pg_shdepend AS dependencia
	                WHERE dependencia.refclassid =
	                      'pg_catalog.pg_authid'::pg_catalog.regclass
	                  AND dependencia.refobjid = login.oid
	           ),
	           false
	       ),
	       COALESCE(
	           grupo.rolconfig IS NULL
	           AND NOT EXISTS (
	               SELECT 1
	                 FROM pg_catalog.pg_db_role_setting AS ajuste
	                WHERE ajuste.setrole = grupo.oid
	           )
	           AND NOT EXISTS (
	               SELECT 1
	                 FROM pg_catalog.pg_default_acl AS predeterminada
	                 LEFT JOIN LATERAL pg_catalog.aclexplode(
	                     COALESCE(predeterminada.defaclacl, '{}'::aclitem[])
	                 ) AS acl ON true
	                WHERE predeterminada.defaclrole = grupo.oid
	                   OR acl.grantee = grupo.oid
	                   OR acl.grantor = grupo.oid
	           )
	           AND NOT EXISTS (
	               SELECT 1
	                 FROM pg_catalog.pg_policy AS politica
	                WHERE grupo.oid = ANY(politica.polroles)
	           )
	           AND (
	               SELECT count(*) = 4
	                      AND bool_and(
	                          dependencia.deptype = 'a'
	                          AND dependencia.objsubid = 0
	                          AND (
	                              (
	                                  dependencia.classid =
	                                      'pg_catalog.pg_database'::pg_catalog.regclass
	                                  AND dependencia.objid = base.oid
	                              ) OR (
	                                  dependencia.classid =
	                                      'pg_catalog.pg_namespace'::pg_catalog.regclass
	                                  AND dependencia.objid = esquema.oid
	                              ) OR (
	                                  dependencia.classid =
	                                      'pg_catalog.pg_proc'::pg_catalog.regclass
	                                  AND dependencia.objid IN (
	                                      SELECT funcion.oid
	                                        FROM funciones_manifestadas AS funcion
	                                  )
	                              )
	                          )
	                      )
	                 FROM pg_catalog.pg_shdepend AS dependencia
	                 CROSS JOIN base_actual AS base
	                 CROSS JOIN esquema_identidad AS esquema
	                WHERE dependencia.refclassid =
	                      'pg_catalog.pg_authid'::pg_catalog.regclass
	                  AND dependencia.refobjid = grupo.oid
	           )
	           AND (
	               SELECT count(*) = 1
	                      AND bool_and(acl.privilege_type = 'CONNECT')
	                      AND bool_and(NOT acl.is_grantable)
	                 FROM base_actual AS base
	                 CROSS JOIN LATERAL pg_catalog.aclexplode(
	                     COALESCE(
	                         base.datacl,
	                         pg_catalog.acldefault('d', base.datdba)
	                     )
	                 ) AS acl
	                WHERE acl.grantee = grupo.oid
	           )
	           AND (
	               SELECT count(*) = 1
	                      AND bool_and(acl.privilege_type = 'USAGE')
	                      AND bool_and(NOT acl.is_grantable)
	                 FROM esquema_identidad AS esquema
	                 CROSS JOIN LATERAL pg_catalog.aclexplode(
	                     COALESCE(
	                         esquema.nspacl,
	                         pg_catalog.acldefault('n', esquema.nspowner)
	                     )
	                 ) AS acl
	                WHERE acl.grantee = grupo.oid
	           )
	           AND (
	               SELECT count(*) = 2
	                      AND count(DISTINCT procedimiento.oid) = 2
	                      AND bool_and(acl.privilege_type = 'EXECUTE')
	                      AND bool_and(NOT acl.is_grantable)
	                 FROM pg_catalog.pg_proc AS procedimiento
	                 JOIN funciones_manifestadas AS funcion
	                   ON funcion.oid = procedimiento.oid
	                 CROSS JOIN LATERAL pg_catalog.aclexplode(
	                     COALESCE(
	                         procedimiento.proacl,
	                         pg_catalog.acldefault('f', procedimiento.proowner)
	                     )
	                 ) AS acl
	                WHERE acl.grantee = grupo.oid
	           ),
	           false
	       )
	  FROM pg_catalog.pg_roles AS login
	  JOIN pg_catalog.pg_roles AS grupo ON grupo.rolname = $1
	 WHERE login.rolname = session_user`

type consultorFilaCapacidad interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func acreditarCapacidadPool(
	ctx context.Context,
	pool *pgxpool.Pool,
	capacidadEsperada string,
) (string, error) {
	if ctx == nil || pool == nil {
		return "", httpseguridad.ErrRegistroSesionesAusente
	}
	return acreditarCapacidad(ctx, pool, capacidadEsperada)
}

func acreditarCapacidad(
	ctx context.Context,
	consultor consultorFilaCapacidad,
	capacidadEsperada string,
) (string, error) {
	manifiesto, encontrado := manifiestoParaCapacidad(capacidadEsperada)
	if ctx == nil || valorNulo(consultor) || !encontrado || !manifiesto.valido() {
		return "", httpseguridad.ErrRegistroSesionesAusente
	}
	var usuarioSesion, usuarioActual string
	var loginSeguro, grupoSeguro bool
	var membresiaDirectaSegura, membresiaTotalExclusiva bool
	var manifiestoResuelto, loginSinAutoridad, grupoConAutoridadExacta bool
	err := consultor.QueryRow(
		ctx,
		consultaAcreditarCapacidad,
		manifiesto.grupo,
		manifiesto.firmasFunciones(),
	).Scan(
		&usuarioSesion,
		&usuarioActual,
		&loginSeguro,
		&grupoSeguro,
		&membresiaDirectaSegura,
		&membresiaTotalExclusiva,
		&manifiestoResuelto,
		&loginSinAutoridad,
		&grupoConAutoridadExacta,
	)
	if err != nil || usuarioSesion == "" || usuarioSesion != usuarioActual ||
		!loginSeguro || !grupoSeguro || !membresiaDirectaSegura ||
		!membresiaTotalExclusiva || !manifiestoResuelto ||
		!loginSinAutoridad || !grupoConAutoridadExacta {
		return "", httpseguridad.ErrRegistroSesionesAusente
	}
	return usuarioSesion, nil
}
