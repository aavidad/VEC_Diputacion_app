\set ON_ERROR_STOP on
-- CT120 DOWN: solo sin historia. Con alguna fiscalización de un expediente
-- modificado ya confirmada no se deshace: su reserva, su terminal y la
-- versión que produjo dependen de estas funciones para recuperarse.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000120',0));
DO $pre$
BEGIN
    IF pg_catalog.to_regprocedure('vec_contratacion_temporal.preparar_fiscalizacion_v2(jsonb)') IS NULL THEN
        RAISE EXCEPTION 'CT120 no instalada' USING ERRCODE='55000';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.reserva_fiscalizacion r
                WHERE vec_contratacion_temporal.es_antecedente_fiscalizacion_modificacion_ct120(r.expediente_anterior_json)) THEN
        RAISE EXCEPTION 'reversión denegada: hay fiscalizaciones de expedientes modificados' USING ERRCODE='55000';
    END IF;
END
$pre$;
DROP FUNCTION vec_contratacion_temporal.confirmar_fiscalizacion_v2(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.preparar_fiscalizacion_v2(jsonb);
DROP FUNCTION vec_contratacion_temporal.confirmar_fiscalizacion_tras_modificacion_ct120(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.preparar_fiscalizacion_tras_modificacion_ct120(jsonb);
DROP FUNCTION vec_contratacion_temporal.antecedente_fiscalizacion_modificacion_ct120(jsonb);
DROP FUNCTION vec_contratacion_temporal.es_antecedente_fiscalizacion_modificacion_ct120(jsonb);
COMMIT;
