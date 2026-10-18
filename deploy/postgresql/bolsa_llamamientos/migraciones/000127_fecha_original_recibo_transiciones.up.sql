\set ON_ERROR_STOP on
-- B127. La lectura y el replay muestran la fecha real del acto guardado.
-- No modifica recibos, outbox, V3, historia ni la firma de las funciones.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000127',0));
DO $pre$
DECLARE p pg_proc%ROWTYPE; r record; f oid;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR to_regclass('vec_bolsa_llamamientos.politica_transiciones_gobierno') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.politica_transiciones_situacion') IS NULL
    OR NOT EXISTS(SELECT 1 FROM pg_attribute a
      WHERE a.attrelid='vec_bolsa_llamamientos.politica_transiciones_gobierno'::regclass
        AND a.attname='publicada_en' AND a.atttypid='timestamp with time zone'::regtype AND NOT a.attisdropped AND a.attnotnull)
    OR NOT EXISTS(SELECT 1 FROM pg_attribute a
      WHERE a.attrelid='vec_bolsa_llamamientos.politica_transiciones_situacion'::regclass
        AND a.attname='publicada_en' AND a.atttypid='timestamp with time zone'::regtype AND NOT a.attisdropped AND a.attnotnull)
 THEN RAISE EXCEPTION 'B127: tablas o columnas ausentes' USING ERRCODE='55000'; END IF;
 FOR r IN SELECT * FROM (VALUES
  ('vec_bolsa_llamamientos.leer_politica_transiciones_gobierno_v1(integer,text)','ce971ad899322a6b2f835701337d8e96c570c7582d0d44205d2516ec3ffd0e76','5ff3a182c4928bc710895e0ffb0f6386416755df2cadf68061f4633357870f8e','{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}','{"search_path=pg_catalog, pg_temp",lock_timeout=2s}'),
  ('vec_bolsa_llamamientos.publicar_politica_transiciones_gobierno_v1(bigint,text[],text,text,text,text,integer,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','53861b75927f226c02dd94a65a148553badf6373a5398e442fe30c4d029b3036','83994f2ae85d866c58b3647a49953f60b33d7e2f6bde2c25c7354a445f20fb18','{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}','{"search_path=pg_catalog, pg_temp",TimeZone=UTC,lock_timeout=2s}')
 ) AS x(firma,huella,fuente,acl,configuracion) LOOP
 f:=to_regprocedure(r.firma);
 IF f IS NULL THEN RAISE EXCEPTION 'B127: función ausente %',r.firma USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM r.huella
    OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM r.fuente
    OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
    OR NOT p.prosecdef OR p.proacl::text IS DISTINCT FROM r.acl
    OR p.proconfig::text IS DISTINCT FROM r.configuracion
 THEN RAISE EXCEPTION 'B127: preimagen o permisos divergentes %',r.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;

-- Cuerpo final literal de leer_politica_transiciones_gobierno_v1.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.leer_politica_transiciones_gobierno_v1(p_catalogo_version integer, p_catalogo_sha256 text)
 RETURNS jsonb
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET lock_timeout TO '2s'
AS $function$
DECLARE v_limites jsonb; v_politica record; v_gobierno record;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' THEN
  RAISE EXCEPTION 'B122: lectura denegada' USING ERRCODE='42501'; END IF;
 v_limites:=vec_catalogos_configurables.leer_limites_transiciones_situacion_bolsa_v1(p_catalogo_version,p_catalogo_sha256);
 SELECT * INTO STRICT v_politica FROM vec_bolsa_llamamientos.politica_transiciones_situacion ORDER BY version DESC LIMIT 1;
 SELECT * INTO v_gobierno FROM vec_bolsa_llamamientos.politica_transiciones_gobierno WHERE version=v_politica.version;
 RETURN jsonb_build_object('version',v_politica.version,
  'transiciones_guardadas',to_jsonb(v_politica.transiciones),
  'limites',v_limites,
  'catalogo_activado',v_gobierno.version IS NOT NULL AND
   v_gobierno.catalogo_version=p_catalogo_version AND v_gobierno.catalogo_sha256=p_catalogo_sha256,
  'recibo_ref',v_gobierno.recibo_ref,
  'publicada_en',CASE WHEN v_gobierno.version IS NULL THEN NULL
   ELSE to_char(v_gobierno.publicada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END);
END $function$;

-- Cuerpo final literal de publicar_politica_transiciones_gobierno_v1.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.publicar_politica_transiciones_gobierno_v1(p_version_esperada bigint, p_transiciones text[], p_motivo text, p_clave text, p_recibo text, p_actor text, p_catalogo_version integer, p_catalogo_sha256 text, p_capacidad bytea, p_decision bytea, p_motivo_autorizacion bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS TABLE(politica jsonb, reutilizada boolean)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE v_consumo record; v_decision jsonb; v_limites jsonb; v_permitidas text[];
 v_canonica text[]; v_vigente record; v_previa record; v_anterior record; v_huella text; v_ahora timestamptz:=clock_timestamp();
 v_contexto text; v_perfil_ref text; v_ambito_sha256 text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_version_esperada IS NULL OR p_version_esperada<1
    OR current_setting('TimeZone')<>'UTC'
    OR (SELECT setting::bigint FROM pg_settings WHERE name='statement_timeout') NOT BETWEEN 1 AND 15000
    OR (SELECT setting::bigint FROM pg_settings WHERE name='idle_in_transaction_session_timeout') NOT BETWEEN 1 AND 20000
    OR p_motivo IS NULL OR p_motivo<>btrim(p_motivo) OR octet_length(p_motivo) NOT BETWEEN 1 AND 1000
    OR p_clave IS NULL OR (octet_length(p_clave) NOT BETWEEN 8 AND 256 OR p_clave !~ '^[A-Za-z0-9:_-]+$')
    OR p_recibo IS NULL OR p_recibo !~ '^recibo:politica-transiciones:[0-9a-f]{64}$'
    OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
 THEN RAISE EXCEPTION 'B122: propuesta inválida' USING ERRCODE='22023'; END IF;
 v_canonica:=vec_bolsa_llamamientos.transiciones_situacion_canonicas(p_transiciones);
 IF v_canonica IS NULL OR v_canonica IS DISTINCT FROM p_transiciones THEN
  RAISE EXCEPTION 'B122: transiciones inválidas' USING ERRCODE='22023'; END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_politica_transiciones_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN v_decision:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B122: autorización inválida' USING ERRCODE='42501'; END;
 v_perfil_ref:=v_decision#>>'{vinculo_autenticacion_actor,perfil_activo_ref}';
 IF v_consumo.consumo_nuevo IS NOT TRUE
    OR v_decision->>'principal_id' IS DISTINCT FROM p_actor
    OR v_decision->>'recurso_ref' IS DISTINCT FROM 'bolsa:politica:transiciones:global'
    OR v_perfil_ref IS NULL OR (octet_length(v_perfil_ref) NOT BETWEEN 3 AND 256 OR v_perfil_ref !~ '^[A-Za-z0-9:_-]+$')
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_decision->>'contexto_recurso_huella_sha256'
 THEN RAISE EXCEPTION 'B122: ámbito global denegado' USING ERRCODE='42501'; END IF;
 -- También en replay se valida V3 fresca contra la configuración normativa
 -- actual. La norma original queda congelada en el acto guardado.
 v_limites:=vec_catalogos_configurables.leer_limites_transiciones_situacion_bolsa_v1(p_catalogo_version,p_catalogo_sha256);
 v_contexto:=format('{"ambitos":{"ambito_ref":"%s","unidad_ref":"%s"},"atributos":{'
  ||'"actor_ref":"%s","catalogo_sha256":"%s","catalogo_version":"%s",'
  ||'"clave_sha256":"%s","motivo_sha256":"%s","perfil_activo_ref":"%s",'
  ||'"transiciones_sha256":"%s","version_esperada":"%s"}}',
   v_limites->>'ambito_ref',v_limites->>'unidad_ref',p_actor,p_catalogo_sha256,p_catalogo_version,
   encode(sha256(convert_to(p_clave,'UTF8')),'hex'),encode(sha256(convert_to(p_motivo,'UTF8')),'hex'),
   v_perfil_ref,encode(sha256(convert_to(array_to_string(v_canonica,','),'UTF8')),'hex'),p_version_esperada);
 IF v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM encode(sha256(convert_to(v_contexto,'UTF8')),'hex') THEN
  RAISE EXCEPTION 'B122: intención no ligada a V3' USING ERRCODE='42501'; END IF;
 v_ambito_sha256:=encode(sha256(convert_to(format('{"ambito_ref":"%s","unidad_ref":"%s"}',
  v_limites->>'ambito_ref',v_limites->>'unidad_ref'),'UTF8')),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:politica_transiciones_situacion',0));
 v_huella:=encode(sha256(convert_to(p_version_esperada::text||chr(31)||array_to_string(v_canonica,',')
  ||chr(31)||p_motivo||chr(31)||p_clave,'UTF8')),'hex');
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.politica_transiciones_gobierno WHERE clave_idempotencia=p_clave;
 IF FOUND THEN
  IF v_previa.actor_ref IS DISTINCT FROM p_actor OR v_previa.version_esperada IS DISTINCT FROM p_version_esperada
     OR v_previa.contenido_sha256 IS DISTINCT FROM v_huella OR v_previa.recibo_ref IS DISTINCT FROM p_recibo
  THEN RAISE EXCEPTION 'B122: clave reutilizada' USING ERRCODE='VBT01'; END IF;
  IF v_previa.perfil_ref IS DISTINCT FROM v_perfil_ref OR v_previa.ambito_sha256 IS DISTINCT FROM v_ambito_sha256 THEN
   RAISE EXCEPTION 'B122: ámbito global denegado' USING ERRCODE='42501'; END IF;
  SELECT * INTO STRICT v_anterior FROM vec_bolsa_llamamientos.politica_transiciones_situacion
   WHERE version=v_previa.version;
  IF v_anterior.transiciones IS DISTINCT FROM v_canonica
     OR v_anterior.catalogo_sha256 IS DISTINCT FROM v_previa.catalogo_sha256
     OR NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.politica_transiciones_outbox o
                    WHERE o.recibo_ref=v_previa.recibo_ref AND o.version=v_previa.version
                      AND o.contenido_sha256=v_previa.contenido_sha256
                      AND o.evento->>'recibo_ref'=v_previa.recibo_ref
                      AND o.evento->>'version'=v_previa.version::text) THEN
   RAISE EXCEPTION 'B122: recibo incompleto' USING ERRCODE='55000'; END IF;
  RETURN QUERY SELECT jsonb_build_object('version',v_previa.version,
   'transiciones_guardadas',to_jsonb(v_anterior.transiciones),
   'limites',v_previa.limites_snapshot,'catalogo_activado',true,'recibo_ref',v_previa.recibo_ref,
   'publicada_en',to_char(v_previa.publicada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),true;
  RETURN;
 END IF;
 SELECT array_agg(valor ORDER BY ordinal) INTO v_permitidas FROM jsonb_array_elements_text(v_limites->'transiciones') WITH ORDINALITY AS t(valor,ordinal);
 IF vec_bolsa_llamamientos.transiciones_situacion_canonicas(v_permitidas) IS DISTINCT FROM v_permitidas
    OR NOT (v_canonica <@ v_permitidas) THEN
  RAISE EXCEPTION 'B122: propuesta fuera de límites normativos' USING ERRCODE='22023'; END IF;
 SELECT * INTO STRICT v_vigente FROM vec_bolsa_llamamientos.politica_transiciones_situacion ORDER BY version DESC LIMIT 1;
 IF v_vigente.version<>p_version_esperada THEN
  RAISE EXCEPTION 'B122: versión obsoleta' USING ERRCODE='VBT01'; END IF;
 INSERT INTO vec_bolsa_llamamientos.politica_transiciones_situacion(version,catalogo_ref,catalogo_sha256,transiciones,publicada_en)
 VALUES(v_vigente.version+1,v_limites->>'catalogo_ref',p_catalogo_sha256,v_canonica,v_ahora);
 INSERT INTO vec_bolsa_llamamientos.politica_transiciones_gobierno
  (version,version_esperada,catalogo_version,catalogo_sha256,limites_snapshot,contexto_recurso_sha256,ambito_sha256,
   actor_ref,perfil_ref,motivo,clave_idempotencia,recibo_ref,
   contenido_sha256,decision_ref,auditoria_ref,publicada_en)
 VALUES(v_vigente.version+1,p_version_esperada,p_catalogo_version,p_catalogo_sha256,v_limites,
   v_decision->>'contexto_recurso_huella_sha256',v_ambito_sha256,p_actor,v_perfil_ref,p_motivo,p_clave,p_recibo,
   v_huella,v_consumo.decision_ref,v_consumo.auditoria_ref,v_ahora);
 INSERT INTO vec_bolsa_llamamientos.politica_transiciones_outbox
  (recibo_ref,version,contenido_sha256,esquema,evento,evento_sha256,creada_en)
 SELECT p_recibo,v_vigente.version+1,v_huella,'vec.bolsa.politica-transiciones.publicada.v1',e,
        encode(sha256(convert_to(e::text,'UTF8')),'hex'),v_ahora
 FROM (SELECT jsonb_build_object('esquema','vec.bolsa.politica-transiciones.publicada.v1',
   'recibo_ref',p_recibo,'version',v_vigente.version+1,'contenido_sha256',v_huella,
   'catalogo_ref',v_limites->>'catalogo_ref','catalogo_sha256',p_catalogo_sha256) e) v;
 RETURN QUERY SELECT vec_bolsa_llamamientos.leer_politica_transiciones_gobierno_v1(p_catalogo_version,p_catalogo_sha256),false;
END $function$;

DO $post$
DECLARE p pg_proc%ROWTYPE; r record; f oid;
BEGIN
 FOR r IN SELECT * FROM (VALUES
  ('vec_bolsa_llamamientos.leer_politica_transiciones_gobierno_v1(integer,text)','4554f89718954ab1547cfa1ceaab532ebc4a8b489dbd8989473fb64fed369ece','c49ba166bda9284475a5473f5f6167024cd85691f3fb0d4b539dca625db9fd54','{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}','{"search_path=pg_catalog, pg_temp",lock_timeout=2s}'),
  ('vec_bolsa_llamamientos.publicar_politica_transiciones_gobierno_v1(bigint,text[],text,text,text,text,integer,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','1a569260e99e4003b52d559eae6e15a247afbe40abc69bc2202a96ea35698a56','90fbc696aeaa1819cdfbe81ec3ef6a0616f36bee94d0fe231fa0329d8c6eea8f','{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}','{"search_path=pg_catalog, pg_temp",TimeZone=UTC,lock_timeout=2s}')
 ) AS x(firma,huella,fuente,acl,configuracion) LOOP
 f:=to_regprocedure(r.firma);
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM r.huella
    OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM r.fuente
    OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
    OR NOT p.prosecdef OR p.proacl::text IS DISTINCT FROM r.acl
    OR p.proconfig::text IS DISTINCT FROM r.configuracion
 THEN RAISE EXCEPTION 'B127: postimagen o permisos divergentes %',r.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $post$;
COMMIT;
