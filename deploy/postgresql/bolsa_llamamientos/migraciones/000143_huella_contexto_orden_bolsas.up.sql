\set ON_ERROR_STOP on
-- B143 / F1-13 parte 2: el guardado del orden de bolsas (B104 y B116) vuelve a
-- calcular la huella de su contexto fijo {candidato_ref, propiedad}. Una
-- decisión emitida para guardar modalidades (B138, contexto con
-- tipo_preferencia y material_sha256) comparte acción y audiencia, pero ya no
-- puede canjearse para cambiar el orden de bolsas. Solo añade esa condición:
-- el resto del cuerpo, dueño, configuración y ACL quedan exactos.
-- Instalar después de B138. Solo UP; no se reaplica.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000143',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR current_setting('server_encoding')<>'UTF8'
 OR to_regprocedure('vec_bolsa_llamamientos.huella_contexto_modalidades_v1(text,text)') IS NULL
 OR (SELECT encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') FROM pg_proc p
     WHERE p.oid='vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
      AND p.prokind='f' AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef
      AND p.provolatile='v' AND obj_description(p.oid,'pg_proc') IS NULL
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
      AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_portal_externo=X/vec_bolsa_llamamientos_propietario}')
    IS DISTINCT FROM 'e5be29211ea6892365bcf780798bcbb5962304461ee1351127093e5dfcf4f394'
 OR (SELECT encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') FROM pg_proc p
     WHERE p.oid='vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
      AND p.prokind='f' AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef
      AND p.provolatile='v' AND obj_description(p.oid,'pg_proc') IS NULL
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
      AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_preferencias_interna=X/vec_bolsa_llamamientos_propietario}')
    IS DISTINCT FROM '8e64be00a17eff91385f2486d6dbf830e931c49506db3c701e9324e0bff395bb'
 THEN RAISE EXCEPTION 'B143: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(p_candidato_ref text, p_orden text[], p_version_esperada bigint, p_clave text, p_registrada_en timestamp with time zone, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS TABLE(reutilizada boolean, recibo_ref text, version bigint, orden text[], registrada_en timestamp with time zone)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE c jsonb; d jsonb; x jsonb; v_n integer; v_actuales text[]; v_ordenado text[];
 v_previa vec_bolsa_llamamientos.preferencia_bolsas_version%ROWTYPE;
 v_consumo record; v_version bigint; v_huella text; v_conjunto text; v_recibo text;
 v_ahora timestamptz:=clock_timestamp();
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_orden IS NULL OR array_ndims(p_orden) IS DISTINCT FROM 1 OR cardinality(p_orden) NOT BETWEEN 1 AND 256
    OR array_position(p_orden,NULL) IS NOT NULL
    OR p_version_esperada IS NULL OR p_version_esperada NOT BETWEEN 0 AND 2147483646
    OR p_clave IS NULL OR p_clave !~ '^[A-Za-z0-9._:-]{8,128}$'
    OR p_registrada_en IS NULL OR NOT isfinite(p_registrada_en)
    OR p_registrada_en < v_ahora-interval '5 minutes' OR p_registrada_en > v_ahora+interval '1 minute'
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL
    OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL
 THEN RAISE EXCEPTION 'B104: orden inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B104: material inválido' USING ERRCODE='22023'; END;
 IF c->>'operacion' IS DISTINCT FROM 'bolsa.preferencias_bolsas.ordenar_propias'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec.bolsa.preferencias-bolsas.ordenar.v1'
    OR c->>'efecto_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'participaciones_candidato'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_preferencias_bolsas_propias'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM encode(sha256(convert_to(
     '{"ambitos":{"candidato_ref":'||to_json(p_candidato_ref)::text||'},"atributos":{"propiedad":"candidato"}}',
     'UTF8')),'hex')
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'B104: orden denegado' USING ERRCODE='42501'; END IF;
 SELECT count(*) INTO v_n FROM jsonb_array_elements(x->'vinculos') e
 WHERE e->>'tipo'='candidato' AND e->>'estado'='activo';
 IF v_n<>1 OR NOT EXISTS (
  SELECT 1 FROM jsonb_array_elements(x->'vinculos') e
  WHERE e->>'tipo'='candidato' AND e->>'estado'='activo' AND e->>'referencia'=p_candidato_ref)
 THEN RAISE EXCEPTION 'B104: candidato denegado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_consumo
 FROM vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
 THEN RAISE EXCEPTION 'B104: consumo denegado' USING ERRCODE='42501'; END IF;

 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:preferencias_bolsas:'||p_candidato_ref,0));
 IF EXISTS (SELECT 1 FROM unnest(p_orden) AS u(ref)
  WHERE u.ref IS NULL OR NOT vec_bolsa_llamamientos.constitucion_texto_valido(u.ref,512)) THEN
  RAISE EXCEPTION 'B104: referencia inválida' USING ERRCODE='22023'; END IF;
 -- La huella usa separador de control prohibido en las referencias canónicas.
 v_huella:=encode(sha256(convert_to(p_candidato_ref||chr(31)||p_version_esperada::text||chr(31)||
  array_to_string(p_orden,chr(31)),'UTF8')),'hex');
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.preferencia_bolsas_version p
 WHERE p.candidato_ref=p_candidato_ref AND p.clave_idempotencia=p_clave;
 IF FOUND THEN
  IF v_previa.solicitud_huella_sha256<>v_huella OR v_previa.version<>p_version_esperada+1
  THEN RAISE EXCEPTION 'B104: clave reutilizada con otro orden' USING ERRCODE='VBP02'; END IF;
  RETURN QUERY SELECT true,v_previa.recibo_ref,v_previa.version,v_previa.orden,v_previa.registrada_en;
  RETURN;
 END IF;
 SELECT coalesce(max(p.version),0) INTO v_version FROM vec_bolsa_llamamientos.preferencia_bolsas_version p
 WHERE p.candidato_ref=p_candidato_ref;
 IF v_version<>p_version_esperada THEN
  RAISE EXCEPTION 'B104: versión de preferencias desactualizada' USING ERRCODE='VBP01';
 END IF;
 v_actuales:=vec_bolsa_llamamientos.preferencias_bolsas_actuales_interna_v1(p_candidato_ref,v_ahora);
 IF cardinality(v_actuales)<>cardinality(p_orden) THEN
  RAISE EXCEPTION 'B104: conjunto de bolsas cambiado' USING ERRCODE='VBP03'; END IF;
 SELECT array_agg(u.ref ORDER BY u.ref) INTO v_ordenado FROM unnest(p_orden) AS u(ref);
 IF v_ordenado IS DISTINCT FROM v_actuales THEN
  RAISE EXCEPTION 'B104: orden incompleto, duplicado o ajeno' USING ERRCODE='VBP03'; END IF;
 v_conjunto:=encode(sha256(convert_to(array_to_string(v_actuales,chr(31)),'UTF8')),'hex');
 v_recibo:='recibo:preferencias-bolsas:'||encode(sha256(convert_to(
  p_candidato_ref||chr(31)||p_clave||chr(31)||v_huella,'UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.preferencia_bolsas_version(
  candidato_ref,version,orden,conjunto_huella_sha256,clave_idempotencia,
  solicitud_huella_sha256,recibo_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(p_candidato_ref,v_version+1,p_orden,v_conjunto,p_clave,v_huella,v_recibo,
  v_consumo.decision_ref,v_consumo.auditoria_ref,v_ahora);
 INSERT INTO vec_bolsa_llamamientos.preferencia_bolsas_outbox(
  evento_ref,candidato_ref,version,recibo_ref,decision_ref,auditoria_ref,esquema,evento,creada_en)
 VALUES('evento:preferencias-bolsas:'||substr(v_recibo,length('recibo:preferencias-bolsas:')+1),
  p_candidato_ref,v_version+1,v_recibo,v_consumo.decision_ref,v_consumo.auditoria_ref,
  'vec.bolsa.preferencias.ordenadas.v1',
  jsonb_build_object('candidato_ref',p_candidato_ref,'version',v_version+1,
   'orden',p_orden,'conjunto_huella_sha256',v_conjunto,'recibo_ref',v_recibo,
   'decision_ref',v_consumo.decision_ref,'auditoria_ref',v_consumo.auditoria_ref),v_ahora);
 RETURN QUERY SELECT false,v_recibo,v_version+1,p_orden,v_ahora;
END $function$;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(p_candidato_ref text, p_orden text[], p_version_esperada bigint, p_clave text, p_registrada_en timestamp with time zone, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS TABLE(reutilizada boolean, recibo_ref text, version bigint, orden text[], registrada_en timestamp with time zone)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE c jsonb; d jsonb; x jsonb; v_n integer; v_actuales text[]; v_ordenado text[];
 v_previa vec_bolsa_llamamientos.preferencia_bolsas_version%ROWTYPE;
 v_consumo record; v_version bigint; v_huella text; v_conjunto text; v_recibo text; v_clave_interna text;
 v_ahora timestamptz:=clock_timestamp();
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR session_user <> 'vec_interno_bolsa_preferencias_desarrollo'
    OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_orden IS NULL OR array_ndims(p_orden) IS DISTINCT FROM 1 OR cardinality(p_orden) NOT BETWEEN 1 AND 256
    OR array_position(p_orden,NULL) IS NOT NULL
    OR p_version_esperada IS NULL OR p_version_esperada NOT BETWEEN 0 AND 2147483646
    OR p_clave IS NULL OR p_clave !~ '^[A-Za-z0-9._:-]{8,128}$'
    OR p_registrada_en IS NULL OR NOT isfinite(p_registrada_en)
    OR p_registrada_en < v_ahora-interval '5 minutes' OR p_registrada_en > v_ahora+interval '1 minute'
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL
    OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL
 THEN RAISE EXCEPTION 'B116: orden inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B116: material inválido' USING ERRCODE='22023'; END;
 IF c->>'operacion' IS DISTINCT FROM 'bolsa.preferencias_bolsas.ordenar_propias'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec.bolsa.preferencias-bolsas.ordenar.interna.v1'
    OR c->>'efecto_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'participaciones_candidato'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_preferencias_bolsas_propias'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM encode(sha256(convert_to(
     '{"ambitos":{"candidato_ref":'||to_json(p_candidato_ref)::text||'},"atributos":{"propiedad":"candidato"}}',
     'UTF8')),'hex')
    OR d->>'principal_id' IS DISTINCT FROM x->>'principal_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM x->>'perfil_activo_ref'
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'B116: orden denegado' USING ERRCODE='42501'; END IF;
 SELECT count(*) INTO v_n FROM jsonb_array_elements(x->'vinculos') e
 WHERE e->>'tipo'='candidato' AND e->>'estado'='activo';
 IF v_n<>1 OR NOT EXISTS (
  SELECT 1 FROM jsonb_array_elements(x->'vinculos') e
  WHERE e->>'tipo'='candidato' AND e->>'estado'='activo' AND e->>'referencia'=p_candidato_ref)
 THEN RAISE EXCEPTION 'B116: candidato denegado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_consumo
 FROM vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
 THEN RAISE EXCEPTION 'B116: consumo denegado' USING ERRCODE='42501'; END IF;

 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:preferencias_bolsas:'||p_candidato_ref,0));
 IF EXISTS (SELECT 1 FROM unnest(p_orden) AS u(ref)
  WHERE u.ref IS NULL OR NOT vec_bolsa_llamamientos.constitucion_texto_valido(u.ref,512)) THEN
  RAISE EXCEPTION 'B116: referencia inválida' USING ERRCODE='22023'; END IF;
 -- La huella usa separador de control prohibido en las referencias canónicas.
 v_huella:=encode(sha256(convert_to(p_candidato_ref||chr(31)||p_version_esperada::text||chr(31)||
  array_to_string(p_orden,chr(31)),'UTF8')),'hex');
 v_clave_interna:='@'||encode(sha256(convert_to('interna_corporativa'||chr(31)||p_clave,'UTF8')),'hex');
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.preferencia_bolsas_version p
 WHERE p.candidato_ref=p_candidato_ref AND p.clave_idempotencia=v_clave_interna;
 IF FOUND THEN
  IF v_previa.actor_ref_interno IS DISTINCT FROM x->>'principal_ref'
   OR v_previa.perfil_ref_interno IS DISTINCT FROM x->>'perfil_activo_ref'
   OR v_previa.solicitud_huella_sha256<>v_huella OR v_previa.version<>p_version_esperada+1
  THEN RAISE EXCEPTION 'B116: clave reutilizada con otro orden' USING ERRCODE='VBP02'; END IF;
  RETURN QUERY SELECT true,v_previa.recibo_ref,v_previa.version,v_previa.orden,v_previa.registrada_en;
  RETURN;
 END IF;
 SELECT coalesce(max(p.version),0) INTO v_version FROM vec_bolsa_llamamientos.preferencia_bolsas_version p
 WHERE p.candidato_ref=p_candidato_ref;
 IF v_version<>p_version_esperada THEN
  RAISE EXCEPTION 'B116: versión de preferencias desactualizada' USING ERRCODE='VBP01';
 END IF;
 v_actuales:=vec_bolsa_llamamientos.preferencias_bolsas_actuales_interna_v1(p_candidato_ref,v_ahora);
 IF cardinality(v_actuales)<>cardinality(p_orden) THEN
  RAISE EXCEPTION 'B116: conjunto de bolsas cambiado' USING ERRCODE='VBP03'; END IF;
 SELECT array_agg(u.ref ORDER BY u.ref) INTO v_ordenado FROM unnest(p_orden) AS u(ref);
 IF v_ordenado IS DISTINCT FROM v_actuales THEN
  RAISE EXCEPTION 'B116: orden incompleto, duplicado o ajeno' USING ERRCODE='VBP03'; END IF;
 v_conjunto:=encode(sha256(convert_to(array_to_string(v_actuales,chr(31)),'UTF8')),'hex');
 v_recibo:='recibo:preferencias-bolsas:'||encode(sha256(convert_to(
  p_candidato_ref||chr(31)||v_clave_interna||chr(31)||v_huella,'UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.preferencia_bolsas_version(
  candidato_ref,version,orden,conjunto_huella_sha256,clave_idempotencia,
  solicitud_huella_sha256,recibo_ref,decision_ref,auditoria_ref,registrada_en,
  actor_ref_interno,perfil_ref_interno)
 VALUES(p_candidato_ref,v_version+1,p_orden,v_conjunto,v_clave_interna,v_huella,v_recibo,
  v_consumo.decision_ref,v_consumo.auditoria_ref,v_ahora,x->>'principal_ref',x->>'perfil_activo_ref');
 INSERT INTO vec_bolsa_llamamientos.preferencia_bolsas_outbox(
  evento_ref,candidato_ref,version,recibo_ref,decision_ref,auditoria_ref,esquema,evento,creada_en)
 VALUES('evento:preferencias-bolsas:'||substr(v_recibo,length('recibo:preferencias-bolsas:')+1),
  p_candidato_ref,v_version+1,v_recibo,v_consumo.decision_ref,v_consumo.auditoria_ref,
  'vec.bolsa.preferencias.ordenadas.v1',
  jsonb_build_object('candidato_ref',p_candidato_ref,'version',v_version+1,
   'orden',p_orden,'conjunto_huella_sha256',v_conjunto,'recibo_ref',v_recibo,
   'decision_ref',v_consumo.decision_ref,'auditoria_ref',v_consumo.auditoria_ref),v_ahora);
 RETURN QUERY SELECT false,v_recibo,v_version+1,p_orden,v_ahora;
END $function$;

DO $post$
BEGIN
 IF has_function_privilege('public','vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 OR has_function_privilege('public','vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 OR (SELECT encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') FROM pg_proc p
     WHERE p.oid='vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
      AND p.prokind='f' AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef
      AND p.provolatile='v' AND obj_description(p.oid,'pg_proc') IS NULL
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
      AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_portal_externo=X/vec_bolsa_llamamientos_propietario}')
    IS DISTINCT FROM '09e84cd5c85318f3f047b73f98dbbfc1ee00d8f695f597de84d31c943b56c11e'
 OR (SELECT encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') FROM pg_proc p
     WHERE p.oid='vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
      AND p.prokind='f' AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef
      AND p.provolatile='v' AND obj_description(p.oid,'pg_proc') IS NULL
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
      AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_preferencias_interna=X/vec_bolsa_llamamientos_propietario}')
    IS DISTINCT FROM 'ce0e92ef647ea0a4c2e3935e3914c1ffd9da376b58a830520eff32c56ff67a21'
 THEN RAISE EXCEPTION 'B143: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
