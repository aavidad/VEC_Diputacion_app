\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
DO $f$ BEGIN
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.operacion_situacion_participacion) THEN
  RAISE EXCEPTION 'hay historia B8; no se deshace' USING ERRCODE='55000';
 END IF;
END $f$;
DROP FUNCTION vec_bolsa_llamamientos.listar_operaciones_situacion_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v1(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP TABLE vec_bolsa_llamamientos.operacion_situacion_participacion RESTRICT;
COMMIT;
