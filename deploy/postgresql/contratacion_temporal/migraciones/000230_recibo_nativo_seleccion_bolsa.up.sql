\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:ct230:recibo-nativo-o6',0));
-- CT230: compatibilidad cerrada del recibo nativo B7; la autoridad de Bolsa
-- y el HMAC se verifican en Go. SQL conserva el ejecutor nominal y el artefacto.
DO $pre_ct230$
DECLARE r record; actual text; p pg_catalog.pg_proc%ROWTYPE;
BEGIN
    IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 THEN
        RAISE EXCEPTION 'CT230: versión PostgreSQL incompatible' USING ERRCODE='55000';
    END IF;
    FOR r IN SELECT * FROM (VALUES
        ('vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(jsonb)', '6368311d4740658c725767fd0e5dee0c11c2f7ef537a0c350623eabe96102d0f', 'f', '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}'),
        ('vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(jsonb,boolean)', 'c635b5f0c4f1eb9db311df2df62523424d6c8dd44ec97f04eed2cab6b2349cf1', 'f', '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}'),
        ('vec_contratacion_temporal.confirmacion_canonica_seleccion_llamamiento_o6_v1(jsonb,text,jsonb,text)', 'a15229c49f0226dd693cebabcfe74cbb23ca8818e9cc2a143bece24e033fe39f', 'f', '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}'),
        ('vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(uuid,text,text,text,text,text)', '505047a3fd76348e85979b53c6b7f45023c99bac9894218ec8705984befed79f', 't', '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}')
    ) AS esperado(firma, huella, definer, acl) LOOP
        SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure(r.firma);
        IF NOT FOUND OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
           OR p.prosecdef IS DISTINCT FROM (r.definer='t')
           OR p.proacl::text IS DISTINCT FROM r.acl
           OR (r.definer='t' AND p.proconfig IS DISTINCT FROM ARRAY[
                'search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC',
                'lock_timeout=2s','statement_timeout=15s',
                'idle_in_transaction_session_timeout=20s'])
           OR (r.definer='f' AND p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']) THEN
            RAISE EXCEPTION 'CT230: propiedad, ACL o configuración divergente: %',r.firma USING ERRCODE='55000';
        END IF;
        actual := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex');
        IF actual IS DISTINCT FROM r.huella THEN
            RAISE EXCEPTION 'CT230: preimagen divergente: %',r.firma USING ERRCODE='55000';
        END IF;
    END LOOP;
END $pre_ct230$;

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
    IF (v_procedencia ? 'sin_retencion_declarada' AND
        v_procedencia->'sin_retencion_declarada' IS DISTINCT FROM 'true'::jsonb)
       OR (v_procedencia ? 'politica_provisional' AND
           (v_procedencia->'sin_retencion_declarada' IS DISTINCT FROM 'true'::jsonb OR
            v_procedencia->'politica_provisional' IS DISTINCT FROM 'true'::jsonb)) THEN
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
        ',"retener_hasta":' || (v_evidencia->'retener_hasta')::text || '}' ||
        CASE WHEN v_procedencia->'sin_retencion_declarada' = 'true'::jsonb
             THEN ',"sin_retencion_declarada":true' ELSE '' END ||
        CASE WHEN v_procedencia->'politica_provisional' = 'true'::jsonb
             THEN ',"politica_provisional":true' ELSE '' END || '}';
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

    IF (v_procedencia ? 'sin_retencion_declarada' AND
        v_procedencia->'sin_retencion_declarada' IS DISTINCT FROM 'true'::jsonb)
       OR (v_procedencia ? 'politica_provisional' AND
           (v_procedencia->'sin_retencion_declarada' IS DISTINCT FROM 'true'::jsonb OR
            v_procedencia->'politica_provisional' IS DISTINCT FROM 'true'::jsonb))
       OR (v_procedencia->'sin_retencion_declarada' IS DISTINCT FROM
           v_evidencia->'sin_retencion_declarada') THEN
        RETURN NULL;
    END IF;
    v_procedencia_texto := '{"autoridad_ref":' || (v_procedencia->'autoridad_ref')::text ||
        ',"respuesta_ref":' || (v_procedencia->'respuesta_ref')::text ||
        ',"contrato_version":' || vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(v_procedencia->'contrato_version') ||
        ',"fuente":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(v_procedencia->'fuente') ||
        ',"evidencia":{"evidencia_ref":' || (v_evidencia_nominal->'evidencia_ref')::text ||
        ',"clave_verificacion_ref":' || (v_evidencia_nominal->'clave_verificacion_ref')::text ||
        ',"sello_hmac":' || (v_evidencia_nominal->'sello_hmac')::text ||
        ',"emitida_en":' || (v_evidencia_nominal->'emitida_en')::text ||
        ',"valida_hasta":' || (v_evidencia_nominal->'valida_hasta')::text ||
        ',"retener_hasta":' || (v_evidencia_nominal->'retener_hasta')::text || '}' ||
        CASE WHEN v_procedencia->'sin_retencion_declarada' = 'true'::jsonb
             THEN ',"sin_retencion_declarada":true' ELSE '' END ||
        CASE WHEN v_procedencia->'politica_provisional' = 'true'::jsonb
             THEN ',"politica_provisional":true' ELSE '' END || '}';
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
        ',"retener_hasta":' || (v_evidencia->'retener_hasta')::text ||
        CASE WHEN v_evidencia->'sin_retencion_declarada' = 'true'::jsonb
             THEN ',"sin_retencion_declarada":true' ELSE '' END || '}';
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

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.confirmacion_canonica_seleccion_llamamiento_o6_v1(p_artefacto jsonb, p_texto text, p_recibo jsonb, p_recibo_texto text)
 RETURNS boolean
 LANGUAGE plpgsql
 IMMUTABLE PARALLEL SAFE STRICT
 SET search_path TO 'pg_catalog'
AS $function$
DECLARE
    v_comando jsonb := p_artefacto->'comando';
    v_datos jsonb := p_artefacto #> '{comando,contexto,datos}';
    v_recibo_artefacto jsonb := p_artefacto->'recibo';
    v_huella text := p_artefacto->>'huella_artefacto_sha256';
    v_canon text;
    v_preimagen text;
BEGIN
    IF pg_catalog.jsonb_typeof(p_artefacto->'version') IS DISTINCT FROM 'number'
       OR p_artefacto->>'version' IS DISTINCT FROM '1'
       OR pg_catalog.jsonb_typeof(v_datos->'contrato_version') IS DISTINCT FROM 'number'
       OR v_datos->>'contrato_version' IS DISTINCT FROM '1'
       OR pg_catalog.jsonb_typeof(v_datos->'version_expediente') IS DISTINCT FROM 'number'
       OR v_datos->>'version_expediente' !~ '^[1-9][0-9]*$'
       OR pg_catalog.jsonb_typeof(p_recibo->'version_expediente') IS DISTINCT FROM 'number'
       OR p_recibo->>'version_expediente' !~ '^[1-9][0-9]*$'
       OR pg_catalog.jsonb_typeof(p_recibo->'propuesta_generada') IS DISTINCT FROM 'boolean'
       OR p_recibo->'propuesta_generada' IS DISTINCT FROM 'true'::jsonb
       OR pg_catalog.jsonb_typeof(p_recibo->'llamamiento_ref') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(p_recibo->'seleccion_ref') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(p_recibo->'orden_seleccionado') IS DISTINCT FROM 'number'
       OR p_recibo->>'orden_seleccionado' !~ '^[1-9][0-9]*$'
       OR pg_catalog.jsonb_typeof(p_recibo #> '{procedencia,contrato_version}')
          IS DISTINCT FROM 'number'
       OR p_recibo #>> '{procedencia,contrato_version}' IS DISTINCT FROM '1'
       OR pg_catalog.jsonb_typeof(v_comando->'total_posiciones_orden') IS DISTINCT FROM 'number'
       OR pg_catalog.jsonb_typeof(v_comando->'maxima_posicion_evaluable') IS DISTINCT FROM 'number'
       OR v_comando->>'total_posiciones_orden' !~ '^[1-9][0-9]*$'
       OR v_comando->>'maxima_posicion_evaluable' !~ '^[1-9][0-9]*$'
       OR v_recibo_artefacto IS DISTINCT FROM p_recibo
       OR (p_recibo #> '{procedencia,sin_retencion_declarada}') IS DISTINCT FROM
          (p_artefacto #> '{evidencia,sin_retencion_declarada}')
       OR v_recibo_artefacto->'propuesta' IS DISTINCT FROM p_recibo->'propuesta'
       OR v_recibo_artefacto->'accion_evento' IS DISTINCT FROM p_recibo->'accion_evento'
       OR v_recibo_artefacto->'llamamiento_ref' IS DISTINCT FROM p_recibo->'llamamiento_ref'
       OR v_recibo_artefacto->'seleccion_ref' IS DISTINCT FROM p_recibo->'seleccion_ref'
       OR v_recibo_artefacto->'retencion_seleccion' IS DISTINCT FROM p_recibo->'retencion_seleccion'
       OR v_recibo_artefacto->'orden_seleccionado' IS DISTINCT FROM p_recibo->'orden_seleccionado'
       OR (p_recibo->>'orden_seleccionado')::numeric >
          (v_comando->>'maxima_posicion_evaluable')::numeric
       OR NOT vec_contratacion_temporal.materiales_ligados_seleccion_llamamiento_o6_v1(
           p_artefacto
       ) THEN
        RETURN false;
    END IF;
    v_canon := vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(
        p_artefacto, false
    );
    v_preimagen := vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(
        p_artefacto, true
    );
    RETURN p_recibo_texto IS NOT DISTINCT FROM
           vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(p_recibo)
       AND p_texto IS NOT DISTINCT FROM v_canon
       AND v_huella ~ '^[0-9a-f]{64}$'
       AND v_huella <> pg_catalog.repeat('0', 64)
       AND pg_catalog.encode(pg_catalog.sha256(
           pg_catalog.convert_to(v_preimagen, 'UTF8')
       ), 'hex') IS NOT DISTINCT FROM v_huella;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$function$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(p_clave uuid, p_huella text, p_reserva text, p_solicitud_texto text, p_recibo_texto text, p_artefacto text)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
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
           'fuente','evidencia','sin_retencion_declarada','politica_provisional'] IS DISTINCT FROM '{}'::jsonb
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
           'valida_hasta','retener_hasta','sin_retencion_declarada'] IS DISTINCT FROM '{}'::jsonb THEN
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
            (p_recibo->'accion_evento'),
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

    -- La marca sola no habilita el recibo nativo: se exige la tupla nominal
    -- del emisor constituido, sus dos evidencias y la representación cero de Go.
    IF v_procedencia ? 'sin_retencion_declarada' THEN
        IF v_procedencia->'sin_retencion_declarada' IS DISTINCT FROM 'true'::jsonb
           OR v_evidencia->'sin_retencion_declarada' IS DISTINCT FROM 'true'::jsonb
           OR (v_procedencia ? 'politica_provisional' AND
               v_procedencia->'politica_provisional' IS DISTINCT FROM 'true'::jsonb)
           OR v_procedencia->>'autoridad_ref' IS DISTINCT FROM
                'autoridad:bolsa:llamamiento:desarrollo'
           OR p_recibo->'accion_evento' IS DISTINCT FROM pg_catalog.jsonb_build_object(
                'referencia','llamamiento.emitir.v1','version',1,
                'huella_sha256','2472cb427ec518ae7f2fe8671284d890f184918cdd3d6d88d9eeb12c61e3a811')
           OR p_recibo->'retencion_seleccion' IS DISTINCT FROM
                '{"referencia":"","version":0,"huella_sha256":""}'::jsonb
           OR v_evidencia_recibo->>'retener_hasta' IS DISTINCT FROM '0001-01-01T00:00:00Z'
           OR v_evidencia->>'retener_hasta' IS DISTINCT FROM '0001-01-01T00:00:00Z'
           OR p_recibo #>> '{resultado,referencia}' IS DISTINCT FROM p_recibo->>'recibo_ref'
           OR p_recibo #>> '{resultado,version}' IS DISTINCT FROM '1'
           OR p_recibo #>> '{propuesta,version}' IS DISTINCT FROM '1'
           OR p_recibo #>> '{propuesta,huella_sha256}' IS DISTINCT FROM
                p_recibo #>> '{resultado,huella_sha256}'
           OR p_recibo #>> '{propuesta,referencia}' !~ '^decision:[0-9a-f]{32}$'
           OR p_recibo->>'auditoria_ref' !~ '^aud_v3_[0-9a-f]{32}$'
           OR v_procedencia->>'respuesta_ref' !~ '^respuesta-llamamiento:[a-p]{64}$'
           OR v_evidencia_recibo->>'evidencia_ref' !~ '^evidencia:[a-p]{64}$' THEN
            RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'confirmacion O6 invalida';
        END IF;
    ELSE
        -- El recibo anterior mantiene su referencia y su fecha de retención.
        IF v_procedencia ? 'politica_provisional'
           OR v_evidencia ? 'sin_retencion_declarada'
           OR EXISTS (
               SELECT 1 FROM (VALUES (p_recibo->'retencion_seleccion')) AS retencion(valor)
               WHERE pg_catalog.jsonb_typeof(valor) IS DISTINCT FROM 'object'
                  OR (valor ?& ARRAY['referencia','version','huella_sha256']) IS DISTINCT FROM true
                  OR valor - ARRAY['referencia','version','huella_sha256'] IS DISTINCT FROM '{}'::jsonb
                  OR pg_catalog.jsonb_typeof(valor->'referencia') IS DISTINCT FROM 'string'
                  OR valor->>'referencia' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
                  OR NOT (CASE WHEN pg_catalog.jsonb_typeof(valor->'version') = 'number' THEN
                       (valor->>'version')::numeric BETWEEN 1 AND 9007199254740991 AND
                       valor->>'version' ~ '^[1-9][0-9]*$' ELSE false END)
                  OR pg_catalog.jsonb_typeof(valor->'huella_sha256') IS DISTINCT FROM 'string'
                  OR valor->>'huella_sha256' !~ '^[0-9a-f]{64}$'
                  OR valor->>'huella_sha256' = pg_catalog.repeat('0', 64)
           ) THEN
            RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'confirmacion O6 invalida';
        END IF;
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
       OR (NOT (v_procedencia ? 'sin_retencion_declarada') AND
           (v_evidencia_recibo->>'retener_hasta')::timestamptz <=
           (v_evidencia_recibo->>'valida_hasta')::timestamptz)
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

DO $post_ct230$
DECLARE r record; actual text; p pg_catalog.pg_proc%ROWTYPE;
BEGIN
    FOR r IN SELECT * FROM (VALUES
        ('vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(jsonb)', 'b44fac672d69921fe48b57142895f7a4603c6f138808b41aae716412bea26a84', '3ccbd05c78fdb9adfcf9f0a98351961e62f09a175b6673042b82006a9036fb74', '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}', 'f'),
        ('vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(jsonb,boolean)', 'f5c2c1a88e9700c6473ecf2f707eaa779774af0af6e952f614a0f7f92fdc4ee9', '3cafbed6ca85c9faaa0ce7a6dcbb589cccff9a0b8463bd9c918b01f6cc4b03fd', '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}', 'f'),
        ('vec_contratacion_temporal.confirmacion_canonica_seleccion_llamamiento_o6_v1(jsonb,text,jsonb,text)', 'e413e04d3723ccc97606dcf0e3547cf806224626fc7cdaba8a836866ac17e6a3', '7c21b9bbbebec20af0fe13a3492b7bf1fd45161658f860da603995b458115a0a', '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}', 'f'),
        ('vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(uuid,text,text,text,text,text)', '4aeb436aa99b051ea819cbf54f1e73b4b8a21eda54b0f68b7454b7e1b43450b3', 'd765be31c44dfc810d575d6dd8ab253d3ad55c1cf7bc5a54ab45c316a801f8e0', '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}', 't')
    ) AS esperado(firma, definicion, fuente, acl, definer) LOOP
        SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure(r.firma);
        IF NOT FOUND OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
           OR p.proacl::text IS DISTINCT FROM r.acl
           OR p.prosecdef IS DISTINCT FROM (r.definer='t')
           OR (r.definer='t' AND p.proconfig IS DISTINCT FROM ARRAY[
                'search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC',
                'lock_timeout=2s','statement_timeout=15s',
                'idle_in_transaction_session_timeout=20s'])
           OR (r.definer='f' AND p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']) THEN
            RAISE EXCEPTION 'CT230: función o ACL alterada: %',r.firma USING ERRCODE='55000';
        END IF;
        actual := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex');
        IF actual IS DISTINCT FROM r.definicion OR
           pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex')
           IS DISTINCT FROM r.fuente THEN
            RAISE EXCEPTION 'CT230: postimagen divergente: %',r.firma USING ERRCODE='55000';
        END IF;
    END LOOP;
END $post_ct230$;
COMMIT;
