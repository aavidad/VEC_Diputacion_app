-- Ejecutar en la misma sesion tras 000002_proyeccion_bolsas_v1.sql,
-- con 000003 instalada ANTES de la publicacion de esa prueba.
\set ON_ERROR_STOP on
DO $acl$
BEGIN
    IF NOT has_function_privilege(
        'vec_bolsa_publica_publicador_login',
        'vec_bolsa_publica_publicacion.confirmar_replay_proyeccion_v3(jsonb,jsonb,text)', 'EXECUTE'
    ) OR has_function_privilege(
        'vec_bolsa_publica_publicador_login',
        'vec_bolsa_publica_publicacion.publicar_proyeccion_v3_original(jsonb,jsonb,text)', 'EXECUTE'
    ) OR has_table_privilege(
        'vec_bolsa_publica_publicador_login',
        'vec_bolsa_publica_publicacion.testigo_replay_v3', 'SELECT,INSERT,UPDATE,DELETE'
    ) OR has_column_privilege(
        'vec_bolsa_publica_publicador_login',
        'vec_bolsa_publica_datos.fuente', 'manifiesto_sha256', 'SELECT'
    ) THEN
        RAISE EXCEPTION 'ACL del replay V3 inesperadas';
    END IF;
END
$acl$;

SET SESSION AUTHORIZATION vec_bolsa_publica_publicador_login;
SET application_name = 'vec-bolsa-publicador';
SET search_path = 'pg_catalog,pg_temp';
SET statement_timeout = '60s';
SET lock_timeout = '5s';
SET idle_in_transaction_session_timeout = '5s';
SET transaction_timeout = '2min';
SET log_parameter_max_length_on_error = 0;
DO $replay$
DECLARE
    v_fallo text;
BEGIN
    IF vec_bolsa_publica_publicacion.confirmar_replay_proyeccion_v3(
        current_setting('vec.prueba_v2')::jsonb,
        current_setting('vec.prueba_b10')::jsonb, repeat('a',64)
    ) IS NOT TRUE THEN
        RAISE EXCEPTION 'replay exacto no confirmado';
    END IF;
    IF vec_bolsa_publica_publicacion.confirmar_replay_proyeccion_v3(
        current_setting('vec.prueba_v2')::jsonb,
        jsonb_set(current_setting('vec.prueba_b10')::jsonb,
            '{bolsas,0,posiciones,0,estado_clave}', '"ocupado"'), repeat('a',64)
    ) IS NOT FALSE
       OR vec_bolsa_publica_publicacion.confirmar_replay_proyeccion_v3(
        jsonb_set(current_setting('vec.prueba_v2')::jsonb,
            '{fuente,revision}', '"otra-revision"'),
        current_setting('vec.prueba_b10')::jsonb, repeat('a',64)
    ) IS NOT FALSE
       OR vec_bolsa_publica_publicacion.confirmar_replay_proyeccion_v3(
        current_setting('vec.prueba_v2')::jsonb,
        current_setting('vec.prueba_b10')::jsonb, repeat('b',64)
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'replay distinto confirmado';
    END IF;
    BEGIN
        PERFORM vec_bolsa_publica_publicacion.publicar_proyeccion_v3(
            current_setting('vec.prueba_v2')::jsonb,
            current_setting('vec.prueba_b10')::jsonb, repeat('a',64)
        );
        RAISE EXCEPTION 'V3 republico un ancla consumida';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        GET STACKED DIAGNOSTICS v_fallo = MESSAGE_TEXT;
        IF v_fallo <> 'publicacion rechazada: contenido invalido' THEN
            RAISE EXCEPTION 'error de duplicado inesperado: %', v_fallo;
        END IF;
    END;
    BEGIN
        PERFORM vec_bolsa_publica_publicacion.publicar_proyeccion_v3(
            current_setting('vec.prueba_v2')::jsonb,
            current_setting('vec.prueba_b10')::jsonb, repeat('b',64)
        );
        RAISE EXCEPTION 'V3 admitio una fuente no posterior';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        GET STACKED DIAGNOSTICS v_fallo = MESSAGE_TEXT;
        IF v_fallo <> 'publicacion rechazada: fuente no posterior' THEN
            RAISE EXCEPTION 'error temporal inesperado: %', v_fallo;
        END IF;
    END;
END
$replay$;
BEGIN ISOLATION LEVEL REPEATABLE READ;
SELECT count(*) FROM pg_catalog.pg_class;
DO $snapshot$
DECLARE v_fallo text;
BEGIN
    BEGIN
        PERFORM vec_bolsa_publica_publicacion.confirmar_replay_proyeccion_v3(
            current_setting('vec.prueba_v2')::jsonb,
            current_setting('vec.prueba_b10')::jsonb, repeat('a',64)
        );
        RAISE EXCEPTION 'replay admitio snapshot repetible';
    EXCEPTION WHEN SQLSTATE '55000' THEN
        GET STACKED DIAGNOSTICS v_fallo = MESSAGE_TEXT;
        IF v_fallo <> 'replay publico rechazado: LOGIN sin limites gobernados' THEN
            RAISE EXCEPTION 'rechazo de aislamiento inesperado: %', v_fallo;
        END IF;
    END;
END
$snapshot$;
ROLLBACK;
RESET SESSION AUTHORIZATION;

BEGIN;
SET LOCAL ROLE vec_bolsa_publica_publicacion_propietario;
DO $historia$
BEGIN
    IF (SELECT count(*) FROM vec_bolsa_publica_publicacion.testigo_replay_v3) <> 1 THEN
        RAISE EXCEPTION 'replay altero testigo';
    END IF;
END
$historia$;
RESET ROLE;
SET LOCAL ROLE vec_bolsa_publica_propietario;
DO $fuente$
BEGIN
    IF (SELECT count(*) FROM vec_bolsa_publica_datos.manifiesto_consumido) <> 1
       OR (SELECT manifiesto_sha256 FROM vec_bolsa_publica_datos.fuente WHERE control_id) <> repeat('a',64) THEN
        RAISE EXCEPTION 'replay altero historia o ancla';
    END IF;
END
$fuente$;
ROLLBACK;

-- A->B: una publicacion posterior gana, y A deja de ser ancla actual aunque
-- su testigo privado siga conservado para la historia.
SELECT set_config('vec.prueba_v2_b', jsonb_set(
    current_setting('vec.prueba_v2')::jsonb,
    '{fuente,actualizada_en}', '"2026-09-21T10:00:00Z"'
)::text, false);
SELECT set_config('vec.prueba_b10_b', jsonb_set(
    current_setting('vec.prueba_b10')::jsonb,
    '{generado_en}', '"2026-09-21T10:00:00Z"'
)::text, false);
SET SESSION AUTHORIZATION vec_bolsa_publica_publicador_login;
SET application_name = 'vec-bolsa-publicador';
SET search_path = 'pg_catalog,pg_temp';
SET statement_timeout = '60s';
SET lock_timeout = '5s';
SET idle_in_transaction_session_timeout = '5s';
SET transaction_timeout = '2min';
SET log_parameter_max_length_on_error = 0;
SELECT vec_bolsa_publica_publicacion.publicar_proyeccion_v3(
    current_setting('vec.prueba_v2_b')::jsonb,
    current_setting('vec.prueba_b10_b')::jsonb, repeat('b',64)
);
DO $sucesion$
BEGIN
    IF vec_bolsa_publica_publicacion.confirmar_replay_proyeccion_v3(
        current_setting('vec.prueba_v2')::jsonb,
        current_setting('vec.prueba_b10')::jsonb, repeat('a',64)
    ) IS NOT FALSE
       OR vec_bolsa_publica_publicacion.confirmar_replay_proyeccion_v3(
        current_setting('vec.prueba_v2_b')::jsonb,
        current_setting('vec.prueba_b10_b')::jsonb, repeat('b',64)
    ) IS NOT TRUE THEN
        RAISE EXCEPTION 'replay tras A->B no respeto el ancla actual';
    END IF;
END
$sucesion$;
RESET SESSION AUTHORIZATION;

-- Cambio lateral de B10 invalida el ancla y la recuperacion deja de confirmar.
BEGIN;
SET LOCAL ROLE vec_bolsa_publica_propietario;
UPDATE vec_bolsa_publica_datos.bolsa_publica
   SET categoria = 'Prueba lateral'
 WHERE bolsa_ref = 'bolsa:administrativo:2026';
RESET ROLE;
SET SESSION AUTHORIZATION vec_bolsa_publica_publicador_login;
SET application_name = 'vec-bolsa-publicador';
SET search_path = 'pg_catalog,pg_temp';
SET statement_timeout = '60s';
SET lock_timeout = '5s';
SET idle_in_transaction_session_timeout = '5s';
SET transaction_timeout = '2min';
SET log_parameter_max_length_on_error = 0;
DO $invalida$
BEGIN
    IF vec_bolsa_publica_publicacion.confirmar_replay_proyeccion_v3(
        current_setting('vec.prueba_v2_b')::jsonb,
        current_setting('vec.prueba_b10_b')::jsonb, repeat('b',64)
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'replay confirmo ancla invalidada';
    END IF;
END
$invalida$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
