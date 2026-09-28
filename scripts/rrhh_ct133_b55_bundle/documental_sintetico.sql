\set ON_ERROR_STOP on
-- Solo demuestra la fachada SQL CT137/AD3-100 con consumidor V3 sintético.
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $prueba$
DECLARE
  org text:='organizacion:desarrollo:dipgra';
  expediente text:='expediente:ensayo-ct137';
  actor text:='actor:rrhh-ensayo';
  material jsonb; capacidad bytea; decision bytea; respuesta jsonb;
  material_h text; contexto_h text; procedimiento text;
BEGIN
  material:=jsonb_build_object('operacion','descargar','organizacion_ref',org,
   'clase_ambito','organizacion','ambito_ref',org,'expediente_ref',expediente,
   'version_observada',1,'consulta_huella_sha256',repeat('a',64),
   'tipo','prueba_tipo_rrhh','formato','pdf');
  material_h:=encode(sha256(convert_to(material::text,'UTF8')),'hex');
  contexto_h:=encode(sha256(convert_to('{"ambitos":{"ambito_ref":"'||org||
   '","clase_ambito":"organizacion","organizacion_ref":"'||org||
   '"},"atributos":{"material_sha256":"'||material_h||'"}}','UTF8')),'hex');
  procedimiento:='contratacion_temporal.plantillas_documentos.documental_descargar';
  capacidad:=convert_to(jsonb_build_object('audiencia_consumo',
   'vec_contratacion_temporal.catalogo_plantillas_documental.v1','operacion',procedimiento,
   'efecto_ref',expediente,'huella_efecto_sha256',contexto_h)::text,'UTF8');
  decision:=convert_to(jsonb_build_object('accion',procedimiento,'modulo_id','contratacion_temporal',
   'tipo_recurso','catalogo_plantillas_documental_ct','finalidad','consultar_borradores_expediente',
   'recurso_ref',expediente,'principal_id',actor,'contexto_recurso_huella_sha256',contexto_h,
   'campos_permitidos',jsonb_build_array('catalogo','catalogo_huella_sha256',
    'contenido_json_sha256','procedencia_ref','revision','version'),
   'obligaciones','[]'::jsonb)::text,'UTF8');
  respuesta:=vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_ambitos_v1(
   material,capacidad,decision,''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea);
  IF NOT EXISTS (SELECT 1 FROM jsonb_array_elements(respuesta->'catalogo'->'entradas') e
                 WHERE e->>'clave'='prueba_tipo_rrhh')
     OR respuesta->>'procedencia_ref' IS NULL
     OR respuesta->>'catalogo_huella_sha256' !~ '^[a-f0-9]{64}$'
     OR respuesta->>'contenido_json_sha256' !~ '^[a-f0-9]{64}$'
  THEN RAISE EXCEPTION 'CT137: tipo nuevo, procedencia o huellas ausentes'; END IF;
  BEGIN
    PERFORM vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_ambitos_v1(
     material||'{"organizacion_ref":"organizacion:ajena"}'::jsonb,
     capacidad,decision,''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea);
    RAISE EXCEPTION 'CT137: organización ajena aceptada';
  EXCEPTION WHEN insufficient_privilege THEN NULL; END;
  BEGIN
    PERFORM vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_ambitos_v1(
     material||'{"ambito_ref":"organizacion:ajena"}'::jsonb,
     capacidad,decision,''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea);
    RAISE EXCEPTION 'CT137: ámbito ajeno aceptado';
  EXCEPTION WHEN insufficient_privilege THEN NULL; END;
  BEGIN
    PERFORM vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_ambitos_v1(
     material,capacidad,replace(convert_from(decision,'UTF8'),actor,'!')::bytea,
     ''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea);
    RAISE EXCEPTION 'CT137: actor mal formado aceptado';
  EXCEPTION WHEN insufficient_privilege THEN NULL; END;
  RAISE NOTICE 'CT137/AD3-100: tipo, tríada, procedencia y denegaciones locales correctas';
END $prueba$;
ROLLBACK;
