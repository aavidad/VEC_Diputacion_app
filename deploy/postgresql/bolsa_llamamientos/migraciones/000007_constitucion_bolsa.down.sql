-- Reversión de CT-000007 (Bolsa). Se deniega si existe alguna constitución:
-- las bolsas constituidas son actos administrativos y no se borran.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000007', 0));
DO $preservar$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.constitucion) THEN
        RAISE EXCEPTION 'reversión denegada: existen bolsas constituidas' USING ERRCODE = '55000';
    END IF;
END
$preservar$;
DROP FUNCTION vec_bolsa_llamamientos.listar_entradas_constitucion_v1(text, bigint);
DROP FUNCTION vec_bolsa_llamamientos.listar_constituciones_v1();
DROP FUNCTION vec_bolsa_llamamientos.constituir_bolsa_v1(
    text, text, text, text, bigint, bytea, timestamptz, text, bigint, bytea, timestamptz, timestamptz, jsonb, timestamptz
);
DROP TABLE vec_bolsa_llamamientos.constitucion_entrada;
DROP TABLE vec_bolsa_llamamientos.constitucion;
DROP TABLE vec_bolsa_llamamientos.instantanea_orden_bolsa;
DROP TABLE vec_bolsa_llamamientos.bolsa_constituida;
DROP FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
DROP FUNCTION vec_bolsa_llamamientos.constitucion_texto_valido(text, integer);
COMMIT;
