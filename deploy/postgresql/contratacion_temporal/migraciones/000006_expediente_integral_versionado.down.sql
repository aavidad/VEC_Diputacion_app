BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000006_expediente_integral_versionado', 0
    )
);

DO $proteccion$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.expediente_version_integral
    )
    AND pg_catalog.current_setting(
        'vec.confirmar_destruccion_contratacion_temporal', true
    ) IS DISTINCT FROM
        'DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada del expediente integral protegida';
    END IF;
END
$proteccion$;

DROP TRIGGER expediente_alta_version_materializar_integral
    ON vec_contratacion_temporal.expediente_alta_version;
DROP TRIGGER expediente_version_integral_inmutable
    ON vec_contratacion_temporal.expediente_version_integral;
DROP FUNCTION vec_contratacion_temporal.materializar_alta_integral_v1();
DROP FUNCTION vec_contratacion_temporal.materializar_version_inicial_v1(
    text, numeric, bytea, text, numeric, text, text, text, timestamptz
);
DROP TABLE vec_contratacion_temporal.expediente_integral_actual;
DROP TABLE vec_contratacion_temporal.expediente_version_integral;

COMMIT;
