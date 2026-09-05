\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000056_respuesta_recibida_rrhh', 0));
LOCK TABLE vec_contratacion_temporal.respuesta_recibida_rrhh,
    vec_contratacion_temporal.historia_respuesta_recibida_rrhh,
    vec_contratacion_temporal.outbox_respuesta_recibida_rrhh IN ACCESS EXCLUSIVE MODE;
DO $conservar$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.respuesta_recibida_rrhh)
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.historia_respuesta_recibida_rrhh)
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.outbox_respuesta_recibida_rrhh) THEN
        RAISE EXCEPTION 'reversión denegada: se conserva declaración, historia y evento de respuesta' USING ERRCODE = '55000';
    END IF;
END
$conservar$;
DROP FUNCTION vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_contratacion_temporal.outbox_respuesta_recibida_rrhh;
DROP TABLE vec_contratacion_temporal.historia_respuesta_recibida_rrhh;
DROP TABLE vec_contratacion_temporal.respuesta_recibida_rrhh;
COMMIT;
