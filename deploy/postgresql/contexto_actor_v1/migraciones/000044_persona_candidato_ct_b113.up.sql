\set ON_ERROR_STOP on
-- CA44: sólo amplía la guarda de sesión de CTX18 para la llamada anidada
-- desde B113 bajo ACT V3 CT. ACL sigue limitada a propietarios CTX y Bolsa.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:000044',0));
DO $pre$
DECLARE p record; f regprocedure:='vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(text)'::regprocedure;
BEGIN
 SELECT proowner,prosecdef,provolatile,proparallel,proconfig,proacl,prosrc,
  pg_get_functiondef(f) definicion INTO STRICT p FROM pg_proc WHERE oid=f;
 IF NOT EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname=current_user AND r.rolsuper)
    OR p.proowner IS DISTINCT FROM 'vec_contexto_actor_v1_propietario'::regrole
    OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v'
    OR p.proparallel IS DISTINCT FROM 'u'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on']
    OR encode(sha256(convert_to(p.definicion,'UTF8')),'hex')
       IS DISTINCT FROM 'aafe66c07f2ae1d5bb5ad670f88bb533ec3ab50c72727bd2a6a9b276d62d991b'
    OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')
       IS DISTINCT FROM 'fee2748f03d23f53e5e366a3b2c0e3244482ea28d48753a7c38603f2eba68ce7'
    OR p.proacl::text IS DISTINCT FROM
      '{vec_contexto_actor_v1_propietario=X/vec_contexto_actor_v1_propietario,vec_bolsa_llamamientos_propietario=X/vec_contexto_actor_v1_propietario}'
    OR to_regrole('vec_contratacion_temporal_ejecutor') IS NULL
 THEN RAISE EXCEPTION 'CA44: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(p_candidato_ref text)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
AS $function$
DECLARE
    pendiente constant jsonb := pg_catalog.jsonb_build_object(
        'estado', 'pendiente', 'persona', NULL, 'vinculo', NULL);
    ahora timestamptz;
    internos integer;
    externos integer;
    vinculo record;
    persona record;
    externo record;
    componente jsonb;
    persona_externa jsonb;
    etiqueta text;
    parte jsonb;
    max_int64 constant numeric := 9223372036854775807;
BEGIN
    -- El propietario de Bolsa llama desde una transacción de efecto V3.
    -- SECURITY DEFINER cambia current_user; session_user sigue siendo el
    -- runtime original y no puede ser el propietario o migrador.
    IF current_user <> 'vec_contexto_actor_v1_propietario'
       OR session_user = current_user
       OR NOT (pg_catalog.pg_has_role(session_user,
              'vec_bolsa_llamamientos_ejecutor', 'MEMBER')
          OR pg_catalog.pg_has_role(session_user,
              'vec_contratacion_temporal_ejecutor', 'MEMBER'))
       OR pg_catalog.pg_has_role(session_user,
              'vec_bolsa_llamamientos_propietario', 'MEMBER')
       OR pg_catalog.pg_has_role(session_user,
              'vec_bolsa_llamamientos_migrador', 'MEMBER')
       OR pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION 'acreditación de persona no disponible'
            USING ERRCODE = '42501';
    END IF;
    IF vec_contexto_actor_v1.referencia_valida(p_candidato_ref, 'can_') IS NOT TRUE THEN
        RETURN pendiente;
    END IF;

    -- Ambos mutadores toman sus advisory exclusivos antes de avanzar la fila
    -- de generación. FOR SHARE obliga al lector SERIALIZABLE a descubrir una
    -- revocación confirmada después de su instantánea (40001).
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:mutacion_punteros_actuales:v2', 0));
    PERFORM 1 FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 c
     WHERE c.control_id = true FOR SHARE OF c;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'generación compartida ausente' USING ERRCODE = '55000';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:externo:mutacion:v1', 0));
    PERFORM 1 FROM vec_contexto_actor_v1.control_generacion_contexto_externo_v1 c
     WHERE c.control_id = true FOR SHARE OF c;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'generación externa ausente' USING ERRCODE = '55000';
    END IF;

    -- Se incluyen los punteros revocados: uno activo junto a uno revocado
    -- tampoco permite escoger persona por descarte.
    PERFORM 1
      FROM vec_contexto_actor_v1.vinculo_referencia_actual a
      JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v
        USING (vinculo_ref, version)
     WHERE v.tipo = 'candidato' AND v.referencia = p_candidato_ref
     FOR SHARE OF a;
    SELECT pg_catalog.count(*) INTO internos
      FROM vec_contexto_actor_v1.vinculo_referencia_actual a
      JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v
        USING (vinculo_ref, version)
     WHERE v.tipo = 'candidato' AND v.referencia = p_candidato_ref;

    PERFORM 1
      FROM vec_contexto_actor_v1.contexto_externo_actual a
      JOIN vec_contexto_actor_v1.contexto_externo_versiones v
        USING (provision_ref, version)
     WHERE v.familia = 'candidato'
       AND v.snapshot #>> '{vinculo_candidato,candidato_ref}' = p_candidato_ref
     FOR SHARE OF a;
    SELECT pg_catalog.count(*) INTO externos
      FROM vec_contexto_actor_v1.contexto_externo_actual a
      JOIN vec_contexto_actor_v1.contexto_externo_versiones v
        USING (provision_ref, version)
     WHERE v.familia = 'candidato'
       AND v.snapshot #>> '{vinculo_candidato,candidato_ref}' = p_candidato_ref;
    IF internos + externos <> 1 THEN
        RETURN pendiente;
    END IF;
    ahora := pg_catalog.clock_timestamp();

    IF internos = 1 THEN
        SELECT v.* INTO STRICT vinculo
          FROM vec_contexto_actor_v1.vinculo_referencia_actual a
          JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v
            USING (vinculo_ref, version)
         WHERE v.tipo = 'candidato' AND v.referencia = p_candidato_ref
         FOR SHARE OF a;
        IF vinculo.estado IS DISTINCT FROM 'activo'
           OR vinculo.procedencia_autoridad IS DISTINCT FROM 'autoridad_maestra_acreditada'
           OR ahora < vinculo.vigente_desde OR ahora >= vinculo.vigente_hasta
           OR vinculo.version > max_int64
           OR vinculo.procedencia_version > max_int64 THEN
            RETURN pendiente;
        END IF;
        SELECT v.* INTO persona
          FROM vec_contexto_actor_v1.persona_actual a
          JOIN vec_contexto_actor_v1.persona_versiones v
            USING (persona_ref, version)
         WHERE a.persona_ref = vinculo.persona_ref
         FOR SHARE OF a;
        IF NOT FOUND OR persona.estado IS DISTINCT FROM 'activo'
           OR persona.procedencia_autoridad IS DISTINCT FROM 'autoridad_maestra_acreditada'
           OR ahora < persona.vigente_desde OR ahora >= persona.vigente_hasta
           OR persona.version > max_int64 THEN
            RETURN pendiente;
        END IF;
        RETURN pg_catalog.jsonb_build_object(
            'estado', 'acreditado',
            'persona', pg_catalog.jsonb_build_object(
                'ref', persona.persona_ref, 'version', persona.version),
            'vinculo', pg_catalog.jsonb_build_object(
                'ref', vinculo.vinculo_ref, 'version', vinculo.version,
                'procedencia_ref', vinculo.procedencia_ref,
                'procedencia_version', vinculo.procedencia_version,
                'procedencia_sha256', vinculo.procedencia_huella_sha256,
                'poblacion', 'interna',
                'vigente_hasta', pg_catalog.to_char(
                    vinculo.vigente_hasta AT TIME ZONE 'UTC',
                    'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
    END IF;

    SELECT v.* INTO STRICT externo
      FROM vec_contexto_actor_v1.contexto_externo_actual a
      JOIN vec_contexto_actor_v1.contexto_externo_versiones v
        USING (provision_ref, version)
     WHERE v.familia = 'candidato'
       AND v.snapshot #>> '{vinculo_candidato,candidato_ref}' = p_candidato_ref
     FOR SHARE OF a;
    componente := externo.snapshot -> 'vinculo_candidato';
    persona_externa := externo.snapshot -> 'persona';
    IF externo.snapshot ->> 'estado' IS DISTINCT FROM 'activo'
       OR componente ->> 'procedencia_autoridad' IS DISTINCT FROM 'autoridad_maestra_acreditada'
       OR persona_externa ->> 'procedencia_autoridad' IS DISTINCT FROM 'autoridad_maestra_acreditada'
       OR (componente ->> 'version')::numeric > max_int64
       OR (componente ->> 'procedencia_version')::numeric > max_int64
       OR (persona_externa ->> 'version')::numeric > max_int64 THEN
        RETURN pendiente;
    END IF;
    -- CTX15 solo considera vigente la provisión cuando todos sus componentes
    -- están activos y dentro de su ventana. No acreditar desde una fracción
    -- de un snapshot ya vencido, aunque Persona y vínculo aún lo estén.
    FOREACH etiqueta IN ARRAY ARRAY[
        'cuenta', 'persona', 'perfil', 'contexto', 'vinculo_candidato'] LOOP
        parte := externo.snapshot -> etiqueta;
        IF parte ->> 'estado' IS DISTINCT FROM 'activo'
           OR ahora < (parte ->> 'vigente_desde')::timestamptz
           OR ahora >= (parte ->> 'vigente_hasta')::timestamptz THEN
            RETURN pendiente;
        END IF;
    END LOOP;
    RETURN pg_catalog.jsonb_build_object(
        'estado', 'acreditado',
        'persona', pg_catalog.jsonb_build_object(
            'ref', persona_externa ->> 'referencia',
            'version', (persona_externa ->> 'version')::numeric),
        'vinculo', pg_catalog.jsonb_build_object(
            'ref', componente ->> 'referencia',
            'version', (componente ->> 'version')::numeric,
            'procedencia_ref', componente ->> 'procedencia_ref',
            'procedencia_version', (componente ->> 'procedencia_version')::numeric,
            'procedencia_sha256', componente ->> 'procedencia_huella_sha256',
            'poblacion', 'externa',
            'vigente_hasta', componente ->> 'vigente_hasta'));
END
$function$;
DO $post$
DECLARE p record; f regprocedure:='vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(text)'::regprocedure;
BEGIN
 SELECT proowner,prosecdef,provolatile,proparallel,proconfig,proacl,prosrc INTO STRICT p
 FROM pg_proc WHERE oid=f;
 IF current_user<>'vec_contexto_actor_v1_propietario'
    OR p.proowner IS DISTINCT FROM 'vec_contexto_actor_v1_propietario'::regrole
    OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v'
    OR p.proparallel IS DISTINCT FROM 'u'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on']
    OR p.proacl::text IS DISTINCT FROM
      '{vec_contexto_actor_v1_propietario=X/vec_contexto_actor_v1_propietario,vec_bolsa_llamamientos_propietario=X/vec_contexto_actor_v1_propietario}'
    OR strpos(p.prosrc,$guard$OR NOT (pg_catalog.pg_has_role(session_user,
              'vec_bolsa_llamamientos_ejecutor', 'MEMBER')
          OR pg_catalog.pg_has_role(session_user,
              'vec_contratacion_temporal_ejecutor', 'MEMBER'))$guard$)=0
 THEN RAISE EXCEPTION 'CA44: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
