BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000011_transicion_confirmacion_analisis', 0
    )
);

DROP FUNCTION
    vec_contratacion_temporal.transicion_confirmacion_analisis_valida_v1(
        jsonb, jsonb
    );

COMMIT;
