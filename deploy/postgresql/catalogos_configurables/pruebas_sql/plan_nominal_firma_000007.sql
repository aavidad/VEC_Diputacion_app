\set ON_ERROR_STOP on
-- Prueba estructural CC7. Se ejecuta después de L, AD177 y CC7 sobre clon.
-- No inserta decisiones, consumos ni concesiones de prueba.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='15s';
DO $prueba$
DECLARE t text; f regprocedure; p record; permiso record;
 propietario oid:='vec_catalogos_configurables_propietario'::regrole;
 ad oid:='vec_autorizacion_atestada_v3_propietario'::regrole;
 ct oid:='vec_contratacion_temporal_propietario'::regrole;
BEGIN
 FOREACH t IN ARRAY ARRAY['plan_firma_control','plan_firma_historia','plan_firma_publicacion',
  'plan_firma_efecto','plan_firma_outbox'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class c
    WHERE c.oid=pg_catalog.to_regclass('vec_catalogos_configurables.'||t)
      AND c.relowner=propietario AND c.relrowsecurity AND c.relforcerowsecurity) THEN
   RAISE EXCEPTION 'CC7: tabla sin propietario o RLS: %',t USING ERRCODE='55000'; END IF;
  IF EXISTS(SELECT 1 FROM pg_catalog.pg_class c
    CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(c.relacl,
      pg_catalog.acldefault('r',c.relowner))) x
    WHERE c.oid=pg_catalog.to_regclass('vec_catalogos_configurables.'||t)
      AND x.grantee<>propietario) THEN
   RAISE EXCEPTION 'CC7: tabla concedida fuera de propietario: %',t USING ERRCODE='55000'; END IF;
  IF EXISTS(SELECT 1 FROM pg_catalog.pg_type y
    CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(y.typacl,
      pg_catalog.acldefault('T',y.typowner))) x
    WHERE y.typrelid=pg_catalog.to_regclass('vec_catalogos_configurables.'||t)
      AND x.grantee<>propietario) THEN
   RAISE EXCEPTION 'CC7: tipo de fila concedido fuera de propietario: %',t USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH f IN ARRAY ARRAY[
  'vec_catalogos_configurables.json_plan_sin_duplicados_v1(json,integer)'::regprocedure,
  'vec_catalogos_configurables.validar_plan_nominal_firma_v1(bytea,text)'::regprocedure,
  'vec_catalogos_configurables.leer_plan_nominal_firma_v1(text,bigint,text,text,jsonb)'::regprocedure,
  'vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(bytea,jsonb)'::regprocedure] LOOP
  SELECT x.proowner,x.prosecdef,x.provolatile,x.proconfig INTO STRICT p
   FROM pg_catalog.pg_proc x WHERE x.oid=f;
  IF p.proowner<>propietario OR NOT p.prosecdef OR
     (f IN ('vec_catalogos_configurables.json_plan_sin_duplicados_v1(json,integer)'::regprocedure,
       'vec_catalogos_configurables.validar_plan_nominal_firma_v1(bytea,text)'::regprocedure) AND p.provolatile<>'i') OR
     (f NOT IN ('vec_catalogos_configurables.json_plan_sin_duplicados_v1(json,integer)'::regprocedure,
       'vec_catalogos_configurables.validar_plan_nominal_firma_v1(bytea,text)'::regprocedure) AND p.provolatile<>'v')
     OR NOT (p.proconfig @> ARRAY['search_path=pg_catalog','row_security=on']) THEN
   RAISE EXCEPTION 'CC7: función sin frontera fija: %',f USING ERRCODE='55000'; END IF;
  IF f='vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(bytea,jsonb)'::regprocedure
     AND EXISTS(SELECT 1 FROM pg_catalog.pg_proc x WHERE x.oid=f AND
       (x.proargnames[8] IS DISTINCT FROM 'actor_ref' OR
        x.proargnames[9] IS DISTINCT FROM 'confirmado_en' OR
        x.proargnames[10] IS DISTINCT FROM 'auditoria_ref' OR
        x.proargnames[11] IS DISTINCT FROM 'outbox_recibo_ref' OR
        x.proallargtypes[8] IS DISTINCT FROM 'text'::regtype OR
        x.proallargtypes[9] IS DISTINCT FROM 'timestamptz'::regtype OR
        x.proallargtypes[10] IS DISTINCT FROM 'text'::regtype OR
        x.proallargtypes[11] IS DISTINCT FROM 'text'::regtype)) THEN
   RAISE EXCEPTION 'CC7: retorno original de gobierno incompatible' USING ERRCODE='55000'; END IF;
  FOR permiso IN SELECT a.grantee FROM pg_catalog.pg_proc pp
   CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(pp.proacl,
     pg_catalog.acldefault('f',pp.proowner))) a
   WHERE pp.oid=f LOOP
   IF permiso.grantee=0 OR
      (f IN ('vec_catalogos_configurables.json_plan_sin_duplicados_v1(json,integer)'::regprocedure,
        'vec_catalogos_configurables.validar_plan_nominal_firma_v1(bytea,text)'::regprocedure) AND permiso.grantee<>propietario) OR
      (f='vec_catalogos_configurables.leer_plan_nominal_firma_v1(text,bigint,text,text,jsonb)'::regprocedure AND permiso.grantee NOT IN (propietario,ct)) OR
      (f='vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(bytea,jsonb)'::regprocedure AND permiso.grantee NOT IN (propietario,ad)) THEN
    RAISE EXCEPTION 'CC7: ACL de función demasiado amplia: %',f USING ERRCODE='55000'; END IF;
  END LOOP;
 END LOOP;
 IF NOT pg_catalog.has_function_privilege(ct,
   'vec_catalogos_configurables.leer_plan_nominal_firma_v1(text,bigint,text,text,jsonb)','EXECUTE')
    OR pg_catalog.has_function_privilege(ct,
   'vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(bytea,jsonb)','EXECUTE')
    OR NOT pg_catalog.has_function_privilege(ad,
   'vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(bytea,jsonb)','EXECUTE') THEN
  RAISE EXCEPTION 'CC7: ejecutores técnicos incompatibles' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_trigger g
    WHERE g.tgrelid='vec_catalogos_configurables.plan_firma_publicacion'::regclass
      AND g.tgname='plan_firma_publicacion_inmutable' AND g.tgenabled IN ('O','A')) THEN
  RAISE EXCEPTION 'CC7: publicación mutable' USING ERRCODE='55000'; END IF;
END $prueba$;
-- Vectores de bytes exactos generados con CatalogoConfigurable.ClonarCanonico
-- y json.Marshal de Go; las copias legibles están en testdata/.
DO $vector$
DECLARE borrador bytea:=pg_catalog.decode('eyJpZCI6ImN0LnBsYW4uZmlybWEuc2ludGV0aWNvIiwidmVyc2lvbiI6MSwicmV2aXNpb24iOjEsIm1vZHVsb19pZCI6ImNvbnRyYXRhY2lvbl90ZW1wb3JhbCIsIm5vbWJyZSI6IlBsYW4gZGUgZmlybWEgc2ludMOpdGljbyIsImZ1ZW50ZV9yZWYiOiJmdWVudGU6cnJoaDpzaW50ZXRpY2EiLCJtb3Rpdm9fY3JlYWNpb24iOiJFamVyY2ljaW8gc2ludMOpdGljbyIsImVudHJhZGFzIjpbeyJjbGF2ZSI6InBhc29fMSIsImV0aXF1ZXRhIjoiUGFzbyAxIiwib3JkZW4iOjEsInZpZ2VudGVfZGVzZGUiOiIyMDI2LTAxLTAxVDAwOjAwOjAwWiIsInZpZ2VudGVfaGFzdGEiOiIwMDAxLTAxLTAxVDAwOjAwOjAwWiIsImF0cmlidXRvcyI6eyJhY2Npb25fY29tcGV0ZW5jaWFsIjoiY29udHJhdGFjaW9uX3RlbXBvcmFsLmRvY3VtZW50by5maXJtYV92ZWMucmVnaXN0cmFyIiwiY2FyZ29fcmVmIjoiY2FyZ286ZGlyZWNjaW9uIiwiY2lyY3VpdG9fcmVmIjoiY2F0YWxvZ286Y2lyY3VpdG86c2ludGV0aWNvIiwiY2lyY3VpdG9fc2hhMjU2IjoiYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYSIsImNpcmN1aXRvX3ZlcnNpb24iOiIxIiwiZG9jdW1lbnRvIjoiaW5mb3JtZV9kZWZpbml0aXZvIiwiZXNxdWVtYSI6ImN0LnBsYW4tY29tcGV0ZW5jaWEtZmlybWEudjIiLCJlc3F1ZW1hX2NvbnRleHRvIjoidmVjLmNvbnRleHRvLmZpcm1hLmN0LnYxIiwiZmluYWxpZGFkIjoiZ2VzdGlvbmFyX2NvbnRyYXRhY2lvbl90ZW1wb3JhbCIsIm1hcGVvX2Z1ZW50ZV9yZWYiOiJmdWVudGU6cGxhbjpjdCIsIm1hcGVvX3ZlcnNpb24iOiIxIiwib3JnYW5pemFjaW9uX3JlZiI6Im9yZ2FuaXphY2lvbjpjZW50cmFsIiwicGFzb19vcmRlbiI6IjEiLCJwYXNvX3JlZiI6InBhc286ZGlyZWNjaW9uIiwicGVyZmlsX2VzcGVyYWRvX3JlZiI6InBlcmZpbDpmaXJtYTpkaXJlY2Npb24iLCJyb2xfaWQiOiJjdF9kaXJlY2Npb25fcnJoaCIsInRpcG9fcmVjdXJzbyI6ImRvY3VtZW50b19jb250cmF0YWNpb25fdGVtcG9yYWwiLCJ1bmlkYWRfcmVmIjoidW5pZGFkOnJyaGgifX1dLCJlc3RhZG8iOiJib3JyYWRvciIsImNyZWFkb19wb3IiOiJhY3RvcjpjcmVhZG9yOjAwMSIsImNyZWFkb19lbiI6IjIwMjYtMTAtMDNUMTA6MDA6MDBaIiwidWx0aW1hX21vZGlmaWNhY2lvbl9lbiI6IjAwMDEtMDEtMDFUMDA6MDA6MDBaIiwicHVibGljYWRvX2VuIjoiMDAwMS0wMS0wMVQwMDowMDowMFoiLCJyZXRpcmFkb19lbiI6IjAwMDEtMDEtMDFUMDA6MDA6MDBaIn0=','base64');
 publicado bytea:=pg_catalog.decode('eyJpZCI6ImN0LnBsYW4uZmlybWEuc2ludGV0aWNvIiwidmVyc2lvbiI6MSwicmV2aXNpb24iOjEsIm1vZHVsb19pZCI6ImNvbnRyYXRhY2lvbl90ZW1wb3JhbCIsIm5vbWJyZSI6IlBsYW4gZGUgZmlybWEgc2ludMOpdGljbyIsImZ1ZW50ZV9yZWYiOiJmdWVudGU6cnJoaDpzaW50ZXRpY2EiLCJtb3Rpdm9fY3JlYWNpb24iOiJFamVyY2ljaW8gc2ludMOpdGljbyIsImVudHJhZGFzIjpbeyJjbGF2ZSI6InBhc29fMSIsImV0aXF1ZXRhIjoiUGFzbyAxIiwib3JkZW4iOjEsInZpZ2VudGVfZGVzZGUiOiIyMDI2LTAxLTAxVDAwOjAwOjAwWiIsInZpZ2VudGVfaGFzdGEiOiIwMDAxLTAxLTAxVDAwOjAwOjAwWiIsImF0cmlidXRvcyI6eyJhY2Npb25fY29tcGV0ZW5jaWFsIjoiY29udHJhdGFjaW9uX3RlbXBvcmFsLmRvY3VtZW50by5maXJtYV92ZWMucmVnaXN0cmFyIiwiY2FyZ29fcmVmIjoiY2FyZ286ZGlyZWNjaW9uIiwiY2lyY3VpdG9fcmVmIjoiY2F0YWxvZ286Y2lyY3VpdG86c2ludGV0aWNvIiwiY2lyY3VpdG9fc2hhMjU2IjoiYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYSIsImNpcmN1aXRvX3ZlcnNpb24iOiIxIiwiZG9jdW1lbnRvIjoiaW5mb3JtZV9kZWZpbml0aXZvIiwiZXNxdWVtYSI6ImN0LnBsYW4tY29tcGV0ZW5jaWEtZmlybWEudjIiLCJlc3F1ZW1hX2NvbnRleHRvIjoidmVjLmNvbnRleHRvLmZpcm1hLmN0LnYxIiwiZmluYWxpZGFkIjoiZ2VzdGlvbmFyX2NvbnRyYXRhY2lvbl90ZW1wb3JhbCIsIm1hcGVvX2Z1ZW50ZV9yZWYiOiJmdWVudGU6cGxhbjpjdCIsIm1hcGVvX3ZlcnNpb24iOiIxIiwib3JnYW5pemFjaW9uX3JlZiI6Im9yZ2FuaXphY2lvbjpjZW50cmFsIiwicGFzb19vcmRlbiI6IjEiLCJwYXNvX3JlZiI6InBhc286ZGlyZWNjaW9uIiwicGVyZmlsX2VzcGVyYWRvX3JlZiI6InBlcmZpbDpmaXJtYTpkaXJlY2Npb24iLCJyb2xfaWQiOiJjdF9kaXJlY2Npb25fcnJoaCIsInRpcG9fcmVjdXJzbyI6ImRvY3VtZW50b19jb250cmF0YWNpb25fdGVtcG9yYWwiLCJ1bmlkYWRfcmVmIjoidW5pZGFkOnJyaGgifX1dLCJlc3RhZG8iOiJwdWJsaWNhZG8iLCJjcmVhZG9fcG9yIjoiYWN0b3I6Y3JlYWRvcjowMDEiLCJjcmVhZG9fZW4iOiIyMDI2LTEwLTAzVDEwOjAwOjAwWiIsInVsdGltYV9tb2RpZmljYWNpb25fZW4iOiIwMDAxLTAxLTAxVDAwOjAwOjAwWiIsInB1YmxpY2Fkb19wb3IiOiJhY3RvcjpwdWJsaWNhZG9yOjAwMSIsInB1YmxpY2Fkb19lbiI6IjIwMjYtMTAtMDNUMTE6MDA6MDBaIiwiYXByb2JhY2lvbl9yZWYiOiJhcHJvYmFjaW9uOnNpbnRldGljYTowMDEiLCJtb3Rpdm9fcHVibGljYWNpb24iOiJSZXZpc2nDs24gc2ludMOpdGljYSIsInJldGlyYWRvX2VuIjoiMDAwMS0wMS0wMVQwMDowMDowMFoifQ==','base64');
 retirado bytea:=pg_catalog.decode('eyJpZCI6ImN0LnBsYW4uZmlybWEuc2ludGV0aWNvIiwidmVyc2lvbiI6MSwicmV2aXNpb24iOjEsIm1vZHVsb19pZCI6ImNvbnRyYXRhY2lvbl90ZW1wb3JhbCIsIm5vbWJyZSI6IlBsYW4gZGUgZmlybWEgc2ludMOpdGljbyIsImZ1ZW50ZV9yZWYiOiJmdWVudGU6cnJoaDpzaW50ZXRpY2EiLCJtb3Rpdm9fY3JlYWNpb24iOiJFamVyY2ljaW8gc2ludMOpdGljbyIsImVudHJhZGFzIjpbeyJjbGF2ZSI6InBhc29fMSIsImV0aXF1ZXRhIjoiUGFzbyAxIiwib3JkZW4iOjEsInZpZ2VudGVfZGVzZGUiOiIyMDI2LTAxLTAxVDAwOjAwOjAwWiIsInZpZ2VudGVfaGFzdGEiOiIwMDAxLTAxLTAxVDAwOjAwOjAwWiIsImF0cmlidXRvcyI6eyJhY2Npb25fY29tcGV0ZW5jaWFsIjoiY29udHJhdGFjaW9uX3RlbXBvcmFsLmRvY3VtZW50by5maXJtYV92ZWMucmVnaXN0cmFyIiwiY2FyZ29fcmVmIjoiY2FyZ286ZGlyZWNjaW9uIiwiY2lyY3VpdG9fcmVmIjoiY2F0YWxvZ286Y2lyY3VpdG86c2ludGV0aWNvIiwiY2lyY3VpdG9fc2hhMjU2IjoiYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYSIsImNpcmN1aXRvX3ZlcnNpb24iOiIxIiwiZG9jdW1lbnRvIjoiaW5mb3JtZV9kZWZpbml0aXZvIiwiZXNxdWVtYSI6ImN0LnBsYW4tY29tcGV0ZW5jaWEtZmlybWEudjIiLCJlc3F1ZW1hX2NvbnRleHRvIjoidmVjLmNvbnRleHRvLmZpcm1hLmN0LnYxIiwiZmluYWxpZGFkIjoiZ2VzdGlvbmFyX2NvbnRyYXRhY2lvbl90ZW1wb3JhbCIsIm1hcGVvX2Z1ZW50ZV9yZWYiOiJmdWVudGU6cGxhbjpjdCIsIm1hcGVvX3ZlcnNpb24iOiIxIiwib3JnYW5pemFjaW9uX3JlZiI6Im9yZ2FuaXphY2lvbjpjZW50cmFsIiwicGFzb19vcmRlbiI6IjEiLCJwYXNvX3JlZiI6InBhc286ZGlyZWNjaW9uIiwicGVyZmlsX2VzcGVyYWRvX3JlZiI6InBlcmZpbDpmaXJtYTpkaXJlY2Npb24iLCJyb2xfaWQiOiJjdF9kaXJlY2Npb25fcnJoaCIsInRpcG9fcmVjdXJzbyI6ImRvY3VtZW50b19jb250cmF0YWNpb25fdGVtcG9yYWwiLCJ1bmlkYWRfcmVmIjoidW5pZGFkOnJyaGgifX1dLCJlc3RhZG8iOiJyZXRpcmFkbyIsImNyZWFkb19wb3IiOiJhY3RvcjpjcmVhZG9yOjAwMSIsImNyZWFkb19lbiI6IjIwMjYtMTAtMDNUMTA6MDA6MDBaIiwidWx0aW1hX21vZGlmaWNhY2lvbl9lbiI6IjAwMDEtMDEtMDFUMDA6MDA6MDBaIiwicHVibGljYWRvX3BvciI6ImFjdG9yOnB1YmxpY2Fkb3I6MDAxIiwicHVibGljYWRvX2VuIjoiMjAyNi0xMC0wM1QxMTowMDowMFoiLCJhcHJvYmFjaW9uX3JlZiI6ImFwcm9iYWNpb246c2ludGV0aWNhOjAwMSIsIm1vdGl2b19wdWJsaWNhY2lvbiI6IlJldmlzacOzbiBzaW50w6l0aWNhIiwicmV0aXJhZG9fcG9yIjoiYWN0b3I6cmV0aXJhZG9yOjAwMSIsInJldGlyYWRvX2VuIjoiMjAyNi0xMC0wM1QxMjowMDowMFoiLCJyZXRpcmFkYV9hcHJvYmFjaW9uX3JlZiI6ImFwcm9iYWNpb246c2ludGV0aWNhOnJldGlybzowMDEiLCJtb3Rpdm9fcmV0aXJhZGEiOiJSZXRpcmFkYSBzaW50w6l0aWNhIn0=','base64');
 c jsonb; borrador_c jsonb; publicado_c jsonb; retirado_c jsonb; rechazo boolean:=false; alterado bytea; mala text;
BEGIN
 c:=vec_catalogos_configurables.validar_plan_nominal_firma_v1(borrador,'ce029352e249d4260bcc2b5717de0fd9aebbff3f1595af5d7170930805f78e04');
 IF c->>'estado' IS DISTINCT FROM 'borrador' OR c->>'publicado_en' IS DISTINCT FROM '0001-01-01T00:00:00Z'
    OR c->'entradas'->0->>'vigente_hasta' IS DISTINCT FROM '0001-01-01T00:00:00Z' THEN
  RAISE EXCEPTION 'CC7: vector Go de borrador incompatible' USING ERRCODE='55000'; END IF;
 borrador_c:=c;
 c:=vec_catalogos_configurables.validar_plan_nominal_firma_v1(publicado,'2a88331f403f3de34386a5b2e4930e22f7533282ae1f6e9f9faa85b3ac9f9dd6');
 IF c->>'estado' IS DISTINCT FROM 'publicado' OR c->>'retirado_en' IS DISTINCT FROM '0001-01-01T00:00:00Z'
    OR c->'entradas'->0->>'vigente_hasta' IS DISTINCT FROM '0001-01-01T00:00:00Z' THEN
  RAISE EXCEPTION 'CC7: vector Go publicado incompatible' USING ERRCODE='55000'; END IF;
 publicado_c:=c;
 c:=vec_catalogos_configurables.validar_plan_nominal_firma_v1(retirado,'817a7bb30120412671e8a1208ace781a1548f9c591a15a2c21dcf4f4254bc457');
 retirado_c:=c;
 IF retirado_c->>'estado' IS DISTINCT FROM 'retirado'
    OR publicado_c - 'estado' - 'publicado_por' - 'publicado_en' - 'aprobacion_ref' - 'motivo_publicacion'
       IS DISTINCT FROM borrador_c - 'estado' - 'publicado_en'
    OR retirado_c - 'estado' - 'retirado_por' - 'retirado_en' - 'retirada_aprobacion_ref' - 'motivo_retirada'
       IS DISTINCT FROM publicado_c - 'estado' - 'retirado_en' THEN
  RAISE EXCEPTION 'CC7: transición de vectores Go incompatible' USING ERRCODE='55000'; END IF;
 IF vec_catalogos_configurables.json_plan_sin_duplicados_v1('{"x":1,"x":2}'::json,0) THEN
  RAISE EXCEPTION 'CC7: JSON duplicado admitido' USING ERRCODE='55000'; END IF;
 alterado:=pg_catalog.convert_to(pg_catalog.replace(pg_catalog.convert_from(publicado,'UTF8'),
  '"paso_orden":"1"','"paso_orden":1'),'UTF8');
 BEGIN
  PERFORM vec_catalogos_configurables.validar_plan_nominal_firma_v1(alterado,
   pg_catalog.encode(pg_catalog.sha256(alterado),'hex'));
 EXCEPTION WHEN SQLSTATE '22023' THEN rechazo:=true; END;
 IF NOT rechazo THEN
  RAISE EXCEPTION 'CC7: atributo numérico en mapa string aceptado' USING ERRCODE='55000'; END IF;
 FOREACH mala IN ARRAY ARRAY['a','b','ct:plan'] LOOP
  alterado:=pg_catalog.convert_to(pg_catalog.replace(pg_catalog.convert_from(publicado,'UTF8'),
    '"id":"ct.plan.firma.sintetico"','"id":"'||mala||'"'),'UTF8');
  rechazo:=false;
  BEGIN
   PERFORM vec_catalogos_configurables.validar_plan_nominal_firma_v1(alterado,
    pg_catalog.encode(pg_catalog.sha256(alterado),'hex'));
  EXCEPTION WHEN SQLSTATE '22023' THEN rechazo:=true; END;
  IF NOT rechazo THEN RAISE EXCEPTION 'CC7: ID fuera de intersección aceptado: %',mala USING ERRCODE='55000'; END IF;
  alterado:=pg_catalog.convert_to(pg_catalog.replace(pg_catalog.convert_from(publicado,'UTF8'),
    '"clave":"paso_1"','"clave":"'||mala||'"'),'UTF8');
  rechazo:=false;
  BEGIN
   PERFORM vec_catalogos_configurables.validar_plan_nominal_firma_v1(alterado,
    pg_catalog.encode(pg_catalog.sha256(alterado),'hex'));
  EXCEPTION WHEN SQLSTATE '22023' THEN rechazo:=true; END;
  IF NOT rechazo THEN RAISE EXCEPTION 'CC7: entrada fuera de intersección aceptada: %',mala USING ERRCODE='55000'; END IF;
 END LOOP;
END $vector$;
ROLLBACK;
