\set ON_ERROR_STOP on
-- B55 DOWN sólo sin historia de lecturas; prohibido sobre bases conservadas.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000055',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.reincorporacion_titular_lectura_v3') IS NULL
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.reincorporacion_titular_lectura_v3)
 THEN RAISE EXCEPTION 'B55 DOWN: ausente o con historia de lectura' USING ERRCODE='55000'; END IF;
END $pre$;
DROP FUNCTION vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_bolsa_llamamientos.reincorporacion_titular_lectura_v3;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
