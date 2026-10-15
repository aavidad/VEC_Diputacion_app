\set ON_ERROR_STOP on
-- D23: candidatura como sujeto de los actos Bolsa; persona como titular de bytes.
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
DO $patch$
DECLARE f text;a text;b text;h text;firma text;
BEGIN
 firma:='vec_documentos.consumir_preimagen_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)';
 f:=pg_get_functiondef(to_regprocedure(firma));
 a:=$m$  p_auth->>'principal_id' IS DISTINCT FROM j->>'persona_ref' OR$m$;
 b:=$m$  p_auth->>'principal_id' IS DISTINCT FROM
   (CASE WHEN s->>'modulo'='bolsa' THEN d->>'principal_id' ELSE j->>'persona_ref' END) OR
  (s->>'modulo'='bolsa' AND vec_documentos.candidato_propio_contexto_justificante_bolsa_v1(
    x,j->>'persona_ref',d->>'principal_id',clock_timestamp()) IS NOT TRUE) OR$m$;
 IF (length(f)-length(replace(f,a,'')))/length(a)<>1
 THEN RAISE EXCEPTION 'D23: marca principal divergente' USING ERRCODE='55000'; END IF;
 f:=replace(f,a,b);
 a:=$m$ esperado:=encode(sha256(convert_to(format(
  '{"ambitos":{"persona_ref":"%s"},"atributos":{"preimagen_sha256":"%s"}}',
  j->>'persona_ref',pre_sha),'UTF8')),'hex');$m$;
 b:=$m$ esperado:=encode(sha256(convert_to(format(
  CASE WHEN s->>'modulo'='bolsa' THEN
   '{"ambitos":{"candidato_ref":"%s"},"atributos":{"preimagen_sha256":"%s"}}'
  ELSE '{"ambitos":{"persona_ref":"%s"},"atributos":{"preimagen_sha256":"%s"}}' END,
  CASE WHEN s->>'modulo'='bolsa' THEN d->>'principal_id' ELSE j->>'persona_ref' END,
  pre_sha),'UTF8')),'hex');$m$;
 IF (length(f)-length(replace(f,a,'')))/length(a)<>1
 THEN RAISE EXCEPTION 'D23: marca ámbito divergente' USING ERRCODE='55000'; END IF;
 f:=replace(f,a,b);
 h:=encode(sha256(convert_to(f,'UTF8')),'hex');
 EXECUTE f;
 IF encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(firma)),'UTF8')),'hex') IS DISTINCT FROM h
 THEN RAISE EXCEPTION 'D23: posthuella preimagen divergente' USING ERRCODE='55000'; END IF;
 firma:='vec_documentos.resolver_justificante_renuncia_bolsa_v1(text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)';
 f:=pg_get_functiondef(to_regprocedure(firma));
 a:=$m$  d->>'principal_id' IS DISTINCT FROM p_persona_ref OR$m$;
 b:=$m$  d->>'principal_id' IS DISTINCT FROM p_candidato_ref OR$m$;
 IF (length(f)-length(replace(f,a,'')))/length(a)<>1
 THEN RAISE EXCEPTION 'D23: marca preparación divergente' USING ERRCODE='55000'; END IF;
 f:=replace(f,a,b);
 h:=encode(sha256(convert_to(f,'UTF8')),'hex');
 EXECUTE f;
 IF encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(firma)),'UTF8')),'hex') IS DISTINCT FROM h
 THEN RAISE EXCEPTION 'D23: posthuella preparación divergente' USING ERRCODE='55000'; END IF;
END $patch$;
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
COMMIT;
