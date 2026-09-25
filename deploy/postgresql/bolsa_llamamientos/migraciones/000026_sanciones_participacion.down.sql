\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000026:down', 0));
-- Solo para ensayo aislado: con historia de sanciones no se deshace.
DO $f$ BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.sancion_participacion') IS NULL THEN
  RAISE EXCEPTION 'estado incompatible para retirar las sanciones de Bolsa' USING ERRCODE='55000';
 END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.sancion_participacion)
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.recurso_sancion_participacion) THEN
  RAISE EXCEPTION 'hay historia de sanciones; no se deshace' USING ERRCODE='55000';
 END IF;
END $f$;
DROP FUNCTION vec_bolsa_llamamientos.listar_sanciones_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v1(text,text,text,date,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.registrar_sancion_participacion_v1(text,text,text,text,text,text,text,date,text,text,text,text,text,date,date,text,text,timestamptz,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.consumir_autorizacion_sancion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP TABLE vec_bolsa_llamamientos.recurso_sancion_participacion RESTRICT;
DROP TABLE vec_bolsa_llamamientos.sancion_participacion RESTRICT;
COMMIT;
