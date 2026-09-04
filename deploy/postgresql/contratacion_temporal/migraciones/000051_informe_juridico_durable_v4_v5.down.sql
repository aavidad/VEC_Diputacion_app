BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000051_informe_juridico_v4_v5', 0
    )
);

DO $proteccion$
BEGIN
    IF EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.reserva_informe_juridico
       )
       OR EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.documento_informe_juridico_desarrollo
       )
       OR EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.terminal_informe_juridico
       )
       OR EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.expediente_version_integral
            WHERE origen_version = 'informe_juridico_o5'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'rollback de informe jurídico rechazado con datos';
    END IF;
END
$proteccion$;

DROP FUNCTION vec_contratacion_temporal.confirmar_informe_juridico_v1(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea
);
DROP FUNCTION vec_contratacion_temporal.preparar_informe_juridico_v1(jsonb);
DROP FUNCTION vec_contratacion_temporal.recibo_informe_juridico_v1(text);
DROP FUNCTION vec_contratacion_temporal.informe_juridico_claves_exactas_v1(
    jsonb,text[]
);
DROP TABLE vec_contratacion_temporal.terminal_informe_juridico;
DROP TABLE vec_contratacion_temporal.documento_informe_juridico_desarrollo;
DROP TABLE vec_contratacion_temporal.reserva_informe_juridico;

ALTER TABLE vec_contratacion_temporal.expediente_version_integral
    DROP CONSTRAINT expediente_version_integral_origen_version_check;
ALTER TABLE vec_contratacion_temporal.expediente_version_integral
    ADD CONSTRAINT expediente_version_integral_origen_version_check
    CHECK (origen_version IN (
        'alta_o2', 'analisis_o3', 'cobertura_o4', 'asignacion_o5'
    ));

COMMIT;
