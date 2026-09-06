\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000063',0));
-- Solo reversión controlada sin peticiones en curso. Nunca borra respuestas.
LOCK TABLE vec_contratacion_temporal.respuesta_recibida_rrhh IN ACCESS EXCLUSIVE MODE;
DO $respuesta_sucesor_reversa$
DECLARE
    v_antes record; v_despues record; v_cuerpo text; v_definicion text;
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.respuesta_recibida_rrhh r
        JOIN vec_contratacion_temporal.comunicacion_llamamiento_local c USING(comunicacion_ref)
        WHERE (c.material_json->'solicitud') ? 'TipoAntecedente') THEN
        RAISE EXCEPTION 'se conserva historia de respuesta del sucesor' USING ERRCODE='55000';
    END IF;
    SELECT p.oid,p.prosrc AS cuerpo,pg_get_functiondef(p.oid) AS definicion,
           p.proacl AS acl,p.proowner AS propietario,p.proconfig AS configuracion
      INTO v_antes FROM pg_proc p
     WHERE p.oid=to_regprocedure('vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
    IF NOT FOUND OR encode(sha256(convert_to(v_antes.cuerpo,'UTF8')),'hex')
       IS DISTINCT FROM 'ca76c1e1dc2a3746afd39e978e49d7a4f1c3787c59f6507f2c666b28e0023d79' THEN
        RAISE EXCEPTION 'definición CT63 incompatible, no se sobrescribe' USING ERRCODE='55000';
    END IF;
    v_cuerpo:=regexp_replace(v_antes.cuerpo,
        '(?s)    -- CT63 antecedente:.*?    -- CT63 fin antecedente\.\n','');
    v_cuerpo:=replace(v_cuerpo,
        $nuevo$       OR (v_comunicacion.material_json->'solicitud'->>'TipoAntecedente' IS DISTINCT FROM 'continuacion_confirmada'
           AND v_seleccion.recibo_json ->> 'llamamiento_ref' IS DISTINCT FROM s ->> 'LlamamientoRef')$nuevo$,
        $anterior$       OR v_seleccion.recibo_json ->> 'llamamiento_ref' IS DISTINCT FROM s ->> 'LlamamientoRef'$anterior$);
    v_cuerpo:=replace(v_cuerpo,
        $nuevo$       OR (v_comunicacion.material_json->'solicitud'->>'TipoAntecedente' IS DISTINCT FROM 'continuacion_confirmada'
           AND v_seleccion.recibo_json ->> 'recibo_ref' IS DISTINCT FROM
           v_comunicacion.material_json -> 'solicitud' ->> 'PruebaEntregaRef')$nuevo$,
        $anterior$       OR v_seleccion.recibo_json ->> 'recibo_ref' IS DISTINCT FROM
           v_comunicacion.material_json -> 'solicitud' ->> 'PruebaEntregaRef'$anterior$);
    IF encode(sha256(convert_to(v_cuerpo,'UTF8')),'hex')
       IS DISTINCT FROM '3fbbc01c66c85959cd922738a2fb72580c872221cd5387706ceea25d73da442d' THEN
        RAISE EXCEPTION 'reversión CT56 no exacta' USING ERRCODE='55000';
    END IF;
    v_definicion:=replace(v_antes.definicion,v_antes.cuerpo,v_cuerpo);
    EXECUTE v_definicion;
    SELECT pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,p.proowner AS propietario,
           p.proconfig AS configuracion,p.prosecdef AS definidor
      INTO STRICT v_despues FROM pg_proc p WHERE p.oid=v_antes.oid;
    IF v_despues.definicion IS DISTINCT FROM v_definicion
       OR v_despues.acl IS DISTINCT FROM v_antes.acl
       OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
       OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
       OR v_despues.definidor IS NOT TRUE THEN
        RAISE EXCEPTION 'reversión CT63 alteró definición o permisos' USING ERRCODE='55000';
    END IF;
END
$respuesta_sucesor_reversa$;
COMMIT;
