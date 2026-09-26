\set ON_ERROR_STOP on
-- Retirada de CT123. Se deniega si algún informe nuevo tras subsanar consta
-- ya reservado o emitido: su historia depende de estas funciones.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000123', 0));
DO $pre$
BEGIN
    IF pg_catalog.to_regprocedure('vec_contratacion_temporal.preparar_informe_juridico_tras_subsanacion_v1(jsonb)') IS NULL THEN
        RAISE EXCEPTION 'CT123 no instalada' USING ERRCODE = '55000';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.reserva_informe_juridico
                WHERE version_expediente <> 4) THEN
        RAISE EXCEPTION 'CT123: reversión denegada, hay informes nuevos tras subsanar' USING ERRCODE = '55000';
    END IF;
END
$pre$;
DROP FUNCTION vec_contratacion_temporal.inicio_ronda_informe_nuevo_v1(text,text);
DROP FUNCTION vec_contratacion_temporal.confirmar_informe_juridico_tras_subsanacion_v1(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.preparar_informe_juridico_tras_subsanacion_v1(jsonb);
DROP FUNCTION vec_contratacion_temporal.informe_nuevo_admisible_ct123(jsonb);
ALTER TABLE vec_contratacion_temporal.reserva_informe_juridico
    DROP CONSTRAINT reserva_informe_juridico_version_expediente_check,
    ADD CONSTRAINT reserva_informe_juridico_version_expediente_check
        CHECK (version_expediente = 4);
COMMIT;
