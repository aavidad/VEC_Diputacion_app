\set ON_ERROR_STOP on
-- Documentos-4 (DBA, una sola vez, antes de migraciones/000004): rol técnico
-- que sólo registra denegaciones de la frontera HTTP. No es miembro del
-- ejecutor ni del propietario; el LOGIN de aplicación que lo reciba no puede
-- tener ninguna otra membresía vec_.
BEGIN;
SET LOCAL search_path = pg_catalog;
DO $$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = current_user AND rolsuper)
    OR to_regnamespace('vec_documentos') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'vec_documentos_ejecutor')
    OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'vec_documentos_auditor')
 THEN RAISE EXCEPTION 'documentos 000004: bootstrap del auditor incompatible' USING ERRCODE = '55000'; END IF;
END $$;
CREATE ROLE vec_documentos_auditor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
DO $connect$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_documentos_auditor',current_database()); END $connect$;
COMMIT;
