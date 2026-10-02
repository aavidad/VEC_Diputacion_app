\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_meritos:bootstrap:v1',0));
DO $pre$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regnamespace('vec_meritos') IS NOT NULL
 OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN ('vec_meritos_propietario','vec_meritos_migrador','vec_meritos_ejecutor','vec_meritos_externo','vec_meritos_interno'))
 THEN RAISE EXCEPTION 'meritos.error.bootstrap_incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_meritos_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_meritos_migrador NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_meritos_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_meritos_externo NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_meritos_interno NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_meritos_propietario TO vec_meritos_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
CREATE SCHEMA vec_meritos AUTHORIZATION vec_meritos_propietario;
REVOKE ALL ON SCHEMA vec_meritos FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_meritos TO vec_meritos_ejecutor,vec_meritos_migrador;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_meritos_propietario REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_meritos_propietario REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_meritos_propietario REVOKE USAGE ON TYPES FROM PUBLIC;
DO $db$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_meritos_ejecutor,vec_meritos_migrador',current_database()); END $db$;
COMMIT;
