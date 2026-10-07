\set ON_ERROR_STOP on
-- Se ejecuta en un clon desechable después de AUT30. Todo se revierte.
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $acl$
DECLARE f regprocedure := 'vec_autorizacion.revalidar_competencia_firmante_ct_v1(text)'::regprocedure;
BEGIN
  IF pg_catalog.pg_get_userbyid((SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f))
        IS DISTINCT FROM 'vec_autorizacion_propietario'
     OR NOT pg_catalog.has_function_privilege(
          'vec_contratacion_temporal_propietario',f,'EXECUTE')
     OR pg_catalog.has_function_privilege('public',f,'EXECUTE')
     OR pg_catalog.has_function_privilege(
          'vec_contratacion_temporal_ejecutor',f,'EXECUTE')
     OR pg_catalog.has_table_privilege('vec_contratacion_temporal_propietario',
          'vec_autorizacion.asignacion_perfil_actual','SELECT')
     OR pg_catalog.has_table_privilege('vec_contratacion_temporal_propietario',
          'vec_autorizacion.control_vigencia_version_rol_actual','SELECT') THEN
    RAISE EXCEPTION 'AUT30: ACL o propietario incompatibles';
  END IF;
END $acl$;

CREATE ROLE aut30_prueba_login LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_ejecutor TO aut30_prueba_login;
GRANT USAGE ON SCHEMA public TO aut30_prueba_login;

-- Simula la llamada desde el único propietario CT, con sesión de ejecutor.
CREATE FUNCTION public.aut30_prueba_llamada(p_material text)
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
  SELECT vec_autorizacion.revalidar_competencia_firmante_ct_v1(p_material)
$funcion$;
ALTER FUNCTION public.aut30_prueba_llamada(text)
  OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION public.aut30_prueba_llamada(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.aut30_prueba_llamada(text)
  TO aut30_prueba_login;

SET SESSION AUTHORIZATION aut30_prueba_login;
DO $cierre$
BEGIN
  IF public.aut30_prueba_llamada('{}') IS DISTINCT FROM false THEN
    RAISE EXCEPTION 'AUT30: material sin fuente nominal concedido';
  END IF;
END $cierre$;
RESET SESSION AUTHORIZATION;

ROLLBACK;
