\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion_atestada_v3:migracion:000011', 0));

-- Bloquea emisiones y consumos mientras se comprueba que no hay historia.
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3
    IN SHARE MODE;
DO $proteger$
BEGIN
    IF pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_bolsa_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR EXISTS (
           SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version k
           WHERE k.audiencia_consumo =
               'vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1')
       OR EXISTS (
           SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 a
           WHERE pg_catalog.convert_from(a.capacidad_canonica, 'UTF8')::jsonb
                 ->> 'audiencia_consumo' = 'vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1')
    THEN
        RAISE EXCEPTION 'reversión protegida: consumidor, clave o historia de Bolsa'
            USING ERRCODE = '55000';
    END IF;
END
$proteger$;

-- Retira exclusivamente los dos fragmentos de 000011. Las ramas de 000012
-- y cualquier perfil posterior compatible permanecen intactas.
DO $retirar$
DECLARE
    v_def text;
    v_acl aclitem[];
    v_runtime_anterior text := $runtime_anterior$       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_propietario', 'MEMBER') THEN$runtime_anterior$;
    v_runtime_nuevo text := $runtime_nuevo$       OR NOT (
           (
               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
               AND pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'bolsa_llamamiento'
               AND pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
           )
       ) THEN$runtime_nuevo$;
    v_extension text := $perfil$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'bolsa_llamamiento'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1'
               AND (
                   c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.orden.preparar'
                   OR c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.abrir'
               )
               AND d ->> 'accion' IS NOT DISTINCT FROM c ->> 'operacion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'bolsa'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'integracion_llamamientos_bolsa'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
$perfil$;
BEGIN
    SELECT pg_catalog.pg_get_functiondef(p.oid), p.proacl INTO STRICT v_def, v_acl
      FROM pg_catalog.pg_proc p
     WHERE p.oid = 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner = 'vec_autorizacion_atestada_v3_propietario'::regrole
       AND p.prosecdef;
    IF (pg_catalog.length(v_def) -
        pg_catalog.length(pg_catalog.replace(v_def, v_runtime_nuevo, ''))) <>
       pg_catalog.length(v_runtime_nuevo)
       OR (pg_catalog.length(v_def) -
           pg_catalog.length(pg_catalog.replace(v_def, v_extension, ''))) <>
          pg_catalog.length(v_extension) THEN
        RAISE EXCEPTION 'núcleo incompatible para retirar únicamente Bolsa'
            USING ERRCODE = '55000';
    END IF;
    v_def := pg_catalog.replace(v_def, v_extension, '');
    v_def := pg_catalog.replace(v_def, v_runtime_nuevo, v_runtime_anterior);
    EXECUTE v_def;
    IF (SELECT p.proacl FROM pg_catalog.pg_proc p
        WHERE p.oid = 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl THEN
        RAISE EXCEPTION 'la reversión alteró permisos del núcleo'
            USING ERRCODE = '55000';
    END IF;
END
$retirar$;

DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_bolsa_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
-- USAGE no da acceso a tablas ni funciones privadas y puede ser compartido
-- por otros consumidores. No se retiran concesiones de esquema preexistentes.

-- Dos listas cerradas: CT sola o las siete audiencias del código.
-- Solo se añade/retira Bolsa; cualquier otra definición se rechaza.
DO $audiencias$
DECLARE
    v_def text;
    v_esperada text;
    v_legada text;
    v_audiencias text[] := ARRAY[
        'vec_contratacion_temporal.confirmar_alta_atestada.v1',
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
        'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1',
        'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
        'vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1',
        'vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1',
        'vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1',
        'vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1'
    ];
BEGIN
    SELECT pg_catalog.regexp_replace(
        pg_catalog.pg_get_constraintdef(c.oid, true), '\s+', ' ', 'g')
      INTO STRICT v_def
      FROM pg_catalog.pg_constraint c
     WHERE c.conrelid =
           'vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND c.conname = 'clave_capacidad_version_audiencia_consumo_check'
       AND c.contype = 'c' AND c.convalidated
       AND c.conkey = ARRAY[8]::smallint[];
    v_esperada := 'CHECK (audiencia_consumo = ANY (ARRAY[' ||
        pg_catalog.array_to_string(ARRAY(
            SELECT pg_catalog.quote_literal(a) || '::text'
              FROM pg_catalog.unnest(v_audiencias) a
        ), ', ') || ']))';
    v_legada := 'CHECK (audiencia_consumo = ANY (ARRAY[' ||
        pg_catalog.quote_literal(v_audiencias[1]) || '::text, ' ||
        pg_catalog.quote_literal(v_audiencias[8]) || '::text]))';
    IF v_def = v_legada THEN
        v_audiencias := v_audiencias[1:1];
    ELSIF v_def = v_esperada THEN
        v_audiencias := v_audiencias[1:7];
    ELSE
        RAISE EXCEPTION 'gobierno de audiencias incompatible con 000011'
            USING ERRCODE = '55000';
    END IF;
    ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
        DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
    EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ' ||
        'ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check ' ||
        'CHECK (audiencia_consumo IN (' ||
        pg_catalog.array_to_string(ARRAY(
            SELECT pg_catalog.quote_literal(a)
              FROM pg_catalog.unnest(v_audiencias) a
        ), ', ') || '))';
END
$audiencias$;

COMMIT;
