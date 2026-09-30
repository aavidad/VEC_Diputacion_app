\set ON_ERROR_STOP on
-- Ejecutar en un clon desechable tras CT150 y CT151. No escribe datos.
BEGIN;
SET LOCAL search_path=pg_catalog;
DO $prueba$
DECLARE
    p pg_proc%ROWTYPE;
    s text;
BEGIN
    SELECT * INTO STRICT p FROM pg_proc
     WHERE oid='vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    s:=p.prosrc;
    IF p.proowner<>'vec_contratacion_temporal_propietario'::regrole
       OR NOT p.prosecdef OR p.provolatile<>'v'
       OR p.prorettype<>'jsonb'::regtype
       OR p.proconfig IS DISTINCT FROM ARRAY[
           'search_path=pg_catalog','row_security=on','TimeZone=UTC',
           'lock_timeout=2s','statement_timeout=15s',
           'idle_in_transaction_session_timeout=20s'
       ]::text[]
       OR p.proacl IS DISTINCT FROM ARRAY[
           'vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
           'vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario'
       ]::aclitem[]
       OR encode(sha256(convert_to(s,'UTF8')),'hex')<>'1faa7d805e292180a5d16f41f05afe18e1c4fd1f05a966d64b44dd8aa80be02a' THEN
        RAISE EXCEPTION 'CT151: firma, permisos o definición divergentes';
    END IF;
    IF regexp_count(s,'reserva_creada_ahora')<>6
       OR regexp_count(s,'confirmacion_creada_ahora')<>6
       OR regexp_count(s,'''reserva_creada_ahora'',true')<>1
       OR regexp_count(s,'''confirmacion_creada_ahora'',true')<>1
       OR regexp_count(s,'''reserva_creada_ahora'',false,''confirmacion_creada_ahora'',false')<>4
       OR strpos(s,'RETURN v_resultado;')>=strpos(s,'''reserva_creada_ahora''')
       OR strpos(s,'RETURNING * INTO v_reserva;')>=strpos(s,'''reserva_creada_ahora'',true')
       OR strpos(s,'INSERT INTO vec_contratacion_temporal.entrega_peticion_centro_outbox')>=strpos(s,'''confirmacion_creada_ahora'',true') THEN
        RAISE EXCEPTION 'CT151: señales fuera de las ramas de inserción o replay';
    END IF;
END
$prueba$;
ROLLBACK;
