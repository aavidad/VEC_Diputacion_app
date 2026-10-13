\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000110',0));
DO $historia$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.origen_llamamiento_ct_v1)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.origen_llamamiento_ct_outbox_v1) THEN
  RAISE EXCEPTION 'B110 DOWN denegado con historia de origen CT' USING ERRCODE='55000';
 END IF;
END $historia$;
DROP FUNCTION vec_bolsa_llamamientos.reservar_llamamiento_ct_v1(
 text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,
 numeric,numeric,bytea,bytea,bytea,bytea,jsonb);
DROP FUNCTION vec_bolsa_llamamientos.recuperar_origen_llamamiento_ct_v1(text,text);
DROP FUNCTION vec_bolsa_llamamientos.recuperar_lectura_origen_ct_v1(text,text);
DROP FUNCTION vec_bolsa_llamamientos.recuperar_reserva_llamamiento_ct_v1(text,text);
DROP FUNCTION vec_bolsa_llamamientos.resolver_bolsa_origen_ct_v1(text,text,text,text);
DROP FUNCTION vec_bolsa_llamamientos.leer_origen_llamamiento_ct_v1(text);
DROP TABLE vec_bolsa_llamamientos.origen_llamamiento_ct_outbox_v1;
DROP TABLE vec_bolsa_llamamientos.origen_llamamiento_ct_v1;
COMMIT;
