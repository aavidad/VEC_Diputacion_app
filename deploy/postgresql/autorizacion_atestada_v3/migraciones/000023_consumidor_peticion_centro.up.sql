\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000023',0));

-- Añade un perfil consumidor; no duplica validación criptográfica de AD3.
DO $ampliar$
DECLARE
    v_def text; v_acl aclitem[]; v_owner oid;
    v_marca text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
    v_extension text := $perfil$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'peticion_centro'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND c ->> 'operacion' = ANY (ARRAY[
                   'contratacion_temporal.peticion_centro.presentar',
                   'contratacion_temporal.peticion_centro.ratificar',
                   'contratacion_temporal.peticion_centro.consultar'
               ])
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'peticion_centro'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_peticion_centro'
           )
$perfil$;
BEGIN
    IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_peticion_centro_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
        RAISE EXCEPTION 'petición de centro ya instalada' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_functiondef(p.oid),p.proacl,p.proowner
      INTO STRICT v_def,v_acl,v_owner
      FROM pg_proc p
     WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
       AND p.prosecdef;
    IF length(v_def)-length(replace(v_def,v_marca,''))<>length(v_marca)
       OR strpos(v_def,'peticion_centro')<>0
       OR strpos(v_def,'p_perfil_mutacion IS NOT DISTINCT FROM ''organizacion_preparacion''')=0
       OR strpos(v_def,'(d ->> ''valida_hasta'')::timestamptz')=0 THEN
        RAISE EXCEPTION 'núcleo incompatible para petición de centro' USING ERRCODE='55000';
    END IF;
    EXECUTE replace(v_def,v_marca,v_extension||v_marca);
    IF (SELECT proacl FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl
       OR (SELECT proowner FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_owner THEN
        RAISE EXCEPTION 'petición de centro alteró autoridad del núcleo' USING ERRCODE='55000';
    END IF;
END
$ampliar$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_peticion_centro_v3_atestada(
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE (
    decision_ref text,efecto_ref text,huella_efecto_sha256 text,
    consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET lock_timeout = '2s'
AS $funcion$
DECLARE v_consumo record;
BEGIN
    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
        'peticion_centro',p_capacidad,p_decision,p_motivo,p_contexto,
        p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'petición de centro requiere consumo nuevo' USING ERRCODE='P0673';
    END IF;
    RETURN QUERY SELECT v_consumo.decision_ref,v_consumo.efecto_ref,
        v_consumo.huella_efecto_sha256,v_consumo.consumo_huella_sha256,
        v_consumo.auditoria_ref,v_consumo.consumida_en,true;
END
$funcion$;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_peticion_centro_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_peticion_centro_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC,vec_autorizacion_atestada_v3_consumidor,vec_autorizacion_atestada_v3_emisor,
    vec_contratacion_temporal_ejecutor,vec_contratacion_temporal_migrador,
    vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_peticion_centro_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_propietario;
COMMIT;
