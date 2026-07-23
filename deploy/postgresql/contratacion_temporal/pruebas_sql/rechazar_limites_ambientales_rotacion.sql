\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;

DO $prueba$
DECLARE
    v_mensaje text;
    v_esperado text :=
        'límites ambientales de ejecución ausentes o inválidos';
BEGIN
    BEGIN
        PERFORM *
        FROM vec_contratacion_temporal.preparar_alta_v2('{}'::jsonb);
        RAISE EXCEPTION 'se aceptó una invocación sin límites ambientales';
    EXCEPTION
        WHEN invalid_parameter_value THEN
            GET STACKED DIAGNOSTICS v_mensaje = MESSAGE_TEXT;
            IF v_mensaje <> v_esperado THEN
                RAISE EXCEPTION
                    'fallo distinto al límite ausente: %',
                    v_mensaje;
            END IF;
    END;

    PERFORM pg_catalog.set_config('statement_timeout', '0', true);
    PERFORM pg_catalog.set_config(
        'idle_in_transaction_session_timeout',
        '0',
        true
    );
    BEGIN
        PERFORM *
        FROM vec_contratacion_temporal.preparar_alta_v2('{}'::jsonb);
        RAISE EXCEPTION 'se aceptaron límites ambientales nulos';
    EXCEPTION
        WHEN invalid_parameter_value THEN
            GET STACKED DIAGNOSTICS v_mensaje = MESSAGE_TEXT;
            IF v_mensaje <> v_esperado THEN
                RAISE EXCEPTION 'fallo distinto al límite nulo: %', v_mensaje;
            END IF;
    END;

    PERFORM pg_catalog.set_config('statement_timeout', '15001ms', true);
    PERFORM pg_catalog.set_config(
        'idle_in_transaction_session_timeout',
        '20s',
        true
    );
    BEGIN
        PERFORM *
        FROM vec_contratacion_temporal.preparar_alta_v2('{}'::jsonb);
        RAISE EXCEPTION 'se aceptó statement_timeout por encima del máximo';
    EXCEPTION
        WHEN invalid_parameter_value THEN
            GET STACKED DIAGNOSTICS v_mensaje = MESSAGE_TEXT;
            IF v_mensaje <> v_esperado THEN
                RAISE EXCEPTION
                    'fallo distinto al exceso de sentencia: %',
                    v_mensaje;
            END IF;
    END;

    PERFORM pg_catalog.set_config('statement_timeout', '15s', true);
    PERFORM pg_catalog.set_config(
        'idle_in_transaction_session_timeout',
        '20001ms',
        true
    );
    BEGIN
        PERFORM *
        FROM vec_contratacion_temporal.preparar_alta_v2('{}'::jsonb);
        RAISE EXCEPTION 'se aceptó idle timeout por encima del máximo';
    EXCEPTION
        WHEN invalid_parameter_value THEN
            GET STACKED DIAGNOSTICS v_mensaje = MESSAGE_TEXT;
            IF v_mensaje <> v_esperado THEN
                RAISE EXCEPTION
                    'fallo distinto al exceso de inactividad: %',
                    v_mensaje;
            END IF;
    END;
END
$prueba$;

ROLLBACK;
