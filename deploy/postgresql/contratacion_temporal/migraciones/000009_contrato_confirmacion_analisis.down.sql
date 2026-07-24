BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000009_contrato_confirmacion_analisis', 0
    )
);

DO $proteccion$
BEGIN
    IF EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.consumo_fuentes_analisis
       )
       AND pg_catalog.current_setting(
           'vec.confirmar_destruccion_contratacion_temporal', true
       ) IS DISTINCT FROM
           'DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada del contrato de fuentes protegida';
    END IF;
END
$proteccion$;

DROP FUNCTION
    vec_contratacion_temporal.huella_contexto_recurso_analisis_v1(jsonb);
ALTER TABLE vec_contratacion_temporal.consumo_fuentes_analisis
    DROP CONSTRAINT consumo_fuentes_prueba_canonica_huella,
    DROP CONSTRAINT consumo_fuentes_prueba_canonica_tamano,
    DROP COLUMN prueba_canonica;

COMMIT;
