\set ON_ERROR_STOP on
-- Sólo en clon frío desechable con AUT30 y AD160. Todo se revierte.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
DO $acl$
DECLARE fuente regprocedure:='vec_autorizacion.leer_competencia_cargo_ct_v1(text,text,text)'::regprocedure;
 efecto regprocedure:='vec_autorizacion.revalidar_competencia_firmante_ct_v2(text)'::regprocedure;
BEGIN
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) a
   WHERE p.oid IN (fuente,efecto) AND a.grantee=0 AND a.privilege_type='EXECUTE')
  OR NOT pg_catalog.has_function_privilege('vec_autorizacion_fuente',fuente,'EXECUTE')
  OR NOT pg_catalog.has_function_privilege('vec_contratacion_temporal_propietario',efecto,'EXECUTE')
  OR pg_catalog.has_function_privilege('vec_contratacion_temporal_ejecutor',efecto,'EXECUTE')
  OR pg_catalog.has_table_privilege('vec_contratacion_temporal_propietario','vec_autorizacion.cargo_ct_plan','SELECT')
  OR pg_catalog.has_table_privilege('vec_autorizacion_fuente','vec_autorizacion.cargo_ct_recibo','SELECT')
  OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc WHERE oid IN (fuente,efecto)
   AND (proowner<>'vec_autorizacion_propietario'::regrole OR NOT prosecdef))
 THEN RAISE EXCEPTION 'AUT31: fronteras ACL incompatibles'; END IF;
END $acl$;
CREATE ROLE aut31_prueba_fuente LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_fuente TO aut31_prueba_fuente WITH INHERIT TRUE, SET FALSE;
CREATE ROLE aut31_prueba_ct LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_ejecutor TO aut31_prueba_ct WITH INHERIT TRUE, SET FALSE;
CREATE FUNCTION public.aut31_prueba_ct(p text) RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog AS $f$ SELECT vec_autorizacion.revalidar_competencia_firmante_ct_v2(p) $f$;
ALTER FUNCTION public.aut31_prueba_ct(text) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION public.aut31_prueba_ct(text) FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO aut31_prueba_ct;
GRANT EXECUTE ON FUNCTION public.aut31_prueba_ct(text) TO aut31_prueba_ct;
SET SESSION AUTHORIZATION aut31_prueba_fuente;
DO $ausencia$
BEGIN
 IF vec_autorizacion.leer_competencia_cargo_ct_v1('per_ausente_sintetica',
  'ct_cargo_direccion_rrhh','organizacion:aut31:ausente') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT31: fuente ausente acreditada'; END IF;
END $ausencia$;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION aut31_prueba_ct;
DO $denegaciones$
BEGIN
 IF public.aut31_prueba_ct('{}') IS DISTINCT FROM false
  OR public.aut31_prueba_ct('{"Via":"certificado_vec","Via":"portafirmas_registro_rrhh"}') IS DISTINCT FROM false
  OR public.aut31_prueba_ct('null') IS DISTINCT FROM false
  OR public.aut31_prueba_ct('{') IS DISTINCT FROM false
 THEN RAISE EXCEPTION 'AUT31: material incompleto o ambiguo concedido'; END IF;
END $denegaciones$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
