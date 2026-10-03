\set ON_ERROR_STOP on
-- RUM04: sólo el rol técnico lector; no provisiona logins ni permisos personales.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_meritos:consulta_propia:roles:v1',0));
DO $pre$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regclass('vec_meritos.hecho_version') IS NULL
 OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_meritos_consulta_propia_interno')
 THEN RAISE EXCEPTION 'meritos.error.lector_bootstrap_incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_meritos_consulta_propia_interno NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
GRANT USAGE ON SCHEMA vec_meritos TO vec_meritos_consulta_propia_interno;
DO $db$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_meritos_consulta_propia_interno',current_database()); END $db$;
COMMIT;
