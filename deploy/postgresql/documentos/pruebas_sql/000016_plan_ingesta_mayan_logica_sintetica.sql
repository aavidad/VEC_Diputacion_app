\set ON_ERROR_STOP on
-- SOLO PG18 desechable. Sustituye AD3 durante ROLLBACK para ensayar la lógica
-- de D16; no acredita COSE, concesión V3 real ni despliegue.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE c jsonb:=convert_from(p_capacidad,'UTF8')::jsonb; d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 RETURN QUERY SELECT d->>'decision_ref',c->>'efecto_ref',c->>'huella_efecto_sha256',
  repeat('b',64),'audit:d16:sintetico',clock_timestamp(),NOT coalesce((c->>'replay')::boolean,false);
END $f$;
RESET ROLE;
SET LOCAL ROLE vec_documentos_propietario;
CREATE FUNCTION vec_documentos.ensayo_reclamar_d16(
 p_recurso text,p_persona text,p_clave text,p_huella text,p_version bigint,p_indice text,p_huella_carga text) RETURNS jsonb
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT vec_documentos.reclamar_reserva_recurso_carga_v1(p_recurso,p_persona,p_clave,p_huella,p_version,p_indice,p_huella_carga)
$f$;
CREATE FUNCTION vec_documentos.ensayo_contar_outbox_d16(p_plan text) RETURNS bigint
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
 SELECT count(*) FROM vec_documentos.outbox WHERE tipo='ingesta_mayan_verificada' AND recurso_ref=p_plan
$f$;
RESET ROLE;
CREATE ROLE vec_documentos_d16_ensayo LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT;
GRANT vec_documentos_ejecutor TO vec_documentos_d16_ensayo WITH INHERIT TRUE, SET FALSE;
GRANT EXECUTE ON FUNCTION vec_documentos.ensayo_reclamar_d16(text,text,text,text,bigint,text,text) TO vec_documentos_d16_ensayo;
GRANT EXECUTE ON FUNCTION vec_documentos.ensayo_contar_outbox_d16(text) TO vec_documentos_d16_ensayo;
CREATE SCHEMA ensayo_doc16 AUTHORIZATION postgres;
GRANT USAGE ON SCHEMA ensayo_doc16 TO vec_documentos_d16_ensayo;
CREATE FUNCTION ensayo_doc16.material(p_pre bytea,p_accion text,p_recurso text,p_ambito text,p_decision text,p_replay boolean DEFAULT false)
RETURNS TABLE(capacidad bytea,decision bytea,auth jsonb)
LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $f$
DECLARE h text:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"preimagen_sha256":"'||encode(sha256(p_pre),'hex')||'"}}','UTF8')),'hex');
BEGIN
 capacidad:=convert_to(jsonb_build_object('audiencia_consumo','vec_documentos.operacion.v1',
  'operacion',p_accion,'efecto_ref',p_recurso,'huella_efecto_sha256',h,'replay',p_replay)::text,'UTF8');
 decision:=convert_to(jsonb_build_object('decision_ref',p_decision,'accion',p_accion,'modulo_id','documentos',
  'recurso_ref',p_recurso,'contexto_recurso_huella_sha256',h,
  'principal_id','per_'||repeat('a',32),
  'perfil_activo_ref','perfil:00000000-0000-4000-8000-000000000001',
  'finalidad','custodiar_evidencia_meritos',
  'correlacion_ref','corr:00000000-0000-4000-8000-000000000001',
  'obligaciones','[]'::jsonb)::text,'UTF8');
 auth:=jsonb_build_object('accion',p_accion,'recurso_ref',p_recurso,'ambito_ref',p_ambito,
  'principal_id','per_'||repeat('a',32),
  'perfil_activo_ref','perfil:00000000-0000-4000-8000-000000000001',
  'finalidad','custodiar_evidencia_meritos',
  'correlacion_ref','corr:00000000-0000-4000-8000-000000000001');
 RETURN NEXT;
END $f$;
GRANT EXECUTE ON FUNCTION ensayo_doc16.material(bytea,text,text,text,text,boolean) TO vec_documentos_d16_ensayo;
SET SESSION AUTHORIZATION vec_documentos_d16_ensayo;
DO $probar$
DECLARE persona text:='per_'||repeat('a',32);
 hecho text:='ref:'||repeat('1',64); candidato2 text:='ref:'||repeat('9',64);
 clave text:='ref:'||repeat('2',64); carga text:='ref:'||repeat('3',64);
 sha text:=repeat('c',64); hpolitica text:=repeat('d',64);
 reservar text:='documentos.ingesta_mayan.reservar_recurso';
 preparar text:='documentos.ingesta_mayan.preparar';
 despachar text:='documentos.ingesta_mayan.despachar';
 resolver text:='documentos.ingesta_mayan.resolver';
 conciliar text:='documentos.ingesta_mayan.conciliar';
 pre bytea; mat record; r1 jsonb; r2 jsonb; r3 jsonb; p1 jsonb; p2 jsonb;
 plan text; doc jsonb; dato jsonb; resultado jsonb;
BEGIN
 dato:=jsonb_build_object('recurso_ref',hecho,'persona_ref',persona,
  'registro_contexto_ref','rca_'||repeat('b',32),'huella_contexto_sha256',repeat('a',64),
  'clave_idempotencia',clave,'huella_solicitud_sha256',sha,'estado','','version',0,
  'creada_en','0001-01-01T00:00:00Z','expira_en','0001-01-01T00:00:00Z');
 pre:=convert_to(jsonb_build_object('accion',reservar,'dato',dato)::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc16.material(pre,reservar,clave,clave,'decision:d16:1',false);
 r1:=vec_documentos.operar_reserva_recurso_mayan_v1(pre,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF r1#>>'{reserva,recurso_ref}' IS DISTINCT FROM hecho OR r1#>>'{reserva,version}' IS DISTINCT FROM '1'
 THEN RAISE EXCEPTION 'D16: primera reserva no fijó recurso'; END IF;
 dato:=dato||jsonb_build_object('recurso_ref',candidato2);
 pre:=convert_to(jsonb_build_object('accion',reservar,'dato',dato)::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc16.material(pre,reservar,clave,clave,'decision:d16:2',false);
 r2:=vec_documentos.operar_reserva_recurso_mayan_v1(pre,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF r2#>>'{reserva,recurso_ref}' IS DISTINCT FROM hecho
 THEN RAISE EXCEPTION 'D16: replay reemitió otro hecho_ref'; END IF;
 dato:=dato||jsonb_build_object('huella_solicitud_sha256',repeat('e',64));
 pre:=convert_to(jsonb_build_object('accion',reservar,'dato',dato)::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc16.material(pre,reservar,clave,clave,'decision:d16:3',false);
 BEGIN
  PERFORM vec_documentos.operar_reserva_recurso_mayan_v1(pre,mat.auth,mat.capacidad,mat.decision,
   '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
  RAISE EXCEPTION 'D16: clave divergente aceptada';
 EXCEPTION WHEN unique_violation THEN NULL; END;
 resultado:=vec_documentos.ensayo_reclamar_d16(hecho,persona,clave,sha,1,
  'hmac-sha256:indice_v1:'||repeat('a',64),'hmac-sha256:solicitud_v1:'||repeat('b',64));
 IF resultado->>'estado' IS DISTINCT FROM 'vinculada' OR resultado->>'version_reserva' IS DISTINCT FROM '2'
 THEN RAISE EXCEPTION 'D16: S3 no vinculó reserva'; END IF;
 dato:=jsonb_build_object(
  'plan_ref','','carga_id',carga,'persona_ref',persona,'registro_contexto_ref','rca_'||repeat('b',32),
  'huella_contexto_sha256',repeat('a',64),'expediente_ref',hecho,'evidencia_ref','ref:'||repeat('4',64),
  'tipo_ref','ref:'||repeat('5',64),'clave_reserva',clave,'clave_idempotencia','ref:'||repeat('6',64),
  'huella_solicitud_sha256',repeat('7',64),'huella_sha256',sha,'mime','application/pdf','tamano',100,
  'carga_version',5,'politica_ref','ref:'||repeat('8',64),'version_politica',1,
  'huella_politica_sha256',hpolitica,'conservacion_hasta','2035-01-01T00:00:00Z',
  'estado_politica','provisional','proteccion','conservacion','version',0,'estado','',
  'documento_id','','archivo_id','','version_id','','creado_en','0001-01-01T00:00:00Z',
  'actualizado_en','0001-01-01T00:00:00Z');
 pre:=convert_to(jsonb_build_object('accion',preparar,'dato',dato)::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc16.material(pre,preparar,carga,hecho,'decision:d16:4',false);
 p1:=vec_documentos.operar_ingesta_mayan_v1(pre,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 plan:=p1#>>'{plan,plan_ref}';
 IF plan !~ '^plan:[0-9a-f-]{36}$' OR p1#>>'{plan,expediente_ref}' IS DISTINCT FROM hecho
 THEN RAISE EXCEPTION 'D16: plan Mayan perdió recurso'; END IF;
 SELECT * INTO STRICT mat FROM ensayo_doc16.material(pre,preparar,carga,hecho,'decision:d16:5',true);
 p2:=vec_documentos.operar_ingesta_mayan_v1(pre,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF p2#>>'{plan,plan_ref}' IS DISTINCT FROM plan THEN RAISE EXCEPTION 'D16: replay plan distinto'; END IF;
 doc:=jsonb_build_object('documento_id','','archivo_id','','version_id','','huella_sha256','','mime','','tamano',0);
 dato:=jsonb_build_object('plan_ref',plan,'expediente_ref',hecho,'version_esperada',1,'estado','despachado','documento',doc);
 pre:=convert_to(jsonb_build_object('accion',despachar,'dato',dato)::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc16.material(pre,despachar,plan,hecho,'decision:d16:6',false);
 p1:=vec_documentos.operar_ingesta_mayan_v1(pre,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF p1->>'nuevo_despacho' IS DISTINCT FROM 'true' OR p1#>>'{plan,version}' IS DISTINCT FROM '2'
 THEN RAISE EXCEPTION 'D16: no marcó despacho antes de POST'; END IF;
 SELECT * INTO STRICT mat FROM ensayo_doc16.material(pre,despachar,plan,hecho,'decision:d16:7',true);
 p2:=vec_documentos.operar_ingesta_mayan_v1(pre,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF p2->>'nuevo_despacho' IS DISTINCT FROM 'false' THEN RAISE EXCEPTION 'D16: replay permitió segundo POST'; END IF;
 doc:=jsonb_build_object('documento_id','15','archivo_id','','version_id','','huella_sha256',sha,'mime','application/pdf','tamano',100);
 dato:=jsonb_build_object('plan_ref',plan,'expediente_ref',hecho,'version_esperada',2,'estado','indeterminado','documento',doc);
 pre:=convert_to(jsonb_build_object('accion',resolver,'dato',dato)::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc16.material(pre,resolver,plan,hecho,'decision:d16:8',false);
 p1:=vec_documentos.operar_ingesta_mayan_v1(pre,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF p1#>>'{plan,estado}' IS DISTINCT FROM 'indeterminado' THEN RAISE EXCEPTION 'D16: timeout no quedó indeterminado'; END IF;
 doc:=jsonb_build_object('documento_id','16','archivo_id','20','version_id','30','huella_sha256',sha,'mime','application/pdf','tamano',100);
 dato:=jsonb_build_object('plan_ref',plan,'expediente_ref',hecho,'version_esperada',3,'estado','confirmado','documento',doc);
 pre:=convert_to(jsonb_build_object('accion',conciliar,'dato',dato)::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc16.material(pre,conciliar,plan,hecho,'decision:d16:9',false);
 BEGIN
  PERFORM vec_documentos.operar_ingesta_mayan_v1(pre,mat.auth,mat.capacidad,mat.decision,
   '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
  RAISE EXCEPTION 'D16: documento Mayan distinto aceptado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 doc:=doc||jsonb_build_object('documento_id','15');
 dato:=dato||jsonb_build_object('documento',doc);
 pre:=convert_to(jsonb_build_object('accion',conciliar,'dato',dato)::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc16.material(pre,conciliar,plan,hecho,'decision:d16:10',false);
 p1:=vec_documentos.operar_ingesta_mayan_v1(pre,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF p1#>>'{plan,estado}' IS DISTINCT FROM 'confirmado' OR p1#>>'{plan,version}' IS DISTINCT FROM '4'
   OR vec_documentos.ensayo_contar_outbox_d16(plan)<>1
 THEN RAISE EXCEPTION 'D16: conciliación no confirmó evidencia única'; END IF;
END $probar$;
ROLLBACK;
