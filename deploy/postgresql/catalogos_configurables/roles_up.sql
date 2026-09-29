-- Aprovisionamiento DBA, una sola vez; ninguna cuenta de aplicación hereda al propietario.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:roles', 0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:migracion:000001', 0));
DO $pre$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = current_user AND rolsuper)
       OR pg_catalog.to_regnamespace('vec_catalogos_configurables') IS NOT NULL
       OR pg_catalog.to_regrole('vec_catalogos_configurables_propietario') IS NOT NULL
       OR pg_catalog.to_regrole('vec_catalogos_configurables_migrador') IS NOT NULL
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_database AS d,
                  LATERAL pg_catalog.aclexplode(COALESCE(d.datacl,
                    pg_catalog.acldefault('d', d.datdba))) AS a
                   WHERE d.datname = pg_catalog.current_database() AND a.grantee = 0)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_namespace AS n,
                  LATERAL pg_catalog.aclexplode(COALESCE(n.nspacl,
                    pg_catalog.acldefault('n', n.nspowner))) AS a
                   WHERE n.nspname = 'public' AND a.grantee = 0 AND a.privilege_type = 'CREATE') THEN
        RAISE EXCEPTION 'provision de catalogos configurables rechazada' USING ERRCODE = '55000';
    END IF;
END $pre$;
CREATE ROLE vec_catalogos_configurables_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_catalogos_configurables_migrador NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_catalogos_configurables_propietario TO vec_catalogos_configurables_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
DO $base$
BEGIN
    EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO vec_catalogos_configurables_propietario', pg_catalog.current_database());
    EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO vec_catalogos_configurables_migrador', pg_catalog.current_database());
END $base$;
CREATE SCHEMA vec_catalogos_configurables AUTHORIZATION vec_catalogos_configurables_propietario;
REVOKE ALL ON SCHEMA vec_catalogos_configurables FROM PUBLIC;
COMMIT;
