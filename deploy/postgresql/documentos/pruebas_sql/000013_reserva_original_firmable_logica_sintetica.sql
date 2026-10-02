\set ON_ERROR_STOP on
-- SOLO clon desechable PG18, después de Documentos 13. Sustituye temporalmente
-- el consumidor AD3 por material sintético para probar lógica SQL de reserva,
-- reintento, confirmación y conflicto. ROLLBACK restaura la función real.
-- No demuestra firma COSE, concesión V3 real ni instalación causal AD158.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog, pg_temp AS $f$
DECLARE c jsonb:=convert_from(p_capacidad,'UTF8')::jsonb; d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 RETURN QUERY SELECT d->>'decision_ref',c->>'efecto_ref',c->>'huella_efecto_sha256',
  repeat('b',64),'audit:sintetico',clock_timestamp(),NOT coalesce((c->>'replay')::boolean,false);
END $f$;
RESET ROLE;

CREATE ROLE vec_documentos_original_ensayo LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT;
GRANT vec_documentos_ejecutor TO vec_documentos_original_ensayo WITH INHERIT TRUE, SET FALSE;
CREATE SCHEMA ensayo_original_13 AUTHORIZATION postgres;
GRANT USAGE ON SCHEMA ensayo_original_13 TO vec_documentos_original_ensayo;
CREATE FUNCTION ensayo_original_13.material(p_pre bytea,p_accion text,p_id text,p_decision text,p_replay boolean DEFAULT false)
RETURNS TABLE(capacidad bytea,decision bytea,auth jsonb)
LANGUAGE plpgsql SET search_path=pg_catalog, pg_temp AS $f$
DECLARE h text:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"preimagen_sha256":"'||encode(sha256(p_pre),'hex')||'"}}','UTF8')),'hex');
BEGIN
 capacidad:=convert_to(jsonb_build_object('audiencia_consumo','vec_documentos.operacion.v1',
  'operacion',p_accion,'efecto_ref',p_id,'huella_efecto_sha256',h,'replay',p_replay)::text,'UTF8');
 decision:=convert_to(jsonb_build_object('decision_ref',p_decision,'accion',p_accion,
  'modulo_id','documentos','recurso_ref',p_id,'contexto_recurso_huella_sha256',h,
  'principal_id','per_'||repeat('a',32),
  'perfil_activo_ref','perfil:00000000-0000-4000-8000-000000000001',
  'finalidad','custodiar_original_firmable',
  'correlacion_ref','corr:00000000-0000-4000-8000-000000000001',
  'obligaciones','[]'::jsonb)::text,'UTF8');
 auth:=jsonb_build_object('accion',p_accion,'recurso_ref',p_id,
  'ambito_ref','exp:00000000-0000-4000-8000-000000000013',
  'principal_id','per_'||repeat('a',32),
  'perfil_activo_ref','perfil:00000000-0000-4000-8000-000000000001',
  'finalidad','custodiar_original_firmable',
  'correlacion_ref','corr:00000000-0000-4000-8000-000000000001');
 RETURN NEXT;
END $f$;
GRANT EXECUTE ON FUNCTION ensayo_original_13.material(bytea,text,text,text,boolean) TO vec_documentos_original_ensayo;

SET SESSION AUTHORIZATION vec_documentos_original_ensayo;
DO $probar$
<<prueba>>
DECLARE
 id text:='ref:'||repeat('1',64); tipo text:='ref:6923120e1e4fac09364caf85f99ea0761a9f21c638485e6032e6335b3227a19c';
 a text:='documentos.original_firmable.reservar'; b text:='documentos.original_firmable.confirmar';
 p1 bytea; p2 bytea; pc bytea; p_conflicto bytea; obj jsonb; mat record;
 r1 jsonb; r2 jsonb; r3 jsonb; d1 jsonb; d2 jsonb;
BEGIN
 p1:=convert_to(jsonb_build_object('accion',a,'id',id,
  'clave_idempotencia','idem:00000000-0000-4000-8000-000000000013',
  'modulo_id','contratacion_temporal','expediente_ref','exp:00000000-0000-4000-8000-000000000013',
  'tipo_ref',tipo,'version',1,'mime','application/pdf','tamano',3,'huella_sha256',repeat('a',64),
  'politica_ref','pol:00000000-0000-4000-8000-000000000013','version_politica',2,
  'huella_politica_sha256',repeat('c',64),'conservacion_hasta','2036-01-01T00:00:00Z',
  'proteccion','conservacion','estado_politica','provisional')::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_original_13.material(p1,a,id,'dec:00000000-0000-4000-8000-000000000001',false);
 r1:=vec_documentos.reservar_original_firmable_v1(p1,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF r1->>'estado'<>'pendiente' OR r1->>'intento_num'<>'1' THEN RAISE EXCEPTION 'reserva inicial inválida'; END IF;
 SELECT * INTO STRICT mat FROM ensayo_original_13.material(p1,a,id,'dec:00000000-0000-4000-8000-000000000001',true);
 r2:=vec_documentos.reservar_original_firmable_v1(p1,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF r2 IS DISTINCT FROM r1 THEN RAISE EXCEPTION 'replay de reserva alteró intento'; END IF;
 p2:=convert_to((convert_from(p1,'UTF8')::jsonb||jsonb_build_object('conservacion_hasta','2037-01-01T00:00:00Z'))::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_original_13.material(p2,a,id,'dec:00000000-0000-4000-8000-000000000002',false);
 r2:=vec_documentos.reservar_original_firmable_v1(p2,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF r2->>'intento_num'<>'2' OR r2->>'clave_almacen_ref'=r1->>'clave_almacen_ref'
 THEN RAISE EXCEPTION 'reintento no renovó clave'; END IF;
 SELECT * INTO STRICT mat FROM ensayo_original_13.material(p2,a,id,'dec:00000000-0000-4000-8000-000000000002',true);
 r3:=vec_documentos.reservar_original_firmable_v1(p2,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF r3 IS DISTINCT FROM r2 THEN RAISE EXCEPTION 'replay nuevo no recuperó intento'; END IF;
 -- El intento anterior queda sustituido y un cambio de bytes bajo la misma
 -- identidad natural no puede crear otro original.
 BEGIN
  SELECT * INTO STRICT mat FROM ensayo_original_13.material(p1,a,id,'dec:00000000-0000-4000-8000-000000000001',true);
  PERFORM vec_documentos.reservar_original_firmable_v1(p1,mat.auth,mat.capacidad,mat.decision,
   '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
  RAISE EXCEPTION 'intento sustituido aceptado';
 EXCEPTION WHEN unique_violation THEN NULL; END;
 p_conflicto:=convert_to((convert_from(p2,'UTF8')::jsonb||jsonb_build_object('huella_sha256',repeat('e',64)))::text,'UTF8');
 BEGIN
  SELECT * INTO STRICT mat FROM ensayo_original_13.material(p_conflicto,a,id,'dec:00000000-0000-4000-8000-000000000004',false);
  PERFORM vec_documentos.reservar_original_firmable_v1(p_conflicto,mat.auth,mat.capacidad,mat.decision,
   '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
  RAISE EXCEPTION 'contenido divergente aceptado';
 EXCEPTION WHEN unique_violation THEN NULL; END;
 obj:=jsonb_build_object('clave_almacen_ref',r2->>'clave_almacen_ref',
  'objeto_ref','obj:00000000-0000-4000-8000-000000000013','objeto_version','ov1',
  'conector_ref','ficheros_ensayo','recibo_objeto_ref','recibo:00000000-0000-4000-8000-000000000013',
  'recibo_objeto_huella_sha256',repeat('d',64),'retenido_hasta',NULL,'inmovilizado',false,
  'mime','application/pdf','tamano',3,'huella_sha256',repeat('a',64));
 pc:=convert_to(jsonb_build_object('accion',b,'reserva_ref',r2->>'reserva_ref',
  'intento_num',2,'clave_almacen_ref',r2->>'clave_almacen_ref','id',id,
  'huella_sha256',repeat('a',64),'objeto',obj)::text,'UTF8');
 SELECT * INTO STRICT mat FROM ensayo_original_13.material(pc,b,id,'dec:00000000-0000-4000-8000-000000000003',false);
 d1:=vec_documentos.confirmar_original_firmable_v1(pc,obj,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF d1->>'id'<>id OR d1->>'estado'<>'confirmado' THEN RAISE EXCEPTION 'confirmación inválida'; END IF;
 SELECT * INTO STRICT mat FROM ensayo_original_13.material(pc,b,id,'dec:00000000-0000-4000-8000-000000000003',true);
 d2:=vec_documentos.confirmar_original_firmable_v1(pc,obj,mat.auth,mat.capacidad,mat.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 IF d2 IS DISTINCT FROM d1 THEN RAISE EXCEPTION 'replay de confirmación alteró documento'; END IF;
END $probar$;
RESET SESSION AUTHORIZATION;
SET CONSTRAINTS ALL IMMEDIATE;
-- El tipo es global: cambiar módulo no debe abrir el alta genérica.
DO $tipo_ajeno$
DECLARE d vec_documentos.documento%ROWTYPE;
BEGIN
 SELECT * INTO STRICT d FROM vec_documentos.documento
  WHERE id='ref:'||repeat('1',64);
 BEGIN
  INSERT INTO vec_documentos.documento
  SELECT (jsonb_populate_record(NULL::vec_documentos.documento,
    to_jsonb(d)||jsonb_build_object(
      'id','ref:'||repeat('2',64),
      'numero_vec','VEC-2099-999999999999',
      'clave_idempotencia','idem:00000000-0000-4000-8000-000000000014',
      'modulo_id','bolsa','version',2))).*;
  RAISE EXCEPTION 'tipo original admitido en módulo ajeno';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $tipo_ajeno$;
DO $conteos$
DECLARE original_id text:='ref:'||repeat('1',64); reserva_id text;
BEGIN
 SELECT reserva_ref INTO STRICT reserva_id FROM vec_documentos.reserva_original_firmable
  WHERE documento_id=original_id;
 IF (SELECT count(*) FROM vec_documentos.documento d WHERE d.id=original_id)<>1
    OR (SELECT count(*) FROM vec_documentos.confirmacion_original_firmable WHERE documento_id=original_id)<>1
    OR (SELECT count(*) FROM vec_documentos.intento_original_firmable WHERE reserva_ref=reserva_id)<>2
    OR (SELECT count(*) FROM vec_documentos.outbox WHERE recurso_ref=original_id)<>1
 THEN RAISE EXCEPTION 'duplicación documental'; END IF;
END $conteos$;
ROLLBACK;
