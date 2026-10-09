\set ON_ERROR_STOP on
-- Grupo técnico sin LOGIN para la llamada nominal desde AD232. No concede perfiles.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:roles:000012',0));
DO $pre$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.to_regrole('vec_catalogos_configurables_propietario') IS NULL
    OR pg_catalog.to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
    OR pg_catalog.to_regrole('vec_catalogos_configurables_rpt_consumidor') IS NOT NULL THEN
  RAISE EXCEPTION 'CC12: roles preimagen incompatible' USING ERRCODE='55000';
 END IF;
END $pre$;
CREATE ROLE vec_catalogos_configurables_rpt_consumidor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
DO $connect$
BEGIN
 EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO vec_catalogos_configurables_rpt_consumidor',pg_catalog.current_database());
END $connect$;
COMMIT;
