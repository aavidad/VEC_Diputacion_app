\set ON_ERROR_STOP on
-- Sólo clon desechable PG18 después de U14. No instala ni reaplica SQL.
-- Comprueba la denegación real al LOGIN interno y el inventario operacional.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=session_user AND rolsuper)
 OR to_regrole('vec_u14_fachada_prueba') IS NOT NULL
 OR to_regclass('vec_usuarios_correos_externo.avisos_inbox') IS NULL
 THEN RAISE EXCEPTION 'U14 prueba requiere DBA de clon y U14 preparada'; END IF;
END $pre$;
CREATE ROLE vec_u14_fachada_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_usuarios_ejecutor_interno TO vec_u14_fachada_prueba WITH INHERIT TRUE,SET FALSE,ADMIN FALSE;
DO $cerrada$ DECLARE f regprocedure:='vec_usuarios_correos_avisos.correo_activo_avisos_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure; BEGIN
 IF NOT pg_has_role('vec_u14_fachada_prueba','vec_usuarios_ejecutor_interno','MEMBER')
 OR has_function_privilege('vec_u14_fachada_prueba',f,'EXECUTE')
 OR has_schema_privilege('vec_u14_fachada_prueba','vec_usuarios_correos_avisos','USAGE')
 THEN RAISE EXCEPTION 'U14 LOGIN interno conserva acceso a fachada externa'; END IF;
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolcanlogin AND NOT r.rolsuper
  AND pg_has_role(r.oid,'vec_usuarios_ejecutor_interno','MEMBER')
  AND (has_function_privilege(r.oid,f,'EXECUTE') OR has_schema_privilege(r.oid,'vec_usuarios_correos_avisos','USAGE')))
 THEN RAISE EXCEPTION 'U14 inventario operacional no está cerrado'; END IF;
END $cerrada$;
SET SESSION AUTHORIZATION vec_u14_fachada_prueba;
DO $ejecucion$ DECLARE denegada boolean:=false; BEGIN
 BEGIN
  PERFORM vec_usuarios_correos_avisos.correo_activo_avisos_llamamiento_v1(
   'can_aaaaaaaaaaaaaaaaaaaaaa',''::bytea,''::bytea,''::bytea,''::bytea,
   1::numeric,1::numeric,''::bytea,''::bytea,''::bytea,''::bytea);
 EXCEPTION WHEN insufficient_privilege THEN denegada:=true; END;
 IF NOT denegada THEN RAISE EXCEPTION 'U14 fachada ejecutable desde LOGIN interno'; END IF;
END $ejecucion$;
RESET SESSION AUTHORIZATION;
-- Una concesión accidental a un LOGIN operativo debe seguir detectándose.
GRANT USAGE ON SCHEMA vec_usuarios_correos_avisos TO vec_u14_fachada_prueba;
DO $anomalia$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname='vec_u14_fachada_prueba'
  AND r.rolcanlogin AND NOT r.rolsuper
  AND pg_has_role(r.oid,'vec_usuarios_ejecutor_interno','MEMBER')
  AND has_schema_privilege(r.oid,'vec_usuarios_correos_avisos','USAGE'))
 THEN RAISE EXCEPTION 'U14 concesión operacional anómala omitida'; END IF;
END $anomalia$;
ROLLBACK;
\echo U14-FACHADA-OK: LOGIN interno denegado; DBA fuera de inventario; concesión anómala detectada
