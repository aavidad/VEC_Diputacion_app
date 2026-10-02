-- BORRADOR: solo retirada efímera, después de IS12 DOWN sin historia.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:migracion:000023',0));
DO $guarda$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.to_regclass('vec_identidad_sesiones_v1.sesion_admin_perfiles_v1') IS NOT NULL
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE roleid='vec_identidad_sesiones_v1_admin_perfiles'::regrole)
 THEN RAISE EXCEPTION 'CA23: consumidores conservados, retirada denegada' USING ERRCODE='55000'; END IF;
END $guarda$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DROP FUNCTION vec_contexto_actor_v1.acreditar_runtime_admin_perfiles_v1();
DROP FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_perfiles_v1(text,text,text,timestamptz);
DROP FUNCTION vec_contexto_actor_v1.reconciliar_contexto_admin_perfiles_v1(text,text,text,timestamptz);
DROP FUNCTION vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1();
DROP FUNCTION vec_contexto_actor_v1.consultar_perfil_admin_perfiles_v1(text);
REVOKE USAGE ON SCHEMA vec_contexto_actor_v1 FROM vec_identidad_sesiones_v1_admin_perfiles;
RESET ROLE;
DO $conexion$ BEGIN EXECUTE pg_catalog.format('REVOKE CONNECT ON DATABASE %I FROM vec_identidad_sesiones_v1_admin_perfiles',current_database()); END $conexion$;
DROP ROLE vec_identidad_sesiones_v1_admin_perfiles;
COMMIT;
