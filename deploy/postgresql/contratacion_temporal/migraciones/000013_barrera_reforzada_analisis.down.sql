BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000013_barrera_reforzada_analisis', 0
    )
);

DO $proteccion$
BEGIN
    IF (
           EXISTS (
               SELECT 1
                 FROM vec_contratacion_temporal
                      .confirmacion_operacion_analisis
           )
           OR EXISTS (
               SELECT 1
                 FROM vec_contratacion_temporal
                      .expediente_version_integral
                WHERE origen_version = 'analisis_o3'
           )
       )
       AND pg_catalog.current_setting(
           'vec.confirmar_destruccion_contratacion_temporal', true
       ) IS DISTINCT FROM
           'DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada de validadores O3 v2 protegida por historia';
    END IF;
END
$proteccion$;

DROP FUNCTION
    vec_contratacion_temporal.expediente_analisis_valido_v2(jsonb, boolean);
DROP FUNCTION
    vec_contratacion_temporal.huella_analisis_derivado_v2(jsonb);
DROP FUNCTION
    vec_contratacion_temporal.actuacion_analisis_valida_v2(jsonb);
DROP FUNCTION
    vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(jsonb);
DROP FUNCTION
    vec_contratacion_temporal.texto_instante_utc_go_v2(text);
DROP FUNCTION
    vec_contratacion_temporal.instante_utc_json_canonico_v2(jsonb, boolean);
DROP FUNCTION
    vec_contratacion_temporal.campos_texto_json_v2(jsonb, text[]);
DROP FUNCTION
    vec_contratacion_temporal.numero_entero_json_canonico_v2(
        jsonb, numeric, numeric
    );

COMMIT;
