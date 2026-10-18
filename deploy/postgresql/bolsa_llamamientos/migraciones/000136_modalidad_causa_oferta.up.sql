\set ON_ERROR_STOP on
-- F1-11: modalidad y causa gobernadas de la oferta; sin reescribir historia.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000136',0));
DO $pre$ BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR to_regprocedure('vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(text,text,text,bigint)') IS NULL
  OR to_regprocedure('vec_bolsa_llamamientos.huella_operacion_oferta_v6(text,text,text,text,integer,jsonb,jsonb,jsonb,timestamptz,timestamptz,jsonb)') IS NOT NULL
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(text,text,text,bigint)'::regprocedure AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='777f09f2514dad4ac9a552de4a992d15ab2b6f1a8dad94cfd8401e1d5498fe23' AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef IS TRUE AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=5s']::text[] AND p.proacl::text='{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_contratacion_temporal_propietario,vec_bolsa_llamamientos_propietario=X/vec_contratacion_temporal_propietario}')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_bolsa_llamamientos.huella_operacion_oferta_v5(text,text,text,text,integer,jsonb,jsonb,jsonb,timestamp with time zone,timestamp with time zone,jsonb)'::regprocedure AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='96d0ec527481490a53e4cbf70e3e34458be936fb10e72d9c125f0ae47d4a5ad9' AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef IS TRUE AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on']::text[] AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_bolsa_llamamientos.publicar_oferta_v5(text,text,text,text,text,jsonb,jsonb,timestamp with time zone,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer,jsonb)'::regprocedure AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='3811f4fbb392f0b8c37f2583ae163e0aefafdf71fa34e2f7fc62454ef236d995' AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef IS TRUE AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[] AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_bolsa_llamamientos.recuperar_oferta_operacion_v1(text,text,text,jsonb,integer,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)'::regprocedure AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='1ee7a4fcfc564fa81fda0008775c9e15eac0d55dada447bf2f33edc4caf326f5' AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef IS TRUE AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[] AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}')
 THEN RAISE EXCEPTION 'B136: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE FUNCTION vec_bolsa_llamamientos.huella_operacion_oferta_v6(
 p_prefijo text,p_bolsa text,p_clave text,p_actor text,p_numero integer,
 p_datos jsonb,p_origen jsonb,p_notificacion jsonb,p_publicada timestamptz,
 p_vence timestamptz,p_plazo jsonb)
RETURNS text LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE v5 text;v_canon text;
BEGIN
 v5:=vec_bolsa_llamamientos.huella_operacion_oferta_v5(
  p_prefijo,p_bolsa,p_clave,p_actor,p_numero,p_datos,p_origen,p_notificacion,
  p_publicada,p_vence,p_plazo);
 IF p_origen IS NULL OR jsonb_typeof(p_datos) IS DISTINCT FROM 'object'
    OR NOT (p_datos ?& ARRAY['modalidad_clave','causa_clave'])
    OR jsonb_typeof(p_datos->'modalidad_clave') IS DISTINCT FROM 'string'
    OR jsonb_typeof(p_datos->'causa_clave') IS DISTINCT FROM 'string'
    OR coalesce(p_datos->>'modalidad_clave' ~ '^[a-z][a-z0-9._-]{1,79}$',false) IS NOT TRUE
    OR coalesce(p_datos->>'causa_clave' ~ '^[a-z][a-z0-9._-]{1,79}$',false) IS NOT TRUE THEN
  RAISE EXCEPTION 'B136: par de modalidad y causa inválido' USING ERRCODE='22023'; END IF;
 SELECT string_agg(octet_length(convert_to(x.valor,'UTF8'))::text||':'||x.valor,'' ORDER BY x.ord)
 INTO v_canon FROM unnest(ARRAY['vec.bolsa.oferta.canon.v6',v5,
  p_datos->>'modalidad_clave',p_datos->>'causa_clave']) WITH ORDINALITY x(valor,ord);
 RETURN encode(sha256(convert_to(v_canon,'UTF8')),'hex');
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.huella_operacion_oferta_v6(
 text,text,text,text,integer,jsonb,jsonb,jsonb,timestamptz,timestamptz,jsonb) FROM PUBLIC;
DO $patch$ DECLARE v_def text; BEGIN
 SELECT pg_get_functiondef('vec_bolsa_llamamientos.publicar_oferta_v5(text,text,text,text,text,jsonb,jsonb,timestamp with time zone,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer,jsonb)'::regprocedure) INTO v_def;
 IF (length(v_def)-length(replace(v_def,$m08d1fco0$-- El contrato histórico B71 mantiene los mismos bytes, columnas y recibos.$m08d1fco0$,'')))<>length($m08d1fco0$-- El contrato histórico B71 mantiene los mismos bytes, columnas y recibos.$m08d1fco0$) THEN RAISE EXCEPTION 'F1-11: marca SQL incompatible: publicar_oferta_v5/0' USING ERRCODE='55000'; END IF;
 v_def:=replace(v_def,$m08d1fco0$-- El contrato histórico B71 mantiene los mismos bytes, columnas y recibos.$m08d1fco0$,$m08d1fcn0$-- El contrato histórico B71 mantiene los mismos bytes, columnas y recibos.
 IF (p_datos ? 'modalidad_clave') IS DISTINCT FROM (p_datos ? 'causa_clave')
    OR ((p_datos ? 'modalidad_clave') AND
      (p_origen IS NULL
       OR jsonb_typeof(p_datos->'modalidad_clave') IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_datos->'causa_clave') IS DISTINCT FROM 'string'
       OR coalesce(p_datos->>'modalidad_clave' ~ '^[a-z][a-z0-9._-]{1,79}$',false) IS NOT TRUE
       OR coalesce(p_datos->>'causa_clave' ~ '^[a-z][a-z0-9._-]{1,79}$',false) IS NOT TRUE)) THEN
  RAISE EXCEPTION 'B136: modalidad y causa incompletas' USING ERRCODE='22023'; END IF;$m08d1fcn0$);
 IF (length(v_def)-length(replace(v_def,$m08d1fco1$WHERE k NOT IN ('categoria','centro','fecha_inicio','fecha_fin','descripcion')$m08d1fco1$,'')))<>length($m08d1fco1$WHERE k NOT IN ('categoria','centro','fecha_inicio','fecha_fin','descripcion')$m08d1fco1$) THEN RAISE EXCEPTION 'F1-11: marca SQL incompatible: publicar_oferta_v5/1' USING ERRCODE='55000'; END IF;
 v_def:=replace(v_def,$m08d1fco1$WHERE k NOT IN ('categoria','centro','fecha_inicio','fecha_fin','descripcion')$m08d1fco1$,$m08d1fcn1$WHERE k NOT IN ('categoria','centro','fecha_inicio','fecha_fin','descripcion','modalidad_clave','causa_clave')$m08d1fcn1$);
 IF (length(v_def)-length(replace(v_def,$m08d1fco2$h:=vec_bolsa_llamamientos.huella_operacion_oferta_v5(
  'vec.bolsa.oferta.publicar.v1',p_bolsa,p_clave,p_actor,p_numero_plazas,
  p_datos,p_origen,NULL,p_publicada,p_vence,p_plazo);$m08d1fco2$,'')))<>length($m08d1fco2$h:=vec_bolsa_llamamientos.huella_operacion_oferta_v5(
  'vec.bolsa.oferta.publicar.v1',p_bolsa,p_clave,p_actor,p_numero_plazas,
  p_datos,p_origen,NULL,p_publicada,p_vence,p_plazo);$m08d1fco2$) THEN RAISE EXCEPTION 'F1-11: marca SQL incompatible: publicar_oferta_v5/2' USING ERRCODE='55000'; END IF;
 v_def:=replace(v_def,$m08d1fco2$h:=vec_bolsa_llamamientos.huella_operacion_oferta_v5(
  'vec.bolsa.oferta.publicar.v1',p_bolsa,p_clave,p_actor,p_numero_plazas,
  p_datos,p_origen,NULL,p_publicada,p_vence,p_plazo);$m08d1fco2$,$m08d1fcn2$IF p_datos ? 'modalidad_clave' THEN
  h:=vec_bolsa_llamamientos.huella_operacion_oferta_v6(
   'vec.bolsa.oferta.publicar.v1',p_bolsa,p_clave,p_actor,p_numero_plazas,
   p_datos,p_origen,NULL,p_publicada,p_vence,p_plazo);
 ELSE
  h:=vec_bolsa_llamamientos.huella_operacion_oferta_v5(
   'vec.bolsa.oferta.publicar.v1',p_bolsa,p_clave,p_actor,p_numero_plazas,
   p_datos,p_origen,NULL,p_publicada,p_vence,p_plazo);
 END IF;$m08d1fcn2$);
 IF (length(v_def)-length(replace(v_def,$m08d1fco3$v_comando:=encode(sha256(convert_to(p_bolsa||chr(31)||p_datos::text,'UTF8')),'hex');$m08d1fco3$,'')))<>length($m08d1fco3$v_comando:=encode(sha256(convert_to(p_bolsa||chr(31)||p_datos::text,'UTF8')),'hex');$m08d1fco3$) THEN RAISE EXCEPTION 'F1-11: marca SQL incompatible: publicar_oferta_v5/3' USING ERRCODE='55000'; END IF;
 v_def:=replace(v_def,$m08d1fco3$v_comando:=encode(sha256(convert_to(p_bolsa||chr(31)||p_datos::text,'UTF8')),'hex');$m08d1fco3$,$m08d1fcn3$IF p_datos ? 'modalidad_clave' THEN
  PERFORM 1 FROM vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(
   p_origen->>'expediente_ref',p_origen->>'categoria_ref',
   p_origen->>'organizacion_ref',(p_origen->>'version_expediente')::bigint) x
  WHERE x.modalidad_clave=p_datos->>'modalidad_clave'
    AND x.causa_clave=p_datos->>'causa_clave'
    AND x.version_expediente=(p_origen->>'version_expediente')::bigint;
  IF NOT FOUND THEN RAISE EXCEPTION 'B136: modalidad o causa distinta del análisis vigente' USING ERRCODE='42501'; END IF;
 END IF;
 v_comando:=encode(sha256(convert_to(p_bolsa||chr(31)||p_datos::text,'UTF8')),'hex');$m08d1fcn3$);
 EXECUTE v_def;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_bolsa_llamamientos.publicar_oferta_v5(text,text,text,text,text,jsonb,jsonb,timestamp with time zone,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer,jsonb)'::regprocedure AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='e93537e69893580cd4bf1dc51cb7768112fa8e39fa595a30e22f941336fb8e72' AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef IS TRUE AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[] AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}') THEN RAISE EXCEPTION 'F1-11: postimagen SQL incompatible: publicar_oferta_v5' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef('vec_bolsa_llamamientos.recuperar_oferta_operacion_v1(text,text,text,jsonb,integer,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)'::regprocedure) INTO v_def;
 IF (length(v_def)-length(replace(v_def,$ma9d43ao0$h:=vec_bolsa_llamamientos.huella_operacion_oferta_v5(
  'vec.bolsa.oferta.recuperar.v1',p_bolsa,p_clave,p_actor,p_numero_plazas,p_datos,p_origen,p_notificacion,NULL,NULL,NULL);$ma9d43ao0$,'')))<>length($ma9d43ao0$h:=vec_bolsa_llamamientos.huella_operacion_oferta_v5(
  'vec.bolsa.oferta.recuperar.v1',p_bolsa,p_clave,p_actor,p_numero_plazas,p_datos,p_origen,p_notificacion,NULL,NULL,NULL);$ma9d43ao0$) THEN RAISE EXCEPTION 'F1-11: marca SQL incompatible: recuperar_oferta_operacion_v1/0' USING ERRCODE='55000'; END IF;
 v_def:=replace(v_def,$ma9d43ao0$h:=vec_bolsa_llamamientos.huella_operacion_oferta_v5(
  'vec.bolsa.oferta.recuperar.v1',p_bolsa,p_clave,p_actor,p_numero_plazas,p_datos,p_origen,p_notificacion,NULL,NULL,NULL);$ma9d43ao0$,$ma9d43an0$IF (p_datos ? 'modalidad_clave') IS DISTINCT FROM (p_datos ? 'causa_clave')
    OR ((p_datos ? 'modalidad_clave') AND
      (p_origen IS NULL
       OR jsonb_typeof(p_datos->'modalidad_clave') IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_datos->'causa_clave') IS DISTINCT FROM 'string'
       OR coalesce(p_datos->>'modalidad_clave' ~ '^[a-z][a-z0-9._-]{1,79}$',false) IS NOT TRUE
       OR coalesce(p_datos->>'causa_clave' ~ '^[a-z][a-z0-9._-]{1,79}$',false) IS NOT TRUE)) THEN
  RAISE EXCEPTION 'B136: modalidad y causa incompletas' USING ERRCODE='22023'; END IF;
 IF p_datos ? 'modalidad_clave' THEN
  h:=vec_bolsa_llamamientos.huella_operacion_oferta_v6(
   'vec.bolsa.oferta.recuperar.v1',p_bolsa,p_clave,p_actor,p_numero_plazas,
   p_datos,p_origen,p_notificacion,NULL,NULL,NULL);
 ELSE
  h:=vec_bolsa_llamamientos.huella_operacion_oferta_v5(
   'vec.bolsa.oferta.recuperar.v1',p_bolsa,p_clave,p_actor,p_numero_plazas,
   p_datos,p_origen,p_notificacion,NULL,NULL,NULL);
 END IF;$ma9d43an0$);
 EXECUTE v_def;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_bolsa_llamamientos.recuperar_oferta_operacion_v1(text,text,text,jsonb,integer,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)'::regprocedure AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='b5eaaefceb2c988b670afdb3a89ffef129fb226ef173e8c9c8885215efcf023c' AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef IS TRUE AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[] AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}') THEN RAISE EXCEPTION 'F1-11: postimagen SQL incompatible: recuperar_oferta_operacion_v1' USING ERRCODE='55000'; END IF;
END $patch$;
DO $post$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_bolsa_llamamientos.huella_operacion_oferta_v6(text,text,text,text,integer,jsonb,jsonb,jsonb,timestamptz,timestamptz,jsonb)'::regprocedure
  AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on']::text[]
  AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}')
 THEN RAISE EXCEPTION 'B136: helper sin ACL o configuración esperada' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
