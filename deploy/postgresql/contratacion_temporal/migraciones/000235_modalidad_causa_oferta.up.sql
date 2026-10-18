\set ON_ERROR_STOP on
-- F1-11: modalidad y causa gobernadas de la oferta; sin reescribir historia.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000235',0));
DO $pre$ BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR to_regprocedure('vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(text,text,text,bigint)') IS NULL
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(text,text,text,bigint)'::regprocedure AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='777f09f2514dad4ac9a552de4a992d15ab2b6f1a8dad94cfd8401e1d5498fe23' AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef IS TRUE AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=5s']::text[] AND p.proacl::text='{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_contratacion_temporal_propietario,vec_bolsa_llamamientos_propietario=X/vec_contratacion_temporal_propietario}')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(jsonb,jsonb)'::regprocedure AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='e6c2b110a6536232368a200df631c04af3b67ab63d6a6260fdcd7480c0222132' AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef IS FALSE AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']::text[] AND p.proacl::text='{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.reservar_seleccion_oferta_ct220(jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='f83d3f430f67ef60f886836cc7ac170984a3ff0e44c83eee59300d8c0f86c6d9' AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef IS TRUE AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[] AND p.proacl::text='{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}')
 THEN RAISE EXCEPTION 'CT235: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
DO $patch$ DECLARE v_def text; BEGIN
 SELECT pg_get_functiondef('vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(jsonb,jsonb)'::regprocedure) INTO v_def;
 IF (length(v_def)-length(replace(v_def,$m18fbbbo0$AND (c->'datos')-ARRAY['categoria','centro','fecha_inicio','fecha_fin','descripcion']='{}'::jsonb$m18fbbbo0$,'')))<>length($m18fbbbo0$AND (c->'datos')-ARRAY['categoria','centro','fecha_inicio','fecha_fin','descripcion']='{}'::jsonb$m18fbbbo0$) THEN RAISE EXCEPTION 'F1-11: marca SQL incompatible: comando_seleccion_oferta_valido_ct220/0' USING ERRCODE='55000'; END IF;
 v_def:=replace(v_def,$m18fbbbo0$AND (c->'datos')-ARRAY['categoria','centro','fecha_inicio','fecha_fin','descripcion']='{}'::jsonb$m18fbbbo0$,$m18fbbbn0$AND (c->'datos')-ARRAY['categoria','centro','fecha_inicio','fecha_fin','descripcion','modalidad_clave','causa_clave']='{}'::jsonb
  AND ((NOT (c->'datos' ? 'modalidad_clave') AND NOT (c->'datos' ? 'causa_clave'))
       OR ((c->'datos' ?& ARRAY['modalidad_clave','causa_clave'])
         AND c#>>'{datos,modalidad_clave}' ~ '^[a-z][a-z0-9._-]{1,79}$'
         AND c#>>'{datos,causa_clave}' ~ '^[a-z][a-z0-9._-]{1,79}$'))$m18fbbbn0$);
 EXECUTE v_def;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(jsonb,jsonb)'::regprocedure AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='922201aec88e29e2e5254670d37333372fef0ba3042d5132bf65b8a12f6bd988' AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef IS FALSE AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']::text[] AND p.proacl::text='{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}') THEN RAISE EXCEPTION 'F1-11: postimagen SQL incompatible: comando_seleccion_oferta_valido_ct220' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef('vec_contratacion_temporal.reservar_seleccion_oferta_ct220(jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) INTO v_def;
 IF (length(v_def)-length(replace(v_def,$me967c1o0$IF v_cabecera.version IS NULL THEN$me967c1o0$,'')))<>length($me967c1o0$IF v_cabecera.version IS NULL THEN$me967c1o0$) THEN RAISE EXCEPTION 'F1-11: marca SQL incompatible: reservar_seleccion_oferta_ct220/0' USING ERRCODE='55000'; END IF;
 v_def:=replace(v_def,$me967c1o0$IF v_cabecera.version IS NULL THEN$me967c1o0$,$me967c1n0$IF NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 previa
   WHERE previa.clave_idempotencia=(p_intencion->>'clave_idempotencia')::uuid) THEN
 IF p_comando->'datos' ?& ARRAY['modalidad_clave','causa_clave'] IS NOT TRUE THEN
  RAISE EXCEPTION 'CT235: la nueva reserva exige modalidad y causa' USING ERRCODE='22023'; END IF;
 IF v_cabecera.version IS NULL THEN$me967c1n0$);
 IF (length(v_def)-length(replace(v_def,$me967c1o1$v_ahora:=date_trunc('microseconds',clock_timestamp());$me967c1o1$,'')))<>length($me967c1o1$v_ahora:=date_trunc('microseconds',clock_timestamp());$me967c1o1$) THEN RAISE EXCEPTION 'F1-11: marca SQL incompatible: reservar_seleccion_oferta_ct220/1' USING ERRCODE='55000'; END IF;
 v_def:=replace(v_def,$me967c1o1$v_ahora:=date_trunc('microseconds',clock_timestamp());$me967c1o1$,$me967c1n1$PERFORM 1 FROM vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(
   p_intencion->>'expediente_ref',p_intencion->>'categoria_ref',
   p_intencion->>'organizacion_ref',(p_intencion->>'version_expediente')::bigint) x
  WHERE x.modalidad_clave=p_comando#>>'{datos,modalidad_clave}'
    AND x.causa_clave=p_comando#>>'{datos,causa_clave}'
    AND x.version_expediente=(p_intencion->>'version_expediente')::bigint;
 IF NOT FOUND THEN RAISE EXCEPTION 'CT235: modalidad o causa distinta del análisis vigente' USING ERRCODE='42501'; END IF;
 END IF;
 v_ahora:=date_trunc('microseconds',clock_timestamp());$me967c1n1$);
 EXECUTE v_def;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.reservar_seleccion_oferta_ct220(jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='3897e9a3f0129c1db8bdeaff18c10ca2ce18d0e1eff84823200412004a0b221b' AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef IS TRUE AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[] AND p.proacl::text='{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}') THEN RAISE EXCEPTION 'F1-11: postimagen SQL incompatible: reservar_seleccion_oferta_ct220' USING ERRCODE='55000'; END IF;
END $patch$;
COMMIT;
