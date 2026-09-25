\set ON_ERROR_STOP on
-- Calendarios: esquema propio y roles NOLOGIN. El despliegue concede el rol
-- lector a una identidad de conexión distinta; ningún rol puede asumir el
-- propietario salvo el migrador.
BEGIN;
SET LOCAL search_path=pg_catalog;
DO $$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) OR to_regnamespace('vec_calendarios') IS NOT NULL
    OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN ('vec_calendarios_propietario','vec_calendarios_migrador','vec_calendarios_lector'))
 THEN RAISE EXCEPTION 'bootstrap Calendarios incompatible' USING ERRCODE='55000'; END IF;
END $$;
CREATE ROLE vec_calendarios_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_calendarios_migrador NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_calendarios_lector NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_calendarios_propietario TO vec_calendarios_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
CREATE SCHEMA vec_calendarios AUTHORIZATION vec_calendarios_propietario;
REVOKE ALL ON SCHEMA vec_calendarios FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_calendarios TO vec_calendarios_migrador, vec_calendarios_lector;
DO $conectar$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_calendarios_migrador, vec_calendarios_lector', current_database()); END $conectar$;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_calendarios_propietario REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_calendarios_propietario REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_calendarios_propietario REVOKE USAGE ON TYPES FROM PUBLIC;
COMMIT;
