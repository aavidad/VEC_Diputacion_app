-- T17 B1: conciliación, outbox y conservación gobernada.
BEGIN;
SET LOCAL ROLE vec_bolsa_importacion_convoca_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';
SET LOCAL idle_in_transaction_session_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_importacion_convoca:migraciones', 0
    )
);

DO $barrera$
BEGIN
    IF pg_catalog.to_regprocedure(
        'vec_bolsa_importacion_convoca.recuperar_lote_pagina_v1(text,integer,integer)'
    ) IS NULL OR pg_catalog.to_regprocedure(
        'vec_bolsa_importacion_convoca.conciliar_v1(text,text,text,text,text,text)'
    ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'version incompatible para migracion Convoca 000003';
    END IF;
END
$barrera$;

CREATE FUNCTION vec_bolsa_importacion_convoca.anexar_evento_estado(
    p_evento_ref text, p_importacion_ref text, p_tipo text,
    p_actor_ref text, p_estado_conciliacion text, p_estado_staging text,
    p_bloqueo_retencion boolean, p_registrada_en timestamptz
)
RETURNS text LANGUAGE plpgsql
SET search_path = pg_catalog AS $funcion$
DECLARE
    actual vec_bolsa_importacion_convoca.lote%ROWTYPE;
    secuencia bigint;
    huella text;
BEGIN
    SELECT * INTO actual FROM vec_bolsa_importacion_convoca.lote
     WHERE importacion_ref = p_importacion_ref FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = 'B1704',
            MESSAGE = 'importacion Convoca no encontrada';
    END IF;
    secuencia := actual.secuencia_historia + 1;
    huella := vec_bolsa_importacion_convoca.huella_evento_estado(
        p_evento_ref, p_importacion_ref, secuencia,
        actual.cabeza_historia_sha256, p_tipo, p_actor_ref,
        p_estado_conciliacion, p_estado_staging, p_bloqueo_retencion,
        p_registrada_en
    );
    UPDATE vec_bolsa_importacion_convoca.lote
       SET estado_conciliacion = p_estado_conciliacion,
           estado_staging = p_estado_staging,
           bloqueo_retencion = p_bloqueo_retencion,
           version = version + 1,
           secuencia_historia = secuencia,
           cabeza_historia_sha256 = huella
     WHERE importacion_ref = p_importacion_ref;
    INSERT INTO vec_bolsa_importacion_convoca.historia_estado (
        evento_ref, importacion_ref, secuencia, huella_anterior_sha256,
        tipo, actor_ref, estado_conciliacion, estado_staging,
        bloqueo_retencion, registrada_en, huella_evento_sha256
    ) VALUES (
        p_evento_ref, p_importacion_ref, secuencia,
        actual.cabeza_historia_sha256, p_tipo, p_actor_ref,
        p_estado_conciliacion, p_estado_staging, p_bloqueo_retencion,
        p_registrada_en, huella
    );
    RETURN huella;
END
$funcion$;

CREATE FUNCTION vec_bolsa_importacion_convoca.conciliar_v1(
    p_importacion_ref text,
    p_conciliacion_ref text,
    p_registro_corporativo_ref text,
    p_resultado text,
    p_actor_ref text,
    p_motivo_codigo text
)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = on AS $funcion$
DECLARE
    actual vec_bolsa_importacion_convoca.lote%ROWTYPE;
    previa vec_bolsa_importacion_convoca.conciliacion%ROWTYPE;
    instante timestamptz;
    huella text;
    evento_ref text;
    payload jsonb;
BEGIN
    IF vec_bolsa_importacion_convoca.texto_opaco_valido(
           p_importacion_ref, 512
       ) IS NOT TRUE
       OR vec_bolsa_importacion_convoca.texto_opaco_valido(
           p_conciliacion_ref, 512
       ) IS NOT TRUE
       OR vec_bolsa_importacion_convoca.texto_opaco_valido(
           p_registro_corporativo_ref, 512
       ) IS NOT TRUE
       OR p_resultado NOT IN ('confirmada','descartada')
       OR p_resultado IS NULL
       OR vec_bolsa_importacion_convoca.codigo_gobernado_valido(
           p_actor_ref
       ) IS NOT TRUE
       OR vec_bolsa_importacion_convoca.codigo_gobernado_valido(
           p_motivo_codigo
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'B1701',
            MESSAGE = 'conciliacion Convoca no confiable';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(p_importacion_ref, 17002)
    );
    SELECT * INTO actual FROM vec_bolsa_importacion_convoca.lote
     WHERE importacion_ref = p_importacion_ref FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = 'B1704',
            MESSAGE = 'importacion Convoca no encontrada';
    END IF;
    IF vec_bolsa_importacion_convoca.lote_integro(
        p_importacion_ref
    ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'B1701',
            MESSAGE = 'lote Convoca durable no confiable';
    END IF;
    SELECT * INTO previa FROM vec_bolsa_importacion_convoca.conciliacion
     WHERE conciliacion_ref = p_conciliacion_ref;
    IF FOUND THEN
        IF previa.importacion_ref <> p_importacion_ref
           OR previa.registro_corporativo_ref <>
              p_registro_corporativo_ref
           OR previa.resultado <> p_resultado
           OR previa.actor_ref <> p_actor_ref
           OR previa.motivo_codigo <> p_motivo_codigo THEN
            RAISE EXCEPTION USING ERRCODE = 'B1702',
                MESSAGE = 'conciliacion Convoca en conflicto';
        END IF;
        RETURN pg_catalog.jsonb_build_object(
            'importacion_ref', previa.importacion_ref,
            'conciliacion_ref', previa.conciliacion_ref,
            'resultado', previa.resultado,
            'registrada_en', pg_catalog.to_char(
                previa.registrada_en AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
            'reutilizada', true
        );
    END IF;
    IF actual.estado_conciliacion <> 'pendiente'
       OR actual.estado_staging <> 'disponible'
       OR EXISTS (
           SELECT 1 FROM vec_bolsa_importacion_convoca.conciliacion
            WHERE importacion_ref = p_importacion_ref
       ) THEN
        RAISE EXCEPTION USING ERRCODE = 'B1702',
            MESSAGE = 'conciliacion Convoca en conflicto';
    END IF;
    instante := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    huella := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
        pg_catalog.concat_ws(pg_catalog.chr(31),
            p_importacion_ref, p_conciliacion_ref,
            p_registro_corporativo_ref, p_resultado, p_actor_ref,
            p_motivo_codigo, pg_catalog.to_char(
                instante AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            )
        ), 'UTF8'
    )), 'hex');
    INSERT INTO vec_bolsa_importacion_convoca.conciliacion VALUES (
        p_conciliacion_ref, p_importacion_ref,
        p_registro_corporativo_ref, p_resultado, p_actor_ref,
        p_motivo_codigo, instante, huella
    );
    evento_ref := 'evento:conciliacion:'::text || p_conciliacion_ref;
    PERFORM vec_bolsa_importacion_convoca.anexar_evento_estado(
        evento_ref, p_importacion_ref, 'conciliacion', p_actor_ref,
        p_resultado, actual.estado_staging, actual.bloqueo_retencion,
        instante
    );
    IF p_resultado = 'confirmada' THEN
        evento_ref := 'outbox:importacion-convoca-confirmada:'::text ||
            p_conciliacion_ref;
        payload := pg_catalog.jsonb_build_object(
            'esquema',
                'vec.bolsa.importacion-convoca.confirmada.v1',
            'evento_ref', evento_ref,
            'importacion_ref', p_importacion_ref,
            'acta_ref', actual.acta_ref,
            'fichero_custodiado_ref',
                actual.acta_canonica->>'fichero_custodiado_ref',
            'huella_fichero_sha256', actual.huella_fichero_sha256,
            'huella_acta_sha256', actual.huella_acta_sha256,
            'conciliacion_ref', p_conciliacion_ref,
            'registro_corporativo_ref', p_registro_corporativo_ref,
            'procedencia', actual.acta_canonica->'procedencia'
        );
        INSERT INTO vec_bolsa_importacion_convoca.outbox VALUES (
            evento_ref, p_importacion_ref,
            'importacion_convoca_confirmada_v1', payload, instante,
            pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
                payload::text, 'UTF8'
            )), 'hex')
        );
    END IF;
    IF vec_bolsa_importacion_convoca.lote_integro(
        p_importacion_ref
    ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'B1701',
            MESSAGE = 'conciliacion Convoca no quedo integra';
    END IF;
    RETURN pg_catalog.jsonb_build_object(
        'importacion_ref', p_importacion_ref,
        'conciliacion_ref', p_conciliacion_ref,
        'resultado', p_resultado,
        'registrada_en', pg_catalog.to_char(
            instante AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'reutilizada', false
    );
END
$funcion$;

CREATE FUNCTION vec_bolsa_importacion_convoca.cambiar_bloqueo_retencion_v1(
    p_importacion_ref text,
    p_decision_ref text,
    p_actor_ref text,
    p_motivo_codigo text,
    p_bloqueado boolean
)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = on AS $funcion$
DECLARE
    actual vec_bolsa_importacion_convoca.lote%ROWTYPE;
    previa vec_bolsa_importacion_convoca.decision_retencion%ROWTYPE;
    instante timestamptz;
    huella text;
BEGIN
    IF vec_bolsa_importacion_convoca.texto_opaco_valido(
           p_importacion_ref, 512
       ) IS NOT TRUE
       OR vec_bolsa_importacion_convoca.texto_opaco_valido(
           p_decision_ref, 512
       ) IS NOT TRUE
       OR vec_bolsa_importacion_convoca.codigo_gobernado_valido(
           p_actor_ref
       ) IS NOT TRUE
       OR vec_bolsa_importacion_convoca.codigo_gobernado_valido(
           p_motivo_codigo
       ) IS NOT TRUE
       OR p_bloqueado IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = 'B1701',
            MESSAGE = 'decision de retencion Convoca no confiable';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(p_importacion_ref, 17003)
    );
    SELECT * INTO actual FROM vec_bolsa_importacion_convoca.lote
     WHERE importacion_ref = p_importacion_ref FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = 'B1704',
            MESSAGE = 'importacion Convoca no encontrada';
    END IF;
    IF vec_bolsa_importacion_convoca.lote_integro(
        p_importacion_ref
    ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'B1701',
            MESSAGE = 'lote Convoca durable no confiable';
    END IF;
    SELECT * INTO previa
      FROM vec_bolsa_importacion_convoca.decision_retencion
     WHERE decision_ref = p_decision_ref;
    IF FOUND THEN
        IF previa.importacion_ref <> p_importacion_ref
           OR previa.actor_ref <> p_actor_ref
           OR previa.motivo_codigo <> p_motivo_codigo
           OR previa.bloqueado <> p_bloqueado THEN
            RAISE EXCEPTION USING ERRCODE = 'B1703',
                MESSAGE = 'decision de retencion Convoca en conflicto';
        END IF;
        RETURN pg_catalog.jsonb_build_object(
            'importacion_ref', previa.importacion_ref,
            'decision_ref', previa.decision_ref,
            'bloqueado', previa.bloqueado,
            'registrada_en', pg_catalog.to_char(
                previa.registrada_en AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
            'reutilizada', true
        );
    END IF;
    IF actual.estado_staging <> 'disponible'
       OR actual.bloqueo_retencion = p_bloqueado THEN
        RAISE EXCEPTION USING ERRCODE = 'B1703',
            MESSAGE = 'decision de retencion Convoca en conflicto';
    END IF;
    instante := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    huella := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
        pg_catalog.concat_ws(pg_catalog.chr(31), p_importacion_ref,
            p_decision_ref, p_actor_ref, p_motivo_codigo,
            p_bloqueado::text, pg_catalog.to_char(
                instante AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            )
        ), 'UTF8'
    )), 'hex');
    INSERT INTO vec_bolsa_importacion_convoca.decision_retencion VALUES (
        p_decision_ref, p_importacion_ref, p_actor_ref, p_motivo_codigo,
        p_bloqueado, instante, huella
    );
    PERFORM vec_bolsa_importacion_convoca.anexar_evento_estado(
        'evento:retencion:'::text || p_decision_ref,
        p_importacion_ref, 'bloqueo_retencion', p_actor_ref,
        actual.estado_conciliacion, actual.estado_staging, p_bloqueado,
        instante
    );
    IF vec_bolsa_importacion_convoca.lote_integro(
        p_importacion_ref
    ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'B1701',
            MESSAGE = 'retencion Convoca no quedo integra';
    END IF;
    RETURN pg_catalog.jsonb_build_object(
        'importacion_ref', p_importacion_ref,
        'decision_ref', p_decision_ref,
        'bloqueado', p_bloqueado,
        'registrada_en', pg_catalog.to_char(
            instante AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'reutilizada', false
    );
END
$funcion$;

CREATE FUNCTION vec_bolsa_importacion_convoca.expurgar_staging_vencido_v1(
    p_ejecucion_ref text,
    p_actor_ref text,
    p_politica_retencion_ref text,
    p_politica_retencion_version bigint,
    p_limite integer
)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = on AS $funcion$
DECLARE
    previa vec_bolsa_importacion_convoca.ejecucion_retencion%ROWTYPE;
    referencias text[];
    referencia text;
    instante timestamptz;
    numero_lotes integer;
    numero_filas integer;
    huella text;
    actual vec_bolsa_importacion_convoca.lote%ROWTYPE;
BEGIN
    IF vec_bolsa_importacion_convoca.texto_opaco_valido(
           p_ejecucion_ref, 512
       ) IS NOT TRUE
       OR vec_bolsa_importacion_convoca.codigo_gobernado_valido(
           p_actor_ref
       ) IS NOT TRUE
       OR vec_bolsa_importacion_convoca.texto_opaco_valido(
           p_politica_retencion_ref, 512
       ) IS NOT TRUE
       OR p_politica_retencion_version < 1
       OR p_politica_retencion_version IS NULL
       OR p_limite NOT BETWEEN 1 AND 1000
       OR p_limite IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = 'B1701',
            MESSAGE = 'expurgo Convoca no confiable';
    END IF;
    IF vec_bolsa_importacion_convoca.politica_retencion_integra()
       IS NOT TRUE
       OR NOT EXISTS (
           SELECT 1
             FROM vec_bolsa_importacion_convoca.politica_retencion
            WHERE politica_retencion_ref =
                  p_politica_retencion_ref
              AND politica_retencion_version =
                  p_politica_retencion_version
       ) THEN
        RAISE EXCEPTION USING ERRCODE = 'B1701',
            MESSAGE = 'politica de expurgo Convoca no confiable';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(p_ejecucion_ref, 17004)
    );
    SELECT * INTO previa
      FROM vec_bolsa_importacion_convoca.ejecucion_retencion
     WHERE ejecucion_ref = p_ejecucion_ref;
    IF FOUND THEN
        IF previa.actor_ref <> p_actor_ref
           OR previa.politica_retencion_ref <>
              p_politica_retencion_ref
           OR previa.politica_retencion_version <>
              p_politica_retencion_version
           OR previa.limite <> p_limite THEN
            RAISE EXCEPTION USING ERRCODE = 'B1703',
                MESSAGE = 'expurgo Convoca en conflicto';
        END IF;
        RETURN pg_catalog.jsonb_build_object(
            'ejecucion_ref', previa.ejecucion_ref,
            'lotes', previa.lotes, 'filas', previa.filas,
            'ejecutada_en', pg_catalog.to_char(
                previa.ejecutada_en AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
            'reutilizada', true
        );
    END IF;
    instante := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    SELECT COALESCE(
               pg_catalog.array_agg(
                   importacion_ref ORDER BY conservar_staging_hasta,
                   importacion_ref
               ), ARRAY[]::text[]
           )
      INTO referencias
      FROM (
          SELECT importacion_ref, conservar_staging_hasta
            FROM vec_bolsa_importacion_convoca.lote
           WHERE politica_retencion_ref =
                 p_politica_retencion_ref
             AND politica_retencion_version =
                 p_politica_retencion_version
             AND estado_staging = 'disponible'
             AND NOT bloqueo_retencion
             AND conservar_staging_hasta <= instante
           ORDER BY conservar_staging_hasta, importacion_ref
           FOR UPDATE SKIP LOCKED
           LIMIT p_limite
      ) AS candidatas;
    FOREACH referencia IN ARRAY referencias
    LOOP
        IF vec_bolsa_importacion_convoca.lote_integro(referencia)
           IS NOT TRUE THEN
            RAISE EXCEPTION USING ERRCODE = 'B1701',
                MESSAGE = 'lote candidato a expurgo no confiable';
        END IF;
    END LOOP;
    numero_lotes := pg_catalog.cardinality(referencias);
    SELECT pg_catalog.count(*) INTO numero_filas
      FROM vec_bolsa_importacion_convoca.fila_staging
     WHERE importacion_ref = ANY(referencias);
    DELETE FROM vec_bolsa_importacion_convoca.fila_staging
     WHERE importacion_ref = ANY(referencias);
    FOREACH referencia IN ARRAY referencias
    LOOP
        SELECT * INTO actual FROM vec_bolsa_importacion_convoca.lote
         WHERE importacion_ref = referencia FOR UPDATE;
        PERFORM vec_bolsa_importacion_convoca.anexar_evento_estado(
            'evento:expurgo:'::text || p_ejecucion_ref || ':' ||
                referencia,
            referencia, 'expurgo', p_actor_ref,
            actual.estado_conciliacion, 'expurgado', false, instante
        );
        IF vec_bolsa_importacion_convoca.lote_integro(referencia)
           IS NOT TRUE THEN
            RAISE EXCEPTION USING ERRCODE = 'B1701',
                MESSAGE = 'expurgo Convoca no quedo integro';
        END IF;
    END LOOP;
    huella := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
        pg_catalog.concat_ws(pg_catalog.chr(31), p_ejecucion_ref,
            p_actor_ref, p_politica_retencion_ref,
            p_politica_retencion_version::text, p_limite::text,
            numero_lotes::text, numero_filas::text,
            pg_catalog.to_char(instante AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        ), 'UTF8'
    )), 'hex');
    INSERT INTO vec_bolsa_importacion_convoca.ejecucion_retencion VALUES (
        p_ejecucion_ref, p_actor_ref, p_politica_retencion_ref,
        p_politica_retencion_version, p_limite, numero_lotes,
        numero_filas, instante, huella
    );
    RETURN pg_catalog.jsonb_build_object(
        'ejecucion_ref', p_ejecucion_ref,
        'lotes', numero_lotes, 'filas', numero_filas,
        'ejecutada_en', pg_catalog.to_char(
            instante AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'reutilizada', false
    );
END
$funcion$;

REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_importacion_convoca FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_importacion_convoca
    TO vec_bolsa_importacion_convoca_conciliador,
       vec_bolsa_importacion_convoca_retencion;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_importacion_convoca.conciliar_v1(
        text,text,text,text,text,text
    )
    TO vec_bolsa_importacion_convoca_conciliador;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_importacion_convoca.cambiar_bloqueo_retencion_v1(
        text,text,text,text,boolean
    ),
    vec_bolsa_importacion_convoca.expurgar_staging_vencido_v1(
        text,text,text,bigint,integer
    )
    TO vec_bolsa_importacion_convoca_retencion;
COMMIT;
