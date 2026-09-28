\set ON_ERROR_STOP on
-- B51 DOWN únicamente sin historia de lecturas; no usar con bases conservadas.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000051',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_politica_ofertas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.politica_ofertas_lectura_v3') IS NULL
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.politica_ofertas_lectura_v3)
 THEN RAISE EXCEPTION 'B51 DOWN: ausente o con historia de lectura' USING ERRCODE='55000'; END IF;
END $pre$;
DROP FUNCTION vec_bolsa_llamamientos.consultar_politica_ofertas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_bolsa_llamamientos.politica_ofertas_lectura_v3;
REVOKE EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_politica_ofertas_v1(text)
 FROM vec_bolsa_llamamientos_calculador_politica;
REVOKE USAGE ON SCHEMA vec_bolsa_llamamientos FROM vec_bolsa_llamamientos_calculador_politica;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_politica_ofertas_v1(text)
 TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
