\set ON_ERROR_STOP on
-- Bolsa 000082 DOWN: retira las dos lecturas de conjunto. No toca nada más.
-- Solo en entornos desechables; con la aplicación nueva en marcha, el cuadro
-- de RRHH volvería a fallar cerrado (503) hasta reinstalarla.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000082',0));
DROP FUNCTION vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz);
DROP FUNCTION vec_bolsa_llamamientos.leer_politicas_orden_vigentes_v1(timestamptz);
COMMIT;
