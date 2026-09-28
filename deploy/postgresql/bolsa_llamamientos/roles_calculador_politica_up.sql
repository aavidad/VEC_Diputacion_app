\set ON_ERROR_STOP on
-- DBA, antes de Bolsa 000051. LOGIN nominal y secreto se crean fuera de Git;
-- debe heredar exclusivamente este grupo NOLOGIN en un pool independiente.
BEGIN;
SET LOCAL search_path=pg_catalog;
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:rol:calculador_politica',0));
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_propietario' AND NOT rolcanlogin)
    OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_calculador_politica')
 THEN RAISE EXCEPTION 'rol calculador de política: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_bolsa_llamamientos_calculador_politica NOLOGIN NOSUPERUSER
 NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
DO $conectar$ BEGIN
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_bolsa_llamamientos_calculador_politica',current_database());
END $conectar$;
COMMIT;
