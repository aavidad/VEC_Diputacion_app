\set ON_ERROR_STOP on
-- Solo PostgreSQL 18 desechable con CA39 instalada; referencias sintéticas.
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.procedencias
    (procedencia_ref, procedencia_version, procedencia_huella_sha256,
     procedencia_autoridad)
VALUES ('prc_ca39_sintetica_00000000000001', 1, repeat('a', 64),
        'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.persona_versiones
    (persona_ref, version, procedencia_ref, procedencia_version,
     procedencia_huella_sha256, procedencia_autoridad, estado,
     vigente_desde, vigente_hasta)
SELECT persona_ref, 1, 'prc_ca39_sintetica_00000000000001', 1,
       repeat('a', 64), 'autoridad_maestra_acreditada', 'activo',
       clock_timestamp() - interval '1 hour', clock_timestamp() + interval '1 hour'
  FROM (VALUES
    ('per_ca39_activa_00000000000000001'),
    ('per_ca39_sin_empleado_00000000001'),
    ('per_ca39_perfil_revocado_0000001')
  ) AS personas(persona_ref);
INSERT INTO vec_contexto_actor_v1.persona_actual(persona_ref, version)
SELECT persona_ref, version FROM vec_contexto_actor_v1.persona_versiones
 WHERE persona_ref LIKE 'per_ca39_%';
INSERT INTO vec_contexto_actor_v1.perfil_versiones
    (perfil_ref, version, persona_ref, procedencia_ref,
     procedencia_version, procedencia_huella_sha256, procedencia_autoridad,
     estado, vigente_desde, vigente_hasta)
SELECT perfil_ref, 1, persona_ref,
       'prc_ca39_sintetica_00000000000001', 1, repeat('a', 64),
       'autoridad_maestra_acreditada', estado,
       clock_timestamp() - interval '1 hour', clock_timestamp() + interval '1 hour'
  FROM (VALUES
    ('prf_ca39_activo_0000000000000001',
     'per_ca39_activa_00000000000000001', 'activo'),
    ('prf_ca39_sin_empleado_00000001',
     'per_ca39_sin_empleado_00000000001', 'activo'),
    ('prf_ca39_revocado_0000000000001',
     'per_ca39_perfil_revocado_0000001', 'revocado')
  ) AS perfiles(perfil_ref, persona_ref, estado);
INSERT INTO vec_contexto_actor_v1.perfil_actual(perfil_ref, version)
SELECT perfil_ref, version FROM vec_contexto_actor_v1.perfil_versiones
 WHERE perfil_ref LIKE 'prf_ca39_%';
COMMIT;

BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SELECT * FROM vec_personal.publicar_proyeccion_empleado_persona_v1(
    'pep_ca39_unica_00000000000000001', 1,
    'per_ca39_activa_00000000000000001',
    'emp_ca39_unico_00000000000000001', 'activa',
    clock_timestamp() - interval '1 hour',
    clock_timestamp() + interval '1 hour', NULL,
    'prc_ca39_sintetica_00000000000001', 1, repeat('a', 64));
SELECT * FROM vec_personal.publicar_proyeccion_empleado_persona_v1(
    'pep_ca39_revocado_0000000000001', 1,
    'per_ca39_perfil_revocado_0000001',
    'emp_ca39_revocado_0000000000001', 'activa',
    clock_timestamp() - interval '1 hour',
    clock_timestamp() + interval '1 hour', NULL,
    'prc_ca39_sintetica_00000000000001', 1, repeat('a', 64));
COMMIT;

-- El perfil externo existe en CTX15, separado del perfil interno.
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $externo$
DECLARE
    etiqueta text;
    referencia text;
    componente jsonb;
    snapshot jsonb := pg_catalog.jsonb_build_object(
        'provision_ref', 'pce_ca39_externa_0000000000000001',
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
            WHEN 'cuenta' THEN 'cta_ca39_externa_0000000000000001'
            WHEN 'persona' THEN 'per_ca39_externa_0000000000000001'
            WHEN 'perfil' THEN 'prf_ca39_externa_0000000000000001'
            WHEN 'contexto' THEN 'vca_ca39_externa_0000000000000001'
            ELSE 'vin_ca39_externa_0000000000000001' END;
        componente := pg_catalog.jsonb_build_object(
            'referencia', referencia, 'version', 1,
            'procedencia_ref', 'prc_ca39_sintetica_00000000000001',
            'procedencia_version', 1,
            'procedencia_huella_sha256', repeat('a', 64),
            'procedencia_autoridad', 'autoridad_maestra_acreditada',
            'estado', 'activo', 'vigente_desde', desde, 'vigente_hasta', hasta);
        IF etiqueta = 'vinculo_candidato' THEN
            componente := componente || pg_catalog.jsonb_build_object(
                'candidato_ref', 'can_ca39_externo_0000000000000001');
        END IF;
        snapshot := snapshot || pg_catalog.jsonb_build_object(etiqueta, componente);
    END LOOP;
    IF vec_contexto_actor_v1.snapshot_contexto_externo_valido_v1(snapshot) IS NOT TRUE THEN
        RAISE EXCEPTION 'CA39: fixture externa invalida';
    END IF;
    INSERT INTO vec_contexto_actor_v1.contexto_externo_identidad
        (provision_ref, cuenta_ref, perfil_ref, persona_ref, familia, contexto_ref)
    VALUES ('pce_ca39_externa_0000000000000001',
            'cta_ca39_externa_0000000000000001',
            'prf_ca39_externa_0000000000000001',
            'per_ca39_externa_0000000000000001', 'candidato',
            'vca_ca39_externa_0000000000000001');
    INSERT INTO vec_contexto_actor_v1.contexto_externo_versiones
        (provision_ref, version, cuenta_ref, perfil_ref, persona_ref, familia,
         contexto_ref, snapshot, huella_sha256)
    VALUES ('pce_ca39_externa_0000000000000001', 1,
            'cta_ca39_externa_0000000000000001',
            'prf_ca39_externa_0000000000000001',
            'per_ca39_externa_0000000000000001', 'candidato',
            'vca_ca39_externa_0000000000000001', snapshot,
            vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(snapshot, 1));
    INSERT INTO vec_contexto_actor_v1.contexto_externo_actual
        (provision_ref, version)
    VALUES ('pce_ca39_externa_0000000000000001', 1);
END
$externo$;
COMMIT;

\connect postgres postgres
DO $acl$
BEGIN
    IF pg_catalog.has_function_privilege(
           'vec_bolsa_llamamientos_ejecutor',
           'vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(text,text)',
           'EXECUTE')
       OR NOT pg_catalog.has_function_privilege(
           'vec_autorizacion_propietario',
           'vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(text,text)',
           'EXECUTE')
       OR pg_catalog.has_table_privilege(
           'vec_autorizacion_propietario',
           'vec_personal.proyeccion_empleado_persona_historia', 'SELECT') THEN
        RAISE EXCEPTION 'CA39: ACL abierta';
    END IF;
END
$acl$;
SET SESSION AUTHORIZATION vec_autorizacion_propietario;
DO $aislamiento$
BEGIN
    BEGIN
        PERFORM vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(
            'per_ca39_activa_00000000000000001',
            'prf_ca39_activo_0000000000000001');
        RAISE EXCEPTION 'CA39: READ COMMITTED acreditó empleado';
    EXCEPTION WHEN invalid_transaction_state THEN NULL;
    END;
END
$aislamiento$;
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $casos$
DECLARE r jsonb;
BEGIN
    r := vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(
        'per_ca39_activa_00000000000000001',
        'prf_ca39_activo_0000000000000001');
    IF r ->> 'estado' IS DISTINCT FROM 'acreditado'
       OR r ->> 'persona_ref' IS DISTINCT FROM 'per_ca39_activa_00000000000000001'
       OR r ->> 'perfil_ref' IS DISTINCT FROM 'prf_ca39_activo_0000000000000001'
       OR r ->> 'empleado_ref' IS DISTINCT FROM 'emp_ca39_unico_00000000000000001'
       OR r ->> 'proyeccion_ref' IS DISTINCT FROM 'pep_ca39_unica_00000000000000001'
       OR r ->> 'version' IS DISTINCT FROM '1'
       OR r ->> 'procedencia_ref' IS DISTINCT FROM 'prc_ca39_sintetica_00000000000001'
       OR r ->> 'procedencia_version' IS DISTINCT FROM '1'
       OR r ->> 'procedencia_huella_sha256' IS DISTINCT FROM repeat('a', 64) THEN
        RAISE EXCEPTION 'CA39: positivo incorrecto: %', r;
    END IF;
    IF vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(
           'per_ca39_activa_00000000000000001',
           'prf_ca39_sin_empleado_00000001')
       IS DISTINCT FROM '{"estado":"perfil_no_vigente"}'::jsonb
       OR vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(
           'per_ca39_sin_empleado_00000000001',
           'prf_ca39_sin_empleado_00000001')
       IS DISTINCT FROM '{"estado":"sin_empleado"}'::jsonb
       OR vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(
           'per_ca39_perfil_revocado_0000001',
           'prf_ca39_revocado_0000000000001')
       IS DISTINCT FROM '{"estado":"perfil_no_vigente"}'::jsonb
       OR vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(
           'per_ca39_externa_0000000000000001',
           'prf_ca39_externa_0000000000000001')
       IS DISTINCT FROM '{"estado":"perfil_no_vigente"}'::jsonb THEN
        RAISE EXCEPTION 'CA39: negativo divulgó empleado';
    END IF;
END
$casos$;
COMMIT;
RESET SESSION AUTHORIZATION;

-- La segunda proyección activa de la misma persona fuerza ambigüedad.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SELECT * FROM vec_personal.publicar_proyeccion_empleado_persona_v1(
    'pep_ca39_segunda_000000000000001', 1,
    'per_ca39_activa_00000000000000001',
    'emp_ca39_segundo_0000000000000001', 'activa',
    clock_timestamp() - interval '1 hour',
    clock_timestamp() + interval '1 hour', NULL,
    'prc_ca39_sintetica_00000000000001', 1, repeat('a', 64));
COMMIT;
SET SESSION AUTHORIZATION vec_autorizacion_propietario;
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $ambiguo$
BEGIN
    IF vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(
           'per_ca39_activa_00000000000000001',
           'prf_ca39_activo_0000000000000001')
       IS DISTINCT FROM '{"estado":"ambiguo"}'::jsonb THEN
        RAISE EXCEPTION 'CA39: eligió empleado entre dos proyecciones';
    END IF;
END
$ambiguo$;
COMMIT;
RESET SESSION AUTHORIZATION;
\echo CA39-PRUEBA-EMPLEADO-PERFIL-ACL-NEGATIVOS-OK

-- La revocación terminal de Personal cierra un perfil interno aún activo.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SELECT * FROM vec_personal.publicar_proyeccion_empleado_persona_v1(
    'pep_ca39_retirada_00000000000001', 1,
    'per_ca39_sin_empleado_00000000001',
    'emp_ca39_retirado_0000000000001', 'activa',
    clock_timestamp() - interval '1 hour',
    clock_timestamp() + interval '1 hour', NULL,
    'prc_ca39_sintetica_00000000000001', 1, repeat('a', 64));
COMMIT;
SET SESSION AUTHORIZATION vec_autorizacion_propietario;
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $antes_retirada$
BEGIN
    IF vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(
           'per_ca39_sin_empleado_00000000001',
           'prf_ca39_sin_empleado_00000001') ->> 'estado'
       IS DISTINCT FROM 'acreditado' THEN
        RAISE EXCEPTION 'CA39: empleado previo a revocación no acreditado';
    END IF;
END
$antes_retirada$;
COMMIT;
RESET SESSION AUTHORIZATION;
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SELECT * FROM vec_personal.publicar_proyeccion_empleado_persona_v1(
    'pep_ca39_retirada_00000000000001', 2,
    'per_ca39_sin_empleado_00000000001',
    'emp_ca39_retirado_0000000000001', 'revocada',
    clock_timestamp() - interval '1 hour',
    clock_timestamp() + interval '1 hour', 'baja',
    'prc_ca39_sintetica_00000000000001', 1, repeat('a', 64));
COMMIT;
SET SESSION AUTHORIZATION vec_autorizacion_propietario;
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $despues_retirada$
BEGIN
    IF vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(
           'per_ca39_sin_empleado_00000000001',
           'prf_ca39_sin_empleado_00000001')
       IS DISTINCT FROM '{"estado":"sin_empleado"}'::jsonb THEN
        RAISE EXCEPTION 'CA39: empleado revocado acreditado';
    END IF;
END
$despues_retirada$;
COMMIT;
RESET SESSION AUTHORIZATION;
\echo CA39-PRUEBA-REVOCACION-PERSONAL-OK
