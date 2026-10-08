\set ON_ERROR_STOP on
-- CA39: fachada nominal de ContextoActor para el gobierno del rol Bolsa
-- empleado. Personal sigue siendo la autoridad del hecho empleado.
-- No crea perfil, empleado, asignacion ni concesion. Sin DOWN tras consumo.
BEGIN;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contexto_actor_v1:migracion:empleado_perfil_inscripcion:000039', 0));

DO $preimagen$
DECLARE
    dueno oid := pg_catalog.to_regrole('vec_contexto_actor_v1_propietario');
    autoriza oid := pg_catalog.to_regrole('vec_autorizacion_propietario');
    nombre text;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                    WHERE rolname = current_user AND rolsuper)
       OR pg_catalog.current_setting('transaction_isolation') <> 'read committed'
       OR pg_catalog.current_setting('server_version_num')::integer < 180000
       OR pg_catalog.current_setting('server_version_num')::integer >= 190000
       OR dueno IS NULL OR autoriza IS NULL
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                   WHERE oid = autoriza AND (rolcanlogin OR rolbypassrls OR rolsuper))
       OR pg_catalog.to_regprocedure(
           'vec_contexto_actor_v1.proyeccion_empleado_personal_v2(text,timestamptz)') IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(text)') IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(text,text)') IS NOT NULL THEN
        RAISE EXCEPTION 'CA39: preimagen incompatible' USING ERRCODE = '55000';
    END IF;
    FOREACH nombre IN ARRAY ARRAY[
        'persona_actual', 'persona_versiones', 'perfil_actual',
        'perfil_versiones', 'contexto_externo_identidad',
        'control_generacion_punteros_actuales_v2',
        'control_generacion_contexto_externo_v1'
    ] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c
                        WHERE c.oid = pg_catalog.to_regclass(
                            'vec_contexto_actor_v1.' || nombre)
                          AND c.relowner = dueno AND c.relkind = 'r') THEN
            RAISE EXCEPTION 'CA39: fuente ausente o ajena: %', nombre
                USING ERRCODE = '55000';
        END IF;
    END LOOP;
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_trigger
                    WHERE tgrelid = 'vec_contexto_actor_v1.perfil_actual'::regclass
                      AND tgname = 'serializar_mutacion_punteros_actuales_v2'
                      AND tgenabled = 'O')
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_trigger
                    WHERE tgrelid = 'vec_contexto_actor_v1.persona_actual'::regclass
                      AND tgname = 'serializar_mutacion_punteros_actuales_v2'
                      AND tgenabled = 'O') THEN
        RAISE EXCEPTION 'CA39: barrera de generacion ausente'
            USING ERRCODE = '55000';
    END IF;
END
$preimagen$;

SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog, pg_temp;

CREATE FUNCTION vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(
    p_persona_ref text, p_perfil_ref text
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET row_security = on
AS $funcion$
DECLARE
    perfil record;
    persona record;
    empleado record;
    instante timestamptz;
    desde timestamptz;
    hasta timestamptz;
BEGIN
    IF current_user <> 'vec_contexto_actor_v1_propietario'
       OR pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION 'CA39: acreditacion requiere transaccion de efecto'
            USING ERRCODE = '25000';
    END IF;
    IF vec_contexto_actor_v1.referencia_valida(p_persona_ref, 'per_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref, 'prf_') IS NOT TRUE THEN
        RAISE EXCEPTION 'CA39: referencias invalidas' USING ERRCODE = '22023';
    END IF;

    -- Mismo orden que la resolucion interna: generacion ContextoActor y
    -- despues generacion Personal. Una revocacion anterior a la instantanea
    -- SERIALIZABLE termina en 40001; una posterior espera al commit.
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:mutacion_punteros_actuales:v2', 0));
    PERFORM 1 FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 c
     WHERE c.control_id = true FOR SHARE OF c;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'CA39: generacion de contexto ausente'
            USING ERRCODE = '55000';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:externo:mutacion:v1', 0));
    PERFORM 1 FROM vec_contexto_actor_v1.control_generacion_contexto_externo_v1 c
     WHERE c.control_id = true FOR SHARE OF c;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'CA39: generacion externa ausente'
            USING ERRCODE = '55000';
    END IF;
    SELECT v.* INTO perfil
      FROM vec_contexto_actor_v1.perfil_actual a
      JOIN vec_contexto_actor_v1.perfil_versiones v USING (perfil_ref, version)
     WHERE a.perfil_ref = p_perfil_ref
     FOR SHARE OF a;
    IF NOT FOUND OR perfil.persona_ref IS DISTINCT FROM p_persona_ref THEN
        RETURN pg_catalog.jsonb_build_object('estado', 'perfil_no_vigente');
    END IF;
    SELECT v.* INTO persona
      FROM vec_contexto_actor_v1.persona_actual a
      JOIN vec_contexto_actor_v1.persona_versiones v USING (persona_ref, version)
     WHERE a.persona_ref = p_persona_ref
     FOR SHARE OF a;
    IF NOT FOUND OR EXISTS (
        SELECT 1 FROM vec_contexto_actor_v1.contexto_externo_identidad e
         WHERE e.perfil_ref = p_perfil_ref
    ) THEN
        RETURN pg_catalog.jsonb_build_object('estado', 'perfil_no_vigente');
    END IF;

    PERFORM vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(
        p_persona_ref);
    instante := pg_catalog.clock_timestamp();
    IF perfil.estado IS DISTINCT FROM 'activo'
       OR perfil.procedencia_autoridad IS DISTINCT FROM 'autoridad_maestra_acreditada'
       OR instante < perfil.vigente_desde OR instante >= perfil.vigente_hasta
       OR persona.estado IS DISTINCT FROM 'activo'
       OR persona.procedencia_autoridad IS DISTINCT FROM 'autoridad_maestra_acreditada'
       OR instante < persona.vigente_desde OR instante >= persona.vigente_hasta THEN
        RETURN pg_catalog.jsonb_build_object('estado', 'perfil_no_vigente');
    END IF;

    SELECT * INTO STRICT empleado
      FROM vec_contexto_actor_v1.proyeccion_empleado_personal_v2(
          p_persona_ref, instante);
    IF empleado.resultado = 'sin_empleado' THEN
        RETURN pg_catalog.jsonb_build_object('estado', 'sin_empleado');
    ELSIF empleado.resultado = 'ambiguo' THEN
        RETURN pg_catalog.jsonb_build_object('estado', 'ambiguo');
    ELSIF empleado.resultado IS DISTINCT FROM 'empleado'
       OR vec_contexto_actor_v1.referencia_valida(
           empleado.empleado_ref, 'emp_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(
           empleado.proyeccion_ref, 'pep_') IS NOT TRUE THEN
        RAISE EXCEPTION 'CA39: proyeccion Personal divergente'
            USING ERRCODE = '55000';
    END IF;
    desde := greatest(
        perfil.vigente_desde, persona.vigente_desde, empleado.vigente_desde);
    hasta := least(
        perfil.vigente_hasta, persona.vigente_hasta, empleado.vigente_hasta);
    IF instante < desde OR instante >= hasta THEN
        RETURN pg_catalog.jsonb_build_object('estado', 'perfil_no_vigente');
    END IF;
    RETURN pg_catalog.jsonb_build_object(
        'estado', 'acreditado',
        'persona_ref', p_persona_ref, 'perfil_ref', p_perfil_ref,
        'empleado_ref', empleado.empleado_ref,
        'proyeccion_ref', empleado.proyeccion_ref,
        'version', empleado.version,
        'procedencia_ref', empleado.procedencia_ref,
        'procedencia_version', empleado.procedencia_version,
        'procedencia_huella_sha256', empleado.procedencia_huella_sha256,
        'vigente_desde', pg_catalog.to_char(
            desde AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'vigente_hasta', pg_catalog.to_char(
            hasta AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
END
$funcion$;

REVOKE ALL ON FUNCTION vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(text,text)
    FROM PUBLIC, vec_contexto_actor_v1_runtime;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(text,text)
    TO vec_autorizacion_propietario;
RESET ROLE;

DO $postimagen$
DECLARE
    f oid := 'vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(text,text)'::regprocedure;
    dueno oid := 'vec_contexto_actor_v1_propietario'::regrole;
    autoriza oid := 'vec_autorizacion_propietario'::regrole;
BEGIN
    IF (SELECT proowner FROM pg_catalog.pg_proc WHERE oid = f) IS DISTINCT FROM dueno
       OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid = f) IS NOT TRUE
       OR (SELECT proconfig FROM pg_catalog.pg_proc WHERE oid = f)
          IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp', 'row_security=on']::text[]
       OR NOT pg_catalog.has_function_privilege(autoriza, f, 'EXECUTE')
       OR pg_catalog.has_function_privilege('vec_contexto_actor_v1_runtime', f, 'EXECUTE')
       OR pg_catalog.has_function_privilege(autoriza,
           'vec_personal.resolver_empleado_canonico_persona_v1(text,timestamptz)',
           'EXECUTE')
       OR pg_catalog.has_table_privilege(autoriza,
           'vec_personal.proyeccion_empleado_persona_historia', 'SELECT')
       OR pg_catalog.has_table_privilege(autoriza,
           'vec_contexto_actor_v1.perfil_versiones', 'SELECT')
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
                    CROSS JOIN LATERAL pg_catalog.aclexplode(
                        coalesce(p.proacl,
                            pg_catalog.acldefault('f', p.proowner))) acl
                   WHERE p.oid = f
                     AND (acl.grantee = 0
                          OR acl.grantee NOT IN (dueno, autoriza)
                          OR acl.privilege_type <> 'EXECUTE'
                          OR acl.is_grantable)) THEN
        RAISE EXCEPTION 'CA39: ACL incompatible' USING ERRCODE = '55000';
    END IF;
END
$postimagen$;
COMMIT;
