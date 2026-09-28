\set ON_ERROR_STOP on
-- SQL CT135 y AD3-99 reales con núcleo V3 sintético. No acredita el PDP.
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
DO $pruebas$
DECLARE
 org text:='organizacion:desarrollo:dipgra';
 material jsonb; material_h text; contexto_h text; capacidad bytea; decision bytea;
 respuesta jsonb;
BEGIN
 material:=jsonb_build_object('operacion','consultar','organizacion_ref',org);
 material_h:=encode(sha256(convert_to(material::text,'UTF8')),'hex');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||org||'"},"atributos":{"material_sha256":"'||material_h||'"}}','UTF8')),'hex');
 capacidad:=convert_to(jsonb_build_object(
  'audiencia_consumo','vec_contratacion_temporal.catalogo_plantillas.v1',
  'operacion','contratacion_temporal.plantillas_documentos.consultar',
  'efecto_ref','vec.contratacion_temporal.plantillas_documentos',
  'huella_efecto_sha256',contexto_h)::text,'UTF8');
 decision:=convert_to(jsonb_build_object(
  'accion','contratacion_temporal.plantillas_documentos.consultar',
  'modulo_id','contratacion_temporal',
  'tipo_recurso','catalogo_plantillas_contratacion_temporal',
  'finalidad','gestionar_catalogo_plantillas_contratacion_temporal',
  'recurso_ref','vec.contratacion_temporal.plantillas_documentos',
  'contexto_recurso_huella_sha256',contexto_h,
  'campos_permitidos',jsonb_build_array('borrador','editor_de_esta_version','publicado'),
  'obligaciones','[]'::jsonb,'principal_id','actor:sintetico')::text,'UTF8');
 respuesta:=vec_contratacion_temporal.operar_catalogo_plantillas_v1(
  material,capacidad,decision,''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea);
 IF respuesta->'publicado' IS NULL THEN
  RAISE EXCEPTION 'CT135: consulta organizativa sin publicación';
 END IF;
 BEGIN
  PERFORM vec_contratacion_temporal.operar_catalogo_plantillas_v1(
   material-'organizacion_ref',capacidad,decision,''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea);
  RAISE EXCEPTION 'CT135: organización ausente aceptada';
 EXCEPTION WHEN sqlstate '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_contratacion_temporal.operar_catalogo_plantillas_v1(
   material||'{"organizacion_ref":"organizacion:ajena"}'::jsonb,
   capacidad,decision,''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea);
  RAISE EXCEPTION 'CT135: organización ajena aceptada';
 EXCEPTION WHEN sqlstate '42501' THEN NULL; END;
 RAISE NOTICE 'CT135: organización correcta consulta; ausente y ajena rechazan 42501';
END $pruebas$;
ROLLBACK;
