\set ON_ERROR_STOP on
-- Sólo objetivo fresco con AUT42 final y plan/aprobación externos vigentes.
-- El DBA suministra los GUC de bytes exactos y LOGIN real aprobado.
-- No configura una aprobación favorable ni instala/reaplica SQL.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
SELECT current_setting('vec.ensayo.operador_mantenimiento') AS operador_mantenimiento \gset
SET SESSION AUTHORIZATION :"operador_mantenimiento";
DO $control$
DECLARE p text:=current_setting('vec.ensayo.plan_mantenimiento');r jsonb;
BEGIN
 r:=vec_autorizacion.mantener_version_perfil_fijo_admin_v1(p,encode(sha256(convert_to(p,'UTF8')),'hex'));
 IF r->>'estado' IS DISTINCT FROM 'permitido' OR r->'recibo' IS NULL OR r->'recibo'='null'::jsonb THEN
  RAISE EXCEPTION 'AUT42 prueba: PARO clave=baseline_ACL actual=no_permitido esperado=plan_real_aprobado_vigente';
 END IF;
 PERFORM set_config('vec.ensayo.recibo_mantenimiento_sha256',encode(sha256(convert_to((r->'recibo')::text,'UTF8')),'hex'),true);
END $control$;
RESET SESSION AUTHORIZATION;

SAVEPOINT acl_conexion;
DO $grant$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_admin_mantenimiento_fijo_ejecutor WITH GRANT OPTION',current_database()); END $grant$;
SET SESSION AUTHORIZATION :"operador_mantenimiento";
DO $rechazo$
DECLARE p text:=current_setting('vec.ensayo.plan_mantenimiento');r jsonb;
BEGIN
 r:=vec_autorizacion.mantener_version_perfil_fijo_admin_v1(p,encode(sha256(convert_to(p,'UTF8')),'hex'));
 IF r->>'estado' IS DISTINCT FROM 'denegado' OR r->>'codigo' IS DISTINCT FROM 'mantenimiento_rechazado' OR r->'recibo' IS DISTINCT FROM 'null'::jsonb OR r->>'replay' IS DISTINCT FROM 'false' OR r#>>'{auditoria_intento,auditoria_ref}' IS NULL THEN
  RAISE EXCEPTION 'AUT42 prueba: PARO clave=ACL_conexion actual=sin_rechazo_auditado esperado=GRANT_OPTION_denegado';
 END IF;
END $rechazo$;
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT acl_conexion;

SAVEPOINT acl_esquema;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_mantenimiento_fijo_ejecutor WITH GRANT OPTION;
SET SESSION AUTHORIZATION :"operador_mantenimiento";
DO $rechazo$
DECLARE p text:=current_setting('vec.ensayo.plan_mantenimiento');r jsonb;
BEGIN
 r:=vec_autorizacion.mantener_version_perfil_fijo_admin_v1(p,encode(sha256(convert_to(p,'UTF8')),'hex'));
 IF r->>'estado' IS DISTINCT FROM 'denegado' OR r->>'codigo' IS DISTINCT FROM 'mantenimiento_rechazado' OR r->'recibo' IS DISTINCT FROM 'null'::jsonb OR r->>'replay' IS DISTINCT FROM 'false' OR r#>>'{auditoria_intento,auditoria_ref}' IS NULL THEN
  RAISE EXCEPTION 'AUT42 prueba: PARO clave=ACL_esquema actual=sin_rechazo_auditado esperado=GRANT_OPTION_denegado';
 END IF;
END $rechazo$;
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT acl_esquema;

SAVEPOINT acl_fachada;
GRANT EXECUTE ON FUNCTION vec_autorizacion.mantener_version_perfil_fijo_admin_v1(text,text) TO vec_admin_mantenimiento_fijo_ejecutor WITH GRANT OPTION;
SET SESSION AUTHORIZATION :"operador_mantenimiento";
DO $rechazo$
DECLARE p text:=current_setting('vec.ensayo.plan_mantenimiento');r jsonb;
BEGIN
 r:=vec_autorizacion.mantener_version_perfil_fijo_admin_v1(p,encode(sha256(convert_to(p,'UTF8')),'hex'));
 IF r->>'estado' IS DISTINCT FROM 'denegado' OR r->>'codigo' IS DISTINCT FROM 'mantenimiento_rechazado' OR r->'recibo' IS DISTINCT FROM 'null'::jsonb OR r->>'replay' IS DISTINCT FROM 'false' OR r#>>'{auditoria_intento,auditoria_ref}' IS NULL THEN
  RAISE EXCEPTION 'AUT42 prueba: PARO clave=ACL_fachada actual=sin_rechazo_auditado esperado=GRANT_OPTION_denegado';
 END IF;
END $rechazo$;
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT acl_fachada;

SET SESSION AUTHORIZATION :"operador_mantenimiento";
DO $recuperacion$
DECLARE p text:=current_setting('vec.ensayo.plan_mantenimiento');r jsonb;
BEGIN
 r:=vec_autorizacion.mantener_version_perfil_fijo_admin_v1(p,encode(sha256(convert_to(p,'UTF8')),'hex'));
 IF r->>'estado' IS DISTINCT FROM 'permitido' OR r->>'replay' IS DISTINCT FROM 'true'
 OR encode(sha256(convert_to((r->'recibo')::text,'UTF8')),'hex') IS DISTINCT FROM current_setting('vec.ensayo.recibo_mantenimiento_sha256') THEN
  RAISE EXCEPTION 'AUT42 prueba: PARO clave=ACL_recuperacion actual=divergente esperado=replay_original_con_ACL_normales';
 END IF;
END $recuperacion$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
