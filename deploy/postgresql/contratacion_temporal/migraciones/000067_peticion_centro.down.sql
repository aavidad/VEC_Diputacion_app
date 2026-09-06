\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000067',0));
LOCK TABLE vec_contratacion_temporal.peticion_centro_revision,
    vec_contratacion_temporal.peticion_centro_outbox,
    vec_contratacion_temporal.peticion_centro_acceso IN ACCESS EXCLUSIVE MODE;
DO $rechazar_historia$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.peticion_centro_revision)
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.peticion_centro_outbox)
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.peticion_centro_acceso) THEN
        RAISE EXCEPTION 'reversión denegada: existe historia de petición de centro' USING ERRCODE='55000';
    END IF;
END
$rechazar_historia$;
DROP FUNCTION vec_contratacion_temporal.consultar_peticiones_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.registrar_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_contratacion_temporal.peticion_centro_acceso;
DROP TABLE vec_contratacion_temporal.peticion_centro_outbox;
DROP TABLE vec_contratacion_temporal.peticion_centro_revision;
COMMIT;
