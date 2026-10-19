\set ON_ERROR_STOP on
-- AD297 / F1-10: el estado actual se exige y copia solo para comprobaciones pendientes.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000297',0));
DO $patch$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(bytea,bytea,jsonb,jsonb,text,text,text)'::regprocedure;
        definicion text; nueva text; marca_validacion text; marca_copia text;
BEGIN
 SELECT pg_get_functiondef(f) INTO STRICT definicion;
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 OR current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR to_regprocedure('vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text,jsonb)') IS NULL
 OR encode(sha256(convert_to(definicion,'UTF8')),'hex') IS DISTINCT FROM
   '81549c426b997d792e0c3e5d56adf9720f25633ceb518a4d989ba120e210de32'
 OR (SELECT p.proacl::text FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM
  '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_autorizacion_atestada_v3_registrador_intentos=X/vec_autorizacion_atestada_v3_propietario}'
 THEN RAISE EXCEPTION 'AD297: preimagen incompatible' USING ERRCODE='55000'; END IF;
 marca_validacion:='WHERE (v_estado_actual OR e ? ''comprobacion_ofrecimiento'') AND (';
 marca_copia:='CASE WHEN v_estado_actual OR (v_estado AND reserva IS NOT NULL AND x.valor ? ''comprobacion_ofrecimiento'') THEN';
 IF (length(definicion)-length(replace(definicion,marca_validacion,'')))<>length(marca_validacion)
 OR (length(definicion)-length(replace(definicion,marca_copia,'')))<>length(marca_copia)
 THEN RAISE EXCEPTION 'AD297: marcas ambiguas' USING ERRCODE='55000'; END IF;
 nueva:=replace(definicion,marca_validacion,
  'WHERE ((v_estado_actual AND e->>''razon''=''pendiente_comprobacion_rrhh'') OR e ? ''comprobacion_ofrecimiento'') AND (');
 nueva:=replace(nueva,marca_copia,
  'CASE WHEN (v_estado_actual AND x.valor->>''razon''=''pendiente_comprobacion_rrhh'') OR (v_estado AND reserva IS NOT NULL AND x.valor ? ''comprobacion_ofrecimiento'') THEN');
 PERFORM set_config('vec.ad297_huella_esperada',encode(sha256(convert_to(nueva,'UTF8')),'hex'),true);
 EXECUTE nueva;
END $patch$;
DO $post$ DECLARE p record; BEGIN
 SELECT pg_get_userbyid(q.proowner) AS dueno,q.prosecdef,q.provolatile,q.proacl::text AS acl,
  q.proconfig,encode(sha256(convert_to(pg_get_functiondef(q.oid),'UTF8')),'hex') AS huella
 INTO STRICT p FROM pg_proc q WHERE q.oid=
  'vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(bytea,bytea,jsonb,jsonb,text,text,text)'::regprocedure;
 IF p.dueno<>'vec_autorizacion_atestada_v3_propietario' OR p.prosecdef IS NOT TRUE OR p.provolatile<>'v'
 OR p.acl IS DISTINCT FROM '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_autorizacion_atestada_v3_registrador_intentos=X/vec_autorizacion_atestada_v3_propietario}'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']
 OR p.huella IS DISTINCT FROM current_setting('vec.ad297_huella_esperada',true)
 THEN RAISE EXCEPTION 'AD297: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
