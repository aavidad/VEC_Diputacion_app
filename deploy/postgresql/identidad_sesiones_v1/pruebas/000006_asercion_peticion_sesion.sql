-- Ejecutar únicamente tras las migraciones V1 000001..000006 en PostgreSQL
-- efímero. Usa referencias y huellas sintéticas; no instala ni revierte nada.
BEGIN;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prueba$
DECLARE
    cuenta text;
    autenticacion text;
    sesion text;
    control text;
    revision text;
    sesion_caducada text;
    sesion_reemitida text;
    filas integer;
    cuerpo text := repeat('c', 64);
BEGIN
    SELECT cuenta_ref INTO cuenta
      FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(
        'opr_' || repeat('a', 24), 'vec.identidad.hmac-sha256.v1',
        'idh_' || repeat('a', 24), 'clave-prueba-000006', 1,
        decode(repeat('2', 64), 'hex'), decode(repeat('1', 64), 'hex'),
        false, NULL
      );

    SELECT autenticacion_ref, sesion_ref, control_sesion_ref,
           control_sesion_revision_texto
      INTO autenticacion, sesion, control, revision
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
        'opr_' || repeat('b', 24), 'vec.identidad.hmac-sha256.v1',
        'idh_' || repeat('a', 24), 'clave-prueba-000006', 1,
        decode(repeat('3', 64), 'hex'), decode(repeat('4', 64), 'hex'),
        decode(repeat('1', 64), 'hex'), decode(repeat('2', 64), 'hex'),
        NULL, false, 'externa_personal', 'dnie', 'sustancial',
        repeat('d', 64), clock_timestamp() - interval '2 seconds',
        clock_timestamp() - interval '1 second',
        clock_timestamp() + interval '4 minutes',
        'pga_' || repeat('a', 24), repeat('e', 64)
      );
    IF autenticacion IS NULL OR sesion IS NULL OR control IS NULL OR revision <> '1' THEN
        RAISE EXCEPTION 'fixture 000006 de sesión no válida';
    END IF;

    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.consumir_asercion_peticion_sesion_v1(
        'vec.identidad.hmac-sha256.v1', 'idh_' || repeat('a', 24),
        'clave-prueba-000006', 1, decode(repeat('4', 64), 'hex'),
        repeat('f', 64), 'canal-prueba-000006', 'externa_personal', 'POST',
        '/area-personal/prueba', cuerpo, clock_timestamp() - interval '1 second',
        clock_timestamp() + interval '1 minute'
      );
    IF filas <> 1 THEN
        RAISE EXCEPTION 'consumo inicial de petición fue rechazado';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM vec_identidad_sesiones_v1.consumo_asercion_peticion_sesion_v1
         WHERE nonce_sha256 = repeat('f', 64)
           AND sesion_ref = sesion
           AND autenticacion_ref = autenticacion
           AND metodo = 'POST'
           AND destino = '/area-personal/prueba'
           AND cuerpo_sha256 = cuerpo
    ) THEN
        RAISE EXCEPTION 'evidencia de petición incompleta';
    END IF;

    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.consumir_asercion_peticion_sesion_v1(
        'vec.identidad.hmac-sha256.v1', 'idh_' || repeat('a', 24),
        'clave-prueba-000006', 1, decode(repeat('4', 64), 'hex'),
        repeat('f', 64), 'canal-prueba-000006', 'externa_personal', 'POST',
        '/area-personal/prueba', cuerpo, clock_timestamp() - interval '1 second',
        clock_timestamp() + interval '1 minute'
      );
    IF filas <> 0 THEN
        RAISE EXCEPTION 'nonce repetido aceptado';
    END IF;

    -- La aserción puede adelantarse hasta cinco minutos por la tolerancia de
    -- reloj de la frontera, pero no más. Ambas siguen dentro de la sesión.
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.consumir_asercion_peticion_sesion_v1(
        'vec.identidad.hmac-sha256.v1', 'idh_' || repeat('a', 24),
        'clave-prueba-000006', 1, decode(repeat('4', 64), 'hex'),
        repeat('7', 64), 'canal-prueba-000006', 'externa_personal', 'POST',
        '/area-personal/prueba', cuerpo, clock_timestamp() + interval '2 minutes',
        clock_timestamp() + interval '3 minutes'
      );
    IF filas <> 1 THEN
        RAISE EXCEPTION 'skew admitido por la frontera fue rechazado';
    END IF;
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.consumir_asercion_peticion_sesion_v1(
        'vec.identidad.hmac-sha256.v1', 'idh_' || repeat('a', 24),
        'clave-prueba-000006', 1, decode(repeat('4', 64), 'hex'),
        repeat('6', 64), 'canal-prueba-000006', 'externa_personal', 'POST',
        '/area-personal/prueba', cuerpo, clock_timestamp() + interval '5 minutes 1 second',
        clock_timestamp() + interval '5 minutes 2 seconds'
      );
    IF filas <> 0 THEN
        RAISE EXCEPTION 'skew fuera de tolerancia aceptado';
    END IF;

    -- Una sesion externa puede reemitirse con las mismas coordenadas HMAC
    -- cuando la anterior ya caduco. El control historico conserva estado
    -- "activa", pero no debe volver ambigua la sesion nueva y vigente.
    SELECT sesion_ref INTO sesion_caducada
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
        'opr_' || repeat('d', 24), 'vec.identidad.hmac-sha256.v1',
        'idh_' || repeat('a', 24), 'clave-prueba-000006', 1,
        decode(repeat('a', 64), 'hex'), decode(repeat('5', 64), 'hex'),
        decode(repeat('1', 64), 'hex'), decode(repeat('2', 64), 'hex'),
        NULL, false, 'externa_personal', 'dnie', 'sustancial',
        repeat('a', 64), clock_timestamp() - interval '2 seconds',
        clock_timestamp() - interval '1 second',
        clock_timestamp() + interval '500 milliseconds',
        'pga_' || repeat('a', 24), repeat('b', 64)
      );
    IF sesion_caducada IS NULL THEN
        RAISE EXCEPTION 'no se creo la sesion breve para reemision';
    END IF;
    PERFORM pg_sleep(0.7);
    SELECT sesion_ref INTO sesion_reemitida
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
        'opr_' || repeat('e', 24), 'vec.identidad.hmac-sha256.v1',
        'idh_' || repeat('a', 24), 'clave-prueba-000006', 1,
        decode(repeat('b', 64), 'hex'), decode(repeat('5', 64), 'hex'),
        decode(repeat('1', 64), 'hex'), decode(repeat('2', 64), 'hex'),
        NULL, false, 'externa_personal', 'dnie', 'sustancial',
        repeat('b', 64), clock_timestamp() - interval '2 seconds',
        clock_timestamp() - interval '1 second',
        clock_timestamp() + interval '4 minutes',
        'pga_' || repeat('a', 24), repeat('b', 64)
      );
    IF sesion_reemitida IS NULL OR sesion_reemitida = sesion_caducada THEN
        RAISE EXCEPTION 'no se reemitio la sesion tras caducar';
    END IF;
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.consumir_asercion_peticion_sesion_v1(
        'vec.identidad.hmac-sha256.v1', 'idh_' || repeat('a', 24),
        'clave-prueba-000006', 1, decode(repeat('5', 64), 'hex'),
        repeat('4', 64), 'canal-prueba-000006', 'externa_personal', 'GET',
        '/area-personal/reemitida', cuerpo, clock_timestamp() - interval '1 second',
        clock_timestamp() + interval '1 minute'
      );
    IF filas <> 1 OR NOT EXISTS (
        SELECT 1
          FROM vec_identidad_sesiones_v1.consumo_asercion_peticion_sesion_v1
         WHERE nonce_sha256 = repeat('4', 64)
           AND sesion_ref = sesion_reemitida
    ) THEN
        RAISE EXCEPTION 'la sesion reemitida quedo ambigua por su historia caducada';
    END IF;

    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.consumir_asercion_peticion_sesion_v1(
        'vec.identidad.hmac-sha256.v1', 'idh_' || repeat('a', 24),
        'clave-prueba-000006', 1, decode(repeat('4', 64), 'hex'),
        repeat('9', 64), 'canal-prueba-000006', 'externa_personal', 'TRACE',
        '/area-personal/prueba', cuerpo, clock_timestamp() - interval '1 second',
        clock_timestamp() + interval '1 minute'
      );
    IF filas <> 0 THEN
        RAISE EXCEPTION 'método fuera del contrato aceptado';
    END IF;

    PERFORM vec_identidad_sesiones_v1.revocar_sesion_v1(
        sesion, control, revision, 'opr_' || repeat('c', 24)
    );
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.consumir_asercion_peticion_sesion_v1(
        'vec.identidad.hmac-sha256.v1', 'idh_' || repeat('a', 24),
        'clave-prueba-000006', 1, decode(repeat('4', 64), 'hex'),
        repeat('8', 64), 'canal-prueba-000006', 'externa_personal', 'POST',
        '/area-personal/prueba', cuerpo, clock_timestamp() - interval '1 second',
        clock_timestamp() + interval '1 minute'
      );
    IF filas <> 0 THEN
        RAISE EXCEPTION 'sesión revocada aceptó una petición';
    END IF;
END
$prueba$;
ROLLBACK;
