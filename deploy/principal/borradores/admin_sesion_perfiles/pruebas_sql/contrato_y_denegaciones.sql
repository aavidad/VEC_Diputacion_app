-- Ejecutar solo en clon efímero tras la cadena causal completa; revierte roles y efectos.
\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $acl$
DECLARE f record; t record;
BEGIN
 FOR f IN SELECT p.* FROM pg_catalog.pg_proc p WHERE p.proname IN
 ('revalidar_sesion_admin_perfiles_v1','vincular_sesion_admin_perfiles_v1','resolver_cuenta_admin_perfiles_v1','resolver_cuenta_admin_perfiles_propietaria_v1',
 'consultar_perfil_admin_perfiles_v1','exigir_runtime_admin_perfiles_v1','acreditar_runtime_admin_perfiles_v1','resolver_y_registrar_contexto_admin_perfiles_v1','reconciliar_contexto_admin_perfiles_v1') LOOP
  IF NOT f.prosecdef OR NOT (f.proconfig @> ARRAY['search_path=pg_catalog'] OR f.proconfig @> ARRAY['search_path=pg_catalog, pg_temp'])
  OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(COALESCE(f.proacl,pg_catalog.acldefault('f',f.proowner))) a WHERE a.grantee=0)
  THEN RAISE EXCEPTION 'contrato ADMIN: definidor, search_path o ACL incompatible'; END IF;
 END LOOP;
 SELECT c.* INTO STRICT t FROM pg_catalog.pg_class c WHERE c.oid='vec_identidad_sesiones_v1.sesion_admin_perfiles_v1'::regclass;
 IF NOT t.relrowsecurity OR NOT t.relforcerowsecurity OR t.relowner<>'vec_identidad_sesiones_v1_propietario'::regrole THEN RAISE EXCEPTION 'contrato ADMIN: RLS incompatible'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_trigger WHERE tgrelid=t.oid AND NOT tgisinternal AND tgname='sesion_admin_perfiles_historia') THEN RAISE EXCEPTION 'contrato ADMIN: historia mutable'; END IF;
 IF EXISTS(SELECT 1 FROM pg_catalog.aclexplode(COALESCE((SELECT proacl FROM pg_catalog.pg_proc WHERE oid='vec_identidad_sesiones_v1.revalidar_sesion_admin_perfiles_v1(text,text,text,text,text,text,text,text)'::regprocedure),pg_catalog.acldefault('f','vec_identidad_sesiones_v1_propietario'::regrole))) a WHERE a.grantee NOT IN('vec_identidad_sesiones_v1_propietario'::regrole,'vec_autorizacion_atestada_v3_propietario'::regrole) OR a.is_grantable) THEN RAISE EXCEPTION 'contrato ADMIN: consumidor excesivo'; END IF;
END $acl$;
CREATE ROLE vec_fixture_is12_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_identidad_sesiones_v1_admin_perfiles TO vec_fixture_is12_login WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
SET SESSION AUTHORIZATION vec_fixture_is12_login;
DO $runtime$
DECLARE n integer;
BEGIN
 SELECT count(*) INTO n FROM vec_contexto_actor_v1.acreditar_runtime_admin_perfiles_v1() WHERE acreditada AND identidad_login='vec_fixture_is12_login';
 IF n<>1 THEN RAISE EXCEPTION 'contrato ADMIN: runtime nominal no acreditado'; END IF;
 IF pg_catalog.has_table_privilege(session_user,'vec_identidad_sesiones_v1.sesion_admin_perfiles_v1','SELECT,INSERT,UPDATE,DELETE')
 OR pg_catalog.has_function_privilege(session_user,'vec_identidad_sesiones_v1.revalidar_sesion_admin_perfiles_v1(text,text,text,text,text,text,text,text)','EXECUTE')
 OR pg_catalog.has_function_privilege(session_user,'vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2_propietaria_admin_v1(text,text,text,text,text,text,timestamptz,text[])','EXECUTE')
 OR pg_catalog.has_function_privilege(session_user,'vec_identidad_sesiones_v1.resolver_cuenta_admin_copias_v1(text,text,text,text,text,timestamptz,timestamptz)','EXECUTE')
 THEN RAISE EXCEPTION 'contrato ADMIN: privilegio fuera de propósito'; END IF;
 SELECT count(*) INTO n FROM vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_v1('desarrollo','admin.test.invalid','fixture_is12',repeat('0',64),repeat('0',64),clock_timestamp(),clock_timestamp());
 IF n<>0 THEN RAISE EXCEPTION 'contrato ADMIN: certificado nulo admitido'; END IF;
 IF vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1('desarrollo','admin.test.invalid','fixture_is12',repeat('a',64),repeat('b',64),clock_timestamp(),clock_timestamp(),clock_timestamp()-interval '1 second',clock_timestamp()+interval '1 minute','aut_fixture_is12','ses_fixture_is12') IS DISTINCT FROM false THEN RAISE EXCEPTION 'contrato ADMIN: CRL vencida admitida'; END IF;
 IF vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1('desarrollo','admin.test.invalid','fixture_is12',repeat('a',64),repeat('b',64),clock_timestamp(),clock_timestamp(),'infinity',clock_timestamp()+interval '1 minute','aut_fixture_is12','ses_fixture_is12') IS DISTINCT FROM false THEN RAISE EXCEPTION 'contrato ADMIN: CRL infinita admitida'; END IF;
END $runtime$;
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $sin_vinculo$ BEGIN
 IF vec_identidad_sesiones_v1.revalidar_sesion_admin_perfiles_v1('aut_fixture_is12','ses_fixture_is12','cta_fixture_is12','cta_fixture_ord12','per_fixture_is12','prf_fixture_is12','pga_fixture_is12',repeat('c',64)) IS DISTINCT FROM false THEN RAISE EXCEPTION 'contrato ADMIN: sesión no vinculada admitida'; END IF;
END $sin_vinculo$;
RESET ROLE;
-- La acreditación debe cerrar al añadir una segunda membresía.
GRANT vec_identidad_sesiones_v1_admin_copias TO vec_fixture_is12_login WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
SET SESSION AUTHORIZATION vec_fixture_is12_login;
DO $mezcla$ DECLARE rechazada boolean:=false; BEGIN
 BEGIN PERFORM vec_contexto_actor_v1.acreditar_runtime_admin_perfiles_v1(); EXCEPTION WHEN insufficient_privilege THEN rechazada:=true; END;
 IF NOT rechazada THEN RAISE EXCEPTION 'contrato ADMIN: LOGIN multipropósito admitido'; END IF;
END $mezcla$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
