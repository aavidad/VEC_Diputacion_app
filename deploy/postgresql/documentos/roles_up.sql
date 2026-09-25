\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
DO $$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = current_user AND rolsuper)
    OR to_regnamespace('vec_documentos') IS NOT NULL
    OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN
       ('vec_documentos_propietario','vec_documentos_migrador','vec_documentos_ejecutor'))
 THEN RAISE EXCEPTION 'documentos: bootstrap incompatible' USING ERRCODE = '55000'; END IF;
END $$;
CREATE ROLE vec_documentos_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_documentos_migrador NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_documentos_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_documentos_propietario TO vec_documentos_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
CREATE SCHEMA vec_documentos AUTHORIZATION vec_documentos_propietario;
REVOKE ALL ON SCHEMA vec_documentos FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_documentos TO vec_documentos_migrador, vec_documentos_ejecutor;
DO $connect$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_documentos_migrador, vec_documentos_ejecutor',current_database()); END $connect$;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_documentos_propietario REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_documentos_propietario REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
COMMIT;
