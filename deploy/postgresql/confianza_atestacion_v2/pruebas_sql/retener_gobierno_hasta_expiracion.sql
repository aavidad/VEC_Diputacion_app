BEGIN;
SET LOCAL search_path = pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_confianza_atestacion_v2:gobierno:v1', 0
    )
);
SELECT pg_catalog.pg_advisory_lock(721702026);
SELECT pg_catalog.pg_sleep(7);
COMMIT;
SELECT pg_catalog.pg_advisory_unlock(721702026);
