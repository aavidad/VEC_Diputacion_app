\set ON_ERROR_STOP on
-- Provisión DBA anterior a Usuarios 000004. La identidad LOGIN y su secreto
-- se provisionan fuera de Git, con una sola membresía heredada y sin SET.
BEGIN;
SET LOCAL search_path=pg_catalog;
DO $pre$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_despachador')
 THEN RAISE EXCEPTION 'provisión despacho Usuarios rechazada' USING ERRCODE='42501'; END IF;
END $pre$;
CREATE ROLE vec_usuarios_despachador NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
DO $base$ BEGIN
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_usuarios_despachador',current_database());
END $base$;
COMMIT;
