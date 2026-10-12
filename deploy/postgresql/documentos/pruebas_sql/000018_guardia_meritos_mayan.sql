\set ON_ERROR_STOP on
-- Focal lógico D18. Se ejecuta solo tras AD238/D16, AD240/D17 y D18 en
-- PostgreSQL 18 desechable. Las filas sintéticas se retiran con ROLLBACK.
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL vec.documentos.expediente_ref='ref:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb';
SET LOCAL vec.documentos.persona_ref='per_0123456789abcdefghijkl';
-- Solo en este ROLLBACK se sustituye el consumo AD3 por un doble transaccional
-- que distingue alta de replay. No acredita COSE ni cambia la función viva.
CREATE TABLE vec_documentos.ensayo_consumo_d18(huella text PRIMARY KEY);
CREATE OR REPLACE FUNCTION vec_documentos.consumir_v3_v2(
 p_preimagen bytea,p_accion text,p_recurso text,p_principal text,p_perfil text,p_finalidad text,p_correlacion text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $doble$
DECLARE h text:=encode(sha256(p_preimagen),'hex'); n integer;
BEGIN
 INSERT INTO vec_documentos.ensayo_consumo_d18(huella) VALUES(h) ON CONFLICT DO NOTHING;
 GET DIAGNOSTICS n=ROW_COUNT;
 RETURN QUERY SELECT 'decision:prueba'::text,p_recurso,h,repeat('b',64),
  'audit:prueba'::text,clock_timestamp(),n=1;
END $doble$;
DO $test$
DECLARE hecho text:='ref:'||repeat('b',64);
 persona text:='per_0123456789abcdefghijkl'; plan text:='plan:123e4567-e89b-42d3-a456-426614174018';
 carga text:='ref:'||repeat('d',64); h text:=repeat('c',64);
 ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp());
 fila vec_documentos.referencia_externa%ROWTYPE; otra vec_documentos.referencia_externa%ROWTYPE;
 respuesta jsonb; repetida jsonb; n bigint; preimagen bytea; auth jsonb;
BEGIN
 IF to_regprocedure('vec_documentos.validar_plan_carga_para_lectura_v1(text,text,text,text,bigint)') IS NULL
    OR has_function_privilege('vec_documentos_ejecutor',
      'vec_documentos.validar_plan_carga_para_lectura_v1(text,text,text,text,bigint)','EXECUTE')
    OR has_function_privilege('vec_documentos_ejecutor',
      'vec_documentos.guardar_referencia_meritos_mayan_v1()','EXECUTE')
    OR has_table_privilege('vec_documentos_ejecutor','vec_documentos.plan_ingesta_mayan','SELECT')
 THEN RAISE EXCEPTION 'D18: ACL/fachada incorrecta'; END IF;

 fila.id:='ref:'||encode(sha256(convert_to('vec.documentos.meritos.mayan.referencia.v1:'||plan,'UTF8')),'hex');
 fila.numero_vec:='VEC-2026-18001'; fila.clave_idempotencia:=plan;
 fila.principal_ref:=persona; fila.modulo_id:='meritos'; fila.expediente_ref:=hecho;
 fila.tipo_ref:='ref:'||repeat('1',64); fila.version:=1; fila.huella_sha256:=h;
 fila.mime:='application/pdf'; fila.tamano:=100; fila.custodio_id:='mayan';
 fila.custodia_ref:='mayan:21:34:55'; fila.politica_ref:='ref:'||repeat('4',64);
 fila.version_politica:=1; fila.huella_politica_sha256:=h;
 fila.conservacion_hasta:='2035-01-01 UTC'; fila.proteccion:='conservacion';
 fila.estado_politica:='aprobada'; fila.estado_firma:='pendiente_proveedor';
 fila.huella_preimagen_sha256:=h; fila.decision_ref:='decision:prueba';
 fila.auditoria_ad3_ref:='audit:prueba'; fila.creada_en:=ahora;
 BEGIN
  INSERT INTO vec_documentos.referencia_externa SELECT (fila).*;
  RAISE EXCEPTION 'D18: alta directa sin plan aceptada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;

 INSERT INTO vec_documentos.reserva_recurso_documental(
  recurso_ref,persona_ref,clave_idempotencia,huella_solicitud_sha256,registro_contexto_ref,
  huella_contexto_sha256,estado,version,decision_reserva_ref,creada_en,expira_en,actualizada_en)
 VALUES(hecho,persona,'ref:'||repeat('3',64),h,'rca_0123456789abcdefghijkl',h,
  'vinculada',2,'decision:prueba',ahora,ahora+interval '10 minutes',ahora);
 INSERT INTO vec_documentos.plan_ingesta_mayan(
  plan_ref,carga_id,persona_ref,expediente_ref,evidencia_ref,tipo_ref,principal_ref,clave_reserva,clave_idempotencia,
  registro_contexto_ref,huella_contexto_sha256,huella_solicitud_sha256,huella_sha256,mime,tamano,carga_version,
  politica_ref,version_politica,huella_politica_sha256,conservacion_hasta,estado_politica,proteccion,
  version,estado,decision_preparacion_ref,creada_en,actualizada_en)
 VALUES(plan,carga,persona,hecho,'ref:'||repeat('f',64),fila.tipo_ref,persona,
  'ref:'||repeat('3',64),'ref:'||repeat('5',64),'rca_0123456789abcdefghijkl',h,h,h,'application/pdf',100,5,
  fila.politica_ref,1,h,fila.conservacion_hasta,'aprobada','conservacion',
  1,'preparado','decision:prueba',ahora,ahora);
 IF vec_documentos.validar_plan_carga_para_lectura_v1(plan,carga,persona,hecho,5) IS DISTINCT FROM true
 THEN RAISE EXCEPTION 'D18: plan preparado exacto no validado'; END IF;
 BEGIN
  PERFORM vec_documentos.validar_plan_carga_para_lectura_v1(plan,carga,'per_abcdefghijklmnopqrstuvwxyz',hecho,5);
  RAISE EXCEPTION 'D18: persona ajena pudo leer';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_documentos.validar_plan_carga_para_lectura_v1(plan,carga,persona,hecho,4);
  RAISE EXCEPTION 'D18: carga v4 pudo leerse';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  INSERT INTO vec_documentos.referencia_externa SELECT (fila).*;
  RAISE EXCEPTION 'D18: plan preparado registró custodia';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;

 UPDATE vec_documentos.plan_ingesta_mayan SET estado='confirmado',version=3,
  documento_id='21',archivo_id='34',version_id='55' WHERE plan_ref=plan;
 BEGIN
  PERFORM vec_documentos.validar_plan_carga_para_lectura_v1(plan,carga,persona,hecho,5);
  RAISE EXCEPTION 'D18: plan ya despachado abrió S3';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  INSERT INTO vec_documentos.referencia_externa SELECT (fila).*;
  RAISE EXCEPTION 'D18: faltan outbox/auditoría confirmados';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 INSERT INTO vec_documentos.outbox(tipo,recurso_ref,expediente_ref,huella_sha256,registrada_en)
 VALUES('ingesta_mayan_verificada',plan,hecho,h,ahora);
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,
  perfil_ref,finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES('documentos.ingesta_mayan.resolver',plan,hecho,persona,'perfil:prueba',
  'custodiar_evidencia_meritos','corr:prueba','decision:prueba','audit:prueba','confirmado',ahora);

 FOR n IN 1..9 LOOP
  otra:=fila;
  CASE n
   WHEN 1 THEN otra.principal_ref:='per_abcdefghijklmnopqrstuvwxyz';
   WHEN 2 THEN otra.expediente_ref:='ref:'||repeat('8',64);
   WHEN 3 THEN otra.custodio_id:='otro';
   WHEN 4 THEN otra.custodia_ref:='mayan:21:34:56';
   WHEN 5 THEN otra.huella_sha256:=repeat('e',64);
   WHEN 6 THEN otra.politica_ref:='ref:'||repeat('9',64);
   WHEN 7 THEN otra.version_politica:=2;
   WHEN 8 THEN otra.tamano:=101;
   WHEN 9 THEN otra.clave_idempotencia:='ref:'||repeat('a',64);
  END CASE;
  BEGIN
   INSERT INTO vec_documentos.referencia_externa SELECT (otra).*;
   RAISE EXCEPTION 'D18: cruce % aceptado',n;
  EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 END LOOP;

 preimagen:=convert_to(jsonb_build_object(
  'accion','documentos.externo.registrar','id',fila.id,'clave_idempotencia',fila.clave_idempotencia,
  'modulo_id',fila.modulo_id,'expediente_ref',fila.expediente_ref,'tipo_ref',fila.tipo_ref,
  'version',fila.version,'mime',fila.mime,'tamano',fila.tamano,'huella_sha256',fila.huella_sha256,
  'custodio_id',fila.custodio_id,'custodia_ref',fila.custodia_ref,
  'politica_ref',fila.politica_ref,'version_politica',fila.version_politica,
  'huella_politica_sha256',fila.huella_politica_sha256,'proteccion',fila.proteccion,
  'conservacion_hasta','2035-01-01T00:00:00Z','estado_politica',fila.estado_politica)::text,'UTF8');
 auth:=jsonb_build_object('accion','documentos.externo.registrar','recurso_ref',fila.id,
  'ambito_ref',hecho,'principal_id',persona,'perfil_activo_ref','perfil:prueba',
  'finalidad','registrar_documento_externo','correlacion_ref','corr:prueba');
 SELECT vec_documentos.registrar_referencia_externa_v1(preimagen,auth,
  NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea,1,1,
  NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea) INTO respuesta;
 SELECT vec_documentos.registrar_referencia_externa_v1(preimagen,auth,
  NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea,1,1,
  NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea) INTO repetida;
 IF respuesta IS NULL OR respuesta IS DISTINCT FROM repetida OR
    respuesta->>'custodia_ref' IS DISTINCT FROM 'mayan:21:34:55' OR
    (SELECT count(*) FROM vec_documentos.referencia_externa WHERE id=fila.id)<>1 OR
    (SELECT count(*) FROM vec_documentos.outbox WHERE tipo='documento_externo_registrado' AND recurso_ref=fila.id)<>1 OR
    (SELECT count(*) FROM vec_documentos.auditoria_operacion WHERE accion='documentos.externo.registrar'
      AND recurso_ref=fila.id AND resultado='repetido')<>1
 THEN RAISE EXCEPTION 'D18: proyección/replay documental divergente'; END IF;
 BEGIN
  UPDATE vec_documentos.referencia_externa SET huella_sha256=repeat('e',64) WHERE id=fila.id;
  GET DIAGNOSTICS n=ROW_COUNT;
  IF n<>0 THEN RAISE EXCEPTION 'D18: UPDATE eludió inmutabilidad D3'; END IF;
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  DELETE FROM vec_documentos.referencia_externa WHERE id=fila.id;
  GET DIAGNOSTICS n=ROW_COUNT;
  IF n<>0 THEN RAISE EXCEPTION 'D18: DELETE eludió inmutabilidad D3'; END IF;
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 otra:=fila; otra.id:='ref:'||repeat('7',64); otra.numero_vec:='VEC-2026-18002';
 otra.clave_idempotencia:='ref:'||repeat('6',64); otra.modulo_id:='dietas';
 otra.custodio_id:='dietas'; otra.custodia_ref:='dietas:1';
 INSERT INTO vec_documentos.referencia_externa SELECT (otra).*;
 IF (SELECT count(*) FROM vec_documentos.referencia_externa WHERE id=otra.id)<>1
 THEN RAISE EXCEPTION 'D18: otro módulo alterado'; END IF;
END $test$;
ROLLBACK;
