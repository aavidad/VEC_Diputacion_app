\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000024', 0));
DO $f$ BEGIN
 IF to_regclass('vec_bolsa_llamamientos.contrato_participacion') IS NULL THEN
  RAISE EXCEPTION 'B13 no está instalada' USING ERRCODE='55000';
 END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.contrato_participacion) THEN
  RAISE EXCEPTION 'hay historia B13; no se deshace' USING ERRCODE='55000';
 END IF;
END $f$;
DROP FUNCTION vec_bolsa_llamamientos.listar_contratos_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.cursor_contratos_participacion_v1() RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.registrar_contrato_participacion_v1(jsonb,text,timestamptz) RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.instante_contrato_valido(jsonb,boolean) RESTRICT;
DROP TABLE vec_bolsa_llamamientos.contrato_participacion RESTRICT;
COMMIT;
