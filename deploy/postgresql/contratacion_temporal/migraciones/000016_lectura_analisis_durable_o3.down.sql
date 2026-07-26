BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000016_lectura_analisis_durable_o3',
        0
    )
);

REVOKE EXECUTE ON FUNCTION
vec_contratacion_temporal.leer_expediente_analisis_durable_o3_v1(
    text, text, numeric
)
FROM vec_contratacion_temporal_ejecutor;

DROP FUNCTION
vec_contratacion_temporal.leer_expediente_analisis_durable_o3_v1(
    text, text, numeric
);

COMMIT;
