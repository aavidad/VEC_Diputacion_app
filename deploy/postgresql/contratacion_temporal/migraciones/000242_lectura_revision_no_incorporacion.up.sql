\set ON_ERROR_STOP on
-- CT242 / F1-47: la lectura exacta del seguimiento entrega la versión actual
-- y los datos mínimos de identificación para la segunda revisión de RRHH.
-- Conserva firma, propietario, RLS y ACL de CT115; no crea otro lector.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000242',0));

DO $pre$
DECLARE f oid := to_regprocedure('vec_contratacion_temporal.consultar_cese_cierre_expediente_v1(text,text)'); p record;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR f IS NULL
    OR to_regclass('vec_contratacion_temporal.expediente_integral_actual') IS NULL
    OR to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.numero_visible_vigente_v1(text,text)') IS NULL THEN
  RAISE EXCEPTION 'CT242: preimagen ausente' USING ERRCODE='55000'; END IF;
 SELECT proowner,prosecdef,provolatile,proacl::text AS acl,proconfig,
        encode(sha256(convert_to(pg_get_functiondef(oid),'UTF8')),'hex') AS huella
 INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 's'
    OR p.acl IS DISTINCT FROM
       '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC']
    OR p.huella IS DISTINCT FROM 'd40b42b767e919c4d26d8e34bd9560cb21f733cee6e0cc10b0689044f324b5a0' THEN
  RAISE EXCEPTION 'CT242: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.consultar_cese_cierre_expediente_v1(p_organizacion text, p_expediente text)
 RETURNS jsonb
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
AS $function$
DECLARE c record; k record; i record; e record;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') THEN
        RAISE EXCEPTION 'CT115: consulta no autorizada' USING ERRCODE='42501';
    END IF;
    IF NOT vec_contratacion_temporal.referencia_valida_ct115(p_organizacion) OR NOT vec_contratacion_temporal.referencia_valida_ct115(p_expediente) THEN
        RAISE EXCEPTION 'CT115: consulta inválida' USING ERRCODE='22023';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_integral_actual a
                 JOIN vec_contratacion_temporal.expediente_version_integral v
                   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
                WHERE a.expediente_ref=p_expediente
                  AND v.agregado_json->>'organizacion_ref' IS DISTINCT FROM p_organizacion) THEN
        RAISE EXCEPTION 'CT115: consulta no autorizada' USING ERRCODE='42501';
    END IF;
    PERFORM set_config('vec.ct115.organizacion_ref',p_organizacion,true);
    PERFORM set_config('vec.ct115.expediente_ref',p_expediente,true);
    -- Fase y estado salen de las columnas de la versión vigente; el número es
    -- el mismo que muestra el cuadro de RRHH (numeración anual de CT142).
    SELECT a.version AS version_expediente, v.fase_clave, v.estado AS estado_clave,
           vec_contratacion_temporal.numero_visible_vigente_v1(p_expediente, v.agregado_json->>'numero_visible') AS numero_expediente
      INTO e FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v
        ON v.expediente_ref=a.expediente_ref AND v.version=a.version
     WHERE a.expediente_ref=p_expediente
       AND v.agregado_json->>'organizacion_ref'=p_organizacion;
    IF NOT FOUND OR e.version_expediente NOT BETWEEN 1 AND 9007199254740991
       OR e.fase_clave IS NULL OR e.estado_clave IS NULL
       OR e.numero_expediente IS NULL THEN
        RAISE EXCEPTION 'CT242: expediente no disponible' USING ERRCODE='23503';
    END IF;
    SELECT * INTO c FROM vec_contratacion_temporal.cese_nombramiento_v1 WHERE organizacion_ref=p_organizacion AND expediente_ref=p_expediente;
    SELECT * INTO k FROM vec_contratacion_temporal.cierre_expediente_v1 WHERE organizacion_ref=p_organizacion AND expediente_ref=p_expediente;
    SELECT * INTO i FROM vec_contratacion_temporal.incorporacion_expediente_ct115(p_organizacion,p_expediente);
    RETURN jsonb_build_object('esquema','vec.contratacion-temporal.cese-cierre-expediente.v1',
        'expediente_ref',p_expediente,
        'version_expediente',e.version_expediente,'fase_clave',e.fase_clave,
        'estado_clave',e.estado_clave,'numero_expediente',e.numero_expediente,
        'incorporacion',CASE WHEN i.recibo_ref IS NULL THEN NULL ELSE jsonb_build_object('recibo_ref',i.recibo_ref,'inicio',to_char(i.inicio,'YYYY-MM-DD')) END,
        'cese',CASE WHEN c.recibo_ref IS NULL THEN NULL ELSE jsonb_build_object('causa_clave',c.causa_clave,
            'fecha_efecto',to_char(c.fecha_efecto,'YYYY-MM-DD'),'justificante_tipo',c.justificante_tipo,'justificante_ref',c.justificante_ref,
            'justificante_sha256',c.justificante_sha256,
            'recibo',jsonb_build_object('recibo_ref',c.recibo_json->'recibo_ref','registrada_en',c.recibo_json->'registrada_en')) END,
        'cierre',CASE WHEN k.recibo_ref IS NULL THEN NULL ELSE jsonb_build_object('condiciones',to_jsonb(k.condiciones),
            'ginpix_numero',k.ginpix_numero,'ginpix_confirmada_en',coalesce(to_char(k.ginpix_confirmada_en,'YYYY-MM-DD'),''),
            'recibo',jsonb_build_object('recibo_ref',k.recibo_json->'recibo_ref','registrada_en',k.recibo_json->'registrada_en')) END);
END
$function$;

DO $post$
DECLARE p record;
BEGIN
 SELECT proowner,prosecdef,provolatile,proacl::text AS acl,proconfig
 INTO STRICT p FROM pg_proc
 WHERE oid='vec_contratacion_temporal.consultar_cese_cierre_expediente_v1(text,text)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 's'
    OR p.acl IS DISTINCT FROM
       '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC']
    OR has_function_privilege('public', 'vec_contratacion_temporal.consultar_cese_cierre_expediente_v1(text,text)', 'EXECUTE') THEN
  RAISE EXCEPTION 'CT242: ACL o postimagen incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
