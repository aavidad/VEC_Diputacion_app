\set ON_ERROR_STOP on
-- CA38: lectura nominal del vinculo actual de candidato de una persona.
-- La persona de la solicitud fue acreditada por V3 al presentarla; esta
-- consulta revalida el vinculo actual para la incorporacion posterior.
-- No publica, elige ni crea candidaturas. Sin DOWN tras uso por B97.
BEGIN;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contexto_actor_v1:migracion:candidato_persona_inscripcion:000038', 0));

DO $preimagen$
DECLARE
    dueno oid := pg_catalog.to_regrole('vec_contexto_actor_v1_propietario');
    bolsa oid := pg_catalog.to_regrole('vec_bolsa_llamamientos_propietario');
    nombre text;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                    WHERE rolname = current_user AND rolsuper)
       OR pg_catalog.current_setting('transaction_isolation') <> 'read committed'
       OR pg_catalog.current_setting('server_version_num')::integer < 180000
       OR pg_catalog.current_setting('server_version_num')::integer >= 190000
       OR dueno IS NULL OR bolsa IS NULL
       OR pg_catalog.to_regrole('vec_bolsa_llamamientos_ejecutor') IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(text)') IS NULL
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                   WHERE oid = bolsa AND (rolcanlogin OR rolbypassrls OR rolsuper))
       OR pg_catalog.to_regprocedure(
           'vec_contexto_actor_v1.resolver_candidato_persona_inscripcion_v1(text)') IS NOT NULL THEN
        RAISE EXCEPTION 'CA38: preimagen incompatible' USING ERRCODE = '55000';
    END IF;
    FOREACH nombre IN ARRAY ARRAY[
        'persona_actual', 'persona_versiones',
        'vinculo_referencia_actual', 'vinculo_referencia_versiones',
        'control_generacion_punteros_actuales_v2',
        'contexto_externo_identidad', 'contexto_externo_actual',
        'contexto_externo_versiones', 'control_generacion_contexto_externo_v1'
    ] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c
                        WHERE c.oid = pg_catalog.to_regclass(
                            'vec_contexto_actor_v1.' || nombre)
                          AND c.relowner = dueno AND c.relkind = 'r') THEN
            RAISE EXCEPTION 'CA38: fuente ausente o ajena: %', nombre
                USING ERRCODE = '55000';
        END IF;
    END LOOP;
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_trigger
                    WHERE tgrelid = 'vec_contexto_actor_v1.vinculo_referencia_actual'::regclass
                      AND tgname = 'serializar_mutacion_punteros_actuales_v2'
                      AND tgenabled = 'O')
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_trigger
                    WHERE tgrelid = 'vec_contexto_actor_v1.contexto_externo_actual'::regclass
                      AND tgname = 'controlar_generacion' AND tgenabled = 'O') THEN
        RAISE EXCEPTION 'CA38: barrera de generacion ausente' USING ERRCODE = '55000';
    END IF;
END
$preimagen$;

SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog, pg_temp;

-- CTX15 no tiene indice por persona; la consulta empieza por esta identidad
-- opaca y nunca recorre snapshots JSON de toda la poblacion externa.
CREATE INDEX contexto_externo_identidad_candidato_persona_ca38
    ON vec_contexto_actor_v1.contexto_externo_identidad
       (persona_ref, provision_ref) WHERE familia = 'candidato';

-- CA18 coteja la referencia en sentido inverso y cuenta ambas poblaciones.
-- Estos indices acotan ese cotejo por can_* incluso con historia extensa.
CREATE INDEX vinculo_referencia_candidato_ca38
    ON vec_contexto_actor_v1.vinculo_referencia_versiones
       (referencia, vinculo_ref, version) WHERE tipo = 'candidato';
CREATE INDEX contexto_externo_candidato_ca38
    ON vec_contexto_actor_v1.contexto_externo_versiones
       ((snapshot #>> '{vinculo_candidato,candidato_ref}'), provision_ref, version)
    WHERE familia = 'candidato';

CREATE FUNCTION vec_contexto_actor_v1.resolver_candidato_persona_inscripcion_v1(
    p_persona_ref text
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET row_security = on
AS $funcion$
DECLARE
    instante timestamptz;
    internos integer;
    externos integer;
    persona record;
    vinculo record;
    externo record;
    inverso jsonb;
    parte jsonb;
    etiqueta text;
    max_int64 constant numeric := 9223372036854775807;
BEGIN
    -- El unico llamador es el propietario Bolsa dentro de su efecto V3.
    -- session_user sigue siendo el ejecutor aun bajo SECURITY DEFINER anidado.
    IF current_user <> 'vec_contexto_actor_v1_propietario'
       OR session_user = current_user
       OR NOT pg_catalog.pg_has_role(session_user,
              'vec_bolsa_llamamientos_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(session_user,
              'vec_bolsa_llamamientos_propietario', 'MEMBER')
       OR pg_catalog.pg_has_role(session_user,
              'vec_bolsa_llamamientos_migrador', 'MEMBER')
       OR pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION 'CA38: lectura de candidato no disponible'
            USING ERRCODE = '42501';
    END IF;
    IF vec_contexto_actor_v1.referencia_valida(p_persona_ref, 'per_') IS NOT TRUE THEN
        RAISE EXCEPTION 'CA38: referencia de persona invalida'
            USING ERRCODE = '22023';
    END IF;

    -- El mismo orden de barreras que CA18. Los mutadores avanzan sus filas de
    -- generacion; una instantanea SERIALIZABLE anterior termina en 40001.
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:mutacion_punteros_actuales:v2', 0));
    PERFORM 1 FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 c
     WHERE c.control_id = true FOR SHARE OF c;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'CA38: generacion compartida ausente'
            USING ERRCODE = '55000';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:externo:mutacion:v1', 0));
    PERFORM 1 FROM vec_contexto_actor_v1.control_generacion_contexto_externo_v1 c
     WHERE c.control_id = true FOR SHARE OF c;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'CA38: generacion externa ausente'
            USING ERRCODE = '55000';
    END IF;
    instante := pg_catalog.clock_timestamp();

    -- Contar hasta dos punteros, incluidos los revocados: no se elige uno
    -- activo descartando otro de la misma persona. El indice de persona de
    -- CA1 acota los internos; el indice CA38 acota los externos.
    SELECT pg_catalog.count(*) INTO internos FROM (
        SELECT 1
          FROM vec_contexto_actor_v1.vinculo_referencia_versiones v
          JOIN vec_contexto_actor_v1.vinculo_referencia_actual a
            USING (vinculo_ref, version)
         WHERE v.persona_ref = p_persona_ref AND v.tipo = 'candidato'
         LIMIT 2
    ) x;
    SELECT pg_catalog.count(*) INTO externos FROM (
        SELECT 1
          FROM vec_contexto_actor_v1.contexto_externo_identidad i
          JOIN vec_contexto_actor_v1.contexto_externo_actual a
            USING (provision_ref)
         WHERE i.persona_ref = p_persona_ref AND i.familia = 'candidato'
         LIMIT 2
    ) x;
    IF internos + externos > 1 THEN
        RETURN pg_catalog.jsonb_build_object('estado', 'ambiguo');
    ELSIF internos + externos = 0 THEN
        RETURN pg_catalog.jsonb_build_object('estado', 'sin_vinculo');
    END IF;

    IF internos = 1 THEN
        SELECT v.* INTO STRICT vinculo
          FROM vec_contexto_actor_v1.vinculo_referencia_versiones v
          JOIN vec_contexto_actor_v1.vinculo_referencia_actual a
            USING (vinculo_ref, version)
         WHERE v.persona_ref = p_persona_ref AND v.tipo = 'candidato'
         FOR SHARE OF a;
        SELECT v.* INTO persona
          FROM vec_contexto_actor_v1.persona_actual a
          JOIN vec_contexto_actor_v1.persona_versiones v
            USING (persona_ref, version)
         WHERE a.persona_ref = p_persona_ref
         FOR SHARE OF a;
        IF NOT FOUND OR persona.estado IS DISTINCT FROM 'activo'
           OR persona.procedencia_autoridad IS DISTINCT FROM 'autoridad_maestra_acreditada'
           OR instante < persona.vigente_desde OR instante >= persona.vigente_hasta
           OR vinculo.estado IS DISTINCT FROM 'activo'
           OR vinculo.procedencia_autoridad IS DISTINCT FROM 'autoridad_maestra_acreditada'
           OR instante < vinculo.vigente_desde OR instante >= vinculo.vigente_hasta
           OR persona.version > max_int64 OR vinculo.version > max_int64
           OR vinculo.procedencia_version > max_int64 THEN
            RETURN pg_catalog.jsonb_build_object('estado', 'sin_vinculo');
        END IF;
        inverso := vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(
            vinculo.referencia);
        IF inverso ->> 'estado' IS DISTINCT FROM 'acreditado'
           OR inverso #>> '{persona,ref}' IS DISTINCT FROM p_persona_ref
           OR (inverso #>> '{persona,version}')::numeric IS DISTINCT FROM persona.version
           OR inverso #>> '{vinculo,ref}' IS DISTINCT FROM vinculo.vinculo_ref
           OR (inverso #>> '{vinculo,version}')::numeric IS DISTINCT FROM vinculo.version
           OR inverso #>> '{vinculo,procedencia_ref}' IS DISTINCT FROM vinculo.procedencia_ref
           OR (inverso #>> '{vinculo,procedencia_version}')::numeric IS DISTINCT FROM vinculo.procedencia_version
           OR inverso #>> '{vinculo,procedencia_sha256}' IS DISTINCT FROM vinculo.procedencia_huella_sha256
           OR inverso #>> '{vinculo,poblacion}' IS DISTINCT FROM 'interna' THEN
            RETURN pg_catalog.jsonb_build_object('estado', 'ambiguo');
        END IF;
        RETURN pg_catalog.jsonb_build_object(
            'estado', 'acreditado', 'persona_ref', p_persona_ref,
            'persona_version', persona.version,
            'candidato_ref', vinculo.referencia,
            'vinculo_ref', vinculo.vinculo_ref, 'version', vinculo.version,
            'procedencia_ref', vinculo.procedencia_ref,
            'procedencia_version', vinculo.procedencia_version,
            'procedencia_sha256', vinculo.procedencia_huella_sha256,
            'poblacion', 'interna', 'acreditado_en', instante);
    END IF;

    SELECT v.* INTO STRICT externo
      FROM vec_contexto_actor_v1.contexto_externo_identidad i
      JOIN vec_contexto_actor_v1.contexto_externo_actual a USING (provision_ref)
      JOIN vec_contexto_actor_v1.contexto_externo_versiones v
        USING (provision_ref, version)
     WHERE i.persona_ref = p_persona_ref AND i.familia = 'candidato'
     FOR SHARE OF a;
    IF externo.persona_ref IS DISTINCT FROM p_persona_ref
       OR externo.familia IS DISTINCT FROM 'candidato'
       OR externo.snapshot ->> 'estado' IS DISTINCT FROM 'activo'
       OR externo.snapshot ->> 'poblacion' IS DISTINCT FROM 'candidato'
       OR externo.huella_sha256 IS DISTINCT FROM
          vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(
              externo.snapshot, externo.version)
       OR externo.version > max_int64 THEN
        RETURN pg_catalog.jsonb_build_object('estado', 'sin_vinculo');
    END IF;
    FOREACH etiqueta IN ARRAY ARRAY[
        'cuenta', 'persona', 'perfil', 'contexto', 'vinculo_candidato'
    ] LOOP
        parte := externo.snapshot -> etiqueta;
        IF parte ->> 'estado' IS DISTINCT FROM 'activo'
           OR parte ->> 'procedencia_autoridad' IS DISTINCT FROM
              'autoridad_maestra_acreditada'
           OR instante < (parte ->> 'vigente_desde')::timestamptz
           OR instante >= (parte ->> 'vigente_hasta')::timestamptz
           OR (parte ->> 'version')::numeric > max_int64
           OR (parte ->> 'procedencia_version')::numeric > max_int64 THEN
            RETURN pg_catalog.jsonb_build_object('estado', 'sin_vinculo');
        END IF;
    END LOOP;
    IF externo.snapshot #>> '{persona,referencia}' IS DISTINCT FROM p_persona_ref
       OR vec_contexto_actor_v1.referencia_valida(
           externo.snapshot #>> '{vinculo_candidato,candidato_ref}', 'can_') IS NOT TRUE THEN
        RETURN pg_catalog.jsonb_build_object('estado', 'sin_vinculo');
    END IF;
    parte := externo.snapshot -> 'vinculo_candidato';
    inverso := vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(
        parte ->> 'candidato_ref');
    IF inverso ->> 'estado' IS DISTINCT FROM 'acreditado'
       OR inverso #>> '{persona,ref}' IS DISTINCT FROM p_persona_ref
       OR (inverso #>> '{persona,version}')::numeric IS DISTINCT FROM
          (externo.snapshot #>> '{persona,version}')::numeric
       OR inverso #>> '{vinculo,ref}' IS DISTINCT FROM parte ->> 'referencia'
       OR (inverso #>> '{vinculo,version}')::numeric IS DISTINCT FROM
          (parte ->> 'version')::numeric
       OR inverso #>> '{vinculo,procedencia_ref}' IS DISTINCT FROM parte ->> 'procedencia_ref'
       OR (inverso #>> '{vinculo,procedencia_version}')::numeric IS DISTINCT FROM
          (parte ->> 'procedencia_version')::numeric
       OR inverso #>> '{vinculo,procedencia_sha256}' IS DISTINCT FROM
          parte ->> 'procedencia_huella_sha256'
       OR inverso #>> '{vinculo,poblacion}' IS DISTINCT FROM 'externa' THEN
        RETURN pg_catalog.jsonb_build_object('estado', 'ambiguo');
    END IF;
    RETURN pg_catalog.jsonb_build_object(
        'estado', 'acreditado', 'persona_ref', p_persona_ref,
        'persona_version', (externo.snapshot #>> '{persona,version}')::numeric,
        'candidato_ref', parte ->> 'candidato_ref',
        'vinculo_ref', parte ->> 'referencia',
        'version', (parte ->> 'version')::numeric,
        'procedencia_ref', parte ->> 'procedencia_ref',
        'procedencia_version', (parte ->> 'procedencia_version')::numeric,
        'procedencia_sha256', parte ->> 'procedencia_huella_sha256',
        'poblacion', 'externa',
        'contexto_ref', externo.contexto_ref,
        'contexto_version', (externo.snapshot #>> '{contexto,version}')::numeric,
        'acreditado_en', instante);
END
$funcion$;

REVOKE ALL ON FUNCTION vec_contexto_actor_v1.resolver_candidato_persona_inscripcion_v1(text)
    FROM PUBLIC, vec_contexto_actor_v1_runtime,
         vec_bolsa_llamamientos_ejecutor, vec_bolsa_llamamientos_migrador;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1
    TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.resolver_candidato_persona_inscripcion_v1(text)
    TO vec_bolsa_llamamientos_propietario;
RESET ROLE;

DO $postimagen$
DECLARE
    f oid := 'vec_contexto_actor_v1.resolver_candidato_persona_inscripcion_v1(text)'::regprocedure;
    dueno oid := 'vec_contexto_actor_v1_propietario'::regrole;
    bolsa oid := 'vec_bolsa_llamamientos_propietario'::regrole;
BEGIN
    IF (SELECT proowner FROM pg_catalog.pg_proc WHERE oid = f) IS DISTINCT FROM dueno
       OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid = f) IS NOT TRUE
       OR (SELECT proconfig FROM pg_catalog.pg_proc WHERE oid = f)
          IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp', 'row_security=on']::text[]
       OR NOT pg_catalog.has_function_privilege(bolsa, f, 'EXECUTE')
       OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor', f, 'EXECUTE')
       OR pg_catalog.has_table_privilege(bolsa,
           'vec_contexto_actor_v1.contexto_externo_identidad', 'SELECT')
       OR pg_catalog.has_table_privilege(bolsa,
           'vec_contexto_actor_v1.vinculo_referencia_versiones', 'SELECT')
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
                    CROSS JOIN LATERAL pg_catalog.aclexplode(
                        coalesce(p.proacl,
                            pg_catalog.acldefault('f', p.proowner))) acl
                   WHERE p.oid = f
                     AND (acl.grantee = 0
                          OR acl.grantee NOT IN (dueno, bolsa)
                          OR acl.privilege_type <> 'EXECUTE'
                          OR acl.is_grantable)) THEN
        RAISE EXCEPTION 'CA38: postimagen ACL incompatible'
            USING ERRCODE = '55000';
    END IF;
END
$postimagen$;
COMMIT;
