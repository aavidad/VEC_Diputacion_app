\set ON_ERROR_STOP on
-- Solo clon PG18 desechable DESPUÉS del fixture D17 en COMMIT y D18 instalado.
-- AD3 está sustituida por doble del fixture; no prueba COSE ni permiso PDP real.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL vec.documentos.expediente_ref='ref:1111111111111111111111111111111111111111111111111111111111111111';
SET LOCAL vec.documentos.persona_ref='per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';
INSERT INTO vec_documentos.plan_ingesta_mayan(
 plan_ref,carga_id,persona_ref,expediente_ref,evidencia_ref,tipo_ref,principal_ref,
 clave_reserva,clave_idempotencia,registro_contexto_ref,huella_contexto_sha256,
 huella_solicitud_sha256,huella_sha256,mime,tamano,carga_version,politica_ref,
 version_politica,huella_politica_sha256,conservacion_hasta,estado_politica,
 proteccion,version,estado,decision_preparacion_ref,creada_en,actualizada_en)
VALUES('plan:123e4567-e89b-42d3-a456-426614174018',
 'ref:c90e3b0990907c655e20fac278eb6fc274605a047e368306fbd778f519e53e57',
 'per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
 'ref:1111111111111111111111111111111111111111111111111111111111111111',
 'ref:'||repeat('f',64),'ref:'||repeat('7',64),
 'per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','ref:'||repeat('2',64),
 'ref:'||repeat('5',64),'rca_'||repeat('b',32),repeat('a',64),repeat('c',64),
 repeat('d',64),'application/pdf',100,5,'ref:'||repeat('4',64),1,repeat('e',64),
 '2035-01-01 UTC','aprobada','conservacion',1,'preparado','decision:prueba',
 clock_timestamp(),clock_timestamp());
RESET ROLE;
SET SESSION AUTHORIZATION vec_documentos_d17_ensayo;
DO $prueba$
DECLARE carga text:='ref:c90e3b0990907c655e20fac278eb6fc274605a047e368306fbd778f519e53e57';
 hecho text:='ref:'||repeat('1',64); persona text:='per_'||repeat('a',32);
 plan text:='plan:123e4567-e89b-42d3-a456-426614174018';
 m jsonb; pre bytea; mat record; z jsonb; h text; abierta jsonb;
BEGIN
 m:=jsonb_build_object('accion','documentos.carga_admitida.consultar_para_custodia',
  'fase','consulta','carga_id',carga,'recurso_ref',hecho,'persona_ref',persona,
  'plan_ref',plan,'version_esperada',5,'huella_carga_esperada_sha256','',
  'objeto_ref','','objeto_version','','huella_contenido_sha256','',
  'tamano',0,'mime','','evidencia_analisis_ref','',
  'perfil_activo_ref','perfil:00000000-0000-4000-8000-000000000001',
  'registro_contexto_ref','rca_'||repeat('b',32),'huella_contexto_sha256',repeat('a',64),
  'finalidad','custodiar_evidencia_meritos',
  'correlacion_ref','corr:00000000-0000-4000-8000-000000000001');
 pre:=convert_to(m::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,m->>'accion',carga,hecho,'decision:d17:custodia:consulta');
 z:=vec_documentos.consultar_carga_admitida_custodia_v1(pre,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF z#>>'{carga,id}' IS DISTINCT FROM carga OR z#>>'{carga,version}' IS DISTINCT FROM '5'
  OR z#>>'{carga,analisis,estado}' IS DISTINCT FROM 'limpio'
 THEN RAISE EXCEPTION 'D17: consulta admitida no autorizada'; END IF;
 h:=encode(sha256(convert_to((z->'carga')::text,'UTF8')),'hex');
 m:=m||jsonb_build_object('fase','abrir','huella_carga_esperada_sha256',h,
  'objeto_ref',z#>>'{carga,contenido_admitido,referencia}',
  'objeto_version',z#>>'{carga,contenido_admitido,version}',
  'huella_contenido_sha256',z#>>'{carga,contenido_admitido,huella_sha256}',
  'tamano',(z#>>'{carga,contenido_admitido,tamano}')::bigint,
  'mime',z#>>'{carga,contenido_admitido,mime}',
  'evidencia_analisis_ref',z#>>'{carga,analisis,evidencia_ref}');
 pre:=convert_to(m::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,m->>'accion',carga,hecho,'decision:d17:custodia:abrir');
 abierta:=vec_documentos.consultar_carga_admitida_custodia_v1(pre,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF abierta#>>'{carga,id}' IS DISTINCT FROM carga OR abierta->'carga' IS DISTINCT FROM z->'carga'
 THEN RAISE EXCEPTION 'D17: fase abrir no conservó misma carga'; END IF;
 BEGIN
  PERFORM vec_documentos.consultar_carga_admitida_custodia_v1(pre,mat.auth,
   NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea,1,1,
   NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea);
  RAISE EXCEPTION 'D17: lectura sin V3 aceptada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;

 m:=m||jsonb_build_object('persona_ref','per_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb');
 pre:=convert_to(m::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,m->>'accion',carga,hecho,'decision:d17:custodia:persona-ajena');
 BEGIN
  PERFORM vec_documentos.consultar_carga_admitida_custodia_v1(pre,mat.auth,mat.capacidad,mat.decision,
   '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
  RAISE EXCEPTION 'D17: persona ajena visible';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 m:=m||jsonb_build_object('persona_ref',persona,'objeto_ref','objeto:ajeno');
 pre:=convert_to(m::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,m->>'accion',carga,hecho,'decision:d17:custodia:objeto-ajeno');
 abierta:=vec_documentos.consultar_carga_admitida_custodia_v1(pre,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF abierta IS DISTINCT FROM '{"denegado": true}'::jsonb THEN RAISE EXCEPTION 'D17: objeto ajeno visible'; END IF;
 m:=m||jsonb_build_object('objeto_ref',z#>>'{carga,contenido_admitido,referencia}',
  'huella_contenido_sha256',repeat('a',64));
 pre:=convert_to(m::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,m->>'accion',carga,hecho,'decision:d17:custodia:sha-ajeno');
 abierta:=vec_documentos.consultar_carga_admitida_custodia_v1(pre,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF abierta IS DISTINCT FROM '{"denegado": true}'::jsonb THEN RAISE EXCEPTION 'D17: SHA ajeno visible'; END IF;

 m:=m||jsonb_build_object('huella_contenido_sha256',z#>>'{carga,contenido_admitido,huella_sha256}',
  'objeto_version','version:ajena');
 pre:=convert_to(m::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,m->>'accion',carga,hecho,'decision:d17:custodia:version-ajena');
 abierta:=vec_documentos.consultar_carga_admitida_custodia_v1(pre,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF abierta IS DISTINCT FROM '{"denegado": true}'::jsonb THEN RAISE EXCEPTION 'D17: versión ajena visible'; END IF;
 m:=m||jsonb_build_object('objeto_version',z#>>'{carga,contenido_admitido,version}',
  'plan_ref','plan:223e4567-e89b-42d3-a456-426614174018');
 pre:=convert_to(m::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,m->>'accion',carga,hecho,'decision:d17:custodia:plan-ajeno');
 BEGIN
  PERFORM vec_documentos.consultar_carga_admitida_custodia_v1(pre,mat.auth,mat.capacidad,mat.decision,
   '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
  RAISE EXCEPTION 'D17: plan ajeno abrió';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 m:=m||jsonb_build_object('plan_ref',plan,'huella_carga_esperada_sha256',repeat('f',64));
 pre:=convert_to(m::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,m->>'accion',carga,hecho,'decision:d17:custodia:snapshot-ajeno');
 abierta:=vec_documentos.consultar_carga_admitida_custodia_v1(pre,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF abierta IS DISTINCT FROM '{"denegado": true}'::jsonb THEN RAISE EXCEPTION 'D17: snapshot ajeno visible'; END IF;
 RAISE NOTICE 'D17_CUSTODIA_V5_OK=%',carga;
END $prueba$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
