-- AD221. Auditoría nominal de presentaciones; depende de la autoridad AD3.
-- Instalar antes de IS18. No registra una decisión V3 ni una firma documental.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion_atestada_v3:migracion:000221', 0));
DO $pre$
BEGIN
    IF pg_catalog.to_regclass('vec_autorizacion_atestada_v3.auditoria_consumo_v3') IS NULL
       OR pg_catalog.to_regrole('vec_identidad_sesiones_v1_propietario') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1') IS NOT NULL
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_auditoria_presentacion_certificado_v1(text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,timestamptz)') IS NOT NULL THEN
        RAISE EXCEPTION 'AD221: preimagen incompatible' USING ERRCODE='55000';
    END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

CREATE TABLE vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1 (
    operacion_ref text PRIMARY KEY CHECK (operacion_ref ~ '^opr_[A-Za-z0-9_-]{22,128}$'),
    presentacion_ref text NOT NULL CHECK (presentacion_ref ~ '^prs_[A-Za-z0-9_-]{22,128}$'),
    autenticacion_original_ref text NOT NULL CHECK (autenticacion_original_ref ~ '^aut_[A-Za-z0-9_-]{22,128}$'),
    sesion_original_ref text NOT NULL CHECK (sesion_original_ref ~ '^ses_[A-Za-z0-9_-]{22,128}$'),
    cuenta_ref text NOT NULL CHECK (cuenta_ref ~ '^cta_[A-Za-z0-9_-]{22,128}$'),
    actor_cuenta_ref text NOT NULL CHECK (actor_cuenta_ref ~ '^cta_[A-Za-z0-9_-]{22,128}$'),
    actor_autenticacion_ref text NOT NULL CHECK (actor_autenticacion_ref ~ '^aut_[A-Za-z0-9_-]{22,128}$'),
    actor_sesion_ref text NOT NULL CHECK (actor_sesion_ref ~ '^ses_[A-Za-z0-9_-]{22,128}$'),
    superficie text NOT NULL CHECK (superficie IN (
        'externa_personal','interna_corporativa','administracion_privilegiada')),
    tipo text NOT NULL CHECK (tipo IN ('apertura','reanudacion','renovacion','revocacion')),
    recibo_sha256 text NOT NULL CHECK (recibo_sha256 ~ '^[0-9a-f]{64}$'
        AND recibo_sha256 <> pg_catalog.repeat('0',64)),
    acr_original text NOT NULL CHECK (
        acr_original='urn:vec:acr:certificado-desarrollo-protegido'),
    acr_actual text,
    politica_original_ref text NOT NULL CHECK (
        politica_original_ref ~ '^pga_[A-Za-z0-9_-]{22,128}$'),
    politica_original_sha256 text NOT NULL CHECK (
        politica_original_sha256 ~ '^[0-9a-f]{64}$'
        AND politica_original_sha256 <> pg_catalog.repeat('0',64)),
    politica_actual_ref text,
    politica_actual_sha256 text,
    CHECK ((tipo='revocacion' AND acr_actual IS NULL
        AND politica_actual_ref IS NULL AND politica_actual_sha256 IS NULL)
       OR (tipo<>'revocacion' AND acr_actual IS NOT NULL
        AND politica_actual_ref IS NOT NULL AND politica_actual_sha256 IS NOT NULL
        AND acr_actual=acr_original
        AND politica_actual_ref=politica_original_ref
        AND politica_actual_sha256=politica_original_sha256)),
    registrada_en timestamptz(6) NOT NULL CHECK (pg_catalog.isfinite(registrada_en)),
    UNIQUE (presentacion_ref, operacion_ref)
);
CREATE INDEX auditoria_presentacion_certificado_sesion_idx
    ON vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1
    (sesion_original_ref, registrada_en);
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1
    FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto
    ON vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1
    FOR ALL TO vec_autorizacion_atestada_v3_propietario
    USING (current_user = 'vec_autorizacion_atestada_v3_propietario')
    WITH CHECK (current_user = 'vec_autorizacion_atestada_v3_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE
    ON vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1
    FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE
    ON vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1
    FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado();
REVOKE ALL ON TABLE vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1 FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_auditoria_presentacion_certificado_v1(
    p_operacion_ref text, p_presentacion_ref text,
    p_autenticacion_original_ref text, p_sesion_original_ref text,
    p_cuenta_ref text, p_actor_cuenta_ref text,
    p_actor_autenticacion_ref text, p_actor_sesion_ref text,
    p_superficie text, p_tipo text,
    p_recibo_sha256 text,p_acr_original text,p_acr_actual text,
    p_politica_original_ref text,p_politica_original_sha256 text,
    p_politica_actual_ref text,p_politica_actual_sha256 text,
    p_registrada_en timestamptz
) RETURNS void
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path = pg_catalog, pg_temp
SET row_security = on
SET lock_timeout = '2s'
AS $funcion$
BEGIN
    IF pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('role') <> 'none'
       OR pg_catalog.pg_is_in_recovery() THEN
        RAISE EXCEPTION 'AD221: transaccion no acreditada' USING ERRCODE='42501';
    END IF;
    INSERT INTO vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1 (
        operacion_ref, presentacion_ref, autenticacion_original_ref,
        sesion_original_ref, cuenta_ref, actor_cuenta_ref,
        actor_autenticacion_ref,actor_sesion_ref,
        superficie, tipo, recibo_sha256,acr_original,acr_actual,
        politica_original_ref,politica_original_sha256,
        politica_actual_ref,politica_actual_sha256,
        registrada_en
    ) VALUES (
        p_operacion_ref, p_presentacion_ref, p_autenticacion_original_ref,
        p_sesion_original_ref, p_cuenta_ref, p_actor_cuenta_ref,
        p_actor_autenticacion_ref,p_actor_sesion_ref,p_superficie, p_tipo,
        p_recibo_sha256,p_acr_original,p_acr_actual,
        p_politica_original_ref,p_politica_original_sha256,
        p_politica_actual_ref,p_politica_actual_sha256,p_registrada_en
    );
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_auditoria_presentacion_certificado_v1(
    text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,timestamptz) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3
    TO vec_identidad_sesiones_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_auditoria_presentacion_certificado_v1(
    text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,timestamptz)
    TO vec_identidad_sesiones_v1_propietario;
COMMIT;
