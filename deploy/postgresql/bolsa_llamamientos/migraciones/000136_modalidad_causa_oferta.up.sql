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
-- Definición final obtenida de la copia ensayada; la preimagen anterior limita su aplicación.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.publicar_oferta_v5(p_oferta text, p_recibo text, p_bolsa text, p_actor text, p_clave text, p_datos jsonb, p_plazo jsonb, p_publicada timestamp with time zone, p_vence timestamp with time zone, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea, p_unidad text, p_ambito text, p_numero_plazas integer, p_origen jsonb)
 RETURNS TABLE(oferta jsonb, reutilizada boolean)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE h text;ctx text;d jsonb;c record;previa record;plazas record;origen_previo record;
 v_ahora timestamptz;v_recibo text;v_evento text;v_elegibles integer;v_comando text;v_version bigint;
 v_constraint text;v_constraint_schema text;v_constraint_table text;
BEGIN
 -- El contrato histórico B71 mantiene los mismos bytes, columnas y recibos.
 IF (p_datos ? 'modalidad_clave') IS DISTINCT FROM (p_datos ? 'causa_clave')
    OR ((p_datos ? 'modalidad_clave') AND
      (p_origen IS NULL
       OR jsonb_typeof(p_datos->'modalidad_clave') IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_datos->'causa_clave') IS DISTINCT FROM 'string'
       OR coalesce(p_datos->>'modalidad_clave' ~ '^[a-z][a-z0-9._-]{1,79}$',false) IS NOT TRUE
       OR coalesce(p_datos->>'causa_clave' ~ '^[a-z][a-z0-9._-]{1,79}$',false) IS NOT TRUE)) THEN
  RAISE EXCEPTION 'B136: modalidad y causa incompletas' USING ERRCODE='22023'; END IF;
 IF p_plazo->>'inicio'='notificacion' AND p_origen IS NULL THEN
  RETURN QUERY SELECT x.oferta,x.reutilizada FROM vec_bolsa_llamamientos.publicar_oferta_v4(
   p_oferta,p_recibo,p_bolsa,p_actor,p_clave,p_datos,p_plazo-'inicio',p_publicada,p_vence,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
   p_payload,p_sobre,p_evidencia,p_raiz,p_unidad,p_ambito,p_numero_plazas) x;
  RETURN;
 END IF;
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER') IS TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_plazo->>'inicio' IS DISTINCT FROM 'publicacion'
    OR p_plazo ? 'notificacion'
    OR p_oferta !~ '^oferta:[0-9a-f]{64}$'
    OR p_recibo !~ '^recibo:oferta:[0-9a-f]{64}$'
    OR p_bolsa IS NULL OR octet_length(p_bolsa) NOT BETWEEN 1 AND 256
    OR position(chr(31) in p_bolsa)>0 OR position(chr(31) in p_clave)>0
    OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR p_datos IS NULL OR jsonb_typeof(p_datos) IS DISTINCT FROM 'object'
    OR NOT (p_datos ?& ARRAY['categoria','centro','fecha_inicio','descripcion'])
    OR EXISTS(SELECT 1 FROM jsonb_object_keys(p_datos) k
       WHERE k NOT IN ('categoria','centro','fecha_inicio','fecha_fin','descripcion','modalidad_clave','causa_clave'))
    OR EXISTS(SELECT 1 FROM jsonb_each(p_datos) e
       WHERE jsonb_typeof(e.value)<>'string' OR octet_length(e.value#>>'{}') NOT BETWEEN 2 AND 2000
         OR e.value#>>'{}'<>btrim(e.value#>>'{}'))
    OR p_datos->>'fecha_inicio' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
    OR (p_datos ? 'fecha_fin' AND (p_datos->>'fecha_fin' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
       OR p_datos->>'fecha_fin'<p_datos->>'fecha_inicio'))
    OR p_numero_plazas NOT BETWEEN 1 AND 100
    OR p_unidad IS NULL OR p_ambito IS NULL
    OR p_unidad !~ '^[A-Za-z0-9:_-]+$' OR p_ambito !~ '^[A-Za-z0-9:_-]+$'
    OR p_publicada IS NULL OR p_vence IS NULL OR p_vence<=p_publicada
    OR p_vence>p_publicada+interval '120 days' THEN
  RAISE EXCEPTION 'B117: publicación inválida' USING ERRCODE='22023';
 END IF;
 v_recibo:='recibo:oferta:'||encode(sha256(convert_to(p_bolsa||chr(31)||p_clave,'UTF8')),'hex');
 IF p_recibo IS DISTINCT FROM v_recibo OR p_oferta IS DISTINCT FROM replace(v_recibo,'recibo:oferta:','oferta:') THEN
  RAISE EXCEPTION 'B117: referencias no ligadas a la clave' USING ERRCODE='22023';
 END IF;
 IF p_datos ? 'modalidad_clave' THEN
  h:=vec_bolsa_llamamientos.huella_operacion_oferta_v6(
   'vec.bolsa.oferta.publicar.v1',p_bolsa,p_clave,p_actor,p_numero_plazas,
   p_datos,p_origen,NULL,p_publicada,p_vence,p_plazo);
 ELSE
  h:=vec_bolsa_llamamientos.huella_operacion_oferta_v5(
   'vec.bolsa.oferta.publicar.v1',p_bolsa,p_clave,p_actor,p_numero_plazas,
   p_datos,p_origen,NULL,p_publicada,p_vence,p_plazo);
 END IF;
 ctx:=encode(sha256(convert_to(
  '{"ambitos":{"ambito_ref":"'||p_ambito||'","unidad_ref":"'||p_unidad||
  '"},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B117: decisión ilegible' USING ERRCODE='42501'; END;
 IF d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM ctx
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_llamamientos_bolsa'
    OR d->>'recurso_ref' IS DISTINCT FROM p_bolsa
    OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida' THEN
  RAISE EXCEPTION 'B117: decisión no ligada a la publicación' USING ERRCODE='42501';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:oferta:'||p_bolsa||':'||p_clave,0));
 SELECT * INTO STRICT c FROM vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF c.efecto_ref IS DISTINCT FROM p_bolsa OR c.huella_efecto_sha256 IS DISTINCT FROM ctx
    OR c.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'B117: consumo de publicación divergente' USING ERRCODE='42501';
 END IF;
 SELECT * INTO previa FROM vec_bolsa_llamamientos.oferta_publicada o
 WHERE o.bolsa_ref=p_bolsa AND o.clave_idempotencia=p_clave;
 IF FOUND THEN
  SELECT * INTO plazas FROM vec_bolsa_llamamientos.plazas_oferta p WHERE p.oferta_ref=previa.oferta_ref;
  SELECT * INTO origen_previo FROM vec_bolsa_llamamientos.origen_oferta_ct_v1 x
   WHERE x.oferta_ref=previa.oferta_ref;
  IF previa.oferta_ref IS DISTINCT FROM p_oferta OR previa.recibo_ref IS DISTINCT FROM p_recibo
     OR previa.actor_ref IS DISTINCT FROM p_actor OR previa.datos IS DISTINCT FROM p_datos
     OR previa.plazo->>'inicio' IS DISTINCT FROM 'publicacion'
     OR plazas.numero_plazas IS DISTINCT FROM p_numero_plazas
     OR (p_origen IS NULL) IS DISTINCT FROM (origen_previo.oferta_ref IS NULL)
     OR (p_origen IS NOT NULL AND (origen_previo.origen_canonico IS DISTINCT FROM p_origen
       OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.origen_oferta_ct_outbox_v1 ev
         WHERE ev.evento_ref=origen_previo.evento_ref AND ev.oferta_ref=previa.oferta_ref))) THEN
   RAISE EXCEPTION 'B117: clave reutilizada con otro cuerpo u origen' USING ERRCODE='VBO01';
  END IF;
  RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(previa.oferta_ref,clock_timestamp()),true;
  RETURN;
 END IF;
 v_ahora:=clock_timestamp();
 IF p_publicada>v_ahora+interval '1 minute' OR p_publicada<v_ahora-interval '5 minutes'
    OR p_vence<=v_ahora OR p_publicada<>date_trunc('microseconds',p_publicada) THEN
  RAISE EXCEPTION 'B117: publicación fuera de plazo' USING ERRCODE='22023';
 END IF;
 IF jsonb_typeof(p_plazo) IS DISTINCT FROM 'object'
    OR (p_plazo->>'unidad'='horas_naturales' AND
      (SELECT count(*) FROM jsonb_object_keys(p_plazo))<>13)
    OR (p_plazo->>'unidad'<>'horas_naturales' AND
      (SELECT count(*) FROM jsonb_object_keys(p_plazo))<>11)
    OR NOT (p_plazo ?& ARRAY['inicio','regla_ref','huella_catalogo','unidad','cantidad',
      'computo','ultimo_dia','ejemplo','calendarios','politica_version','municipio_sede'])
    OR coalesce(p_plazo->>'politica_version','') !~ '^[1-9][0-9]{0,9}$' THEN
  RAISE EXCEPTION 'B117: material de plazo incompleto' USING ERRCODE='22023';
 END IF;
 v_version:=(p_plazo->>'politica_version')::bigint;
 IF p_numero_plazas>1 AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.politica_ofertas_version pv
     WHERE pv.bolsa_ref=p_bolsa AND pv.version=v_version AND jsonb_typeof(pv.politica->'plazas')='object') THEN
  RAISE EXCEPTION 'B117: política sin varias plazas' USING ERRCODE='VBO08';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion co
   JOIN vec_bolsa_llamamientos.bolsa_constituida b
     ON b.bolsa_ref=co.bolsa_ref AND b.version=co.version_bolsa
   WHERE co.bolsa_ref=p_bolsa AND b.estado='vigente' AND b.vigente_desde<=p_publicada
     AND (b.vigente_hasta IS NULL OR p_publicada<b.vigente_hasta)) THEN
  RAISE EXCEPTION 'B117: bolsa no vigente' USING ERRCODE='23503';
 END IF;
 IF p_origen IS NOT NULL THEN
  IF p_origen->>'unidad_ref' IS DISTINCT FROM p_unidad THEN
   RAISE EXCEPTION 'B117: unidad del origen distinta' USING ERRCODE='42501';
  END IF;
  v_elegibles:=vec_bolsa_llamamientos.validar_origen_oferta_ct_v1(p_origen,p_bolsa,p_numero_plazas,p_publicada);
  PERFORM pg_advisory_xact_lock(hashtextextended(
   'vec_bolsa_llamamientos:origen-oferta-ct:'||
    jsonb_build_array(p_origen->>'expediente_ref',p_origen->>'clave_ct')::text,0));
  IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.origen_oferta_ct_v1 x
    WHERE x.expediente_ref=p_origen->>'expediente_ref'
      AND x.clave_ct=p_origen->>'clave_ct') THEN
   RAISE EXCEPTION 'B117: clave CT ya publicada para el expediente' USING ERRCODE='VBO01';
  END IF;
 END IF;
 IF p_datos ? 'modalidad_clave' THEN
  PERFORM 1 FROM vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(
   p_origen->>'expediente_ref',p_origen->>'categoria_ref',
   p_origen->>'organizacion_ref',(p_origen->>'version_expediente')::bigint) x
  WHERE x.modalidad_clave=p_datos->>'modalidad_clave'
    AND x.causa_clave=p_datos->>'causa_clave'
    AND x.version_expediente=(p_origen->>'version_expediente')::bigint;
  IF NOT FOUND THEN RAISE EXCEPTION 'B136: modalidad o causa distinta del análisis vigente' USING ERRCODE='42501'; END IF;
 END IF;
 v_comando:=encode(sha256(convert_to(p_bolsa||chr(31)||p_datos::text,'UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.oferta_publicada(
  oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,
  publicada_en,vence_antes_de,huella_comando_sha256,decision_ref)
 VALUES(p_oferta,p_recibo,p_bolsa,p_actor,p_clave,p_datos,p_plazo,
  p_publicada,p_vence,v_comando,c.decision_ref);
 INSERT INTO vec_bolsa_llamamientos.plazas_oferta(
  oferta_ref,bolsa_ref,numero_plazas,politica_version,registrada_en)
 VALUES(p_oferta,p_bolsa,p_numero_plazas,v_version,clock_timestamp());
 IF p_origen IS NOT NULL THEN
  v_evento:='evento:bolsa:oferta-ct:'||encode(sha256(convert_to(
   (p_origen->>'expediente_ref')||chr(31)||(p_origen->>'clave_ct'),'UTF8')),'hex');
  INSERT INTO vec_bolsa_llamamientos.origen_oferta_ct_v1(
   oferta_ref,recibo_ref,organizacion_ref,unidad_ref,expediente_ref,version_expediente,
   clave_ct,necesidad_ref,acta_ref,categoria_ref,bolsa_ref,version_bolsa,
   huella_bolsa_sha256,instantanea_ref,version_instantanea,huella_instantanea_sha256,
   politica_ref,version_politica,huella_politica_sha256,numero_plazas,elegibles,
   origen_canonico,origen_huella_sha256,decision_ref,auditoria_ref,evento_ref,publicada_en)
  VALUES(p_oferta,p_recibo,p_origen->>'organizacion_ref',p_origen->>'unidad_ref',
   p_origen->>'expediente_ref',(p_origen->>'version_expediente')::bigint,
   p_origen->>'clave_ct',p_origen->>'necesidad_ref',p_origen->>'acta_ref',
   p_origen->>'categoria_ref',p_bolsa,(p_origen->>'version_bolsa')::bigint,
   p_origen->>'huella_bolsa_sha256',p_origen->>'instantanea_ref',
   (p_origen->>'version_instantanea')::bigint,p_origen->>'huella_instantanea_sha256',
   p_origen->>'politica_ref',(p_origen->>'version_politica')::bigint,
   p_origen->>'huella_politica_sha256',p_numero_plazas,v_elegibles,p_origen,
   encode(sha256(convert_to(p_origen::text,'UTF8')),'hex'),
   c.decision_ref,c.auditoria_ref,v_evento,p_publicada);
  INSERT INTO vec_bolsa_llamamientos.origen_oferta_ct_outbox_v1(
   evento_ref,oferta_ref,tipo,estado,creada_en)
  VALUES(v_evento,p_oferta,'bolsa.oferta.ct.origen_confirmado','pendiente',p_publicada);
 END IF;
 RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,clock_timestamp()),false;
EXCEPTION WHEN unique_violation THEN
 GET STACKED DIAGNOSTICS v_constraint=CONSTRAINT_NAME,
  v_constraint_schema=SCHEMA_NAME,v_constraint_table=TABLE_NAME;
 IF v_constraint_schema='vec_bolsa_llamamientos'
    AND v_constraint_table='origen_oferta_ct_v1'
    AND v_constraint IN ('origen_oferta_ct_v1_expediente_ref_clave_ct_key',
      'origen_oferta_ct_v1_evento_ref_key') THEN
  RAISE EXCEPTION 'B117: clave CT ya publicada para el expediente' USING ERRCODE='VBO01';
 END IF;
 RAISE;
END $function$;

-- Definición final obtenida de la copia ensayada; la preimagen anterior limita su aplicación.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.recuperar_oferta_operacion_v1(p_bolsa text, p_actor text, p_clave text, p_datos jsonb, p_numero_plazas integer, p_origen jsonb, p_notificacion jsonb, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea, p_unidad text, p_ambito text)
 RETURNS TABLE(oferta jsonb, encontrada boolean)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE h text;ctx text;d jsonb;c record;o record;origen record;plazas record;recibo text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER') IS TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_bolsa IS NULL OR position(chr(31) in p_bolsa)>0 OR position(chr(31) in p_clave)>0 OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR p_unidad IS NULL OR p_unidad !~ '^[A-Za-z0-9:_-]+$'
    OR p_ambito IS NULL OR p_ambito !~ '^[A-Za-z0-9:_-]+$' THEN
  RAISE EXCEPTION 'B117: recuperación denegada' USING ERRCODE='42501';
 END IF;
 IF (p_datos ? 'modalidad_clave') IS DISTINCT FROM (p_datos ? 'causa_clave')
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
 END IF;
 ctx:=encode(sha256(convert_to(
  '{"ambitos":{"ambito_ref":"'||p_ambito||'","unidad_ref":"'||p_unidad||
  '"},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B117: decisión ilegible' USING ERRCODE='42501'; END;
 IF d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM ctx
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_llamamientos_bolsa'
    OR d->>'recurso_ref' IS DISTINCT FROM p_bolsa
    OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida' THEN
  RAISE EXCEPTION 'B117: recuperación no ligada a la intención' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT c FROM vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF c.efecto_ref IS DISTINCT FROM p_bolsa OR c.huella_efecto_sha256 IS DISTINCT FROM ctx
    OR c.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'B117: consumo de recuperación divergente' USING ERRCODE='42501';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:oferta:'||p_bolsa||':'||p_clave,0));
 SELECT * INTO o FROM vec_bolsa_llamamientos.oferta_publicada x
  WHERE x.bolsa_ref=p_bolsa AND x.clave_idempotencia=p_clave;
 IF NOT FOUND THEN
  RETURN QUERY SELECT NULL::jsonb,false;
  RETURN;
 END IF;
 recibo:='recibo:oferta:'||encode(sha256(convert_to(p_bolsa||chr(31)||p_clave,'UTF8')),'hex');
 SELECT * INTO plazas FROM vec_bolsa_llamamientos.plazas_oferta p WHERE p.oferta_ref=o.oferta_ref;
 SELECT * INTO origen FROM vec_bolsa_llamamientos.origen_oferta_ct_v1 x WHERE x.oferta_ref=o.oferta_ref;
 IF o.actor_ref IS DISTINCT FROM p_actor OR o.recibo_ref IS DISTINCT FROM recibo
    OR o.datos IS DISTINCT FROM p_datos
    OR (p_notificacion IS NULL AND (o.plazo->>'inicio' IS DISTINCT FROM 'publicacion'
       OR o.plazo ? 'notificacion'))
    OR (p_notificacion IS NOT NULL AND o.plazo->'notificacion' IS DISTINCT FROM p_notificacion)
    OR plazas.numero_plazas IS DISTINCT FROM p_numero_plazas
    OR (p_origen IS NULL) IS DISTINCT FROM (origen.oferta_ref IS NULL)
    OR (p_origen IS NOT NULL AND (origen.origen_canonico IS DISTINCT FROM p_origen
      OR origen.recibo_ref IS DISTINCT FROM o.recibo_ref
      OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.origen_oferta_ct_outbox_v1 ev
        WHERE ev.evento_ref=origen.evento_ref AND ev.oferta_ref=o.oferta_ref))) THEN
  RAISE EXCEPTION 'B117: clave reutilizada con otra oferta' USING ERRCODE='VBO01';
 END IF;
 RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(o.oferta_ref,clock_timestamp()),true;
END $function$;
DO $post_funciones$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_bolsa_llamamientos.publicar_oferta_v5(text,text,text,text,text,jsonb,jsonb,timestamp with time zone,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer,jsonb)'::regprocedure
  AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='e93537e69893580cd4bf1dc51cb7768112fa8e39fa595a30e22f941336fb8e72'
  AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef IS TRUE
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[] AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}')
 THEN RAISE EXCEPTION 'F1-11: postimagen incompatible: publicar_oferta_v5' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_bolsa_llamamientos.recuperar_oferta_operacion_v1(text,text,text,jsonb,integer,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)'::regprocedure
  AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='b5eaaefceb2c988b670afdb3a89ffef129fb226ef173e8c9c8885215efcf023c'
  AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef IS TRUE
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[] AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}')
 THEN RAISE EXCEPTION 'F1-11: postimagen incompatible: recuperar_oferta_operacion_v1' USING ERRCODE='55000'; END IF;
END $post_funciones$;
DO $post$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_bolsa_llamamientos.huella_operacion_oferta_v6(text,text,text,text,integer,jsonb,jsonb,jsonb,timestamptz,timestamptz,jsonb)'::regprocedure
  AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on']::text[]
  AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}')
 THEN RAISE EXCEPTION 'B136: helper sin ACL o configuración esperada' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
