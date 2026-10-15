\set ON_ERROR_STOP on
-- D25: la preparación propia liga la huella al ámbito candidato de Bolsa.
-- La persona sigue siendo principal de decisión y titular del documento.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000025',0));
DO $pre$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'D25: entorno incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid='vec_documentos.resolver_justificante_renuncia_bolsa_v1(text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_documentos_propietario'::regrole
 OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
 OR p.proacl::text IS DISTINCT FROM '{vec_documentos_propietario=X/vec_documentos_propietario,vec_documentos_bolsa_subida_datos=X/vec_documentos_propietario}'
 OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM '81484f3b70ab7f4d80d71530f96e21aa206ea00c893ebce554bc9f6444d46083'
 THEN RAISE EXCEPTION 'D25: preimagen divergente' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_documentos_propietario;
CREATE OR REPLACE FUNCTION vec_documentos.resolver_justificante_renuncia_bolsa_v1(p_persona_ref text, p_candidato_ref text, p_bolsa_ref text, p_registro_contexto_ref text, p_corte timestamp with time zone, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE c jsonb;d jsonb;x jsonb;v record;p record;h text;
BEGIN
 IF vec_documentos.subida_local_login_valido_v1(false) IS NOT TRUE OR
  session_user<>'vec_externo_documentos_bolsa_datos' OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' OR
  coalesce(p_persona_ref ~ '^[A-Za-z0-9:_-]{1,128}$',false) IS NOT TRUE OR
  coalesce(p_candidato_ref ~ '^[A-Za-z0-9:_-]+$' AND length(p_candidato_ref) BETWEEN 1 AND 256,false) IS NOT TRUE OR
  coalesce(p_bolsa_ref ~ '^[A-Za-z0-9:_-]+$' AND length(p_bolsa_ref) BETWEEN 1 AND 256,false) IS NOT TRUE OR
  nullif(p_registro_contexto_ref,'') IS NULL OR
  p_corte IS NULL OR NOT isfinite(p_corte) OR
  p_corte<clock_timestamp()-interval '30 seconds' OR
  p_corte>clock_timestamp()+interval '1 second'
 THEN RAISE EXCEPTION 'D22: preparación propia denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation OR character_not_in_repertoire THEN
  RAISE EXCEPTION 'D22: material propio ilegible' USING ERRCODE='22023'; END;
 h:=encode(sha256(convert_to(format(
  '{"ambitos":{"candidato_ref":"%s"},"atributos":{"bolsa_ref":"%s","candidato_ref":"%s"}}',
  p_candidato_ref,p_bolsa_ref,p_candidato_ref),'UTF8')),'hex');
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object' OR
  jsonb_typeof(x) IS DISTINCT FROM 'object' OR
  x->>'persona_ref' IS DISTINCT FROM p_persona_ref OR
  c->>'contexto_ref' IS DISTINCT FROM p_registro_contexto_ref OR
  c->>'huella_contexto_sha256' IS DISTINCT FROM encode(sha256(p_contexto),'hex') OR
  c->>'efecto_ref' IS DISTINCT FROM p_bolsa_ref OR
  c->>'operacion' IS DISTINCT FROM 'bolsa.justificante_renuncia.subir_propio' OR
  c->>'audiencia_consumo' IS DISTINCT FROM 'vec_documentos.bolsa.justificante_renuncia.preparar_subida.v1' OR
  c->>'huella_efecto_sha256' IS DISTINCT FROM h OR
  d->>'accion' IS DISTINCT FROM c->>'operacion' OR
  d->>'recurso_ref' IS DISTINCT FROM p_bolsa_ref OR
  d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR
  d->>'tipo_recurso' IS DISTINCT FROM 'justificante_renuncia' OR
  d->>'finalidad' IS DISTINCT FROM 'custodiar_justificante_renuncia_bolsa' OR
  d->>'principal_id' IS DISTINCT FROM p_persona_ref OR
  d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h OR
  d->'campos_permitidos' IS DISTINCT FROM '["contenido","estado","recibo"]'::jsonb OR
  d->'obligaciones' IS DISTINCT FROM '[]'::jsonb OR
  d#>>'{vinculo_autenticacion_actor,registro_contexto_ref}' IS DISTINCT FROM p_registro_contexto_ref OR
  d#>>'{vinculo_autenticacion_actor,contexto_actor_huella_sha256}' IS DISTINCT FROM
   encode(sha256(p_contexto),'hex') OR
  d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'externa_personal' OR
  vec_documentos.candidato_propio_contexto_justificante_bolsa_v1(
   x,p_persona_ref,p_candidato_ref,clock_timestamp()) IS NOT TRUE
 THEN RAISE EXCEPTION 'D22: preparación propia divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_subida_local_bolsa_v1(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS DISTINCT FROM true OR v.efecto_ref IS DISTINCT FROM p_bolsa_ref OR
  v.huella_efecto_sha256 IS DISTINCT FROM h OR v.decision_ref IS NULL OR v.auditoria_ref IS NULL
 THEN RAISE EXCEPTION 'D22: consumo propio inválido' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT p FROM vec_bolsa_llamamientos.proyectar_justificante_renuncia_v1(
  p_persona_ref,p_candidato_ref,p_bolsa_ref,p_registro_contexto_ref,p_corte);
 IF coalesce(p.recurso_ref ~ '^justificante-renuncia:[0-9a-f]{64}$',false) IS NOT TRUE OR
  p.version_dueno IS DISTINCT FROM 1 OR
  p.candidato_ref IS DISTINCT FROM p_candidato_ref OR
  p.persona_ref IS DISTINCT FROM p_persona_ref OR
  nullif(p.participacion_ref,'') IS NULL OR nullif(p.llamamiento_ref,'') IS NULL
 THEN RAISE EXCEPTION 'D22: proyección propia divergente' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('recurso_ref',p.recurso_ref,'version_dueno',p.version_dueno,
  'participacion_ref',p.participacion_ref,'llamamiento_ref',p.llamamiento_ref,
  'candidato_ref',p.candidato_ref,'persona_ref',p.persona_ref,
  'decision_ref',v.decision_ref,'auditoria_ref',v.auditoria_ref);
END $function$
;
RESET ROLE;
DO $post$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid='vec_documentos.resolver_justificante_renuncia_bolsa_v1(text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_documentos_propietario'::regrole
 OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
 OR p.proacl::text IS DISTINCT FROM '{vec_documentos_propietario=X/vec_documentos_propietario,vec_documentos_bolsa_subida_datos=X/vec_documentos_propietario}'
 OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM '606edd1ed9c21e52dd937a55029d406625787c6cfc266a007ee82915ebf23e39'
 THEN RAISE EXCEPTION 'D25: postimagen divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
