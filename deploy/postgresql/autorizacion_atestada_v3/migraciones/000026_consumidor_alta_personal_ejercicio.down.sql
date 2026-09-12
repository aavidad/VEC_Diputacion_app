\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
-- Orden común con roles y la futura migración de negocio Personal.
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000026',0));

-- Bloqueo antes de leer historia o retirar el perfil; nunca borra filas AD3.
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 IN SHARE MODE;
DO $proteger$
BEGIN
    IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_alta_personal_ejercicio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
                   WHERE n.nspname='vec_personal' AND p.proname='registrar_alta_ejercicio_v1')
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
                   WHERE audiencia_consumo='vec_personal.alta_ejercicio.v1')
       OR EXISTS (
           SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3
            WHERE convert_from(capacidad_canonica,'UTF8')::jsonb->>'audiencia_consumo'='vec_personal.alta_ejercicio.v1'
               OR convert_from(capacidad_canonica,'UTF8')::jsonb->>'operacion'='personal.alta_ejercicio.registrar'
       ) THEN
        RAISE EXCEPTION 'retirada AD3-26 denegada: consumidor, clave o historia Personal' USING ERRCODE='55000';
    END IF;
    -- USAGE se concedió por primera vez en este UP; no se retira si ahora
    -- existe otra función accesible al propietario que pudiera depender de él.
    IF EXISTS (
        SELECT 1 FROM pg_proc p
         WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace
           AND p.oid<>'vec_autorizacion_atestada_v3.registrar_y_consumir_alta_personal_ejercicio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
           AND has_function_privilege('vec_personal_propietario',p.oid,'EXECUTE')
    ) THEN
        RAISE EXCEPTION 'retirada AD3-26 denegada: ACL de otro consumidor Personal' USING ERRCODE='55000';
    END IF;
END
$proteger$;

DO $retirar$
DECLARE
    v_def text; v_nueva text; v_acl aclitem[]; v_owner oid; v_config text[];
    v_runtime_anterior text := $runtime_anterior$       OR NOT (
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
       ) THEN$runtime_anterior$;
    v_runtime_nuevo text := $runtime_nuevo$       OR NOT (
           (
               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
               AND p_perfil_mutacion IS DISTINCT FROM 'alta_personal_ejercicio'
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_migrador', 'MEMBER')
               AND pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'bolsa_llamamiento'
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_migrador', 'MEMBER')
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
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'alta_personal_ejercicio'
               AND pg_catalog.pg_has_role(
                   session_user, 'vec_personal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_autorizacion_atestada_v3_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_autorizacion_atestada_v3_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_autorizacion_atestada_v3_emisor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_autorizacion_atestada_v3_consumidor', 'MEMBER')
           )
       ) THEN$runtime_nuevo$;
    v_extension text := $perfil$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'alta_personal_ejercicio'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_personal.alta_ejercicio.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'personal.alta_ejercicio.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'personal.alta_ejercicio.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'personal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'alta_personal_ejercicio'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'registrar_relacion_ocupacion_sinteticas'
           )
$perfil$;
BEGIN
    SELECT pg_get_functiondef(p.oid),p.proacl,p.proowner,p.proconfig
      INTO STRICT v_def,v_acl,v_owner,v_config
      FROM pg_proc p
     WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
       AND p.prosecdef;
    IF length(v_def)-length(replace(v_def,v_runtime_nuevo,''))<>length(v_runtime_nuevo)
       OR length(v_def)-length(replace(v_def,v_extension,''))<>length(v_extension) THEN
        RAISE EXCEPTION 'núcleo incompatible; no retirar otros perfiles' USING ERRCODE='55000';
    END IF;
    v_nueva := replace(replace(v_def,v_extension,''),v_runtime_nuevo,v_runtime_anterior);
    EXECUTE v_nueva;
    IF EXISTS (
        SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
         AND (p.proacl IS DISTINCT FROM v_acl OR p.proowner IS DISTINCT FROM v_owner
              OR p.proconfig IS DISTINCT FROM v_config OR NOT p.prosecdef
              OR pg_get_functiondef(p.oid) IS DISTINCT FROM v_nueva)
    ) THEN
        RAISE EXCEPTION 'AD3-26 alteró definición o autoridad ajena del núcleo' USING ERRCODE='55000';
    END IF;
END
$retirar$;

-- Dos topologías cerradas: AD3-7/11 completa o AD3-11 legacy seguida de 3/4.
-- Solo Personal se añade/retira; no se admiten subconjuntos, reordenaciones ni extras.
DO $audiencias$
DECLARE
    v_def text; v_esperada text; v_legada text;
    v_audiencias_legadas text[] := ARRAY[
        'vec_contratacion_temporal.confirmar_alta_atestada.v1',
        'vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1',
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
        'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1',
        'vec_personal.alta_ejercicio.v1'
    ];
    v_audiencias text[] := ARRAY[
        'vec_contratacion_temporal.confirmar_alta_atestada.v1',
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
        'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1',
        'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
        'vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1',
        'vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1',
        'vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1',
        'vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1',
        'vec_personal.alta_ejercicio.v1'
    ];
BEGIN
    LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
    SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g')
      INTO STRICT v_def FROM pg_constraint c
     WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND c.conname='clave_capacidad_version_audiencia_consumo_check'
       AND c.contype='c' AND c.convalidated AND c.conkey=ARRAY[8]::smallint[];
    v_esperada := 'CHECK (audiencia_consumo = ANY (ARRAY[' ||
        array_to_string(ARRAY(SELECT quote_literal(a)||'::text'
            FROM unnest(v_audiencias) WITH ORDINALITY AS u(a,n) ORDER BY n),', ') || ']))';
    v_legada := 'CHECK (audiencia_consumo = ANY (ARRAY[' ||
        array_to_string(ARRAY(SELECT quote_literal(a)||'::text'
            FROM unnest(v_audiencias_legadas) WITH ORDINALITY AS u(a,n) ORDER BY n),', ') || ']))';
    IF v_def = v_legada THEN
        v_audiencias := v_audiencias_legadas[1:4];
    ELSIF v_def = v_esperada THEN
        v_audiencias := v_audiencias[1:8];
    ELSE
        RAISE EXCEPTION 'lista de audiencias incompatible con AD3-26' USING ERRCODE='55000';
    END IF;
    ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
        DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
    EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ' ||
        'ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check ' ||
        'CHECK (audiencia_consumo IN (' ||
        array_to_string(ARRAY(SELECT quote_literal(a)
            FROM unnest(v_audiencias) WITH ORDINALITY AS u(a,n) ORDER BY n),', ') || '))';
END
$audiencias$;

DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_alta_personal_ejercicio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
REVOKE USAGE ON SCHEMA vec_autorizacion_atestada_v3 FROM vec_personal_propietario;
COMMIT;
