\set ON_ERROR_STOP on
-- Ejecutar sólo sobre PostgreSQL 18 desechable con CTX18 instalada.
-- Todas las referencias son sintéticas y se conservan sólo hasta destruir el clon.
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog, pg_temp;
INSERT INTO vec_contexto_actor_v1.procedencias
    (procedencia_ref, procedencia_version, procedencia_huella_sha256, procedencia_autoridad)
VALUES ('prc_ctx18_sintetica_00000000000001', 1, repeat('a', 64),
        'autoridad_maestra_acreditada'),
       ('prc_ctx18_no_autoritativa_00000001', 1, repeat('b', 64),
        'no_autoritativa');
INSERT INTO vec_contexto_actor_v1.persona_versiones
    (persona_ref, version, procedencia_ref, procedencia_version,
     procedencia_huella_sha256, procedencia_autoridad, estado,
     vigente_desde, vigente_hasta)
VALUES
    ('per_ctx18_positiva_0000000000000001', 1,
     'prc_ctx18_sintetica_00000000000001', 1, repeat('a', 64),
     'autoridad_maestra_acreditada', 'activo',
     clock_timestamp() - interval '1 hour', clock_timestamp() + interval '2 hours'),
    ('per_ctx18_ambigua_0000000000000001', 1,
     'prc_ctx18_sintetica_00000000000001', 1, repeat('a', 64),
     'autoridad_maestra_acreditada', 'activo',
     clock_timestamp() - interval '1 hour', clock_timestamp() + interval '2 hours'),
    ('per_ctx18_revocada_000000000000001', 1,
     'prc_ctx18_sintetica_00000000000001', 1, repeat('a', 64),
     'autoridad_maestra_acreditada', 'revocado',
     clock_timestamp() - interval '1 hour', clock_timestamp() + interval '2 hours'),
    ('per_ctx18_noaut_000000000000000001', 1,
     'prc_ctx18_no_autoritativa_00000001', 1, repeat('b', 64),
     'no_autoritativa', 'activo',
     clock_timestamp() - interval '1 hour', clock_timestamp() + interval '2 hours'),
    ('per_ctx18_caducada_00000000000001', 1,
     'prc_ctx18_sintetica_00000000000001', 1, repeat('a', 64),
     'autoridad_maestra_acreditada', 'activo',
     clock_timestamp() - interval '2 hours', clock_timestamp() - interval '1 hour');
INSERT INTO vec_contexto_actor_v1.persona_actual(persona_ref, version)
SELECT persona_ref, version FROM vec_contexto_actor_v1.persona_versiones
 WHERE persona_ref LIKE 'per_ctx18_%';

INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones
    (vinculo_ref, version, persona_ref, tipo, referencia,
     procedencia_ref, procedencia_version, procedencia_huella_sha256,
     procedencia_autoridad, estado, vigente_desde, vigente_hasta)
VALUES
    ('vin_ctx18_positivo_000000000000001', 1,
     'per_ctx18_positiva_0000000000000001', 'candidato',
     'can_ctx18_positivo_000000000000001',
     'prc_ctx18_sintetica_00000000000001', 1, repeat('a', 64),
     'autoridad_maestra_acreditada', 'activo',
     clock_timestamp() - interval '1 hour', clock_timestamp() + interval '1 hour'),
    ('vin_ctx18_ambiguo_a_0000000000001', 1,
     'per_ctx18_positiva_0000000000000001', 'candidato',
     'can_ctx18_ambiguo_0000000000000001',
     'prc_ctx18_sintetica_00000000000001', 1, repeat('a', 64),
     'autoridad_maestra_acreditada', 'activo',
     clock_timestamp() - interval '1 hour', clock_timestamp() + interval '1 hour'),
    ('vin_ctx18_ambiguo_b_0000000000001', 1,
     'per_ctx18_ambigua_0000000000000001', 'candidato',
     'can_ctx18_ambiguo_0000000000000001',
     'prc_ctx18_sintetica_00000000000001', 1, repeat('a', 64),
     'autoridad_maestra_acreditada', 'revocado',
     clock_timestamp() - interval '1 hour', clock_timestamp() + interval '1 hour'),
    ('vin_ctx18_revocado_0000000000001', 1,
     'per_ctx18_revocada_000000000000001', 'candidato',
     'can_ctx18_revocado_00000000000001',
     'prc_ctx18_sintetica_00000000000001', 1, repeat('a', 64),
     'autoridad_maestra_acreditada', 'activo',
     clock_timestamp() - interval '1 hour', clock_timestamp() + interval '1 hour'),
    ('vin_ctx18_noaut_0000000000000001', 1,
     'per_ctx18_noaut_000000000000000001', 'candidato',
     'can_ctx18_noaut_00000000000000001',
     'prc_ctx18_no_autoritativa_00000001', 1, repeat('b', 64),
     'no_autoritativa', 'activo',
     clock_timestamp() - interval '1 hour', clock_timestamp() + interval '1 hour'),
    ('vin_ctx18_caducado_0000000000001', 1,
     'per_ctx18_caducada_00000000000001', 'candidato',
     'can_ctx18_caducado_00000000000001',
     'prc_ctx18_sintetica_00000000000001', 1, repeat('a', 64),
     'autoridad_maestra_acreditada', 'activo',
     clock_timestamp() - interval '1 hour', clock_timestamp() + interval '1 hour');
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_actual(vinculo_ref, version)
SELECT vinculo_ref, version FROM vec_contexto_actor_v1.vinculo_referencia_versiones
 WHERE vinculo_ref LIKE 'vin_ctx18_%';

-- La fuente externa es un snapshot CTX15 propio; no se copian sus componentes
-- a persona_actual ni a vinculo_referencia_actual.
DO $externo$
DECLARE
    etiqueta text;
    referencia text;
    componente jsonb;
    snapshot jsonb := pg_catalog.jsonb_build_object(
        'provision_ref', 'pce_ctx18_externa_0000000000000001',
        'poblacion', 'candidato', 'estado', 'activo');
    desde text := pg_catalog.to_char(
        (pg_catalog.clock_timestamp() - interval '1 hour') AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
    hasta text := pg_catalog.to_char(
        (pg_catalog.clock_timestamp() + interval '1 hour') AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
BEGIN
    FOREACH etiqueta IN ARRAY ARRAY[
        'cuenta', 'persona', 'perfil', 'contexto', 'vinculo_candidato'] LOOP
        referencia := CASE etiqueta
            WHEN 'cuenta' THEN 'cta_ctx18_externa_0000000000000001'
            WHEN 'persona' THEN 'per_ctx18_externa_0000000000000001'
            WHEN 'perfil' THEN 'prf_ctx18_externa_0000000000000001'
            WHEN 'contexto' THEN 'vca_ctx18_externa_0000000000000001'
            ELSE 'vin_ctx18_externa_0000000000000001' END;
        componente := pg_catalog.jsonb_build_object(
            'referencia', referencia, 'version', 1,
            'procedencia_ref', 'prc_ctx18_sintetica_00000000000001',
            'procedencia_version', 1,
            'procedencia_huella_sha256', repeat('a', 64),
            'procedencia_autoridad', 'autoridad_maestra_acreditada',
            'estado', 'activo', 'vigente_desde', desde, 'vigente_hasta', hasta);
        IF etiqueta = 'vinculo_candidato' THEN
            componente := componente || pg_catalog.jsonb_build_object(
                'candidato_ref', 'can_ctx18_externo_0000000000000001');
        END IF;
        snapshot := snapshot || pg_catalog.jsonb_build_object(etiqueta, componente);
    END LOOP;
    IF vec_contexto_actor_v1.snapshot_contexto_externo_valido_v1(snapshot) IS NOT TRUE
       OR EXISTS (SELECT 1 FROM vec_contexto_actor_v1.persona_actual
                   WHERE persona_ref = 'per_ctx18_externa_0000000000000001') THEN
        RAISE EXCEPTION 'CTX18: fuente externa sintética inválida';
    END IF;
    INSERT INTO vec_contexto_actor_v1.contexto_externo_identidad
        (provision_ref, cuenta_ref, perfil_ref, persona_ref, familia, contexto_ref)
    VALUES ('pce_ctx18_externa_0000000000000001',
            'cta_ctx18_externa_0000000000000001',
            'prf_ctx18_externa_0000000000000001',
            'per_ctx18_externa_0000000000000001', 'candidato',
            'vca_ctx18_externa_0000000000000001');
    INSERT INTO vec_contexto_actor_v1.contexto_externo_versiones
        (provision_ref, version, cuenta_ref, perfil_ref, persona_ref, familia,
         contexto_ref, snapshot, huella_sha256)
    VALUES ('pce_ctx18_externa_0000000000000001', 1,
            'cta_ctx18_externa_0000000000000001',
            'prf_ctx18_externa_0000000000000001',
            'per_ctx18_externa_0000000000000001', 'candidato',
            'vca_ctx18_externa_0000000000000001', snapshot,
            vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(snapshot, 1));
    INSERT INTO vec_contexto_actor_v1.contexto_externo_actual
        (provision_ref, version)
    VALUES ('pce_ctx18_externa_0000000000000001', 1);
END
$externo$;
COMMIT;

-- Sólo este puente de ensayo representa la llamada de la función de Bolsa
-- propietaria; el runtime no recibe EXECUTE directo sobre la fachada CTX18.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
CREATE FUNCTION vec_bolsa_llamamientos.probar_ctx18_v1(p_candidato_ref text)
RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $f$
    SELECT vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(
        p_candidato_ref)
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.probar_ctx18_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.probar_ctx18_v1(text)
    TO vec_bolsa_llamamientos_ejecutor;
COMMIT;

\connect postgres vec_bolsa_llamamientos_desarrollo
DO $acl$
BEGIN
    BEGIN
        PERFORM vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(
            'can_ctx18_positivo_000000000000001');
        RAISE EXCEPTION 'CTX18: ejecución directa del runtime abierta';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
END
$acl$;
DO $aislamiento$
BEGIN
    BEGIN
        PERFORM vec_bolsa_llamamientos.probar_ctx18_v1(
            'can_ctx18_positivo_000000000000001');
        RAISE EXCEPTION 'CTX18: READ COMMITTED acreditó persona';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
END
$aislamiento$;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
DO $solo_lectura$
BEGIN
    BEGIN
        PERFORM vec_bolsa_llamamientos.probar_ctx18_v1(
            'can_ctx18_positivo_000000000000001');
        RAISE EXCEPTION 'CTX18: transacción de solo lectura acreditó persona';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
END
$solo_lectura$;
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $prueba$
DECLARE r jsonb;
BEGIN
    r := vec_bolsa_llamamientos.probar_ctx18_v1(
        'can_ctx18_positivo_000000000000001');
    IF r ->> 'estado' <> 'acreditado'
       OR r #>> '{persona,ref}' <> 'per_ctx18_positiva_0000000000000001'
       OR r #>> '{persona,version}' <> '1'
       OR r #>> '{vinculo,ref}' <> 'vin_ctx18_positivo_000000000000001'
       OR r #>> '{vinculo,version}' <> '1'
       OR r #>> '{vinculo,poblacion}' <> 'interna'
       OR r #>> '{vinculo,procedencia_ref}' <> 'prc_ctx18_sintetica_00000000000001'
       OR r #>> '{vinculo,procedencia_version}' <> '1'
       OR r #>> '{vinculo,procedencia_sha256}' <> repeat('a', 64)
       OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(r)) <> 3
       OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(r -> 'vinculo')) <> 7 THEN
        RAISE EXCEPTION 'CTX18: acreditación interna incorrecta';
    END IF;
    r := vec_bolsa_llamamientos.probar_ctx18_v1(
        'can_ctx18_externo_0000000000000001');
    IF r ->> 'estado' <> 'acreditado'
       OR r #>> '{persona,ref}' <> 'per_ctx18_externa_0000000000000001'
       OR r #>> '{persona,version}' <> '1'
       OR r #>> '{vinculo,ref}' <> 'vin_ctx18_externa_0000000000000001'
       OR r #>> '{vinculo,poblacion}' <> 'externa'
       OR r #>> '{vinculo,procedencia_sha256}' <> repeat('a', 64) THEN
        RAISE EXCEPTION 'CTX18: acreditación externa incorrecta';
    END IF;
    FOREACH r IN ARRAY ARRAY[
        vec_bolsa_llamamientos.probar_ctx18_v1('can_ctx18_ambiguo_0000000000000001'),
        vec_bolsa_llamamientos.probar_ctx18_v1('can_ctx18_revocado_00000000000001'),
        vec_bolsa_llamamientos.probar_ctx18_v1('can_ctx18_noaut_00000000000000001'),
        vec_bolsa_llamamientos.probar_ctx18_v1('can_ctx18_caducado_00000000000001'),
        vec_bolsa_llamamientos.probar_ctx18_v1('can_ctx18_ausente_0000000000000001'),
        vec_bolsa_llamamientos.probar_ctx18_v1('referencia_invalida')
    ] LOOP
        IF r IS DISTINCT FROM '{"estado":"pendiente","persona":null,"vinculo":null}'::jsonb THEN
            RAISE EXCEPTION 'CTX18: pendiente divulgó persona o vínculo';
        END IF;
    END LOOP;
END
$prueba$;
COMMIT;
\echo CTX18-PRUEBA-FUENTES-OK

-- Revocar un snapshot externo conserva su historia y debe cerrar la fachada.
\connect postgres postgres
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog, pg_temp;
DO $revocar$
DECLARE s jsonb; k text;
BEGIN
    SELECT snapshot INTO STRICT s
      FROM vec_contexto_actor_v1.contexto_externo_versiones
     WHERE provision_ref = 'pce_ctx18_externa_0000000000000001' AND version = 1;
    s := pg_catalog.jsonb_set(s, '{estado}', '"revocado"'::jsonb);
    FOREACH k IN ARRAY ARRAY[
        'cuenta', 'persona', 'perfil', 'contexto', 'vinculo_candidato'] LOOP
        s := pg_catalog.jsonb_set(s, ARRAY[k, 'estado'], '"revocado"'::jsonb);
    END LOOP;
    INSERT INTO vec_contexto_actor_v1.contexto_externo_versiones
        (provision_ref, version, cuenta_ref, perfil_ref, persona_ref, familia,
         contexto_ref, snapshot, huella_sha256)
    SELECT provision_ref, 2, cuenta_ref, perfil_ref, persona_ref, familia,
           contexto_ref, s,
           vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(s, 2)
      FROM vec_contexto_actor_v1.contexto_externo_versiones
     WHERE provision_ref = 'pce_ctx18_externa_0000000000000001' AND version = 1;
    UPDATE vec_contexto_actor_v1.contexto_externo_actual
       SET version = 2
     WHERE provision_ref = 'pce_ctx18_externa_0000000000000001';
END
$revocar$;
COMMIT;

\connect postgres vec_bolsa_llamamientos_desarrollo
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $revocado$
BEGIN
    IF vec_bolsa_llamamientos.probar_ctx18_v1(
        'can_ctx18_externo_0000000000000001')
       IS DISTINCT FROM '{"estado":"pendiente","persona":null,"vinculo":null}'::jsonb THEN
        RAISE EXCEPTION 'CTX18: snapshot externo revocado acreditado';
    END IF;
END
$revocado$;
COMMIT;

-- Un puntero compartido nuevo no puede ocultar el snapshot externo revocado
-- que reclama el mismo candidato.
\connect postgres postgres
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones
    (vinculo_ref, version, persona_ref, tipo, referencia,
     procedencia_ref, procedencia_version, procedencia_huella_sha256,
     procedencia_autoridad, estado, vigente_desde, vigente_hasta)
VALUES ('vin_ctx18_cruce_00000000000000001', 1,
        'per_ctx18_positiva_0000000000000001', 'candidato',
        'can_ctx18_externo_0000000000000001',
        'prc_ctx18_sintetica_00000000000001', 1, repeat('a', 64),
        'autoridad_maestra_acreditada', 'activo',
        clock_timestamp() - interval '1 hour', clock_timestamp() + interval '1 hour');
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_actual(vinculo_ref, version)
VALUES ('vin_ctx18_cruce_00000000000000001', 1);
COMMIT;

\connect postgres vec_bolsa_llamamientos_desarrollo
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $cruce$
BEGIN
    IF vec_bolsa_llamamientos.probar_ctx18_v1(
        'can_ctx18_externo_0000000000000001')
       IS DISTINCT FROM '{"estado":"pendiente","persona":null,"vinculo":null}'::jsonb THEN
        RAISE EXCEPTION 'CTX18: cruce compartido/externo eligió una fuente';
    END IF;
END
$cruce$;
COMMIT;
\echo CTX18-PRUEBA-REVOCACION-Y-CRUCE-OK
