\set ON_ERROR_STOP on
-- CT-133: lectura documental de la última publicación del catálogo CT-131.
-- Consume una decisión V3 documental nominal antes de leer. El consumidor
-- autoriza también la consulta de detalle del expediente por su propio V3.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000133',0));
DO $pre$
DECLARE f oid;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR to_regclass('vec_contratacion_temporal.catalogo_plantillas_historia_v1') IS NULL
    OR to_regclass('vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1') IS NULL
    OR to_regclass('vec_contratacion_temporal.catalogo_plantillas_outbox_v1') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT-133: preimagen incompatible' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE')
 THEN RAISE EXCEPTION 'CT-133: AD3-96 requerido' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(
 p_material jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $lectura$
DECLARE b record; a record; p record; o record; sesion record; consumo record;
 d jsonb; c jsonb; operacion text; expediente text; version_observada bigint;
 material_h text; contexto_h text; procedencia text;
BEGIN
 SELECT rolsuper,rolbypassrls INTO sesion FROM pg_roles WHERE rolname=session_user;
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_gobernador','MEMBER')
    OR sesion.rolsuper IS DISTINCT FROM false OR sesion.rolbypassrls IS DISTINCT FROM false
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'CT-133: lectura denegada' USING ERRCODE='42501'; END IF;
 IF p_material IS NULL OR jsonb_typeof(p_material)<>'object'
    OR octet_length(p_material::text)>4096
 THEN RAISE EXCEPTION 'CT-133: material inválido' USING ERRCODE='22023'; END IF;
 operacion:=p_material->>'operacion'; expediente:=p_material->>'expediente_ref';
 IF operacion IS NULL OR operacion<>ALL(ARRAY['listar','descargar'])
    OR expediente IS NULL OR expediente !~ '^expediente:[A-Za-z0-9._:/#-]{1,149}$'
    OR jsonb_typeof(p_material->'version_observada') IS DISTINCT FROM 'number'
    OR p_material->>'consulta_huella_sha256' IS NULL
    OR p_material->>'consulta_huella_sha256' !~ '^[0-9a-f]{64}$'
    OR (operacion='listar' AND
        (p_material-ARRAY['operacion','expediente_ref','version_observada','consulta_huella_sha256'])<>'{}'::jsonb)
    OR (operacion='descargar' AND
        ((p_material-ARRAY['operacion','expediente_ref','version_observada','consulta_huella_sha256','tipo','formato'])<>'{}'::jsonb
         OR p_material->>'tipo' IS NULL OR p_material->>'tipo' !~ '^[a-z][a-z0-9._-]{1,79}$'
         OR p_material->>'formato' IS NULL OR p_material->>'formato' <>ALL(ARRAY['pdf','docx'])))
 THEN RAISE EXCEPTION 'CT-133: material documental inválido' USING ERRCODE='22023'; END IF;
 BEGIN version_observada:=(p_material->>'version_observada')::bigint;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CT-133: versión inválida' USING ERRCODE='22023'; END;
 IF version_observada<0 OR version_observada>9007199254740991
 THEN RAISE EXCEPTION 'CT-133: versión inválida' USING ERRCODE='22023'; END IF;
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CT-133: decisión inválida' USING ERRCODE='22023'; END;
 material_h:=encode(sha256(convert_to(p_material::text,'UTF8')),'hex');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{},"atributos":{"material_sha256":"'||material_h||'"}}','UTF8')),'hex');
 IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.plantillas_documentos.documental_'||operacion
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'catalogo_plantillas_documental_ct'
    OR d->>'finalidad' IS DISTINCT FROM 'consultar_borradores_expediente'
    OR d->>'recurso_ref' IS DISTINCT FROM expediente
    OR d->>'principal_id' IS NULL
    OR d->>'principal_id' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
    OR d->'campos_permitidos' IS DISTINCT FROM '["catalogo","catalogo_huella_sha256","contenido_json_sha256","procedencia_ref","revision","version"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.catalogo_plantillas_documental.v1'
    OR c->>'operacion' IS DISTINCT FROM d->>'accion'
    OR c->>'efecto_ref' IS DISTINCT FROM expediente
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
 THEN RAISE EXCEPTION 'CT-133: decisión documental divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM expediente
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_h
    OR consumo.decision_ref IS NULL OR consumo.auditoria_ref IS NULL
 THEN RAISE EXCEPTION 'CT-133: consumo documental divergente' USING ERRCODE='42501'; END IF;

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
REVOKE ALL ON FUNCTION vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_contratacion_temporal_ejecutor;
DO $acl$
DECLARE f regprocedure:='vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
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
