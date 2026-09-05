\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:000054_comunicacion_llamamiento', 0));
LOCK TABLE vec_contratacion_temporal.comunicacion_llamamiento_local,
    vec_contratacion_temporal.historia_comunicacion_llamamiento_local,
    vec_contratacion_temporal.outbox_comunicacion_llamamiento_local IN ACCESS EXCLUSIVE MODE;
DO $proteccion$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.comunicacion_llamamiento_local)
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.historia_comunicacion_llamamiento_local)
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.outbox_comunicacion_llamamiento_local) THEN
        RAISE EXCEPTION 'reversión protegida: existe historia de comunicación' USING ERRCODE = '55000';
    END IF;
END
$proteccion$;
DROP FUNCTION vec_contratacion_temporal.registrar_comunicacion_llamamiento_local_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_contratacion_temporal.outbox_comunicacion_llamamiento_local;
DROP TABLE vec_contratacion_temporal.historia_comunicacion_llamamiento_local;
DROP TABLE vec_contratacion_temporal.comunicacion_llamamiento_local;
COMMIT;
