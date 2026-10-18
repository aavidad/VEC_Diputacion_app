\set ON_ERROR_STOP on
-- B135: una plaza adjudicada de la oferta CT del padre, con disposición
-- telemática anterior y persona acreditada. CT consume V3 antes de llamar.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000135',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR EXISTS(SELECT 1 FROM unnest(ARRAY[
      'origen_oferta_ct_v1','origen_oferta_ct_outbox_v1','oferta_publicada',
      'plazas_oferta','acto_plaza_oferta','acto_plaza_oferta_outbox',
      'disposicion_oferta','disposicion_oferta_candidato','politica_ofertas_version']) n(nombre)
      WHERE to_regclass('vec_bolsa_llamamientos.'||n.nombre) IS NULL)
    OR to_regprocedure('vec_bolsa_llamamientos.verificar_oferta_ct_v1(text,text,bigint,text,text,text,text,text,text,integer,jsonb,text)') IS NULL
    OR to_regprocedure('vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(text)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.verificar_plaza_adjudicada_ct_v1(text,text,text,integer,integer,text,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'B135: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_bolsa_llamamientos.verificar_plaza_adjudicada_ct_v1(
 p_organizacion text,p_padre text,p_oferta text,p_plaza integer,
 p_secuencia integer,p_participacion text,p_recibo_adjudicacion text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE
 origen record; oferta record; plazas record; politica record; acto record;
 disposicion record; candidato record; persona jsonb; v_ahora timestamptz;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER') IS TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') IS TRUE
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER') IS TRUE
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER') IS TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'B135: sesión denegada' USING ERRCODE='42501'; END IF;
 IF EXISTS(SELECT 1 FROM unnest(ARRAY[p_organizacion,p_padre,p_participacion]) x(referencia)
     WHERE x.referencia IS NULL OR x.referencia<>btrim(x.referencia)
       OR octet_length(x.referencia) NOT BETWEEN 1 AND 256)
    OR p_oferta IS NULL OR p_oferta !~ '^oferta:[0-9a-f]{64}$'
    OR p_plaza IS NULL OR p_plaza NOT BETWEEN 1 AND 100
    OR p_secuencia IS NULL OR p_secuencia NOT BETWEEN 1 AND 10000
    OR p_recibo_adjudicacion IS NULL
    OR p_recibo_adjudicacion !~ '^recibo:plaza-oferta:[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'B135: selector inválido' USING ERRCODE='22023'; END IF;

 -- B28/B29/B58 toman este cerrojo antes de mutar la oferta. El llamador CT
 -- conserva el mismo cerrojo de sesión adquirido antes de BEGIN SERIALIZABLE:
 -- así esta lectura no usa una instantánea anterior al último acto confirmado.
 -- ContextoActor se acredita después de este cerrojo.
 PERFORM pg_advisory_xact_lock(hashtextextended(
  'vec_bolsa_llamamientos:resolucion-oferta:'||p_oferta,0));
 v_ahora:=clock_timestamp();
 SELECT * INTO origen FROM vec_bolsa_llamamientos.origen_oferta_ct_v1
  WHERE oferta_ref=p_oferta AND organizacion_ref=p_organizacion
    AND expediente_ref=p_padre;
 SELECT * INTO oferta FROM vec_bolsa_llamamientos.oferta_publicada
  WHERE oferta_ref=p_oferta;
 SELECT * INTO plazas FROM vec_bolsa_llamamientos.plazas_oferta
  WHERE oferta_ref=p_oferta;
 IF origen.oferta_ref IS NULL OR oferta.oferta_ref IS NULL OR plazas.oferta_ref IS NULL
    OR origen.recibo_ref IS DISTINCT FROM oferta.recibo_ref
    OR origen.decision_ref IS DISTINCT FROM oferta.decision_ref
    OR origen.publicada_en IS DISTINCT FROM oferta.publicada_en
    OR origen.bolsa_ref IS DISTINCT FROM oferta.bolsa_ref
    OR origen.bolsa_ref IS DISTINCT FROM plazas.bolsa_ref
    OR origen.numero_plazas IS DISTINCT FROM plazas.numero_plazas
    OR p_plaza>plazas.numero_plazas
    OR plazas.politica_version::text IS DISTINCT FROM oferta.plazo->>'politica_version'
    OR origen.origen_huella_sha256 IS DISTINCT FROM
       encode(sha256(convert_to(origen.origen_canonico::text,'UTF8')),'hex')
    OR origen.origen_canonico->>'organizacion_ref' IS DISTINCT FROM p_organizacion
    OR origen.origen_canonico->>'expediente_ref' IS DISTINCT FROM p_padre
    OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.origen_oferta_ct_outbox_v1 e
       WHERE e.evento_ref=origen.evento_ref AND e.oferta_ref=p_oferta
         AND e.tipo='bolsa.oferta.ct.origen_confirmado'
         AND e.creada_en=origen.publicada_en)
 THEN RAISE EXCEPTION 'B135: origen de oferta divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO politica FROM vec_bolsa_llamamientos.politica_ofertas_version
  WHERE bolsa_ref=plazas.bolsa_ref AND version=plazas.politica_version;
 IF politica.version IS NULL
    OR jsonb_typeof(politica.politica->'plazas') IS DISTINCT FROM 'object'
    OR politica.politica#>>'{adjudicacion,confirmacion}' IS DISTINCT FROM 'aceptacion_previa'
    OR politica.huella_sha256 IS DISTINCT FROM
       encode(sha256(convert_to(politica.politica::text,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'B135: política de aceptación incompatible' USING ERRCODE='42501'; END IF;

 SELECT * INTO acto FROM vec_bolsa_llamamientos.acto_plaza_oferta
  WHERE oferta_ref=p_oferta AND numero_de_plaza=p_plaza
  ORDER BY secuencia DESC LIMIT 1;
 IF acto.tipo IS DISTINCT FROM 'adjudicada'
    OR acto.secuencia IS DISTINCT FROM p_secuencia
    OR acto.participacion_ref IS DISTINCT FROM p_participacion
    OR acto.recibo_ref IS DISTINCT FROM p_recibo_adjudicacion
    OR acto.responder_antes_de IS NOT NULL
    OR acto.registrado_en<oferta.vence_antes_de
    OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.acto_plaza_oferta_outbox e
       WHERE e.recibo_ref=acto.recibo_ref AND e.oferta_ref=p_oferta
         AND e.bolsa_ref=origen.bolsa_ref AND e.numero_de_plaza=p_plaza
         AND e.secuencia=p_secuencia AND e.tipo='adjudicada'
         AND e.creada_en=acto.registrado_en)
 THEN RAISE EXCEPTION 'B135: plaza no adjudicada actualmente' USING ERRCODE='42501'; END IF;
 SELECT * INTO disposicion FROM vec_bolsa_llamamientos.disposicion_oferta
  WHERE oferta_ref=p_oferta AND participacion_ref=p_participacion;
 SELECT * INTO candidato FROM vec_bolsa_llamamientos.disposicion_oferta_candidato
  WHERE oferta_ref=p_oferta AND participacion_ref=p_participacion;
 IF disposicion.oferta_ref IS NULL OR candidato.oferta_ref IS NULL
    OR disposicion.manifestada_en<oferta.publicada_en
    OR disposicion.manifestada_en>=oferta.vence_antes_de
    OR disposicion.manifestada_en>acto.registrado_en
    OR candidato.registrada_en IS DISTINCT FROM disposicion.manifestada_en
    OR candidato.decision_ref IS NULL
 THEN RAISE EXCEPTION 'B135: disposición previa no acreditada' USING ERRCODE='42501'; END IF;

 persona:=vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(candidato.candidato_ref);
 IF jsonb_typeof(persona) IS DISTINCT FROM 'object'
    OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(persona) k)
       IS DISTINCT FROM ARRAY['estado','persona','vinculo']
    OR persona->>'estado' IS DISTINCT FROM 'acreditado'
    OR jsonb_typeof(persona->'persona') IS DISTINCT FROM 'object'
    OR jsonb_typeof(persona->'vinculo') IS DISTINCT FROM 'object'
    OR persona#>>'{persona,ref}' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR persona#>>'{vinculo,vigente_hasta}' IS NULL
    OR (persona#>>'{vinculo,vigente_hasta}')::timestamptz<=v_ahora
 THEN RAISE EXCEPTION 'B135: persona no acreditada' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object(
  'bolsa_ref',origen.bolsa_ref,'persona_ref',persona#>>'{persona,ref}',
  'candidato_ref',candidato.candidato_ref,'participacion_ref',p_participacion,
  'numero_de_plaza',p_plaza,'secuencia',p_secuencia,
  'recibo_adjudicacion_ref',acto.recibo_ref,
  'adjudicada_en',to_char(acto.registrado_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'politica_version',plazas.politica_version,'recibo_oferta_ref',oferta.recibo_ref,
  'version_padre',origen.version_expediente);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.verificar_plaza_adjudicada_ct_v1(text,text,text,integer,integer,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.verificar_plaza_adjudicada_ct_v1(text,text,text,integer,integer,text,text)
 TO vec_contratacion_temporal_propietario;
DO $post$
DECLARE f oid:=to_regprocedure(
 'vec_bolsa_llamamientos.verificar_plaza_adjudicada_ct_v1(text,text,text,integer,integer,text,text)');
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p
    WHERE p.oid=f AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole
      AND p.prosecdef AND p.provolatile='v'
      AND 'search_path=pg_catalog,pg_temp'=ANY(p.proconfig))
    OR EXISTS(SELECT 1 FROM pg_proc p
       CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE p.oid=f AND a.privilege_type='EXECUTE'
         AND a.grantee NOT IN (p.proowner,'vec_contratacion_temporal_propietario'::regrole))
    OR NOT has_function_privilege('vec_contratacion_temporal_propietario',f,'EXECUTE')
 THEN RAISE EXCEPTION 'B135: propietario, configuración o ACL divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
