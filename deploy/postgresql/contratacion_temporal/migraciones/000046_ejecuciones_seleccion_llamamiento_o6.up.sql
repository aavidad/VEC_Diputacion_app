\set ON_ERROR_STOP on
-- CT-LITE-O6-03: idempotencia durable; no ejecuta efectos de Bolsa.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o6:ejecuciones-seleccion-llamamiento:v1', 0
));

DO $puerta$
BEGIN
    IF pg_catalog.current_setting('server_version_num') <> '180004'
       OR pg_catalog.getdatabaseencoding() IS DISTINCT FROM 'UTF8'
       OR pg_catalog.to_regnamespace('vec_contratacion_temporal') IS NULL
       OR pg_catalog.to_regrole('vec_contratacion_temporal_propietario') IS NULL
       OR pg_catalog.to_regrole('vec_contratacion_temporal_ejecutor') IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'
       ) IS NOT NULL
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc funcion
            WHERE funcion.pronamespace =
                  'vec_contratacion_temporal'::pg_catalog.regnamespace
              AND funcion.proname = ANY(ARRAY[
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

CREATE TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 (
    clave_idempotencia uuid PRIMARY KEY,
    huella_semantica text NOT NULL
        CHECK (huella_semantica ~ '^[0-9a-f]{64}$'),
    solicitud_json jsonb NOT NULL
        CHECK (pg_catalog.jsonb_typeof(solicitud_json) = 'object')
        CHECK (pg_catalog.octet_length(solicitud_json::text) <= 1048576)
        CHECK (solicitud_json ?& ARRAY[
            'clave_idempotencia', 'huella_semantica', 'organizacion_ref',
            'expediente_ref', 'version_expediente', 'correlacion_ref',
            'accion_orden', 'finalidad', 'necesidad', 'bolsa', 'politica',
            'maximo_posiciones', 'cantidad_disponible'
        ])
        CHECK (solicitud_json - ARRAY[
            'clave_idempotencia', 'huella_semantica', 'organizacion_ref',
            'expediente_ref', 'version_expediente', 'correlacion_ref',
            'accion_orden', 'finalidad', 'necesidad', 'bolsa', 'politica',
            'maximo_posiciones', 'cantidad_disponible'
        ] = '{}'::jsonb)
        CHECK (
            pg_catalog.jsonb_typeof(solicitud_json->'clave_idempotencia') = 'string' AND
            pg_catalog.jsonb_typeof(solicitud_json->'huella_semantica') = 'string' AND
            pg_catalog.jsonb_typeof(solicitud_json->'organizacion_ref') = 'string' AND
            pg_catalog.jsonb_typeof(solicitud_json->'expediente_ref') = 'string' AND
            pg_catalog.jsonb_typeof(solicitud_json->'version_expediente') = 'number' AND
            pg_catalog.jsonb_typeof(solicitud_json->'correlacion_ref') = 'string' AND
            pg_catalog.jsonb_typeof(solicitud_json->'maximo_posiciones') = 'number' AND
            pg_catalog.jsonb_typeof(solicitud_json->'cantidad_disponible') = 'number'
        )
        CHECK (CASE WHEN
            pg_catalog.jsonb_typeof(solicitud_json->'version_expediente') = 'number' AND
            pg_catalog.jsonb_typeof(solicitud_json->'maximo_posiciones') = 'number' AND
            pg_catalog.jsonb_typeof(solicitud_json->'cantidad_disponible') = 'number'
        THEN
            (solicitud_json->>'version_expediente')::numeric BETWEEN 1 AND 9007199254740991 AND
            (solicitud_json->>'version_expediente')::numeric =
                pg_catalog.trunc((solicitud_json->>'version_expediente')::numeric) AND
            (solicitud_json->>'maximo_posiciones')::numeric BETWEEN 1 AND 250000 AND
            (solicitud_json->>'maximo_posiciones')::numeric =
                pg_catalog.trunc((solicitud_json->>'maximo_posiciones')::numeric) AND
            (solicitud_json->>'cantidad_disponible')::numeric BETWEEN 1 AND
                (solicitud_json->>'maximo_posiciones')::numeric AND
            (solicitud_json->>'cantidad_disponible')::numeric =
                pg_catalog.trunc((solicitud_json->>'cantidad_disponible')::numeric)
        ELSE false END)
        CHECK (
            solicitud_json->>'organizacion_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' AND
            solicitud_json->>'expediente_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' AND
            solicitud_json->>'correlacion_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        )
        CHECK (
            pg_catalog.jsonb_typeof(solicitud_json->'accion_orden') = 'object' AND
            pg_catalog.jsonb_typeof(solicitud_json->'finalidad') = 'object' AND
            pg_catalog.jsonb_typeof(solicitud_json->'necesidad') = 'object' AND
            pg_catalog.jsonb_typeof(solicitud_json->'bolsa') = 'object' AND
            pg_catalog.jsonb_typeof(solicitud_json->'politica') = 'object'
        )
        CHECK (
            (solicitud_json->'accion_orden') ?& ARRAY['referencia', 'version', 'huella_sha256'] AND
            (solicitud_json->'finalidad') ?& ARRAY['referencia', 'version', 'huella_sha256'] AND
            (solicitud_json->'necesidad') ?& ARRAY['referencia', 'version', 'huella_sha256'] AND
            (solicitud_json->'bolsa') ?& ARRAY['referencia', 'version', 'huella_sha256'] AND
            (solicitud_json->'politica') ?& ARRAY['referencia', 'version', 'huella_sha256'] AND
            ((solicitud_json->'accion_orden') || (solicitud_json->'finalidad') ||
             (solicitud_json->'necesidad') || (solicitud_json->'bolsa') ||
             (solicitud_json->'politica')) -
                ARRAY['referencia', 'version', 'huella_sha256'] = '{}'::jsonb
        )
        CHECK (
            pg_catalog.jsonb_typeof(solicitud_json #> '{accion_orden,referencia}') = 'string' AND
            pg_catalog.jsonb_typeof(solicitud_json #> '{accion_orden,version}') = 'number' AND
            pg_catalog.jsonb_typeof(solicitud_json #> '{accion_orden,huella_sha256}') = 'string' AND
            pg_catalog.jsonb_typeof(solicitud_json #> '{finalidad,referencia}') = 'string' AND
            pg_catalog.jsonb_typeof(solicitud_json #> '{finalidad,version}') = 'number' AND
            pg_catalog.jsonb_typeof(solicitud_json #> '{finalidad,huella_sha256}') = 'string' AND
            pg_catalog.jsonb_typeof(solicitud_json #> '{necesidad,referencia}') = 'string' AND
            pg_catalog.jsonb_typeof(solicitud_json #> '{necesidad,version}') = 'number' AND
            pg_catalog.jsonb_typeof(solicitud_json #> '{necesidad,huella_sha256}') = 'string' AND
            pg_catalog.jsonb_typeof(solicitud_json #> '{bolsa,referencia}') = 'string' AND
            pg_catalog.jsonb_typeof(solicitud_json #> '{bolsa,version}') = 'number' AND
            pg_catalog.jsonb_typeof(solicitud_json #> '{bolsa,huella_sha256}') = 'string' AND
            pg_catalog.jsonb_typeof(solicitud_json #> '{politica,referencia}') = 'string' AND
            pg_catalog.jsonb_typeof(solicitud_json #> '{politica,version}') = 'number' AND
            pg_catalog.jsonb_typeof(solicitud_json #> '{politica,huella_sha256}') = 'string'
        )
        CHECK (
            solicitud_json #>> '{accion_orden,referencia}' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' AND
            solicitud_json #>> '{finalidad,referencia}' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' AND
            solicitud_json #>> '{necesidad,referencia}' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' AND
            solicitud_json #>> '{bolsa,referencia}' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' AND
            solicitud_json #>> '{politica,referencia}' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' AND
            solicitud_json #>> '{accion_orden,huella_sha256}' ~ '^[0-9a-f]{64}$' AND
            solicitud_json #>> '{finalidad,huella_sha256}' ~ '^[0-9a-f]{64}$' AND
            solicitud_json #>> '{necesidad,huella_sha256}' ~ '^[0-9a-f]{64}$' AND
            solicitud_json #>> '{bolsa,huella_sha256}' ~ '^[0-9a-f]{64}$' AND
            solicitud_json #>> '{politica,huella_sha256}' ~ '^[0-9a-f]{64}$'
        )
        CHECK (CASE WHEN
            pg_catalog.jsonb_typeof(solicitud_json #> '{accion_orden,version}') = 'number' AND
            pg_catalog.jsonb_typeof(solicitud_json #> '{finalidad,version}') = 'number' AND
            pg_catalog.jsonb_typeof(solicitud_json #> '{necesidad,version}') = 'number' AND
            pg_catalog.jsonb_typeof(solicitud_json #> '{bolsa,version}') = 'number' AND
            pg_catalog.jsonb_typeof(solicitud_json #> '{politica,version}') = 'number'
        THEN
            (solicitud_json #>> '{accion_orden,version}')::numeric BETWEEN 1 AND 9007199254740991 AND
            (solicitud_json #>> '{accion_orden,version}')::numeric =
                pg_catalog.trunc((solicitud_json #>> '{accion_orden,version}')::numeric) AND
            (solicitud_json #>> '{finalidad,version}')::numeric BETWEEN 1 AND 9007199254740991 AND
            (solicitud_json #>> '{finalidad,version}')::numeric =
                pg_catalog.trunc((solicitud_json #>> '{finalidad,version}')::numeric) AND
            (solicitud_json #>> '{necesidad,version}')::numeric BETWEEN 1 AND 9007199254740991 AND
            (solicitud_json #>> '{necesidad,version}')::numeric =
                pg_catalog.trunc((solicitud_json #>> '{necesidad,version}')::numeric) AND
            (solicitud_json #>> '{bolsa,version}')::numeric BETWEEN 1 AND 9007199254740991 AND
            (solicitud_json #>> '{bolsa,version}')::numeric =
                pg_catalog.trunc((solicitud_json #>> '{bolsa,version}')::numeric) AND
            (solicitud_json #>> '{politica,version}')::numeric BETWEEN 1 AND 9007199254740991 AND
            (solicitud_json #>> '{politica,version}')::numeric =
                pg_catalog.trunc((solicitud_json #>> '{politica,version}')::numeric)
        ELSE false END
        ),
    reserva_ref text NOT NULL UNIQUE
        DEFAULT (
            'reserva:seleccion-llamamiento:' ||
            pg_catalog.gen_random_uuid()::text
        )
        CHECK (
            reserva_ref ~
            '^reserva:seleccion-llamamiento:[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
        ),
    situacion text NOT NULL DEFAULT 'propietaria'
        CHECK (situacion IN ('propietaria', 'indeterminada', 'confirmada')),
    efecto text
        CHECK (efecto IN ('preparar_orden', 'solicitar_llamamiento')),
    ventana_orden_abierta boolean NOT NULL DEFAULT false,
    ventana_llamamiento_abierta boolean NOT NULL DEFAULT false,
    recibo_json jsonb,
    artefacto_json jsonb,
    creada_en timestamptz NOT NULL DEFAULT
        pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp()),
    actualizada_en timestamptz NOT NULL DEFAULT
        pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp()),
    CHECK (NOT ventana_llamamiento_abierta OR ventana_orden_abierta),
    CHECK (
        (recibo_json IS NULL AND artefacto_json IS NULL) OR
        (pg_catalog.jsonb_typeof(recibo_json) = 'object' AND
            pg_catalog.jsonb_typeof(artefacto_json) = 'object' AND
            pg_catalog.octet_length(recibo_json::text) +
            pg_catalog.octet_length(artefacto_json::text) <= 1048576)
    ),
    CHECK (
        (situacion = 'propietaria' AND recibo_json IS NULL AND
            artefacto_json IS NULL AND (
                (NOT ventana_orden_abierta AND NOT ventana_llamamiento_abierta AND efecto IS NULL) OR
                (ventana_orden_abierta AND NOT ventana_llamamiento_abierta AND efecto = 'preparar_orden') OR
                (ventana_orden_abierta AND ventana_llamamiento_abierta AND efecto = 'solicitar_llamamiento')
            )) OR
        (situacion = 'indeterminada' AND ventana_orden_abierta AND
            efecto IN ('preparar_orden', 'solicitar_llamamiento') AND
            recibo_json IS NULL AND artefacto_json IS NULL) OR
        (situacion = 'confirmada' AND ventana_orden_abierta AND
            ventana_llamamiento_abierta AND efecto IS NULL AND
            recibo_json IS NOT NULL AND artefacto_json IS NOT NULL)
    ),
    CHECK (
        solicitud_json->>'clave_idempotencia' IS NOT DISTINCT FROM
            clave_idempotencia::text AND
        solicitud_json->>'huella_semantica' IS NOT DISTINCT FROM
            huella_semantica
    ),
    CHECK (actualizada_en >= creada_en)
);

ALTER TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
    FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_ejecucion_seleccion_llamamiento_o6
    ON vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
    AS PERMISSIVE FOR ALL TO vec_contratacion_temporal_propietario
    USING (true) WITH CHECK (true);

CREATE FUNCTION vec_contratacion_temporal.resolver_terminal_seleccion_llamamiento_o6_v1(
    p_clave uuid
) RETURNS TABLE (
    situacion text, solicitud_json text, reserva_ref text,
    efecto text, recibo_json text, artefacto_json text
) LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET TimeZone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '15s'
SET idle_in_transaction_session_timeout = '20s'
AS $funcion$
BEGIN
    IF NOT pg_catalog.pg_has_role(
        session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
    ) OR p_clave IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'operacion O6 denegada';
    END IF;
    RETURN QUERY
    SELECT ejecucion.situacion, ejecucion.solicitud_json::text, ''::text,
           pg_catalog.coalesce(ejecucion.efecto, ''),
           pg_catalog.coalesce(ejecucion.recibo_json::text, ''),
           pg_catalog.coalesce(ejecucion.artefacto_json::text, '')
      FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 ejecucion
     WHERE ejecucion.clave_idempotencia = p_clave
       AND ejecucion.situacion IN ('confirmada', 'indeterminada');
    IF NOT FOUND THEN
        RETURN QUERY SELECT '', '', '', '', '', '';
    END IF;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(
    p_clave uuid, p_huella text, p_solicitud jsonb
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
    v_ejecucion vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
BEGIN
    IF NOT pg_catalog.pg_has_role(
        session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
    ) OR p_clave IS NULL OR p_huella IS NULL
       OR p_huella !~ '^[0-9a-f]{64}$'
       OR pg_catalog.jsonb_typeof(p_solicitud) IS DISTINCT FROM 'object'
       OR pg_catalog.octet_length(p_solicitud::text) > 1048576
       OR p_solicitud->>'clave_idempotencia' IS DISTINCT FROM p_clave::text
       OR p_solicitud->>'huella_semantica' IS DISTINCT FROM p_huella THEN
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

    IF v_insertadas = 1 THEN
        RETURN QUERY SELECT 'propietaria', v_ejecucion.solicitud_json::text,
            v_ejecucion.reserva_ref, '', '', '';
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
            pg_catalog.coalesce(v_ejecucion.artefacto_json::text, '');
    END IF;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.abrir_ventana_seleccion_llamamiento_o6_v1(
    p_clave uuid, p_huella text, p_reserva text, p_solicitud jsonb, p_efecto text
) RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET TimeZone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '15s'
SET idle_in_transaction_session_timeout = '20s'
AS $funcion$
DECLARE
    v_ejecucion vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
BEGIN
    IF NOT pg_catalog.pg_has_role(
        session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
    ) OR p_efecto IS NULL
       OR p_efecto NOT IN ('preparar_orden', 'solicitar_llamamiento') THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'ventana O6 denegada';
    END IF;
    SELECT ejecucion.* INTO STRICT v_ejecucion
      FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 ejecucion
     WHERE ejecucion.clave_idempotencia = p_clave FOR UPDATE;
    IF v_ejecucion.huella_semantica IS DISTINCT FROM p_huella
       OR v_ejecucion.solicitud_json IS DISTINCT FROM p_solicitud
       OR v_ejecucion.reserva_ref IS DISTINCT FROM p_reserva
       OR v_ejecucion.situacion <> 'propietaria'
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
    p_clave uuid, p_huella text, p_reserva text, p_solicitud jsonb, p_efecto text
) RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET TimeZone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '15s'
SET idle_in_transaction_session_timeout = '20s'
AS $funcion$
DECLARE
    v_ejecucion vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
BEGIN
    IF NOT pg_catalog.pg_has_role(
        session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
    ) OR p_efecto IS NULL
       OR p_efecto NOT IN ('preparar_orden', 'solicitar_llamamiento') THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'marca O6 denegada';
    END IF;
    SELECT ejecucion.* INTO STRICT v_ejecucion
      FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 ejecucion
     WHERE ejecucion.clave_idempotencia = p_clave FOR UPDATE;
    IF v_ejecucion.huella_semantica IS DISTINCT FROM p_huella
       OR v_ejecucion.solicitud_json IS DISTINCT FROM p_solicitud
       OR v_ejecucion.reserva_ref IS DISTINCT FROM p_reserva
       OR v_ejecucion.situacion <> 'propietaria'
       OR NOT v_ejecucion.ventana_orden_abierta THEN
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
    p_clave uuid, p_huella text, p_reserva text, p_solicitud jsonb
) RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET TimeZone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '15s'
SET idle_in_transaction_session_timeout = '20s'
AS $funcion$
DECLARE
    v_eliminadas integer;
BEGIN
    IF NOT pg_catalog.pg_has_role(
        session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'liberacion O6 denegada';
    END IF;
    DELETE FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
     WHERE clave_idempotencia = p_clave AND huella_semantica = p_huella
       AND reserva_ref = p_reserva AND solicitud_json = p_solicitud
       AND situacion = 'propietaria' AND NOT ventana_orden_abierta
       AND NOT ventana_llamamiento_abierta AND efecto IS NULL
       AND recibo_json IS NULL AND artefacto_json IS NULL;
    GET DIAGNOSTICS v_eliminadas = ROW_COUNT;
    IF v_eliminadas <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'liberacion O6 incompatible';
    END IF;
    RETURN true;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(
    p_clave uuid, p_huella text, p_reserva text, p_solicitud jsonb,
    p_recibo jsonb, p_artefacto jsonb
) RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET TimeZone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '15s'
SET idle_in_transaction_session_timeout = '20s'
AS $funcion$
DECLARE
    v_ejecucion vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
BEGIN
    IF NOT pg_catalog.pg_has_role(
        session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
    ) OR pg_catalog.jsonb_typeof(p_recibo) IS DISTINCT FROM 'object'
       OR pg_catalog.jsonb_typeof(p_artefacto) IS DISTINCT FROM 'object'
       OR pg_catalog.octet_length(p_recibo::text) +
          pg_catalog.octet_length(p_artefacto::text) > 1048576
       OR p_artefacto->>'esquema' IS DISTINCT FROM
          'vec.contratacion-temporal.artefacto-bolsa'
       OR p_artefacto->>'version' IS DISTINCT FROM '1'
       OR p_artefacto->>'tipo' IS DISTINCT FROM 'recibo_llamamiento'
       OR p_artefacto->'recibo' IS DISTINCT FROM p_recibo
       OR p_recibo->>'propuesta_generada' IS DISTINCT FROM 'true' THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'confirmacion O6 invalida';
    END IF;
    SELECT ejecucion.* INTO STRICT v_ejecucion
      FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 ejecucion
     WHERE ejecucion.clave_idempotencia = p_clave FOR UPDATE;

    IF v_ejecucion.situacion = 'confirmada'
       AND v_ejecucion.huella_semantica = p_huella
       AND v_ejecucion.solicitud_json = p_solicitud
       AND v_ejecucion.reserva_ref = p_reserva
       AND v_ejecucion.recibo_json = p_recibo
       AND v_ejecucion.artefacto_json = p_artefacto THEN
        RETURN true;
    END IF;
    IF v_ejecucion.huella_semantica IS DISTINCT FROM p_huella
       OR v_ejecucion.solicitud_json IS DISTINCT FROM p_solicitud
       OR v_ejecucion.reserva_ref IS DISTINCT FROM p_reserva
       OR v_ejecucion.situacion <> 'propietaria'
       OR NOT v_ejecucion.ventana_orden_abierta
       OR NOT v_ejecucion.ventana_llamamiento_abierta
       OR v_ejecucion.efecto <> 'solicitar_llamamiento' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'confirmacion O6 incompatible';
    END IF;
    UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
       SET situacion = 'confirmada', efecto = NULL,
           recibo_json = p_recibo, artefacto_json = p_artefacto,
           actualizada_en = pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
     WHERE clave_idempotencia = p_clave;
    RETURN true;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.consultar_seleccion_llamamiento_o6_v1(
    p_clave uuid, p_huella text, p_solicitud jsonb
) RETURNS TABLE (
    situacion text, solicitud_json text, reserva_ref text,
    efecto text, recibo_json text, artefacto_json text
) LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET TimeZone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '15s'
SET idle_in_transaction_session_timeout = '20s'
AS $funcion$
DECLARE
    v_ejecucion vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
BEGIN
    IF NOT pg_catalog.pg_has_role(
        session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
    ) OR p_clave IS NULL OR p_huella IS NULL
       OR pg_catalog.jsonb_typeof(p_solicitud) IS DISTINCT FROM 'object'
       OR pg_catalog.octet_length(p_solicitud::text) > 1048576 THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'consulta O6 denegada';
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
            pg_catalog.coalesce(v_ejecucion.artefacto_json::text, '');
    END IF;
END
$funcion$;

REVOKE ALL ON TABLE
    vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.resolver_terminal_seleccion_llamamiento_o6_v1(uuid),
    vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(uuid, text, jsonb),
    vec_contratacion_temporal.abrir_ventana_seleccion_llamamiento_o6_v1(uuid, text, text, jsonb, text),
    vec_contratacion_temporal.marcar_indeterminada_seleccion_llamamiento_o6_v1(uuid, text, text, jsonb, text),
    vec_contratacion_temporal.liberar_seleccion_llamamiento_o6_v1(uuid, text, text, jsonb),
    vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(uuid, text, text, jsonb, jsonb, jsonb),
    vec_contratacion_temporal.consultar_seleccion_llamamiento_o6_v1(uuid, text, jsonb)
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT USAGE ON SCHEMA vec_contratacion_temporal
    TO vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.resolver_terminal_seleccion_llamamiento_o6_v1(uuid),
    vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(uuid, text, jsonb),
    vec_contratacion_temporal.abrir_ventana_seleccion_llamamiento_o6_v1(uuid, text, text, jsonb, text),
    vec_contratacion_temporal.marcar_indeterminada_seleccion_llamamiento_o6_v1(uuid, text, text, jsonb, text),
    vec_contratacion_temporal.liberar_seleccion_llamamiento_o6_v1(uuid, text, text, jsonb),
    vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(uuid, text, text, jsonb, jsonb, jsonb),
    vec_contratacion_temporal.consultar_seleccion_llamamiento_o6_v1(uuid, text, jsonb)
    TO vec_contratacion_temporal_ejecutor;

COMMENT ON TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 IS
'CT-LITE-O6-03: ejecuciones idempotentes minimizadas; no contiene identidad directa';

COMMIT;
