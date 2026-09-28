\set ON_ERROR_STOP on
-- Aprovisionamiento DBA único. Las identidades LOGIN se crean fuera de Git.
BEGIN;
SET LOCAL search_path=pg_catalog;
DO $pre$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN
      ('vec_usuarios_propietario','vec_usuarios_migrador','vec_usuarios_ejecutor'))
    OR EXISTS (SELECT 1 FROM pg_database d CROSS JOIN LATERAL
      aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a
      WHERE d.datname=current_database() AND a.grantee=0 AND a.privilege_type='CREATE')
 THEN RAISE EXCEPTION 'provisión Usuarios rechazada' USING ERRCODE='42501'; END IF;
END $pre$;
CREATE ROLE vec_usuarios_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_usuarios_migrador NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_usuarios_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_usuarios_propietario TO vec_usuarios_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
DO $base$ BEGIN
 EXECUTE format('GRANT CONNECT, CREATE ON DATABASE %I TO vec_usuarios_propietario',current_database());
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_usuarios_migrador,vec_usuarios_ejecutor',current_database());
END $base$;
COMMIT;
