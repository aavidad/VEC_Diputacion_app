BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000001_preparacion_altas',
        0
    )
);

DO $proteccion$
DECLARE
    v_existe_historia boolean;
BEGIN
    IF to_regclass(
        'vec_contratacion_temporal.identidad_reserva_alta'
    ) IS NULL THEN
        RETURN;
    END IF;
    SELECT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.identidad_reserva_alta
    )
      INTO v_existe_historia;
    IF v_existe_historia AND current_setting(
        'vec.confirmar_destruccion_contratacion_temporal',
        true
    ) IS DISTINCT FROM
        'DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'down rechazado: existe historia de contratación temporal';
    END IF;
END
$proteccion$;

REVOKE EXECUTE ON FUNCTION
    vec_contratacion_temporal.preparar_alta_v1(jsonb)
    FROM vec_contratacion_temporal_ejecutor;
REVOKE USAGE ON SCHEMA vec_contratacion_temporal
    FROM vec_contratacion_temporal_ejecutor;

DROP FUNCTION vec_contratacion_temporal.preparar_alta_v1(jsonb) RESTRICT;
DROP TABLE vec_contratacion_temporal.reserva_alta_actual RESTRICT;
DROP TRIGGER reserva_alta_version_inmutable
    ON vec_contratacion_temporal.reserva_alta_version;
DROP TABLE vec_contratacion_temporal.reserva_alta_version RESTRICT;
DROP TRIGGER identidad_reserva_alta_inmutable
    ON vec_contratacion_temporal.identidad_reserva_alta;
DROP TABLE vec_contratacion_temporal.identidad_reserva_alta RESTRICT;
DROP FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1()
    RESTRICT;
DROP SCHEMA vec_contratacion_temporal RESTRICT;

COMMIT;
