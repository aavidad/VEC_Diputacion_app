\set ON_ERROR_STOP on
-- Usuarios 000003: grupos separados para ingresos interno y exterior.
-- Cada LOGIN privado se aprovisiona fuera de Git, con una sola membresía
-- INHERIT TRUE, SET FALSE, ADMIN FALSE en su grupo y DSN/pool propios.
BEGIN;
SET LOCAL search_path=pg_catalog;
SELECT pg_advisory_xact_lock(hashtextextended('vec_usuarios:rol_frontera:000003',0));
DO $pre$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regnamespace('vec_usuarios') IS NULL
    OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN
      ('vec_usuarios_registrador_frontera_interno','vec_usuarios_registrador_frontera_externo'))
 THEN RAISE EXCEPTION 'Usuarios 000003: provision de rol incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_usuarios_registrador_frontera_interno NOLOGIN NOSUPERUSER NOCREATEDB
 NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_usuarios_registrador_frontera_externo NOLOGIN NOSUPERUSER NOCREATEDB
 NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
DO $grant$ BEGIN
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_usuarios_registrador_frontera_interno, vec_usuarios_registrador_frontera_externo',current_database());
END $grant$;
COMMIT;
