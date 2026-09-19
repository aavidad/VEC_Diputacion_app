-- Delta DBA: grupo técnico exclusivo del registro minimizado de denegaciones.
-- Las cuentas LOGIN nominales y su pertenencia se aprovisionan fuera de Git.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:rol_registrador_frontera:v1', 0
    )
);

DO $delta$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = current_user AND rolsuper
    ) OR pg_catalog.to_regnamespace('vec_contratacion_temporal') IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'delta del registrador de frontera requiere DBA sobre CT';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_contratacion_temporal_registrador_frontera'
    ) THEN
        EXECUTE
            'CREATE ROLE vec_contratacion_temporal_registrador_frontera '
            || 'NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT '
            || 'NOREPLICATION NOBYPASSRLS';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_contratacion_temporal_registrador_frontera'
           AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
           AND NOT rolcreaterole AND rolinherit AND NOT rolreplication
           AND NOT rolbypassrls
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members membresia
        JOIN pg_catalog.pg_roles miembro ON miembro.oid = membresia.member
        WHERE miembro.rolname = 'vec_contratacion_temporal_registrador_frontera'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'rol registrador de frontera existente no es mínimo';
    END IF;
    EXECUTE pg_catalog.format(
        'GRANT CONNECT ON DATABASE %I TO vec_contratacion_temporal_registrador_frontera',
        pg_catalog.current_database()
    );
END
$delta$;

COMMIT;
