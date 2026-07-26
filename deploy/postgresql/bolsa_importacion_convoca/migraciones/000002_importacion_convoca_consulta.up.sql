-- T17 B1: alta idempotente, consulta y recuperación paginada.
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
        'vec_bolsa_importacion_convoca.lote_integro(text)'
    ) IS NULL OR pg_catalog.to_regprocedure(
        'vec_bolsa_importacion_convoca.guardar_lote_v1(jsonb,jsonb)'
    ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'version incompatible para migracion Convoca 000002';
    END IF;
END
$barrera$;

CREATE FUNCTION vec_bolsa_importacion_convoca.huella_publicacion_retencion(
    p_politica_ref text, p_version bigint, p_duracion_segundos bigint,
    p_secuencia bigint, p_huella_anterior text, p_actor_ref text,
    p_publicada_en timestamptz
)
RETURNS text LANGUAGE sql IMMUTABLE
SET search_path = pg_catalog AS $funcion$
    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
        pg_catalog.concat_ws(pg_catalog.chr(31), p_politica_ref,
            p_version::text, p_duracion_segundos::text, p_secuencia::text,
            p_huella_anterior, p_actor_ref,
            pg_catalog.to_char(p_publicada_en AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        ), 'UTF8'
    )), 'hex')
$funcion$;

CREATE FUNCTION vec_bolsa_importacion_convoca.politica_retencion_integra()
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = on AS $funcion$
DECLARE
    politica vec_bolsa_importacion_convoca.politica_retencion%ROWTYPE;
    puntero vec_bolsa_importacion_convoca.politica_retencion_actual%ROWTYPE;
    anterior text := pg_catalog.repeat('0', 64);
    secuencia bigint := 0;
    ultima_ref text;
    ultima_version bigint;
BEGIN
    SELECT * INTO puntero
      FROM vec_bolsa_importacion_convoca.politica_retencion_actual
     WHERE ambito = 'staging_convoca';
    IF NOT FOUND THEN RETURN false; END IF;
    FOR politica IN
        SELECT * FROM vec_bolsa_importacion_convoca.politica_retencion
         ORDER BY secuencia_publicacion
    LOOP
        secuencia := secuencia + 1;
        IF politica.secuencia_publicacion <> secuencia
           OR politica.huella_anterior_sha256 <> anterior
           OR politica.huella_publicacion_sha256 <>
              vec_bolsa_importacion_convoca.huella_publicacion_retencion(
                  politica.politica_retencion_ref,
                  politica.politica_retencion_version,
                  politica.duracion_segundos,
                  politica.secuencia_publicacion,
                  politica.huella_anterior_sha256,
                  politica.actor_ref, politica.publicada_en
              ) THEN
            RETURN false;
        END IF;
        anterior := politica.huella_publicacion_sha256;
        ultima_ref := politica.politica_retencion_ref;
        ultima_version := politica.politica_retencion_version;
    END LOOP;
    RETURN secuencia = puntero.secuencia_publicacion
       AND ultima_ref = puntero.politica_retencion_ref
       AND ultima_version = puntero.politica_retencion_version;
END
$funcion$;

CREATE FUNCTION vec_bolsa_importacion_convoca.publicar_politica_retencion_v1(
    p_politica_retencion_ref text,
    p_politica_retencion_version bigint,
    p_duracion_segundos bigint,
    p_actor_ref text
)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = on AS $funcion$
DECLARE
    puntero vec_bolsa_importacion_convoca.politica_retencion_actual%ROWTYPE;
    previa vec_bolsa_importacion_convoca.politica_retencion%ROWTYPE;
    existente vec_bolsa_importacion_convoca.politica_retencion%ROWTYPE;
    secuencia bigint;
    anterior text;
    instante timestamptz;
    huella text;
    puntero_encontrado boolean;
    politica_encontrada boolean;
BEGIN
    IF vec_bolsa_importacion_convoca.texto_opaco_valido(
        p_politica_retencion_ref, 512
    ) IS NOT TRUE OR p_politica_retencion_version < 1
      OR p_politica_retencion_version IS NULL
      OR p_duracion_segundos NOT BETWEEN 3600 AND 3153600000
      OR p_duracion_segundos IS NULL
      OR vec_bolsa_importacion_convoca.codigo_gobernado_valido(
          p_actor_ref
      ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'B1701',
            MESSAGE = 'politica de retencion Convoca no confiable';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(
            'vec_bolsa_importacion_convoca:politica_retencion', 17005
        )
    );
    SELECT * INTO puntero
      FROM vec_bolsa_importacion_convoca.politica_retencion_actual
     WHERE ambito = 'staging_convoca' FOR UPDATE;
    puntero_encontrado := FOUND;
    SELECT * INTO existente
      FROM vec_bolsa_importacion_convoca.politica_retencion
     WHERE politica_retencion_ref = p_politica_retencion_ref
       AND politica_retencion_version = p_politica_retencion_version;
    politica_encontrada := FOUND;
    IF politica_encontrada THEN
        IF existente.duracion_segundos <> p_duracion_segundos
           OR existente.actor_ref <> p_actor_ref
           OR NOT puntero_encontrado
           OR puntero.politica_retencion_ref <>
              p_politica_retencion_ref
           OR puntero.politica_retencion_version <>
              p_politica_retencion_version THEN
            RAISE EXCEPTION USING ERRCODE = 'B1703',
                MESSAGE = 'politica de retencion Convoca en conflicto';
        END IF;
        RETURN pg_catalog.jsonb_build_object(
            'politica_retencion_ref', existente.politica_retencion_ref,
            'politica_retencion_version',
                existente.politica_retencion_version,
            'duracion_segundos', existente.duracion_segundos,
            'reutilizada', true
        );
    END IF;
    IF NOT puntero_encontrado THEN
        IF p_politica_retencion_version <> 1 THEN
            RAISE EXCEPTION USING ERRCODE = 'B1703',
                MESSAGE = 'primera version de politica Convoca no permitida';
        END IF;
        secuencia := 1;
        anterior := pg_catalog.repeat('0', 64);
    ELSE
        IF vec_bolsa_importacion_convoca.politica_retencion_integra()
           IS NOT TRUE THEN
            RAISE EXCEPTION USING ERRCODE = 'B1701',
                MESSAGE = 'cadena de politica Convoca no confiable';
        END IF;
        IF (p_politica_retencion_ref =
              puntero.politica_retencion_ref AND
            p_politica_retencion_version <>
              puntero.politica_retencion_version + 1)
           OR (p_politica_retencion_ref <>
                 puntero.politica_retencion_ref AND
               p_politica_retencion_version <> 1) THEN
            RAISE EXCEPTION USING ERRCODE = 'B1703',
                MESSAGE = 'secuencia de politica Convoca no permitida';
        END IF;
        SELECT * INTO previa
          FROM vec_bolsa_importacion_convoca.politica_retencion
         WHERE politica_retencion_ref = puntero.politica_retencion_ref
           AND politica_retencion_version =
               puntero.politica_retencion_version;
        IF NOT FOUND THEN
            RAISE EXCEPTION USING ERRCODE = 'B1701',
                MESSAGE = 'cadena de politica Convoca no confiable';
        END IF;
        secuencia := puntero.secuencia_publicacion + 1;
        anterior := previa.huella_publicacion_sha256;
    END IF;
    instante := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    huella := vec_bolsa_importacion_convoca.huella_publicacion_retencion(
        p_politica_retencion_ref, p_politica_retencion_version,
        p_duracion_segundos, secuencia, anterior, p_actor_ref, instante
    );
    INSERT INTO vec_bolsa_importacion_convoca.politica_retencion VALUES (
        p_politica_retencion_ref, p_politica_retencion_version,
        p_duracion_segundos, secuencia, anterior, instante, p_actor_ref,
        huella
    );
    INSERT INTO vec_bolsa_importacion_convoca.politica_retencion_actual
    VALUES (
        'staging_convoca', p_politica_retencion_ref,
        p_politica_retencion_version, secuencia, instante, p_actor_ref
    )
    ON CONFLICT (ambito) DO UPDATE SET
        politica_retencion_ref = EXCLUDED.politica_retencion_ref,
        politica_retencion_version =
            EXCLUDED.politica_retencion_version,
        secuencia_publicacion = EXCLUDED.secuencia_publicacion,
        actualizada_en = EXCLUDED.actualizada_en,
        actor_ref = EXCLUDED.actor_ref;
    RETURN pg_catalog.jsonb_build_object(
        'politica_retencion_ref', p_politica_retencion_ref,
        'politica_retencion_version', p_politica_retencion_version,
        'duracion_segundos', p_duracion_segundos, 'reutilizada', false
    );
END
$funcion$;

CREATE FUNCTION vec_bolsa_importacion_convoca.guardar_lote_v1(
    p_acta jsonb,
    p_filas jsonb
)
RETURNS TABLE(acta_canonica jsonb, reutilizada boolean)
LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = on AS $funcion$
DECLARE
    importacion text;
    registrada timestamptz;
    lote_existente vec_bolsa_importacion_convoca.lote%ROWTYPE;
    huella_staging text;
    huella_staging_semantica text;
    evento_ref text;
    huella_evento text;
    politica vec_bolsa_importacion_convoca.politica_retencion%ROWTYPE;
    conservar_staging_hasta timestamptz;
BEGIN
    -- Se valida el acta antes de extraer o convertir ninguno de sus campos.
    -- PostgreSQL puede reordenar expresiones booleanas; separar las barreras
    -- evita que una entrada mal tipada eluda el error de contrato B1701.
    IF vec_bolsa_importacion_convoca.acta_valida(p_acta) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'B1701',
            MESSAGE = 'lote Convoca no confiable';
    END IF;
    IF vec_bolsa_importacion_convoca.filas_protegidas_validas(
           p_filas, (p_acta->>'filas_aceptadas')::integer
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'B1701',
            MESSAGE = 'lote Convoca no confiable';
    END IF;
    importacion := p_acta->>'importacion_ref';
    registrada := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    p_acta := pg_catalog.jsonb_set(
        p_acta, '{registrada_en}',
        pg_catalog.to_jsonb(pg_catalog.to_char(
            registrada AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )), false
    );
    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               COALESCE(pg_catalog.string_agg(
                   pg_catalog.concat_ws(pg_catalog.chr(30),
                       value->>'numero', value->>'esquema_proteccion',
                       value->>'clave_ref',
                       value->>'clave_derivacion_ref',
                       value->>'clave_atestacion_ref',
                       value->>'nonce_hex',
                       value->>'contenido_cifrado_hex',
                       value->>'huella_contenido_cifrado_sha256',
                       value->>'derivacion_documento_hmac_sha256',
                       value->>'atestacion_fila_hmac_sha256'
                   ), ',' ORDER BY (value->>'numero')::integer
               ), ''), 'UTF8'
           )), 'hex'),
           pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               COALESCE(pg_catalog.string_agg(
                   (value->>'numero') || ':' ||
                   (value->>'clave_atestacion_ref') || ':' ||
                   (value->>'atestacion_fila_hmac_sha256'),
                   ',' ORDER BY (value->>'numero')::integer
               ), ''), 'UTF8'
           )), 'hex')
      INTO huella_staging, huella_staging_semantica
      FROM pg_catalog.jsonb_array_elements(p_filas);
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(importacion, 17001)
    );
    SELECT lote.* INTO lote_existente
      FROM vec_bolsa_importacion_convoca.lote AS lote
     WHERE lote.importacion_ref = importacion
     FOR SHARE;
    IF FOUND THEN
        IF lote_existente.acta_canonica - 'registrada_en' <>
           p_acta - 'registrada_en'
           OR lote_existente.huella_staging_semantica_sha256 <>
              huella_staging_semantica THEN
            RAISE EXCEPTION USING ERRCODE = 'B1706',
                MESSAGE = 'acta Convoca en conflicto para la misma huella';
        END IF;
        IF vec_bolsa_importacion_convoca.lote_integro(importacion)
           IS NOT TRUE THEN
            RAISE EXCEPTION USING ERRCODE = 'B1701',
                MESSAGE = 'lote Convoca durable no confiable';
        END IF;
        RETURN QUERY SELECT lote_existente.acta_canonica, true;
        RETURN;
    END IF;
    SELECT politica_durable.* INTO politica
      FROM vec_bolsa_importacion_convoca.politica_retencion_actual AS puntero
      JOIN vec_bolsa_importacion_convoca.politica_retencion AS politica_durable
        ON politica_durable.politica_retencion_ref =
           puntero.politica_retencion_ref
       AND politica_durable.politica_retencion_version =
           puntero.politica_retencion_version
     WHERE puntero.ambito = 'staging_convoca'
     FOR SHARE OF puntero, politica_durable;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = 'B1701',
            MESSAGE = 'politica de retencion Convoca no publicada';
    END IF;
    IF vec_bolsa_importacion_convoca.politica_retencion_integra()
       IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'B1701',
            MESSAGE = 'cadena de politica Convoca no confiable';
    END IF;
    conservar_staging_hasta := registrada + pg_catalog.make_interval(
        secs => politica.duracion_segundos::double precision
    );
    evento_ref := 'evento:importacion:'::text ||
        (p_acta->>'huella_fichero_sha256');
    huella_evento :=
        vec_bolsa_importacion_convoca.huella_evento_estado(
            evento_ref, importacion, 1, pg_catalog.repeat('0', 64),
            'importacion', p_acta->>'actor_ref', 'pendiente',
            'disponible', false, registrada
        );
    INSERT INTO vec_bolsa_importacion_convoca.lote (
        importacion_ref, acta_ref, huella_fichero_sha256, acta_canonica,
        huella_acta_sha256, huella_staging_sha256,
        huella_staging_semantica_sha256, registrada_en,
        politica_retencion_ref, politica_retencion_version,
        conservar_staging_hasta, secuencia_historia, cabeza_historia_sha256
    ) VALUES (
        importacion, p_acta->>'acta_ref', p_acta->>'huella_fichero_sha256',
        p_acta, pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            p_acta::text, 'UTF8'
        )), 'hex'), huella_staging, huella_staging_semantica, registrada,
        politica.politica_retencion_ref,
        politica.politica_retencion_version, conservar_staging_hasta, 1,
        huella_evento
    );
    INSERT INTO vec_bolsa_importacion_convoca.fila_staging (
        importacion_ref, numero, esquema_proteccion, clave_ref, nonce,
        contenido_cifrado, huella_contenido_cifrado_sha256,
        derivacion_documento_hmac_sha256, clave_derivacion_ref,
        clave_atestacion_ref, atestacion_fila_hmac_sha256
    )
    SELECT importacion, (value->>'numero')::integer,
           value->>'esquema_proteccion', value->>'clave_ref',
           pg_catalog.decode(value->>'nonce_hex', 'hex'),
           pg_catalog.decode(value->>'contenido_cifrado_hex', 'hex'),
           value->>'huella_contenido_cifrado_sha256',
           pg_catalog.decode(
               value->>'derivacion_documento_hmac_sha256', 'hex'
           ),
           value->>'clave_derivacion_ref',
           value->>'clave_atestacion_ref',
           pg_catalog.decode(value->>'atestacion_fila_hmac_sha256', 'hex')
      FROM pg_catalog.jsonb_array_elements(p_filas);
    INSERT INTO vec_bolsa_importacion_convoca.historia_estado (
        evento_ref, importacion_ref, secuencia, huella_anterior_sha256,
        tipo, actor_ref, estado_conciliacion, estado_staging,
        bloqueo_retencion, registrada_en, huella_evento_sha256
    ) VALUES (
        evento_ref, importacion, 1, pg_catalog.repeat('0', 64),
        'importacion', p_acta->>'actor_ref', 'pendiente', 'disponible',
        false, registrada, huella_evento
    );
    IF vec_bolsa_importacion_convoca.lote_integro(importacion)
       IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'B1701',
            MESSAGE = 'lote Convoca no quedo integro';
    END IF;
    RETURN QUERY SELECT p_acta, false;
END
$funcion$;

CREATE FUNCTION vec_bolsa_importacion_convoca.consultar_estado_v1(
    p_huella text
)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = on AS $funcion$
DECLARE actual vec_bolsa_importacion_convoca.lote%ROWTYPE;
BEGIN
    IF vec_bolsa_importacion_convoca.huella_valida(p_huella)
       IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'B1701',
            MESSAGE = 'huella Convoca no valida';
    END IF;
    SELECT * INTO actual FROM vec_bolsa_importacion_convoca.lote
     WHERE huella_fichero_sha256 = p_huella;
    IF NOT FOUND THEN RETURN NULL; END IF;
    IF vec_bolsa_importacion_convoca.lote_integro(
        actual.importacion_ref
    ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'B1701',
            MESSAGE = 'lote Convoca durable no confiable';
    END IF;
    RETURN pg_catalog.jsonb_build_object(
        'acta', actual.acta_canonica,
        'estado_conciliacion', actual.estado_conciliacion,
        'estado_staging', actual.estado_staging,
        'politica_retencion_ref', actual.politica_retencion_ref,
        'politica_retencion_version', actual.politica_retencion_version,
        'conservar_staging_hasta',
            pg_catalog.to_char(
                actual.conservar_staging_hasta AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
        'bloqueo_retencion', actual.bloqueo_retencion,
        'version', actual.version
    );
END
$funcion$;

CREATE FUNCTION vec_bolsa_importacion_convoca.recuperar_lote_pagina_v1(
    p_huella text, p_desde integer, p_limite integer
)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = on AS $funcion$
DECLARE
    estado jsonb;
    filas jsonb;
    importacion text;
    ultimo integer;
    siguiente integer;
BEGIN
    IF p_desde IS NULL OR p_desde < 2
       OR p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 512 THEN
        RAISE EXCEPTION USING ERRCODE = 'B1701',
            MESSAGE = 'pagina de staging Convoca no valida';
    END IF;
    IF p_desde = 2 THEN
        estado := vec_bolsa_importacion_convoca.consultar_estado_v1(
            p_huella
        );
    ELSE
        SELECT pg_catalog.jsonb_build_object(
            'acta', acta_canonica,
            'estado_conciliacion', estado_conciliacion,
            'estado_staging', estado_staging,
            'politica_retencion_ref', politica_retencion_ref,
            'politica_retencion_version', politica_retencion_version,
            'conservar_staging_hasta',
                pg_catalog.to_char(
                    conservar_staging_hasta AT TIME ZONE 'UTC',
                    'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
                ),
            'bloqueo_retencion', bloqueo_retencion,
            'version', version
        ) INTO estado
        FROM vec_bolsa_importacion_convoca.lote
        WHERE huella_fichero_sha256 = p_huella
          AND vec_bolsa_importacion_convoca.historia_integra(
              importacion_ref
          );
    END IF;
    IF estado IS NULL THEN RETURN NULL; END IF;
    IF estado->>'estado_staging' = 'expurgado' THEN
        RAISE EXCEPTION USING ERRCODE = 'B1705',
            MESSAGE = 'staging Convoca expurgado';
    END IF;
    importacion := estado#>>'{acta,importacion_ref}';
    WITH preparadas AS (
        SELECT numero, pg_catalog.jsonb_build_object(
            'numero', numero,
            'esquema_proteccion', esquema_proteccion,
            'clave_ref', clave_ref,
            'clave_derivacion_ref', clave_derivacion_ref,
            'clave_atestacion_ref', clave_atestacion_ref,
            'nonce_hex', pg_catalog.encode(nonce, 'hex'),
            'contenido_cifrado_hex',
                pg_catalog.encode(contenido_cifrado, 'hex'),
            'huella_contenido_cifrado_sha256',
                huella_contenido_cifrado_sha256,
            'derivacion_documento_hmac_sha256',
                pg_catalog.encode(
                    derivacion_documento_hmac_sha256, 'hex'
                ),
            'atestacion_fila_hmac_sha256',
                pg_catalog.encode(atestacion_fila_hmac_sha256, 'hex')
        ) AS fila,
        pg_catalog.row_number() OVER (ORDER BY numero) AS posicion
        FROM vec_bolsa_importacion_convoca.fila_staging
        WHERE importacion_ref = importacion AND numero >= p_desde
        ORDER BY numero
        LIMIT p_limite
    ), pesadas AS (
        SELECT numero, fila, posicion,
               pg_catalog.sum(
                   pg_catalog.octet_length(fila::text) + 1
               ) OVER (ORDER BY numero) AS acumulado
        FROM preparadas
    ), incluidas AS (
        SELECT numero, fila FROM pesadas
         WHERE acumulado <= 4194000 ORDER BY numero
    )
    SELECT COALESCE(
               pg_catalog.jsonb_agg(fila ORDER BY numero), '[]'::jsonb
           ),
           pg_catalog.max(numero)
      INTO filas, ultimo
      FROM incluidas;
    IF ultimo IS NULL AND EXISTS (
        SELECT 1 FROM vec_bolsa_importacion_convoca.fila_staging
         WHERE importacion_ref = importacion AND numero >= p_desde
    ) THEN
        RAISE EXCEPTION USING ERRCODE = 'B1701',
            MESSAGE = 'pagina de staging Convoca excede presupuesto';
    END IF;
    IF ultimo IS NOT NULL THEN
        SELECT pg_catalog.min(numero) INTO siguiente
          FROM vec_bolsa_importacion_convoca.fila_staging
         WHERE importacion_ref = importacion AND numero > ultimo;
    END IF;
    RETURN pg_catalog.jsonb_build_object(
        'estado', estado, 'filas', filas, 'siguiente_numero', siguiente
    );
END
$funcion$;

REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_importacion_convoca FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_importacion_convoca
    TO vec_bolsa_importacion_convoca_ejecutor,
       vec_bolsa_importacion_convoca_recuperador,
       vec_bolsa_importacion_convoca_gobernanza;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_importacion_convoca.guardar_lote_v1(jsonb,jsonb)
    TO vec_bolsa_importacion_convoca_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_importacion_convoca.consultar_estado_v1(text),
    vec_bolsa_importacion_convoca.recuperar_lote_pagina_v1(
        text,integer,integer
    )
    TO vec_bolsa_importacion_convoca_recuperador;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_importacion_convoca.publicar_politica_retencion_v1(
        text,bigint,bigint,text
    )
    TO vec_bolsa_importacion_convoca_gobernanza;
COMMIT;
