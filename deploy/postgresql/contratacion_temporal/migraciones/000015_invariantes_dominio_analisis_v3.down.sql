BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000015_invariantes_dominio_analisis_v3',
        0
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
            MESSAGE = 'retirada de invariantes O3 v3 protegida por historia';
    END IF;
END
$proteccion$;

REVOKE EXECUTE ON FUNCTION
vec_contratacion_temporal.confirmar_operacion_analisis_v3(jsonb)
FROM vec_contratacion_temporal_ejecutor;

DROP FUNCTION
    vec_contratacion_temporal.confirmar_operacion_analisis_v3(jsonb);
DROP FUNCTION
    vec_contratacion_temporal.ejecutar_confirmacion_analisis_base_v3(jsonb);
DROP FUNCTION
    vec_contratacion_temporal.analisis_rrhh_valido_v3(jsonb);
DROP FUNCTION
    vec_contratacion_temporal.texto_dominio_analisis_valido_v3(
        text, integer, boolean
    );
DROP FUNCTION
    vec_contratacion_temporal.huella_dominio_analisis_valida_v3(text);
DROP FUNCTION
    vec_contratacion_temporal.grupo_dominio_analisis_valido_v3(text);
DROP FUNCTION
    vec_contratacion_temporal.clave_dominio_analisis_valida_v3(text);
DROP FUNCTION
    vec_contratacion_temporal.referencia_dominio_analisis_valida_v3(text);

GRANT EXECUTE ON FUNCTION
vec_contratacion_temporal.confirmar_operacion_analisis_v2(jsonb)
TO vec_contratacion_temporal_ejecutor;

COMMIT;
