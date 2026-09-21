-- B9: la bolsa constituida sustituye a la anterior vigente de su categoría.
-- Se ejecuta como superusuario sobre una base con roles_up, 000007, 000008 y 000014.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

-- Dos actas de la misma categoría, confirmadas en instantes distintos, y una
-- tercera de otra categoría que no debe verse afectada.
DO $constitucion$
DECLARE
    v_recibo_a jsonb;
    v_recibo_b jsonb;
    v_fila record;
BEGIN
    v_recibo_a := vec_bolsa_llamamientos.constituir_bolsa_v1(
        'acta:b9:a', 'per_actorb9', 'categoria:b9:x', 'bolsa:b9:a', 1, '{"bolsa":"a"}'::bytea, '2026-01-10T08:00:00Z',
        'instantanea:b9:a', 1, '{"instantanea":"a"}'::bytea, '2026-01-09T08:00:00Z', '2026-01-09T09:00:00Z',
        '[{"orden":1,"participacion_ref":"participacion:b9:a1","fila_numero":2}]'::jsonb, '2026-01-10T08:00:00Z');
    PERFORM vec_bolsa_llamamientos.constituir_bolsa_v1(
        'acta:b9:c', 'per_actorb9', 'categoria:b9:y', 'bolsa:b9:c', 1, '{"bolsa":"c"}'::bytea, '2026-01-11T08:00:00Z',
        'instantanea:b9:c', 1, '{"instantanea":"c"}'::bytea, '2026-01-10T08:00:00Z', '2026-01-10T09:00:00Z',
        '[{"orden":1,"participacion_ref":"participacion:b9:c1","fila_numero":2}]'::jsonb, '2026-01-11T08:00:00Z');
    v_recibo_b := vec_bolsa_llamamientos.constituir_bolsa_v1(
        'acta:b9:b', 'per_actorb9', 'categoria:b9:x', 'bolsa:b9:b', 1, '{"bolsa":"b"}'::bytea, '2026-03-01T08:00:00Z',
        'instantanea:b9:b', 1, '{"instantanea":"b"}'::bytea, '2026-02-28T08:00:00Z', '2026-02-28T09:00:00Z',
        '[{"orden":1,"participacion_ref":"participacion:b9:b1","fila_numero":2}]'::jsonb, '2026-03-01T08:00:00Z');
    IF v_recibo_a -> 'sustituye_a' <> '[]'::jsonb THEN
        RAISE EXCEPTION 'la primera constitución de la categoría no sustituye a nadie: %', v_recibo_a;
    END IF;
    IF v_recibo_b -> 'sustituye_a' <> '[{"bolsa_ref": "bolsa:b9:a", "version_bolsa": 1}]'::jsonb THEN
        RAISE EXCEPTION 'la segunda constitución debe sustituir a la primera: %', v_recibo_b;
    END IF;
    -- El cuadro RRHH solo ve la última de la categoría, vigente.
    SELECT * INTO STRICT v_fila FROM vec_bolsa_llamamientos.listar_constituciones_v1() WHERE categoria_ref = 'categoria:b9:x';
    IF v_fila.bolsa_ref <> 'bolsa:b9:b' OR v_fila.estado <> 'vigente' OR v_fila.vigente_hasta IS NOT NULL THEN
        RAISE EXCEPTION 'la categoría x debe listar la bolsa b vigente: % % %', v_fila.bolsa_ref, v_fila.estado, v_fila.vigente_hasta;
    END IF;
    SELECT * INTO STRICT v_fila FROM vec_bolsa_llamamientos.listar_constituciones_v1() WHERE categoria_ref = 'categoria:b9:y';
    IF v_fila.bolsa_ref <> 'bolsa:b9:c' OR v_fila.estado <> 'vigente' THEN
        RAISE EXCEPTION 'la categoría y no debe verse afectada: % %', v_fila.bolsa_ref, v_fila.estado;
    END IF;
END
$constitucion$;

-- Replay de ambas actas: recibos reutilizados, sin sustitución nueva; el de b
-- conserva a quién sustituyó.
DO $replay$
DECLARE
    v_recibo_a2 jsonb;
    v_recibo_b2 jsonb;
BEGIN
    v_recibo_a2 := vec_bolsa_llamamientos.constituir_bolsa_v1(
        'acta:b9:a', 'per_actorb9', 'categoria:b9:x', 'bolsa:b9:a', 1, '{"bolsa":"a"}'::bytea, '2026-01-10T08:00:00Z',
        'instantanea:b9:a', 1, '{"instantanea":"a"}'::bytea, '2026-01-09T08:00:00Z', '2026-01-09T09:00:00Z',
        '[{"orden":1,"participacion_ref":"participacion:b9:a1","fila_numero":2}]'::jsonb, '2026-01-10T08:00:00Z');
    v_recibo_b2 := vec_bolsa_llamamientos.constituir_bolsa_v1(
        'acta:b9:b', 'per_actorb9', 'categoria:b9:x', 'bolsa:b9:b', 1, '{"bolsa":"b"}'::bytea, '2026-03-01T08:00:00Z',
        'instantanea:b9:b', 1, '{"instantanea":"b"}'::bytea, '2026-02-28T08:00:00Z', '2026-02-28T09:00:00Z',
        '[{"orden":1,"participacion_ref":"participacion:b9:b1","fila_numero":2}]'::jsonb, '2026-03-01T08:00:00Z');
    IF (v_recibo_a2 ->> 'reutilizada')::boolean IS NOT TRUE OR v_recibo_a2 -> 'sustituye_a' <> '[]'::jsonb THEN
        RAISE EXCEPTION 'replay de a: %', v_recibo_a2;
    END IF;
    IF (v_recibo_b2 ->> 'reutilizada')::boolean IS NOT TRUE
       OR v_recibo_b2 -> 'sustituye_a' <> '[{"bolsa_ref": "bolsa:b9:a", "version_bolsa": 1}]'::jsonb THEN
        RAISE EXCEPTION 'replay de b: %', v_recibo_b2;
    END IF;
END
$replay$;

-- La tabla de sustituciones no la lee el ejecutor.
DO $acl$
BEGIN
    BEGIN
        PERFORM 1 FROM vec_bolsa_llamamientos.sustitucion_bolsa;
        RAISE EXCEPTION 'el ejecutor no debe leer sustitucion_bolsa';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
END
$acl$;

RESET ROLE;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
-- Vínculo de un candidato con la bolsa sustituida (solo el propietario registra
-- vínculos): su participación se lee extinguida desde la nueva constitución.
DO $candidato$
DECLARE v_fila record;
BEGIN
    PERFORM vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(
        'acta:b9:a',
        '[{"candidato_ref":"can_b9_candidato_000000000001","participacion_ref":"participacion:b9:a1"}]'::jsonb,
        '2026-01-10T08:00:00Z');
    SELECT * INTO STRICT v_fila FROM vec_bolsa_llamamientos.listar_participaciones_candidato_v1('can_b9_candidato_000000000001');
    IF v_fila.estado <> 'extinguida' OR v_fila.vigente_hasta IS DISTINCT FROM '2026-03-01T08:00:00Z'::timestamptz THEN
        RAISE EXCEPTION 'la participación en la bolsa sustituida debe leerse extinguida desde la nueva constitución: % %', v_fila.estado, v_fila.vigente_hasta;
    END IF;
END
$candidato$;

-- La tabla de sustituciones es inmutable y contiene exactamente una fila.
DO $inmutable$
BEGIN
    BEGIN
        DELETE FROM vec_bolsa_llamamientos.sustitucion_bolsa WHERE bolsa_ref_sustituida = 'bolsa:b9:a';
        RAISE EXCEPTION 'sustitucion_bolsa debe ser inmutable';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
    END;
    IF (SELECT count(*) FROM vec_bolsa_llamamientos.sustitucion_bolsa) <> 1 THEN
        RAISE EXCEPTION 'debe haber exactamente una sustitución registrada';
    END IF;
END
$inmutable$;
ROLLBACK;
