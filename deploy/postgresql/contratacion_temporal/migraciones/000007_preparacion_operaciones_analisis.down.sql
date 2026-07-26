BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000007_preparacion_analisis', 0
    )
);

DO $proteccion$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.reserva_operacion_analisis
    )
    AND pg_catalog.current_setting(
        'vec.confirmar_destruccion_contratacion_temporal', true
    ) IS DISTINCT FROM
        'DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada de operaciones de análisis protegida';
    END IF;
END
$proteccion$;

DROP FUNCTION
    vec_contratacion_temporal.consultar_operacion_analisis_v1(jsonb);
DROP FUNCTION
    vec_contratacion_temporal.preparar_operacion_analisis_v1(jsonb);
DROP TABLE vec_contratacion_temporal.alias_consulta_operacion_analisis;
DROP TABLE vec_contratacion_temporal.confirmacion_operacion_analisis;
DROP TABLE vec_contratacion_temporal.reserva_operacion_analisis_actual;
DROP TABLE vec_contratacion_temporal.reserva_operacion_analisis_version;
DROP TABLE vec_contratacion_temporal.alias_operacion_analisis;
DROP TABLE vec_contratacion_temporal.reserva_operacion_analisis;

COMMIT;
