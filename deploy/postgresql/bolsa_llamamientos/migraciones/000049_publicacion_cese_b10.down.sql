\set ON_ERROR_STOP on
-- Solo se revierte una instalación sin ninguna confirmación pública.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000049',0));
DO $pre$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
    OR to_regclass('vec_bolsa_llamamientos.publicacion_cese_b10') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_publicador_cese') THEN
  RAISE EXCEPTION 'Bolsa 000049 DOWN: sesión o instalación incompatible' USING ERRCODE='55000';
 END IF;
 -- La comprobación de historia y el DROP deben excluir un ACK concurrente.
 -- ACCESS SHARE de SELECT por sí solo permitiría insertar tras el conteo.
 LOCK TABLE vec_bolsa_llamamientos.publicacion_cese_b10 IN ACCESS EXCLUSIVE MODE;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.publicacion_cese_b10)
    OR EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid='vec_bolsa_llamamientos_publicador_cese'::regrole) THEN
  RAISE EXCEPTION 'Bolsa 000049 DOWN: historia o membresía conservada' USING ERRCODE='55000';
 END IF;
END $pre$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DROP FUNCTION vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1(bigint,text,text,text);
DROP FUNCTION vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1();
DROP TABLE vec_bolsa_llamamientos.publicacion_cese_b10;
DROP INDEX vec_bolsa_llamamientos.restriccion_cese_b10_feed;
REVOKE USAGE ON SCHEMA vec_bolsa_llamamientos FROM vec_bolsa_llamamientos_publicador_cese;
RESET ROLE;
DO $fin$ BEGIN EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM vec_bolsa_llamamientos_publicador_cese',current_database()); END $fin$;
DROP ROLE vec_bolsa_llamamientos_publicador_cese;
COMMIT;
