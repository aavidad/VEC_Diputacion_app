BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '1s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:identidad:consulta_rrhh:v1', 0
    )
);

SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;

DO $prevalidacion$
DECLARE
    v_funcion oid := pg_catalog.to_regprocedure(
        'vec_identidad_sesiones_v1.'
        || 'revalidar_consulta_rrhh_v1(text,text)'
    );
BEGIN
    IF v_funcion IS NULL
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc p
             JOIN pg_catalog.pg_namespace n
               ON n.oid = p.pronamespace
             JOIN pg_catalog.pg_roles r ON r.oid = p.proowner
            WHERE p.oid = v_funcion
              AND n.nspname = 'vec_identidad_sesiones_v1'
              AND r.rolname =
                  'vec_identidad_sesiones_v1_propietario'
              AND p.prosecdef
              AND p.provolatile = 'v'
              AND p.proparallel = 'u'
              AND p.proconfig @> ARRAY[
                  'search_path=pg_catalog',
                  'lock_timeout=1s'
              ]::text[]
       )
       OR pg_catalog.has_function_privilege(
           'public', v_funcion, 'EXECUTE'
       )
       OR NOT pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_propietario',
           v_funcion, 'EXECUTE'
       )
       OR pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_consultor_rrhh',
           v_funcion, 'EXECUTE'
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.aclexplode((
                 SELECT p.proacl
                   FROM pg_catalog.pg_proc p
                  WHERE p.oid = v_funcion
             )) a
            WHERE a.privilege_type = 'EXECUTE'
              AND a.grantee NOT IN (
                  (
                      SELECT oid FROM pg_catalog.pg_roles
                       WHERE rolname =
                         'vec_identidad_sesiones_v1_propietario'
                  ),
                  (
                      SELECT oid FROM pg_catalog.pg_roles
                       WHERE rolname =
                         'vec_contratacion_temporal_propietario'
                  )
              )
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'down de identidad RRHH rechazado por deriva';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_depend d
         WHERE d.refclassid = 'pg_catalog.pg_proc'::pg_catalog.regclass
           AND d.refobjid = v_funcion
           AND d.deptype IN ('n', 'a')
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '2BP01',
            MESSAGE = 'down de identidad RRHH rechazado por dependencias';
    END IF;
END
$prevalidacion$;

REVOKE EXECUTE ON FUNCTION
    vec_identidad_sesiones_v1.revalidar_consulta_rrhh_v1(text, text)
    FROM vec_contratacion_temporal_propietario;
DROP FUNCTION
    vec_identidad_sesiones_v1.revalidar_consulta_rrhh_v1(text, text)
    RESTRICT;
REVOKE USAGE ON SCHEMA vec_identidad_sesiones_v1
    FROM vec_contratacion_temporal_propietario;

COMMIT;
