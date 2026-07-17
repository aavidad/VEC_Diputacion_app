\set ON_ERROR_STOP 1

BEGIN;

SET LOCAL ROLE vec_autorizacion_motivos_proyector;

DO $prueba$
BEGIN
    BEGIN
        PERFORM count(*) FROM vec_autorizacion.motivo_v2_catalogo_publicado;
        RAISE EXCEPTION 'el proyector pudo leer tablas';
    EXCEPTION WHEN insufficient_privilege THEN
        NULL;
    END;

    IF vec_autorizacion.publicar_motivos_autorizacion_v2(
        'evento_00000000000000000000000000000001', 1, repeat('1', 64),
        'motivos_autorizacion', 1, repeat('a', 64),
        '2026-01-01T00:00:00.000000Z'::timestamptz,
        '[{"clave":"dni_12345678z","vigente_desde":"2026-01-01T00:00:00.000000Z","vigente_hasta":null}]'::jsonb
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'se acepto una clave semantica o con PII';
    END IF;

    IF vec_autorizacion.publicar_motivos_autorizacion_v2(
        'evento_00000000000000000000000000000001', 1, repeat('1', 64),
        'motivos_autorizacion', 1, repeat('a', 64), '-infinity'::timestamptz,
        '[{"clave":"motivo_11111111111111111111111111111111","vigente_desde":"2026-01-01T00:00:00.000000Z","vigente_hasta":null}]'::jsonb
    ) IS NOT FALSE OR vec_autorizacion.publicar_motivos_autorizacion_v2(
        'evento_00000000000000000000000000000001', 1, repeat('1', 64),
        'motivos_autorizacion', 1, repeat('a', 64),
        '10000-01-01T00:00:00Z'::timestamptz,
        '[{"clave":"motivo_11111111111111111111111111111111","vigente_desde":"2026-01-01T00:00:00.000000Z","vigente_hasta":null}]'::jsonb
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'se acepto una publicacion no finita o fuera de rango';
    END IF;

    IF vec_autorizacion.publicar_motivos_autorizacion_v2(
        'evento_00000000000000000000000000000001', 1, repeat('1', 64),
        'motivos_autorizacion', 1, repeat('a', 64),
        '2026-01-01T00:00:00.000000Z'::timestamptz,
        '[{"clave":"motivo_11111111111111111111111111111111","vigente_desde":"2026-01-01T24:00:00.000000Z","vigente_hasta":null}]'::jsonb
    ) IS NOT FALSE OR vec_autorizacion.publicar_motivos_autorizacion_v2(
        'evento_00000000000000000000000000000001', 1, repeat('1', 64),
        'motivos_autorizacion', 1, repeat('a', 64),
        '2026-01-01T00:00:00.000000Z'::timestamptz,
        '[{"clave":"motivo_11111111111111111111111111111111","vigente_desde":"2026-01-01T23:59:60.000000Z","vigente_hasta":null}]'::jsonb
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'se acepto un instante normalizable no canonico';
    END IF;

    IF vec_autorizacion.publicar_motivos_autorizacion_v2(
        'evento_00000000000000000000000000000001', 1, repeat('1', 64),
        'motivos_autorizacion', 1, repeat('a', 64),
        '2026-01-01T00:00:00.000000Z'::timestamptz,
        '[{"clave":"motivo_11111111111111111111111111111111","vigente_desde":"2026-01-01T00:00:00.000000Z","vigente_hasta":null}]'::jsonb
    ) IS NOT TRUE THEN
        RAISE EXCEPTION 'fallo la publicacion valida';
    END IF;

    -- Replay exacto: confirma idempotencia sin crear evento ni avance nuevos.
    IF vec_autorizacion.publicar_motivos_autorizacion_v2(
        'evento_00000000000000000000000000000001', 1, repeat('1', 64),
        'motivos_autorizacion', 1, repeat('a', 64),
        '2026-01-01T00:00:00.000000Z'::timestamptz,
        '[{"vigente_hasta":null,"vigente_desde":"2026-01-01T00:00:00.000000Z","clave":"motivo_11111111111111111111111111111111"}]'::jsonb
    ) IS NOT TRUE THEN
        RAISE EXCEPTION 'el replay exacto no fue idempotente';
    END IF;

    IF vec_autorizacion.publicar_motivos_autorizacion_v2(
        'evento_00000000000000000000000000000001', 1, repeat('1', 64),
        'motivos_autorizacion', 1, repeat('a', 64),
        '2026-01-01T00:00:00.000000Z'::timestamptz,
        '[{"clave":"motivo_22222222222222222222222222222222","vigente_desde":"2026-01-01T00:00:00.000000Z","vigente_hasta":null}]'::jsonb
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'un replay alterado fue aceptado';
    END IF;

    IF vec_autorizacion.publicar_motivos_autorizacion_v2(
        'evento_99999999999999999999999999999999', 1, repeat('9', 64),
        'motivos_autorizacion', 1, repeat('a', 64),
        '2026-01-01T00:00:00.000000Z'::timestamptz,
        '[{"clave":"motivo_11111111111111111111111111111111","vigente_desde":"2026-01-01T00:00:00.000000Z","vigente_hasta":null}]'::jsonb
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'se reutilizo una secuencia con otro evento';
    END IF;

    IF vec_autorizacion.publicar_motivos_autorizacion_v2(
        'evento_00000000000000000000000000000003', 3, repeat('3', 64),
        'motivos_autorizacion', 2, repeat('b', 64),
        '2026-01-02T00:00:00.000000Z'::timestamptz,
        '[{"clave":"motivo_11111111111111111111111111111111","vigente_desde":"2026-01-01T00:00:00.000000Z","vigente_hasta":null}]'::jsonb
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'se acepto un salto del checkpoint';
    END IF;

    BEGIN
        PERFORM vec_autorizacion.resolver_motivo_autorizacion_v2_actual(
            'motivos_autorizacion', 1, repeat('a', 64),
            'motivo_11111111111111111111111111111111'
        );
        RAISE EXCEPTION 'el proyector pudo usar el resolver';
    EXCEPTION WHEN insufficient_privilege THEN
        NULL;
    END;
END
$prueba$;

RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;

DO $prueba$
DECLARE
    entradas_grandes jsonb;
    entradas_sobredimensionadas jsonb;
BEGIN
    SELECT jsonb_agg(jsonb_build_object(
               'clave', 'motivo_' || lpad(to_hex(numero), 32, '0'),
               'vigente_desde', '2026-01-01T00:00:00.000000Z',
               'vigente_hasta', NULL
           ) ORDER BY numero)
      INTO entradas_grandes
      FROM generate_series(1, 10000) AS numero;
    IF vec_autorizacion.motivo_v2_entradas_validas(entradas_grandes) IS NOT TRUE
       OR vec_autorizacion.motivo_v2_entradas_validas(
            jsonb_build_array(entradas_grandes -> 0, entradas_grandes -> 0)
       ) IS NOT FALSE
       OR vec_autorizacion.motivo_v2_entradas_validas(
            entradas_grandes || jsonb_build_array(entradas_grandes -> 0)
       ) IS NOT FALSE THEN
        RAISE EXCEPTION 'limite grande o duplicados del catalogo incorrectos';
    END IF;
    entradas_sobredimensionadas := jsonb_build_array(jsonb_build_object(
        'clave', 'motivo_11111111111111111111111111111111',
        'vigente_desde', '2026-01-01T00:00:00.000000Z',
        'vigente_hasta', repeat('x', 17 * 1024 * 1024)
    ));
    IF pg_column_size(entradas_sobredimensionadas) <= 16 * 1024 * 1024
       OR vec_autorizacion.motivo_v2_entradas_validas(
            entradas_sobredimensionadas
       ) IS NOT FALSE THEN
        RAISE EXCEPTION 'techo de 16 MiB no aplicado';
    END IF;

    IF (SELECT ultima_secuencia
          FROM vec_autorizacion.motivo_v2_checkpoint_origen
         WHERE control_id) <> 1
       OR (SELECT count(*) FROM vec_autorizacion.motivo_v2_evento_origen) <> 1
       OR (SELECT count(*) FROM vec_autorizacion.motivo_v2_entrada) <> 1 THEN
        RAISE EXCEPTION 'el replay o un rechazo mutaron la proyeccion';
    END IF;
END
$prueba$;

SET LOCAL ROLE vec_autorizacion_motivos_evaluador;

DO $prueba$
BEGIN
    BEGIN
        PERFORM count(*) FROM vec_autorizacion.motivo_v2_entrada;
        RAISE EXCEPTION 'el evaluador pudo leer tablas';
    EXCEPTION WHEN insufficient_privilege THEN
        NULL;
    END;

    IF vec_autorizacion.resolver_motivo_autorizacion_v2_historico(
        'motivos_autorizacion', 1, repeat('a', 64),
        'motivo_11111111111111111111111111111111',
        '2026-01-15T00:00:00.000000Z'::timestamptz
    ) IS NOT TRUE THEN
        RAISE EXCEPTION 'no se resolvio una referencia historica exacta';
    END IF;
    IF vec_autorizacion.resolver_motivo_autorizacion_v2_historico(
        'motivos_autorizacion', 1, repeat('b', 64),
        'motivo_11111111111111111111111111111111',
        '2026-01-15T00:00:00.000000Z'::timestamptz
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'se resolvio una huella fabricada';
    END IF;
    IF vec_autorizacion.resolver_motivo_autorizacion_v2_historico(
        'Motivos RRHH', 1, repeat('a', 64),
        'motivo_11111111111111111111111111111111',
        '2026-01-15T00:00:00.000000Z'::timestamptz
    ) IS NOT FALSE OR vec_autorizacion.resolver_motivo_autorizacion_v2_historico(
        'motivos_autorizacion', 1, repeat('0', 64),
        'motivo_11111111111111111111111111111111',
        '2026-01-15T00:00:00.000000Z'::timestamptz
    ) IS NOT FALSE OR vec_autorizacion.resolver_motivo_autorizacion_v2_historico(
        'motivos_autorizacion', 1, repeat('a', 64),
        'dni_12345678z', '2026-01-15T00:00:00.000000Z'::timestamptz
    ) IS NOT FALSE OR vec_autorizacion.resolver_motivo_autorizacion_v2_historico(
        'motivos_autorizacion', 1, repeat('a', 64),
        'motivo_11111111111111111111111111111111', 'infinity'::timestamptz
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'el resolver historico acepto coordenadas malformadas';
    END IF;

    BEGIN
        PERFORM vec_autorizacion.resolver_motivo_autorizacion_v2_actual(
            'motivos_autorizacion', 1, repeat('a', 64),
            'motivo_11111111111111111111111111111111'
        );
        RAISE EXCEPTION 'el evaluador pudo usar la barrera actual privada';
    EXCEPTION WHEN insufficient_privilege THEN
        NULL;
    END;

    BEGIN
        PERFORM vec_autorizacion.retirar_motivos_autorizacion_v2(
            'evento_00000000000000000000000000000002', 2, repeat('2', 64),
            'motivos_autorizacion', 1, repeat('a', 64), repeat('c', 64),
            '2026-02-01T00:00:00.000000Z'::timestamptz
        );
        RAISE EXCEPTION 'el evaluador pudo retirar';
    EXCEPTION WHEN insufficient_privilege THEN
        NULL;
    END;
END
$prueba$;

RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;

DO $prueba$
BEGIN
    IF vec_autorizacion.resolver_motivo_autorizacion_v2_historico(
        'motivos_autorizacion', 1, repeat('a', 64),
        'motivo_11111111111111111111111111111111',
        '2026-01-15T00:00:00.000000Z'::timestamptz
    ) IS NOT TRUE THEN
        RAISE EXCEPTION 'la consulta historica anterior a retirada fallo';
    END IF;
    IF vec_autorizacion.resolver_motivo_autorizacion_v2_actual(
        'motivos_autorizacion', 1, repeat('a', 64),
        'motivo_11111111111111111111111111111111'
    ) IS NOT TRUE THEN
        RAISE EXCEPTION 'el helper actual privado no resolvio como propietario';
    END IF;
    IF vec_autorizacion.resolver_motivo_autorizacion_v2_actual(
        'Motivos RRHH', 1, repeat('a', 64),
        'motivo_11111111111111111111111111111111'
    ) IS NOT FALSE OR vec_autorizacion.resolver_motivo_autorizacion_v2_actual(
        'motivos_autorizacion', 1, repeat('0', 64),
        'motivo_11111111111111111111111111111111'
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'el helper actual acepto coordenadas malformadas';
    END IF;
END
$prueba$;

RESET ROLE;
SET LOCAL ROLE vec_autorizacion_motivos_proyector;

DO $prueba$
BEGIN
    IF vec_autorizacion.retirar_motivos_autorizacion_v2(
        'evento_00000000000000000000000000000002', 2, repeat('2', 64),
        'motivos_autorizacion', 1, repeat('a', 64), repeat('c', 64),
        '-infinity'::timestamptz
    ) IS NOT FALSE OR vec_autorizacion.retirar_motivos_autorizacion_v2(
        'evento_00000000000000000000000000000002', 2, repeat('2', 64),
        'motivos_autorizacion', 1, repeat('a', 64), repeat('c', 64),
        '10000-01-01T00:00:00Z'::timestamptz
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'se acepto una retirada no finita o fuera de rango';
    END IF;
    IF vec_autorizacion.retirar_motivos_autorizacion_v2(
        'evento_00000000000000000000000000000002', 2, repeat('2', 64),
        'motivos_autorizacion', 1, repeat('a', 64), repeat('c', 64),
        '2026-02-01T00:00:00.000000Z'::timestamptz
    ) IS NOT TRUE THEN
        RAISE EXCEPTION 'fallo la retirada valida';
    END IF;
    IF vec_autorizacion.retirar_motivos_autorizacion_v2(
        'evento_00000000000000000000000000000002', 2, repeat('2', 64),
        'motivos_autorizacion', 1, repeat('a', 64), repeat('c', 64),
        '2026-02-01T00:00:00.000000Z'::timestamptz
    ) IS NOT TRUE THEN
        RAISE EXCEPTION 'el replay exacto de retirada no fue idempotente';
    END IF;
    IF vec_autorizacion.retirar_motivos_autorizacion_v2(
        'evento_00000000000000000000000000000002', 2, repeat('2', 64),
        'motivos_autorizacion', 1, repeat('a', 64), repeat('d', 64),
        '2026-02-01T00:00:00.000000Z'::timestamptz
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'un replay alterado de retirada fue aceptado';
    END IF;
    IF vec_autorizacion.retirar_motivos_autorizacion_v2(
        'evento_00000000000000000000000000000002', 2, repeat('2', 64),
        'motivos_autorizacion', 1, repeat('f', 64), repeat('c', 64),
        '2026-02-01T00:00:00.000000Z'::timestamptz
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'un replay con otra huella publicada fue aceptado';
    END IF;
END
$prueba$;

RESET ROLE;
SET LOCAL ROLE vec_autorizacion_motivos_evaluador;

DO $prueba$
BEGIN
    IF vec_autorizacion.resolver_motivo_autorizacion_v2_historico(
        'motivos_autorizacion', 1, repeat('a', 64),
        'motivo_11111111111111111111111111111111',
        '2026-01-15T00:00:00.000000Z'::timestamptz
    ) IS NOT TRUE OR vec_autorizacion.resolver_motivo_autorizacion_v2_historico(
        'motivos_autorizacion', 1, repeat('a', 64),
        'motivo_11111111111111111111111111111111',
        '2026-02-15T00:00:00.000000Z'::timestamptz
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'la evaluacion historica de retirada es incorrecta';
    END IF;
    BEGIN
        PERFORM vec_autorizacion.resolver_motivo_autorizacion_v2_actual(
            'motivos_autorizacion', 1, repeat('a', 64),
            'motivo_11111111111111111111111111111111'
        );
        RAISE EXCEPTION 'el evaluador pudo usar la barrera actual privada';
    EXCEPTION WHEN insufficient_privilege THEN
        NULL;
    END;
END
$prueba$;

RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;

DO $prueba$
BEGIN
    IF vec_autorizacion.resolver_motivo_autorizacion_v2_actual(
        'motivos_autorizacion', 1, repeat('a', 64),
        'motivo_11111111111111111111111111111111'
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'una retirada siguio vigente en la barrera actual';
    END IF;
    IF (SELECT catalogo_huella_publicada_sha256
          FROM vec_autorizacion.motivo_v2_catalogo_publicado
         WHERE catalogo_id = 'motivos_autorizacion'
           AND catalogo_version = 1) <> repeat('a', 64) THEN
        RAISE EXCEPTION 'la retirada altero la huella publicada';
    END IF;
    IF (SELECT ultima_secuencia
          FROM vec_autorizacion.motivo_v2_checkpoint_origen
         WHERE control_id) <> 2
       OR (SELECT count(*) FROM vec_autorizacion.motivo_v2_evento_origen) <> 2
       OR (SELECT count(*) FROM vec_autorizacion.motivo_v2_retirada) <> 1 THEN
        RAISE EXCEPTION 'replay o retirada dejaron un estado inesperado';
    END IF;

    BEGIN
        UPDATE vec_autorizacion.motivo_v2_catalogo_publicado
           SET catalogo_huella_publicada_sha256 = repeat('f', 64)
         WHERE catalogo_id = 'motivos_autorizacion'
           AND catalogo_version = 1;
        RAISE EXCEPTION 'se pudo reescribir la cabecera publicada';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN
        NULL;
    END;
END
$prueba$;

ROLLBACK;
