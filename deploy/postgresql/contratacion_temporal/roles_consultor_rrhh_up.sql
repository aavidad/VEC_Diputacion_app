-- Delta DBA del grupo técnico de consultas RRHH O4-05.
-- Las cuentas LOGIN son nominativas y se aprovisionan fuera del repositorio.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:rol_consultor_rrhh:v1',
        0
    )
);

DO $delta$
DECLARE
    v_version integer;
    v_version_consultas integer;
    v_control_consultas regclass;
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = current_user
           AND rolsuper
    ) OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.control_migracion_cobertura_o4'
    ) IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'delta del consultor RRHH requiere DBA sobre CT';
    END IF;

    EXECUTE
        'SELECT min(version_esquema) '
        || 'FROM vec_contratacion_temporal.control_migracion_cobertura_o4 '
        || 'HAVING count(*) = 1 AND bool_and(control)'
        INTO v_version;
    v_control_consultas := pg_catalog.to_regclass(
        'vec_contratacion_temporal.control_migracion_consultas_rrhh'
    );
    IF v_control_consultas IS NOT NULL THEN
        EXECUTE
            'SELECT min(version_esquema) '
            || 'FROM vec_contratacion_temporal.'
            || 'control_migracion_consultas_rrhh '
            || 'HAVING count(*) = 1 AND bool_and(control)'
            INTO v_version_consultas;
    END IF;
    IF NOT COALESCE((
        (
            v_version = 15
            AND v_control_consultas IS NULL
        )
        OR (
            v_version = 16
            AND v_control_consultas IS NOT NULL
            AND v_version_consultas = 1
        )
    ), false) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'barrera CT incompatible con consultor RRHH';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_contratacion_temporal_consultor_rrhh'
    ) THEN
        EXECUTE
            'CREATE ROLE vec_contratacion_temporal_consultor_rrhh '
            || 'NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT '
            || 'NOREPLICATION NOBYPASSRLS';
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
          JOIN pg_catalog.pg_roles miembro
            ON miembro.oid = membresia.member
         WHERE miembro.rolname =
               'vec_contratacion_temporal_consultor_rrhh'
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members membresia
          JOIN pg_catalog.pg_roles grupo
            ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles miembro
            ON miembro.oid = membresia.member
         WHERE grupo.rolname =
               'vec_contratacion_temporal_consultor_rrhh'
           AND (
               membresia.admin_option
               OR NOT membresia.inherit_option
               OR membresia.set_option
               OR NOT miembro.rolcanlogin
               OR miembro.rolsuper
               OR miembro.rolcreatedb
               OR miembro.rolcreaterole
               OR miembro.rolreplication
               OR miembro.rolbypassrls
               OR (
                   SELECT pg_catalog.count(*)
                     FROM pg_catalog.pg_auth_members directas
                    WHERE directas.member = miembro.oid
               ) <> 1
           )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members directa
          JOIN pg_catalog.pg_roles grupo
            ON grupo.oid = directa.roleid
          JOIN pg_catalog.pg_auth_members puente
            ON puente.roleid = directa.member
         WHERE grupo.rolname =
               'vec_contratacion_temporal_consultor_rrhh'
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'rol consultor RRHH existente no es grupo mínimo';
    END IF;

    EXECUTE pg_catalog.format(
        'GRANT CONNECT ON DATABASE %I '
        || 'TO vec_contratacion_temporal_consultor_rrhh',
        pg_catalog.current_database()
    );
    IF NOT pg_catalog.has_database_privilege(
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
            MESSAGE = 'ACL inicial del consultor RRHH no es mínima';
    END IF;
END
$delta$;

COMMIT;
