\set ON_ERROR_STOP on
-- Ensayo estructural D16 en PostgreSQL 18 desechable tras AD238 y D16.
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL vec.documentos.expediente_ref='ref:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb';
SET LOCAL vec.documentos.persona_ref='per_0123456789abcdefghijkl';
DO $test$
DECLARE p text:='plan:123e4567-e89b-42d3-a456-426614174000';
 e text:='ref:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb';
 persona text:='per_0123456789abcdefghijkl'; clave text:='ref:'||repeat('3',64);
 h text:=repeat('c',64); ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp());
 vinculo jsonb;
BEGIN
 IF to_regprocedure('vec_documentos.operar_reserva_recurso_mayan_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
   OR to_regprocedure('vec_documentos.operar_ingesta_mayan_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
   OR has_table_privilege('vec_documentos_ejecutor','vec_documentos.reserva_recurso_documental','SELECT')
   OR has_table_privilege('vec_documentos_ejecutor','vec_documentos.plan_ingesta_mayan','INSERT')
   OR has_function_privilege('vec_documentos_ejecutor',
      'vec_documentos.reclamar_reserva_recurso_carga_v1(text,text,text,text,bigint,text,text)','EXECUTE')
   OR NOT has_function_privilege('vec_documentos_ejecutor',
      'vec_documentos.operar_reserva_recurso_mayan_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'D16: ACL/fachada divergente'; END IF;
 INSERT INTO vec_documentos.reserva_recurso_documental(
  recurso_ref,persona_ref,clave_idempotencia,huella_solicitud_sha256,registro_contexto_ref,
  huella_contexto_sha256,estado,version,decision_reserva_ref,creada_en,expira_en,actualizada_en)
 VALUES(e,persona,clave,h,'rca_0123456789abcdefghijkl',h,'reservada',1,'decision:prueba',ahora,ahora+interval '10 minutes',ahora);
 SELECT vec_documentos.reclamar_reserva_recurso_carga_v1(
  e,persona,clave,h,1,'hmac-sha256:indice:'||repeat('a',64),'hmac-sha256:solicitud:'||repeat('b',64)) INTO vinculo;
 IF vinculo->>'recurso_ref' IS DISTINCT FROM e OR vinculo->>'persona_ref' IS DISTINCT FROM persona
   OR vinculo->>'estado' IS DISTINCT FROM 'vinculada' OR vinculo->>'version_reserva' IS DISTINCT FROM '2'
 THEN RAISE EXCEPTION 'D16: reclamación S3 no vinculó la misma reserva'; END IF;
 BEGIN
  PERFORM vec_documentos.reclamar_reserva_recurso_carga_v1(
   e,persona,clave,h,1,'hmac-sha256:indice:'||repeat('a',64),'hmac-sha256:solicitud:'||repeat('b',64));
  RAISE EXCEPTION 'D16: reclamación aceptó versión antigua';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 INSERT INTO vec_documentos.plan_ingesta_mayan(
  plan_ref,carga_id,persona_ref,expediente_ref,evidencia_ref,tipo_ref,principal_ref,clave_reserva,clave_idempotencia,
  registro_contexto_ref,huella_contexto_sha256,huella_solicitud_sha256,huella_sha256,mime,tamano,carga_version,
  politica_ref,version_politica,huella_politica_sha256,conservacion_hasta,estado_politica,proteccion,
  version,estado,decision_preparacion_ref,creada_en,actualizada_en)
 VALUES(p,'ref:'||repeat('d',64),persona,e,'ref:'||repeat('f',64),'ref:'||repeat('1',64),persona,
  clave,'ref:'||repeat('5',64),'rca_0123456789abcdefghijkl',h,h,h,'application/pdf',100,5,
  'ref:'||repeat('4',64),1,h,'2035-01-01 UTC','provisional','conservacion',1,'preparado','decision:prueba',ahora,ahora);
 IF (SELECT count(*) FROM vec_documentos.plan_ingesta_mayan WHERE plan_ref=p)<>1
 THEN RAISE EXCEPTION 'D16: plan no visible en ámbito propio'; END IF;
 BEGIN
  UPDATE vec_documentos.plan_ingesta_mayan SET estado='confirmado',version=3 WHERE plan_ref=p;
  RAISE EXCEPTION 'D16: confirmación sin evidencia admitida';
 EXCEPTION WHEN check_violation THEN NULL; END;
 BEGIN
  INSERT INTO vec_documentos.reserva_recurso_documental(
   recurso_ref,persona_ref,clave_idempotencia,huella_solicitud_sha256,registro_contexto_ref,
   huella_contexto_sha256,estado,version,decision_reserva_ref,creada_en,expira_en,actualizada_en)
  VALUES('ref:'||repeat('9',64),persona,clave,h,'rca_0123456789abcdefghijkl',h,'reservada',1,'decision:prueba',ahora,ahora+interval '10 minutes',ahora);
  RAISE EXCEPTION 'D16: la misma clave creó otro recurso';
 EXCEPTION WHEN unique_violation THEN NULL; END;
 PERFORM set_config('vec.documentos.persona_ref','per_abcdefghijklmnopqrstuvwxyz',true);
 IF EXISTS(SELECT 1 FROM vec_documentos.reserva_recurso_documental WHERE recurso_ref=e)
 THEN RAISE EXCEPTION 'D16: RLS leyó reserva ajena'; END IF;
 PERFORM set_config('vec.documentos.expediente_ref','ref:'||repeat('8',64),true);
 IF EXISTS(SELECT 1 FROM vec_documentos.plan_ingesta_mayan WHERE plan_ref=p)
 THEN RAISE EXCEPTION 'D16: RLS leyó plan ajeno'; END IF;
END $test$;
DO $denegacion$
DECLARE accion text; dato jsonb; auth jsonb; recurso text; ambito text;
 persona text:='per_0123456789abcdefghijkl'; clave text:='ref:'||repeat('3',64);
 hecho text:='ref:'||repeat('b',64); plan text:='plan:123e4567-e89b-42d3-a456-426614174000'; aceptada boolean;
BEGIN
 FOREACH accion IN ARRAY ARRAY[
  'documentos.ingesta_mayan.reservar_recurso','documentos.ingesta_mayan.consultar_reserva',
  'documentos.ingesta_mayan.abandonar_recurso','documentos.ingesta_mayan.preparar',
  'documentos.ingesta_mayan.despachar','documentos.ingesta_mayan.resolver',
  'documentos.ingesta_mayan.conciliar','documentos.ingesta_mayan.consultar'] LOOP
  IF accion IN ('documentos.ingesta_mayan.reservar_recurso','documentos.ingesta_mayan.consultar_reserva',
    'documentos.ingesta_mayan.abandonar_recurso') THEN
   recurso:=clave; ambito:=clave;
   dato:=CASE WHEN accion='documentos.ingesta_mayan.reservar_recurso'
    THEN jsonb_build_object('recurso_ref',hecho,'persona_ref',persona,'clave_idempotencia',clave)
    ELSE jsonb_build_object('clave_idempotencia',clave) END;
  ELSIF accion='documentos.ingesta_mayan.preparar' THEN
   recurso:='ref:'||repeat('d',64); ambito:=hecho;
   dato:=jsonb_build_object('carga_id',recurso,'expediente_ref',ambito);
  ELSE
   recurso:=plan; ambito:=hecho;
   dato:=jsonb_build_object('plan_ref',recurso,'expediente_ref',ambito);
  END IF;
  auth:=jsonb_build_object('accion',accion,'recurso_ref',recurso,'ambito_ref',ambito,
   'principal_id',persona,'perfil_activo_ref','prf_0123456789abcdefghijkl',
   'finalidad','custodiar_evidencia_meritos','correlacion_ref','corr:prueba');
  aceptada:=false;
  BEGIN
   IF accion IN ('documentos.ingesta_mayan.reservar_recurso','documentos.ingesta_mayan.consultar_reserva',
     'documentos.ingesta_mayan.abandonar_recurso') THEN
    PERFORM vec_documentos.operar_reserva_recurso_mayan_v1(
     convert_to(jsonb_build_object('accion',accion,'dato',dato)::text,'UTF8'),auth,
     NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
   ELSE
    PERFORM vec_documentos.operar_ingesta_mayan_v1(
     convert_to(jsonb_build_object('accion',accion,'dato',dato)::text,'UTF8'),auth,
     NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
   END IF;
   aceptada:=true;
  EXCEPTION WHEN OTHERS THEN NULL; END;
  IF aceptada THEN RAISE EXCEPTION 'D16: % aceptó efecto sin material V3',accion; END IF;
 END LOOP;
END $denegacion$;
ROLLBACK;
