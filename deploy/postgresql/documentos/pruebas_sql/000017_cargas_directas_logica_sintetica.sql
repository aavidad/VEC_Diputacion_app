\set ON_ERROR_STOP on
-- SOLO clon PG18 desechable. Sustituye AD3 dentro de ROLLBACK para probar
-- atomicidad, CAS y replay D17; no acredita COSE ni concesión V3 real.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE c jsonb:=convert_from(p_capacidad,'UTF8')::jsonb; d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 RETURN QUERY SELECT d->>'decision_ref',c->>'efecto_ref',c->>'huella_efecto_sha256',
  repeat('b',64),'audit:d17:sintetico',clock_timestamp(),true;
END $f$;
RESET ROLE;
SET LOCAL ROLE vec_documentos_propietario;
CREATE FUNCTION vec_documentos.ensayo_audiencia_carga_d17(
 p_pre bytea,p_accion text,p_recurso text,p_capacidad bytea,p_decision bytea) RETURNS boolean
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE v record;
BEGIN
 SELECT * INTO STRICT v FROM vec_documentos.consumir_carga_directa_v3_v1(
  p_pre,p_accion,p_recurso,'per_'||repeat('a',32),
  'perfil:00000000-0000-4000-8000-000000000001','custodiar_evidencia_meritos',
  'corr:00000000-0000-4000-8000-000000000001',p_capacidad,p_decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 RETURN v.consumo_nuevo;
END $f$;
RESET ROLE;
CREATE ROLE vec_documentos_d17_ensayo LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT;
GRANT vec_documentos_ejecutor TO vec_documentos_d17_ensayo WITH INHERIT TRUE, SET FALSE;
GRANT EXECUTE ON FUNCTION vec_documentos.ensayo_audiencia_carga_d17(bytea,text,text,bytea,bytea) TO vec_documentos_d17_ensayo;
CREATE SCHEMA ensayo_doc17 AUTHORIZATION postgres;
GRANT USAGE ON SCHEMA ensayo_doc17 TO vec_documentos_d17_ensayo;
CREATE FUNCTION ensayo_doc17.material(p_pre bytea,p_accion text,p_recurso text,p_ambito text,p_decision text)
RETURNS TABLE(capacidad bytea,decision bytea,auth jsonb)
LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $f$
DECLARE h text:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"preimagen_sha256":"'||encode(sha256(p_pre),'hex')||'"}}','UTF8')),'hex');
 audiencia text;
BEGIN
 audiencia:=CASE p_accion
  WHEN 'documentos.carga.preparar_reserva' THEN 'vec_documentos.carga.preparar_reserva.v1'
  WHEN 'documentos.carga.preparar_confirmacion' THEN 'vec_documentos.carga.preparar_confirmacion.v1'
  WHEN 'documentos.carga.preparar_respuesta' THEN 'vec_documentos.carga.preparar_respuesta.v1'
  WHEN 'documentos.carga.consultar_propia' THEN 'vec_documentos.carga.consultar_propia.v1'
  WHEN 'documentos.carga.conciliar' THEN 'vec_documentos.carga.conciliar.v1'
  WHEN 'documentos.carga.confirmar_intencion' THEN 'vec_documentos.carga.confirmar_intencion.v1'
  WHEN 'documentos.carga.confirmar_resultado' THEN 'vec_documentos.carga.confirmar_resultado.v1'
  WHEN 'vec.documentos.carga.analizar' THEN 'vec_documentos.carga.analizar.v1'
  WHEN 'documentos.carga.promover_intencion' THEN 'vec_documentos.carga.promover_intencion.v1'
  WHEN 'documentos.carga.promover_resultado' THEN 'vec_documentos.carga.promover_resultado.v1'
  ELSE 'vec_documentos.operacion.v1' END;
 capacidad:=convert_to(jsonb_build_object('audiencia_consumo',audiencia,'decision_ref',p_decision,
  'operacion',p_accion,'efecto_ref',p_recurso,'huella_efecto_sha256',h)::text,'UTF8');
 decision:=convert_to(jsonb_build_object('decision_ref',p_decision,'accion',p_accion,
  'modulo_id','documentos','recurso_ref',p_recurso,'contexto_recurso_huella_sha256',h,
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
GRANT EXECUTE ON FUNCTION ensayo_doc17.material(bytea,text,text,text,text) TO vec_documentos_d17_ensayo;
SET SESSION AUTHORIZATION vec_documentos_d17_ensayo;
DO $probar$
DECLARE persona text:='per_'||repeat('a',32); hecho text:='ref:'||repeat('1',64);
 carga text; clave text:='ref:'||repeat('2',64);
 indice text:='hmac-sha256:indice_v1:'||repeat('a',64);
 huella text:='hmac-sha256:solicitud_v1:'||repeat('b',64);
 reserva jsonb; pre bytea; j1 jsonb; j2 jsonb; d jsonb; m jsonb;
 mat record; x jsonb; y jsonb; z jsonb; mala jsonb; carga_mala jsonb;
 pre_carga bytea; pre_manifiesto bytea; pre_mala bytea;
 recibo jsonb; orden jsonb; j3 jsonb; j4 jsonb; j5 jsonb; objeto jsonb; admitido jsonb; analisis jsonb;
 expira text:=to_char(clock_timestamp()+interval '5 minutes','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 accion text; capacidad_cruzada bytea;
BEGIN
 carga:='ref:'||encode(sha256(convert_to('vec:carga-documental:id:v3','UTF8')||decode('00','hex')||
   convert_to(indice,'UTF8')),'hex');
 FOREACH accion IN ARRAY ARRAY[
  'documentos.carga.preparar_reserva','documentos.carga.preparar_confirmacion',
  'documentos.carga.preparar_respuesta','documentos.carga.consultar_propia',
  'documentos.carga.conciliar','documentos.carga.confirmar_intencion',
  'documentos.carga.confirmar_resultado',
  'vec.documentos.carga.analizar','documentos.carga.promover_intencion',
  'documentos.carga.promover_resultado'] LOOP
  pre:=convert_to(jsonb_build_object('accion',accion,'carga_id',carga)::text,'UTF8');
  SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,accion,carga,hecho,'decision:d17:audiencia:'||accion);
  IF vec_documentos.ensayo_audiencia_carga_d17(pre,accion,carga,mat.capacidad,mat.decision) IS NOT TRUE
  THEN RAISE EXCEPTION 'D17: audiencia correcta rechazada %',accion; END IF;
  capacidad_cruzada:=convert_to((convert_from(mat.capacidad,'UTF8')::jsonb||
   jsonb_build_object('audiencia_consumo','vec_documentos.carga.ajena.v1'))::text,'UTF8');
  BEGIN
   PERFORM vec_documentos.ensayo_audiencia_carga_d17(pre,accion,carga,capacidad_cruzada,mat.decision);
   RAISE EXCEPTION 'D17: audiencia cruzada aceptada %',accion;
  EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 END LOOP;
 -- Límites se comprueban antes de convertir los bytea a JSON y antes de V3.
 BEGIN
  PERFORM vec_documentos.reservar_carga_directa_v1(convert_to(repeat('x',16385),'UTF8'),
   '\x7b7d'::bytea,'{}'::jsonb,'{}'::jsonb,'\x01'::bytea,'\x01'::bytea,
   '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
  RAISE EXCEPTION 'D17: preimagen enorme aceptada';
 EXCEPTION WHEN OTHERS THEN IF SQLSTATE<>'42501' THEN RAISE; END IF; END;
 BEGIN
  PERFORM vec_documentos.reservar_carga_directa_v1('\x7b7d'::bytea,
   convert_to(repeat('x',65537),'UTF8'),'{}'::jsonb,'{}'::jsonb,
   '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,1,1,
   '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
  RAISE EXCEPTION 'D17: carga enorme aceptada';
 EXCEPTION WHEN OTHERS THEN IF SQLSTATE<>'42501' THEN RAISE; END IF; END;
 reserva:=jsonb_build_object('recurso_ref',hecho,'persona_ref',persona,
  'registro_contexto_ref','rca_'||repeat('b',32),'huella_contexto_sha256',repeat('a',64),
  'clave_idempotencia',clave,'huella_solicitud_sha256',repeat('c',64),'estado','',
  'version',0,'creada_en','0001-01-01T00:00:00Z','expira_en','0001-01-01T00:00:00Z');
 pre:=convert_to(jsonb_build_object('accion','documentos.ingesta_mayan.reservar_recurso','dato',reserva)::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,'documentos.ingesta_mayan.reservar_recurso',clave,clave,'decision:d17:d16');
 x:=vec_documentos.operar_reserva_recurso_mayan_v1(pre,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF x#>>'{reserva,recurso_ref}' IS DISTINCT FROM hecho THEN RAISE EXCEPTION 'D17: D16 no reservó'; END IF;
 j1:=jsonb_build_object('id',carga,'version',1,'estado','reservada',
  'principal_id',persona,'recurso_ref',hecho,'modulo_id','meritos','tipo_recurso','hecho',
  'finalidad','custodiar_evidencia_meritos','correlacion_ref','corr:00000000-0000-4000-8000-000000000001',
  'indice_idempotencia_hmac',indice,'huella_solicitud_hmac',huella,
  'huella_declarada_sha256',repeat('d',64),'mime_declarado','application/pdf',
  'tamano_declarado',100,'expira_en',expira);
 pre_carga:=convert_to(j1::text,'UTF8');
 m:=jsonb_build_object('accion','documentos.carga.preparar_reserva','carga_id',carga,
  'recurso_ref',hecho,'persona_ref',persona,'clave_reserva',clave,
  'huella_reserva_sha256',repeat('c',64),'version_reserva',1,
  'indice_carga_hmac',indice,'huella_carga_hmac',huella,'mime','application/pdf',
  'tamano',100,'huella_contenido_sha256',repeat('d',64),
  'huella_carga_json_sha256',encode(sha256(pre_carga),'hex'),
  'huella_token_sha256',repeat('e',64));
 pre:=convert_to(m::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,m->>'accion',carga,hecho,'decision:d17:1');
 x:=vec_documentos.reservar_carga_directa_v1(pre,pre_carga,
  jsonb_build_object('recurso_ref',hecho,'persona_ref',persona,'clave_reserva',clave,
   'huella_reserva_sha256',repeat('c',64),'version_reserva',1),mat.auth,mat.capacidad,
  mat.decision,'\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF x->>'carga_id' IS DISTINCT FROM carga OR x->>'estado_operativo' IS DISTINCT FROM 'preparacion_iniciada'
  OR x->>'pendiente' IS DISTINCT FROM 'false' THEN RAISE EXCEPTION 'D17: TX1 no fijó intención'; END IF;
 -- Índice A con capacidad sobre ID B jamás puede proyectar A.
 carga_mala:=j1||jsonb_build_object('id','ref:'||repeat('4',64));
 pre_mala:=convert_to(carga_mala::text,'UTF8');
 mala:=m||jsonb_build_object('carga_id','ref:'||repeat('4',64),
  'huella_carga_json_sha256',encode(sha256(pre_mala),'hex'));
 pre:=convert_to(mala::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,mala->>'accion',mala->>'carga_id',hecho,'decision:d17:B');
 BEGIN
  PERFORM vec_documentos.reservar_carga_directa_v1(pre,pre_mala,
   jsonb_build_object('recurso_ref',hecho,'persona_ref',persona,'clave_reserva',clave,
    'huella_reserva_sha256',repeat('c',64),'version_reserva',1),mat.auth,mat.capacidad,
   mat.decision,'\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
  RAISE EXCEPTION 'D17: decisión B leyó A';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 m:=m||jsonb_build_object('huella_token_sha256',repeat('f',64));
 pre:=convert_to(m::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,m->>'accion',carga,hecho,'decision:d17:2');
 y:=vec_documentos.reservar_carga_directa_v1(pre,pre_carga,
  jsonb_build_object('recurso_ref',hecho,'persona_ref',persona,'clave_reserva',clave,
   'huella_reserva_sha256',repeat('c',64),'version_reserva',1),mat.auth,mat.capacidad,
  mat.decision,'\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF y->>'carga_id' IS DISTINCT FROM carga OR y->>'pendiente' IS DISTINCT FROM 'true'
 THEN RAISE EXCEPTION 'D17: replay creó otra carga'; END IF;
 BEGIN
  PERFORM vec_documentos.confirmar_preparacion_carga_directa_v1('\x7b7d'::bytea,
   '\x7b7d'::bytea,convert_to(repeat('x',65537),'UTF8'),'{}'::jsonb,
   '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,1,1,
   '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
  RAISE EXCEPTION 'D17: manifiesto enorme aceptado';
 EXCEPTION WHEN OTHERS THEN IF SQLSTATE<>'42501' THEN RAISE; END IF; END;
 j2:=j1||jsonb_build_object('version',2,'estado','preparada',
  'vinculo_sesion_hmac','hmac-sha256:sesion_v1:'||repeat('1',64));
 pre_carga:=convert_to(j2::text,'UTF8');
 d:=jsonb_build_object('carga_id',carga,'version_carga',2,
  'huella_carga_sha256',encode(sha256(pre_carga),'hex'),
  'huella_manifiesto_sha256',repeat('7',64),'recurso_ref',hecho,
  'vinculo_sesion_hmac',j2->>'vinculo_sesion_hmac',
  'indice_idempotencia_hmac',indice,'huella_solicitud_hmac',huella,
  'decision_ref','decision:preparar:v1');
 pre_manifiesto:=convert_to(d::text,'UTF8');
 m:=jsonb_build_object('accion','documentos.carga.preparar_confirmacion','carga_id',carga,
  'recurso_ref',hecho,'persona_ref',persona,'huella_token_sha256',repeat('e',64),
  'huella_manifiesto_sha256',repeat('7',64),
  'huella_manifiesto_json_sha256',encode(sha256(pre_manifiesto),'hex'),
  'version_esperada',1,'huella_anterior_sha256',encode(sha256(convert_to(j1::text,'UTF8')),'hex'),
  'version_siguiente',2,'huella_siguiente_sha256',encode(sha256(pre_carga),'hex'));
 pre:=convert_to(m::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,m->>'accion',carga,hecho,'decision:d17:3');
 z:=vec_documentos.confirmar_preparacion_carga_directa_v1(pre,pre_carga,pre_manifiesto,
  mat.auth,mat.capacidad,mat.decision,'\x01'::bytea,'\x01'::bytea,1,1,
  '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF z->>'estado_operativo' IS DISTINCT FROM 'emision_recibo_iniciada'
 THEN RAISE EXCEPTION 'D17: TX2 no fijó manifiesto'; END IF;
 recibo:=jsonb_build_object('indice_hmac','hmac-sha256:indice_v1:'||repeat('8',64),
  'grupo_hmac','hmac-sha256:grupo_v1:'||repeat('9',64),
  'vinculo_hmac','hmac-sha256:vinculo_v1:'||repeat('a',64),
  'evidencia_alta_ref','evidencia:alta:uno','autorizacion_emision_ref','decision:preparar:v1',
  'expira_en',expira);
 m:=jsonb_build_object('accion','documentos.carga.preparar_respuesta','carga_id',carga,
  'recurso_ref',hecho,'version_carga',2,'persona_ref',persona,
  'registro_contexto_ref','rca_'||repeat('b',32),'huella_contexto_sha256',repeat('a',64),
  'huella_manifiesto_sha256',repeat('7',64),'indice_hmac',recibo->>'indice_hmac',
  'grupo_hmac',recibo->>'grupo_hmac','vinculo_hmac',recibo->>'vinculo_hmac',
  'evidencia_alta_ref',recibo->>'evidencia_alta_ref','expira_en',expira);
 pre:=convert_to(m::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,m->>'accion',carga,hecho,'decision:d17:4');
 z:=vec_documentos.registrar_recibo_carga_directa_v1(pre,convert_to(recibo::text,'UTF8'),mat.auth,mat.capacidad,
  mat.decision,'\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF z->>'estado_operativo' IS DISTINCT FROM 'recibo_registrado'
  OR z->>'entrega_acreditada' IS DISTINCT FROM 'false'
 THEN RAISE EXCEPTION 'D17: TX3 confundió registro con entrega'; END IF;
 RAISE NOTICE 'D17_TX3_JSON=%',z;
 orden:=jsonb_build_object('indice_hmac',recibo->>'indice_hmac',
  'grupo_hmac',recibo->>'grupo_hmac','vinculo_hmac',recibo->>'vinculo_hmac',
  'evidencia_consumo_ref','evidencia:consumo:uno',
  'intencion_confirmacion_ref','intencion:confirmacion:uno',
  'huella_intencion_hmac','hmac-sha256:intencion_v1:'||repeat('b',64),
  'registrado_en',z->>'registrado_en','valida_hasta',expira);
 m:=jsonb_build_object('accion','documentos.carga.confirmar_intencion','carga_id',carga,
  'recurso_ref',hecho,'persona_ref',persona,'version_carga',2,
  'registro_contexto_ref','rca_'||repeat('b',32),'huella_contexto_sha256',repeat('a',64),
  'indice_hmac',orden->>'indice_hmac','grupo_hmac',orden->>'grupo_hmac',
  'vinculo_hmac',orden->>'vinculo_hmac','evidencia_consumo_ref',orden->>'evidencia_consumo_ref',
  'intencion_ref',orden->>'intencion_confirmacion_ref',
  'huella_intencion_hmac',orden->>'huella_intencion_hmac',
  'registrado_en',orden->>'registrado_en','valida_hasta',expira);
 pre:=convert_to(m::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,m->>'accion',carga,hecho,'decision:d17:5');
 z:=vec_documentos.consumir_recibo_carga_directa_v3_v1(pre,convert_to(orden::text,'UTF8'),
  mat.auth,mat.capacidad,mat.decision,'\x01'::bytea,'\x01'::bytea,1,1,
  '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF z->>'estado_operativo' IS DISTINCT FROM 'confirmacion_iniciada'
  OR z->>'intencion_confirmacion_ref' IS DISTINCT FROM 'intencion:confirmacion:uno'
 THEN RAISE EXCEPTION 'D17: TX4 no fijó intención antes de S3'; END IF;
 BEGIN
  PERFORM vec_documentos.consumir_recibo_carga_directa_v3_v1(pre,convert_to(orden::text,'UTF8'),
   mat.auth,mat.capacidad,mat.decision,'\x01'::bytea,'\x01'::bytea,1,1,
   '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
  RAISE EXCEPTION 'D17: segundo consumo aceptado';
 EXCEPTION WHEN unique_violation THEN NULL; END;
 objeto:=jsonb_build_object('conector_id','almacen_s3_corporativo',
  'referencia','objeto:cuarentena:uno','version','v1','zona','cuarentena',
  'mime','application/pdf','tamano',100,'huella_sha256',repeat('d',64),
  'evidencia_ref','evidencia:s3:uno','registrado_en',clock_timestamp());
 j3:=j2||jsonb_build_object('version',3,'estado','cuarentena',
  'autorizacion_recepcion_ref','decision:v1:confirm','contenido_cuarentena',objeto);
 pre_carga:=convert_to(j3::text,'UTF8');
 m:=jsonb_build_object('accion','documentos.carga.confirmar_resultado','carga_id',carga,
  'recurso_ref',hecho,'persona_ref',persona,'version_esperada',2,
  'huella_anterior_sha256',encode(sha256(convert_to(j2::text,'UTF8')),'hex'),
  'version_siguiente',3,'huella_siguiente_sha256',encode(sha256(pre_carga),'hex'),
  'autorizacion_v1_ref','decision:v1:confirm','evento','vec.documentos.carga.recibida');
 pre:=convert_to(m::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,m->>'accion',carga,hecho,'decision:d17:6');
 z:=vec_documentos.confirmar_resultado_carga_directa_v3_v1(pre,pre_carga,mat.auth,
  mat.capacidad,mat.decision,'\x01'::bytea,'\x01'::bytea,1,1,
  '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF z->>'estado_operativo' IS DISTINCT FROM 'cuarentena' OR z#>>'{carga,version}' IS DISTINCT FROM '3'
 THEN RAISE EXCEPTION 'D17: resultado S3 no conservado'; END IF;
 RAISE NOTICE 'D17_RESULTADO_JSON=%',z;
 analisis:=jsonb_build_object('objeto_referencia',objeto->>'referencia',
  'objeto_version',objeto->>'version','huella_objeto_sha256',repeat('d',64),
  'conector_analizador_id','antivirus:desarrollo','version_conector',1,
  'estado','limpio','codigo_resultado','limpio','evidencia_ref','evidencia:antivirus:uno',
  'huella_evidencia_sha256',repeat('e',64),'completado_en',clock_timestamp());
 j4:=j3||jsonb_build_object('version',4,'estado','analizada_limpia',
  'autorizacion_analisis_ref','decision:v1:analizar','analisis',analisis);
 pre_carga:=convert_to(j4::text,'UTF8');
 m:=jsonb_build_object('accion','vec.documentos.carga.analizar','carga_id',carga,
  'recurso_ref',hecho,'persona_ref',persona,'version_esperada',3,
  'huella_anterior_sha256',encode(sha256(convert_to(j3::text,'UTF8')),'hex'),
  'version_siguiente',4,'huella_siguiente_sha256',encode(sha256(pre_carga),'hex'),
  'autorizacion_v1_ref','decision:v1:analizar','evento','vec.documentos.carga.analizada');
 pre:=convert_to(m::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,m->>'accion',carga,hecho,'decision:d17:7');
 z:=vec_documentos.registrar_analisis_carga_directa_v3_v1(pre,pre_carga,mat.auth,
  mat.capacidad,mat.decision,'\x01'::bytea,'\x01'::bytea,1,1,
  '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF z->>'estado_operativo' IS DISTINCT FROM 'analizada_limpia' OR z#>>'{carga,version}' IS DISTINCT FROM '4'
 THEN RAISE EXCEPTION 'D17: análisis limpio no conservado'; END IF;
 m:=jsonb_build_object('accion','documentos.carga.promover_intencion','carga_id',carga,
  'recurso_ref',hecho,'persona_ref',persona,'version_carga',4,
  'huella_carga_json_sha256',encode(sha256(convert_to(j4::text,'UTF8')),'hex'),
  'indice_promocion_hmac','hmac-sha256:promocion_v1:'||repeat('f',64),
  'origen_ref',objeto->>'referencia','origen_version',objeto->>'version',
  'huella_origen_sha256',repeat('d',64),'evidencia_analisis_ref',analisis->>'evidencia_ref',
  'registro_contexto_ref','rca_'||repeat('b',32),'huella_contexto_sha256',repeat('a',64));
 pre:=convert_to(m::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,m->>'accion',carga,hecho,'decision:d17:8');
 z:=vec_documentos.iniciar_promocion_carga_directa_v3_v1(pre,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF z->>'estado_operativo' IS DISTINCT FROM 'promocion_iniciada'
 THEN RAISE EXCEPTION 'D17: intención promoción no durable'; END IF;
 BEGIN
  PERFORM vec_documentos.iniciar_promocion_carga_directa_v3_v1(pre,mat.auth,mat.capacidad,mat.decision,
   '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
  RAISE EXCEPTION 'D17: segunda promoción permitida';
 EXCEPTION WHEN unique_violation THEN NULL; END;
 admitido:=objeto||jsonb_build_object('referencia','objeto:admitido:uno',
  'version','v2','zona','admitida','evidencia_ref','evidencia:s3:promocion');
 j5:=j4||jsonb_build_object('version',5,'estado','admitida',
  'autorizacion_promocion_ref','decision:v1:promover','contenido_admitido',admitido);
 pre_carga:=convert_to(j5::text,'UTF8');
 m:=jsonb_build_object('accion','documentos.carga.promover_resultado','carga_id',carga,
  'recurso_ref',hecho,'persona_ref',persona,'version_esperada',4,
  'huella_anterior_sha256',encode(sha256(convert_to(j4::text,'UTF8')),'hex'),
  'version_siguiente',5,'huella_siguiente_sha256',encode(sha256(pre_carga),'hex'),
  'autorizacion_v1_ref','decision:v1:promover','evento','vec.documentos.carga.promovida',
  'indice_promocion_hmac','hmac-sha256:promocion_v1:'||repeat('f',64));
 pre:=convert_to(m::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,m->>'accion',carga,hecho,'decision:d17:9');
 z:=vec_documentos.confirmar_promocion_carga_directa_v3_v1(pre,pre_carga,mat.auth,
  mat.capacidad,mat.decision,'\x01'::bytea,'\x01'::bytea,1,1,
  '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF z->>'estado_operativo' IS DISTINCT FROM 'admitida' OR z#>>'{carga,version}' IS DISTINCT FROM '5'
 THEN RAISE EXCEPTION 'D17: resultado promoción no conservado'; END IF;
END $probar$;
RESET SESSION AUTHORIZATION;
DO $historia$
DECLARE n bigint; m bigint;
BEGIN
 PERFORM set_config('vec.documentos.expediente_ref','ref:'||repeat('1',64),true);
 SELECT count(*) INTO n FROM vec_documentos.carga_directa_operacion;
 SELECT count(*) INTO m FROM vec_documentos.carga_directa_historia;
 IF n<>1 OR m<>9 THEN RAISE EXCEPTION 'D17: duplicado tras replay %/%',n,m; END IF;
END $historia$;
ROLLBACK;
