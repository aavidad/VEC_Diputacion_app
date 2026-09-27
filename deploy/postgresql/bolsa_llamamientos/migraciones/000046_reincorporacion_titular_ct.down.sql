\set ON_ERROR_STOP on
-- Bolsa 000046 DOWN: permitido solo antes de recibir retornos del titular.
-- La historia de 2.05 y sus recibos no se eliminan ni se reconstruyen.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000046', 0));
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $guardia$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.reincorporacion_titular_ct') IS NULL THEN
  RAISE EXCEPTION 'Bolsa 000046 DOWN: no instalada o rol incompatible' USING ERRCODE='55000';
 END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.reincorporacion_titular_ct) THEN
  RAISE EXCEPTION 'Bolsa 000046 DOWN: historia de reincorporaciones conservada' USING ERRCODE='55000';
 END IF;
END $guardia$;
DROP FUNCTION vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.cursor_reincorporaciones_titular_ct_v1() RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.registrar_reincorporacion_titular_ct_v1(text,text,bigint) RESTRICT;
DROP TABLE vec_bolsa_llamamientos.reincorporacion_titular_ct RESTRICT;
COMMIT;
