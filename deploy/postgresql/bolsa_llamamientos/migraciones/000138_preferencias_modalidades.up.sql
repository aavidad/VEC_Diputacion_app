\set ON_ERROR_STOP on
-- B138: selección propia de modalidades, con historia separada del orden de bolsas.
-- Instalar después de AD296 y AUT104. Las lecturas reutilizan B104 y B116.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000138',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR current_setting('server_encoding')<>'UTF8'
 OR to_regclass('vec_bolsa_llamamientos.preferencia_modalidades_version') IS NOT NULL
 OR to_regprocedure('vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(text,text[],jsonb,bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regrole('vec_bolsa_llamamientos_portal_externo') IS NULL
 OR to_regrole('vec_bolsa_llamamientos_preferencias_interna') IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_autorizacion'
   AND p.proname='cotejar_campos_preferencias_modalidades_aut104'
   AND p.proargtypes='3802 3802'::oidvector AND p.pronargs=2
   AND p.prorettype=16::oid AND p.prokind='f')
 -- AD296 debe dejar las dos fachadas auditadas con proyecciones de 8/5 campos.
 -- Se cotejan cuerpo y privilegios antes de sustituir las lectoras de Bolsa.
 OR (SELECT encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') FROM pg_proc p
     WHERE p.oid='vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_v1(bytea,bytea,jsonb)'::regprocedure
      AND p.prokind='f' AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s','statement_timeout=10s']::text[]
      AND p.proacl::text='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_bolsa_llamamientos_propietario=X/vec_autorizacion_atestada_v3_propietario}')
    IS DISTINCT FROM '014ea7fbf8229619c736d0ebcea7964634dbd7c545b9b6a76d659d5a2a334bf8'
 OR (SELECT encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') FROM pg_proc p
     WHERE p.oid='vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_interna_v1(bytea,bytea,jsonb)'::regprocedure
      AND p.prokind='f' AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s','statement_timeout=10s']::text[]
      AND p.proacl::text='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_bolsa_llamamientos_propietario=X/vec_autorizacion_atestada_v3_propietario}')
    IS DISTINCT FROM '0a6ec140c5b7b4c52a2242269418397c527f446fc21c131af91a7cd0d728c5dc'
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR has_function_privilege('vec_bolsa_llamamientos_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') IS NOT TRUE
 OR has_function_privilege('vec_bolsa_llamamientos_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') IS NOT TRUE
 OR (SELECT encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') FROM pg_proc p
     WHERE p.oid='vec_bolsa_llamamientos.consultar_preferencias_bolsas_v1(text,bytea,bytea,jsonb)'::regprocedure
      AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=10s']::text[]
      AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_portal_externo=X/vec_bolsa_llamamientos_propietario}')
    IS DISTINCT FROM 'a96b43a8bfc87ca040fa1d49591f07d71569f88921053d56e556e2e9d5113c07'
 OR (SELECT encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') FROM pg_proc p
     WHERE p.oid='vec_bolsa_llamamientos.consultar_preferencias_bolsas_interna_v1(text,bytea,bytea,jsonb)'::regprocedure
      AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=10s']::text[]
      AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_preferencias_interna=X/vec_bolsa_llamamientos_propietario}')
    IS DISTINCT FROM '2e2965e33cdf46f45f612667eddf746d4674f1fb82f50103891e7713840056d0'
 THEN RAISE EXCEPTION 'B138: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_bolsa_llamamientos.preferencia_modalidades_version (
 candidato_ref text NOT NULL CHECK(candidato_ref ~ '^can_[A-Za-z0-9_-]{22,128}$'),
 version bigint NOT NULL CHECK(version BETWEEN 1 AND 2147483647),
 modalidades text[] CHECK(modalidades IS NULL OR
  (coalesce(array_ndims(modalidades),1)=1 AND cardinality(modalidades)<=256
   AND array_position(modalidades,NULL) IS NULL)),
 catalogo_snapshot jsonb NOT NULL CHECK(jsonb_typeof(catalogo_snapshot)='array'),
 clave_idempotencia text NOT NULL CHECK(clave_idempotencia ~ '^@[0-9a-f]{64}$'),
 solicitud_huella_sha256 text NOT NULL CHECK(solicitud_huella_sha256 ~ '^[0-9a-f]{64}$'),
 contexto_huella_sha256 text NOT NULL CHECK(contexto_huella_sha256 ~ '^[0-9a-f]{64}$'),
 actor_ref text NOT NULL CHECK(octet_length(actor_ref) BETWEEN 1 AND 256),
 perfil_ref text NOT NULL CHECK(octet_length(perfil_ref) BETWEEN 1 AND 256),
 superficie text NOT NULL CHECK(superficie IN ('externa_personal','interna_corporativa')),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^recibo:preferencias-modalidades:[0-9a-f]{64}$'),
 decision_ref text NOT NULL UNIQUE CHECK(octet_length(decision_ref) BETWEEN 1 AND 256),
 auditoria_ref text NOT NULL CHECK(octet_length(auditoria_ref) BETWEEN 1 AND 256),
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(candidato_ref,version), UNIQUE(candidato_ref,clave_idempotencia)
);
ALTER TABLE vec_bolsa_llamamientos.preferencia_modalidades_version ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.preferencia_modalidades_version FORCE ROW LEVEL SECURITY;
CREATE POLICY preferencia_modalidades_version_propietario ON vec_bolsa_llamamientos.preferencia_modalidades_version
 TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.preferencia_modalidades_version FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.preferencia_modalidades_version FROM PUBLIC;
CREATE TRIGGER preferencia_modalidades_version_inmutable BEFORE UPDATE OR DELETE
 ON vec_bolsa_llamamientos.preferencia_modalidades_version FOR EACH ROW
 EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE TRIGGER preferencia_modalidades_version_no_truncar BEFORE TRUNCATE
 ON vec_bolsa_llamamientos.preferencia_modalidades_version FOR EACH STATEMENT
 EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- El contexto se forma con los mismos bytes que encoding/json en Go. Candidato,
-- claves y huellas son ASCII validados; nunca se usa jsonb::text para una huella.
CREATE FUNCTION vec_bolsa_llamamientos.huella_solicitud_modalidades_v1(
 p_candidato text,p_version bigint,p_modalidades text[],p_disponibles jsonb)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE v_canon text:='';v_valor text;v_opcion jsonb;v_seleccion text[];v_snapshot jsonb;
BEGIN
 IF p_candidato IS NULL OR p_candidato !~ '^can_[A-Za-z0-9_-]{22,128}$'
 OR p_version IS NULL OR p_version NOT BETWEEN 0 AND 2147483646
 OR jsonb_typeof(p_disponibles) IS DISTINCT FROM 'array' OR jsonb_array_length(p_disponibles) NOT BETWEEN 1 AND 256
 THEN RAISE EXCEPTION 'B138: huella de selección inválida' USING ERRCODE='22023'; END IF;
 IF p_modalidades IS NOT NULL THEN
  SELECT coalesce(array_agg(s ORDER BY s),ARRAY[]::text[]) INTO v_seleccion FROM unnest(p_modalidades) s;
 END IF;
 SELECT jsonb_agg(e ORDER BY e->>'clave') INTO v_snapshot FROM jsonb_array_elements(p_disponibles) e;
 FOREACH v_valor IN ARRAY ARRAY['vec.bolsa.preferencias-modalidades.v1',p_candidato,p_version::text,
  CASE WHEN p_modalidades IS NULL THEN 'todas' ELSE 'seleccion' END,
  coalesce(cardinality(v_seleccion),0)::text] LOOP
  v_canon:=v_canon||octet_length(v_valor)::text||':'||v_valor;
 END LOOP;
 IF v_seleccion IS NOT NULL THEN FOREACH v_valor IN ARRAY v_seleccion LOOP
  v_canon:=v_canon||octet_length(v_valor)::text||':'||v_valor;
 END LOOP; END IF;
 v_valor:=jsonb_array_length(v_snapshot)::text;
 v_canon:=v_canon||octet_length(v_valor)::text||':'||v_valor;
 FOR v_opcion IN SELECT e FROM jsonb_array_elements(v_snapshot) e LOOP
  FOREACH v_valor IN ARRAY ARRAY[v_opcion->>'clave',v_opcion->>'catalogo_id',
   v_opcion->>'catalogo_version',v_opcion->>'catalogo_huella_sha256',v_opcion->>'entrada_clave'] LOOP
   v_canon:=v_canon||octet_length(v_valor)::text||':'||v_valor;
  END LOOP;
 END LOOP;
 RETURN encode(sha256(convert_to(v_canon,'UTF8')),'hex');
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.huella_solicitud_modalidades_v1(text,bigint,text[],jsonb) FROM PUBLIC;
CREATE FUNCTION vec_bolsa_llamamientos.huella_contexto_modalidades_v1(p_candidato text,p_huella text)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE v_contexto text;
BEGIN
 IF p_candidato IS NULL OR p_candidato !~ '^can_[A-Za-z0-9_-]{22,128}$'
 OR p_huella IS NULL OR p_huella !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'B138: contexto de modalidades inválido' USING ERRCODE='22023'; END IF;
 v_contexto:='{"ambitos":{"candidato_ref":'||to_json(p_candidato)::text||
  '},"atributos":{"material_sha256":"'||p_huella||
  '","propiedad":"candidato","tipo_preferencia":"modalidades"}}';
 RETURN encode(sha256(convert_to(v_contexto,'UTF8')),'hex');
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.huella_contexto_modalidades_v1(text,text) FROM PUBLIC;

CREATE FUNCTION vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(
 p_candidato text,p_modalidades text[],p_disponibles jsonb,p_version_esperada bigint,
 p_clave text,p_consultada_en timestamptz,p_capacidad bytea,p_decision bytea,
 p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(reutilizada boolean,recibo_ref text,version bigint,modalidades text[],registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE c jsonb;d jsonb;x jsonb;v_snapshot jsonb;
 v_superficie text;v_huella text;v_contexto_h text;
 v_seleccion text[];v_version bigint;v_previa vec_bolsa_llamamientos.preferencia_modalidades_version%ROWTYPE;
 v_clave_interna text;v_recibo text;v_consumo record;v_ahora timestamptz(6):=clock_timestamp();
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR current_setting('role')<>'none'
 OR session_user NOT IN ('vec_externo_bolsa_desarrollo','vec_interno_bolsa_preferencias_desarrollo')
 OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR p_candidato IS NULL OR p_candidato !~ '^can_[A-Za-z0-9_-]{22,128}$'
 OR p_version_esperada IS NULL OR p_version_esperada NOT BETWEEN 0 AND 2147483646
 OR p_clave IS NULL OR p_clave !~ '^[A-Za-z0-9._:-]{8,128}$'
 OR p_modalidades IS NOT NULL AND (coalesce(array_ndims(p_modalidades),1)<>1
      OR cardinality(p_modalidades)>256 OR array_position(p_modalidades,NULL) IS NOT NULL)
 OR jsonb_typeof(p_disponibles) IS DISTINCT FROM 'array'
 OR jsonb_array_length(p_disponibles) NOT BETWEEN 1 AND 256
 OR p_consultada_en IS NULL OR NOT isfinite(p_consultada_en)
 OR p_consultada_en<v_ahora-interval '5 minutes' OR p_consultada_en>v_ahora+interval '1 minute'
 OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
 OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL
 OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL
 THEN RAISE EXCEPTION 'B138: selección inválida' USING ERRCODE='22023'; END IF;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(p_disponibles) e WHERE jsonb_typeof(e)<>'object'
   OR (SELECT count(*) FROM jsonb_object_keys(e))<>5
   OR (e ?& ARRAY['clave','catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave']) IS NOT TRUE
   OR jsonb_typeof(e->'clave') IS DISTINCT FROM 'string'
   OR e->>'clave' !~ '^[a-z][a-z0-9._-]{1,79}$'
   OR e->>'catalogo_id' IS DISTINCT FROM 'vec.contratacion_temporal.reglas'
   OR jsonb_typeof(e->'catalogo_version') IS DISTINCT FROM 'number'
   OR e->>'catalogo_version' !~ '^[1-9][0-9]{0,18}$'
   OR jsonb_typeof(e->'catalogo_huella_sha256') IS DISTINCT FROM 'string'
   OR (e->>'catalogo_huella_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
   OR e->>'entrada_clave' IS DISTINCT FROM 'c12.modalidad.'||(e->>'clave'))
 OR (SELECT count(DISTINCT e->>'clave') FROM jsonb_array_elements(p_disponibles) e)
    <>jsonb_array_length(p_disponibles)
 OR EXISTS(SELECT 1 FROM unnest(p_modalidades) s WHERE s !~ '^[a-z][a-z0-9._-]{1,79}$')
 OR (p_modalidades IS NOT NULL AND
    (SELECT count(DISTINCT s) FROM unnest(p_modalidades) s) IS DISTINCT FROM cardinality(p_modalidades))
 THEN RAISE EXCEPTION 'B138: catálogo o selección divergente' USING ERRCODE='22023'; END IF;
 SELECT jsonb_agg(e ORDER BY e->>'clave') INTO v_snapshot FROM jsonb_array_elements(p_disponibles) e;
 IF p_modalidades IS NOT NULL THEN
  SELECT coalesce(array_agg(s ORDER BY s),ARRAY[]::text[]) INTO v_seleccion FROM unnest(p_modalidades) s;
 END IF;
 v_superficie:=CASE session_user WHEN 'vec_externo_bolsa_desarrollo' THEN 'externa_personal'
  ELSE 'interna_corporativa' END;
 v_huella:=vec_bolsa_llamamientos.huella_solicitud_modalidades_v1(
  p_candidato,p_version_esperada,p_modalidades,v_snapshot);
 v_contexto_h:=vec_bolsa_llamamientos.huella_contexto_modalidades_v1(p_candidato,v_huella);
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'B138: material ilegible' USING ERRCODE='22023'; END;
 IF c->>'operacion' IS DISTINCT FROM 'bolsa.preferencias_bolsas.ordenar_propias'
 OR c->>'audiencia_consumo' IS DISTINCT FROM
  (CASE WHEN v_superficie='externa_personal' THEN 'vec.bolsa.preferencias-bolsas.ordenar.v1'
   ELSE 'vec.bolsa.preferencias-bolsas.ordenar.interna.v1' END)
 OR c->>'efecto_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato
 OR c->>'huella_efecto_sha256' IS DISTINCT FROM v_contexto_h
 OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'participaciones_candidato'
 OR d->>'finalidad' IS DISTINCT FROM 'gestion_preferencias_bolsas_propias'
 OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_h
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM v_superficie
 OR d->>'principal_id' IS DISTINCT FROM x->>'principal_ref'
 OR d->>'perfil_activo_ref' IS DISTINCT FROM x->>'perfil_activo_ref'
 OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array'
 OR (SELECT count(*) FROM jsonb_array_elements(x->'vinculos') e
      WHERE e->>'tipo'='candidato' AND e->>'estado'='activo')<>1
 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(x->'vinculos') e
      WHERE e->>'tipo'='candidato' AND e->>'estado'='activo' AND e->>'referencia'=p_candidato)
 THEN RAISE EXCEPTION 'B138: selección no ligada a la decisión' USING ERRCODE='42501'; END IF;
 IF v_superficie='externa_personal' THEN
  SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ELSE
  SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 END IF;
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM c->>'efecto_ref'
 OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_h
 THEN RAISE EXCEPTION 'B138: consumo divergente' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:preferencias-modalidades:'||p_candidato,0));
 v_clave_interna:='@'||encode(sha256(convert_to(v_superficie||chr(31)||p_clave,'UTF8')),'hex');
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.preferencia_modalidades_version p
 WHERE p.candidato_ref=p_candidato AND p.clave_idempotencia=v_clave_interna;
 IF FOUND THEN
  IF v_previa.actor_ref IS DISTINCT FROM x->>'principal_ref'
   OR v_previa.perfil_ref IS DISTINCT FROM x->>'perfil_activo_ref'
   OR v_previa.superficie IS DISTINCT FROM v_superficie
   OR v_previa.version IS DISTINCT FROM p_version_esperada+1
   OR v_previa.modalidades IS DISTINCT FROM v_seleccion
   OR v_previa.solicitud_huella_sha256 IS DISTINCT FROM
    vec_bolsa_llamamientos.huella_solicitud_modalidades_v1(
     p_candidato,p_version_esperada,v_previa.modalidades,v_previa.catalogo_snapshot)
   OR v_previa.contexto_huella_sha256 IS DISTINCT FROM
    vec_bolsa_llamamientos.huella_contexto_modalidades_v1(
     p_candidato,v_previa.solicitud_huella_sha256)
   OR v_previa.recibo_ref IS DISTINCT FROM 'recibo:preferencias-modalidades:'||encode(sha256(convert_to(
    p_candidato||chr(31)||v_clave_interna||chr(31)||v_previa.solicitud_huella_sha256,'UTF8')),'hex')
  THEN RAISE EXCEPTION 'B138: clave reutilizada con otra selección' USING ERRCODE='VBP02'; END IF;
  RETURN QUERY SELECT true,v_previa.recibo_ref,v_previa.version,v_previa.modalidades,v_previa.registrada_en;
  RETURN;
 END IF;
 IF EXISTS(SELECT 1 FROM unnest(p_modalidades) s
  WHERE NOT EXISTS(SELECT 1 FROM jsonb_array_elements(v_snapshot) e WHERE e->>'clave'=s))
 THEN RAISE EXCEPTION 'B138: modalidad no disponible' USING ERRCODE='22023'; END IF;
 SELECT coalesce(max(p.version),0) INTO v_version FROM vec_bolsa_llamamientos.preferencia_modalidades_version p
 WHERE p.candidato_ref=p_candidato;
 IF v_version<>p_version_esperada THEN
  RAISE EXCEPTION 'B138: versión de modalidades desactualizada' USING ERRCODE='VBP01'; END IF;
 v_recibo:='recibo:preferencias-modalidades:'||encode(sha256(convert_to(
  p_candidato||chr(31)||v_clave_interna||chr(31)||v_huella,'UTF8')),'hex');
 v_ahora:=clock_timestamp();
 INSERT INTO vec_bolsa_llamamientos.preferencia_modalidades_version(
  candidato_ref,version,modalidades,catalogo_snapshot,clave_idempotencia,solicitud_huella_sha256,
  contexto_huella_sha256,actor_ref,perfil_ref,superficie,recibo_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(p_candidato,v_version+1,v_seleccion,v_snapshot,v_clave_interna,v_huella,
  v_contexto_h,x->>'principal_ref',x->>'perfil_activo_ref',v_superficie,v_recibo,
  v_consumo.decision_ref,v_consumo.auditoria_ref,v_ahora);
 RETURN QUERY SELECT false,v_recibo,v_version+1,v_seleccion,v_ahora;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(
 text,text[],jsonb,bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(
 text,text[],jsonb,bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_portal_externo,vec_bolsa_llamamientos_preferencias_interna;

-- Dos definiciones literales instaladas B104/B116, con una bifurcación fija
-- por los cinco u ocho campos del descriptor. Se generan desde pg_get_functiondef
-- y conservan dueño, firma, ACL y demás cuerpo exacto.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.consultar_preferencias_bolsas_v1(p_candidato_ref text, p_contexto_canonico bytea, p_vinculo_canonico bytea, p_orden jsonb)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '10s'
AS $function$
DECLARE v_estado jsonb; v_auditoria record; v_corte timestamptz(6):=clock_timestamp();
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR session_user <> 'vec_externo_bolsa_desarrollo'
    OR current_setting('transaction_isolation') <> 'serializable'
    OR current_setting('transaction_read_only') <> 'off'
    OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_contexto_canonico IS NULL OR p_vinculo_canonico IS NULL
    OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
    OR p_orden->>'recurso_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
 THEN RAISE EXCEPTION 'B104: lectura nominal inválida' USING ERRCODE='42501'; END IF;
 IF p_orden->'campos_permitidos' IS NOT DISTINCT FROM
    '["version","modalidades_guardadas","modalidades_disponibles","recibo_ref","registrado_en"]'::jsonb THEN
  SELECT jsonb_build_object('version',p.version,'modalidades_guardadas',p.modalidades,
   'modalidades_disponibles','[]'::jsonb,'recibo_ref',p.recibo_ref,'registrado_en',p.registrada_en)
  INTO v_estado FROM vec_bolsa_llamamientos.preferencia_modalidades_version p
  WHERE p.candidato_ref=p_candidato_ref ORDER BY p.version DESC LIMIT 1;
  IF NOT FOUND THEN v_estado:=jsonb_build_object('version',0,'modalidades_guardadas',NULL,
   'modalidades_disponibles','[]'::jsonb,'recibo_ref',NULL,'registrado_en',NULL); END IF;
 ELSIF p_orden->'campos_permitidos' IS NOT DISTINCT FROM
    '["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","recibo_ref","registrado_en"]'::jsonb THEN
  v_estado:=vec_bolsa_llamamientos.preferencias_bolsas_estado_interno_v1(p_candidato_ref,v_corte);
 ELSE RAISE EXCEPTION 'B138: proyección de preferencias inválida' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_auditoria
 FROM vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_v1(
  p_contexto_canonico,p_vinculo_canonico,p_orden);
 IF v_estado IS NULL OR jsonb_typeof(v_estado) IS DISTINCT FROM 'object'
    OR v_auditoria.auditoria_ref IS NULL OR v_auditoria.huella_sha256 !~ '^[0-9a-f]{64}$'
    OR v_auditoria.correlacion_ref IS DISTINCT FROM p_orden->>'correlacion_ref'
    OR v_auditoria.registrada_en IS NULL
 THEN RAISE EXCEPTION 'B104: lectura no asentada' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('estado',v_estado,'auditoria_ref',v_auditoria.auditoria_ref,
  'auditoria_huella_sha256',v_auditoria.huella_sha256,
  'correlacion_ref',v_auditoria.correlacion_ref,'auditada_en',v_auditoria.registrada_en);
END $function$;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.consultar_preferencias_bolsas_interna_v1(p_candidato_ref text, p_contexto_canonico bytea, p_vinculo_canonico bytea, p_orden jsonb)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '10s'
AS $function$
DECLARE v_estado jsonb; v_auditoria record; v_corte timestamptz(6):=clock_timestamp();
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR session_user <> 'vec_interno_bolsa_preferencias_desarrollo'
    OR current_setting('transaction_isolation') <> 'serializable'
    OR current_setting('transaction_read_only') <> 'off'
    OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_contexto_canonico IS NULL OR p_vinculo_canonico IS NULL
    OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
    OR p_orden->>'recurso_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
 THEN RAISE EXCEPTION 'B116: lectura nominal inválida' USING ERRCODE='42501'; END IF;
 IF p_orden->'campos_permitidos' IS NOT DISTINCT FROM
    '["version","modalidades_guardadas","modalidades_disponibles","recibo_ref","registrado_en"]'::jsonb THEN
  SELECT jsonb_build_object('version',p.version,'modalidades_guardadas',p.modalidades,
   'modalidades_disponibles','[]'::jsonb,'recibo_ref',p.recibo_ref,'registrado_en',p.registrada_en)
  INTO v_estado FROM vec_bolsa_llamamientos.preferencia_modalidades_version p
  WHERE p.candidato_ref=p_candidato_ref ORDER BY p.version DESC LIMIT 1;
  IF NOT FOUND THEN v_estado:=jsonb_build_object('version',0,'modalidades_guardadas',NULL,
   'modalidades_disponibles','[]'::jsonb,'recibo_ref',NULL,'registrado_en',NULL); END IF;
 ELSIF p_orden->'campos_permitidos' IS NOT DISTINCT FROM
    '["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","recibo_ref","registrado_en"]'::jsonb THEN
  v_estado:=vec_bolsa_llamamientos.preferencias_bolsas_estado_interno_v1(p_candidato_ref,v_corte);
 ELSE RAISE EXCEPTION 'B138: proyección de preferencias inválida' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_auditoria
 FROM vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_interna_v1(
  p_contexto_canonico,p_vinculo_canonico,p_orden);
 IF v_estado IS NULL OR jsonb_typeof(v_estado) IS DISTINCT FROM 'object'
    OR v_auditoria.auditoria_ref IS NULL OR v_auditoria.huella_sha256 !~ '^[0-9a-f]{64}$'
    OR v_auditoria.correlacion_ref IS DISTINCT FROM p_orden->>'correlacion_ref'
    OR v_auditoria.registrada_en IS NULL
 THEN RAISE EXCEPTION 'B116: lectura no asentada' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('estado',v_estado,'auditoria_ref',v_auditoria.auditoria_ref,
  'auditoria_huella_sha256',v_auditoria.huella_sha256,
  'correlacion_ref',v_auditoria.correlacion_ref,'auditada_en',v_auditoria.registrada_en);
END $function$;

DO $postlecturas$ BEGIN
 IF false
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_bolsa_llamamientos.consultar_preferencias_bolsas_v1(text,bytea,bytea,jsonb)'::regprocedure
  AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='222e0b9b56ce6a8209b8a27ee1502157f6dbe57e808436c4b220f5d7f8645099'
  AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=10s']::text[]
  AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_portal_externo=X/vec_bolsa_llamamientos_propietario}')
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_bolsa_llamamientos.consultar_preferencias_bolsas_interna_v1(text,bytea,bytea,jsonb)'::regprocedure
  AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='b3296f430ed4c984a0e68cea7511f2d844e5861cc7fc803bfd66bcee52572c59'
  AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=10s']::text[]
  AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_preferencias_interna=X/vec_bolsa_llamamientos_propietario}')
 THEN RAISE EXCEPTION 'B138: funciones de lectura divergentes' USING ERRCODE='55000'; END IF;
END $postlecturas$;

DO $post$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_class t WHERE t.oid='vec_bolsa_llamamientos.preferencia_modalidades_version'::regclass
  AND t.relowner='vec_bolsa_llamamientos_propietario'::regrole
  AND t.relrowsecurity AND t.relforcerowsecurity
  AND (SELECT count(*)=8 AND count(DISTINCT a.privilege_type)=8 AND
    bool_and(a.grantee=t.relowner AND a.grantor=t.relowner AND NOT a.is_grantable
     AND a.privilege_type IN ('SELECT','INSERT','UPDATE','DELETE','TRUNCATE','REFERENCES','TRIGGER','MAINTAIN'))
   FROM aclexplode(coalesce(t.relacl,acldefault('r',t.relowner))) a) IS TRUE)
 OR (SELECT count(*) FROM pg_trigger t WHERE t.tgrelid='vec_bolsa_llamamientos.preferencia_modalidades_version'::regclass
  AND t.tgname IN ('preferencia_modalidades_version_inmutable','preferencia_modalidades_version_no_truncar')
  AND t.tgenabled='O')<>2
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid IN ('vec_bolsa_llamamientos.huella_solicitud_modalidades_v1(text,bigint,text[],jsonb)'::regprocedure,
   'vec_bolsa_llamamientos.huella_contexto_modalidades_v1(text,text)'::regprocedure,
   'vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(text,text[],jsonb,bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
   AND a.grantee=0 AND a.privilege_type='EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_portal_externo','vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(text,text[],jsonb,bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') IS NOT TRUE
 OR has_function_privilege('vec_bolsa_llamamientos_preferencias_interna','vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(text,text[],jsonb,bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') IS NOT TRUE
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(text,text[],jsonb,bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
  AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[])
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid='vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(text,text[],jsonb,bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
  AND a.privilege_type='EXECUTE')<>3
 THEN RAISE EXCEPTION 'B138: tabla o ACL divergentes' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
