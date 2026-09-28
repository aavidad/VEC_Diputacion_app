\set ON_ERROR_STOP on
-- CT-135: ámbito corporativo exacto en las fachadas de plantillas ya instaladas.
-- AD3-99 debe preceder a este corte. Las filas anteriores se conservan sin
-- atribuirles retrospectivamente una organización; su replay falla cerrado.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000135',0));
DO $pre$
DECLARE f oid;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR to_regclass('vec_contratacion_temporal.catalogo_plantillas_historia_v1') IS NULL
    OR EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='vec_contratacion_temporal.catalogo_plantillas_historia_v1'::regclass
               AND attname='organizacion_ref' AND NOT attisdropped)
 THEN RAISE EXCEPTION 'CT-135: preimagen incompatible' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE')
 THEN RAISE EXCEPTION 'CT-135: AD3-99 administrativo requerido' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE')
 THEN RAISE EXCEPTION 'CT-135: AD3-99 documental requerido' USING ERRCODE='55000'; END IF;
END $pre$;
ALTER TABLE vec_contratacion_temporal.catalogo_plantillas_historia_v1
 ADD COLUMN organizacion_ref text CHECK (organizacion_ref IS NULL OR organizacion_ref='organizacion:desarrollo:dipgra');

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.operar_catalogo_plantillas_v1(
 p_material jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
 operacion text; d jsonb; c jsonb; nuevo jsonb; solicitud jsonb;
 consumo record; anterior record; previa record; publicado jsonb; borrador jsonb; provision record; provision_auditoria record;
 material_h text; contexto_h text; solicitud_h text; catalogo_h text;
 entrada jsonb; vieja_entradas jsonb; nueva_entradas jsonb; clave_entrada text;
 vieja_cantidad integer; nueva_cantidad integer; vieja_claves integer; nueva_claves integer;
 clave uuid; esperado_v bigint; esperado_r bigint; nuevo_v bigint; nuevo_r bigint;
 actor text; ahora timestamptz(6); recibo text; sec bigint; resp jsonb;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'CT-131: ejecución denegada' USING ERRCODE='42501'; END IF;
 IF p_material IS NULL OR jsonb_typeof(p_material)<>'object'
    OR octet_length(p_material::text)>17000000
 THEN RAISE EXCEPTION 'CT-131: material inválido' USING ERRCODE='22023'; END IF;
 operacion:=p_material->>'operacion';
 IF operacion IS NULL OR operacion<>ALL(ARRAY['consultar','editar','publicar'])
    OR (p_material-ARRAY['organizacion_ref','operacion','clave_idempotencia','version_esperada','revision_esperada','catalogo','catalogo_huella_sha256','catalogo_base_huella_sha256','solicitud'])<>'{}'::jsonb
 THEN RAISE EXCEPTION 'CT-131: operación inválida' USING ERRCODE='22023'; END IF;
 IF p_material->>'organizacion_ref' IS DISTINCT FROM 'organizacion:desarrollo:dipgra'
 THEN RAISE EXCEPTION 'CT-135: organización de catálogo denegada' USING ERRCODE='42501'; END IF;
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CT-131: decisión inválida' USING ERRCODE='22023'; END;
 material_h:=encode(sha256(convert_to(p_material::text,'UTF8')),'hex');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(p_material->>'organizacion_ref')||'"},"atributos":{"material_sha256":"'||material_h||'"}}','UTF8')),'hex');
 IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.plantillas_documentos.'||operacion
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'catalogo_plantillas_contratacion_temporal'
    OR d->>'finalidad' IS DISTINCT FROM 'gestionar_catalogo_plantillas_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM 'vec.contratacion_temporal.plantillas_documentos'
    OR d->'campos_permitidos' IS DISTINCT FROM (CASE WHEN operacion='consultar'
         THEN '["borrador","editor_de_esta_version","publicado"]'::jsonb ELSE '["catalogo","recibo"]'::jsonb END)
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
 THEN RAISE EXCEPTION 'CT-131: decisión divergente' USING ERRCODE='42501'; END IF;
 actor:=d->>'principal_id';
 IF actor IS NULL OR actor !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 THEN RAISE EXCEPTION 'CT-131: actor inválido' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada(
  p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE
    OR consumo.efecto_ref IS DISTINCT FROM 'vec.contratacion_temporal.plantillas_documentos'
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_h
 THEN RAISE EXCEPTION 'CT-131: consumo divergente' USING ERRCODE='42501'; END IF;
 -- Incluso la consulta queda registrada por V3, antes de leer contenido.
 IF operacion='consultar' THEN
  IF (p_material-ARRAY['operacion','organizacion_ref'])<>'{}'::jsonb
  THEN RAISE EXCEPTION 'CT-131: consulta inválida' USING ERRCODE='22023'; END IF;
  SELECT CASE WHEN h.estado='borrador' THEN h.catalogo ELSE NULL END INTO borrador
   FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 h ORDER BY h.secuencia DESC LIMIT 1;
  SELECT h.catalogo INTO publicado FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 h WHERE h.estado='publicado' ORDER BY h.secuencia DESC LIMIT 1;
  RETURN jsonb_build_object('borrador',borrador,'publicado',publicado,
   'editor_de_esta_version',CASE WHEN borrador IS NULL THEN false ELSE EXISTS (
    SELECT 1 FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 e
     WHERE e.estado='borrador' AND e.version=(borrador->>'version')::bigint
       AND e.actor_ref=actor) END);
 END IF;
 IF p_material->>'clave_idempotencia' IS NULL
    OR p_material->>'clave_idempotencia' !~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
    OR jsonb_typeof(p_material->'solicitud')<>'object'
    OR jsonb_typeof(p_material->'version_esperada')<>'number'
    OR jsonb_typeof(p_material->'revision_esperada')<>'number'
 THEN RAISE EXCEPTION 'CT-131: cambio inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  clave:=(p_material->>'clave_idempotencia')::uuid;
  esperado_v:=(p_material->>'version_esperada')::bigint;
  esperado_r:=(p_material->>'revision_esperada')::bigint;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CT-131: versión inválida' USING ERRCODE='22023'; END;
 IF esperado_v<0 OR esperado_r<0 OR esperado_v>9007199254740991 OR esperado_r>9007199254740991
 THEN RAISE EXCEPTION 'CT-131: versión inválida' USING ERRCODE='22023'; END IF;
 nuevo:=p_material->'catalogo'; solicitud:=p_material->'solicitud';
 catalogo_h:=p_material->>'catalogo_huella_sha256';
 solicitud_h:=encode(sha256(convert_to(solicitud::text,'UTF8')),'hex');
 IF solicitud->>'organizacion_ref' IS DISTINCT FROM p_material->>'organizacion_ref'
    OR solicitud->>'version_esperada' IS DISTINCT FROM esperado_v::text
    OR solicitud->>'revision_esperada' IS DISTINCT FROM esperado_r::text
 THEN RAISE EXCEPTION 'CT-131: solicitud y versión divergentes' USING ERRCODE='22023'; END IF;
 -- Serializa clave, OCC y lectura de cabeza. El advisory lock pertenece solo
 -- a este catálogo y se libera al terminar la transacción.
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:catalogo_plantillas_documentos',0));
 SELECT * INTO previa FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 h WHERE h.clave_idempotencia=clave;
 IF FOUND THEN
  IF previa.organizacion_ref IS DISTINCT FROM p_material->>'organizacion_ref'
     OR previa.solicitud_huella_sha256 IS DISTINCT FROM solicitud_h
     OR previa.actor_ref IS DISTINCT FROM actor
     OR previa.origen IS DISTINCT FROM (CASE WHEN operacion='editar' THEN 'edicion' ELSE 'publicacion' END)
  THEN RAISE EXCEPTION 'CT-131: clave reutilizada con otra solicitud' USING ERRCODE='23505'; END IF;
  RETURN jsonb_build_object('catalogo',previa.catalogo,'recibo',jsonb_build_object(
   'recibo_ref',previa.recibo_ref,'clave_idempotencia',clave,'operacion',operacion,
   'version',previa.version,'revision',previa.revision,'catalogo_huella_sha256',previa.catalogo_huella_sha256,
   'decision_ref',previa.decision_ref,'auditoria_ref',previa.auditoria_ref,
   'consumo_huella_sha256',previa.consumo_huella_sha256,'registrado_en',previa.registrada_en,
   'estado_replay','replay'),'replay',true);
 END IF;
 IF jsonb_typeof(nuevo)<>'object' OR catalogo_h IS NULL
    OR catalogo_h !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'CT-131: catálogo de cambio requerido' USING ERRCODE='22023'; END IF;
 IF nuevo->>'id' IS DISTINCT FROM 'vec.contratacion_temporal.plantillas_documentos'
    OR nuevo->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR jsonb_typeof(nuevo->'entradas') IS DISTINCT FROM 'array'
    OR jsonb_array_length(nuevo->'entradas')>10000
    OR (operacion='editar' AND nuevo->>'estado' IS DISTINCT FROM 'borrador')
    OR (operacion='publicar' AND nuevo->>'estado' IS DISTINCT FROM 'publicado')
    OR octet_length(nuevo::text)>16777216
 THEN RAISE EXCEPTION 'CT-131: catálogo inválido' USING ERRCODE='22023'; END IF;
 SELECT * INTO anterior FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 ORDER BY secuencia DESC LIMIT 1;
 IF NOT FOUND THEN RAISE EXCEPTION 'CT-131: catálogo bootstrap no provisionado' USING ERRCODE='55000'; END IF;
 SELECT * INTO provision FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 ORDER BY secuencia LIMIT 1;
 IF provision.origen<>'bootstrap' OR provision.estado<>'publicado'
    OR provision.contenido_json_sha256<>encode(sha256(convert_to(provision.catalogo::text,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'CT-131: control bootstrap incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO provision_auditoria FROM vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1
  WHERE historia_secuencia=provision.secuencia;
 IF NOT FOUND OR provision_auditoria.auditoria_ref<>provision.auditoria_ref
    OR provision_auditoria.solicitud_huella_sha256<>provision.consumo_huella_sha256
    OR provision_auditoria.fuente_ref<>provision.provision_fuente_ref
    OR provision_auditoria.aprobacion_ref<>provision.provision_aprobacion_ref
    OR provision_auditoria.instalador_ref<>provision.actor_ref
    OR provision_auditoria.registrada_en<>provision.registrada_en
 THEN RAISE EXCEPTION 'CT-131: auditoría bootstrap incompatible' USING ERRCODE='55000'; END IF;
 IF anterior.origen='bootstrap' AND (
    operacion<>'editar' OR anterior.version<>esperado_v OR esperado_r<>0
    OR p_material->>'catalogo_base_huella_sha256' IS DISTINCT FROM anterior.catalogo_huella_sha256)
 THEN RAISE EXCEPTION 'CT-131: huella o versión bootstrap divergente' USING ERRCODE='40001'; END IF;
 IF anterior.origen<>'bootstrap' AND p_material ? 'catalogo_base_huella_sha256'
 THEN RAISE EXCEPTION 'CT-131: huella bootstrap tardía' USING ERRCODE='40001'; END IF;
 IF operacion='editar' THEN
  IF anterior.secuencia IS NULL THEN
   RAISE EXCEPTION 'CT-131: catálogo bootstrap requerido' USING ERRCODE='55000';
  ELSIF anterior.estado='publicado' THEN
   IF esperado_v<>anterior.version OR esperado_r<>0 THEN RAISE EXCEPTION 'CT-131: conflicto de versión' USING ERRCODE='40001'; END IF;
   nuevo_v:=anterior.version+1; nuevo_r:=1;
  ELSE
   IF esperado_v<>anterior.version OR esperado_r<>anterior.revision THEN RAISE EXCEPTION 'CT-131: conflicto de revisión' USING ERRCODE='40001'; END IF;
   nuevo_v:=anterior.version; nuevo_r:=anterior.revision+1;
  END IF;
  IF nuevo->>'creado_por' IS DISTINCT FROM (CASE WHEN nuevo_r=1 THEN actor ELSE anterior.catalogo->>'creado_por' END)
     OR nuevo->>'creado_en' IS NULL
     OR (nuevo_r>1 AND nuevo->>'ultima_modificacion_por' IS DISTINCT FROM actor)
     OR (nuevo_r>1 AND nuevo->>'version_anterior_ref' IS DISTINCT FROM anterior.catalogo->>'version_anterior_ref')
     OR (nuevo_r>1 AND nuevo->>'creado_en' IS DISTINCT FROM anterior.catalogo->>'creado_en')
     OR (nuevo_r>1 AND nuevo->>'motivo_creacion' IS DISTINCT FROM anterior.catalogo->>'motivo_creacion')
  THEN RAISE EXCEPTION 'CT-131: autoría o filiación divergente' USING ERRCODE='42501'; END IF;
  entrada:=solicitud->'entrada'; clave_entrada:=entrada->>'clave';
  IF jsonb_typeof(entrada) IS DISTINCT FROM 'object' OR clave_entrada IS NULL OR clave_entrada=''
     OR nuevo->>'fuente_ref' IS DISTINCT FROM solicitud->>'fuente_ref'
     OR nuevo->>'nombre' IS DISTINCT FROM anterior.catalogo->>'nombre'
     OR nuevo->>'descripcion' IS DISTINCT FROM anterior.catalogo->>'descripcion'
     OR (nuevo_r=1 AND nuevo->>'motivo_creacion' IS DISTINCT FROM solicitud->>'motivo')
     OR (nuevo_r>1 AND nuevo->>'motivo_modificacion' IS DISTINCT FROM solicitud->>'motivo')
  THEN RAISE EXCEPTION 'CT-131: edición divergente de solicitud' USING ERRCODE='42501'; END IF;
  SELECT coalesce(jsonb_object_agg(e->>'clave',e),'{}'::jsonb),count(*)::integer,count(DISTINCT e->>'clave')::integer
   INTO vieja_entradas,vieja_cantidad,vieja_claves
   FROM jsonb_array_elements(anterior.catalogo->'entradas') e;
  SELECT coalesce(jsonb_object_agg(e->>'clave',e),'{}'::jsonb),count(*)::integer,count(DISTINCT e->>'clave')::integer
   INTO nueva_entradas,nueva_cantidad,nueva_claves FROM jsonb_array_elements(nuevo->'entradas') e;
  IF vieja_cantidad<>vieja_claves OR nueva_cantidad<>nueva_claves
     OR nueva_entradas IS DISTINCT FROM (vieja_entradas||jsonb_build_object(clave_entrada,entrada))
  THEN RAISE EXCEPTION 'CT-131: entradas distintas de la edición autorizada' USING ERRCODE='42501'; END IF;
 ELSE
  IF anterior.secuencia IS NULL OR anterior.estado<>'borrador'
     OR esperado_v<>anterior.version OR esperado_r<>anterior.revision
  THEN RAISE EXCEPTION 'CT-131: borrador no vigente' USING ERRCODE='40001'; END IF;
  nuevo_v:=anterior.version; nuevo_r:=anterior.revision;
  IF nuevo->>'publicado_por' IS DISTINCT FROM actor
     OR actor=anterior.catalogo->>'creado_por'
     OR actor=anterior.catalogo->>'ultima_modificacion_por'
     OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 e
                 WHERE e.estado='borrador' AND e.version=anterior.version AND e.actor_ref=actor)
     OR nuevo->>'aprobacion_ref' IS NULL OR nuevo->>'aprobacion_ref'=''
     OR nuevo->>'motivo_publicacion' IS NULL OR nuevo->>'motivo_publicacion'=''
     OR nuevo->>'publicado_en' IS NULL OR nuevo->>'publicado_en'=''
     OR nuevo->>'aprobacion_ref' IS DISTINCT FROM solicitud->>'aprobacion_ref'
     OR nuevo->>'motivo_publicacion' IS DISTINCT FROM solicitud->>'motivo'
     OR (nuevo-ARRAY['estado','publicado_por','publicado_en','aprobacion_ref','motivo_publicacion'])
        IS DISTINCT FROM (anterior.catalogo-ARRAY['estado','publicado_por','publicado_en','aprobacion_ref','motivo_publicacion'])
  THEN RAISE EXCEPTION 'CT-131: separación de funciones o contenido divergente' USING ERRCODE='42501'; END IF;
 END IF;
 IF nuevo->>'version' IS DISTINCT FROM nuevo_v::text OR nuevo->>'revision' IS DISTINCT FROM nuevo_r::text
    OR (nuevo_v>1 AND nuevo->>'version_anterior_ref' IS DISTINCT FROM 'vec.contratacion_temporal.plantillas_documentos:'||(nuevo_v-1)::text)
 THEN RAISE EXCEPTION 'CT-131: versión del catálogo divergente' USING ERRCODE='40001'; END IF;
 ahora:=date_trunc('microseconds',clock_timestamp()); recibo:='recibo:'||gen_random_uuid()::text;
 INSERT INTO vec_contratacion_temporal.catalogo_plantillas_historia_v1(
  version,revision,estado,origen,catalogo,catalogo_huella_sha256,contenido_json_sha256,
  clave_idempotencia,solicitud_huella_sha256,recibo_ref,actor_ref,decision_ref,
  consumo_huella_sha256,auditoria_ref,registrada_en,organizacion_ref)
 VALUES(nuevo_v,nuevo_r,CASE WHEN operacion='editar' THEN 'borrador' ELSE 'publicado' END,
  CASE WHEN operacion='editar' THEN 'edicion' ELSE 'publicacion' END,
  nuevo,catalogo_h,encode(sha256(convert_to(nuevo::text,'UTF8')),'hex'),clave,solicitud_h,recibo,
  actor,consumo.decision_ref,consumo.consumo_huella_sha256,consumo.auditoria_ref,ahora,p_material->>'organizacion_ref')
 RETURNING secuencia INTO sec;
 INSERT INTO vec_contratacion_temporal.catalogo_plantillas_outbox_v1(evento_ref,historia_secuencia,recibo_ref,tipo,estado,creada_en)
 VALUES('evento:'||gen_random_uuid()::text,sec,recibo,
  CASE WHEN operacion='editar' THEN 'contratacion_temporal.plantillas_documentos.borrador_actualizado'
       ELSE 'contratacion_temporal.plantillas_documentos.publicado' END,'pendiente',ahora);
 resp:=jsonb_build_object('catalogo',nuevo,'recibo',jsonb_build_object(
  'recibo_ref',recibo,'clave_idempotencia',clave,'operacion',operacion,'version',nuevo_v,'revision',nuevo_r,
  'catalogo_huella_sha256',catalogo_h,'decision_ref',consumo.decision_ref,
  'auditoria_ref',consumo.auditoria_ref,'consumo_huella_sha256',consumo.consumo_huella_sha256,
  'registrado_en',ahora,'estado_replay','registrado'),'replay',false);
 RETURN resp;
END $funcion$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(
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
        (p_material-ARRAY['operacion','organizacion_ref','expediente_ref','version_observada','consulta_huella_sha256'])<>'{}'::jsonb)
    OR (operacion='descargar' AND
        ((p_material-ARRAY['operacion','organizacion_ref','expediente_ref','version_observada','consulta_huella_sha256','tipo','formato'])<>'{}'::jsonb
         OR p_material->>'tipo' IS NULL OR p_material->>'tipo' !~ '^[a-z][a-z0-9._-]{1,79}$'
         OR p_material->>'formato' IS NULL OR p_material->>'formato' <>ALL(ARRAY['pdf','docx'])))
 THEN RAISE EXCEPTION 'CT-133: material documental inválido' USING ERRCODE='22023'; END IF;
 IF p_material->>'organizacion_ref' IS DISTINCT FROM 'organizacion:desarrollo:dipgra'
 THEN RAISE EXCEPTION 'CT-135: organización documental denegada' USING ERRCODE='42501'; END IF;
 BEGIN version_observada:=(p_material->>'version_observada')::bigint;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CT-133: versión inválida' USING ERRCODE='22023'; END;
 IF version_observada<1 OR version_observada>9007199254740991
 THEN RAISE EXCEPTION 'CT-133: versión inválida' USING ERRCODE='22023'; END IF;
 IF NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral e
  WHERE e.expediente_ref=expediente AND e.version=version_observada
    AND e.agregado_json->>'organizacion_ref'=p_material->>'organizacion_ref')
 THEN RAISE EXCEPTION 'CT-135: expediente y organización divergentes' USING ERRCODE='42501'; END IF;
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CT-133: decisión inválida' USING ERRCODE='22023'; END;
 material_h:=encode(sha256(convert_to(p_material::text,'UTF8')),'hex');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(p_material->>'organizacion_ref')||'"},"atributos":{"material_sha256":"'||material_h||'"}}','UTF8')),'hex');
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
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_org_v3_atestada(
  p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
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

DO $acl$
DECLARE f regprocedure; n text;
BEGIN
 FOREACH n IN ARRAY ARRAY[
 'vec_contratacion_temporal.operar_catalogo_plantillas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
 'vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
  f:=n::regprocedure;
  IF (SELECT proowner FROM pg_proc WHERE oid=f)<>'vec_contratacion_temporal_propietario'::regrole
     OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
     OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
     OR EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
                WHERE p.oid=f AND (x.grantee NOT IN (p.proowner,'vec_contratacion_temporal_ejecutor'::regrole)
                                   OR x.privilege_type<>'EXECUTE'))
  THEN RAISE EXCEPTION 'CT-135: ACL incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;
COMMIT;
