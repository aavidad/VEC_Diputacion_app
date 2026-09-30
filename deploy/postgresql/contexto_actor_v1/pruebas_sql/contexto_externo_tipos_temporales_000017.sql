\set ON_ERROR_STOP on
-- Solo en PostgreSQL 18 desechable, CTX15 instalada; termina en ROLLBACK.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
CREATE ROLE prueba_ctx17_candidato LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE prueba_ctx17_usuarios LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contexto_actor_v1_candidato_externo TO prueba_ctx17_candidato WITH INHERIT TRUE, SET FALSE;
GRANT vec_contexto_actor_v1_usuarios_externo TO prueba_ctx17_usuarios WITH INHERIT TRUE, SET FALSE;
-- La configuración acreditada no concede TEMP. Este permiso contaminante solo
-- prepara el adversario en el clon, para comprobar que su rechazo no ejecuta
-- primero un CHECK con la autoridad del propietario de ContextoActor.
DO $temp$ BEGIN
 EXECUTE pg_catalog.format('GRANT TEMP ON DATABASE %I TO prueba_ctx17_candidato,prueba_ctx17_usuarios',pg_catalog.current_database());
END $temp$;
SET LOCAL SESSION AUTHORIZATION prueba_ctx17_candidato;
CREATE TEMP TABLE inicializar_temporal(n integer);
CREATE FUNCTION pg_temp.sentinel_ctx17() RETURNS boolean LANGUAGE plpgsql AS $sentinel$
BEGIN
 IF current_user='vec_contexto_actor_v1_propietario' THEN
  RAISE EXCEPTION 'CTX17: CHECK temporal ejecutado como propietario' USING ERRCODE='P1700';
 END IF;
 RETURN true;
END $sentinel$;
CREATE DOMAIN pg_temp.oid AS pg_catalog.oid CHECK(pg_temp.sentinel_ctx17());
CREATE DOMAIN pg_temp.jsonb AS pg_catalog.jsonb CHECK(pg_temp.sentinel_ctx17());
CREATE DOMAIN pg_temp.text AS pg_catalog.text CHECK(pg_temp.sentinel_ctx17());
CREATE DOMAIN pg_temp.bytea AS pg_catalog.bytea CHECK(pg_temp.sentinel_ctx17());
RESET SESSION AUTHORIZATION;
DO $retirar_temp$ BEGIN
 EXECUTE pg_catalog.format('REVOKE TEMP ON DATABASE %I FROM prueba_ctx17_candidato',pg_catalog.current_database());
END $retirar_temp$;
SET LOCAL SESSION AUTHORIZATION prueba_ctx17_candidato;
DO $denegar_candidato$
BEGIN
 BEGIN
  PERFORM vec_contexto_actor_v1.acreditar_runtime_candidato_externo_v1();
  RAISE EXCEPTION 'CTX17: acreditó LOGIN contaminado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_contexto_actor_v1.resolver_contexto_candidato_externo_v1('oca_ctx17_00000000000000000001','rca_ctx17_00000000000000000001','cta_ctx17_00000000000000000001','prf_ctx17_00000000000000000001',clock_timestamp());
  RAISE EXCEPTION 'CTX17: resolvió LOGIN contaminado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_contexto_actor_v1.reconciliar_contexto_candidato_externo_v1('oca_ctx17_00000000000000000001','rca_ctx17_00000000000000000001','cta_ctx17_00000000000000000001','prf_ctx17_00000000000000000001',clock_timestamp());
  RAISE EXCEPTION 'CTX17: reconcilió LOGIN contaminado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $denegar_candidato$;
RESET SESSION AUTHORIZATION;
DROP DOMAIN pg_temp.oid,pg_temp.jsonb,pg_temp.text,pg_temp.bytea;
DROP FUNCTION pg_temp.sentinel_ctx17();
DROP TABLE pg_temp.inicializar_temporal;
SET LOCAL SESSION AUTHORIZATION prueba_ctx17_usuarios;
CREATE TEMP TABLE inicializar_temporal(n integer);
CREATE FUNCTION pg_temp.sentinel_ctx17() RETURNS boolean LANGUAGE plpgsql AS $sentinel$
BEGIN
 IF current_user='vec_contexto_actor_v1_propietario' THEN
  RAISE EXCEPTION 'CTX17: CHECK temporal ejecutado como propietario' USING ERRCODE='P1700';
 END IF;
 RETURN true;
END $sentinel$;
CREATE DOMAIN pg_temp.oid AS pg_catalog.oid CHECK(pg_temp.sentinel_ctx17());
CREATE DOMAIN pg_temp.jsonb AS pg_catalog.jsonb CHECK(pg_temp.sentinel_ctx17());
CREATE DOMAIN pg_temp.text AS pg_catalog.text CHECK(pg_temp.sentinel_ctx17());
CREATE DOMAIN pg_temp.bytea AS pg_catalog.bytea CHECK(pg_temp.sentinel_ctx17());
RESET SESSION AUTHORIZATION;
DO $retirar_temp$ BEGIN
 EXECUTE pg_catalog.format('REVOKE TEMP ON DATABASE %I FROM prueba_ctx17_usuarios',pg_catalog.current_database());
END $retirar_temp$;
SET LOCAL SESSION AUTHORIZATION prueba_ctx17_usuarios;
DO $denegar_usuarios$
BEGIN
 BEGIN
  PERFORM vec_contexto_actor_v1.acreditar_runtime_usuarios_externo_v1();
  RAISE EXCEPTION 'CTX17: acreditó LOGIN contaminado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_contexto_actor_v1.resolver_contexto_usuarios_externo_v1('oca_ctx17_00000000000000000002','rca_ctx17_00000000000000000002','cta_ctx17_00000000000000000002','prf_ctx17_00000000000000000002',clock_timestamp());
  RAISE EXCEPTION 'CTX17: resolvió LOGIN contaminado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_contexto_actor_v1.reconciliar_contexto_usuarios_externo_v1('oca_ctx17_00000000000000000002','rca_ctx17_00000000000000000002','cta_ctx17_00000000000000000002','prf_ctx17_00000000000000000002',clock_timestamp());
  RAISE EXCEPTION 'CTX17: reconcilió LOGIN contaminado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $denegar_usuarios$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
\echo CTX17-TIPOS-TEMPORALES-OK
