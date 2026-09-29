\set ON_ERROR_STOP on
-- CT-000143 DOWN: retira solo la lectura del número vigente del aviso.
-- No toca datos: la función no escribe ni la usa ninguna otra función SQL.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000143', 0)
);
DROP FUNCTION vec_contratacion_temporal.numero_visible_aviso_confirmado_v1(text, text, text);
COMMIT;
