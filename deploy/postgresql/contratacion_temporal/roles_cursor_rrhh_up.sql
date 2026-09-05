-- Delta DBA mínimo para que el propietario CT genere cursores opacos.
-- La extensión no se convierte en API del runtime: solo se concede el
-- overload exacto al propietario NOLOGIN de funciones SECURITY DEFINER.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:cursor_rrhh:roles:v1',
        0
    )
);

DO $prevalidacion$
DECLARE
    v_funcion oid;
    v_identidad_migrador oid;
    v_identidad_propietario oid;
    v_atestacion_propietario oid;
    v_propietario oid;
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = current_user
           AND rolsuper
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'delta criptográfico de cursor RRHH rechazado';
    END IF;
    SELECT oid
      INTO v_propietario
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_contratacion_temporal_propietario'
       AND NOT rolcanlogin
       AND NOT rolsuper
       AND NOT rolcreatedb
       AND NOT rolcreaterole
       AND NOT rolinherit
       AND NOT rolreplication
       AND NOT rolbypassrls;
    SELECT oid
      INTO v_identidad_migrador
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_identidad_sesiones_v1_migrador'
       AND NOT rolcanlogin
       AND NOT rolsuper
       AND NOT rolcreatedb
       AND NOT rolcreaterole
       AND NOT rolinherit
       AND NOT rolreplication
       AND NOT rolbypassrls;
    SELECT oid
      INTO v_identidad_propietario
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_identidad_sesiones_v1_propietario'
       AND NOT rolcanlogin
       AND NOT rolsuper
       AND NOT rolcreatedb
       AND NOT rolcreaterole
       AND NOT rolinherit
       AND NOT rolreplication
       AND NOT rolbypassrls;
    -- El núcleo V3 ya puede usar este mismo generador. No se retira ni se
    -- concede su permiso aquí: solo se reconoce al propietario nominal exacto.
    SELECT oid
      INTO v_atestacion_propietario
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_autorizacion_atestada_v3_propietario'
       AND NOT rolcanlogin
       AND NOT rolsuper
       AND NOT rolcreatedb
       AND NOT rolcreaterole
       AND NOT rolinherit
       AND NOT rolreplication
       AND NOT rolbypassrls;
    v_funcion := pg_catalog.to_regprocedure(
        'public.gen_random_bytes(integer)'
    );
    IF v_propietario IS NULL
       OR v_funcion IS NULL
       OR NOT EXISTS (
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
       )
       OR COALESCE(
           pg_catalog.has_schema_privilege(
               v_propietario, 'public', 'USAGE'
           ),
           true
       )
       OR COALESCE(
           pg_catalog.has_function_privilege(
               v_propietario, v_funcion, 'EXECUTE'
           ),
           true
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members membresia
            WHERE membresia.roleid = v_identidad_propietario
              AND NOT (
                  v_identidad_migrador IS NOT NULL
                  AND membresia.member = v_identidad_migrador
                  AND NOT membresia.admin_option
                  AND NOT membresia.inherit_option
                  AND membresia.set_option
              )
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_namespace esquema
            CROSS JOIN LATERAL pg_catalog.aclexplode(
                COALESCE(
                    esquema.nspacl,
                    pg_catalog.acldefault('n', esquema.nspowner)
                )
            ) privilegio
            WHERE esquema.nspname = 'public'
              AND privilegio.privilege_type = 'USAGE'
              AND (
                  privilegio.grantee = 0
                  OR (
                      privilegio.grantee = v_identidad_propietario
                      AND privilegio.is_grantable
                  )
              )
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc funcion
            CROSS JOIN LATERAL pg_catalog.aclexplode(
                COALESCE(
                    funcion.proacl,
                    pg_catalog.acldefault('f', funcion.proowner)
                )
            ) privilegio
            WHERE funcion.oid = v_funcion
              AND privilegio.privilege_type = 'EXECUTE'
              AND privilegio.grantee
                    IS DISTINCT FROM funcion.proowner
              AND (
                  privilegio.is_grantable
                  OR (
                      privilegio.grantee IS DISTINCT FROM v_identidad_propietario
                      AND privilegio.grantee IS DISTINCT FROM v_atestacion_propietario
                  )
              )
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'pgcrypto no está endurecido para cursor RRHH';
    END IF;
END
$prevalidacion$;

REVOKE ALL ON FUNCTION public.gen_random_bytes(integer) FROM PUBLIC;
GRANT USAGE ON SCHEMA public
    TO vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION public.gen_random_bytes(integer)
    TO vec_contratacion_temporal_propietario;

COMMIT;
