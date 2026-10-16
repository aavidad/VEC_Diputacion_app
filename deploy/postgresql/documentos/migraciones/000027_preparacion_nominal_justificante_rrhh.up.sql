\set ON_ERROR_STOP on
-- D27: consumo nominal RRHH antes de cualquier consulta relacional en Bolsa.
-- Un ámbito divergente revierte también el consumo y su auditoría en la TX.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000027',0));
DO $pre$
DECLARE p pg_proc%ROWTYPE; f record;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 OR
  NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) OR
  to_regprocedure('vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_documental_v1(text,text,text)') IS NULL
 THEN RAISE EXCEPTION 'D27: entorno incompatible' USING ERRCODE='55000'; END IF;
 FOR f IN SELECT * FROM (VALUES
  ('vec_documentos.consultar_justificante_renuncia_rrhh_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   '755a03b5af42df681d6d043a22dca4b209d437f9b0947b034dba8539050fcb98'),
  ('vec_documentos.resolver_consulta_justificante_renuncia_bolsa_rrhh_v1(text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'ee6af85f11b0d68ac8afc49a62e0cd2f0be9028d60d70b2362721cc87df91379')
 ) v(firma,huella) LOOP
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=to_regprocedure(f.firma);
  IF p.proowner IS DISTINCT FROM 'vec_documentos_propietario'::regrole OR
   p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v' OR
   p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
    'TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[] OR
   p.proacl::text IS DISTINCT FROM
    '{vec_documentos_propietario=X/vec_documentos_propietario,vec_documentos_bolsa_rrhh=X/vec_documentos_propietario}' OR
   encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM f.huella
  THEN RAISE EXCEPTION 'D27: preimagen divergente %',f.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;
SET LOCAL ROLE vec_documentos_propietario;
CREATE OR REPLACE FUNCTION vec_documentos.consultar_justificante_renuncia_rrhh_v1(p_preimagen bytea, p_auth jsonb, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE j jsonb; c jsonb; d jsonb; x jsonb; ambito jsonb; r vec_documentos.subida_local%ROWTYPE;
 v record; v_pre_sha text; v_efecto_sha text; v_claves constant text[]:=ARRAY[
  'fase','accion','carga_id','documento_id','recurso_ref','persona_ref','actor_ref',
  'perfil_activo_ref','bolsa_ref','participacion_ref','registro_contexto_ref',
  'huella_contexto_sha256','correlacion_ref'];
BEGIN
 IF vec_documentos.login_justificante_renuncia_rrhh_valido_v1() IS NOT TRUE OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' OR
  p_preimagen IS NULL OR octet_length(p_preimagen) NOT BETWEEN 1 AND 65536 OR
  jsonb_typeof(p_auth) IS DISTINCT FROM 'object' OR
  (SELECT count(*) FROM jsonb_object_keys(p_auth))<>7 OR
  (p_auth ?& ARRAY['accion','recurso_ref','ambito_ref','principal_id',
    'perfil_activo_ref','finalidad','correlacion_ref']) IS NOT TRUE
 THEN RAISE EXCEPTION 'D22: consulta RRHH denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  j:=convert_from(p_preimagen,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation OR character_not_in_repertoire THEN
  RAISE EXCEPTION 'D22: material RRHH ilegible' USING ERRCODE='22023'; END;
 IF jsonb_typeof(j) IS DISTINCT FROM 'object' OR
  (SELECT count(*) FROM jsonb_object_keys(j))<>array_length(v_claves,1) OR
  (j ?& v_claves) IS NOT TRUE OR
  EXISTS(SELECT 1 FROM unnest(v_claves) AS k(clave) WHERE
   jsonb_typeof(j->k.clave) IS DISTINCT FROM 'string' OR
   octet_length(j->>k.clave)>512 OR
   (k.clave<>'carga_id' AND octet_length(j->>k.clave)=0)) OR
  jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object' OR
  jsonb_typeof(x) IS DISTINCT FROM 'object' OR
  j->>'fase' IS DISTINCT FROM 'abrir_rrhh' OR
  j->>'accion' IS DISTINCT FROM 'bolsa.justificante_renuncia.descargar_rrhh' OR
  (j->>'carga_id'<>'' AND j->>'carga_id' !~ '^ref:[0-9a-f]{64}$') OR
  j->>'documento_id' !~ '^ref:[0-9a-f]{64}$' OR
  j->>'recurso_ref' !~ '^justificante-renuncia:[0-9a-f]{64}$' OR
  j->>'persona_ref' !~ '^[A-Za-z0-9:_-]{1,128}$' OR
  nullif(j->>'actor_ref','') IS NULL OR nullif(j->>'perfil_activo_ref','') IS NULL OR
  (j->>'bolsa_ref' !~ '^[A-Za-z0-9:_-]+$' OR length(j->>'bolsa_ref') NOT BETWEEN 1 AND 256) OR
  (j->>'participacion_ref' !~ '^[A-Za-z0-9:_-]+$' OR length(j->>'participacion_ref') NOT BETWEEN 1 AND 256) OR
  nullif(j->>'correlacion_ref','') IS NULL OR
  nullif(j->>'registro_contexto_ref','') IS NULL OR
  j->>'huella_contexto_sha256' !~ '^[0-9a-f]{64}$' OR
  j->>'registro_contexto_ref' IS DISTINCT FROM c->>'contexto_ref' OR
  j->>'huella_contexto_sha256' IS DISTINCT FROM c->>'huella_contexto_sha256' OR
  j->>'huella_contexto_sha256' IS DISTINCT FROM encode(sha256(p_contexto),'hex') OR
  j->>'actor_ref' IS DISTINCT FROM x->>'principal_ref' OR
  j->>'perfil_activo_ref' IS DISTINCT FROM x->>'perfil_activo_ref' OR
  p_auth->>'accion' IS DISTINCT FROM j->>'accion' OR
  p_auth->>'recurso_ref' IS DISTINCT FROM j->>'documento_id' OR
  p_auth->>'ambito_ref' IS DISTINCT FROM j->>'recurso_ref' OR
  p_auth->>'principal_id' IS DISTINCT FROM j->>'actor_ref' OR
  p_auth->>'perfil_activo_ref' IS DISTINCT FROM j->>'perfil_activo_ref' OR
  p_auth->>'finalidad' IS DISTINCT FROM 'custodiar_justificante_renuncia_bolsa' OR
  p_auth->>'correlacion_ref' IS DISTINCT FROM j->>'correlacion_ref' OR
  c->>'efecto_ref' IS DISTINCT FROM j->>'documento_id' OR
  c->>'operacion' IS DISTINCT FROM j->>'accion' OR
  c->>'audiencia_consumo' IS DISTINCT FROM 'vec_documentos.bolsa.justificante_renuncia.descarga_rrhh.v1' OR
  d->>'accion' IS DISTINCT FROM j->>'accion' OR
  d->>'recurso_ref' IS DISTINCT FROM j->>'documento_id' OR
  d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR
  d->>'tipo_recurso' IS DISTINCT FROM 'justificante_renuncia' OR
  d->>'finalidad' IS DISTINCT FROM 'custodiar_justificante_renuncia_bolsa' OR
  d->>'principal_id' IS DISTINCT FROM j->>'actor_ref' OR
  d->>'perfil_activo_ref' IS DISTINCT FROM j->>'perfil_activo_ref' OR
  d->'campos_permitidos' IS DISTINCT FROM '["contenido"]'::jsonb OR
  d->'obligaciones' IS DISTINCT FROM '[]'::jsonb OR
  d#>>'{vinculo_autenticacion_actor,registro_contexto_ref}' IS DISTINCT FROM j->>'registro_contexto_ref' OR
  d#>>'{vinculo_autenticacion_actor,contexto_actor_huella_sha256}' IS DISTINCT FROM j->>'huella_contexto_sha256' OR
  d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa' OR
  coalesce((c->>'huella_efecto_sha256') ~ '^[0-9a-f]{64}$',false) IS NOT TRUE OR
  d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'D22: decisión RRHH divergente' USING ERRCODE='42501'; END IF;
 v_pre_sha:=encode(sha256(p_preimagen),'hex');
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_subida_local_bolsa_v1(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS DISTINCT FROM true OR
  v.efecto_ref IS DISTINCT FROM j->>'documento_id' OR
  v.decision_ref IS NULL OR v.auditoria_ref IS NULL
 THEN RAISE EXCEPTION 'D22: consumo RRHH inválido' USING ERRCODE='42501'; END IF;
 ambito:=vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_documental_v1(
  j->>'bolsa_ref',j->>'participacion_ref',j->>'documento_id');
 v_efecto_sha:=encode(sha256(convert_to(format(
  '{"ambitos":{"ambito_ref":"%s","unidad_ref":"%s"},"atributos":{"bolsa_ref":"%s","participacion_ref":"%s","preimagen_sha256":"%s"}}',
  ambito->>'ambito_ref',ambito->>'unidad_ref',j->>'bolsa_ref',j->>'participacion_ref',v_pre_sha),'UTF8')),'hex');
 IF c->>'huella_efecto_sha256' IS DISTINCT FROM v_efecto_sha OR
  d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_efecto_sha OR
  v.huella_efecto_sha256 IS DISTINCT FROM v_efecto_sha
 THEN RAISE EXCEPTION 'D22: recurso RRHH no ligado' USING ERRCODE='42501'; END IF;
 SELECT * INTO r FROM vec_documentos.subida_local
 WHERE documento_id=j->>'documento_id' FOR SHARE;
 IF NOT FOUND OR r.modulo_id IS DISTINCT FROM 'bolsa' OR
  (j->>'carga_id'<>'' AND r.carga_id IS DISTINCT FROM j->>'carga_id') OR
  r.tipo_recurso IS DISTINCT FROM 'justificante_renuncia' OR
  r.finalidad IS DISTINCT FROM 'custodiar_justificante_renuncia_bolsa' OR
  r.recurso_ref IS DISTINCT FROM j->>'recurso_ref' OR
  r.persona_ref IS DISTINCT FROM j->>'persona_ref' OR
  r.estado IS DISTINCT FROM 'admitido' OR r.version IS DISTINCT FROM 5 OR
  r.analisis->>'Estado' IS DISTINCT FROM 'limpio' OR
  r.admitido IS NULL OR r.recibo_ref IS NULL OR r.registrada_en IS NULL
 THEN RAISE EXCEPTION 'D22: justificante RRHH no disponible' USING ERRCODE='42501'; END IF;
 PERFORM vec_documentos.verificar_integridad_subida_local_v1(r.carga_id);
 IF vec_bolsa_llamamientos.validar_consulta_justificante_renuncia_rrhh_v1(
  r.recurso_ref,r.persona_ref,j->>'bolsa_ref',j->>'participacion_ref',
  j->>'actor_ref',j->>'perfil_activo_ref') IS NOT TRUE
 THEN RAISE EXCEPTION 'D22: relación RRHH denegada' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_documentos.subida_local_lectura(decision_ref,carga_id,fase,persona_ref,
  auditoria_v3_ref,huella_preimagen_sha256)
 VALUES(v.decision_ref,r.carga_id,'abrir_rrhh',r.persona_ref,v.auditoria_ref,v_pre_sha);
 RETURN jsonb_build_object('resultado',jsonb_build_object('carga_id',r.carga_id,
  'comprobante_ref',r.documento_id,'documento_version',r.documento_version,
  'recibo_ref',r.recibo_ref,'estado',r.estado,'codigo_rechazo',r.codigo_rechazo,
  'registrado_en',r.registrada_en),
  'recurso_ref',r.recurso_ref,'persona_ref',r.persona_ref,'correlacion_ref',r.correlacion_ref,
  'modulo_id',r.modulo_id,'tipo_recurso',r.tipo_recurso,'finalidad',r.finalidad,
  'clasificacion',r.clasificacion,'huella_solicitud_hmac',r.huella_solicitud_hmac,
  'huella_pdf_sha256',r.huella_pdf_sha256,'mime',r.mime,'tamano',r.tamano,
  'version',r.version,'cuarentena',r.cuarentena,'analisis',r.analisis,'admitido',r.admitido);
END $function$
;
CREATE OR REPLACE FUNCTION vec_documentos.resolver_consulta_justificante_renuncia_bolsa_rrhh_v1(p_bolsa_ref text, p_participacion_ref text, p_actor_ref text, p_perfil_ref text, p_registro_contexto_ref text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE c jsonb;d jsonb;x jsonb;ambito jsonb;v record;p record;h text;
BEGIN
 IF vec_documentos.login_justificante_renuncia_rrhh_valido_v1() IS NOT TRUE OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' OR
  coalesce(p_bolsa_ref ~ '^[A-Za-z0-9:_-]+$' AND length(p_bolsa_ref) BETWEEN 1 AND 256,false) IS NOT TRUE OR
  coalesce(p_participacion_ref ~ '^[A-Za-z0-9:_-]+$' AND length(p_participacion_ref) BETWEEN 1 AND 256,false) IS NOT TRUE OR
  nullif(p_actor_ref,'') IS NULL OR nullif(p_perfil_ref,'') IS NULL OR
  nullif(p_registro_contexto_ref,'') IS NULL
 THEN RAISE EXCEPTION 'D22: preparación RRHH denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation OR character_not_in_repertoire THEN
  RAISE EXCEPTION 'D22: material RRHH ilegible' USING ERRCODE='22023'; END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object' OR
  jsonb_typeof(x) IS DISTINCT FROM 'object' OR
  x->>'principal_ref' IS DISTINCT FROM p_actor_ref OR
  x->>'perfil_activo_ref' IS DISTINCT FROM p_perfil_ref OR
  c->>'contexto_ref' IS DISTINCT FROM p_registro_contexto_ref OR
  c->>'huella_contexto_sha256' IS DISTINCT FROM encode(sha256(p_contexto),'hex') OR
  c->>'efecto_ref' IS DISTINCT FROM p_participacion_ref OR
  c->>'operacion' IS DISTINCT FROM 'bolsa.justificante_renuncia.descargar_rrhh' OR
  c->>'audiencia_consumo' IS DISTINCT FROM 'vec_documentos.bolsa.justificante_renuncia.preparar_descarga_rrhh.v1' OR
  d->>'accion' IS DISTINCT FROM c->>'operacion' OR
  d->>'recurso_ref' IS DISTINCT FROM p_participacion_ref OR
  d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR
  d->>'tipo_recurso' IS DISTINCT FROM 'justificante_renuncia' OR
  d->>'finalidad' IS DISTINCT FROM 'custodiar_justificante_renuncia_bolsa' OR
  d->>'principal_id' IS DISTINCT FROM p_actor_ref OR
  d->>'perfil_activo_ref' IS DISTINCT FROM p_perfil_ref OR
  d->'campos_permitidos' IS DISTINCT FROM '["contenido"]'::jsonb OR
  d->'obligaciones' IS DISTINCT FROM '[]'::jsonb OR
  d#>>'{vinculo_autenticacion_actor,registro_contexto_ref}' IS DISTINCT FROM p_registro_contexto_ref OR
  d#>>'{vinculo_autenticacion_actor,contexto_actor_huella_sha256}' IS DISTINCT FROM
   encode(sha256(p_contexto),'hex') OR
  d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa' OR
  coalesce((c->>'huella_efecto_sha256') ~ '^[0-9a-f]{64}$',false) IS NOT TRUE OR
  d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'D22: preparación RRHH divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_subida_local_bolsa_v1(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS DISTINCT FROM true OR v.efecto_ref IS DISTINCT FROM p_participacion_ref OR
  v.decision_ref IS NULL OR v.auditoria_ref IS NULL
 THEN RAISE EXCEPTION 'D22: consumo RRHH de preparación inválido' USING ERRCODE='42501'; END IF;
 ambito:=vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_documental_v1(
  p_bolsa_ref,p_participacion_ref,NULL);
 h:=encode(sha256(convert_to(format(
  '{"ambitos":{"ambito_ref":"%s","unidad_ref":"%s"},"atributos":{"bolsa_ref":"%s","participacion_ref":"%s"}}',
  ambito->>'ambito_ref',ambito->>'unidad_ref',p_bolsa_ref,p_participacion_ref),'UTF8')),'hex');
 IF c->>'huella_efecto_sha256' IS DISTINCT FROM h OR
  d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h OR
  v.huella_efecto_sha256 IS DISTINCT FROM h
 THEN RAISE EXCEPTION 'D22: recurso RRHH no ligado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT p FROM vec_bolsa_llamamientos.proyectar_consulta_justificante_renuncia_rrhh_v1(
  p_bolsa_ref,p_participacion_ref,p_actor_ref,p_perfil_ref,p_registro_contexto_ref);
 IF coalesce(p.documento_ref ~ '^ref:[0-9a-f]{64}$',false) IS NOT TRUE OR
  coalesce(p.documento_sha256 ~ '^[0-9a-f]{64}$',false) IS NOT TRUE OR
  coalesce(p.recurso_ref ~ '^justificante-renuncia:[0-9a-f]{64}$',false) IS NOT TRUE OR
  nullif(p.persona_titular_ref,'') IS NULL
 THEN RAISE EXCEPTION 'D22: proyección RRHH divergente' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('documento_ref',p.documento_ref,
  'documento_sha256',p.documento_sha256,'recurso_ref',p.recurso_ref,
  'persona_titular_ref',p.persona_titular_ref,
  'unidad_ref',ambito->>'unidad_ref','ambito_ref',ambito->>'ambito_ref',
  'decision_ref',v.decision_ref,'auditoria_ref',v.auditoria_ref);
END $function$
;
RESET ROLE;
DO $post$
DECLARE p pg_proc%ROWTYPE; f record;
BEGIN
 FOR f IN SELECT * FROM (VALUES
  ('vec_documentos.consultar_justificante_renuncia_rrhh_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'eb4f7f73213052daa9664b479b98e325c7c6386a9c0eebe81b2f422ba5b633f7'),
  ('vec_documentos.resolver_consulta_justificante_renuncia_bolsa_rrhh_v1(text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   '845b022f144e1372e0bd5fea9d9fe57af116b276d2db68ebcc1d516e41550812')
 ) v(firma,huella) LOOP
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=to_regprocedure(f.firma);
  IF p.proowner IS DISTINCT FROM 'vec_documentos_propietario'::regrole OR
   p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v' OR
   p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
    'TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[] OR
   p.proacl::text IS DISTINCT FROM
    '{vec_documentos_propietario=X/vec_documentos_propietario,vec_documentos_bolsa_rrhh=X/vec_documentos_propietario}' OR
   encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM f.huella
  THEN RAISE EXCEPTION 'D27: postimagen divergente %',f.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $post$;
COMMIT;
