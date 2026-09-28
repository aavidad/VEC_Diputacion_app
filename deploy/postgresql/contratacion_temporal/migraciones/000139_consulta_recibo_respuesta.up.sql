\set ON_ERROR_STOP on
-- CT139: lectura mínima del recibo CT56 con decisión fresca AD3-104.
-- Ausencia, otra organización, expediente o comunicación comparten el mismo resultado.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000139',0));
DO $pre$
BEGIN
 IF to_regclass('vec_contratacion_temporal.respuesta_recibida_rrhh') IS NULL
    OR to_regclass('vec_contratacion_temporal.historia_respuesta_recibida_rrhh') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_recibo_respuesta_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_recibo_respuesta_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT139: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_contratacion_temporal.consultar_recibo_respuesta_rrhh_v1(
    p_material text,
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
 s jsonb; d jsonb; v_contexto jsonb; v_hash text; v_material_huella text;
 v_consumo record; v_respuesta record;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off' THEN
    RAISE EXCEPTION 'CT139: consulta denegada' USING ERRCODE='P1393';
 END IF;
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 1024 THEN
    RAISE EXCEPTION 'CT139: material inválido' USING ERRCODE='P1390'; END IF;
 BEGIN s:=p_material::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'CT139: JSON inválido' USING ERRCODE='P1390'; END;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(s,ARRAY['OrganizacionRef','ExpedienteRef','ComunicacionRef']) IS NOT TRUE
    OR (SELECT count(*) FROM json_each(p_material::json))<>3
    OR jsonb_typeof(s->'OrganizacionRef') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'ExpedienteRef') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'ComunicacionRef') IS DISTINCT FROM 'string'
    OR s->>'OrganizacionRef' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR s->>'ExpedienteRef' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR s->>'ComunicacionRef' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
    RAISE EXCEPTION 'CT139: referencias inválidas' USING ERRCODE='P1390'; END IF;
 v_material_huella:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 v_hash:=encode(sha256(convert_to(
    '{"ambitos":{"expediente_ref":"'||(s->>'ExpedienteRef')||
    '","organizacion_ref":"'||(s->>'OrganizacionRef')||
    '"},"atributos":{"material_sha256":"'||v_material_huella||'"}}','UTF8')),'hex');
 IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
    OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 1 AND 524288 THEN
    RAISE EXCEPTION 'CT139: decisión ausente' USING ERRCODE='P1393'; END IF;
 BEGIN
    d:=convert_from(p_decision,'UTF8')::jsonb;
    v_contexto:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'CT139: decisión inválida' USING ERRCODE='P1393'; END;
 IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.llamamiento.respuesta.consultar_recibo'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'respuesta_recibida_comunicacion_ct'
    OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM s->>'ComunicacionRef'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_hash
    OR d->>'principal_id' IS DISTINCT FROM v_contexto->>'principal_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM v_contexto->>'perfil_activo_ref'
    OR v_contexto->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR v_contexto->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR d->'campos_permitidos' IS DISTINCT FROM
       '["auditoria_ref","comunicacion_ref","estado","expediente_ref","justificante_ref","organizacion_ref","recibo_ref","registrada_en","respuesta"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
    RAISE EXCEPTION 'CT139: decisión divergente' USING ERRCODE='P1393'; END IF;
 -- AD3 conserva un evento de consulta incluso si CT no encuentra fila o la
 -- fila pertenece a otra organización o actor. El llamador confirma ambos casos.
 BEGIN
    SELECT * INTO STRICT v_consumo FROM
      vec_autorizacion_atestada_v3.registrar_y_consumir_recibo_respuesta_ct_v3_atestada(
       p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
       p_payload,p_sobre,p_evidencia,p_raiz);
 EXCEPTION
    WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN
      RAISE EXCEPTION 'CT139: consumo transitorio no disponible' USING ERRCODE='P1394';
    WHEN insufficient_privilege OR invalid_authorization_specification OR SQLSTATE 'P1393' THEN
      RAISE EXCEPTION 'CT139: consumo denegado' USING ERRCODE='P1393';
    WHEN OTHERS THEN
      RAISE EXCEPTION 'CT139: consumo no disponible' USING ERRCODE='P1394';
 END;
 IF v_consumo.consumo_nuevo IS NOT TRUE
    OR v_consumo.efecto_ref IS DISTINCT FROM s->>'ComunicacionRef'
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_hash
    OR v_consumo.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR v_consumo.auditoria_ref IS NULL THEN
    RAISE EXCEPTION 'CT139: consumo divergente' USING ERRCODE='P1393'; END IF;

 SELECT r.organizacion_ref,r.expediente_ref,r.comunicacion_ref,r.respuesta,r.justificante_ref,
        r.recibo_ref,h.auditoria_ref,r.registrada_en,r.estado
   INTO v_respuesta
   FROM vec_contratacion_temporal.respuesta_recibida_rrhh r
   JOIN vec_contratacion_temporal.historia_respuesta_recibida_rrhh h
     ON h.justificante_ref=r.justificante_ref
   JOIN vec_contratacion_temporal.comunicacion_llamamiento_local c
     ON c.comunicacion_ref=r.comunicacion_ref
   JOIN vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
     ON e.clave_idempotencia=r.seleccion_clave
  WHERE r.organizacion_ref=s->>'OrganizacionRef'
    AND r.expediente_ref=s->>'ExpedienteRef'
    AND r.comunicacion_ref=s->>'ComunicacionRef'
    AND h.actor_ref=r.actor_ref AND h.perfil_ref=r.perfil_ref
    AND c.organizacion_ref=r.organizacion_ref AND c.expediente_ref=r.expediente_ref
    AND c.llamamiento_ref=r.llamamiento_ref AND c.seleccion_clave=r.seleccion_clave
    AND c.estado='registrada_localmente' AND c.version_resultante=2
    AND c.recibo_json->>'ComunicacionRef'=c.comunicacion_ref
    AND c.recibo_json->'Solicitud'->>'OrganizacionRef'=r.organizacion_ref
    AND c.recibo_json->'Solicitud'->>'ExpedienteRef'=r.expediente_ref
    AND c.recibo_json->'Solicitud'->>'LlamamientoRef'=r.llamamiento_ref
    AND e.situacion='confirmada'
    AND e.solicitud_json->>'organizacion_ref'=r.organizacion_ref
    AND e.solicitud_json->>'expediente_ref'=r.expediente_ref
    AND e.recibo_json->>'organizacion_ref'=r.organizacion_ref
    AND e.recibo_json->>'expediente_ref'=r.expediente_ref
    AND e.recibo_json->>'llamamiento_ref'=r.llamamiento_ref
    AND e.recibo_json->>'recibo_ref'=c.recibo_json->'Solicitud'->>'PruebaEntregaRef'
    AND r.estado='registrada_por_rrhh'
    AND r.material_json->>'OrganizacionRef'=r.organizacion_ref
    AND r.material_json->>'ExpedienteRef'=r.expediente_ref
    AND r.material_json->>'LlamamientoRef'=r.llamamiento_ref
    AND r.material_json->>'ComunicacionRef'=r.comunicacion_ref
    AND r.recibo_json->>'JustificanteRef'=r.justificante_ref
    AND r.recibo_json->>'ReciboRef'=r.recibo_ref
    AND r.recibo_json->>'AuditoriaRef'=h.auditoria_ref
    AND r.recibo_json->>'Estado'=r.estado
    AND r.recibo_json->>'RegistradaEn'=to_char(r.registrada_en,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    AND r.material_json->>'Respuesta'=r.respuesta
    AND r.recibo_json->'Solicitud'=r.material_json;
 IF NOT FOUND THEN RETURN jsonb_build_object('Encontrado',false,'Recibo',jsonb_build_object()); END IF;
 RETURN jsonb_build_object('Encontrado',true,'Recibo',jsonb_build_object(
    'OrganizacionRef',v_respuesta.organizacion_ref,
    'ExpedienteRef',v_respuesta.expediente_ref,
    'ComunicacionRef',v_respuesta.comunicacion_ref,
    'Respuesta',v_respuesta.respuesta,
    'JustificanteRef',v_respuesta.justificante_ref,
    'ReciboRef',v_respuesta.recibo_ref,
    'AuditoriaRef',v_respuesta.auditoria_ref,
    'RegistradaEn',to_char(v_respuesta.registrada_en,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'Estado',v_respuesta.estado));
END $funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_recibo_respuesta_rrhh_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC,vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_recibo_respuesta_rrhh_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_ejecutor;
COMMIT;
