\set ON_ERROR_STOP on
-- CT-LITE-O6-REM-02: idempotencia durable, lease y fencing; no ejecuta efectos.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o6:ejecuciones-seleccion-llamamiento:v1', 0
));
\ir 000046_ejecuciones_seleccion_llamamiento_o6_canon_r3.sql
CREATE TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 (
    clave_idempotencia uuid PRIMARY KEY,
    huella_semantica text NOT NULL CHECK (
        huella_semantica ~ '^[0-9a-f]{64}$' AND
        huella_semantica <> pg_catalog.repeat('0', 64)
    ),
    solicitud_json jsonb NOT NULL CHECK (
        pg_catalog.jsonb_typeof(solicitud_json) = 'object' AND
        pg_catalog.octet_length(solicitud_json::text) <= 1048576 AND
        vec_contratacion_temporal.huella_solicitud_seleccion_llamamiento_o6_v1(
            solicitud_json
        ) IS NOT DISTINCT FROM huella_semantica AND
        solicitud_json->>'clave_idempotencia' IS NOT DISTINCT FROM clave_idempotencia::text AND
        solicitud_json->>'huella_semantica' IS NOT DISTINCT FROM huella_semantica
    ),
    reserva_ref text NOT NULL UNIQUE DEFAULT
        vec_contratacion_temporal.nuevo_token_fencing_seleccion_llamamiento_o6_v2()
        CHECK (reserva_ref ~ '^reserva:seleccion-llamamiento:v2:[0-9a-f]{64}$' AND
               reserva_ref <> ('reserva:seleccion-llamamiento:v2:' || pg_catalog.repeat('0', 64))),
    fencing_version bigint NOT NULL DEFAULT 1
        CHECK (fencing_version BETWEEN 1 AND 9007199254740991),
    situacion text NOT NULL DEFAULT 'propietaria'
        CHECK (situacion IN ('propietaria', 'indeterminada', 'confirmada')),
    efecto text
        CHECK (efecto IN ('preparar_orden', 'solicitar_llamamiento')),
    ventana_orden_abierta boolean NOT NULL DEFAULT false,
    ventana_llamamiento_abierta boolean NOT NULL DEFAULT false,
    recibo_json jsonb,
    artefacto_canonico text,
    creada_en timestamptz NOT NULL DEFAULT
        pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp()),
    lease_hasta timestamptz NOT NULL DEFAULT pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp() + interval '30 seconds'
    ),
    actualizada_en timestamptz NOT NULL DEFAULT
        pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp()),
    CHECK (NOT ventana_llamamiento_abierta OR ventana_orden_abierta),
    CHECK (
        (recibo_json IS NULL AND artefacto_canonico IS NULL) OR
        (pg_catalog.jsonb_typeof(recibo_json) = 'object' AND
            pg_catalog.jsonb_typeof(artefacto_canonico::jsonb) = 'object' AND
            pg_catalog.octet_length(recibo_json::text) +
            pg_catalog.octet_length(artefacto_canonico) <= 1048576)
    ),
    CHECK (
        (situacion = 'propietaria' AND recibo_json IS NULL AND
            artefacto_canonico IS NULL AND (
                (NOT ventana_orden_abierta AND NOT ventana_llamamiento_abierta AND efecto IS NULL) OR
                (ventana_orden_abierta AND NOT ventana_llamamiento_abierta AND efecto = 'preparar_orden') OR
                (ventana_orden_abierta AND ventana_llamamiento_abierta AND efecto = 'solicitar_llamamiento')
            )) OR
        (situacion = 'indeterminada' AND ventana_orden_abierta AND
            efecto IN ('preparar_orden', 'solicitar_llamamiento') AND
            recibo_json IS NULL AND artefacto_canonico IS NULL) OR
        (situacion = 'confirmada' AND ventana_orden_abierta AND
            ventana_llamamiento_abierta AND efecto IS NULL AND
            recibo_json IS NOT NULL AND artefacto_canonico IS NOT NULL)
    ),
    CHECK (actualizada_en >= creada_en),
    CHECK ((situacion <> 'propietaria' OR lease_hasta > actualizada_en) AND
           lease_hasta <= actualizada_en + interval '30 seconds')
);
ALTER TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
    FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_ejecucion_seleccion_llamamiento_o6
    ON vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
    AS PERMISSIVE FOR ALL TO vec_contratacion_temporal_propietario
    USING (true) WITH CHECK (true);
CREATE FUNCTION vec_contratacion_temporal.resolver_terminal_autorizado_seleccion_llamamiento_o6_v2(
    p_clave uuid, p_consulta_texto text
) RETURNS TABLE (
    situacion text, solicitud_json text, reserva_ref text,
    efecto text, recibo_json text, artefacto_json text
) LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET TimeZone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '15s'
SET idle_in_transaction_session_timeout = '20s'
AS $funcion$
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
    ELSIF v_ejecucion.situacion <> 'confirmada' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'terminal O6 no confirmado';
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
    ELSE
        RETURN QUERY SELECT v_ejecucion.situacion, v_ejecucion.solicitud_json::text,
            '', '', v_ejecucion.recibo_json::text, v_ejecucion.artefacto_canonico;
    END IF;
END
$funcion$;
CREATE FUNCTION vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(
    p_clave uuid, p_huella text, p_solicitud_texto text
) RETURNS TABLE (
    situacion text, solicitud_json text, reserva_ref text,
    efecto text, recibo_json text, artefacto_json text
) LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET TimeZone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '15s'
SET idle_in_transaction_session_timeout = '20s'
AS $funcion$
DECLARE
    v_insertadas integer := 0;
    v_ahora timestamptz;
    p_solicitud jsonb;
    v_ejecucion vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
BEGIN
    IF NOT pg_catalog.pg_has_role(
        session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
    ) OR p_clave IS NULL OR p_huella IS NULL
       OR p_huella !~ '^[0-9a-f]{64}$' OR p_solicitud_texto IS NULL
       OR pg_catalog.octet_length(p_solicitud_texto) > 1048576 THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'reserva O6 invalida';
    END IF;
    p_solicitud := vec_contratacion_temporal.solicitud_desde_texto_seleccion_llamamiento_o6_v1(
        p_solicitud_texto
    );
    IF p_solicitud IS NULL
       OR p_solicitud->>'clave_idempotencia' IS DISTINCT FROM p_clave::text
       OR p_solicitud->>'huella_semantica' IS DISTINCT FROM p_huella
       OR vec_contratacion_temporal.huella_solicitud_seleccion_llamamiento_o6_v1(
           p_solicitud
       ) IS DISTINCT FROM p_huella THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'reserva O6 invalida';
    END IF;
    INSERT INTO vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 (
        clave_idempotencia, huella_semantica, solicitud_json
    ) VALUES (p_clave, p_huella, p_solicitud)
    ON CONFLICT (clave_idempotencia) DO NOTHING;
    GET DIAGNOSTICS v_insertadas = ROW_COUNT;
    SELECT ejecucion.* INTO STRICT v_ejecucion
      FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 ejecucion
     WHERE ejecucion.clave_idempotencia = p_clave
     FOR UPDATE;
    v_ahora := pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp());
    IF v_insertadas = 1 THEN
        RETURN QUERY SELECT 'propietaria', v_ejecucion.solicitud_json::text,
            v_ejecucion.reserva_ref, '', '', '';
    ELSIF v_ejecucion.huella_semantica IS DISTINCT FROM p_huella
       OR v_ejecucion.solicitud_json IS DISTINCT FROM p_solicitud THEN
        RETURN QUERY SELECT 'colision', p_solicitud::text, '', '', '', '';
    ELSIF v_ejecucion.situacion = 'propietaria' AND v_ejecucion.lease_hasta > v_ahora THEN
        RETURN QUERY SELECT 'ocupada', v_ejecucion.solicitud_json::text, '',
            pg_catalog.coalesce(v_ejecucion.efecto, ''), '', '';
    ELSIF v_ejecucion.situacion = 'propietaria' AND
          (v_ejecucion.ventana_orden_abierta OR v_ejecucion.ventana_llamamiento_abierta) THEN
        UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
           SET situacion = 'indeterminada',
               efecto = CASE WHEN ventana_llamamiento_abierta
                   THEN 'solicitar_llamamiento' ELSE 'preparar_orden' END,
               actualizada_en = v_ahora
         WHERE clave_idempotencia = p_clave RETURNING * INTO v_ejecucion;
        RETURN QUERY SELECT 'indeterminada', v_ejecucion.solicitud_json::text, '',
            v_ejecucion.efecto, '', '';
    ELSIF v_ejecucion.situacion = 'propietaria' THEN
        UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
           SET reserva_ref = vec_contratacion_temporal.nuevo_token_fencing_seleccion_llamamiento_o6_v2(),
               fencing_version = fencing_version + 1,
               lease_hasta = v_ahora + interval '30 seconds', actualizada_en = v_ahora
         WHERE clave_idempotencia = p_clave RETURNING * INTO v_ejecucion;
        RETURN QUERY SELECT 'propietaria', v_ejecucion.solicitud_json::text,
            v_ejecucion.reserva_ref, '', '', '';
    ELSE
        RETURN QUERY SELECT v_ejecucion.situacion, v_ejecucion.solicitud_json::text, '',
            pg_catalog.coalesce(v_ejecucion.efecto, ''),
            pg_catalog.coalesce(v_ejecucion.recibo_json::text, ''),
            pg_catalog.coalesce(v_ejecucion.artefacto_canonico, '');
    END IF;
END
$funcion$;
CREATE FUNCTION vec_contratacion_temporal.abrir_ventana_seleccion_llamamiento_o6_v1(
    p_clave uuid, p_huella text, p_reserva text, p_solicitud_texto text, p_efecto text
) RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET TimeZone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '15s'
SET idle_in_transaction_session_timeout = '20s'
AS $funcion$
DECLARE
    p_solicitud jsonb;
    v_ejecucion vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
BEGIN
    IF NOT pg_catalog.pg_has_role(
        session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
    ) OR p_efecto IS NULL
       OR p_efecto NOT IN ('preparar_orden', 'solicitar_llamamiento') THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'ventana O6 denegada';
    END IF;
    p_solicitud := vec_contratacion_temporal.solicitud_desde_texto_seleccion_llamamiento_o6_v1(
        p_solicitud_texto
    );
    IF p_solicitud IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'ventana O6 invalida';
    END IF;
    SELECT ejecucion.* INTO STRICT v_ejecucion
      FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 ejecucion
     WHERE ejecucion.clave_idempotencia = p_clave FOR UPDATE;
    IF v_ejecucion.huella_semantica IS DISTINCT FROM p_huella
       OR v_ejecucion.solicitud_json IS DISTINCT FROM p_solicitud
       OR v_ejecucion.reserva_ref IS DISTINCT FROM p_reserva
       OR v_ejecucion.situacion <> 'propietaria'
       OR v_ejecucion.lease_hasta <= pg_catalog.clock_timestamp()
       OR (p_efecto = 'preparar_orden' AND (
           v_ejecucion.ventana_orden_abierta OR v_ejecucion.ventana_llamamiento_abierta
       )) OR (p_efecto = 'solicitar_llamamiento' AND (
           NOT v_ejecucion.ventana_orden_abierta OR v_ejecucion.ventana_llamamiento_abierta OR
           v_ejecucion.efecto <> 'preparar_orden'
       )) THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'ventana O6 incompatible';
    END IF;
    UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
       SET ventana_orden_abierta = true,
           ventana_llamamiento_abierta = (p_efecto = 'solicitar_llamamiento'),
           efecto = p_efecto,
           actualizada_en = pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
     WHERE clave_idempotencia = p_clave;
    RETURN true;
END
$funcion$;
CREATE FUNCTION vec_contratacion_temporal.marcar_indeterminada_seleccion_llamamiento_o6_v1(
    p_clave uuid, p_huella text, p_reserva text, p_solicitud_texto text, p_efecto text
) RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET TimeZone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '15s'
SET idle_in_transaction_session_timeout = '20s'
AS $funcion$
DECLARE
    p_solicitud jsonb;
    v_ejecucion vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
BEGIN
    IF NOT pg_catalog.pg_has_role(
        session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
    ) OR p_efecto IS NULL
       OR p_efecto NOT IN ('preparar_orden', 'solicitar_llamamiento') THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'marca O6 denegada';
    END IF;
    p_solicitud := vec_contratacion_temporal.solicitud_desde_texto_seleccion_llamamiento_o6_v1(
        p_solicitud_texto
    );
    IF p_solicitud IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'marca O6 invalida';
    END IF;
    SELECT ejecucion.* INTO STRICT v_ejecucion
      FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 ejecucion
     WHERE ejecucion.clave_idempotencia = p_clave FOR UPDATE;
    IF v_ejecucion.huella_semantica IS DISTINCT FROM p_huella
       OR v_ejecucion.solicitud_json IS DISTINCT FROM p_solicitud
       OR v_ejecucion.reserva_ref IS DISTINCT FROM p_reserva
       OR v_ejecucion.situacion <> 'propietaria'
       OR NOT v_ejecucion.ventana_orden_abierta
       OR p_efecto IS DISTINCT FROM (CASE WHEN v_ejecucion.ventana_llamamiento_abierta
              THEN 'solicitar_llamamiento' ELSE 'preparar_orden' END) THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'marca O6 incompatible';
    END IF;
    UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
       SET situacion = 'indeterminada',
           efecto = CASE WHEN ventana_llamamiento_abierta
               THEN 'solicitar_llamamiento' ELSE 'preparar_orden' END,
           actualizada_en = pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
     WHERE clave_idempotencia = p_clave;
    RETURN true;
END
$funcion$;
CREATE FUNCTION vec_contratacion_temporal.liberar_seleccion_llamamiento_o6_v1(
    p_clave uuid, p_huella text, p_reserva text, p_solicitud_texto text
) RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET TimeZone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '15s'
SET idle_in_transaction_session_timeout = '20s'
AS $funcion$
DECLARE
    v_eliminadas integer;
    p_solicitud jsonb;
BEGIN
    IF NOT pg_catalog.pg_has_role(
        session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'liberacion O6 denegada';
    END IF;
    p_solicitud := vec_contratacion_temporal.solicitud_desde_texto_seleccion_llamamiento_o6_v1(
        p_solicitud_texto
    );
    IF p_solicitud IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'liberacion O6 invalida';
    END IF;
    DELETE FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
     WHERE clave_idempotencia = p_clave AND huella_semantica = p_huella
       AND reserva_ref = p_reserva AND solicitud_json = p_solicitud
       AND situacion = 'propietaria' AND NOT ventana_orden_abierta
       AND NOT ventana_llamamiento_abierta AND efecto IS NULL
       AND recibo_json IS NULL AND artefacto_canonico IS NULL;
    GET DIAGNOSTICS v_eliminadas = ROW_COUNT;
    IF v_eliminadas <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'liberacion O6 incompatible';
    END IF;
    RETURN true;
END
$funcion$;
CREATE FUNCTION vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(
    p_clave uuid, p_huella text, p_reserva text, p_solicitud_texto text,
    p_recibo_texto text, p_artefacto text
) RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET TimeZone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '15s'
SET idle_in_transaction_session_timeout = '20s'
AS $funcion$
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
           'auditoria_ref','evento_ref','confirmada_en','procedencia'] IS DISTINCT FROM '{}'::jsonb
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
       OR (p_recibo->>'confirmada_en')::timestamptz <
          (v_datos->>'solicitada_en')::timestamptz
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
$funcion$;
CREATE FUNCTION vec_contratacion_temporal.consultar_seleccion_llamamiento_o6_v1(
    p_clave uuid, p_huella text, p_solicitud_texto text
) RETURNS TABLE (
    situacion text, solicitud_json text, reserva_ref text,
    efecto text, recibo_json text, artefacto_json text
) LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET TimeZone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '15s'
SET idle_in_transaction_session_timeout = '20s'
AS $funcion$
DECLARE
    p_solicitud jsonb;
    v_ejecucion vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
BEGIN
    IF NOT pg_catalog.pg_has_role(
        session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
    ) OR p_clave IS NULL OR p_huella IS NULL
       OR p_solicitud_texto IS NULL OR pg_catalog.octet_length(p_solicitud_texto) > 1048576 THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'consulta O6 denegada';
    END IF;
    p_solicitud := vec_contratacion_temporal.solicitud_desde_texto_seleccion_llamamiento_o6_v1(
        p_solicitud_texto
    );
    IF p_solicitud IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'consulta O6 invalida';
    END IF;
    SELECT ejecucion.* INTO v_ejecucion
      FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 ejecucion
     WHERE ejecucion.clave_idempotencia = p_clave;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'ejecucion O6 ausente';
    ELSIF v_ejecucion.huella_semantica IS DISTINCT FROM p_huella
       OR v_ejecucion.solicitud_json IS DISTINCT FROM p_solicitud THEN
        RETURN QUERY SELECT 'colision', p_solicitud::text, '', '', '', '';
    ELSIF v_ejecucion.situacion = 'propietaria' THEN
        RETURN QUERY SELECT 'ocupada', v_ejecucion.solicitud_json::text, '',
            pg_catalog.coalesce(v_ejecucion.efecto, ''), '', '';
    ELSE
        RETURN QUERY SELECT v_ejecucion.situacion, v_ejecucion.solicitud_json::text, '',
            pg_catalog.coalesce(v_ejecucion.efecto, ''),
            pg_catalog.coalesce(v_ejecucion.recibo_json::text, ''),
            pg_catalog.coalesce(v_ejecucion.artefacto_canonico, '');
    END IF;
END
$funcion$;
REVOKE ALL ON TABLE
    vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.resolver_terminal_autorizado_seleccion_llamamiento_o6_v2(uuid, text),
    vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(uuid, text, text),
    vec_contratacion_temporal.abrir_ventana_seleccion_llamamiento_o6_v1(uuid, text, text, text, text),
    vec_contratacion_temporal.marcar_indeterminada_seleccion_llamamiento_o6_v1(uuid, text, text, text, text),
    vec_contratacion_temporal.liberar_seleccion_llamamiento_o6_v1(uuid, text, text, text),
    vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(uuid, text, text, text, text, text),
    vec_contratacion_temporal.consultar_seleccion_llamamiento_o6_v1(uuid, text, text)
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT USAGE ON SCHEMA vec_contratacion_temporal
    TO vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.resolver_terminal_autorizado_seleccion_llamamiento_o6_v2(uuid, text),
    vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(uuid, text, text),
    vec_contratacion_temporal.abrir_ventana_seleccion_llamamiento_o6_v1(uuid, text, text, text, text),
    vec_contratacion_temporal.marcar_indeterminada_seleccion_llamamiento_o6_v1(uuid, text, text, text, text),
    vec_contratacion_temporal.liberar_seleccion_llamamiento_o6_v1(uuid, text, text, text),
    vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(uuid, text, text, text, text, text),
    vec_contratacion_temporal.consultar_seleccion_llamamiento_o6_v1(uuid, text, text)
    TO vec_contratacion_temporal_ejecutor;

COMMENT ON TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 IS
'CT-LITE-O6-REM-02: ejecuciones minimizadas con lease maximo 30s y fencing v2';
COMMIT;
