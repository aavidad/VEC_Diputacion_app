\set ON_ERROR_STOP on
-- Usuarios 000003: grupo exclusivo de registro de denegaciones HTTP.
-- La cuenta LOGIN nominal se aprovisiona fuera de Git, con una sola membresía
-- INHERIT TRUE, SET FALSE, ADMIN FALSE en este grupo.
BEGIN;
SET LOCAL search_path=pg_catalog;
SELECT pg_advisory_xact_lock(hashtextextended('vec_usuarios:rol_frontera:000003',0));
DO $pre$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regnamespace('vec_usuarios') IS NULL
    OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_registrador_frontera')
 THEN RAISE EXCEPTION 'Usuarios 000003: provision de rol incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_usuarios_registrador_frontera NOLOGIN NOSUPERUSER NOCREATEDB
 NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
DO $grant$ BEGIN
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_usuarios_registrador_frontera',current_database());
END $grant$;
COMMIT;
