BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000008_efectos_integrales', 0
    )
);

DO $proteccion$
BEGIN
    IF (
        EXISTS (
            SELECT 1 FROM
            vec_contratacion_temporal.actuacion_expediente_integral
        )
        OR EXISTS (
            SELECT 1 FROM
            vec_contratacion_temporal.consumo_fuentes_analisis
        )
        OR EXISTS (
            SELECT 1 FROM
            vec_contratacion_temporal.consumo_decision_analisis
        )
        OR EXISTS (
            SELECT 1 FROM
            vec_contratacion_temporal.auditoria_expediente_integral
        )
        OR EXISTS (
            SELECT 1 FROM
            vec_contratacion_temporal.outbox_expediente_integral
        )
    ) AND pg_catalog.current_setting(
        'vec.confirmar_destruccion_contratacion_temporal', true
    ) IS DISTINCT FROM
        'DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada de efectos integrales protegida';
    END IF;
END
$proteccion$;

DROP TABLE vec_contratacion_temporal.outbox_expediente_integral;
DROP TABLE vec_contratacion_temporal.auditoria_expediente_integral;
DROP TABLE
    vec_contratacion_temporal.control_cadenas_expediente_integral;
DROP TABLE vec_contratacion_temporal.consumo_decision_analisis;
DROP TABLE vec_contratacion_temporal.consumo_fuentes_analisis;
DROP TABLE vec_contratacion_temporal.actuacion_expediente_integral;

COMMIT;
