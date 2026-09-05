\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000057_consulta_justificante_respuesta_recibida',0));

-- Sin nuevas tablas ni cambios a CT56. La consulta acredita únicamente la
-- declaración persistida y su selección original; no correo, aceptación o plazo.
DO $dependencias$
BEGIN
    IF to_regclass('vec_contratacion_temporal.respuesta_recibida_rrhh') IS NULL
       OR NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
           WHERE n.nspname='vec_autorizacion_atestada_v3'
             AND p.proname='registrar_y_consumir_justificante_respuesta_ct_v3_atestada'
             AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef) THEN
        RAISE EXCEPTION 'faltan declaración o consumidor nominal de justificante' USING ERRCODE='55000';
    END IF;
END
$dependencias$;

CREATE FUNCTION vec_contratacion_temporal.consultar_justificante_respuesta_recibida_rrhh_v1(
    p_material text,
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    s jsonb; d jsonb; v_referencia text;
    v_material_huella text; v_contexto_huella text; v_consumo record;
    v_respuesta jsonb; v_seleccion jsonb;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario'
       OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
       OR current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'off' THEN
        RAISE EXCEPTION 'consulta de justificante denegada' USING ERRCODE='P0573';
    END IF;
    IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 16384 THEN
        RAISE EXCEPTION 'material de consulta inválido' USING ERRCODE='P0570';
    END IF;
    BEGIN
        s:=p_material::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'JSON de consulta inválido' USING ERRCODE='P0570';
    END;
    IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(s,ARRAY[
        'ClaveIdempotencia','OrganizacionRef','ExpedienteRef','LlamamientoRef',
        'ComunicacionRef','VersionEsperada','Respuesta','PruebaRespuestaRef'
    ]) IS NOT TRUE THEN
        RAISE EXCEPTION 'campos de consulta inválidos' USING ERRCODE='P0570';
    END IF;
    -- El material es json.Marshal de la solicitud directa. No envoltorios,
    -- claves duplicadas ni reserialización previa a calcular su huella.
    IF (SELECT count(*) FROM json_each(p_material::json))<>8
       OR jsonb_typeof(s->'ClaveIdempotencia') IS DISTINCT FROM 'string'
       OR (s->>'ClaveIdempotencia')!~'^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
       OR s->>'ClaveIdempotencia'='00000000-0000-4000-8000-000000000000'
       OR jsonb_typeof(s->'Respuesta') IS DISTINCT FROM 'string'
       OR (s->>'Respuesta') NOT IN ('aceptacion','renuncia') THEN
        RAISE EXCEPTION 'solicitud de consulta inválida' USING ERRCODE='P0570';
    END IF;
    FOREACH v_referencia IN ARRAY ARRAY[
        'OrganizacionRef','ExpedienteRef','LlamamientoRef','ComunicacionRef','PruebaRespuestaRef'
    ] LOOP
        IF jsonb_typeof(s->v_referencia) IS DISTINCT FROM 'string'
           OR (s->>v_referencia)!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
            RAISE EXCEPTION 'referencia de consulta inválida' USING ERRCODE='P0570';
        END IF;
    END LOOP;
    IF s->'VersionEsperada' IS DISTINCT FROM '2'::jsonb THEN
        RAISE EXCEPTION 'versión de comunicación incompatible' USING ERRCODE='P0572';
    END IF;
    v_material_huella:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
    v_contexto_huella:=encode(sha256(convert_to(
        '{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
        '"},"atributos":{"material_sha256":"'||v_material_huella||'"}}','UTF8')),'hex');
    IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
        RAISE EXCEPTION 'decisión de consulta inválida' USING ERRCODE='P0573';
    END IF;
    BEGIN
        d:=convert_from(p_decision,'UTF8')::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'decisión de consulta inválida' USING ERRCODE='P0573';
    END;
    IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.llamamiento.respuesta.consultar_justificante'
       OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'justificante_respuesta_recibida_ct'
       OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
       OR d->>'recurso_ref' IS DISTINCT FROM s->>'ExpedienteRef'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'autorización de consulta divergente' USING ERRCODE='P0573';
    END IF;
    -- También al repetir la consulta: la autorización del registro original
    -- no concede este acceso. No se compara el lector con el actor creador.
    BEGIN
        SELECT * INTO STRICT v_consumo
          FROM vec_autorizacion_atestada_v3.registrar_y_consumir_justificante_respuesta_ct_v3_atestada(
            p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,
            p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    EXCEPTION
        WHEN insufficient_privilege OR data_exception OR SQLSTATE 'P0573' THEN
            RAISE EXCEPTION 'consumo de consulta denegado' USING ERRCODE='P0573';
    END;
    IF v_consumo.consumo_nuevo IS NOT TRUE
       OR v_consumo.efecto_ref IS DISTINCT FROM s->>'ExpedienteRef'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'consulta requiere consumo nuevo ligado al material' USING ERRCODE='P0573';
    END IF;

    -- Una única lectura, después del consumo, solo sobre antecedentes CT.
    -- Las dos referencias FK de selección deben coincidir y conservar la
    -- prueba original que permitió registrar el aviso local de CT54.
    SELECT r.recibo_json,e.recibo_json INTO v_respuesta,v_seleccion
      FROM vec_contratacion_temporal.respuesta_recibida_rrhh r
      JOIN vec_contratacion_temporal.comunicacion_llamamiento_local c
        ON c.comunicacion_ref=r.comunicacion_ref
      JOIN vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
        ON e.clave_idempotencia=r.seleccion_clave AND e.clave_idempotencia=c.seleccion_clave
     WHERE r.justificante_ref=s->>'PruebaRespuestaRef'
       AND r.organizacion_ref=s->>'OrganizacionRef'
       AND r.expediente_ref=s->>'ExpedienteRef'
       AND r.llamamiento_ref=s->>'LlamamientoRef'
       AND r.comunicacion_ref=s->>'ComunicacionRef'
       AND r.respuesta=s->>'Respuesta'
       AND r.version_comunicacion=2 AND r.estado='registrada_por_rrhh'
       AND c.organizacion_ref=r.organizacion_ref AND c.expediente_ref=r.expediente_ref
       AND c.llamamiento_ref=r.llamamiento_ref
       AND c.version_resultante=2 AND c.estado='registrada_localmente'
       AND e.situacion='confirmada'
       AND e.solicitud_json->>'organizacion_ref'=r.organizacion_ref
       AND e.solicitud_json->>'expediente_ref'=r.expediente_ref
       AND e.recibo_json->>'organizacion_ref'=r.organizacion_ref
       AND e.recibo_json->>'expediente_ref'=r.expediente_ref
       AND e.recibo_json->>'llamamiento_ref'=r.llamamiento_ref
       AND e.recibo_json->'propuesta_generada'='true'::jsonb
       AND e.recibo_json->>'recibo_ref'=c.material_json->'solicitud'->>'PruebaEntregaRef'
       AND r.recibo_json->>'JustificanteRef'=r.justificante_ref
       AND r.recibo_json->>'ReciboRef'=r.recibo_ref
       AND r.recibo_json->'Solicitud'=r.material_json
       AND r.recibo_json->>'Estado'='registrada_por_rrhh';
    IF NOT FOUND THEN
        RAISE EXCEPTION 'justificante o antecedentes incompatibles' USING ERRCODE='P0572';
    END IF;
    -- Originales íntegros: no cambiar Estado a replay ni fabricar referencias,
    -- fechas, auditorías comerciales o una evaluación positiva del plazo.
    RETURN jsonb_build_object('Respuesta',v_respuesta,'Seleccion',v_seleccion);
EXCEPTION
    WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN
        RAISE EXCEPTION 'consulta de justificante no disponible' USING ERRCODE='P0574';
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_justificante_respuesta_recibida_rrhh_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_justificante_respuesta_recibida_rrhh_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_ejecutor;
COMMIT;
