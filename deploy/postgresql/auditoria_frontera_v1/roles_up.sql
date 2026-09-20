\set ON_ERROR_STOP on
-- Rol técnico, sin LOGIN ni membresías, para el proceso que registra la
-- denegación ya decidida en la frontera de identidad. Las cuentas LOGIN se
-- aprovisionan fuera del repositorio.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_auditoria_frontera_v1:roles:up', 0)
);

DO $roles$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'roles de auditoria de frontera requieren DBA';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_auditoria_frontera_v1_propietario'
    ) THEN
        CREATE ROLE vec_auditoria_frontera_v1_propietario
            NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_auditoria_frontera_identidad_v1_registrador'
    ) THEN
        CREATE ROLE vec_auditoria_frontera_identidad_v1_registrador
            NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname IN ('vec_auditoria_frontera_v1_propietario', 'vec_auditoria_frontera_identidad_v1_registrador')
           AND (rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolinherit OR rolreplication OR rolbypassrls)
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members AS m
         WHERE m.roleid = 'vec_auditoria_frontera_v1_propietario'::regrole
            OR m.member = 'vec_auditoria_frontera_v1_propietario'::regrole
            OR m.member = 'vec_auditoria_frontera_identidad_v1_registrador'::regrole
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'roles de auditoria de frontera no son minimos';
    END IF;
    EXECUTE pg_catalog.format(
        'GRANT CONNECT ON DATABASE %I TO vec_auditoria_frontera_identidad_v1_registrador',
        pg_catalog.current_database()
    );
    EXECUTE pg_catalog.format(
        'GRANT CREATE ON DATABASE %I TO vec_auditoria_frontera_v1_propietario',
        pg_catalog.current_database()
    );
END
$roles$;
COMMIT;
