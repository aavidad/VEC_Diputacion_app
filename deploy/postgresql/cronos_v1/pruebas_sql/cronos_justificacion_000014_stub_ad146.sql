\set ON_ERROR_STOP on
-- SOLO para el arnés PostgreSQL desechable de CRN14. Estas fachadas AD146 de
-- prueba comprueban sesión, audiencia y nonce; no verifican MAC, COSE,
-- gobierno, revocación ni concesiones V3 reales. No es una migración y no se
-- instala sobre una base con historia.
DO $stub$
DECLARE
    fachada text;
    audiencia text;
BEGIN
    IF to_regclass('vec_autorizacion_atestada_v3.stub_consumo') IS NULL THEN
        RAISE EXCEPTION 'stub AD146 requiere el arnés de autorización previo'
            USING ERRCODE = '55000';
    END IF;

    FOR fachada, audiencia IN VALUES
        ('consumir_cronos_justificacion_consulta_v3_atestada',
         'vec_cronos_v1.justificacion.consultar.v1'),
        ('consumir_cronos_justificacion_recibo_v3_atestada',
         'vec_cronos_v1.justificacion.recibo.consultar.v1'),
        ('registrar_y_consumir_cronos_justificacion_anexo_v3_atestada',
         'vec_cronos_v1.justificacion.v1'),
        ('registrar_y_consumir_cronos_justificacion_revision_v3_atestada',
         'vec_cronos_v1.justificacion.v1')
    LOOP
        EXECUTE format($f$
CREATE FUNCTION vec_autorizacion_atestada_v3.%I(
    p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
    p_persona_version numeric, p_perfil_version numeric, p_payload bytea,
    p_sobre bytea, p_evidencia bytea, p_raiz bytea
)
RETURNS TABLE(
    decision_ref text, efecto_ref text, huella_efecto_sha256 text,
    consumo_huella_sha256 text, auditoria_ref text, consumida_en timestamptz,
    consumo_nuevo boolean
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog
AS $c$
DECLARE
    capacidad jsonb;
    nuevo boolean;
BEGIN
    IF NOT pg_has_role(session_user, 'vec_cronos_v1_ejecutor', 'MEMBER') THEN
        RAISE EXCEPTION 'stub AD146: sesión ajena' USING ERRCODE = '42501';
    END IF;
    capacidad := convert_from(p_capacidad, 'UTF8')::jsonb;
    IF capacidad->>'audiencia_consumo' IS DISTINCT FROM %L THEN
        RAISE EXCEPTION 'stub AD146: audiencia' USING ERRCODE = '42501';
    END IF;
    INSERT INTO vec_autorizacion_atestada_v3.stub_consumo(nonce, audiencia)
    VALUES (capacidad->>'nonce', %L)
    ON CONFLICT DO NOTHING
    RETURNING true INTO nuevo;
    RETURN QUERY
    SELECT capacidad->>'decision_ref', capacidad->>'efecto_ref',
           capacidad->>'huella_efecto_sha256',
           encode(sha256(convert_to(capacidad->>'nonce', 'UTF8')), 'hex'),
           'auditoria:stub:' || (capacidad->>'nonce'), clock_timestamp(),
           coalesce(nuevo, false);
END
$c$;
$f$, fachada, audiencia, audiencia);
        EXECUTE format(
            'REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.%I(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC',
            fachada
        );
        EXECUTE format(
            'GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.%I(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario',
            fachada
        );
    END LOOP;
END
$stub$;
