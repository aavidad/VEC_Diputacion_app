\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:migracion:000022',0));
DO $guarda$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.to_regclass('vec_identidad_sesiones_v1.sesion_admin_copias_v1') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_perfiles_v1(text,text,text,timestamptz)') IS NOT NULL
 OR pg_catalog.to_regclass('vec_identidad_sesiones_v1.sesion_admin_perfiles_v1') IS NOT NULL
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_admin_copias_v1)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE roleid='vec_identidad_sesiones_v1_admin_copias'::regrole)
 THEN RAISE EXCEPTION 'CA22: consumidores o historia conservados, retirada denegada' USING ERRCODE='55000'; END IF;
END $guarda$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DROP FUNCTION vec_contexto_actor_v1.acreditar_runtime_admin_copias_v1();
DROP FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_copias_v1(text,text,text,timestamptz);
DROP FUNCTION vec_contexto_actor_v1.reconciliar_contexto_admin_copias_v1(text,text,text,timestamptz);
DROP FUNCTION vec_contexto_actor_v1.exigir_runtime_admin_copias_v1();
DROP FUNCTION vec_contexto_actor_v1.consultar_perfil_admin_copias_por_actor_v1(text,text);
DROP FUNCTION vec_contexto_actor_v1.consultar_perfil_admin_copias_v1(text);
DROP FUNCTION vec_contexto_actor_v1.provisionar_perfil_admin_copias_v1(text,text,text,text,text,timestamptz,text,text,text);
DROP FUNCTION vec_contexto_actor_v1.revocar_perfil_admin_copias_v1(text,text,text,text,text,text,text);
DROP FUNCTION vec_contexto_actor_v1.preimagen_perfil_admin_copias_v1(text,text,text,text);
DO $restaurar$ DECLARE p record;BEGIN
 FOR p IN SELECT * FROM vec_contexto_actor_v1.preimagen_funciones_admin_copias_v1 LOOP
 IF p.sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.definicion,'UTF8')),'hex') THEN RAISE EXCEPTION 'CA22: preimagen divergente' USING ERRCODE='55000'; END IF;
 EXECUTE p.definicion;
 END LOOP;
END $restaurar$;
DROP FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2_propietaria_admin_v1(text,text,text,text,text,text,timestamptz,text[]);
DROP FUNCTION vec_contexto_actor_v1.reconciliar_contexto_actor_v2_propietaria_admin_v1(text,text,text,text,text,text,timestamptz,text[]);
DO $acl$ BEGIN IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.preimagen_funciones_admin_copias_v1 WHERE NOT uso_identidad_previo) THEN REVOKE USAGE ON SCHEMA vec_contexto_actor_v1 FROM vec_identidad_sesiones_v1_propietario; END IF; IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.preimagen_funciones_admin_copias_v1 WHERE NOT uso_autorizacion_previo) THEN REVOKE USAGE ON SCHEMA vec_contexto_actor_v1 FROM vec_autorizacion_propietario; END IF; END $acl$;
DROP TABLE vec_contexto_actor_v1.perfil_admin_copias_actual_v1;
DROP TABLE vec_contexto_actor_v1.perfil_admin_copias_v1;
DROP TABLE vec_contexto_actor_v1.preimagen_funciones_admin_copias_v1;
REVOKE USAGE ON SCHEMA vec_contexto_actor_v1 FROM vec_identidad_sesiones_v1_admin_copias;
RESET ROLE;
DO $conexion$ BEGIN EXECUTE pg_catalog.format('REVOKE CONNECT ON DATABASE %I FROM vec_identidad_sesiones_v1_admin_copias',current_database()); END $conexion$;
DROP ROLE vec_identidad_sesiones_v1_admin_copias;
COMMIT;
