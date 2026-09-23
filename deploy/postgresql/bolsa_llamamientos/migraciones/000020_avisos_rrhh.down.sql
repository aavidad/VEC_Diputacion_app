\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000020', 0));
DROP FUNCTION vec_bolsa_llamamientos.consultar_avisos_rrhh_v1(timestamptz);
COMMIT;

