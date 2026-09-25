-- Fachada nominal de ContextoActor para alta gobernada de empleado en Personal.
-- No publica datos civiles ni deduce empleo de la identidad. Personal consume
-- esta fila dentro de la misma transaccion SERIALIZABLE que publica su historia.
-- Una revocacion concurrente anterior a la instantanea efectiva provoca 40001;
-- la posterior espera a que termine la transaccion de Personal.
-- DOWN prohibido tras historia: la concesion y los recibos consumidores perduran.
BEGIN;
SET LOCAL search_path = pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contexto_actor_v1:migracion:acreditacion_persona_tercero:v1', 0));

DO $preimagen$
DECLARE
    dueno oid := pg_catalog.to_regrole('vec_contexto_actor_v1_propietario');
    personal oid := pg_catalog.to_regrole('vec_personal_propietario');
    puntero oid := pg_catalog.to_regclass('vec_contexto_actor_v1.persona_actual');
    versiones oid := pg_catalog.to_regclass('vec_contexto_actor_v1.persona_versiones');
    control oid := pg_catalog.to_regclass('vec_contexto_actor_v1.control_generacion_punteros_actuales_v2');
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                    WHERE rolname = current_user AND rolsuper)
       OR pg_catalog.current_setting('transaction_isolation') <> 'read committed' THEN
        RAISE EXCEPTION 'ContextoActor 000008 requiere superusuario y transaccion ordinaria'
            USING ERRCODE = '42501';
    END IF;
    IF dueno IS NULL OR personal IS NULL OR puntero IS NULL OR versiones IS NULL OR control IS NULL
       OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.proyeccion_empleado_personal_v2(text,timestamptz)') IS NULL
       OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_persona_tercero_v1(text)') IS NOT NULL
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class
                      WHERE oid = puntero AND relowner = dueno AND relkind = 'r')
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class
                      WHERE oid = versiones AND relowner = dueno AND relkind = 'r')
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class
                      WHERE oid = control AND relowner = dueno AND relkind = 'r')
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_trigger
                      WHERE tgrelid = puntero AND tgname = 'serializar_mutacion_punteros_actuales_v2'
                        AND tgenabled = 'O')
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_trigger
                      WHERE tgrelid = puntero AND tgname = 'avanzar_generacion_punteros_actuales_v2'
                        AND tgenabled = 'O')
       OR (SELECT count(*) FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
            WHERE control_id = true) <> 1 THEN
        RAISE EXCEPTION 'postimagen ContextoActor 000007 o barrera V2 ausente/divergente'
            USING ERRCODE = '55000';
    END IF;
END
$preimagen$;

SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;

CREATE FUNCTION vec_contexto_actor_v1.acreditar_persona_tercero_v1(p_persona_ref text)
RETURNS TABLE (
    persona_ref text,
    persona_version numeric,
    procedencia_ref text,
    procedencia_version numeric,
    procedencia_huella_sha256 text,
    vigente_desde timestamptz,
    vigente_hasta timestamptz
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = on
AS $funcion$
DECLARE
    version_actual record;
    generacion_observada numeric;
    ahora timestamptz;
BEGIN
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION USING ERRCODE = '25000',
            MESSAGE = 'acreditacion de persona requiere SERIALIZABLE de escritura';
    END IF;
    IF vec_contexto_actor_v1.referencia_valida(p_persona_ref, 'per_') IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'referencia de persona invalida';
    END IF;

    -- Todos los mutadores de persona_actual toman este advisory en BEFORE
    -- STATEMENT y actualizan la fila de generacion en AFTER STATEMENT.
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
        pg_catalog.hashtextextended(
            'vec_contexto_actor_v1:mutacion_punteros_actuales:v2', 0));
    SELECT c.generacion INTO STRICT generacion_observada
      FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 AS c
     WHERE c.control_id = true FOR SHARE OF c;

    SELECT v.version, v.procedencia_ref, v.procedencia_version,
           v.procedencia_huella_sha256, v.procedencia_autoridad,
           v.estado, v.vigente_desde, v.vigente_hasta
      INTO version_actual
      FROM vec_contexto_actor_v1.persona_actual AS a
      JOIN vec_contexto_actor_v1.persona_versiones AS v
        USING (persona_ref, version)
     WHERE a.persona_ref = p_persona_ref
     FOR SHARE OF a;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = 'P0002',
            MESSAGE = 'persona no acreditable';
    END IF;

    ahora := pg_catalog.clock_timestamp();
    IF version_actual.estado IS DISTINCT FROM 'activo'
       OR version_actual.procedencia_autoridad IS DISTINCT FROM 'autoridad_maestra_acreditada'
       OR ahora < version_actual.vigente_desde
       OR ahora >= version_actual.vigente_hasta THEN
        RAISE EXCEPTION USING ERRCODE = 'P0002',
            MESSAGE = 'persona no acreditable';
    END IF;

    RETURN QUERY SELECT p_persona_ref, version_actual.version,
        version_actual.procedencia_ref, version_actual.procedencia_version,
        version_actual.procedencia_huella_sha256,
        version_actual.vigente_desde, version_actual.vigente_hasta;
END
$funcion$;

REVOKE ALL ON FUNCTION vec_contexto_actor_v1.acreditar_persona_tercero_v1(text)
    FROM PUBLIC, vec_contexto_actor_v1_runtime;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_personal_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_persona_tercero_v1(text)
    TO vec_personal_propietario;
COMMIT;
