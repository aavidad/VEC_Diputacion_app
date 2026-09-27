-- Replay exacto para publicaciones V3 posteriores a esta migracion.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_publica:migracion:000003', 0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_publica:publicacion:v2', 0));
DO $prevalidacion$
BEGIN
    IF current_user <> 'vec_bolsa_publica_migrador'
       OR NOT pg_catalog.pg_has_role(current_user, 'vec_bolsa_publica_publicacion_propietario', 'SET')
       OR NOT pg_catalog.pg_has_role(current_user, 'vec_bolsa_publica_propietario', 'SET') THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'migracion de replay publico rechazada: identidad incorrecta';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc AS p
        JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
        WHERE n.nspname = 'vec_bolsa_publica_publicacion'
          AND p.proname = 'publicar_proyeccion_v3'
          AND pg_catalog.oidvectortypes(p.proargtypes) = 'jsonb, jsonb, text'
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc AS p
        JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
        WHERE n.nspname = 'vec_bolsa_publica_publicacion'
          AND p.proname IN ('publicar_proyeccion_v3_original', 'confirmar_replay_proyeccion_v3')
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_class AS c
        JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
        WHERE n.nspname = 'vec_bolsa_publica_publicacion'
          AND c.relname = 'testigo_replay_v3'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'migracion de replay publico rechazada: preimagen incompatible';
    END IF;
END
$prevalidacion$;

SET LOCAL ROLE vec_bolsa_publica_propietario;
GRANT SELECT (control_id, manifiesto_sha256, actualizada_en)
    ON vec_bolsa_publica_datos.fuente
    TO vec_bolsa_publica_publicacion_propietario;
SET LOCAL ROLE vec_bolsa_publica_publicacion_propietario;
ALTER FUNCTION vec_bolsa_publica_publicacion.publicar_proyeccion_v3(jsonb,jsonb,text)
    RENAME TO publicar_proyeccion_v3_original;
REVOKE ALL ON FUNCTION vec_bolsa_publica_publicacion.publicar_proyeccion_v3_original(jsonb,jsonb,text)
    FROM vec_bolsa_publica_publicador_login, PUBLIC;

-- Solo huellas: ni proyeccion ni documentos se duplican. El LOGIN no recibe SELECT.
CREATE TABLE vec_bolsa_publica_publicacion.testigo_replay_v3 (
    ancla_manifiesto_sha256 text PRIMARY KEY CHECK (
        ancla_manifiesto_sha256 ~ '^[a-f0-9]{64}$'
        AND ancla_manifiesto_sha256 <> pg_catalog.repeat('0', 64)
    ),
    proyeccion_v2_sha256 bytea NOT NULL CHECK (pg_catalog.octet_length(proyeccion_v2_sha256) = 32),
    bolsas_v1_sha256 bytea NOT NULL CHECK (pg_catalog.octet_length(bolsas_v1_sha256) = 32)
);
ALTER TABLE vec_bolsa_publica_publicacion.testigo_replay_v3 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_publica_publicacion.testigo_replay_v3 FORCE ROW LEVEL SECURITY;
CREATE POLICY solo_propietario_replay ON vec_bolsa_publica_publicacion.testigo_replay_v3
    TO vec_bolsa_publica_publicacion_propietario
    USING (current_user = 'vec_bolsa_publica_publicacion_propietario')
    WITH CHECK (current_user = 'vec_bolsa_publica_publicacion_propietario');
REVOKE ALL ON vec_bolsa_publica_publicacion.testigo_replay_v3 FROM PUBLIC;

CREATE FUNCTION vec_bolsa_publica_publicacion.publicar_proyeccion_v3(
    p_proyeccion_v2 jsonb, p_bolsas_v1 jsonb, p_ancla_manifiesto_sha256 text
) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET log_parameter_max_length_on_error = 0
AS $funcion$
DECLARE v_anterior timestamptz;
BEGIN
    IF session_user <> 'vec_bolsa_publica_publicador_login'
       OR current_user <> 'vec_bolsa_publica_publicacion_propietario'
       OR NOT pg_catalog.pg_has_role(session_user, 'vec_bolsa_publica_publicador', 'MEMBER') THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'publicacion rechazada: identidad publicadora incorrecta';
    END IF;
    -- Serializar tambien la comprobacion temporal. Un reintento tardio con
    -- ancla nueva no puede sustituir una fuente publica mas reciente.
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended('vec_bolsa_publica:publicacion:v2', 0)
    );
    SELECT actualizada_en INTO v_anterior FROM vec_bolsa_publica_datos.fuente
     WHERE control_id;
    -- La funcion original conserva validaciones, DML y consumo unico.
    PERFORM vec_bolsa_publica_publicacion.publicar_proyeccion_v3_original(
        p_proyeccion_v2, p_bolsas_v1, p_ancla_manifiesto_sha256
    );
    IF v_anterior IS NOT NULL AND (
        SELECT actualizada_en FROM vec_bolsa_publica_datos.fuente WHERE control_id
    ) <= v_anterior THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'publicacion rechazada: fuente no posterior';
    END IF;
    INSERT INTO vec_bolsa_publica_publicacion.testigo_replay_v3(
        ancla_manifiesto_sha256, proyeccion_v2_sha256, bolsas_v1_sha256
    ) VALUES (
        p_ancla_manifiesto_sha256,
        pg_catalog.sha256(pg_catalog.convert_to(p_proyeccion_v2::text, 'UTF8')),
        pg_catalog.sha256(pg_catalog.convert_to(p_bolsas_v1::text, 'UTF8'))
    );
END
$funcion$;
REVOKE ALL ON FUNCTION vec_bolsa_publica_publicacion.publicar_proyeccion_v3(jsonb,jsonb,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_publica_publicacion.publicar_proyeccion_v3(jsonb,jsonb,text)
    TO vec_bolsa_publica_publicador_login;

CREATE FUNCTION vec_bolsa_publica_publicacion.confirmar_replay_proyeccion_v3(
    p_proyeccion_v2 jsonb, p_bolsas_v1 jsonb, p_ancla_manifiesto_sha256 text
) RETURNS boolean
LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET log_parameter_max_length_on_error = 0
AS $funcion$
DECLARE v_confirmado boolean;
BEGIN
    IF session_user <> 'vec_bolsa_publica_publicador_login'
       OR current_user <> 'vec_bolsa_publica_publicacion_propietario'
       OR NOT pg_catalog.pg_has_role(session_user, 'vec_bolsa_publica_publicador', 'MEMBER') THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'replay publico rechazado: identidad incorrecta';
    END IF;
    -- Misma configuracion gobernada que exige V2; sin exponer tabla ni huellas.
    IF pg_catalog.current_setting('application_name') <> 'vec-bolsa-publicador'
       -- Un snapshot RR/SERIALIZABLE podria ser anterior al candado y
       -- confirmar A despues de que B ya haya comprometido el cambio.
       OR pg_catalog.current_setting('transaction_isolation') <> 'read committed'
       OR pg_catalog.replace(pg_catalog.current_setting('search_path'), ' ', '') <> 'pg_catalog,pg_temp'
       OR pg_catalog.current_setting('statement_timeout')::interval <> interval '60 seconds'
       OR pg_catalog.current_setting('lock_timeout')::interval <> interval '5 seconds'
       OR pg_catalog.current_setting('idle_in_transaction_session_timeout')::interval <> interval '5 seconds'
       OR pg_catalog.current_setting('transaction_timeout')::interval <> interval '2 minutes'
       OR pg_catalog.current_setting('log_parameter_max_length_on_error') <> '0'
       OR NOT (
           SELECT COALESCE(identidad.rolconfig, ARRAY[]::text[]) @> ARRAY[
               'application_name=vec-bolsa-publicador', 'search_path="pg_catalog,pg_temp"',
               'statement_timeout=60s', 'lock_timeout=5s',
               'idle_in_transaction_session_timeout=5s', 'transaction_timeout=2min',
               'log_parameter_max_length_on_error=0'
           ] FROM pg_catalog.pg_roles AS identidad WHERE identidad.rolname = session_user
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'replay publico rechazado: LOGIN sin limites gobernados';
    END IF;
    IF p_ancla_manifiesto_sha256 IS NULL OR p_ancla_manifiesto_sha256 !~ '^[a-f0-9]{64}$'
       OR p_ancla_manifiesto_sha256 = pg_catalog.repeat('0', 64)
       OR p_proyeccion_v2 IS NULL OR p_bolsas_v1 IS NULL
       OR pg_catalog.octet_length(p_proyeccion_v2::text) > 268435456
       OR pg_catalog.octet_length(p_bolsas_v1::text) > 67108864 THEN
        RETURN false;
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended('vec_bolsa_publica:publicacion:v2', 0)
    );
    SELECT true INTO v_confirmado
      FROM vec_bolsa_publica_publicacion.testigo_replay_v3 AS testigo
      JOIN vec_bolsa_publica_datos.fuente AS fuente
        ON fuente.control_id AND fuente.manifiesto_sha256 = testigo.ancla_manifiesto_sha256
     WHERE testigo.ancla_manifiesto_sha256 = p_ancla_manifiesto_sha256
       AND testigo.proyeccion_v2_sha256 =
           pg_catalog.sha256(pg_catalog.convert_to(p_proyeccion_v2::text, 'UTF8'))
       AND testigo.bolsas_v1_sha256 =
           pg_catalog.sha256(pg_catalog.convert_to(p_bolsas_v1::text, 'UTF8'));
    RETURN COALESCE(v_confirmado, false);
END
$funcion$;
REVOKE ALL ON FUNCTION vec_bolsa_publica_publicacion.confirmar_replay_proyeccion_v3(jsonb,jsonb,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_publica_publicacion.confirmar_replay_proyeccion_v3(jsonb,jsonb,text)
    TO vec_bolsa_publica_publicador_login;
COMMIT;
