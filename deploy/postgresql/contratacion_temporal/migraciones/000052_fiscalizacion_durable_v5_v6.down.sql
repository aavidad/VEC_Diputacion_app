BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000052_fiscalizacion_v5_v6', 0
    )
);

DO $proteccion$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.reserva_fiscalizacion
    ) OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.terminal_fiscalizacion
    ) OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.retorno_fiscalizacion_unidad
    ) OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.expediente_version_integral
         WHERE origen_version = 'fiscalizacion_o5'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE =
                'rollback fiscalización v5-v6 bloqueado por historia durable';
    END IF;
END
$proteccion$;

DROP FUNCTION vec_contratacion_temporal.confirmar_fiscalizacion_v1(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,
    bytea,bytea,bytea,bytea
);
DROP FUNCTION vec_contratacion_temporal.preparar_fiscalizacion_v1(jsonb);
DROP FUNCTION vec_contratacion_temporal.recibo_fiscalizacion_v1(text);
DROP FUNCTION vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
    jsonb,text[]
);

DROP TABLE vec_contratacion_temporal.retorno_fiscalizacion_unidad;
DROP TABLE vec_contratacion_temporal.terminal_fiscalizacion;
DROP TABLE vec_contratacion_temporal.reserva_fiscalizacion;

ALTER TABLE vec_contratacion_temporal.expediente_version_integral
    DROP CONSTRAINT expediente_version_integral_origen_version_check;
ALTER TABLE vec_contratacion_temporal.expediente_version_integral
    ADD CONSTRAINT expediente_version_integral_origen_version_check
    CHECK (origen_version IN (
        'alta_o2', 'analisis_o3', 'cobertura_o4', 'asignacion_o5',
        'informe_juridico_o5'
    ));

ALTER TABLE vec_contratacion_temporal.expediente_version_integral
    DROP CONSTRAINT expediente_version_integral_estado_check;
ALTER TABLE vec_contratacion_temporal.expediente_version_integral
    ADD CONSTRAINT expediente_version_integral_estado_check
    CHECK (estado IN ('en_curso', 'completado', 'cancelado'));

COMMIT;
