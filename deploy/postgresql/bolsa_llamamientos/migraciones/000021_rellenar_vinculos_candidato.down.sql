\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000021', 0));
DO $proteccion$
BEGIN
  IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.vinculo_candidato) THEN
    RAISE EXCEPTION 'retirada 000021 denegada: existen vinculos con historia' USING ERRCODE='55000';
  END IF;
END $proteccion$;
DROP FUNCTION vec_bolsa_llamamientos.rellenar_vinculos_candidato_v1(text,jsonb,timestamptz);
COMMIT;
