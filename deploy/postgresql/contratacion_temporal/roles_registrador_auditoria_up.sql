-- Delta DBA: grupo técnico exclusivo para denegaciones tempranas de Auditoría.
-- Los LOGIN nominales y sus membresías se aprovisionan fuera de Git.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:rol_registrador_auditoria:v1', 0
    )
);

DO $delta$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = current_user AND rolsuper
    ) OR pg_catalog.to_regnamespace('vec_contratacion_temporal') IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'delta del registrador de auditoria requiere DBA sobre CT';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_contratacion_temporal_registrador_auditoria'
    ) THEN
        EXECUTE
            'CREATE ROLE vec_contratacion_temporal_registrador_auditoria '
            || 'NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT '
            || 'NOREPLICATION NOBYPASSRLS';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_contratacion_temporal_registrador_auditoria'
           AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
           AND NOT rolcreaterole AND rolinherit AND NOT rolreplication
           AND NOT rolbypassrls
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members membresia
          JOIN pg_catalog.pg_roles miembro ON miembro.oid = membresia.member
         WHERE miembro.rolname = 'vec_contratacion_temporal_registrador_auditoria'
    ) OR pg_catalog.has_schema_privilege(
        'vec_contratacion_temporal_registrador_auditoria',
        'vec_contratacion_temporal', 'USAGE'
    ) OR pg_catalog.has_schema_privilege(
        'vec_contratacion_temporal_registrador_auditoria',
        'vec_contratacion_temporal', 'CREATE'
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc funcion
         WHERE funcion.pronamespace = 'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND pg_catalog.has_function_privilege(
                'vec_contratacion_temporal_registrador_auditoria', funcion.oid, 'EXECUTE'
           )
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_class objeto
         WHERE objeto.relnamespace = 'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND objeto.relkind IN ('r', 'p', 'v', 'm')
           AND (
                pg_catalog.has_table_privilege(
                    'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'SELECT'
                ) OR pg_catalog.has_table_privilege(
                    'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'INSERT'
                ) OR pg_catalog.has_table_privilege(
                    'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'UPDATE'
                ) OR pg_catalog.has_table_privilege(
                    'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'DELETE'
                ) OR pg_catalog.has_table_privilege(
                    'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'TRUNCATE'
                ) OR pg_catalog.has_table_privilege(
                    'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'REFERENCES'
                ) OR pg_catalog.has_table_privilege(
                    'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'TRIGGER'
                ) OR pg_catalog.has_table_privilege(
                    'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'MAINTAIN'
                ) OR pg_catalog.has_any_column_privilege(
                    'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'SELECT'
                ) OR pg_catalog.has_any_column_privilege(
                    'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'INSERT'
                ) OR pg_catalog.has_any_column_privilege(
                    'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'UPDATE'
                ) OR pg_catalog.has_any_column_privilege(
                    'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'REFERENCES'
                )
           )
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_class secuencia
         WHERE secuencia.relnamespace = 'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND secuencia.relkind = 'S'
           AND (
                pg_catalog.has_sequence_privilege(
                    'vec_contratacion_temporal_registrador_auditoria', secuencia.oid, 'USAGE'
                ) OR pg_catalog.has_sequence_privilege(
                    'vec_contratacion_temporal_registrador_auditoria', secuencia.oid, 'SELECT'
                ) OR pg_catalog.has_sequence_privilege(
                    'vec_contratacion_temporal_registrador_auditoria', secuencia.oid, 'UPDATE'
                )
           )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'rol registrador de auditoria existente no es minimo';
    END IF;
    EXECUTE pg_catalog.format(
        'GRANT CONNECT ON DATABASE %I TO vec_contratacion_temporal_registrador_auditoria',
        pg_catalog.current_database()
    );
END
$delta$;

COMMIT;
