\set ON_ERROR_STOP on
-- Usuarios 000009: propietarios separados para los correos del personal
-- (portal interno) y de las personas que usan el Área personal (portal
-- externo). Cada uno posee solo su esquema; ninguno tiene permisos sobre el
-- otro ni sobre vec_usuarios. Aprovisionamiento DBA único, antes de AD3-109.
BEGIN;
SET LOCAL search_path=pg_catalog;
SELECT pg_advisory_xact_lock(hashtextextended('vec_usuarios:rol_correos:000009',0));
DO $pre$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regnamespace('vec_usuarios') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_migrador' AND NOT rolcanlogin)
    OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN
      ('vec_usuarios_correos_interno_propietario','vec_usuarios_correos_externo_propietario'))
 THEN RAISE EXCEPTION 'Usuarios 000009: provisión de rol incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_usuarios_correos_interno_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_usuarios_correos_externo_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_usuarios_correos_interno_propietario TO vec_usuarios_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
GRANT vec_usuarios_correos_externo_propietario TO vec_usuarios_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
DO $base$ BEGIN
 EXECUTE format('GRANT CONNECT, CREATE ON DATABASE %I TO vec_usuarios_correos_interno_propietario, vec_usuarios_correos_externo_propietario',current_database());
END $base$;
COMMIT;
