-- Ejecutar después del down del registro RRHH y de retirar sus LOGIN.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:rol_consultor_rrhh:v1',
        0
    )
);

DO $delta$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = current_user
           AND rolsuper
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control
           AND version_esquema = 15
    ) OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.control_migracion_consultas_rrhh'
    ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'down del consultor RRHH fuera de orden';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_contratacion_temporal_consultor_rrhh'
    ) THEN
        RETURN;
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_contratacion_temporal_consultor_rrhh'
           AND NOT rolcanlogin
           AND NOT rolsuper
           AND NOT rolcreatedb
           AND NOT rolcreaterole
           AND rolinherit
           AND NOT rolreplication
           AND NOT rolbypassrls
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members membresia
          JOIN pg_catalog.pg_roles rol
            ON rol.oid = membresia.roleid
            OR rol.oid = membresia.member
         WHERE rol.rolname =
               'vec_contratacion_temporal_consultor_rrhh'
    ) OR NOT pg_catalog.has_database_privilege(
        'vec_contratacion_temporal_consultor_rrhh',
        pg_catalog.current_database(),
        'CONNECT'
    ) OR pg_catalog.has_database_privilege(
        'vec_contratacion_temporal_consultor_rrhh',
        pg_catalog.current_database(),
        'CREATE'
    ) OR pg_catalog.has_database_privilege(
        'vec_contratacion_temporal_consultor_rrhh',
        pg_catalog.current_database(),
        'TEMP'
    ) OR pg_catalog.has_schema_privilege(
        'vec_contratacion_temporal_consultor_rrhh',
        'vec_contratacion_temporal',
        'USAGE'
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'rol consultor RRHH conserva autoridad o membresías';
    END IF;

    EXECUTE pg_catalog.format(
        'REVOKE CONNECT ON DATABASE %I '
        || 'FROM vec_contratacion_temporal_consultor_rrhh',
        pg_catalog.current_database()
    );
    EXECUTE 'DROP ROLE vec_contratacion_temporal_consultor_rrhh';
END
$delta$;

COMMIT;
