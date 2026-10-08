\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:orq1:ct198',0));
-- Definiciones literales instaladas en el clon postHX+HZ14+B85/B86/CT193/B87/CT194.
DO $pre$
DECLARE v_actual text; v_check text;
BEGIN
 SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') INTO v_actual FROM pg_proc
 WHERE oid='vec_contratacion_temporal.resolver_terminal_autorizado_seleccion_llamamiento_o6_v2(uuid,text)'::regprocedure
 AND proowner='vec_contratacion_temporal_propietario'::regrole AND prosecdef;
 IF v_actual IS DISTINCT FROM '5b08176851e50091f16032c122a6dff0d8c01bbfe500eea29e2d7938f23f9def' THEN
  RAISE EXCEPTION 'CT198: PARO clave=resolver.prosrc actual=% esperado=5b08176851e50091f16032c122a6dff0d8c01bbfe500eea29e2d7938f23f9def',coalesce(v_actual,'ausente') USING ERRCODE='55000';
 END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_reanudacion_solicitud_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_contratacion_temporal.reanudar_solicitud_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
  RAISE EXCEPTION 'CT198: PARO clave=dependencias actual=divergente esperado=AD225_sin_CT198' USING ERRCODE='55000';
 END IF;
 SELECT pg_get_constraintdef(c.oid) INTO v_check FROM pg_constraint c
 WHERE c.conrelid='vec_contratacion_temporal.outbox_reanudacion_seleccion_llamamiento'::regclass
 AND c.conname='outbox_reanudacion_seleccion_llamamiento_tipo_check';
 IF v_check IS DISTINCT FROM 'CHECK ((tipo = ''seleccion.preparacion_orden.reanudada''::text))' THEN
  RAISE EXCEPTION 'CT198: PARO clave=outbox_tipo actual=% esperado=solo_preparacion_orden',coalesce(v_check,'ausente') USING ERRCODE='55000';
 END IF;
END
$pre$;
-- El outbox existente conserva su fila histórica; se admite un tipo nuevo exacto.
ALTER TABLE vec_contratacion_temporal.outbox_reanudacion_seleccion_llamamiento
 DROP CONSTRAINT outbox_reanudacion_seleccion_llamamiento_tipo_check;
ALTER TABLE vec_contratacion_temporal.outbox_reanudacion_seleccion_llamamiento
 ADD CONSTRAINT outbox_reanudacion_seleccion_llamamiento_tipo_check CHECK
 (tipo IN ('seleccion.preparacion_orden.reanudada','seleccion.solicitud_llamamiento.reanudada'));
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.resolver_terminal_autorizado_seleccion_llamamiento_o6_v2(p_clave uuid, p_consulta_texto text)
 RETURNS TABLE(situacion text, solicitud_json text, reserva_ref text, efecto text, recibo_json text, artefacto_json text)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO pg_catalog, pg_temp
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
 SET idle_in_transaction_session_timeout TO '20s'
AS $function$
DECLARE
    v_consulta jsonb;
    v_canon text;
    v_ejecucion vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
BEGIN
    IF NOT pg_catalog.pg_has_role(
        session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
    ) OR p_clave IS NULL OR p_consulta_texto IS NULL
       OR pg_catalog.octet_length(p_consulta_texto) > 65536 THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'operacion O6 denegada';
    END IF;
    BEGIN
        v_consulta := p_consulta_texto::jsonb;
    EXCEPTION WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'consulta terminal O6 invalida';
    END;
    v_canon := '{"organizacion_ref":' || (v_consulta->'organizacion_ref')::text ||
        ',"expediente_ref":' || (v_consulta->'expediente_ref')::text ||
        ',"version_expediente":' ||
            vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(v_consulta->'version_expediente') ||
        ',"correlacion_ref":' || (v_consulta->'correlacion_ref')::text ||
        ',"autoridad_solicitante":' || (v_consulta->'autoridad_solicitante')::text ||
        ',"autorizacion":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_consulta->'autorizacion') ||
        ',"accion":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_consulta->'accion') ||
        ',"recurso":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_consulta->'recurso') ||
        ',"finalidad":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_consulta->'finalidad') || '}';
    IF v_canon IS DISTINCT FROM p_consulta_texto
       OR v_consulta->>'version_expediente' !~ '^[1-9][0-9]*$'
       OR (v_consulta->>'version_expediente')::numeric > 9007199254740991
       OR EXISTS (SELECT 1 FROM (VALUES
            (v_consulta->'organizacion_ref'), (v_consulta->'expediente_ref'),
            (v_consulta->'correlacion_ref'), (v_consulta->'autoridad_solicitante')
       ) referencias(valor) WHERE pg_catalog.jsonb_typeof(valor) IS DISTINCT FROM 'string'
          OR valor #>> '{}' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR EXISTS (SELECT 1 FROM (VALUES
            (v_consulta->'autorizacion'), (v_consulta->'accion'),
            (v_consulta->'recurso'), (v_consulta->'finalidad')
       ) referencias(valor) WHERE pg_catalog.jsonb_typeof(valor) IS DISTINCT FROM 'object'
          OR NOT (valor ?& ARRAY['referencia','version','huella_sha256'])
          OR valor - ARRAY['referencia','version','huella_sha256'] <> '{}'::jsonb
          OR pg_catalog.jsonb_typeof(valor->'referencia') IS DISTINCT FROM 'string'
          OR pg_catalog.jsonb_typeof(valor->'version') IS DISTINCT FROM 'number'
          OR pg_catalog.jsonb_typeof(valor->'huella_sha256') IS DISTINCT FROM 'string'
          OR valor->>'referencia' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
          OR valor->>'version' !~ '^[1-9][0-9]*$'
          OR (valor->>'version')::numeric > 9007199254740991
          OR valor->>'huella_sha256' !~ '^[0-9a-f]{64}$'
          OR valor->>'huella_sha256' = pg_catalog.repeat('0', 64)) THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'consulta terminal O6 invalida';
    END IF;
    SELECT ejecucion.* INTO v_ejecucion
      FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 ejecucion
     WHERE ejecucion.clave_idempotencia = p_clave;
    IF NOT FOUND THEN
        RETURN QUERY SELECT '', '', '', '', '', '';
    ELSIF v_consulta->>'organizacion_ref' IS DISTINCT FROM v_ejecucion.solicitud_json->>'organizacion_ref'
       OR v_consulta->>'expediente_ref' IS DISTINCT FROM v_ejecucion.solicitud_json->>'expediente_ref'
       OR v_consulta->'version_expediente' IS DISTINCT FROM v_ejecucion.solicitud_json->'version_expediente'
       OR v_consulta->>'correlacion_ref' IS DISTINCT FROM v_ejecucion.solicitud_json->>'correlacion_ref'
       OR v_consulta->>'autoridad_solicitante' IS DISTINCT FROM v_ejecucion.solicitud_json->>'autoridad_solicitante'
       OR v_consulta->'autorizacion' IS DISTINCT FROM v_ejecucion.solicitud_json->'autorizacion_consulta'
       OR v_consulta->'accion' IS DISTINCT FROM v_ejecucion.solicitud_json->'accion_consulta'
       OR v_consulta->'recurso' IS DISTINCT FROM v_ejecucion.solicitud_json->'recurso_consulta'
       OR v_consulta->'finalidad' IS DISTINCT FROM v_ejecucion.solicitud_json->'finalidad' THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'replay O6 denegado';
    ELSIF v_ejecucion.recibo_json IS NULL AND v_ejecucion.artefacto_canonico IS NULL
       AND v_ejecucion.ventana_orden_abierta IS TRUE
       AND (
         (v_ejecucion.efecto = 'preparar_orden' AND v_ejecucion.ventana_llamamiento_abierta IS FALSE) OR
         (v_ejecucion.efecto = 'solicitar_llamamiento' AND v_ejecucion.ventana_llamamiento_abierta IS TRUE)
       ) AND (
         v_ejecucion.situacion = 'indeterminada' OR
         (v_ejecucion.situacion = 'propietaria' AND v_ejecucion.lease_hasta <= pg_catalog.clock_timestamp())
       ) THEN
        RETURN QUERY SELECT '', '', '', '', '', '';
    ELSIF v_ejecucion.situacion <> 'confirmada' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'terminal O6 no confirmado';
    ELSE
        RETURN QUERY SELECT v_ejecucion.situacion, v_ejecucion.solicitud_json::text,
            '', '', v_ejecucion.recibo_json::text, v_ejecucion.artefacto_canonico;
    END IF;
END
$function$;
CREATE FUNCTION vec_contratacion_temporal.reanudar_solicitud_llamamiento_v1(p_solicitud_texto text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS TABLE(situacion text, solicitud_json text, reserva_ref text, efecto text, recibo_json text, artefacto_json text)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO pg_catalog, pg_temp
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
 SET idle_in_transaction_session_timeout TO '20s'
AS $function$
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
       OR (s->>'version_expediente')::numeric NOT BETWEEN 6 AND 9007199254740991
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
    IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.llamamiento.reanudar_solicitud'
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
       OR v_ejecucion.efecto IS DISTINCT FROM 'solicitar_llamamiento'
       OR v_ejecucion.ventana_orden_abierta IS NOT TRUE
       OR v_ejecucion.ventana_llamamiento_abierta IS NOT TRUE
       OR v_ejecucion.recibo_json IS NOT NULL OR v_ejecucion.artefacto_canonico IS NOT NULL
       OR v_ejecucion.lease_hasta > v_ahora
       OR v_ejecucion.fencing_version >= 9007199254740991 THEN
        RAISE EXCEPTION 'reanudación no disponible para este estado' USING ERRCODE = '55000';
    END IF;
    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3.registrar_y_consumir_reanudacion_solicitud_v3_atestada(
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
    INSERT INTO vec_contratacion_temporal.historia_reanudacion_seleccion_llamamiento (
        auditoria_ref,clave_idempotencia,huella_semantica,fencing_anterior,fencing_nuevo,
        reserva_anterior_sha256,reserva_nueva_sha256,decision_ref,consumo_huella_sha256,
        reanudada_en,lease_hasta) VALUES (
        v_consumo.auditoria_ref, v_ejecucion.clave_idempotencia, v_ejecucion.huella_semantica,
        v_ejecucion.fencing_version, v_ejecucion.fencing_version + 1,
        encode(sha256(convert_to(v_ejecucion.reserva_ref,'UTF8')),'hex'),
        encode(sha256(convert_to(v_reserva,'UTF8')),'hex'),
        v_consumo.decision_ref, v_consumo.consumo_huella_sha256, v_ahora, v_ahora + interval '30 seconds');
    v_evento := 'evento:ct:reanudacion:' || gen_random_uuid()::text;
    INSERT INTO vec_contratacion_temporal.outbox_reanudacion_seleccion_llamamiento (
        evento_ref,auditoria_ref,tipo,carga_json,creada_en) VALUES (
        v_evento, v_consumo.auditoria_ref, 'seleccion.solicitud_llamamiento.reanudada',
        jsonb_build_object('organizacion_ref',s->>'organizacion_ref',
            'expediente_ref',s->>'expediente_ref','clave_idempotencia',v_ejecucion.clave_idempotencia,
            'huella_semantica',v_ejecucion.huella_semantica,
            'fencing_version',v_ejecucion.fencing_version + 1), v_ahora);
    RETURN QUERY SELECT 'propietaria', v_ejecucion.solicitud_json::text,
        v_reserva, 'solicitar_llamamiento', '', '';
END
$function$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.reanudar_solicitud_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_contratacion_temporal_migrador;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.reanudar_solicitud_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;

-- Los cánones instalados conservan byte a byte las respuestas antiguas.
DO $canon_pre$
DECLARE v_actual text;
BEGIN
 SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') INTO v_actual FROM pg_proc WHERE oid='vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(jsonb)'::regprocedure;
 IF v_actual IS DISTINCT FROM 'bd6701ac20a083bb69833b9614f0d90fe6df6ec2c97eb2a3e458764cb86b4a89' THEN
  RAISE EXCEPTION 'CT198: PARO clave=recibo_json.prosrc actual=% esperado=bd6701ac20a083bb69833b9614f0d90fe6df6ec2c97eb2a3e458764cb86b4a89',coalesce(v_actual,'ausente') USING ERRCODE='55000';
 END IF;
 SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') INTO v_actual FROM pg_proc WHERE oid='vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(jsonb,boolean)'::regprocedure;
 IF v_actual IS DISTINCT FROM '72f60cf9d47f3475d6b39749d830d2c136c95afba69fc67c2ab0e4a87e964056' THEN
  RAISE EXCEPTION 'CT198: PARO clave=artefacto_json.prosrc actual=% esperado=72f60cf9d47f3475d6b39749d830d2c136c95afba69fc67c2ab0e4a87e964056',coalesce(v_actual,'ausente') USING ERRCODE='55000';
 END IF;
 SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') INTO v_actual FROM pg_proc WHERE oid='vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(uuid,text,text,text,text,text)'::regprocedure;
 IF v_actual IS DISTINCT FROM '35fa1c7d1e787edcf0ba1fa9f9c56b83c1f24f8592d98fb16abcb10e8743b790' THEN
  RAISE EXCEPTION 'CT198: PARO clave=confirmar.prosrc actual=% esperado=35fa1c7d1e787edcf0ba1fa9f9c56b83c1f24f8592d98fb16abcb10e8743b790',coalesce(v_actual,'ausente') USING ERRCODE='55000';
 END IF;
END
$canon_pre$;
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(p_recibo jsonb)
 RETURNS text
 LANGUAGE plpgsql
 IMMUTABLE PARALLEL SAFE STRICT
 SET search_path TO 'pg_catalog'
AS $function$
DECLARE
    v_procedencia jsonb := p_recibo->'procedencia';
    v_evidencia jsonb := v_procedencia->'evidencia';
    v_procedencia_texto text;
BEGIN
    IF p_recibo ? 'llamamiento_recuperado' AND p_recibo->'llamamiento_recuperado' IS DISTINCT FROM 'true'::jsonb THEN
        RETURN NULL;
    END IF;
    v_procedencia_texto := '{"autoridad_ref":' || (v_procedencia->'autoridad_ref')::text ||
        ',"respuesta_ref":' || (v_procedencia->'respuesta_ref')::text ||
        ',"contrato_version":' || vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(v_procedencia->'contrato_version') ||
        ',"fuente":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_procedencia->'fuente') ||
        ',"evidencia":{"evidencia_ref":' || (v_evidencia->'evidencia_ref')::text ||
        ',"clave_verificacion_ref":' || (v_evidencia->'clave_verificacion_ref')::text ||
        ',"sello_hmac":' || (v_evidencia->'sello_hmac')::text ||
        ',"emitida_en":' || (v_evidencia->'emitida_en')::text ||
        ',"valida_hasta":' || (v_evidencia->'valida_hasta')::text ||
        ',"retener_hasta":' || (v_evidencia->'retener_hasta')::text || '}}';
    RETURN '{"operacion_ref":' || (p_recibo->'operacion_ref')::text ||
        ',"organizacion_ref":' || (p_recibo->'organizacion_ref')::text ||
        ',"expediente_ref":' || (p_recibo->'expediente_ref')::text ||
        ',"version_expediente":' || vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(p_recibo->'version_expediente') ||
        ',"correlacion_ref":' || (p_recibo->'correlacion_ref')::text ||
        ',"necesidad":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(p_recibo->'necesidad') ||
        ',"bolsa":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(p_recibo->'bolsa') ||
        ',"orden":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(p_recibo->'orden') ||
        ',"politica":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(p_recibo->'politica') ||
        ',"resultado":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(p_recibo->'resultado') ||
        ',"propuesta_generada":' || (p_recibo->'propuesta_generada')::text ||
        ',"propuesta":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(p_recibo->'propuesta') ||
        ',"accion_evento":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(p_recibo->'accion_evento') ||
        ',"llamamiento_ref":' || (p_recibo->'llamamiento_ref')::text ||
        ',"seleccion_ref":' || (p_recibo->'seleccion_ref')::text ||
        ',"retencion_seleccion":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(p_recibo->'retencion_seleccion') ||
        ',"orden_seleccionado":' || vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(p_recibo->'orden_seleccionado') ||
        ',"recibo_ref":' || (p_recibo->'recibo_ref')::text ||
        ',"auditoria_ref":' || (p_recibo->'auditoria_ref')::text ||
        ',"evento_ref":' || (p_recibo->'evento_ref')::text ||
        ',"confirmada_en":' || (p_recibo->'confirmada_en')::text ||
        ',"procedencia":' || v_procedencia_texto ||
        CASE WHEN p_recibo->'llamamiento_recuperado' = 'true'::jsonb
             THEN ',"llamamiento_recuperado":true' ELSE '' END || '}';
END
$function$;
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(p_artefacto jsonb, p_vaciar_huella boolean)
 RETURNS text
 LANGUAGE plpgsql
 IMMUTABLE PARALLEL SAFE STRICT
 SET search_path TO 'pg_catalog'
AS $function$
DECLARE
    v_comando jsonb := p_artefacto->'comando';
    v_contexto jsonb := v_comando->'contexto';
    v_datos jsonb := v_contexto->'datos';
    v_recibo jsonb := p_artefacto->'recibo';
    v_procedencia jsonb := v_recibo->'procedencia';
    v_evidencia_nominal jsonb := v_procedencia->'evidencia';
    v_evidencia jsonb := p_artefacto->'evidencia';
    v_datos_texto text;
    v_contexto_texto text;
    v_comando_texto text;
    v_procedencia_texto text;
    v_recibo_texto text;
    v_evidencia_texto text;
BEGIN
    IF v_recibo ? 'llamamiento_recuperado' AND v_recibo->'llamamiento_recuperado' IS DISTINCT FROM 'true'::jsonb THEN
        RETURN NULL;
    END IF;
    v_datos_texto := '{"operacion_ref":' || (v_datos->'operacion_ref')::text ||
        ',"organizacion_ref":' || (v_datos->'organizacion_ref')::text ||
        ',"expediente_ref":' || (v_datos->'expediente_ref')::text ||
        ',"version_expediente":' || vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(v_datos->'version_expediente') ||
        ',"correlacion_ref":' || (v_datos->'correlacion_ref')::text ||
        ',"contrato_version":' || vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(v_datos->'contrato_version') ||
        ',"autoridad_solicitante":' || (v_datos->'autoridad_solicitante')::text ||
        ',"autorizacion":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_datos->'autorizacion') ||
        ',"accion":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_datos->'accion') ||
        ',"recurso":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_datos->'recurso') ||
        ',"finalidad":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_datos->'finalidad') ||
        ',"solicitada_en":' || (v_datos->'solicitada_en')::text ||
        ',"valida_hasta":' || (v_datos->'valida_hasta')::text || '}';
    v_contexto_texto := '{"datos":' || v_datos_texto ||
        ',"clave_verificacion_ref":' || (v_contexto->'clave_verificacion_ref')::text ||
        ',"sello_hmac":' || (v_contexto->'sello_hmac')::text || '}';
    v_comando_texto := '{"contexto":' || v_contexto_texto ||
        ',"necesidad":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_comando->'necesidad') ||
        ',"bolsa":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_comando->'bolsa') ||
        ',"orden":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_comando->'orden') ||
        ',"politica":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_comando->'politica') ||
        ',"total_posiciones_orden":' || vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(v_comando->'total_posiciones_orden') ||
        ',"maxima_posicion_evaluable":' || vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(v_comando->'maxima_posicion_evaluable') ||
        ',"huella_recibo_orden":' || (v_comando->'huella_recibo_orden')::text || '}';

    v_procedencia_texto := '{"autoridad_ref":' || (v_procedencia->'autoridad_ref')::text ||
        ',"respuesta_ref":' || (v_procedencia->'respuesta_ref')::text ||
        ',"contrato_version":' || vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(v_procedencia->'contrato_version') ||
        ',"fuente":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_procedencia->'fuente') ||
        ',"evidencia":{"evidencia_ref":' || (v_evidencia_nominal->'evidencia_ref')::text ||
        ',"clave_verificacion_ref":' || (v_evidencia_nominal->'clave_verificacion_ref')::text ||
        ',"sello_hmac":' || (v_evidencia_nominal->'sello_hmac')::text ||
        ',"emitida_en":' || (v_evidencia_nominal->'emitida_en')::text ||
        ',"valida_hasta":' || (v_evidencia_nominal->'valida_hasta')::text ||
        ',"retener_hasta":' || (v_evidencia_nominal->'retener_hasta')::text || '}}';
    v_recibo_texto := '{"operacion_ref":' || (v_recibo->'operacion_ref')::text ||
        ',"organizacion_ref":' || (v_recibo->'organizacion_ref')::text ||
        ',"expediente_ref":' || (v_recibo->'expediente_ref')::text ||
        ',"version_expediente":' || vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(v_recibo->'version_expediente') ||
        ',"correlacion_ref":' || (v_recibo->'correlacion_ref')::text ||
        ',"necesidad":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_recibo->'necesidad') ||
        ',"bolsa":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_recibo->'bolsa') ||
        ',"orden":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_recibo->'orden') ||
        ',"politica":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_recibo->'politica') ||
        ',"resultado":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_recibo->'resultado') ||
        ',"propuesta_generada":' || (v_recibo->'propuesta_generada')::text ||
        ',"propuesta":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_recibo->'propuesta') ||
        ',"accion_evento":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_recibo->'accion_evento') ||
        ',"llamamiento_ref":' || (v_recibo->'llamamiento_ref')::text ||
        ',"seleccion_ref":' || (v_recibo->'seleccion_ref')::text ||
        ',"retencion_seleccion":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_recibo->'retencion_seleccion') ||
        ',"orden_seleccionado":' || vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(v_recibo->'orden_seleccionado') ||
        ',"recibo_ref":' || (v_recibo->'recibo_ref')::text ||
        ',"auditoria_ref":' || (v_recibo->'auditoria_ref')::text ||
        ',"evento_ref":' || (v_recibo->'evento_ref')::text ||
        ',"confirmada_en":' || (v_recibo->'confirmada_en')::text ||
        ',"procedencia":' || v_procedencia_texto ||
        CASE WHEN v_recibo->'llamamiento_recuperado' = 'true'::jsonb
             THEN ',"llamamiento_recuperado":true' ELSE '' END || '}';

    v_evidencia_texto := '{"esquema":' || (v_evidencia->'esquema')::text ||
        ',"tipo_material":' || (v_evidencia->'tipo_material')::text ||
        ',"autoridad_ref":' || (v_evidencia->'autoridad_ref')::text ||
        ',"clave_verificacion_ref":' || (v_evidencia->'clave_verificacion_ref')::text ||
        ',"evidencia_ref":' || (v_evidencia->'evidencia_ref')::text ||
        ',"peticion_ref":' || (v_evidencia->'peticion_ref')::text ||
        ',"huella_peticion_sha256":' || (v_evidencia->'huella_peticion_sha256')::text ||
        ',"respuesta_ref":' || (v_evidencia->'respuesta_ref')::text ||
        ',"huella_respuesta_sha256":' || (v_evidencia->'huella_respuesta_sha256')::text ||
        ',"sello_hmac":' || (v_evidencia->'sello_hmac')::text ||
        ',"emitida_en":' || (v_evidencia->'emitida_en')::text ||
        ',"valida_hasta":' || (v_evidencia->'valida_hasta')::text ||
        ',"retener_hasta":' || (v_evidencia->'retener_hasta')::text || '}';
    RETURN '{"esquema":' || (p_artefacto->'esquema')::text ||
        ',"version":' || vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(p_artefacto->'version') ||
        ',"tipo":' || (p_artefacto->'tipo')::text ||
        ',"comando":' || v_comando_texto || ',"recibo":' || v_recibo_texto ||
        ',"evidencia":' || v_evidencia_texto ||
        ',"clave_verificacion_ref":' || (p_artefacto->'clave_verificacion_ref')::text ||
        ',"sello_hmac":' || (p_artefacto->'sello_hmac')::text ||
        ',"huella_artefacto_sha256":' || CASE WHEN p_vaciar_huella
            THEN '""' ELSE (p_artefacto->'huella_artefacto_sha256')::text END || '}';
END
$function$;
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(p_clave uuid, p_huella text, p_reserva text, p_solicitud_texto text, p_recibo_texto text, p_artefacto text)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO pg_catalog, pg_temp
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
 SET idle_in_transaction_session_timeout TO '20s'
AS $function$
DECLARE
    p_solicitud jsonb;
    p_recibo jsonb;
    v_ejecucion vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
    v_artefacto jsonb;
    v_comando jsonb;
    v_contexto jsonb;
    v_datos jsonb;
    v_procedencia jsonb;
    v_evidencia_recibo jsonb;
    v_evidencia jsonb;
BEGIN
    IF NOT pg_catalog.pg_has_role(
        session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
    ) OR p_clave IS NULL OR p_huella IS NULL OR p_reserva IS NULL
       OR p_solicitud_texto IS NULL OR p_recibo_texto IS NULL OR p_artefacto IS NULL
       OR pg_catalog.octet_length(p_solicitud_texto) > 1048576
       OR pg_catalog.octet_length(p_artefacto) > 1048576
       OR pg_catalog.octet_length(p_recibo_texto) >
          1048576 - pg_catalog.octet_length(p_artefacto) THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'confirmacion O6 invalida';
    END IF;
    p_solicitud := vec_contratacion_temporal.solicitud_desde_texto_seleccion_llamamiento_o6_v1(
        p_solicitud_texto
    );
    p_recibo := vec_contratacion_temporal.recibo_desde_texto_seleccion_llamamiento_o6_v1(
        p_recibo_texto
    );
    IF p_solicitud IS NULL OR p_recibo IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'confirmacion O6 invalida';
    END IF;
    BEGIN
        v_artefacto := p_artefacto::jsonb;
    EXCEPTION WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'confirmacion O6 invalida';
    END;
    SELECT ejecucion.* INTO STRICT v_ejecucion
      FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 ejecucion
     WHERE ejecucion.clave_idempotencia = p_clave FOR UPDATE;
    IF v_ejecucion.huella_semantica IS DISTINCT FROM p_huella
       OR v_ejecucion.solicitud_json IS DISTINCT FROM p_solicitud
       OR v_ejecucion.reserva_ref IS DISTINCT FROM p_reserva
       OR v_ejecucion.situacion <> 'propietaria'
       OR v_ejecucion.lease_hasta <= pg_catalog.clock_timestamp() THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'confirmacion O6 incompatible';
    END IF;
    v_comando := v_artefacto->'comando';
    v_contexto := v_comando->'contexto';
    v_datos := v_contexto->'datos';
    v_procedencia := p_recibo->'procedencia';
    v_evidencia_recibo := v_procedencia->'evidencia';
    v_evidencia := v_artefacto->'evidencia';
    IF pg_catalog.jsonb_typeof(v_artefacto) IS DISTINCT FROM 'object'
       OR (v_artefacto ?& ARRAY['esquema','version','tipo','comando','recibo',
           'evidencia','clave_verificacion_ref','sello_hmac','huella_artefacto_sha256'])
           IS DISTINCT FROM true
       OR v_artefacto - ARRAY['esquema','version','tipo','comando','recibo',
           'evidencia','clave_verificacion_ref','sello_hmac','huella_artefacto_sha256']
           IS DISTINCT FROM '{}'::jsonb
       OR pg_catalog.jsonb_typeof(v_comando) IS DISTINCT FROM 'object'
       OR (v_comando ?& ARRAY['contexto','necesidad','bolsa','orden','politica',
           'total_posiciones_orden','maxima_posicion_evaluable','huella_recibo_orden'])
           IS DISTINCT FROM true
       OR v_comando - ARRAY['contexto','necesidad','bolsa','orden','politica',
           'total_posiciones_orden','maxima_posicion_evaluable','huella_recibo_orden']
           IS DISTINCT FROM '{}'::jsonb
       OR pg_catalog.jsonb_typeof(v_contexto) IS DISTINCT FROM 'object'
       OR (v_contexto ?& ARRAY['datos','clave_verificacion_ref','sello_hmac'])
           IS DISTINCT FROM true
       OR v_contexto - ARRAY['datos','clave_verificacion_ref','sello_hmac']
           IS DISTINCT FROM '{}'::jsonb
       OR pg_catalog.jsonb_typeof(v_datos) IS DISTINCT FROM 'object'
       OR (v_datos ?& ARRAY['operacion_ref','organizacion_ref','expediente_ref',
           'version_expediente','correlacion_ref','contrato_version','autoridad_solicitante',
           'autorizacion','accion','recurso','finalidad','solicitada_en','valida_hasta'])
           IS DISTINCT FROM true
       OR v_datos - ARRAY['operacion_ref','organizacion_ref','expediente_ref',
           'version_expediente','correlacion_ref','contrato_version','autoridad_solicitante',
           'autorizacion','accion','recurso','finalidad','solicitada_en','valida_hasta']
           IS DISTINCT FROM '{}'::jsonb
       OR (p_recibo ?& ARRAY['operacion_ref','organizacion_ref','expediente_ref',
           'version_expediente','correlacion_ref','necesidad','bolsa','orden','politica',
           'resultado','propuesta_generada','propuesta','accion_evento','llamamiento_ref',
           'seleccion_ref','retencion_seleccion','orden_seleccionado','recibo_ref',
           'auditoria_ref','evento_ref','confirmada_en','procedencia']) IS DISTINCT FROM true
       OR p_recibo - ARRAY['operacion_ref','organizacion_ref','expediente_ref',
           'version_expediente','correlacion_ref','necesidad','bolsa','orden','politica',
           'resultado','propuesta_generada','propuesta','accion_evento','llamamiento_ref',
           'seleccion_ref','retencion_seleccion','orden_seleccionado','recibo_ref',
           'auditoria_ref','evento_ref','confirmada_en','procedencia','llamamiento_recuperado'] IS DISTINCT FROM '{}'::jsonb
       OR pg_catalog.jsonb_typeof(v_procedencia) IS DISTINCT FROM 'object'
       OR (v_procedencia ?& ARRAY['autoridad_ref','respuesta_ref','contrato_version',
           'fuente','evidencia']) IS DISTINCT FROM true
       OR v_procedencia - ARRAY['autoridad_ref','respuesta_ref','contrato_version',
           'fuente','evidencia'] IS DISTINCT FROM '{}'::jsonb
       OR pg_catalog.jsonb_typeof(v_evidencia_recibo) IS DISTINCT FROM 'object'
       OR (v_evidencia_recibo ?& ARRAY['evidencia_ref','clave_verificacion_ref',
           'sello_hmac','emitida_en','valida_hasta','retener_hasta']) IS DISTINCT FROM true
       OR v_evidencia_recibo - ARRAY['evidencia_ref','clave_verificacion_ref',
           'sello_hmac','emitida_en','valida_hasta','retener_hasta'] IS DISTINCT FROM '{}'::jsonb
       OR pg_catalog.jsonb_typeof(v_evidencia) IS DISTINCT FROM 'object'
       OR (v_evidencia ?& ARRAY['esquema','tipo_material','autoridad_ref',
           'clave_verificacion_ref','evidencia_ref','peticion_ref','huella_peticion_sha256',
           'respuesta_ref','huella_respuesta_sha256','sello_hmac','emitida_en',
           'valida_hasta','retener_hasta']) IS DISTINCT FROM true
       OR v_evidencia - ARRAY['esquema','tipo_material','autoridad_ref',
           'clave_verificacion_ref','evidencia_ref','peticion_ref','huella_peticion_sha256',
           'respuesta_ref','huella_respuesta_sha256','sello_hmac','emitida_en',
           'valida_hasta','retener_hasta'] IS DISTINCT FROM '{}'::jsonb THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'confirmacion O6 invalida';
    END IF;
    IF EXISTS (
        SELECT 1 FROM (VALUES
            (v_comando->'necesidad'), (v_comando->'bolsa'),
            (v_comando->'orden'), (v_comando->'politica'),
            (v_datos->'autorizacion'), (v_datos->'accion'),
            (v_datos->'recurso'), (v_datos->'finalidad'),
            (p_recibo->'necesidad'), (p_recibo->'bolsa'), (p_recibo->'orden'),
            (p_recibo->'politica'), (p_recibo->'resultado'), (p_recibo->'propuesta'),
            (p_recibo->'accion_evento'), (p_recibo->'retencion_seleccion'),
            (v_procedencia->'fuente')
        ) AS referencias(valor)
        WHERE pg_catalog.jsonb_typeof(valor) IS DISTINCT FROM 'object'
           OR (valor ?& ARRAY['referencia','version','huella_sha256']) IS DISTINCT FROM true
           OR valor - ARRAY['referencia','version','huella_sha256'] IS DISTINCT FROM '{}'::jsonb
           OR pg_catalog.jsonb_typeof(valor->'referencia') IS DISTINCT FROM 'string'
           OR valor->>'referencia' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR NOT CASE WHEN pg_catalog.jsonb_typeof(valor->'version') = 'number' THEN
                (valor->>'version')::numeric BETWEEN 1 AND 9007199254740991 AND
                valor->>'version' ~ '^[1-9][0-9]*$'
              ELSE false END
           OR pg_catalog.jsonb_typeof(valor->'huella_sha256') IS DISTINCT FROM 'string'
           OR valor->>'huella_sha256' !~ '^[0-9a-f]{64}$'
           OR valor->>'huella_sha256' = pg_catalog.repeat('0', 64)
    ) OR EXISTS (
        SELECT 1 FROM (VALUES
            (v_datos->'operacion_ref'), (v_datos->'organizacion_ref'),
            (v_datos->'expediente_ref'), (v_datos->'correlacion_ref'),
            (v_datos->'autoridad_solicitante'), (v_contexto->'clave_verificacion_ref'),
            (p_recibo->'operacion_ref'), (p_recibo->'organizacion_ref'),
            (p_recibo->'expediente_ref'), (p_recibo->'correlacion_ref'),
            (p_recibo->'llamamiento_ref'), (p_recibo->'recibo_ref'),
            (p_recibo->'auditoria_ref'), (p_recibo->'evento_ref'),
            (v_procedencia->'autoridad_ref'), (v_procedencia->'respuesta_ref'),
            (v_evidencia_recibo->'evidencia_ref'),
            (v_evidencia_recibo->'clave_verificacion_ref'),
            (v_evidencia->'autoridad_ref'), (v_evidencia->'clave_verificacion_ref'),
            (v_evidencia->'evidencia_ref'), (v_evidencia->'peticion_ref'),
            (v_evidencia->'respuesta_ref'), (v_artefacto->'clave_verificacion_ref')
        ) AS referencias(valor)
        WHERE pg_catalog.jsonb_typeof(valor) IS DISTINCT FROM 'string'
           OR valor #>> '{}' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'confirmacion O6 invalida';
    END IF;

    IF v_artefacto->>'esquema' IS DISTINCT FROM
           'vec.contratacion-temporal.artefacto-bolsa'
       OR v_artefacto->>'version' IS DISTINCT FROM '1'
       OR v_artefacto->>'tipo' IS DISTINCT FROM 'recibo_llamamiento'
       OR v_evidencia->>'esquema' IS DISTINCT FROM
           'vec.contratacion-temporal.evidencia-bolsa.v1'
       OR v_evidencia->>'tipo_material' IS DISTINCT FROM 'recibo_llamamiento'
       OR pg_catalog.jsonb_typeof(v_artefacto->'huella_artefacto_sha256')
           IS DISTINCT FROM 'string'
       OR v_artefacto->>'huella_artefacto_sha256' !~ '^[0-9a-f]{64}$'
       OR v_artefacto->>'huella_artefacto_sha256' = pg_catalog.repeat('0', 64)
       OR NOT vec_contratacion_temporal.confirmacion_canonica_seleccion_llamamiento_o6_v1(
           v_artefacto, p_artefacto, p_recibo, p_recibo_texto
       )
       OR pg_catalog.jsonb_typeof(v_comando->'huella_recibo_orden') IS DISTINCT FROM 'string'
       OR v_comando->>'huella_recibo_orden' !~ '^[0-9a-f]{64}$'
       OR v_comando->>'huella_recibo_orden' = pg_catalog.repeat('0', 64)
       OR pg_catalog.jsonb_typeof(v_evidencia->'huella_peticion_sha256') IS DISTINCT FROM 'string'
       OR v_evidencia->>'huella_peticion_sha256' !~ '^[0-9a-f]{64}$'
       OR v_evidencia->>'huella_peticion_sha256' = pg_catalog.repeat('0', 64)
       OR pg_catalog.jsonb_typeof(v_evidencia->'huella_respuesta_sha256') IS DISTINCT FROM 'string'
       OR v_evidencia->>'huella_respuesta_sha256' !~ '^[0-9a-f]{64}$'
       OR v_evidencia->>'huella_respuesta_sha256' = pg_catalog.repeat('0', 64)
       OR v_contexto->>'clave_verificacion_ref' !~
           '^vec[.]contratacion-temporal[.]integracion-bolsa-peticion/v[1-9][0-9]*$'
       OR v_contexto->>'sello_hmac' !~
           '^hmac-sha256:vec[.]contratacion-temporal[.]integracion-bolsa-peticion/v[1-9][0-9]*:[0-9a-f]{64}$'
       OR pg_catalog.split_part(v_contexto->>'sello_hmac', ':', 2)
           IS DISTINCT FROM v_contexto->>'clave_verificacion_ref'
       OR pg_catalog.right(v_contexto->>'sello_hmac', 64) = pg_catalog.repeat('0', 64)
       OR v_evidencia_recibo->>'clave_verificacion_ref' !~
           '^vec[.]contratacion-temporal[.]integracion-bolsa-respuesta/v[1-9][0-9]*$'
       OR v_evidencia_recibo->>'sello_hmac' !~
           '^hmac-sha256:vec[.]contratacion-temporal[.]integracion-bolsa-respuesta/v[1-9][0-9]*:[0-9a-f]{64}$'
       OR pg_catalog.split_part(v_evidencia_recibo->>'sello_hmac', ':', 2)
           IS DISTINCT FROM v_evidencia_recibo->>'clave_verificacion_ref'
       OR pg_catalog.right(v_evidencia_recibo->>'sello_hmac', 64) = pg_catalog.repeat('0', 64)
       OR p_recibo->>'seleccion_ref' !~
           '^hmac-sha256:vec[.]contratacion-temporal[.]seleccion/v[1-9][0-9]*:[0-9a-f]{64}$'
       OR pg_catalog.right(p_recibo->>'seleccion_ref', 64) = pg_catalog.repeat('0', 64) THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'confirmacion O6 invalida';
    END IF;

    IF NOT (CASE WHEN pg_catalog.jsonb_typeof(v_datos->'version_expediente') = 'number'
        THEN (v_datos->>'version_expediente')::numeric BETWEEN 1 AND 9007199254740991
         AND (v_datos->>'version_expediente')::numeric =
             pg_catalog.trunc((v_datos->>'version_expediente')::numeric) ELSE false END)
       OR v_datos->>'contrato_version' IS DISTINCT FROM '1'
       OR NOT (CASE WHEN pg_catalog.jsonb_typeof(v_comando->'total_posiciones_orden') = 'number'
        AND pg_catalog.jsonb_typeof(v_comando->'maxima_posicion_evaluable') = 'number' THEN
            (v_comando->>'total_posiciones_orden')::numeric BETWEEN 1 AND 250000 AND
            (v_comando->>'total_posiciones_orden')::numeric =
                pg_catalog.trunc((v_comando->>'total_posiciones_orden')::numeric) AND
            (v_comando->>'maxima_posicion_evaluable')::numeric =
                (v_comando->>'total_posiciones_orden')::numeric ELSE false END)
       OR NOT (CASE WHEN pg_catalog.jsonb_typeof(p_recibo->'version_expediente') = 'number'
        AND pg_catalog.jsonb_typeof(p_recibo->'orden_seleccionado') = 'number' THEN
            (p_recibo->>'version_expediente')::numeric BETWEEN 1 AND 9007199254740991 AND
            (p_recibo->>'version_expediente')::numeric =
                pg_catalog.trunc((p_recibo->>'version_expediente')::numeric) AND
            (p_recibo->>'orden_seleccionado')::numeric BETWEEN 1 AND
                (v_comando->>'maxima_posicion_evaluable')::numeric AND
            (p_recibo->>'orden_seleccionado')::numeric =
                pg_catalog.trunc((p_recibo->>'orden_seleccionado')::numeric) ELSE false END)
       OR v_procedencia->>'contrato_version' IS DISTINCT FROM '1'
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(v_datos->'solicitada_en', false)
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(v_datos->'valida_hasta', false)
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(p_recibo->'confirmada_en', false)
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(v_evidencia_recibo->'emitida_en', false)
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(v_evidencia_recibo->'valida_hasta', false)
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(v_evidencia_recibo->'retener_hasta', false)
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(v_evidencia->'emitida_en', false)
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(v_evidencia->'valida_hasta', false)
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(v_evidencia->'retener_hasta', false) THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'confirmacion O6 invalida';
    END IF;

    IF (v_datos->>'valida_hasta')::timestamptz <= (v_datos->>'solicitada_en')::timestamptz
       OR (v_datos->>'valida_hasta')::timestamptz -
          (v_datos->>'solicitada_en')::timestamptz > interval '15 minutes'
       OR (v_evidencia_recibo->>'valida_hasta')::timestamptz <=
          (v_evidencia_recibo->>'emitida_en')::timestamptz
       OR (v_evidencia_recibo->>'valida_hasta')::timestamptz -
          (v_evidencia_recibo->>'emitida_en')::timestamptz > interval '15 minutes'
       OR (v_evidencia_recibo->>'retener_hasta')::timestamptz <=
          (v_evidencia_recibo->>'valida_hasta')::timestamptz
       OR (v_evidencia_recibo->>'emitida_en')::timestamptz >=
          (v_datos->>'valida_hasta')::timestamptz
       OR (v_evidencia_recibo->>'valida_hasta')::timestamptz >
          (v_datos->>'valida_hasta')::timestamptz
       OR (p_recibo ? 'llamamiento_recuperado' AND
           p_recibo->'llamamiento_recuperado' IS DISTINCT FROM 'true'::jsonb)
       OR (p_recibo->'llamamiento_recuperado' = 'true'::jsonb AND
           ( (p_recibo->>'confirmada_en')::timestamptz >=
               (v_datos->>'solicitada_en')::timestamptz
             OR NOT EXISTS (
                SELECT 1 FROM vec_contratacion_temporal.historia_reanudacion_seleccion_llamamiento h
                JOIN vec_contratacion_temporal.outbox_reanudacion_seleccion_llamamiento o
                  ON o.auditoria_ref=h.auditoria_ref
                WHERE h.clave_idempotencia=p_clave
                  AND o.tipo='seleccion.solicitud_llamamiento.reanudada'
                  AND h.fencing_nuevo=v_ejecucion.fencing_version
             ) ))
       OR (p_recibo->'llamamiento_recuperado' IS NULL AND
           (p_recibo->>'confirmada_en')::timestamptz <
               (v_datos->>'solicitada_en')::timestamptz)
       OR (p_recibo->>'confirmada_en')::timestamptz >
          (v_evidencia_recibo->>'emitida_en')::timestamptz
       OR v_artefacto->'recibo' IS DISTINCT FROM p_recibo
       OR p_recibo->'propuesta_generada' IS DISTINCT FROM 'true'::jsonb
       OR v_artefacto->>'clave_verificacion_ref' IS DISTINCT FROM
          v_evidencia->>'clave_verificacion_ref'
       OR v_artefacto->>'sello_hmac' IS DISTINCT FROM v_evidencia->>'sello_hmac'
       OR v_evidencia->>'autoridad_ref' IS DISTINCT FROM v_procedencia->>'autoridad_ref'
       OR v_evidencia->>'clave_verificacion_ref' IS DISTINCT FROM
          v_evidencia_recibo->>'clave_verificacion_ref'
       OR v_evidencia->>'evidencia_ref' IS DISTINCT FROM v_evidencia_recibo->>'evidencia_ref'
       OR v_evidencia->>'peticion_ref' IS DISTINCT FROM v_datos->>'operacion_ref'
       OR v_evidencia->>'respuesta_ref' IS DISTINCT FROM v_procedencia->>'respuesta_ref'
       OR v_evidencia->>'sello_hmac' IS DISTINCT FROM v_evidencia_recibo->>'sello_hmac'
       OR v_evidencia->'emitida_en' IS DISTINCT FROM v_evidencia_recibo->'emitida_en'
       OR v_evidencia->'valida_hasta' IS DISTINCT FROM v_evidencia_recibo->'valida_hasta'
       OR v_evidencia->'retener_hasta' IS DISTINCT FROM v_evidencia_recibo->'retener_hasta'
       OR p_recibo->>'operacion_ref' IS DISTINCT FROM v_datos->>'operacion_ref'
       OR p_recibo->>'organizacion_ref' IS DISTINCT FROM v_datos->>'organizacion_ref'
       OR p_recibo->>'expediente_ref' IS DISTINCT FROM v_datos->>'expediente_ref'
       OR p_recibo->'version_expediente' IS DISTINCT FROM v_datos->'version_expediente'
       OR p_recibo->>'correlacion_ref' IS DISTINCT FROM v_datos->>'correlacion_ref'
       OR v_datos->'recurso' IS DISTINCT FROM v_comando->'orden'
       OR p_recibo->'necesidad' IS DISTINCT FROM v_comando->'necesidad'
       OR p_recibo->'bolsa' IS DISTINCT FROM v_comando->'bolsa'
       OR p_recibo->'orden' IS DISTINCT FROM v_comando->'orden'
       OR p_recibo->'politica' IS DISTINCT FROM v_comando->'politica'
       OR v_datos->>'organizacion_ref' IS DISTINCT FROM p_solicitud->>'organizacion_ref'
       OR v_datos->>'expediente_ref' IS DISTINCT FROM p_solicitud->>'expediente_ref'
       OR v_datos->'version_expediente' IS DISTINCT FROM p_solicitud->'version_expediente'
       OR v_datos->>'correlacion_ref' IS DISTINCT FROM p_solicitud->>'correlacion_ref'
       OR v_datos->'finalidad' IS DISTINCT FROM p_solicitud->'finalidad'
       OR v_comando->'necesidad' IS DISTINCT FROM p_solicitud->'necesidad'
       OR v_comando->'bolsa' IS DISTINCT FROM p_solicitud->'bolsa'
       OR v_comando->'politica' IS DISTINCT FROM p_solicitud->'politica'
       OR (v_comando->>'total_posiciones_orden')::numeric <
          (p_solicitud->>'cantidad_disponible')::numeric
       OR (v_comando->>'total_posiciones_orden')::numeric >
          (p_solicitud->>'maximo_posiciones')::numeric THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'confirmacion O6 invalida';
    END IF;
    IF NOT v_ejecucion.ventana_orden_abierta
       OR NOT v_ejecucion.ventana_llamamiento_abierta
       OR v_ejecucion.efecto <> 'solicitar_llamamiento' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'confirmacion O6 incompatible';
    END IF;
    UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
       SET situacion = 'confirmada', efecto = NULL,
           recibo_json = p_recibo, artefacto_canonico = p_artefacto,
           actualizada_en = pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
     WHERE clave_idempotencia = p_clave;
    RETURN true;
END
$function$;
COMMIT;
