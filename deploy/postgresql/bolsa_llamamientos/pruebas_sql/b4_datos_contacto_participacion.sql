-- B4: datos de contacto cifrados de una participación.
-- Se ejecuta como superusuario sobre una base con roles_up, 000007, 000008, 000011,
-- 000012 y 000016. La escritura con concesión V3 se acredita en la réplica de
-- despliegue; aquí se comprueban estructura, ACL, inmutabilidad, lecturas,
-- versiones consecutivas y replay sobre filas insertadas por el propietario.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
-- Participación de referencia.
DO $constitucion$
BEGIN
    PERFORM vec_bolsa_llamamientos.constituir_bolsa_v1(
        'acta:b4:a', 'per_actorb4', 'categoria:b4', 'bolsa:b4:a', 1, '{"bolsa":"a"}'::bytea, '2026-01-10T08:00:00Z',
        'instantanea:b4:a', 1, '{"instantanea":"a"}'::bytea, '2026-01-09T08:00:00Z', '2026-01-09T09:00:00Z',
        '[{"orden":1,"participacion_ref":"participacion:b4:a1","fila_numero":2}]'::jsonb, '2026-01-10T08:00:00Z');
END
$constitucion$;
-- Sin datos registrados, la lectura no devuelve filas y la tabla no la ve el ejecutor.
DO $sin_datos$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.leer_datos_contacto_participacion_v1('participacion:b4:a1')) THEN
        RAISE EXCEPTION 'no debe haber datos de contacto todavía';
    END IF;
    BEGIN
        PERFORM 1 FROM vec_bolsa_llamamientos.datos_contacto_participacion;
        RAISE EXCEPTION 'el ejecutor no debe leer datos_contacto_participacion';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
END
$sin_datos$;
-- La función de alta exige participación de la bolsa y parámetros válidos antes de tocar V3.
DO $validacion$
BEGIN
    BEGIN
        PERFORM vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1('bolsa:b4:a', 'participacion:b4:ajena', 1, 'clave:prueba', decode('000000000000000000000000','hex'), decode('00000000000000000000000000000000','hex'), 'Alta', 'per_actorb4', now(), 'clave-1', 'recibo-1', '\x00'::bytea, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea, 1, 1, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea);
        RAISE EXCEPTION 'una participación ajena debe rechazarse';
    EXCEPTION WHEN foreign_key_violation THEN NULL;
    END;
    BEGIN
        PERFORM vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1('bolsa:b4:a', 'participacion:b4:a1', 1, 'clave:prueba', decode('0000','hex'), decode('00000000000000000000000000000000','hex'), 'Alta', 'per_actorb4', now(), 'clave-1', 'recibo-1', '\x00'::bytea, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea, 1, 1, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea);
        RAISE EXCEPTION 'un nonce corto debe rechazarse';
    EXCEPTION WHEN invalid_parameter_value THEN NULL;
    END;
END
$validacion$;
RESET ROLE;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
-- El propietario registra dos versiones (la primera vía réplica sería con V3); las lecturas devuelven la última.
INSERT INTO vec_bolsa_llamamientos.datos_contacto_participacion (participacion_ref, version, clave_ref, nonce, cifrado, motivo, actor, registrada_en, clave_idempotencia, recibo_ref)
VALUES ('participacion:b4:a1', 1, 'clave:kms:desarrollo:datos-contacto-participacion:v1', decode('000102030405060708090a0b','hex'), decode('00000000000000000000000000000000ff','hex'), 'Alta comunicada', 'per_actorb4', '2026-02-01T10:00:00Z', 'b4-alta-0001', 'recibo:datos-contacto:0001'),
       ('participacion:b4:a1', 2, 'clave:kms:desarrollo:datos-contacto-participacion:v1', decode('0b0a09080706050403020100','hex'), decode('11111111111111111111111111111111ff','hex'), 'Cambio de teléfono', 'per_actorb4', '2026-02-02T10:00:00Z', 'b4-alta-0002', 'recibo:datos-contacto:0002');
DO $lecturas$
DECLARE v record;
BEGIN
    SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.leer_datos_contacto_participacion_v1('participacion:b4:a1');
    IF v.version <> 2 OR v.recibo_ref <> 'recibo:datos-contacto:0002' OR v.motivo <> 'Cambio de teléfono' THEN
        RAISE EXCEPTION 'la lectura debe devolver la última versión: %', v;
    END IF;
    SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.recuperar_datos_contacto_participacion_v1('participacion:b4:a1', 'b4-alta-0001');
    IF v.version <> 1 OR v.recibo_ref <> 'recibo:datos-contacto:0001' THEN
        RAISE EXCEPTION 'la recuperación por clave debe devolver la versión 1: %', v;
    END IF;
    -- Versión no consecutiva y clave repetida.
    BEGIN
        INSERT INTO vec_bolsa_llamamientos.datos_contacto_participacion (participacion_ref, version, clave_ref, nonce, cifrado, motivo, actor, registrada_en, clave_idempotencia, recibo_ref)
        VALUES ('participacion:b4:a1', 3, 'clave:kms:desarrollo:datos-contacto-participacion:v1', decode('000102030405060708090a0b','hex'), decode('00000000000000000000000000000000ff','hex'), 'Repetida', 'per_actorb4', '2026-02-03T10:00:00Z', 'b4-alta-0001', 'recibo:datos-contacto:0003');
        RAISE EXCEPTION 'una clave de idempotencia repetida debe rechazarse';
    EXCEPTION WHEN unique_violation THEN NULL;
    END;
    BEGIN
        UPDATE vec_bolsa_llamamientos.datos_contacto_participacion SET motivo = 'x' WHERE version = 1;
        RAISE EXCEPTION 'la tabla debe ser inmutable';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
    END;
    BEGIN
        DELETE FROM vec_bolsa_llamamientos.datos_contacto_participacion WHERE version = 1;
        RAISE EXCEPTION 'la tabla debe ser inmutable';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
    END;
END
$lecturas$;
-- La bitácora de frontera compartida admite los intentos de B4.
DO $bitacora$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass AND conname='bitacora_intento_borrador_llamamiento_accion_check' AND pg_get_constraintdef(oid,true) LIKE '%registrar_datos_contacto%')
       OR NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass AND conname='bitacora_intento_borrador_llamamiento_ruta_clase_check' AND pg_get_constraintdef(oid,true) LIKE '%datos_contacto%') THEN
        RAISE EXCEPTION 'la bitácora de frontera no admite B4';
    END IF;
END
$bitacora$;
ROLLBACK;
