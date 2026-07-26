BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000012_confirmacion_operacion_analisis', 0
    )
);

DROP FUNCTION
    vec_contratacion_temporal.confirmar_operacion_analisis_v1(jsonb);

COMMIT;
