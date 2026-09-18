\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:migracion:000107', 0
    )
);
DROP FUNCTION vec_contratacion_temporal.consultar_estadisticas_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1, text, date, date
) RESTRICT;
COMMIT;
