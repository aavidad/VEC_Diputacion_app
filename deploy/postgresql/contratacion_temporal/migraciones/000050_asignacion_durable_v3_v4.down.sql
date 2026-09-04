BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000050_asignacion_durable_v3_v4', 0
    )
);

DO $proteccion$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.reserva_asignacion)
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.terminal_asignacion) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'rollback de asignacion rechazado con datos';
    END IF;
END
$proteccion$;

DROP FUNCTION vec_contratacion_temporal.confirmar_asignacion_v1(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea
);
DROP FUNCTION vec_contratacion_temporal.consultar_asignacion_v1(jsonb);
DROP FUNCTION vec_contratacion_temporal.preparar_asignacion_v1(jsonb);
DROP FUNCTION vec_contratacion_temporal.asignacion_claves_exactas_v1(jsonb,text[]);
DROP TABLE vec_contratacion_temporal.terminal_asignacion;
DROP TABLE vec_contratacion_temporal.reserva_asignacion;

COMMIT;
