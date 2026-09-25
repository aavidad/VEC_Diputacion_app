\set ON_ERROR_STOP on
-- Sumidero durable de RegistroResultadoEjecucionMarcaje (fallos confirmados y
-- resultados indeterminados). Segregado del negocio: rol propio, función propia
-- y transacción independiente de la del marcaje. Solo referencias opacas.
BEGIN;
RESET ROLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:000003',0));
DO $pre$
BEGIN
    IF to_regnamespace('vec_cronos_v1') IS NULL OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_cronos_v1_auditor') THEN
        RAISE EXCEPTION 'resultado de ejecución Cronos incompatible' USING ERRCODE='55000';
    END IF;
END
$pre$;
CREATE ROLE vec_cronos_v1_auditor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
GRANT USAGE ON SCHEMA vec_cronos_v1 TO vec_cronos_v1_auditor;
DO $conectar$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_cronos_v1_auditor', current_database()); END $conectar$;
SET LOCAL ROLE vec_cronos_v1_propietario;
CREATE TABLE vec_cronos_v1.resultado_ejecucion_marcaje (
    resultado_ref text PRIMARY KEY DEFAULT 'resultado:cronos:'||gen_random_uuid()::text,
    decision_ref text NOT NULL CHECK (decision_ref ~ '^[-A-Za-z0-9_.:]{1,255}$'),
    contexto_ref text NOT NULL CHECK (contexto_ref ~ '^[-A-Za-z0-9_.:]{1,255}$'),
    actor_ref text NOT NULL CHECK (actor_ref ~ '^per_[-A-Za-z0-9_]{1,128}$'),
    perfil_ref text NOT NULL CHECK (perfil_ref ~ '^prf_[-A-Za-z0-9_]{1,128}$'),
    accion text NOT NULL CHECK (accion='cronos.marcaje.propio.registrar'),
    recurso_ref text NOT NULL CHECK (recurso_ref ~ '^marcaje:cronos:[-A-Za-z0-9_]{1,128}$'),
    resultado text NOT NULL CHECK (resultado IN ('fallo_confirmado','resultado_indeterminado')),
    causa text NOT NULL CHECK (causa IN ('persistencia','conflicto','recibo_invalido','rollback','commit')),
    observada_en timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp()
);
REVOKE ALL ON vec_cronos_v1.resultado_ejecucion_marcaje FROM PUBLIC;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.resultado_ejecucion_marcaje
    FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
CREATE FUNCTION vec_cronos_v1.registrar_resultado_ejecucion_marcaje_v1(
    p_decision_ref text,p_contexto_ref text,p_actor_ref text,p_perfil_ref text,p_accion text,
    p_recurso_ref text,p_resultado text,p_causa text,p_observada_en timestamptz
) RETURNS text
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE ref text;
BEGIN
    IF session_user=current_user OR NOT pg_has_role(session_user,'vec_cronos_v1_auditor','MEMBER')
       OR pg_has_role(session_user,'vec_cronos_v1_ejecutor','MEMBER') THEN
        RAISE EXCEPTION 'auditor Cronos inválido' USING ERRCODE='42501';
    END IF;
    INSERT INTO vec_cronos_v1.resultado_ejecucion_marcaje(decision_ref,contexto_ref,actor_ref,perfil_ref,accion,recurso_ref,resultado,causa,observada_en)
    VALUES (p_decision_ref,p_contexto_ref,p_actor_ref,p_perfil_ref,p_accion,p_recurso_ref,p_resultado,p_causa,p_observada_en)
    RETURNING resultado_ref INTO ref;
    RETURN ref;
END
$f$;
REVOKE ALL ON FUNCTION vec_cronos_v1.registrar_resultado_ejecucion_marcaje_v1(text,text,text,text,text,text,text,text,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_cronos_v1.registrar_resultado_ejecucion_marcaje_v1(text,text,text,text,text,text,text,text,timestamptz) TO vec_cronos_v1_auditor;
COMMIT;
