\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000083', 0));
DO $pre$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR pg_catalog.to_regclass('vec_bolsa_llamamientos.asociacion_oferta_ct') IS NULL
    OR pg_catalog.to_regclass('vec_bolsa_llamamientos.asociacion_oferta_ct_outbox') IS NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.asociar_oferta_ct_v1(text,integer,text,text,text,bigint,text,text,text,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'B83: preimagen distinta' USING ERRCODE='55000';
 END IF;
END $pre$;
LOCK TABLE vec_bolsa_llamamientos.asociacion_oferta_ct,
 vec_bolsa_llamamientos.asociacion_oferta_ct_outbox IN ACCESS EXCLUSIVE MODE;
DO $historia$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.asociacion_oferta_ct)
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.asociacion_oferta_ct_outbox) THEN
  RAISE EXCEPTION 'B83: no se revierte con historia' USING ERRCODE='55000';
 END IF;
END $historia$;
DROP FUNCTION vec_bolsa_llamamientos.asociar_oferta_ct_v1(
 text,integer,text,text,text,bigint,text,text,text,text,text,text,text,text,text,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
REVOKE USAGE ON SCHEMA vec_bolsa_llamamientos FROM vec_contratacion_temporal_propietario;
DROP TABLE vec_bolsa_llamamientos.asociacion_oferta_ct_outbox;
DROP TABLE vec_bolsa_llamamientos.asociacion_oferta_ct;
COMMIT;
