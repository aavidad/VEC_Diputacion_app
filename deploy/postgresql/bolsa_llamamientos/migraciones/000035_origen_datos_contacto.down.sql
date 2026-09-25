\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000035', 0));
DO $f$ BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR to_regclass('vec_bolsa_llamamientos.origen_datos_contacto_participacion') IS NULL THEN
  RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='estado incompatible para retirar el contacto de origen CONVOCA';
 END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.origen_datos_contacto_participacion) THEN RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='hay contactos de origen CONVOCA; no se deshace'; END IF; END $f$;
DROP FUNCTION vec_bolsa_llamamientos.leer_origen_datos_contacto_participacion_v1(text,bigint) RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_origen_participacion_v1(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,timestamptz,date,text,text) RESTRICT;
DROP TABLE vec_bolsa_llamamientos.origen_datos_contacto_participacion RESTRICT;
COMMIT;
