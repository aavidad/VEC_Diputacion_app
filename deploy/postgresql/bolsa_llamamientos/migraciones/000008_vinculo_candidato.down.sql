\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000008', 0)
);
DO $comprobacion$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.vinculo_candidato) THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'hay vinculos de candidato registrados; no se deshace';
    END IF;
END
$comprobacion$;
DROP FUNCTION vec_bolsa_llamamientos.listar_participaciones_candidato_v1(text) RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(text, jsonb, timestamptz) RESTRICT;
DROP TABLE vec_bolsa_llamamientos.vinculo_candidato RESTRICT;
COMMIT;
