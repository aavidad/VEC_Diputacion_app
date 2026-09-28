\set ON_ERROR_STOP on
-- AD3-104: perfil nominal para recuperar el recibo mínimo de respuesta CT.
-- Solo exige el núcleo y la audiencia CT existentes; no depende de AD3-102/103.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000104',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $ampliar$
DECLARE
    f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    v_def text; v_acl aclitem[]; v_propietario oid; v_config text[];
    v_marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
    v_extension text:=$perfil$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_recibo_respuesta_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.respuesta.consultar_recibo'
               AND d ->> 'accion' IS NOT DISTINCT FROM c ->> 'operacion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'respuesta_recibida_comunicacion_ct'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
               AND d ->> 'recurso_ref' IS NOT DISTINCT FROM c ->> 'efecto_ref'
               AND d ->> 'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c ->> 'huella_efecto_sha256'
               AND d -> 'campos_permitidos' IS NOT DISTINCT FROM
                   '["auditoria_ref","comunicacion_ref","estado","expediente_ref","justificante_ref","organizacion_ref","recibo_ref","registrada_en","respuesta"]'::jsonb
               AND d -> 'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
           )
$perfil$;
BEGIN
    IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_justificante_respuesta_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_recibo_respuesta_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
        RAISE EXCEPTION 'AD3-104: preimagen incompatible' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_functiondef(p.oid),p.proacl,p.proowner,p.proconfig
      INTO STRICT v_def,v_acl,v_propietario,v_config FROM pg_proc p WHERE p.oid=f AND p.prosecdef;
    IF v_propietario IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
       OR v_config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
       OR length(v_def)-length(replace(v_def,v_marca,''))<>length(v_marca)
       OR strpos(v_def,'consulta_recibo_respuesta_ct')<>0
       OR strpos(v_def,'consulta_justificante_respuesta_ct')=0 THEN
        RAISE EXCEPTION 'AD3-104: núcleo incompatible' USING ERRCODE='55000';
    END IF;
    EXECUTE replace(v_def,v_marca,v_extension||v_marca);
    IF (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM v_acl
       OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM v_propietario
       OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM v_config THEN
        RAISE EXCEPTION 'AD3-104: autoridad o ACL alterada' USING ERRCODE='55000';
    END IF;
END $ampliar$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_recibo_respuesta_ct_v3_atestada(
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE (
    decision_ref text,efecto_ref text,huella_efecto_sha256 text,
    consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s'
AS $funcion$
DECLARE v_consumo record;
BEGIN
    SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
        'consulta_recibo_respuesta_ct',p_capacidad,p_decision,p_motivo,p_contexto,
        p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'AD3-104: consumo nuevo requerido' USING ERRCODE='P1393';
    END IF;
    RETURN QUERY SELECT v_consumo.decision_ref,v_consumo.efecto_ref,v_consumo.huella_efecto_sha256,
        v_consumo.consumo_huella_sha256,v_consumo.auditoria_ref,v_consumo.consumida_en,true;
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_recibo_respuesta_ct_v3_atestada(
    bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC,vec_autorizacion_atestada_v3_consumidor,vec_autorizacion_atestada_v3_emisor,
    vec_contratacion_temporal_ejecutor,vec_contratacion_temporal_migrador;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_recibo_respuesta_ct_v3_atestada(
    bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_propietario;
COMMIT;
