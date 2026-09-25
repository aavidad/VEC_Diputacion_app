-- Revalidación nominal CT: certificado personal temporal de desarrollo.
-- Depende de Identidad 000006; no crea identidades, permisos ni datos.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '1s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:identidad:consulta_rrhh:v1', 0
    )
);
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;

DO $preimagen$
DECLARE
    v_consulta oid := pg_catalog.to_regprocedure(
        'vec_identidad_sesiones_v1.revalidar_consulta_rrhh_v1(text,text)'
    );
    v_politica oid := pg_catalog.to_regprocedure(
        'vec_identidad_sesiones_v1.admite_politica_certificado_personal_desarrollo_v1(text,text,timestamptz)'
    );
BEGIN
    IF v_consulta IS NULL OR v_politica IS NULL
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_proc p
           JOIN pg_catalog.pg_roles r ON r.oid = p.proowner
           WHERE p.oid = v_consulta
             AND r.rolname = 'vec_identidad_sesiones_v1_propietario'
             AND p.prosecdef AND p.provolatile = 'v'
             AND p.proparallel = 'u'
             AND p.proconfig = ARRAY[
                 'search_path=pg_catalog', 'lock_timeout=1s'
             ]::text[]
             AND pg_catalog.octet_length(p.prosrc) = 3525
             AND pg_catalog.encode(pg_catalog.sha256(
                 pg_catalog.convert_to(p.prosrc, 'UTF8')
             ), 'hex') = '69e0c769db76e8eee040334bba764f5374360f30b01c4d1d59db14899fce6c44'
             AND (
                 SELECT pg_catalog.count(*) = 2
                    AND pg_catalog.bool_and(
                        a.grantor = p.proowner
                        AND a.grantee IN (
                            p.proowner,
                            'vec_contratacion_temporal_propietario'::regrole
                        )
                        AND a.privilege_type = 'EXECUTE'
                        AND NOT a.is_grantable
                    )
                   FROM pg_catalog.aclexplode(p.proacl) a
             )
       )
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_proc p
           JOIN pg_catalog.pg_roles r ON r.oid = p.proowner
           WHERE p.oid = v_politica
             AND r.rolname = 'vec_identidad_sesiones_v1_propietario'
             AND p.prosecdef AND p.provolatile = 'v'
             AND p.prorettype = 'boolean'::regtype
       )
       OR pg_catalog.has_function_privilege('public', v_consulta, 'EXECUTE')
       OR NOT pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_propietario', v_consulta, 'EXECUTE'
       )
       OR pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_consultor_rrhh', v_consulta, 'EXECUTE'
       )
       OR pg_catalog.has_function_privilege('public', v_politica, 'EXECUTE')
       OR pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_propietario', v_politica, 'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'preimagen o dependencia de consulta RRHH incompatible'
            USING ERRCODE = '55000';
    END IF;
END $preimagen$;

CREATE OR REPLACE FUNCTION
    vec_identidad_sesiones_v1.revalidar_consulta_rrhh_v1(
        p_autenticacion_ref text,
        p_sesion_ref text
    )
RETURNS TABLE(
    autenticacion_ref text,
    autenticacion_huella_sha256 text,
    asercion_ref text,
    sesion_ref text,
    control_sesion_ref text,
    control_sesion_revision text,
    control_sesion_huella_sha256 text,
    cuenta_ref text,
    cuenta_ordinaria_ref text,
    cuenta_privilegiada boolean,
    superficie text,
    metodo_observado text,
    garantia_observada text,
    politica_garantia_ref text,
    politica_garantia_huella_sha256 text,
    autenticacion_verificada_en timestamptz,
    sesion_emitida_en timestamptz,
    sesion_valida_hasta timestamptz,
    sesion_revalidada_en timestamptz,
    login_tecnico text
)
LANGUAGE plpgsql
VOLATILE
CALLED ON NULL INPUT
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog
SET lock_timeout = '1s'
AS $funcion$
DECLARE
    v_login pg_catalog.pg_roles%ROWTYPE;
    v_identidad record;
BEGIN
    SELECT *
      INTO STRICT v_login
      FROM pg_catalog.pg_roles
     WHERE rolname = session_user;

    IF current_user <>
           'vec_identidad_sesiones_v1_propietario'
       OR session_user = current_user
       OR NOT v_login.rolcanlogin
       OR NOT v_login.rolinherit
       OR v_login.rolsuper
       OR v_login.rolcreatedb
       OR v_login.rolcreaterole
       OR v_login.rolreplication
       OR v_login.rolbypassrls
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members m
             JOIN pg_catalog.pg_roles g ON g.oid = m.roleid
            WHERE m.member = v_login.oid
              AND g.rolname =
                  'vec_contratacion_temporal_consultor_rrhh'
              AND NOT m.admin_option
              AND m.inherit_option
              AND NOT m.set_option
       )
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_auth_members m
            WHERE m.member = v_login.oid
       ) <> 1
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members m
            WHERE m.roleid = v_login.oid
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members m
             JOIN pg_catalog.pg_roles g ON g.oid = m.member
            WHERE g.rolname =
                  'vec_contratacion_temporal_consultor_rrhh'
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_roles g
            WHERE g.rolname =
                  'vec_contratacion_temporal_consultor_rrhh'
              AND (
                  g.rolcanlogin OR NOT g.rolinherit
                  OR g.rolsuper OR g.rolcreatedb
                  OR g.rolcreaterole OR g.rolreplication
                  OR g.rolbypassrls
              )
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'revalidación de identidad RRHH rechazada';
    END IF;

    SELECT *
      INTO v_identidad
      FROM vec_identidad_sesiones_v1.
           revalidar_autenticacion_actor_v1(
               p_autenticacion_ref, p_sesion_ref
           );
    IF NOT FOUND
       OR v_identidad.superficie <> 'interna_corporativa'
       OR NOT (
           v_identidad.garantia_observada = 'alto'
           OR (
               v_identidad.garantia_observada = 'sustancial'
               AND v_identidad.metodo_observado = 'certificado'
               AND vec_identidad_sesiones_v1.
                   admite_politica_certificado_personal_desarrollo_v1(
                       v_identidad.politica_garantia_ref,
                       v_identidad.politica_garantia_huella_sha256,
                       v_identidad.autenticacion_verificada_en
                   ) IS TRUE
           )
       )
       OR v_identidad.cuenta_privilegiada IS DISTINCT FROM false
       OR v_identidad.cuenta_ref IS DISTINCT FROM
          v_identidad.cuenta_ordinaria_ref THEN
        RETURN;
    END IF;

    RETURN QUERY SELECT
        v_identidad.autenticacion_ref,
        v_identidad.autenticacion_huella_sha256,
        v_identidad.asercion_ref,
        v_identidad.sesion_ref,
        v_identidad.control_sesion_ref,
        v_identidad.control_sesion_revision,
        v_identidad.control_sesion_huella_sha256,
        v_identidad.cuenta_ref,
        v_identidad.cuenta_ordinaria_ref,
        v_identidad.cuenta_privilegiada,
        v_identidad.superficie,
        v_identidad.metodo_observado,
        v_identidad.garantia_observada,
        v_identidad.politica_garantia_ref,
        v_identidad.politica_garantia_huella_sha256,
        v_identidad.autenticacion_verificada_en,
        v_identidad.sesion_emitida_en,
        v_identidad.sesion_valida_hasta,
        v_identidad.sesion_revalidada_en,
        session_user::text;
EXCEPTION
    WHEN no_data_found OR too_many_rows THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'revalidación de identidad RRHH rechazada';
END
$funcion$;


DO $postcondicion$
DECLARE
    v_consulta oid := pg_catalog.to_regprocedure(
        'vec_identidad_sesiones_v1.revalidar_consulta_rrhh_v1(text,text)'
    );
BEGIN
    IF v_consulta IS NULL OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc p
        JOIN pg_catalog.pg_roles r ON r.oid = p.proowner
        WHERE p.oid = v_consulta
          AND r.rolname = 'vec_identidad_sesiones_v1_propietario'
          AND p.prosecdef AND p.provolatile = 'v'
          AND p.proparallel = 'u'
          AND p.proconfig = ARRAY[
              'search_path=pg_catalog', 'lock_timeout=1s'
          ]::text[]
          AND pg_catalog.octet_length(p.prosrc) = 4039
          AND pg_catalog.encode(pg_catalog.sha256(
              pg_catalog.convert_to(p.prosrc, 'UTF8')
          ), 'hex') = '81b25aadd776dbd69fa2ab115c756b8e3c06cefc3b1fe2b52169b1e6f8c0f714'
          AND (
              SELECT pg_catalog.count(*) = 2
                 AND pg_catalog.bool_and(
                     a.grantor = p.proowner
                     AND a.grantee IN (
                         p.proowner,
                         'vec_contratacion_temporal_propietario'::regrole
                     )
                     AND a.privilege_type = 'EXECUTE'
                     AND NOT a.is_grantable
                 )
                FROM pg_catalog.aclexplode(p.proacl) a
          )
    )
    OR pg_catalog.has_function_privilege('public', v_consulta, 'EXECUTE')
    OR NOT pg_catalog.has_function_privilege(
        'vec_contratacion_temporal_propietario', v_consulta, 'EXECUTE'
    )
    OR pg_catalog.has_function_privilege(
        'vec_contratacion_temporal_consultor_rrhh', v_consulta, 'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'postcondición de consulta RRHH incumplida'
            USING ERRCODE = '55000';
    END IF;
END $postcondicion$;
COMMIT;
