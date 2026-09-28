\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
DO $test$
DECLARE m jsonb; mh text; h text; d jsonb; c jsonb; respuesta jsonb; cambiado jsonb; n int:=0;
BEGIN
 m:=jsonb_build_object('operacion','listar','organizacion_ref','organizacion:desarrollo:dipgra',
  'clase_ambito','organizacion','ambito_ref','organizacion:desarrollo:dipgra',
  'expediente_ref','expediente:ct137-sintetico','version_observada',1,
  'consulta_huella_sha256',repeat('d',64));
 mh:=encode(sha256(convert_to(m::text,'UTF8')),'hex');
 h:=encode(sha256(convert_to('{"ambitos":{"ambito_ref":"organizacion:desarrollo:dipgra","clase_ambito":"organizacion","organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"material_sha256":"'||mh||'"}}','UTF8')),'hex');
 d:=jsonb_build_object('accion','contratacion_temporal.plantillas_documentos.documental_listar',
  'modulo_id','contratacion_temporal','tipo_recurso','catalogo_plantillas_documental_ct',
  'finalidad','consultar_borradores_expediente','recurso_ref','expediente:ct137-sintetico',
  'principal_id','actor:sintetico','contexto_recurso_huella_sha256',h,
  'campos_permitidos','["catalogo","catalogo_huella_sha256","contenido_json_sha256","procedencia_ref","revision","version"]'::jsonb,
  'obligaciones','[]'::jsonb);
 c:=jsonb_build_object('audiencia_consumo','vec_contratacion_temporal.catalogo_plantillas_documental.v1',
  'operacion','contratacion_temporal.plantillas_documentos.documental_listar',
  'efecto_ref','expediente:ct137-sintetico','huella_efecto_sha256',h);
 SELECT vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_ambitos_v1(
  m,convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea) INTO respuesta;
 IF respuesta->>'procedencia_ref'<>'recibo:sintetico' OR respuesta->>'version'<>'1'
    OR respuesta->>'catalogo_huella_sha256'<>repeat('a',64) THEN
  RAISE EXCEPTION 'CT137: lectura positiva divergente'; END IF;
 FOREACH cambiado IN ARRAY ARRAY[
  m-'organizacion_ref',m-'clase_ambito',m-'ambito_ref',
  jsonb_set(m,'{organizacion_ref}','"organizacion:ajena"'),
  jsonb_set(m,'{clase_ambito}','"centro"'),
  jsonb_set(m,'{ambito_ref}','"centro:ajeno"'),
  jsonb_set(m,'{version_observada}','2')]
 LOOP
  BEGIN
   PERFORM vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_ambitos_v1(
    cambiado,convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea);
   RAISE EXCEPTION 'CT137: negativo admitido %',cambiado;
  EXCEPTION WHEN SQLSTATE '42501' THEN n:=n+1;
  END;
 END LOOP;
 IF n<>7 THEN RAISE EXCEPTION 'CT137: negativos incompletos: %',n; END IF;
 -- La capacidad ajena debe denegarse antes del consumo V3.
 c:=jsonb_set(c,'{huella_efecto_sha256}',to_jsonb(repeat('0',64)));
 BEGIN
  PERFORM vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_ambitos_v1(
   m,convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea);
  RAISE EXCEPTION 'CT137: capacidad ajena admitida';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
END $test$;
ROLLBACK;
