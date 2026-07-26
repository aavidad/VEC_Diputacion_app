BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000002_rotacion_hmac',
        0
    )
);

DO $proteccion$
DECLARE
    v_existe_historia boolean;
BEGIN
    IF to_regprocedure(
        'vec_contratacion_temporal.preparar_alta_v2(jsonb)'
    ) IS NULL THEN
        RETURN;
    END IF;
    SELECT EXISTS (
        SELECT 1
        FROM vec_contratacion_temporal.identidad_reserva_alta
    ) INTO v_existe_historia;
    IF v_existe_historia AND current_setting(
        'vec.confirmar_destruccion_contratacion_temporal',
        true
    ) IS DISTINCT FROM
        'DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'down v2 rechazado: existe historia de contratación temporal';
    END IF;
END
$proteccion$;

REVOKE EXECUTE ON FUNCTION
    vec_contratacion_temporal.preparar_alta_v2(jsonb)
    FROM vec_contratacion_temporal_ejecutor;

-- Una reversión con historia solo se admite mediante la frase destructiva
-- explícita. PostgreSQL no puede reconstruir los sellos v1 de altas nacidas
-- tras la rotación; por ello no existe una degradación silenciosa.
TRUNCATE TABLE
    vec_contratacion_temporal.reserva_alta_actual,
    vec_contratacion_temporal.reserva_alta_version,
    vec_contratacion_temporal.alias_huella_alta,
    vec_contratacion_temporal.alias_ambito_alta,
    vec_contratacion_temporal.identidad_reserva_alta;

DROP FUNCTION vec_contratacion_temporal.preparar_alta_v2(jsonb) RESTRICT;
DROP TRIGGER alias_huella_alta_inmutable
    ON vec_contratacion_temporal.alias_huella_alta;
DROP TABLE vec_contratacion_temporal.alias_huella_alta RESTRICT;
DROP TRIGGER alias_ambito_alta_inmutable
    ON vec_contratacion_temporal.alias_ambito_alta;
DROP TABLE vec_contratacion_temporal.alias_ambito_alta RESTRICT;
DROP TABLE
    vec_contratacion_temporal.politica_generaciones_hmac_alta
    RESTRICT;

ALTER TABLE vec_contratacion_temporal.identidad_reserva_alta
    DROP CONSTRAINT identidad_reserva_ambito_hmac_valido,
    DROP CONSTRAINT identidad_reserva_huella_peticion_valida;
ALTER TABLE vec_contratacion_temporal.identidad_reserva_alta
    ADD CONSTRAINT identidad_reserva_ambito_hmac_valido CHECK (
        ambito_hmac ~ (
            '^hmac-sha256:vec[.]contratacion-temporal[.]'
            || 'ambito-idempotencia/v1:[a-f0-9]{64}$'
        )
        AND right(ambito_hmac, 64) <> repeat('0', 64)
    ),
    ADD CONSTRAINT identidad_reserva_huella_peticion_valida CHECK (
        huella_peticion_hmac ~ (
            '^hmac-sha256:vec[.]contratacion-temporal[.]'
            || 'huella-peticion/v1:[a-f0-9]{64}$'
        )
        AND right(huella_peticion_hmac, 64) <> repeat('0', 64)
    );

GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.preparar_alta_v1(jsonb)
    TO vec_contratacion_temporal_ejecutor;

COMMIT;
