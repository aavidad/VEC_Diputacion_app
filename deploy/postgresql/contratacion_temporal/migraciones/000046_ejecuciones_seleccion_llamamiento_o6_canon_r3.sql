-- CT-LITE-O6-03-FIX-R4: auxiliares privados del canon Go cerrado.
-- Se incluye dentro de la misma transaccion de 000046.up.sql.

DO $puerta$
BEGIN
    IF pg_catalog.current_setting('server_version_num') <> '180004'
       OR pg_catalog.getdatabaseencoding() IS DISTINCT FROM 'UTF8'
       OR pg_catalog.to_regnamespace('vec_contratacion_temporal') IS NULL
       OR pg_catalog.to_regrole('vec_contratacion_temporal_propietario') IS NULL
       OR pg_catalog.to_regrole('vec_contratacion_temporal_ejecutor') IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.instante_utc_json_canonico_v2(jsonb,boolean)'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'
       ) IS NOT NULL
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_proc funcion
            WHERE funcion.pronamespace =
                  'vec_contratacion_temporal'::pg_catalog.regnamespace
              AND funcion.proname = ANY(ARRAY[
                  'campo_canonico_seleccion_llamamiento_o6_v1',
                  'entero_json_seleccion_llamamiento_o6_v1',
                  'referencia_json_seleccion_llamamiento_o6_v1',
                  'huella_solicitud_seleccion_llamamiento_o6_v1',
                  'solicitud_json_seleccion_llamamiento_o6_v1',
                  'solicitud_desde_texto_seleccion_llamamiento_o6_v1',
                  'recibo_json_seleccion_llamamiento_o6_v1',
                  'recibo_desde_texto_seleccion_llamamiento_o6_v1',
                  'artefacto_json_seleccion_llamamiento_o6_v1',
                  'referencia_material_seleccion_llamamiento_o6_v1',
                  'contexto_material_seleccion_llamamiento_o6_v1',
                  'huellas_materiales_seleccion_llamamiento_o6_v1',
                  'materiales_ligados_seleccion_llamamiento_o6_v1',
                  'confirmacion_canonica_seleccion_llamamiento_o6_v1',
                  'resolver_terminal_seleccion_llamamiento_o6_v1',
                  'reservar_seleccion_llamamiento_o6_v1',
                  'abrir_ventana_seleccion_llamamiento_o6_v1',
                  'marcar_indeterminada_seleccion_llamamiento_o6_v1',
                  'liberar_seleccion_llamamiento_o6_v1',
                  'confirmar_seleccion_llamamiento_o6_v1',
                  'consultar_seleccion_llamamiento_o6_v1'
              ]::name[])
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar ejecuciones O6';
    END IF;
END
$puerta$;

CREATE FUNCTION vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
    p_nombre text, p_valor text
) RETURNS text LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.octet_length(p_nombre)::text || ':' || p_nombre ||
           pg_catalog.octet_length(p_valor)::text || ':' || p_valor
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(
    p_valor jsonb
) RETURNS text LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
    SELECT CASE WHEN pg_catalog.jsonb_typeof(p_valor) = 'number'
                     AND p_valor #>> '{}' ~ '^(0|[1-9][0-9]*)$'
                THEN p_valor #>> '{}' END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(
    p_valor jsonb
) RETURNS text LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
    SELECT '{"referencia":' || (p_valor->'referencia')::text ||
           ',"version":' || vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(p_valor->'version') ||
           ',"huella_sha256":' || (p_valor->'huella_sha256')::text || '}'
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.huella_solicitud_seleccion_llamamiento_o6_v1(
    p_solicitud jsonb
) RETURNS text LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_canon text;
    v_referencia record;
BEGIN
    IF pg_catalog.jsonb_typeof(p_solicitud) IS DISTINCT FROM 'object'
       OR pg_catalog.jsonb_typeof(p_solicitud->'version_expediente') IS DISTINCT FROM 'number'
       OR pg_catalog.jsonb_typeof(p_solicitud->'maximo_posiciones') IS DISTINCT FROM 'number'
       OR pg_catalog.jsonb_typeof(p_solicitud->'cantidad_disponible') IS DISTINCT FROM 'number'
       OR p_solicitud->>'version_expediente' !~ '^[1-9][0-9]*$'
       OR p_solicitud->>'maximo_posiciones' !~ '^[1-9][0-9]*$'
       OR p_solicitud->>'cantidad_disponible' !~ '^[1-9][0-9]*$' THEN
        RETURN NULL;
    END IF;
    FOR v_referencia IN
        SELECT valor FROM (VALUES
            (p_solicitud->'accion_orden'), (p_solicitud->'finalidad'),
            (p_solicitud->'necesidad'), (p_solicitud->'bolsa'),
            (p_solicitud->'politica')
        ) referencias(valor)
    LOOP
        IF pg_catalog.jsonb_typeof(v_referencia.valor) IS DISTINCT FROM 'object'
           OR pg_catalog.jsonb_typeof(v_referencia.valor->'referencia') IS DISTINCT FROM 'string'
           OR pg_catalog.jsonb_typeof(v_referencia.valor->'version') IS DISTINCT FROM 'number'
           OR pg_catalog.jsonb_typeof(v_referencia.valor->'huella_sha256') IS DISTINCT FROM 'string'
           OR v_referencia.valor->>'version' !~ '^[1-9][0-9]*$'
           OR v_referencia.valor->>'huella_sha256' !~ '^[0-9a-f]{64}$'
           OR v_referencia.valor->>'huella_sha256' = pg_catalog.repeat('0', 64) THEN
            RETURN NULL;
        END IF;
    END LOOP;

    v_canon :=
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'esquema', 'vec.contratacion-temporal.integracion-bolsa.v1') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'tipo', 'ejecucion-seleccion-llamamiento') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'clave_idempotencia', p_solicitud->>'clave_idempotencia') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'organizacion_ref', p_solicitud->>'organizacion_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'expediente_ref', p_solicitud->>'expediente_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'version_expediente', p_solicitud->>'version_expediente') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'correlacion_ref', p_solicitud->>'correlacion_ref');

    v_canon := v_canon ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'accion_ref', p_solicitud #>> '{accion_orden,referencia}') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'accion_version', p_solicitud #>> '{accion_orden,version}') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'accion_huella_sha256', p_solicitud #>> '{accion_orden,huella_sha256}') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'recurso_ref', p_solicitud #>> '{bolsa,referencia}') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'recurso_version', p_solicitud #>> '{bolsa,version}') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'recurso_huella_sha256', p_solicitud #>> '{bolsa,huella_sha256}');

    FOR v_referencia IN
        SELECT prefijo, valor FROM (VALUES
            (1, 'finalidad', p_solicitud->'finalidad'),
            (2, 'necesidad', p_solicitud->'necesidad'),
            (3, 'bolsa', p_solicitud->'bolsa'),
            (4, 'politica', p_solicitud->'politica')
        ) referencias(orden, prefijo, valor) ORDER BY orden
    LOOP
        v_canon := v_canon ||
            vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
                v_referencia.prefijo || '_ref', v_referencia.valor->>'referencia') ||
            vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
                v_referencia.prefijo || '_version', v_referencia.valor->>'version') ||
            vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
                v_referencia.prefijo || '_huella_sha256',
                v_referencia.valor->>'huella_sha256');
    END LOOP;
    v_canon := v_canon ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'maximo_posiciones', p_solicitud->>'maximo_posiciones') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'cantidad_disponible', p_solicitud->>'cantidad_disponible');
    RETURN pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(v_canon, 'UTF8')
    ), 'hex');
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.solicitud_json_seleccion_llamamiento_o6_v1(
    p_solicitud jsonb
) RETURNS text LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
    SELECT '{"clave_idempotencia":' || (p_solicitud->'clave_idempotencia')::text ||
        ',"huella_semantica":' || (p_solicitud->'huella_semantica')::text ||
        ',"organizacion_ref":' || (p_solicitud->'organizacion_ref')::text ||
        ',"expediente_ref":' || (p_solicitud->'expediente_ref')::text ||
        ',"version_expediente":' || vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(p_solicitud->'version_expediente') ||
        ',"correlacion_ref":' || (p_solicitud->'correlacion_ref')::text ||
        ',"accion_orden":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(p_solicitud->'accion_orden') ||
        ',"finalidad":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(p_solicitud->'finalidad') ||
        ',"necesidad":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(p_solicitud->'necesidad') ||
        ',"bolsa":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(p_solicitud->'bolsa') ||
        ',"politica":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(p_solicitud->'politica') ||
        ',"maximo_posiciones":' || vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(p_solicitud->'maximo_posiciones') ||
        ',"cantidad_disponible":' || vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(p_solicitud->'cantidad_disponible') || '}'
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.solicitud_desde_texto_seleccion_llamamiento_o6_v1(
    p_texto text
) RETURNS jsonb LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_solicitud jsonb;
BEGIN
    IF p_texto IS NULL OR pg_catalog.octet_length(p_texto) > 1048576 THEN
        RETURN NULL;
    END IF;
    BEGIN
        v_solicitud := p_texto::jsonb;
    EXCEPTION WHEN OTHERS THEN
        RETURN NULL;
    END;
    IF pg_catalog.jsonb_typeof(v_solicitud) IS DISTINCT FROM 'object'
       OR vec_contratacion_temporal.solicitud_json_seleccion_llamamiento_o6_v1(
           v_solicitud
       ) IS DISTINCT FROM p_texto
       OR vec_contratacion_temporal.huella_solicitud_seleccion_llamamiento_o6_v1(
           v_solicitud
       ) IS NULL THEN
        RETURN NULL;
    END IF;
    RETURN v_solicitud;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(
    p_recibo jsonb
) RETURNS text LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_procedencia jsonb := p_recibo->'procedencia';
    v_evidencia jsonb := v_procedencia->'evidencia';
    v_procedencia_texto text;
BEGIN
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
        ',"procedencia":' || v_procedencia_texto || '}';
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.recibo_desde_texto_seleccion_llamamiento_o6_v1(
    p_texto text
) RETURNS jsonb LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_recibo jsonb;
BEGIN
    IF p_texto IS NULL OR pg_catalog.octet_length(p_texto) > 1048576 THEN
        RETURN NULL;
    END IF;
    BEGIN
        v_recibo := p_texto::jsonb;
    EXCEPTION WHEN OTHERS THEN
        RETURN NULL;
    END;
    IF pg_catalog.jsonb_typeof(v_recibo) IS DISTINCT FROM 'object'
       OR vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(
           v_recibo
       ) IS DISTINCT FROM p_texto THEN
        RETURN NULL;
    END IF;
    RETURN v_recibo;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(
    p_artefacto jsonb, p_vaciar_huella boolean
) RETURNS text LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
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
        ',"procedencia":' || v_procedencia_texto || '}';

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
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
    p_prefijo text, p_valor jsonb
) RETURNS text LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
    SELECT
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            p_prefijo || '_ref', p_valor->>'referencia') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            p_prefijo || '_version', p_valor->>'version') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            p_prefijo || '_huella_sha256', p_valor->>'huella_sha256')
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.contexto_material_seleccion_llamamiento_o6_v1(
    p_datos jsonb
) RETURNS text LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
    SELECT
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'operacion_ref', p_datos->>'operacion_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'organizacion_ref', p_datos->>'organizacion_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'expediente_ref', p_datos->>'expediente_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'version_expediente', p_datos->>'version_expediente') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'correlacion_ref', p_datos->>'correlacion_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'contrato_version', p_datos->>'contrato_version') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'autoridad_solicitante', p_datos->>'autoridad_solicitante') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'autorizacion', p_datos->'autorizacion') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'accion', p_datos->'accion') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'recurso', p_datos->'recurso') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'finalidad', p_datos->'finalidad') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'solicitada_en', p_datos->>'solicitada_en') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'peticion_valida_hasta', p_datos->>'valida_hasta')
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.huellas_materiales_seleccion_llamamiento_o6_v1(
    p_artefacto jsonb
) RETURNS text[] LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_comando jsonb := p_artefacto->'comando';
    v_datos jsonb := p_artefacto #> '{comando,contexto,datos}';
    v_recibo jsonb := p_artefacto->'recibo';
    v_procedencia jsonb := v_recibo->'procedencia';
    v_evidencia_nominal jsonb := v_procedencia->'evidencia';
    v_prefijo text;
    v_peticion text;
    v_respuesta text;
BEGIN
    v_prefijo :=
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'esquema', 'vec.contratacion-temporal.integracion-bolsa.v1');
    v_peticion := v_prefijo ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'tipo', 'peticion-llamamiento') ||
        vec_contratacion_temporal.contexto_material_seleccion_llamamiento_o6_v1(v_datos) ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'necesidad', v_comando->'necesidad') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'bolsa', v_comando->'bolsa') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'orden', v_comando->'orden') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'politica', v_comando->'politica') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'total_posiciones_orden', v_comando->>'total_posiciones_orden') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'maxima_posicion_evaluable', v_comando->>'maxima_posicion_evaluable') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'huella_recibo_orden', v_comando->>'huella_recibo_orden');

    v_respuesta := v_prefijo ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'tipo', 'recibo-llamamiento-durable') ||
        vec_contratacion_temporal.contexto_material_seleccion_llamamiento_o6_v1(v_datos) ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'respuesta_operacion_ref', v_recibo->>'operacion_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'respuesta_organizacion_ref', v_recibo->>'organizacion_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'respuesta_expediente_ref', v_recibo->>'expediente_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'respuesta_version_expediente', v_recibo->>'version_expediente') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'respuesta_correlacion_ref', v_recibo->>'correlacion_ref') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'respuesta_necesidad', v_recibo->'necesidad') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'respuesta_resultado', v_recibo->'resultado');
    v_respuesta := v_respuesta ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'autoridad_ref', v_procedencia->>'autoridad_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'respuesta_ref', v_procedencia->>'respuesta_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'procedencia_contrato_version', v_procedencia->>'contrato_version') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'fuente', v_procedencia->'fuente') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'evidencia_ref', v_evidencia_nominal->>'evidencia_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'clave_verificacion_ref', v_evidencia_nominal->>'clave_verificacion_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'evidencia_emitida_en', v_evidencia_nominal->>'emitida_en') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'evidencia_valida_hasta', v_evidencia_nominal->>'valida_hasta') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'evidencia_retener_hasta', v_evidencia_nominal->>'retener_hasta');
    v_respuesta := v_respuesta ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'comando_necesidad', v_comando->'necesidad') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'comando_bolsa', v_comando->'bolsa') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'comando_orden', v_comando->'orden') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'comando_politica', v_comando->'politica') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'comando_total_posiciones', v_comando->>'total_posiciones_orden') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'comando_maxima_posicion', v_comando->>'maxima_posicion_evaluable') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'comando_huella_recibo_orden', v_comando->>'huella_recibo_orden');
    v_respuesta := v_respuesta ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'recibo_bolsa', v_recibo->'bolsa') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'recibo_orden', v_recibo->'orden') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'recibo_politica', v_recibo->'politica') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'recibo_propuesta_generada', v_recibo->>'propuesta_generada') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'recibo_propuesta', v_recibo->'propuesta') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'recibo_accion_evento', v_recibo->'accion_evento') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'llamamiento_ref', v_recibo->>'llamamiento_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'seleccion_ref_seudonimizada', v_recibo->>'seleccion_ref') ||
        vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(
            'retencion_seleccion', v_recibo->'retencion_seleccion') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'orden_seleccionado', v_recibo->>'orden_seleccionado') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'recibo_ref', v_recibo->>'recibo_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'auditoria_ref', v_recibo->>'auditoria_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'evento_ref', v_recibo->>'evento_ref') ||
        vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(
            'confirmada_en', v_recibo->>'confirmada_en');
    RETURN ARRAY[
        pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to(v_peticion, 'UTF8')
        ), 'hex'),
        pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to(v_respuesta, 'UTF8')
        ), 'hex')
    ];
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.materiales_ligados_seleccion_llamamiento_o6_v1(
    p_artefacto jsonb
) RETURNS boolean LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_huellas text[] := vec_contratacion_temporal.huellas_materiales_seleccion_llamamiento_o6_v1(
        p_artefacto
    );
BEGIN
    RETURN v_huellas[1] IS NOT DISTINCT FROM
           p_artefacto #>> '{evidencia,huella_peticion_sha256}'
       AND v_huellas[2] IS NOT DISTINCT FROM
           p_artefacto #>> '{evidencia,huella_respuesta_sha256}';
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.confirmacion_canonica_seleccion_llamamiento_o6_v1(
    p_artefacto jsonb, p_texto text, p_recibo jsonb, p_recibo_texto text
) RETURNS boolean LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
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
$funcion$;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(text, text),
    vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(jsonb),
    vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(jsonb),
    vec_contratacion_temporal.huella_solicitud_seleccion_llamamiento_o6_v1(jsonb),
    vec_contratacion_temporal.solicitud_json_seleccion_llamamiento_o6_v1(jsonb),
    vec_contratacion_temporal.solicitud_desde_texto_seleccion_llamamiento_o6_v1(text),
    vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(jsonb),
    vec_contratacion_temporal.recibo_desde_texto_seleccion_llamamiento_o6_v1(text),
    vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(jsonb, boolean),
    vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(text, jsonb),
    vec_contratacion_temporal.contexto_material_seleccion_llamamiento_o6_v1(jsonb),
    vec_contratacion_temporal.huellas_materiales_seleccion_llamamiento_o6_v1(jsonb),
    vec_contratacion_temporal.materiales_ligados_seleccion_llamamiento_o6_v1(jsonb),
    vec_contratacion_temporal.confirmacion_canonica_seleccion_llamamiento_o6_v1(jsonb, text, jsonb, text)
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
