-- Retira la única capacidad pgcrypto de cursores cuando su esquema y cualquier
-- fachada posterior ya no existen. No restaura privilegios de PUBLIC.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:cursor_rrhh:roles:v1',
        0
    )
);
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:o4_05:consultas_rrhh:migraciones',
        0
    )
);

DO $prevalidacion$
DECLARE
    v_funcion oid := pg_catalog.to_regprocedure(
        'public.gen_random_bytes(integer)'
    );
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = current_user
           AND rolsuper
    ) OR v_funcion IS NULL OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc funcion
          JOIN pg_catalog.pg_namespace esquema
            ON esquema.oid = funcion.pronamespace
          JOIN pg_catalog.pg_depend dependencia
            ON dependencia.objid = funcion.oid
          JOIN pg_catalog.pg_extension extension
            ON extension.oid = dependencia.refobjid
         WHERE funcion.oid = v_funcion
           AND esquema.nspname = 'public'
           AND extension.extname = 'pgcrypto'
           AND extension.extnamespace = esquema.oid
           AND funcion.proowner = extension.extowner
           AND funcion.prosupport = 0
           AND funcion.proconfig IS NULL
           AND pg_catalog.encode(pg_catalog.sha256(
               pg_catalog.convert_to(
                   pg_catalog.pg_get_functiondef(funcion.oid), 'UTF8'
               )
           ), 'hex') =
               '3e5d4a298efb95a8c94a2e47a06244bb747c33f2400461b01531b7b12bc010b6'
           AND dependencia.classid = 'pg_catalog.pg_proc'::regclass
           AND dependencia.refclassid =
               'pg_catalog.pg_extension'::regclass
           AND dependencia.deptype = 'e'
    ) OR NOT pg_catalog.has_schema_privilege(
        'vec_contratacion_temporal_propietario',
        'public',
        'USAGE'
    ) OR NOT pg_catalog.has_function_privilege(
        'vec_contratacion_temporal_propietario',
        'public.gen_random_bytes(integer)',
        'EXECUTE'
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .control_migracion_cobertura_o4
         WHERE control
           AND version_esquema = 17
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .control_migracion_consultas_rrhh
         WHERE control
           AND version_esquema = 1
    ) OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.control_cursores_cuadro_rrhh'
    ) IS NOT NULL OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc funcion
          JOIN pg_catalog.pg_namespace esquema
            ON esquema.oid = funcion.pronamespace
         WHERE esquema.nspname = 'vec_contratacion_temporal'
           AND funcion.proname ~
               '(cursor.*rrhh|rrhh.*cursor|^consultar_(cuadro|detalle)_rrhh)'
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'delta criptográfico de cursor RRHH no retirable';
    END IF;
END
$prevalidacion$;

REVOKE EXECUTE ON FUNCTION public.gen_random_bytes(integer)
    FROM vec_contratacion_temporal_propietario;
REVOKE USAGE ON SCHEMA public
    FROM vec_contratacion_temporal_propietario;

COMMIT;
