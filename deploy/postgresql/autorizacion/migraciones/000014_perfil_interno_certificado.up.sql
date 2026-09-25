\set ON_ERROR_STOP on
-- El rol temporal de consulta CT sólo se entrega a la fuente nominal interna.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_autorizacion:migracion:000014',0));

DO $preimagen$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
  SELECT * INTO p FROM pg_catalog.pg_proc
   WHERE oid=pg_catalog.to_regprocedure('vec_autorizacion.obtener_instantanea(text,text)');
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                  WHERE rolname=current_user AND rolsuper)
     OR p.oid IS NULL
     OR p.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
     OR p.prolang IS DISTINCT FROM (SELECT oid FROM pg_catalog.pg_language WHERE lanname='sql')
     OR p.provolatile IS DISTINCT FROM 's'
     OR p.prosecdef IS DISTINCT FROM true OR p.proretset IS DISTINCT FROM true
     OR p.prorettype IS DISTINCT FROM 'record'::regtype
     OR p.pronargs IS DISTINCT FROM 2
     OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[]
     OR pg_catalog.octet_length(p.prosrc) <> 1533
     OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex')
        <> '20e4cc87bea46cbffb2c5cdb3cae9ee3a58f1ad22c7c397661639d1fc398078b'
     OR (SELECT count(*) FROM pg_catalog.aclexplode(p.proacl)) <> 2
     OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
                WHERE a.grantee NOT IN ('vec_autorizacion_propietario'::regrole,
                                        'vec_autorizacion_fuente'::regrole)
                   OR a.privilege_type <> 'EXECUTE' OR a.is_grantable)
     OR pg_catalog.to_regrole('vec_interno_v3_fuente_autorizacion_desarrollo') IS NULL
     OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles r
                     WHERE r.rolname='vec_interno_v3_fuente_autorizacion_desarrollo'
                       AND r.rolcanlogin AND r.rolinherit AND NOT r.rolsuper
                       AND NOT r.rolcreatedb AND NOT r.rolcreaterole
                       AND NOT r.rolreplication AND NOT r.rolbypassrls)
     OR NOT pg_catalog.pg_has_role('vec_interno_v3_fuente_autorizacion_desarrollo',
                                   'vec_autorizacion_fuente','MEMBER')
     OR NOT pg_catalog.has_function_privilege(
          'vec_interno_v3_fuente_autorizacion_desarrollo',
          'vec_autorizacion.obtener_instantanea(text,text)','EXECUTE')
     OR pg_catalog.has_table_privilege(
          'vec_interno_v3_fuente_autorizacion_desarrollo',
          'vec_autorizacion.asignacion_perfil','SELECT')
     OR pg_catalog.has_table_privilege(
          'vec_interno_v3_fuente_autorizacion_desarrollo',
          'vec_autorizacion.version_rol','SELECT')
     OR (SELECT count(*) FROM pg_catalog.pg_auth_members m
          WHERE m.member='vec_interno_v3_fuente_autorizacion_desarrollo'::regrole) <> 1
     OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
          WHERE m.member='vec_interno_v3_fuente_autorizacion_desarrollo'::regrole
            AND m.roleid='vec_autorizacion_fuente'::regrole
            AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
  THEN RAISE EXCEPTION 'AUT14: preimagen o LOGIN interno incompatibles'
       USING ERRCODE='55000'; END IF;
END $preimagen$;

SET LOCAL ROLE vec_autorizacion_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion.obtener_instantanea(
    p_principal_id text,
    p_perfil_activo_ref text
)
RETURNS TABLE (
    documento_asignacion jsonb,
    documento_rol jsonb,
    documento_control_rol jsonb,
    revision_catalogo text,
    huella_catalogo text,
    documentos_politicas jsonb
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT
        asignacion.documento,
        rol.documento,
        control.documento,
        catalogo.revision::text,
        catalogo.huella_sha256,
        COALESCE((
            SELECT jsonb_agg(politica.documento ORDER BY politica.politica_ref)
              FROM vec_autorizacion.politica_restrictiva_actual AS politica_actual
              JOIN vec_autorizacion.politica_restrictiva AS politica
                ON politica.politica_id = politica_actual.politica_id
               AND politica.politica_ref = politica_actual.politica_ref
        ), '[]'::jsonb)
      FROM vec_autorizacion.asignacion_perfil_actual AS asignacion_actual
      JOIN vec_autorizacion.asignacion_perfil AS asignacion
        ON asignacion.perfil_activo_ref = asignacion_actual.perfil_activo_ref
       AND asignacion.asignacion_ref = asignacion_actual.asignacion_ref
      JOIN vec_autorizacion.version_rol AS rol
        ON rol.version_rol_ref = asignacion.version_rol_ref
      JOIN vec_autorizacion.control_vigencia_version_rol_actual AS control_actual
        ON control_actual.version_rol_ref = rol.version_rol_ref
      JOIN vec_autorizacion.control_vigencia_version_rol AS control
        ON control.version_rol_ref = control_actual.version_rol_ref
       AND control.revision = control_actual.revision
      CROSS JOIN vec_autorizacion.control_catalogo_politicas AS catalogo
     WHERE asignacion_actual.perfil_activo_ref = p_perfil_activo_ref
       AND asignacion.principal_id = p_principal_id
       AND catalogo.control_id = true
       AND (NOT pg_catalog.starts_with(rol.rol_id,
                     'rrhh_interno_certificado_seguimiento_ct_')
            OR (session_user = 'vec_interno_v3_fuente_autorizacion_desarrollo'
                AND pg_catalog.pg_has_role(session_user,'vec_autorizacion_fuente','MEMBER')
                AND EXISTS (SELECT 1 FROM pg_catalog.pg_roles r
                     WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
                       AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole
                       AND NOT r.rolreplication AND NOT r.rolbypassrls)
                AND (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members m
                     WHERE m.member=session_user::regrole) = 1
                AND EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
                     WHERE m.member=session_user::regrole
                       AND m.roleid='vec_autorizacion_fuente'::regrole
                       AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)))
$funcion$;

-- CREATE OR REPLACE conserva owner y ACL. Exigimos que no los haya cambiado.
DO $postimagen$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc
  WHERE oid='vec_autorizacion.obtener_instantanea(text,text)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
    OR (SELECT count(*) FROM pg_catalog.aclexplode(p.proacl))<>2
    OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
      WHERE a.grantee NOT IN ('vec_autorizacion_propietario'::regrole,
                              'vec_autorizacion_fuente'::regrole)
         OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
 THEN RAISE EXCEPTION 'AUT14: ACL/propietario alterados' USING ERRCODE='55000'; END IF;
END $postimagen$;
COMMIT;
