BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000010_validadores_confirmacion_analisis', 0
    )
);

DROP FUNCTION
    vec_contratacion_temporal.entrada_confirmacion_analisis_valida_v1(jsonb);
DROP FUNCTION
    vec_contratacion_temporal.reconstruir_prueba_fuentes_analisis_v1(jsonb);
DROP FUNCTION
    vec_contratacion_temporal.reconstruir_fuente_analisis_v1(
        jsonb, text, text, numeric
    );
DROP FUNCTION
    vec_contratacion_temporal.microsegundos_unix_analisis_v1(text);
DROP FUNCTION
    vec_contratacion_temporal.encuadrar_binario_analisis_v1(text);
DROP FUNCTION
    vec_contratacion_temporal.claves_json_exactas_v1(jsonb, text[]);

COMMIT;
