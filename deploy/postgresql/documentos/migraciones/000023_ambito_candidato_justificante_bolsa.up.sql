\set ON_ERROR_STOP on
-- D23: candidatura como ámbito de los actos Bolsa; persona como principal y titular de bytes.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000023',0));
DO $pre$
DECLARE p pg_proc%ROWTYPE;v record;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regprocedure('vec_documentos.candidato_propio_contexto_justificante_bolsa_v1(jsonb,text,text,timestamptz)') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.validar_propietario_subida_documental_v1(text,bigint,text,text)') IS NULL
 THEN RAISE EXCEPTION 'D23: estructura incompatible' USING ERRCODE='55000'; END IF;
 FOR v IN SELECT * FROM (VALUES
 ('vec_documentos.consumir_preimagen_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  '02bfd58c6723351cc2848a05246617d97649a722104ad90c6c67d22209d10238',
  '{vec_documentos_propietario=X/vec_documentos_propietario}',
  ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s']::text[]),
 ('vec_documentos.resolver_justificante_renuncia_bolsa_v1(text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'f7d705bf8564bf2e94906cb09258d6bc04e04e240a2f3672693c027af420ce58',
  '{vec_documentos_propietario=X/vec_documentos_propietario,vec_documentos_bolsa_subida_datos=X/vec_documentos_propietario}',
  ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[])
 ) x(firma,huella,acl,config) LOOP
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=to_regprocedure(v.firma);
  IF p.proowner IS DISTINCT FROM 'vec_documentos_propietario'::regrole
  OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v'
  OR p.proconfig IS DISTINCT FROM v.config OR p.proacl::text IS DISTINCT FROM v.acl
  OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM v.huella
  THEN RAISE EXCEPTION 'D23: preimagen divergente %',v.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;
SET LOCAL ROLE vec_documentos_propietario;
-- Definiciones literales con persona titular y candidatura autorizada.
CREATE OR REPLACE FUNCTION vec_documentos.consumir_preimagen_subida_local_v1(p_preimagen bytea, p_auth jsonb, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
AS $function$
DECLARE j jsonb; c jsonb; d jsonb; x jsonb; v record; claves text[]; fase text; accion text;
 efecto text; ambito text; pre_sha text; esperado text; s jsonb; candidato text;
BEGIN
 IF vec_documentos.subida_local_login_valido_v1(false) IS NOT TRUE OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' OR
  octet_length(p_preimagen) NOT BETWEEN 1 AND 65536 OR
  jsonb_typeof(p_auth) IS DISTINCT FROM 'object' OR
  (SELECT count(*) FROM jsonb_object_keys(p_auth))<>7 OR
  p_auth ?& ARRAY['accion','recurso_ref','ambito_ref','principal_id','perfil_activo_ref','finalidad','correlacion_ref'] IS NOT TRUE
 THEN RAISE EXCEPTION 'D20: invocación denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  j:=convert_from(p_preimagen,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation OR character_not_in_repertoire THEN
  RAISE EXCEPTION 'D20: material ilegible' USING ERRCODE='22023'; END;
 IF jsonb_typeof(j) IS DISTINCT FROM 'object' OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR
  jsonb_typeof(d) IS DISTINCT FROM 'object' OR jsonb_typeof(x) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'D20: objeto inválido' USING ERRCODE='22023'; END IF;
 fase:=j->>'fase'; accion:=j->>'accion';
 s:=vec_documentos.selector_subida_local_v1();
 IF s IS NULL THEN RAISE EXCEPTION 'D22: selector ausente' USING ERRCODE='42501'; END IF;
 IF s->>'modulo'='bolsa' THEN
  IF jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array'
  THEN RAISE EXCEPTION 'D23: vínculos de candidatura ausentes' USING ERRCODE='42501'; END IF;
  SELECT max(e->>'referencia') INTO candidato
  FROM jsonb_array_elements(x->'vinculos') e
  WHERE e->>'tipo'='candidato' AND e->>'estado'='activo';
  IF vec_documentos.candidato_propio_contexto_justificante_bolsa_v1(
   x,j->>'persona_ref',candidato,clock_timestamp()) IS NOT TRUE
  THEN RAISE EXCEPTION 'D23: candidatura no vigente' USING ERRCODE='42501'; END IF;
 END IF;
 claves:=CASE fase
  WHEN 'reservar' THEN ARRAY['accion','fase','carga_id','recurso_ref','persona_ref','perfil_activo_ref','modulo_id','tipo_recurso','finalidad','clave_idempotencia_hmac','huella_solicitud_hmac','huella_pdf_sha256','tamano','mime','propietario_version','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'confirmar_recepcion' THEN ARRAY['accion','fase','carga_id','recurso_ref','persona_ref','perfil_activo_ref','finalidad','version_esperada','huella_pdf_sha256','objeto','evidencia_almacen','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'iniciar_analisis' THEN ARRAY['accion','fase','carga_id','recurso_ref','persona_ref','perfil_activo_ref','finalidad','version_esperada','recibo_ref','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'registrar_analisis' THEN ARRAY['accion','fase','carga_id','recurso_ref','persona_ref','perfil_activo_ref','finalidad','version_esperada','analisis','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'cerrar_analisis' THEN ARRAY['accion','fase','carga_id','recurso_ref','persona_ref','perfil_activo_ref','finalidad','version_esperada','estado_final','codigo_rechazo','analisis','promocion','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'consulta_carga' THEN ARRAY['accion','fase','carga_id','documento_id','persona_ref','perfil_activo_ref','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'consulta_documento' THEN ARRAY['accion','fase','carga_id','documento_id','persona_ref','perfil_activo_ref','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'abrir' THEN ARRAY['accion','fase','carga_id','documento_id','persona_ref','perfil_activo_ref','objeto_ref','objeto_version','huella_pdf_sha256','tamano','mime','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  ELSE NULL END;
 IF claves IS NULL OR (SELECT count(*) FROM jsonb_object_keys(j))<>array_length(claves,1) OR
  j ?& claves IS NOT TRUE OR
  ((fase IN ('reservar','confirmar_recepcion','iniciar_analisis','registrar_analisis','cerrar_analisis','consulta_carga'))
    <> (accion=s->>'subir')) OR
  accion NOT IN (s->>'subir',s->>'descargar') OR
  j->>'persona_ref' !~ '^[A-Za-z0-9:_-]{1,128}$' OR
  j->>'persona_ref' IS DISTINCT FROM x->>'persona_ref' OR
  j->>'perfil_activo_ref' IS DISTINCT FROM x->>'perfil_activo_ref' OR
  j->>'registro_contexto_ref' IS DISTINCT FROM c->>'contexto_ref' OR
  j->>'huella_contexto_sha256' IS DISTINCT FROM c->>'huella_contexto_sha256' OR
  j->>'huella_contexto_sha256' IS DISTINCT FROM encode(sha256(p_contexto),'hex') OR
  j->>'finalidad' IS DISTINCT FROM (CASE WHEN fase IN ('reservar','confirmar_recepcion','iniciar_analisis','registrar_analisis','cerrar_analisis')
   THEN s->>'finalidad' ELSE j->>'finalidad' END) OR
  p_auth->>'accion' IS DISTINCT FROM accion OR
  p_auth->>'principal_id' IS DISTINCT FROM j->>'persona_ref' OR
  (s->>'modulo'='bolsa' AND d->>'principal_id' IS DISTINCT FROM j->>'persona_ref') OR
  p_auth->>'perfil_activo_ref' IS DISTINCT FROM j->>'perfil_activo_ref' OR
  p_auth->>'finalidad' IS DISTINCT FROM s->>'finalidad' OR
  p_auth->>'correlacion_ref' IS DISTINCT FROM j->>'correlacion_ref'
 THEN RAISE EXCEPTION 'D20: preimagen divergente' USING ERRCODE='42501'; END IF;
 efecto:=CASE WHEN fase='consulta_documento' THEN j->>'documento_id' ELSE j->>'carga_id' END;
 ambito:=CASE WHEN fase IN ('consulta_carga','consulta_documento') THEN efecto
  WHEN fase='abrir' THEN p_auth->>'ambito_ref' ELSE j->>'recurso_ref' END;
 IF efecto IS NULL OR efecto !~ '^ref:[0-9a-f]{64}$' OR
  p_auth->>'recurso_ref' IS DISTINCT FROM efecto OR
  p_auth->>'ambito_ref' IS DISTINCT FROM ambito OR
  (fase IN ('consulta_carga','consulta_documento') AND p_auth->>'ambito_ref' IS DISTINCT FROM efecto) OR
  c->>'efecto_ref' IS DISTINCT FROM efecto OR d->>'recurso_ref' IS DISTINCT FROM efecto OR
  c->>'operacion' IS DISTINCT FROM accion OR d->>'accion' IS DISTINCT FROM accion OR
  c->>'audiencia_consumo' IS DISTINCT FROM (CASE WHEN accion=s->>'subir'
   THEN s->>'audiencia_subir' ELSE s->>'audiencia_descargar' END) OR
  d->>'modulo_id' IS DISTINCT FROM s->>'modulo' OR d->>'tipo_recurso' IS DISTINCT FROM s->>'tipo' OR
  d->>'finalidad' IS DISTINCT FROM s->>'finalidad' OR
  d->'campos_permitidos' IS DISTINCT FROM (CASE WHEN accion=s->>'subir'
   THEN '["contenido","estado","recibo"]'::jsonb ELSE '["contenido"]'::jsonb END) OR
  d->'obligaciones' IS DISTINCT FROM '[]'::jsonb OR
  d#>>'{vinculo_autenticacion_actor,registro_contexto_ref}' IS DISTINCT FROM j->>'registro_contexto_ref' OR
  d#>>'{vinculo_autenticacion_actor,contexto_actor_huella_sha256}' IS DISTINCT FROM j->>'huella_contexto_sha256' OR
  d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM s->>'superficie'
 THEN RAISE EXCEPTION 'D20: decisión divergente' USING ERRCODE='42501'; END IF;
 pre_sha:=encode(sha256(p_preimagen),'hex');
 esperado:=encode(sha256(convert_to(format(
  CASE WHEN s->>'modulo'='bolsa' THEN
   '{"ambitos":{"candidato_ref":"%s"},"atributos":{"preimagen_sha256":"%s"}}'
  ELSE '{"ambitos":{"persona_ref":"%s"},"atributos":{"preimagen_sha256":"%s"}}' END,
  CASE WHEN s->>'modulo'='bolsa' THEN candidato ELSE j->>'persona_ref' END,
  pre_sha),'UTF8')),'hex');
 IF d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM esperado OR
  c->>'huella_efecto_sha256' IS DISTINCT FROM esperado
 THEN RAISE EXCEPTION 'D20: efecto no ligado' USING ERRCODE='42501'; END IF;
 IF s->>'modulo'='bolsa' THEN
  SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_subida_local_bolsa_v1(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
   p_payload,p_sobre,p_evidencia,p_raiz);
 ELSE
  SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_subida_local_modulo_v1(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
   p_payload,p_sobre,p_evidencia,p_raiz);
 END IF;
 IF v.consumo_nuevo IS DISTINCT FROM TRUE OR v.efecto_ref IS DISTINCT FROM efecto OR
  v.huella_efecto_sha256 IS DISTINCT FROM esperado
 THEN RAISE EXCEPTION 'D20: consumo inválido' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('preimagen',j,'decision_ref',v.decision_ref,'auditoria_ref',v.auditoria_ref,
  'preimagen_sha256',pre_sha,'persona_version',p_persona_version);
END $function$
;
-- La preparación D22 conserva principal_id=persona_ref; su preimagen
-- y postimagen exactas se verifican sin reconstruir la función.
RESET ROLE;
DO $post$
DECLARE p pg_proc%ROWTYPE;v record;
BEGIN
 FOR v IN SELECT * FROM (VALUES
 ('vec_documentos.consumir_preimagen_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  '{vec_documentos_propietario=X/vec_documentos_propietario}',
  ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s']::text[]),
 ('vec_documentos.resolver_justificante_renuncia_bolsa_v1(text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  '{vec_documentos_propietario=X/vec_documentos_propietario,vec_documentos_bolsa_subida_datos=X/vec_documentos_propietario}',
  ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[])
 ) x(firma,acl,config) LOOP
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=to_regprocedure(v.firma);
  IF p.proowner IS DISTINCT FROM 'vec_documentos_propietario'::regrole
  OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v'
  OR p.proconfig IS DISTINCT FROM v.config OR p.proacl::text IS DISTINCT FROM v.acl
  THEN RAISE EXCEPTION 'D23: flags o ACL alterados %',v.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $post$;
DO $huellas$
BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef(
  'vec_documentos.consumir_preimagen_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM '05136a837b83010b272d4fc8629452b02adacdeeffde28e85aa5bfeb368372cf'
 OR encode(sha256(convert_to(pg_get_functiondef(
  'vec_documentos.resolver_justificante_renuncia_bolsa_v1(text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM 'f7d705bf8564bf2e94906cb09258d6bc04e04e240a2f3672693c027af420ce58'
 THEN RAISE EXCEPTION 'D23: posthuella literal divergente' USING ERRCODE='55000'; END IF;
END $huellas$;
COMMIT;
