\set ON_ERROR_STOP on
-- Superficie EXCLUSIVA del contenedor efímero de probar_integracion_pg18.sh.
-- Requiere custodia_externa_sintetica.sql (ensayo_externa.preparar y las
-- fachadas AD3 sintéticas) y Documentos 000009 sobre AD3-113. Ejercita la
-- custodia de documentos firmados en un expediente propio (exp ...00f1). No
-- es una prueba COSE ni se instala en otra base.
BEGIN;
CREATE SCHEMA ensayo_firmado AUTHORIZATION postgres;
GRANT USAGE ON SCHEMA ensayo_firmado TO vec_documentos_ensayo;

-- tipo_ref es el reservado de la resolución firmada de Contratación temporal.
CREATE FUNCTION ensayo_firmado.preimagen(p_id text,p_clave text,p_tipo text,p_original text,p_firma text) RETURNS bytea
LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT convert_to('{"accion":"documentos.firmado.custodiar","id":"'||p_id||'","clave_idempotencia":"'||p_clave||'","modulo_id":"contratacion_temporal","expediente_ref":"exp:00000000-0000-4000-8000-0000000000f1","tipo_ref":"'||p_tipo||'","version":1,"mime":"application/pdf","tamano":3,"huella_sha256":"'||repeat('a',64)||'","huella_original_sha256":"'||p_original||'","firma_operacion_ref":"'||p_firma||'","politica_ref":"pol:00000000-0000-4000-8000-000000000001","version_politica":1,"huella_politica_sha256":"'||repeat('c',64)||'","proteccion":"conservacion","conservacion_hasta":"2036-01-01T00:00:00Z","estado_politica":"provisional"}','UTF8')
$f$;

CREATE FUNCTION ensayo_firmado.objeto(p_objeto text) RETURNS jsonb
LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT jsonb_build_object('objeto_ref',p_objeto,'objeto_version','ov1','conector_ref','ficheros_ensayo',
  'recibo_objeto_ref',replace(p_objeto,'obj:','recibo:'),'recibo_objeto_huella_sha256',repeat('d',64),
  'retenido_hasta',NULL,'inmovilizado',false,'mime','application/pdf','tamano',3,'huella_sha256',repeat('a',64))
$f$;

CREATE FUNCTION ensayo_firmado.custodiar(p_caso text,p_preimagen bytea,p_objeto jsonb,p_decision text,
 p_accion text DEFAULT 'documentos.firmado.custodiar',p_finalidad text DEFAULT 'custodiar_documento_firmado') RETURNS jsonb
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE m ensayo_externa.material; r jsonb; id text:=convert_from(p_preimagen,'UTF8')::jsonb->>'id';
BEGIN
 PERFORM ensayo_externa.preparar(p_caso,p_accion,id,'exp:00000000-0000-4000-8000-0000000000f1',
  p_finalidad,CASE WHEN p_accion='documentos.firmado.custodiar' THEN 'documento_firmado' ELSE 'documento_generado' END,
  CASE WHEN p_accion='documentos.firmado.custodiar' THEN '["documento_firmado.custodia","evidencia_custodia"]'::jsonb ELSE '["documento","recibo"]'::jsonb END,
  p_preimagen,p_decision);
 SELECT * INTO STRICT m FROM ensayo_externa.material WHERE caso=p_caso;
 SELECT vec_documentos.custodiar_firmado_v1(m.preimagen,p_objeto,m.auth,m.capacidad,m.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea) INTO r;
 RETURN r;
END $f$;

CREATE FUNCTION ensayo_firmado.denegado(p_sql text,p_estado text,p_mensaje text DEFAULT NULL) RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 BEGIN
  EXECUTE p_sql;
 EXCEPTION WHEN others THEN
  IF SQLSTATE<>p_estado OR (p_mensaje IS NOT NULL AND strpos(SQLERRM,p_mensaje)=0) THEN
   RAISE EXCEPTION 'FALLO: % dio % (%) y no %',p_sql,SQLSTATE,SQLERRM,p_estado;
  END IF;
  RETURN;
 END;
 RAISE EXCEPTION 'FALLO: se aceptó %',p_sql;
END $f$;

CREATE FUNCTION ensayo_firmado.probar() RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE r jsonb; r2 jsonb;
 reservado text:='ref:f0074a505ef2a693dfd5bf2195228c47a7ce2a1435d4acf6a6b9f50e89e5663c';
 id text:='doc:00000000-0000-4000-8000-0000000000f1'; clave text:='idem:00000000-0000-4000-8000-0000000000f1';
 firma text:='firmact:00000000-0000-4000-8000-0000000000f1'; original text:=repeat('b',64);
BEGIN
 -- 1. Custodia correcta: custodia VEC, firma de prueba (proveedor pendiente).
 r:=ensayo_firmado.custodiar('f1',ensayo_firmado.preimagen(id,clave,reservado,original,firma),
  ensayo_firmado.objeto('obj:00000000-0000-4000-8000-0000000000f1'),'decision:00000000-0000-4000-8000-0000000009f1');
 IF r->>'id'<>id OR r->>'custodia'<>'vec' OR r->>'estado_firma'<>'pendiente_proveedor'
    OR r->>'firma_operacion_ref'<>firma OR r->>'huella_original_sha256'<>original OR r->>'huella_sha256'<>repeat('a',64)
 THEN RAISE EXCEPTION 'FALLO: custodia inesperada %',r; END IF;
 -- 2. Recuperación con concesión nueva: mismo documento, sin duplicar.
 r2:=ensayo_firmado.custodiar('f1b',ensayo_firmado.preimagen(id,clave,reservado,original,firma),
  ensayo_firmado.objeto('obj:00000000-0000-4000-8000-0000000000f1'),'decision:00000000-0000-4000-8000-0000000009f2');
 IF r2->>'id'<>id OR r2->>'numero_vec'<>r->>'numero_vec' THEN RAISE EXCEPTION 'FALLO: recuperación distinta %',r2; END IF;
 -- 3. Misma clave con otro objeto: conflicto.
 PERFORM ensayo_firmado.denegado(format($s$SELECT ensayo_firmado.custodiar('f3',ensayo_firmado.preimagen(%L,%L,%L,%L,%L),ensayo_firmado.objeto('obj:00000000-0000-4000-8000-0000000000f3'),'decision:00000000-0000-4000-8000-0000000009f3')$s$,
  id,clave,reservado,original,firma),'23505');
 -- 4. Tipo no reservado, huellas iguales, material de otra acción, otra finalidad: denegados.
 PERFORM ensayo_firmado.denegado(format($s$SELECT ensayo_firmado.custodiar('f4',ensayo_firmado.preimagen('doc:00000000-0000-4000-8000-0000000000f4','idem:00000000-0000-4000-8000-0000000000f4','tipo:00000000-0000-4000-8000-000000000001',%L,'firmact:00000000-0000-4000-8000-0000000000f4'),ensayo_firmado.objeto('obj:00000000-0000-4000-8000-0000000000f4'),'decision:00000000-0000-4000-8000-0000000009f4')$s$,original),'42501');
 PERFORM ensayo_firmado.denegado(format($s$SELECT ensayo_firmado.custodiar('f5',ensayo_firmado.preimagen('doc:00000000-0000-4000-8000-0000000000f5','idem:00000000-0000-4000-8000-0000000000f5',%L,%L,'firmact:00000000-0000-4000-8000-0000000000f5'),ensayo_firmado.objeto('obj:00000000-0000-4000-8000-0000000000f5'),'decision:00000000-0000-4000-8000-0000000009f5')$s$,reservado,repeat('a',64)),'42501');
 PERFORM ensayo_firmado.denegado(format($s$SELECT ensayo_firmado.custodiar('f6',ensayo_firmado.preimagen('doc:00000000-0000-4000-8000-0000000000f6','idem:00000000-0000-4000-8000-0000000000f6',%L,%L,'firmact:00000000-0000-4000-8000-0000000000f6'),ensayo_firmado.objeto('obj:00000000-0000-4000-8000-0000000000f6'),'decision:00000000-0000-4000-8000-0000000009f6','documentos.generado.alta','alta_documento_generado')$s$,reservado,original),'42501');
 PERFORM ensayo_firmado.denegado(format($s$SELECT ensayo_firmado.custodiar('f7',ensayo_firmado.preimagen('doc:00000000-0000-4000-8000-0000000000f7','idem:00000000-0000-4000-8000-0000000000f7',%L,%L,'firmact:00000000-0000-4000-8000-0000000000f7'),ensayo_firmado.objeto('obj:00000000-0000-4000-8000-0000000000f7'),'decision:00000000-0000-4000-8000-0000000009f7','documentos.firmado.custodiar','alta_documento_generado')$s$,reservado,original),'42501');
 -- 5. El alta genérica no puede usar el tipo reservado (comprobación diferida).
 PERFORM ensayo_firmado.denegado($s$DO $d$ BEGIN SET CONSTRAINTS ALL IMMEDIATE;
  PERFORM ensayo_externa.preparar('f8','documentos.generado.alta','doc:00000000-0000-4000-8000-0000000000f8','exp:00000000-0000-4000-8000-0000000000f8',
   'alta_documento_generado','documento_generado','["documento","recibo"]',
   convert_to('{"accion":"documentos.generado.alta","id":"doc:00000000-0000-4000-8000-0000000000f8","clave_idempotencia":"idem:00000000-0000-4000-8000-0000000000f8","modulo_id":"contratacion_temporal","expediente_ref":"exp:00000000-0000-4000-8000-0000000000f8","tipo_ref":"ref:f0074a505ef2a693dfd5bf2195228c47a7ce2a1435d4acf6a6b9f50e89e5663c","version":1,"mime":"application/pdf","tamano":3,"huella_sha256":"'||repeat('a',64)||'","politica_ref":"pol:00000000-0000-4000-8000-000000000001","version_politica":1,"huella_politica_sha256":"'||repeat('c',64)||'","proteccion":"conservacion","conservacion_hasta":"2036-01-01T00:00:00Z","estado_politica":"provisional"}','UTF8'),
   'decision:00000000-0000-4000-8000-0000000009f8');
  PERFORM vec_documentos.confirmar_alta_v2(m.preimagen,ensayo_firmado.objeto('obj:00000000-0000-4000-8000-0000000000f8'),m.auth,m.capacidad,m.decision,
   '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea)
   FROM ensayo_externa.material m WHERE m.caso='f8';
 END $d$$s$,'42501','tipo reservado a la custodia de documentos firmados');
END $f$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA ensayo_firmado TO vec_documentos_ensayo;
COMMIT;
