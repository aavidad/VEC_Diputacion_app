\set ON_ERROR_STOP on
-- Reversión solo para una base desechable sin capturas.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000052',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.captura_cese_b10)
    THEN
  RAISE EXCEPTION 'Bolsa 000052: hay historia conservada' USING ERRCODE='55000';
 END IF;
END $pre$;
DROP FUNCTION vec_bolsa_llamamientos.capturar_cese_b10_v1(bigint,text,text,text,text);
DROP TABLE vec_bolsa_llamamientos.captura_cese_b10;
COMMIT;
