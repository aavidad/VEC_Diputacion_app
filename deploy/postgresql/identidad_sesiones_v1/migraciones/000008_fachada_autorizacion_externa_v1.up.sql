-- AUT externo solo puede acreditar la sesión de un LOGIN aprobado para el
-- proceso externo. El propietario compartido de AUT no puede sondearla desde
-- un LOGIN interno: session_user conserva el origen bajo SECURITY DEFINER.
BEGIN;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

DO $pre$
BEGIN
    IF pg_catalog.to_regclass('vec_identidad_externa_v1.llamante_autorizacion') IS NOT NULL
       OR pg_catalog.to_regprocedure('vec_identidad_externa_v1.acreditar_sesion_externa_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)') IS NOT NULL
       OR pg_catalog.to_regprocedure('vec_identidad_externa_v1.revalidar_sesion_y_cuentas_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)') IS NULL
       OR pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL THEN
        RAISE EXCEPTION 'fachada AUT externa: precondiciones incumplidas' USING ERRCODE='55000';
    END IF;
END $pre$;

-- Solo el propietario de Identidad provisiona esta lista fuera de la petición.
-- Un registro por LOGIN de proceso, con grupo único y aprobación trazable.
CREATE TABLE vec_identidad_externa_v1.llamante_autorizacion (
    login name PRIMARY KEY,
    grupo_esperado name NOT NULL,
    aprobacion_ref text NOT NULL UNIQUE,
    aprobada_por name NOT NULL DEFAULT session_user,
    aprobada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
    CHECK (aprobacion_ref ~ '^apr_[A-Za-z0-9_-]{22,128}$')
);
CREATE TRIGGER llamante_autorizacion_inmutable
    BEFORE UPDATE OR DELETE OR TRUNCATE
    ON vec_identidad_externa_v1.llamante_autorizacion
    FOR EACH STATEMENT EXECUTE FUNCTION
        vec_identidad_externa_v1.rechazar_mutacion_v1();
ALTER TABLE vec_identidad_externa_v1.llamante_autorizacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_identidad_externa_v1.llamante_autorizacion FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_identidad_externa_v1.llamante_autorizacion
    FOR ALL TO vec_identidad_sesiones_v1_propietario
    USING (current_user='vec_identidad_sesiones_v1_propietario')
    WITH CHECK (current_user='vec_identidad_sesiones_v1_propietario');
REVOKE ALL ON TABLE vec_identidad_externa_v1.llamante_autorizacion FROM PUBLIC;
REVOKE ALL ON TYPE vec_identidad_externa_v1.llamante_autorizacion FROM PUBLIC;

CREATE FUNCTION vec_identidad_externa_v1.acreditar_sesion_externa_v1(
    p_autenticacion_ref text, p_autenticacion_huella_sha256 text,
    p_asercion_ref text, p_sesion_ref text, p_cuenta_ref text,
    p_cuenta_ordinaria_ref text, p_cuenta_privilegiada boolean,
    p_superficie text, p_metodo_observado text, p_garantia_observada text,
    p_politica_garantia_ref text, p_politica_garantia_huella_sha256 text,
    p_autenticacion_verificada_en timestamptz, p_sesion_emitida_en timestamptz,
    p_control_sesion_ref text, p_control_sesion_revision_texto text,
    p_control_sesion_estado text, p_control_sesion_huella_sha256 text,
    p_sesion_revalidada_en timestamptz, p_sesion_valida_hasta timestamptz
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp AS $f$
DECLARE aprobado record;
BEGIN
    SELECT l.login,l.grupo_esperado INTO aprobado
      FROM vec_identidad_externa_v1.llamante_autorizacion l
     WHERE l.login=session_user::name;
    IF NOT FOUND OR aprobado.login IS DISTINCT FROM session_user::name
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles r
           JOIN pg_catalog.pg_auth_members m ON m.roleid=r.oid
           JOIN pg_catalog.pg_roles u ON u.oid=m.member
           WHERE r.rolname=aprobado.grupo_esperado
             AND u.rolname=session_user
             AND u.rolcanlogin AND u.rolinherit
             AND NOT u.rolsuper AND NOT u.rolcreatedb
             AND NOT u.rolcreaterole AND NOT u.rolreplication
             AND NOT u.rolbypassrls AND u.rolconfig IS NULL
             AND NOT r.rolcanlogin AND NOT r.rolsuper AND NOT r.rolbypassrls
             AND NOT r.rolcreatedb AND NOT r.rolcreaterole
             AND NOT r.rolreplication AND r.rolconfig IS NULL
             AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
       ) OR (
           SELECT count(*) FROM pg_catalog.pg_auth_members m
           JOIN pg_catalog.pg_roles login ON login.oid=m.member
           WHERE login.rolname=session_user
       ) <> 1 OR (
           SELECT count(*) FROM pg_catalog.pg_roles alcanzable
           JOIN pg_catalog.pg_roles login ON login.rolname=session_user
           WHERE alcanzable.oid<>login.oid
             AND pg_catalog.pg_has_role(login.oid,alcanzable.oid,'MEMBER')
       ) <> 1 THEN
        RETURN false;
    END IF;
    RETURN COALESCE(vec_identidad_externa_v1.revalidar_sesion_y_cuentas_v1(
        p_autenticacion_ref,p_autenticacion_huella_sha256,
        p_asercion_ref,p_sesion_ref,p_cuenta_ref,p_cuenta_ordinaria_ref,
        p_cuenta_privilegiada,p_superficie,p_metodo_observado,
        p_garantia_observada,p_politica_garantia_ref,
        p_politica_garantia_huella_sha256,p_autenticacion_verificada_en,
        p_sesion_emitida_en,p_control_sesion_ref,
        p_control_sesion_revision_texto,p_control_sesion_estado,
        p_control_sesion_huella_sha256,p_sesion_revalidada_en,
        p_sesion_valida_hasta),false);
EXCEPTION WHEN data_exception OR invalid_parameter_value THEN
    RETURN false;
END $f$;

REVOKE ALL ON FUNCTION vec_identidad_externa_v1.acreditar_sesion_externa_v1(
    text,text,text,text,text,text,boolean,text,text,text,text,text,
    timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)
    FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_identidad_externa_v1 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_identidad_externa_v1.acreditar_sesion_externa_v1(
    text,text,text,text,text,text,boolean,text,text,text,text,text,
    timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)
    TO vec_autorizacion_propietario;
COMMIT;
