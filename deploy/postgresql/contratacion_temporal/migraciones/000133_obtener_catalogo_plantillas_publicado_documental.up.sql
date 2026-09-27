\set ON_ERROR_STOP on
-- CT-133: lectura documental de la última publicación del catálogo CT-131.
-- El consumidor debe haber autorizado antes la consulta del expediente por V3.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000133',0));
DO $pre$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR to_regclass('vec_contratacion_temporal.catalogo_plantillas_historia_v1') IS NULL
    OR to_regclass('vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1') IS NULL
    OR to_regclass('vec_contratacion_temporal.catalogo_plantillas_outbox_v1') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1()') IS NOT NULL
 THEN RAISE EXCEPTION 'CT-133: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1()
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $lectura$
DECLARE b record; a record; p record; o record; sesion record; procedencia text;
BEGIN
 SELECT rolsuper,rolbypassrls INTO sesion FROM pg_roles WHERE rolname=session_user;
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_gobernador','MEMBER')
    OR sesion.rolsuper IS DISTINCT FROM false OR sesion.rolbypassrls IS DISTINCT FROM false
 THEN RAISE EXCEPTION 'CT-133: lectura denegada' USING ERRCODE='42501'; END IF;

 -- La primera fila es la publicación provisionada fuera de HTTP. Su recibo y
 -- auditoría deben seguir ligados antes de exponer cualquier versión posterior.
 SELECT * INTO b FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1
  ORDER BY secuencia LIMIT 1;
 IF NOT FOUND OR b.origen<>'bootstrap' OR b.estado<>'publicado'
    OR b.catalogo->>'estado' IS DISTINCT FROM 'publicado'
    OR b.catalogo->>'version' IS DISTINCT FROM b.version::text
    OR b.catalogo->>'revision' IS DISTINCT FROM b.revision::text
    OR b.contenido_json_sha256 IS DISTINCT FROM encode(sha256(convert_to(b.catalogo::text,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'CT-133: publicación inicial ausente o divergente' USING ERRCODE='55000'; END IF;
 SELECT * INTO a FROM vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1
  WHERE historia_secuencia=b.secuencia;
 IF NOT FOUND OR a.auditoria_ref IS DISTINCT FROM b.auditoria_ref
    OR a.solicitud_huella_sha256 IS DISTINCT FROM b.consumo_huella_sha256
    OR a.instalador_ref IS DISTINCT FROM b.actor_ref
    OR a.fuente_ref IS DISTINCT FROM b.provision_fuente_ref
    OR a.aprobacion_ref IS DISTINCT FROM b.provision_aprobacion_ref
    OR a.registrada_en IS DISTINCT FROM b.registrada_en
    OR b.catalogo->>'fuente_ref' IS DISTINCT FROM a.fuente_ref
    OR b.catalogo->>'aprobacion_ref' IS DISTINCT FROM a.aprobacion_ref
 THEN RAISE EXCEPTION 'CT-133: auditoría de provisión ausente o divergente' USING ERRCODE='55000'; END IF;

 SELECT * INTO p FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1
  WHERE estado='publicado' ORDER BY secuencia DESC LIMIT 1;
 IF NOT FOUND OR p.catalogo->>'estado' IS DISTINCT FROM 'publicado'
    OR p.catalogo->>'id' IS DISTINCT FROM 'vec.contratacion_temporal.plantillas_documentos'
    OR p.catalogo->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR p.catalogo->>'version' IS DISTINCT FROM p.version::text
    OR p.catalogo->>'revision' IS DISTINCT FROM p.revision::text
    OR jsonb_typeof(p.catalogo->'entradas') IS DISTINCT FROM 'array'
    OR jsonb_array_length(p.catalogo->'entradas')>10000
    OR octet_length(p.catalogo::text)>16777216
    OR p.catalogo_huella_sha256 !~ '^[0-9a-f]{64}$'
    OR p.contenido_json_sha256 IS DISTINCT FROM encode(sha256(convert_to(p.catalogo::text,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'CT-133: publicación ausente o divergente' USING ERRCODE='55000'; END IF;
 IF p.origen='bootstrap' THEN
  IF p.secuencia<>b.secuencia THEN
   RAISE EXCEPTION 'CT-133: publicación bootstrap divergente' USING ERRCODE='55000';
  END IF;
  procedencia:=a.recibo_ref;
 ELSE
  IF p.origen<>'publicacion' OR p.recibo_ref IS NULL THEN
   RAISE EXCEPTION 'CT-133: procedencia de publicación divergente' USING ERRCODE='55000';
  END IF;
  SELECT * INTO o FROM vec_contratacion_temporal.catalogo_plantillas_outbox_v1
   WHERE historia_secuencia=p.secuencia;
  IF NOT FOUND OR o.recibo_ref IS DISTINCT FROM p.recibo_ref
     OR o.tipo IS DISTINCT FROM 'contratacion_temporal.plantillas_documentos.publicado'
  THEN RAISE EXCEPTION 'CT-133: evento de publicación ausente o divergente' USING ERRCODE='55000'; END IF;
  procedencia:=p.recibo_ref;
 END IF;
 RETURN jsonb_build_object('catalogo',p.catalogo,
  'catalogo_huella_sha256',p.catalogo_huella_sha256,
  'contenido_json_sha256',p.contenido_json_sha256,
  'version',p.version,'revision',p.revision,'procedencia_ref',procedencia);
END $lectura$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1()
 TO vec_contratacion_temporal_ejecutor;
DO $acl$
DECLARE f regprocedure:='vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1()'::regprocedure;
BEGIN
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
               WHERE p.oid=f AND (x.grantee NOT IN (p.proowner,'vec_contratacion_temporal_ejecutor'::regrole)
                                  OR x.privilege_type<>'EXECUTE'))
 THEN RAISE EXCEPTION 'CT-133: ACL incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
