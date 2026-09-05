\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000058_resolucion_manual_respuesta_rrhh',0));
LOCK TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh IN ACCESS EXCLUSIVE MODE;
DO $conservar$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh) THEN
        RAISE EXCEPTION 'reversión denegada: se conserva resolución manual, auditoría y recibo' USING ERRCODE='55000';
    END IF;
END
$conservar$;
-- Sin CASCADE, ni borrado de declaración, comunicación o consumos V3.
DROP FUNCTION vec_contratacion_temporal.registrar_resolucion_manual_respuesta_rrhh_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh;
COMMIT;
