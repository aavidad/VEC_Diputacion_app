\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

-- Adición nominal: CT53/CT55 y sus funciones v1 permanecen intactas.
DO $dependencias$
BEGIN
    IF to_regprocedure('vec_contratacion_temporal.leer_expediente_seleccion_v1(text,text,bigint)') IS NULL
       OR to_regprocedure('vec_contratacion_temporal.reanudar_preparacion_orden_seleccion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
        RAISE EXCEPTION 'faltan lectores y reanudación originales' USING ERRCODE = '55000';
    END IF;
END
$dependencias$;

CREATE FUNCTION vec_contratacion_temporal.leer_expediente_seleccion_v2(
    p_organizacion text, p_expediente text, p_version bigint
) RETURNS TABLE(expediente_json jsonb, version_actual bigint)
LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '5s'
AS $funcion$
DECLARE
    v_expediente jsonb;
    v_actual bigint;
BEGIN
    IF NOT pg_catalog.pg_has_role(session_user,
            'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR p_organizacion IS NULL OR p_expediente IS NULL
       OR p_version IS NULL OR p_version < 7 OR p_version > 9007199254740991
       OR p_organizacion !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_expediente !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'preparacion de seleccion denegada';
    END IF;
    SELECT v.agregado_json, a.version::bigint INTO v_expediente, v_actual
      FROM vec_contratacion_temporal.expediente_version_integral v
      JOIN vec_contratacion_temporal.expediente_integral_actual a
        USING (expediente_ref)
     WHERE v.expediente_ref = p_expediente AND v.version = p_version
       AND v.agregado_json->>'organizacion_ref' = p_organizacion
       AND v.agregado_json->>'fase_actual' = 'fiscalizacion'
       AND v.agregado_json->>'estado_actual' = 'en_curso'
       AND v.agregado_json->'fiscalizacion'->>'resultado' IN
           ('favorable', 'favorable_con_observaciones');
    IF NOT FOUND OR v_actual < p_version THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'preparacion de seleccion denegada';
    END IF;
    RETURN QUERY SELECT v_expediente, v_actual;
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.leer_expediente_seleccion_v2(
    text,text,bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.leer_expediente_seleccion_v2(
    text,text,bigint) TO vec_contratacion_temporal_ejecutor;

CREATE FUNCTION vec_contratacion_temporal.reanudar_preparacion_orden_seleccion_v2(
    p_solicitud_texto text,
    p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
    p_persona_version numeric, p_perfil_version numeric,
    p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea
) RETURNS TABLE (
    situacion text, solicitud_json text, reserva_ref text,
    efecto text, recibo_json text, artefacto_json text
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '15s'
SET idle_in_transaction_session_timeout = '20s'
AS $funcion$
DECLARE
    s jsonb; d jsonb; v_material text; v_material_huella text; v_contexto_huella text;
    v_ejecucion vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
    v_consumo record;
    v_ahora timestamptz(6); v_reserva text; v_evento text;
    v_version bigint; v_actual bigint; v_agregado jsonb;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario' OR session_user = current_user
       OR NOT pg_has_role(session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_has_role(session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
       OR pg_has_role(session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
       OR current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION 'reanudación de selección denegada' USING ERRCODE = '42501';
    END IF;
    IF p_solicitud_texto IS NULL OR octet_length(p_solicitud_texto) NOT BETWEEN 1 AND 1048576
       OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
        RAISE EXCEPTION 'solicitud de reanudación inválida' USING ERRCODE = '22023';
    END IF;
    s := vec_contratacion_temporal.solicitud_desde_texto_seleccion_llamamiento_o6_v1(p_solicitud_texto);
    IF s IS NULL OR jsonb_typeof(s->'version_expediente') IS DISTINCT FROM 'number'
       OR (s->>'version_expediente')::numeric NOT BETWEEN 7 AND 9007199254740991
       OR trunc((s->>'version_expediente')::numeric) <> (s->>'version_expediente')::numeric
       OR vec_contratacion_temporal.huella_solicitud_seleccion_llamamiento_o6_v1(s)
          IS DISTINCT FROM s->>'huella_semantica' THEN
        RAISE EXCEPTION 'solicitud de reanudación inválida' USING ERRCODE = '22023';
    END IF;
    v_version := (s->>'version_expediente')::bigint;
    -- Mismo canon de cinco campos que NuevoRecursoReanudacionSeleccionLlamamiento.
    v_material := '{"organizacion_ref":' || (s->'organizacion_ref')::text ||
        ',"expediente_ref":' || (s->'expediente_ref')::text ||
        ',"version_expediente":' || v_version::text || ',"clave_idempotencia":' || (s->'clave_idempotencia')::text ||
        ',"huella_semantica":' || (s->'huella_semantica')::text || '}';
    v_material_huella := encode(sha256(convert_to(v_material, 'UTF8')), 'hex');
    v_contexto_huella := encode(sha256(convert_to(
        '{"ambitos":{"organizacion_ref":' || (s->'organizacion_ref')::text ||
        '},"atributos":{"material_sha256":"' || v_material_huella || '"}}', 'UTF8')), 'hex');
    d := convert_from(p_decision, 'UTF8')::jsonb;
    IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.llamamiento.reanudar_orden'
       OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'reanudacion_seleccion_contratacion_temporal'
       OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
       OR d->>'recurso_ref' IS DISTINCT FROM s->>'expediente_ref'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'autorización de reanudación divergente' USING ERRCODE = '42501';
    END IF;
    -- La reanudación de N conserva la cabeza favorable N, además de la intención.
    SELECT a.version, v.agregado_json INTO v_actual, v_agregado
      FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v
        ON v.expediente_ref = a.expediente_ref AND v.version = a.version
     WHERE a.expediente_ref = s->>'expediente_ref'
       AND v.agregado_json->>'organizacion_ref' = s->>'organizacion_ref'
     FOR UPDATE OF a;
    IF NOT FOUND OR v_actual IS DISTINCT FROM v_version
       OR v_agregado->>'fase_actual' IS DISTINCT FROM 'fiscalizacion'
       OR v_agregado->>'estado_actual' IS DISTINCT FROM 'en_curso'
       OR coalesce(v_agregado->'fiscalizacion'->>'resultado', '') NOT IN
          ('favorable', 'favorable_con_observaciones') THEN
        RAISE EXCEPTION 'cabeza de reanudación divergente' USING ERRCODE = '42501';
    END IF;
    -- El bloqueo y la comprobación del estado preceden al consumo fresco.
    SELECT e.* INTO v_ejecucion
      FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
     WHERE e.clave_idempotencia = (s->>'clave_idempotencia')::uuid FOR UPDATE;
    IF NOT FOUND OR v_ejecucion.solicitud_json IS DISTINCT FROM s
       OR v_ejecucion.huella_semantica IS DISTINCT FROM s->>'huella_semantica' THEN
        RAISE EXCEPTION 'intención de reanudación divergente' USING ERRCODE = '42501';
    END IF;
    v_ahora := date_trunc('microseconds', clock_timestamp());
    IF v_ejecucion.situacion IS DISTINCT FROM 'indeterminada'
       OR v_ejecucion.efecto IS DISTINCT FROM 'preparar_orden'
       OR v_ejecucion.ventana_orden_abierta IS NOT TRUE
       OR v_ejecucion.ventana_llamamiento_abierta IS NOT FALSE
       OR v_ejecucion.recibo_json IS NOT NULL OR v_ejecucion.artefacto_canonico IS NOT NULL
       OR v_ejecucion.lease_hasta > v_ahora
       OR v_ejecucion.fencing_version >= 9007199254740991 THEN
        RAISE EXCEPTION 'reanudación no disponible para este estado' USING ERRCODE = '55000';
    END IF;
    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3.registrar_y_consumir_reanudacion_seleccion_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.consumo_nuevo IS NOT TRUE
       OR v_consumo.efecto_ref IS DISTINCT FROM s->>'expediente_ref'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'reanudación requiere autorización nueva ligada' USING ERRCODE = '42501';
    END IF;
    v_ahora := date_trunc('microseconds', clock_timestamp());
    v_reserva := vec_contratacion_temporal.nuevo_token_fencing_seleccion_llamamiento_o6_v2();
    IF v_reserva IS NOT DISTINCT FROM v_ejecucion.reserva_ref THEN
        RAISE EXCEPTION 'reserva de reanudación no renovada' USING ERRCODE = '55000';
    END IF;
    -- No se alteran solicitud, UUID, huella, efecto, ventanas ni recibos.
    UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
       SET situacion = 'propietaria', reserva_ref = v_reserva,
           fencing_version = v_ejecucion.fencing_version + 1,
           lease_hasta = v_ahora + interval '30 seconds', actualizada_en = v_ahora
     WHERE e.clave_idempotencia = v_ejecucion.clave_idempotencia;
    INSERT INTO vec_contratacion_temporal.historia_reanudacion_seleccion_llamamiento VALUES (
        v_consumo.auditoria_ref, v_ejecucion.clave_idempotencia, v_ejecucion.huella_semantica,
        v_ejecucion.fencing_version, v_ejecucion.fencing_version + 1,
        encode(sha256(convert_to(v_ejecucion.reserva_ref,'UTF8')),'hex'),
        encode(sha256(convert_to(v_reserva,'UTF8')),'hex'),
        v_consumo.decision_ref, v_consumo.consumo_huella_sha256, v_ahora, v_ahora + interval '30 seconds');
    v_evento := 'evento:ct:reanudacion:' || gen_random_uuid()::text;
    INSERT INTO vec_contratacion_temporal.outbox_reanudacion_seleccion_llamamiento VALUES (
        v_evento, v_consumo.auditoria_ref, 'seleccion.preparacion_orden.reanudada',
        jsonb_build_object('organizacion_ref',s->>'organizacion_ref',
            'expediente_ref',s->>'expediente_ref','clave_idempotencia',v_ejecucion.clave_idempotencia,
            'huella_semantica',v_ejecucion.huella_semantica,
            'fencing_version',v_ejecucion.fencing_version + 1), v_ahora);
    RETURN QUERY SELECT 'propietaria', v_ejecucion.solicitud_json::text,
        v_reserva, 'preparar_orden', '', '';
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.reanudar_preparacion_orden_seleccion_v2(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.reanudar_preparacion_orden_seleccion_v2(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_ejecutor;

COMMIT;
