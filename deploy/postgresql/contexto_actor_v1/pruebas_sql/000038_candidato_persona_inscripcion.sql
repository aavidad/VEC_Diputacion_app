\set ON_ERROR_STOP on
-- Ejecutar solo en PostgreSQL 18 desechable con CA38 instalada.
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog, pg_temp;
INSERT INTO vec_contexto_actor_v1.procedencias
    (procedencia_ref, procedencia_version, procedencia_huella_sha256,
     procedencia_autoridad)
VALUES ('prc_ca38_sintetica_00000000000001', 1, repeat('a', 64),
        'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.persona_versiones
    (persona_ref, version, procedencia_ref, procedencia_version,
     procedencia_huella_sha256, procedencia_autoridad, estado,
     vigente_desde, vigente_hasta)
VALUES
    ('per_ca38_unica_000000000000000001', 1,
     'prc_ca38_sintetica_00000000000001', 1, repeat('a', 64),
     'autoridad_maestra_acreditada', 'activo',
     clock_timestamp() - interval '1 hour', clock_timestamp() + interval '1 hour'),
    ('per_ca38_ambigua_0000000000000001', 1,
     'prc_ca38_sintetica_00000000000001', 1, repeat('a', 64),
     'autoridad_maestra_acreditada', 'activo',
     clock_timestamp() - interval '1 hour', clock_timestamp() + interval '1 hour'),
    ('per_ca38_caducada_00000000000001', 1,
     'prc_ca38_sintetica_00000000000001', 1, repeat('a', 64),
     'autoridad_maestra_acreditada', 'activo',
     clock_timestamp() - interval '2 hours', clock_timestamp() - interval '1 hour');
INSERT INTO vec_contexto_actor_v1.persona_actual(persona_ref, version)
SELECT persona_ref, version FROM vec_contexto_actor_v1.persona_versiones
 WHERE persona_ref LIKE 'per_ca38_%';
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones
    (vinculo_ref, version, persona_ref, tipo, referencia,
     procedencia_ref, procedencia_version, procedencia_huella_sha256,
     procedencia_autoridad, estado, vigente_desde, vigente_hasta)
VALUES
    ('vin_ca38_unico_00000000000000001', 1,
     'per_ca38_unica_000000000000000001', 'candidato',
     'can_ca38_unico_00000000000000001',
     'prc_ca38_sintetica_00000000000001', 1, repeat('a', 64),
     'autoridad_maestra_acreditada', 'activo',
     clock_timestamp() - interval '1 hour', clock_timestamp() + interval '1 hour'),
    ('vin_ca38_ambiguo_a_0000000000001', 1,
     'per_ca38_ambigua_0000000000000001', 'candidato',
     'can_ca38_ambiguo_a_0000000000001',
     'prc_ca38_sintetica_00000000000001', 1, repeat('a', 64),
     'autoridad_maestra_acreditada', 'activo',
     clock_timestamp() - interval '1 hour', clock_timestamp() + interval '1 hour'),
    ('vin_ca38_ambiguo_b_0000000000001', 1,
     'per_ca38_ambigua_0000000000000001', 'candidato',
     'can_ca38_ambiguo_b_0000000000001',
     'prc_ca38_sintetica_00000000000001', 1, repeat('a', 64),
     'autoridad_maestra_acreditada', 'revocado',
     clock_timestamp() - interval '1 hour', clock_timestamp() + interval '1 hour'),
    ('vin_ca38_caducado_0000000000001', 1,
     'per_ca38_caducada_00000000000001', 'candidato',
     'can_ca38_caducado_00000000000001',
     'prc_ca38_sintetica_00000000000001', 1, repeat('a', 64),
     'autoridad_maestra_acreditada', 'activo',
     clock_timestamp() - interval '2 hours', clock_timestamp() - interval '1 hour');
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_actual(vinculo_ref, version)
SELECT vinculo_ref, version FROM vec_contexto_actor_v1.vinculo_referencia_versiones
 WHERE vinculo_ref LIKE 'vin_ca38_%';

DO $externo$
DECLARE
    etiqueta text;
    referencia text;
    componente jsonb;
    snapshot jsonb := pg_catalog.jsonb_build_object(
        'provision_ref', 'pce_ca38_externa_0000000000000001',
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
            WHEN 'cuenta' THEN 'cta_ca38_externa_0000000000000001'
            WHEN 'persona' THEN 'per_ca38_externa_0000000000000001'
            WHEN 'perfil' THEN 'prf_ca38_externa_0000000000000001'
            WHEN 'contexto' THEN 'vca_ca38_externa_0000000000000001'
            ELSE 'vin_ca38_externa_0000000000000001' END;
        componente := pg_catalog.jsonb_build_object(
            'referencia', referencia, 'version', 1,
            'procedencia_ref', 'prc_ca38_sintetica_00000000000001',
            'procedencia_version', 1,
            'procedencia_huella_sha256', repeat('a', 64),
            'procedencia_autoridad', 'autoridad_maestra_acreditada',
            'estado', 'activo', 'vigente_desde', desde, 'vigente_hasta', hasta);
        IF etiqueta = 'vinculo_candidato' THEN
            componente := componente || pg_catalog.jsonb_build_object(
                'candidato_ref', 'can_ca38_externo_0000000000000001');
        END IF;
        snapshot := snapshot || pg_catalog.jsonb_build_object(etiqueta, componente);
    END LOOP;
    IF vec_contexto_actor_v1.snapshot_contexto_externo_valido_v1(snapshot) IS NOT TRUE THEN
        RAISE EXCEPTION 'CA38: fixture externa invalida';
    END IF;
    INSERT INTO vec_contexto_actor_v1.contexto_externo_identidad
        (provision_ref, cuenta_ref, perfil_ref, persona_ref, familia, contexto_ref)
    VALUES ('pce_ca38_externa_0000000000000001',
            'cta_ca38_externa_0000000000000001',
            'prf_ca38_externa_0000000000000001',
            'per_ca38_externa_0000000000000001', 'candidato',
            'vca_ca38_externa_0000000000000001');
    INSERT INTO vec_contexto_actor_v1.contexto_externo_versiones
        (provision_ref, version, cuenta_ref, perfil_ref, persona_ref, familia,
         contexto_ref, snapshot, huella_sha256)
    VALUES ('pce_ca38_externa_0000000000000001', 1,
            'cta_ca38_externa_0000000000000001',
            'prf_ca38_externa_0000000000000001',
            'per_ca38_externa_0000000000000001', 'candidato',
            'vca_ca38_externa_0000000000000001', snapshot,
            vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(snapshot, 1));
    INSERT INTO vec_contexto_actor_v1.contexto_externo_actual
        (provision_ref, version)
    VALUES ('pce_ca38_externa_0000000000000001', 1);
END
$externo$;
COMMIT;

-- Puente temporal para simular el mismo anidamiento SECDEF de B97.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
CREATE FUNCTION vec_bolsa_llamamientos.probar_ca38_v1(p_persona_ref text)
RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $f$
    SELECT vec_contexto_actor_v1.resolver_candidato_persona_inscripcion_v1(
        p_persona_ref)
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.probar_ca38_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.probar_ca38_v1(text)
    TO vec_bolsa_llamamientos_ejecutor;
COMMIT;

\connect postgres vec_bolsa_llamamientos_desarrollo
DO $acl$
BEGIN
    BEGIN
        PERFORM vec_contexto_actor_v1.resolver_candidato_persona_inscripcion_v1(
            'per_ca38_unica_000000000000000001');
        RAISE EXCEPTION 'CA38: EXECUTE directo abierto';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
END
$acl$;
DO $aislamiento$
BEGIN
    BEGIN
        PERFORM vec_bolsa_llamamientos.probar_ca38_v1(
            'per_ca38_unica_000000000000000001');
        RAISE EXCEPTION 'CA38: READ COMMITTED acreditó candidato';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
END
$aislamiento$;
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $casos$
DECLARE r jsonb;
BEGIN
    r := vec_bolsa_llamamientos.probar_ca38_v1(
        'per_ca38_unica_000000000000000001');
    IF r ->> 'estado' IS DISTINCT FROM 'acreditado'
       OR r ->> 'candidato_ref' IS DISTINCT FROM 'can_ca38_unico_00000000000000001'
       OR r ->> 'persona_version' IS DISTINCT FROM '1'
       OR r ->> 'vinculo_ref' IS DISTINCT FROM 'vin_ca38_unico_00000000000000001'
       OR r ->> 'procedencia_sha256' IS DISTINCT FROM repeat('a', 64)
       OR r ->> 'poblacion' IS DISTINCT FROM 'interna'
       OR r ? 'contexto_ref' THEN
        RAISE EXCEPTION 'CA38: interno incorrecto: %', r;
    END IF;
    r := vec_bolsa_llamamientos.probar_ca38_v1(
        'per_ca38_externa_0000000000000001');
    IF r ->> 'estado' IS DISTINCT FROM 'acreditado'
       OR r ->> 'candidato_ref' IS DISTINCT FROM 'can_ca38_externo_0000000000000001'
       OR r ->> 'contexto_ref' IS DISTINCT FROM 'vca_ca38_externa_0000000000000001'
       OR r ->> 'contexto_version' IS DISTINCT FROM '1'
       OR r ->> 'poblacion' IS DISTINCT FROM 'externa' THEN
        RAISE EXCEPTION 'CA38: externo incorrecto: %', r;
    END IF;
    IF vec_bolsa_llamamientos.probar_ca38_v1(
           'per_ca38_ambigua_0000000000000001')
       IS DISTINCT FROM '{"estado":"ambiguo"}'::jsonb
       OR vec_bolsa_llamamientos.probar_ca38_v1(
           'per_ca38_caducada_00000000000001')
       IS DISTINCT FROM '{"estado":"sin_vinculo"}'::jsonb
       OR vec_bolsa_llamamientos.probar_ca38_v1(
           'per_ca38_ausente_0000000000000001')
       IS DISTINCT FROM '{"estado":"sin_vinculo"}'::jsonb THEN
        RAISE EXCEPTION 'CA38: estado cerrado incorrecto';
    END IF;
END
$casos$;
COMMIT;
\echo CA38-PRUEBA-ACL-INTERNO-EXTERNO-AMBIGUO-VIGENCIA-OK

-- Un snapshot externo que reclama la misma persona no puede ocultarse por
-- preferencia de poblacion, aunque su can_* sea distinto del interno.
\connect postgres postgres
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $cruce$
DECLARE s jsonb;
BEGIN
    SELECT snapshot INTO STRICT s
      FROM vec_contexto_actor_v1.contexto_externo_versiones
     WHERE provision_ref = 'pce_ca38_externa_0000000000000001' AND version = 1;
    s := pg_catalog.jsonb_set(s, '{provision_ref}',
        '"pce_ca38_cruce_00000000000000001"'::jsonb);
    s := pg_catalog.jsonb_set(s, '{cuenta,referencia}',
        '"cta_ca38_cruce_00000000000000001"'::jsonb);
    s := pg_catalog.jsonb_set(s, '{perfil,referencia}',
        '"prf_ca38_cruce_00000000000000001"'::jsonb);
    s := pg_catalog.jsonb_set(s, '{persona,referencia}',
        '"per_ca38_unica_000000000000000001"'::jsonb);
    s := pg_catalog.jsonb_set(s, '{contexto,referencia}',
        '"vca_ca38_cruce_00000000000000001"'::jsonb);
    s := pg_catalog.jsonb_set(s, '{vinculo_candidato,referencia}',
        '"vin_ca38_cruce_00000000000000001"'::jsonb);
    s := pg_catalog.jsonb_set(s, '{vinculo_candidato,candidato_ref}',
        '"can_ca38_cruce_00000000000000001"'::jsonb);
    IF vec_contexto_actor_v1.snapshot_contexto_externo_valido_v1(s) IS NOT TRUE THEN
        RAISE EXCEPTION 'CA38: cruce sintético inválido';
    END IF;
    INSERT INTO vec_contexto_actor_v1.contexto_externo_identidad
        (provision_ref, cuenta_ref, perfil_ref, persona_ref, familia, contexto_ref)
    VALUES ('pce_ca38_cruce_00000000000000001',
            'cta_ca38_cruce_00000000000000001',
            'prf_ca38_cruce_00000000000000001',
            'per_ca38_unica_000000000000000001', 'candidato',
            'vca_ca38_cruce_00000000000000001');
    INSERT INTO vec_contexto_actor_v1.contexto_externo_versiones
        (provision_ref, version, cuenta_ref, perfil_ref, persona_ref, familia,
         contexto_ref, snapshot, huella_sha256)
    VALUES ('pce_ca38_cruce_00000000000000001', 1,
            'cta_ca38_cruce_00000000000000001',
            'prf_ca38_cruce_00000000000000001',
            'per_ca38_unica_000000000000000001', 'candidato',
            'vca_ca38_cruce_00000000000000001', s,
            vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(s, 1));
    INSERT INTO vec_contexto_actor_v1.contexto_externo_actual
        (provision_ref, version)
    VALUES ('pce_ca38_cruce_00000000000000001', 1);
END
$cruce$;
COMMIT;
\connect postgres vec_bolsa_llamamientos_desarrollo
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $comprobar_cruce$
BEGIN
    IF vec_bolsa_llamamientos.probar_ca38_v1(
           'per_ca38_unica_000000000000000001')
       IS DISTINCT FROM '{"estado":"ambiguo"}'::jsonb THEN
        RAISE EXCEPTION 'CA38: eligió poblacion en cruce interno/externo';
    END IF;
END
$comprobar_cruce$;
COMMIT;
\echo CA38-PRUEBA-CRUCE-INTERNO-EXTERNO-OK
