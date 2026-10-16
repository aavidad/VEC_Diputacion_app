\set ON_ERROR_STOP on
-- CT226 / F1-37: sonda por organización; conserva provisión, historia y recibos.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000226',0));
DO $pre$
DECLARE p record; actual text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_contratacion_temporal_propietario'
    OR to_regclass('vec_contratacion_temporal.catalogo_plantillas_historia_v1') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2(text,text,bigint,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT226: preimagen incompatible' USING ERRCODE='55000'; END IF;
 SELECT x.* INTO p FROM pg_proc x
  WHERE x.oid=to_regprocedure('vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(text,bigint,text)');
 IF NOT FOUND THEN RAISE EXCEPTION 'CT226: CT218 ausente' USING ERRCODE='55000'; END IF;
 actual:=encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex');
 IF actual IS DISTINCT FROM '8c3d075bef2b6fcd76e8248a209a0967b1e891c00baaf6fa4bd165723403a145'
 OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
 OR p.proacl IS DISTINCT FROM ARRAY[
  'vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
  'vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[]
 OR p.prosecdef IS DISTINCT FROM true OR p.provolatile IS DISTINCT FROM 's'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[]
 THEN RAISE EXCEPTION 'CT226: definición instalada distinta %',actual USING ERRCODE='55000'; END IF;
END $pre$;
LOCK TABLE vec_contratacion_temporal.catalogo_plantillas_historia_v1,
 vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1,
 vec_contratacion_temporal.catalogo_plantillas_outbox_v1 IN SHARE MODE;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2(p_organizacion_ref text, p_huella_sha256 text, p_version bigint, p_fuente_ref text)
 RETURNS boolean
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path = pg_catalog, pg_temp
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE b record; a record; p record; pa record; o record;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR p_organizacion_ref IS NULL OR p_organizacion_ref !~ '^organizacion:[A-Za-z0-9._:/#-]{1,147}$'
    OR p_huella_sha256 IS NULL OR p_huella_sha256 !~ '^[0-9a-f]{64}$'
    OR p_version IS NULL OR p_version<1 OR p_fuente_ref IS NULL
 THEN RAISE EXCEPTION 'CT-131: comprobación denegada' USING ERRCODE='42501'; END IF;

 -- La primera fila sigue siendo la provisión independiente. Sus valores ya
 -- no se comparan con el fichero activo, que puede ser una publicación nueva.
 SELECT * INTO b FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 ORDER BY secuencia LIMIT 1;
 IF NOT FOUND OR b.origen IS DISTINCT FROM 'bootstrap' OR b.estado IS DISTINCT FROM 'publicado'
    OR b.catalogo->>'estado' IS DISTINCT FROM 'publicado'
    OR b.catalogo->>'version' IS DISTINCT FROM b.version::text
    OR b.catalogo->>'revision' IS DISTINCT FROM b.revision::text
    OR b.catalogo->>'fuente_ref' IS DISTINCT FROM b.provision_fuente_ref
    OR b.catalogo->>'aprobacion_ref' IS DISTINCT FROM b.provision_aprobacion_ref
    OR b.catalogo_huella_sha256 !~ '^[0-9a-f]{64}$'
    OR b.contenido_json_sha256 IS DISTINCT FROM encode(sha256(convert_to(b.catalogo::text,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'CT218: publicación inicial no provisionada o divergente' USING ERRCODE='55000'; END IF;
 SELECT * INTO a FROM vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1
  WHERE historia_secuencia=b.secuencia;
 IF NOT FOUND OR a.auditoria_ref IS DISTINCT FROM b.auditoria_ref
    OR a.solicitud_huella_sha256 IS DISTINCT FROM b.consumo_huella_sha256
    OR a.fuente_ref IS DISTINCT FROM b.provision_fuente_ref
    OR a.aprobacion_ref IS DISTINCT FROM b.provision_aprobacion_ref
    OR a.instalador_ref IS DISTINCT FROM b.actor_ref OR a.registrada_en IS DISTINCT FROM b.registrada_en
 THEN RAISE EXCEPTION 'CT218: auditoría de provisión ausente o divergente' USING ERRCODE='55000'; END IF;

 -- La provisión inicial es común. Una publicación de otra organización
 -- nunca desplaza la última publicación de la organización solicitada.
 SELECT * INTO p FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1
  WHERE estado='publicado' AND (organizacion_ref=p_organizacion_ref OR secuencia=b.secuencia)
  ORDER BY secuencia DESC LIMIT 1;
 IF NOT FOUND OR (p.secuencia<>b.secuencia AND p.organizacion_ref IS DISTINCT FROM p_organizacion_ref)
    OR p.catalogo->>'estado' IS DISTINCT FROM 'publicado'
    OR p.catalogo->>'id' IS DISTINCT FROM 'vec.contratacion_temporal.plantillas_documentos'
    OR p.catalogo->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR p.catalogo->>'version' IS DISTINCT FROM p.version::text
    OR p.catalogo->>'revision' IS DISTINCT FROM p.revision::text
    OR p.catalogo_huella_sha256 IS DISTINCT FROM p_huella_sha256
    OR p.version IS DISTINCT FROM p_version
    OR p.catalogo->>'fuente_ref' IS DISTINCT FROM p_fuente_ref
    OR p.contenido_json_sha256 IS DISTINCT FROM encode(sha256(convert_to(p.catalogo::text,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'CT218: publicación vigente ausente o divergente' USING ERRCODE='55000'; END IF;
 IF p.secuencia=b.secuencia THEN
  IF p.origen IS DISTINCT FROM 'bootstrap' THEN
   RAISE EXCEPTION 'CT218: procedencia inicial divergente' USING ERRCODE='55000'; END IF;
 ELSE
  IF p.origen IS DISTINCT FROM 'publicacion' OR p.recibo_ref IS NULL
     OR p.solicitud_huella_sha256 !~ '^[0-9a-f]{64}$'
     OR p.consumo_huella_sha256 !~ '^[0-9a-f]{64}$'
     OR p.provision_fuente_ref IS NOT NULL OR p.provision_aprobacion_ref IS NOT NULL
  THEN RAISE EXCEPTION 'CT218: procedencia de publicación divergente' USING ERRCODE='55000'; END IF;
  SELECT * INTO pa FROM vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1
   WHERE historia_secuencia=p.secuencia;
  IF p.auditoria_ref ~ '^provision:[0-9a-f-]{36}$' THEN
   -- Publicación sintética de datos: conserva su recibo y auditoría de provisión.
   IF NOT FOUND OR p.consumo_huella_sha256 IS DISTINCT FROM p.solicitud_huella_sha256
      OR pa.recibo_ref IS DISTINCT FROM p.recibo_ref
      OR pa.auditoria_ref IS DISTINCT FROM p.auditoria_ref
      OR pa.solicitud_huella_sha256 IS DISTINCT FROM p.solicitud_huella_sha256
      OR pa.instalador_ref IS DISTINCT FROM p.actor_ref
      OR pa.fuente_ref IS DISTINCT FROM p.catalogo->>'fuente_ref'
      OR pa.aprobacion_ref IS DISTINCT FROM p.decision_ref
      OR pa.registrada_en IS DISTINCT FROM p.registrada_en
   THEN RAISE EXCEPTION 'CT218: auditoría de provisión posterior ausente o divergente' USING ERRCODE='55000'; END IF;
  ELSIF p.auditoria_ref IS NOT DISTINCT FROM 'aud_v3_'||substr(p.consumo_huella_sha256,1,32) THEN
   -- AD3 deriva esta referencia del consumo; CT135 la conserva al publicar
   -- sin crear una auditoría de provisión adicional.
   IF FOUND OR p.decision_ref IS NULL OR p.actor_ref IS NULL THEN
    RAISE EXCEPTION 'CT218: auditoría V3 de publicación divergente' USING ERRCODE='55000'; END IF;
  ELSE
   RAISE EXCEPTION 'CT218: procedencia de auditoría desconocida' USING ERRCODE='55000';
  END IF;
  SELECT * INTO o FROM vec_contratacion_temporal.catalogo_plantillas_outbox_v1
   WHERE historia_secuencia=p.secuencia;
  IF NOT FOUND OR o.recibo_ref IS DISTINCT FROM p.recibo_ref
     OR o.tipo IS DISTINCT FROM 'contratacion_temporal.plantillas_documentos.publicado'
     OR o.creada_en IS DISTINCT FROM p.registrada_en
  THEN RAISE EXCEPTION 'CT218: evento de publicación ausente o divergente' USING ERRCODE='55000'; END IF;
 END IF;
 RETURN true;
END $function$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(p_huella_sha256 text, p_version bigint, p_fuente_ref text)
 RETURNS boolean
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path = pg_catalog, pg_temp
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
BEGIN
 -- Compatibilidad del contrato CT135, cuya organización era única.
 RETURN vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2(
  'organizacion:desarrollo:dipgra',p_huella_sha256,p_version,p_fuente_ref);
END
$function$;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2(text,text,bigint,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2(text,text,bigint,text) TO vec_contratacion_temporal_ejecutor;
DO $post$
DECLARE f regprocedure; esperado text; p record; actual text;
BEGIN
 FOR f,esperado IN SELECT v.f,v.h FROM (VALUES
  ('vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(text,bigint,text)'::regprocedure,'021713a64d1c10265e402e211cd885d5d81f55dc5a21bc96d0475792e2e056bc'),
  ('vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2(text,text,bigint,text)'::regprocedure,'0ffe98d71d237a5bd056e6a7e59c69ac16adb9db80144fad1ec566ffc9b0dc82')
 ) v(f,h) LOOP
  SELECT x.* INTO STRICT p FROM pg_proc x WHERE x.oid=f;
  actual:=encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex');
  IF actual IS DISTINCT FROM esperado
  OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
  OR p.proacl IS DISTINCT FROM ARRAY[
   'vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
   'vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[]
  OR p.prosecdef IS DISTINCT FROM true OR p.provolatile IS DISTINCT FROM 's'
  OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[]
  THEN RAISE EXCEPTION 'CT226: posimagen incompatible % %',f,actual USING ERRCODE='55000'; END IF;
 END LOOP;
END $post$;
COMMIT;
