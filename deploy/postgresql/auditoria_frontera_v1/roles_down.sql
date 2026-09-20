\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_auditoria_frontera_v1:roles:up', 0)
);
DO $roles$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = current_user AND rolsuper
    ) OR pg_catalog.to_regnamespace('vec_auditoria_frontera_v1') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'roles de auditoria de frontera fuera de orden';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members m
          JOIN pg_catalog.pg_roles r ON r.oid IN (m.roleid, m.member)
         WHERE r.rolname IN ('vec_auditoria_frontera_v1_propietario', 'vec_auditoria_frontera_identidad_v1_registrador')
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'roles de auditoria de frontera conservan membresias';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'vec_auditoria_frontera_identidad_v1_registrador') THEN
        EXECUTE pg_catalog.format(
            'REVOKE CONNECT ON DATABASE %I FROM vec_auditoria_frontera_identidad_v1_registrador',
            pg_catalog.current_database()
        );
        DROP ROLE vec_auditoria_frontera_identidad_v1_registrador;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'vec_auditoria_frontera_v1_propietario') THEN
        EXECUTE pg_catalog.format(
            'REVOKE CREATE ON DATABASE %I FROM vec_auditoria_frontera_v1_propietario',
            pg_catalog.current_database()
        );
        DROP ROLE vec_auditoria_frontera_v1_propietario;
    END IF;
END
$roles$;
COMMIT;
