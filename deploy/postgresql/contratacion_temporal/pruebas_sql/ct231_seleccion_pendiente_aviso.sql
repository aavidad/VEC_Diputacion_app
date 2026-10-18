\set ON_ERROR_STOP on
-- Focal de la copia R58 de aceptación ficticia, tras instalar CT231.
-- Comprueba precondiciones durables y la función instalada; el GET autorizado
-- se prueba por la ruta HTTP con material V3 nuevo y auditoría en la misma TX.
BEGIN;
SET TRANSACTION READ ONLY;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
DO $ct231_focal$
DECLARE
    v_funcion pg_catalog.pg_proc%ROWTYPE;
    v_pendientes integer;
    v_legadas integer;
    v_validas integer;
BEGIN
    SELECT * INTO STRICT v_funcion FROM pg_catalog.pg_proc WHERE oid=
        'vec_contratacion_temporal.consultar_comunicaciones_expediente_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
        pg_catalog.pg_get_functiondef(v_funcion.oid),'UTF8')),'hex') IS DISTINCT FROM
        '2037b0497f2c5bec4db4940f4079d1c44976fb774cff67dad4ca3063defa80f9'
       OR v_funcion.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
       OR v_funcion.prosecdef IS DISTINCT FROM true
       OR v_funcion.proconfig IS DISTINCT FROM ARRAY[
           'search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s'] THEN
        RAISE EXCEPTION 'CT231: proyección o autoridad instaladas divergentes';
    END IF;
    SELECT count(*) INTO v_legadas FROM vec_contratacion_temporal.comunicacion_llamamiento_local;
    SELECT count(*),count(*) FILTER (WHERE
        e.artefacto_canonico IS NOT NULL AND
        e.recibo_json->>'organizacion_ref' IS NOT DISTINCT FROM e.solicitud_json->>'organizacion_ref' AND
        e.recibo_json->>'expediente_ref' IS NOT DISTINCT FROM e.solicitud_json->>'expediente_ref' AND
        e.recibo_json->>'llamamiento_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/-]{2,94}$' AND
        vec_contratacion_temporal.instante_utc_json_canonico_v2(e.recibo_json->'confirmada_en',false) AND
        vec_contratacion_temporal.confirmacion_canonica_seleccion_llamamiento_o6_v1(
            e.artefacto_canonico::jsonb,e.artefacto_canonico,e.recibo_json,
            vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(e.recibo_json)))
      INTO v_pendientes,v_validas
      FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
     WHERE e.situacion='confirmada' AND NOT EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.comunicacion_llamamiento_local c
         WHERE c.seleccion_clave=e.clave_idempotencia);
    IF v_legadas < 1 OR v_pendientes <> 1 OR v_validas <> 1 THEN
        RAISE EXCEPTION 'CT231: datos ficticios de aceptación incoherentes';
    END IF;
END $ct231_focal$;
ROLLBACK;
