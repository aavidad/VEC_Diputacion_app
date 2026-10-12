\set ON_ERROR_STOP on
-- Ejecutar solo en clon E después del COMMIT sintético y reinicio PG18.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET SESSION AUTHORIZATION vec_documentos_d17_ensayo;
DO $probar$
DECLARE carga text:='ref:'||encode(sha256(convert_to('vec:carga-documental:id:v3','UTF8')||decode('00','hex')||
 convert_to('hmac-sha256:indice_v1:'||repeat('a',64),'UTF8')),'hex'); hecho text:='ref:'||repeat('1',64);
 persona text:='per_'||repeat('a',32); m jsonb; pre bytea; mat record; r jsonb;
BEGIN
 m:=jsonb_build_object('accion','documentos.carga.consultar_propia','carga_id',carga,
  'recurso_ref',hecho,'persona_ref',persona,
  'perfil_activo_ref','perfil:00000000-0000-4000-8000-000000000001',
  'registro_contexto_ref','rca_'||repeat('b',32),'huella_contexto_sha256',repeat('a',64),
  'finalidad','custodiar_evidencia_meritos',
  'correlacion_ref','corr:00000000-0000-4000-8000-000000000001',
  'incluye_manifiesto',true);
 pre:=convert_to(m::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,m->>'accion',carga,hecho,'decision:d17:consulta:1');
 r:=vec_documentos.consultar_carga_directa_propia_v1(pre,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF r#>>'{carga,id}' IS DISTINCT FROM carga OR r->>'estado_operativo' IS DISTINCT FROM 'admitida'
  OR r#>>'{carga,version}' IS DISTINCT FROM '5' OR r#>>'{carga_preparada,version}' IS DISTINCT FROM '2'
  OR r#>>'{carga_preparada,id}' IS DISTINCT FROM carga OR
  r#>>'{manifiesto,huella_carga_sha256}' IS DISTINCT FROM
   encode(sha256(convert_to((r->'carga_preparada')::text,'UTF8')),'hex')
  OR r->>'entrega_acreditada' IS DISTINCT FROM 'false' OR r->'manifiesto' IS NULL
 THEN RAISE EXCEPTION 'D17: recuperación tras reinicio incorrecta'; END IF;
 m:=m||jsonb_build_object('recurso_ref','ref:'||repeat('4',64));
 pre:=convert_to(m::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_doc17.material(pre,m->>'accion',carga,m->>'recurso_ref','decision:d17:consulta:2');
 r:=vec_documentos.consultar_carga_directa_propia_v1(pre,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF r IS DISTINCT FROM '{"denegado": true}'::jsonb THEN RAISE EXCEPTION 'D17: recurso ajeno visible'; END IF;
 RAISE NOTICE 'D17_REINICIO_OK=%',carga;
END $probar$;
ROLLBACK;
