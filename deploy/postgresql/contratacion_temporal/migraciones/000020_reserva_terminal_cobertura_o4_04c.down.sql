BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:o4_04:migraciones', 0
    )
);

SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control
 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.control_migracion_cobertura_o4
            WHERE control AND version_esquema = 1
       )
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.o404c_carga_terminal_v1(text)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.preparar_operacion_decision_cobertura_v1(jsonb,jsonb)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.consultar_operacion_decision_cobertura_confirmada_v1(jsonb)'
       ) IS NOT NULL
       OR EXISTS (
           SELECT 1
             FROM
               vec_contratacion_temporal.reserva_operacion_decision_cobertura
       )
       OR EXISTS (
           SELECT 1
             FROM
               vec_contratacion_temporal.alias_operacion_decision_cobertura
       )
       OR EXISTS (
           SELECT 1
             FROM
               vec_contratacion_temporal.reserva_operacion_decision_cobertura_version
       )
       OR EXISTS (
           SELECT 1
             FROM
               vec_contratacion_temporal.reserva_operacion_decision_cobertura_actual
       )
       OR EXISTS (
           SELECT 1
             FROM
               vec_contratacion_temporal.confirmacion_operacion_decision_cobertura
       )
       OR EXISTS (
           SELECT 1
             FROM
               vec_contratacion_temporal.terminal_operacion_decision_cobertura
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE =
                'retirada O4-04C rechazada: barrera o datos durables';
    END IF;
END
$prevalidacion$;

DROP TABLE vec_contratacion_temporal.terminal_operacion_decision_cobertura;
DROP TABLE
    vec_contratacion_temporal.confirmacion_operacion_decision_cobertura;
DROP TABLE
    vec_contratacion_temporal.reserva_operacion_decision_cobertura_actual;
DROP TABLE
    vec_contratacion_temporal.reserva_operacion_decision_cobertura_version;
DROP TABLE vec_contratacion_temporal.alias_operacion_decision_cobertura;
DROP TABLE vec_contratacion_temporal.reserva_operacion_decision_cobertura;
DROP TABLE vec_contratacion_temporal.control_migracion_cobertura_o4;
DROP FUNCTION
vec_contratacion_temporal.o404c_referencia_derivada_v1(text, text);

COMMIT;
