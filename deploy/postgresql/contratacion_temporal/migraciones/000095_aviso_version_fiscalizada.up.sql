\set ON_ERROR_STOP on
-- Fuente candidata sin reserva de número ni instalación.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

DO $dependencias$
BEGIN
    IF to_regprocedure('vec_contratacion_temporal.leer_expediente_seleccion_v1(text,text,bigint)') IS NULL
       OR to_regprocedure('vec_contratacion_temporal.leer_expediente_seleccion_v2(text,text,bigint)') IS NULL
       OR to_regprocedure('vec_contratacion_temporal.confirmacion_canonica_seleccion_llamamiento_o6_v1(jsonb,text,jsonb,text)') IS NULL
       OR NOT EXISTS (SELECT 1 FROM pg_attribute
          WHERE attrelid='vec_contratacion_temporal.resolucion_manual_respuesta_rrhh'::regclass
            AND attname='continuacion_recibo' AND atttypid='jsonb'::regtype AND NOT attisdropped) THEN
        RAISE EXCEPTION 'faltan lectores o confirmación de llamamiento' USING ERRCODE='55000';
    END IF;
END
$dependencias$;

-- Lector interno preparatorio, sin efectos ni permiso funcional nuevo.
-- La versión procede de la selección confirmada; la cabeza actual puede avanzar.
-- La autorización fresca del aviso continúa en el backend y escritor existentes.
CREATE FUNCTION vec_contratacion_temporal.leer_expediente_aviso_confirmado_v1(
    p_organizacion text, p_expediente text, p_llamamiento text
) RETURNS TABLE(expediente_json jsonb, version_actual bigint)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '5s'
AS $funcion$
DECLARE
    v_seleccion vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
    v_artefacto jsonb;
    v_datos jsonb;
    v_version bigint;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR session_user = current_user
       OR NOT pg_has_role(session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_has_role(session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
       OR pg_has_role(session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
       OR p_organizacion IS NULL OR p_expediente IS NULL OR p_llamamiento IS NULL
       OR p_organizacion !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_expediente !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_llamamiento !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
        RAISE EXCEPTION 'antecedente de aviso no disponible' USING ERRCODE='42501';
    END IF;
    BEGIN
        SELECT candidato.* INTO STRICT v_seleccion FROM (
            SELECT e.* FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
             WHERE e.situacion='confirmada'
               AND e.solicitud_json->>'organizacion_ref'=p_organizacion
               AND e.solicitud_json->>'expediente_ref'=p_expediente
               AND e.recibo_json->>'llamamiento_ref'=p_llamamiento
               AND e.recibo_json->'propuesta_generada'='true'::jsonb
            UNION ALL
            -- Mismo vínculo nominal de CT62: continuación CT60 de una renuncia.
            SELECT e.* FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
              JOIN vec_contratacion_temporal.resolucion_manual_respuesta_rrhh r
                ON r.seleccion_clave=e.clave_idempotencia
              JOIN vec_contratacion_temporal.comunicacion_llamamiento_local c
                ON c.comunicacion_ref=r.comunicacion_ref AND c.seleccion_clave=e.clave_idempotencia
               AND c.organizacion_ref=r.organizacion_ref AND c.expediente_ref=r.expediente_ref
               AND c.llamamiento_ref=r.llamamiento_ref
             WHERE r.organizacion_ref=p_organizacion AND r.expediente_ref=p_expediente
               AND r.estado='confirmado' AND r.solicitud_json->>'Respuesta'='renuncia'
               AND r.continuacion_clave IS NOT NULL
               AND r.continuacion_recibo->>'Estado'='confirmado'
               AND r.continuacion_recibo->'Solicitud'->>'OrganizacionRef'=r.organizacion_ref
               AND r.continuacion_recibo->'Solicitud'->>'ExpedienteRef'=r.expediente_ref
               AND r.continuacion_recibo->'Solicitud'->>'ResolucionRef'=r.resolucion_ref
               AND r.continuacion_recibo->'Solicitud'->>'ClaveIdempotencia'=r.continuacion_clave::text
               AND r.continuacion_recibo->>'LlamamientoAnteriorRef'=r.llamamiento_ref
               AND r.continuacion_recibo->'ReciboBolsa'->>'LlamamientoRef'=p_llamamiento
               AND r.llamamiento_ref<>p_llamamiento
               AND r.continuacion_material_sha256=encode(sha256(convert_to(r.continuacion_material,'UTF8')),'hex')
               AND (r.continuacion_material::jsonb)->>'Etapa'='confirmacion'
               AND (r.continuacion_material::jsonb)->'Solicitud'=r.continuacion_recibo->'Solicitud'
               AND (r.continuacion_material::jsonb)->'ReciboBolsa'=r.continuacion_recibo->'ReciboBolsa'
               AND e.situacion='confirmada' AND e.recibo_json->'propuesta_generada'='true'::jsonb
               AND e.solicitud_json->>'organizacion_ref'=r.organizacion_ref
               AND e.solicitud_json->>'expediente_ref'=r.expediente_ref
               AND e.recibo_json->>'llamamiento_ref'=r.llamamiento_ref
        ) candidato;
    EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception THEN
        RAISE EXCEPTION 'antecedente de aviso no disponible' USING ERRCODE='42501';
    END;
    -- Validar el canon persistido y el enlace solicitud/comando/recibo antes de N.
    v_artefacto := v_seleccion.artefacto_canonico::jsonb;
    v_datos := v_artefacto#>'{comando,contexto,datos}';
    IF v_seleccion.recibo_json IS NULL OR v_seleccion.artefacto_canonico IS NULL
       OR v_seleccion.solicitud_json->>'clave_idempotencia' IS DISTINCT FROM v_seleccion.clave_idempotencia::text
       OR v_seleccion.solicitud_json->>'huella_semantica' IS DISTINCT FROM v_seleccion.huella_semantica
       OR vec_contratacion_temporal.huella_solicitud_seleccion_llamamiento_o6_v1(v_seleccion.solicitud_json)
          IS DISTINCT FROM v_seleccion.huella_semantica
       OR vec_contratacion_temporal.confirmacion_canonica_seleccion_llamamiento_o6_v1(
          v_artefacto, v_seleccion.artefacto_canonico, v_seleccion.recibo_json,
          vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(v_seleccion.recibo_json)) IS NOT TRUE
       OR v_artefacto->>'tipo' IS DISTINCT FROM 'recibo_llamamiento'
       OR v_datos->>'organizacion_ref' IS DISTINCT FROM p_organizacion
       OR v_datos->>'expediente_ref' IS DISTINCT FROM p_expediente
       OR v_seleccion.recibo_json->>'organizacion_ref' IS DISTINCT FROM p_organizacion
       OR v_seleccion.recibo_json->>'expediente_ref' IS DISTINCT FROM p_expediente
       OR v_datos->'version_expediente' IS DISTINCT FROM v_seleccion.solicitud_json->'version_expediente'
       OR v_seleccion.recibo_json->'version_expediente' IS DISTINCT FROM v_datos->'version_expediente'
       OR v_seleccion.recibo_json->'operacion_ref' IS DISTINCT FROM v_datos->'operacion_ref'
       OR v_datos->'correlacion_ref' IS DISTINCT FROM v_seleccion.solicitud_json->'correlacion_ref'
       OR v_seleccion.recibo_json->'correlacion_ref' IS DISTINCT FROM v_datos->'correlacion_ref'
       OR jsonb_typeof(v_seleccion.solicitud_json->'version_expediente') IS DISTINCT FROM 'number'
       OR (v_seleccion.solicitud_json->>'version_expediente') !~ '^[1-9][0-9]{0,15}$' THEN
        RAISE EXCEPTION 'antecedente de aviso no disponible' USING ERRCODE='42501';
    END IF;
    v_version := (v_seleccion.solicitud_json->>'version_expediente')::bigint;
    IF v_version NOT BETWEEN 6 AND 9007199254740991 THEN
        RAISE EXCEPTION 'antecedente de aviso no disponible' USING ERRCODE='42501';
    END IF;
    IF v_version=6 THEN
        RETURN QUERY SELECT * FROM vec_contratacion_temporal.leer_expediente_seleccion_v1(
            p_organizacion,p_expediente,v_version);
    ELSE
        RETURN QUERY SELECT * FROM vec_contratacion_temporal.leer_expediente_seleccion_v2(
            p_organizacion,p_expediente,v_version);
    END IF;
EXCEPTION WHEN data_exception THEN
    RAISE EXCEPTION 'antecedente de aviso no disponible' USING ERRCODE='42501';
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.leer_expediente_aviso_confirmado_v1(
    text,text,text) FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.leer_expediente_aviso_confirmado_v1(
    text,text,text) TO vec_contratacion_temporal_ejecutor;
COMMIT;
