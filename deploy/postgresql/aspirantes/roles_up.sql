\set ON_ERROR_STOP on
-- Aspirantes: aprovisionamiento DBA único. Solo existe el ejecutor del
-- portal externo; las vistas de RRHH tendrán su propio rol por finalidad.
-- El LOGIN técnico se crea fuera de Git como miembro directo exclusivo de
-- vec_aspirantes_ejecutor_externo (INHERIT TRUE, SET FALSE, ADMIN FALSE).
BEGIN;
SET LOCAL search_path=pg_catalog;
DO $pre$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN
      ('vec_aspirantes_propietario','vec_aspirantes_migrador','vec_aspirantes_ejecutor_externo'))
    OR EXISTS (SELECT 1 FROM pg_database d CROSS JOIN LATERAL
      aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a
      WHERE d.datname=current_database() AND a.grantee=0 AND a.privilege_type='CREATE')
 THEN RAISE EXCEPTION 'provisión Aspirantes rechazada' USING ERRCODE='42501'; END IF;
END $pre$;
CREATE ROLE vec_aspirantes_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_aspirantes_migrador NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_aspirantes_ejecutor_externo NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_aspirantes_propietario TO vec_aspirantes_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
DO $base$ BEGIN
 EXECUTE format('GRANT CONNECT, CREATE ON DATABASE %I TO vec_aspirantes_propietario',current_database());
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_aspirantes_migrador,vec_aspirantes_ejecutor_externo',current_database());
END $base$;
COMMIT;
