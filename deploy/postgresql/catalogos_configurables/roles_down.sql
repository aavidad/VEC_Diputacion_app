-- Solo para una instalación vacía; no ejecutar tras publicar historia.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:roles', 0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:migracion:000001', 0));
DO $pre$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = current_user AND rolsuper)
       OR pg_catalog.to_regnamespace('vec_catalogos_configurables') IS NULL
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_namespace
                   WHERE nspname = 'vec_catalogos_configurables'
                     AND nspowner <> 'vec_catalogos_configurables_propietario'::regrole)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_class AS c JOIN pg_catalog.pg_namespace AS n
                   ON n.oid = c.relnamespace WHERE n.nspname = 'vec_catalogos_configurables')
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc AS p JOIN pg_catalog.pg_namespace AS n
                   ON n.oid = p.pronamespace WHERE n.nspname = 'vec_catalogos_configurables')
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_type AS t JOIN pg_catalog.pg_namespace AS n
                   ON n.oid = t.typnamespace WHERE n.nspname = 'vec_catalogos_configurables')
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_default_acl
                   WHERE defaclrole = 'vec_catalogos_configurables_propietario'::regrole)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members WHERE roleid = 'vec_catalogos_configurables_propietario'::regrole AND member <> 'vec_catalogos_configurables_migrador'::regrole)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members WHERE roleid = 'vec_catalogos_configurables_migrador'::regrole) THEN
        RAISE EXCEPTION 'reversion de roles de catalogos rechazada' USING ERRCODE = '55000';
    END IF;
END $pre$;
DROP SCHEMA vec_catalogos_configurables;
DO $base$
BEGIN
    EXECUTE pg_catalog.format('REVOKE ALL ON DATABASE %I FROM vec_catalogos_configurables_propietario, vec_catalogos_configurables_migrador', pg_catalog.current_database());
END $base$;
REVOKE vec_catalogos_configurables_propietario FROM vec_catalogos_configurables_migrador;
DROP ROLE vec_catalogos_configurables_migrador;
DROP ROLE vec_catalogos_configurables_propietario;
COMMIT;
