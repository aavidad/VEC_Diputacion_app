\set ON_ERROR_STOP on
-- Superficie EXCLUSIVA del contenedor efímero de probar_integracion_pg18.sh.
-- Sustituye en ESA base la fachada AD3-60 por un recibo sintético (la de
-- replay AD3-62 ya lo está) para ejercitar la lógica documental de custodia
-- externa y de la lista v2. No es una prueba COSE ni se instala en otra base.
BEGIN;
SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_documentos_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb:=convert_from(p_capacidad,'UTF8')::jsonb; d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 IF EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.ensayo_consumo_documentos ec WHERE ec.decision_ref=d->>'decision_ref')
 THEN RAISE EXCEPTION 'AD3-60: requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_autorizacion_atestada_v3.ensayo_consumo_documentos VALUES
  (d->>'decision_ref',p_capacidad,p_decision,c->>'efecto_ref',c->>'huella_efecto_sha256');
 RETURN QUERY SELECT d->>'decision_ref',c->>'efecto_ref',c->>'huella_efecto_sha256',repeat('b',64),'audit:synthetic',clock_timestamp(),true;
END $f$;
RESET ROLE;

CREATE SCHEMA ensayo_externa AUTHORIZATION postgres;
GRANT USAGE ON SCHEMA ensayo_externa TO vec_documentos_ensayo;
CREATE TABLE ensayo_externa.material(caso text PRIMARY KEY, preimagen bytea NOT NULL, capacidad bytea NOT NULL,
 decision bytea NOT NULL, auth jsonb NOT NULL, recibo jsonb);
GRANT SELECT,INSERT,UPDATE ON ensayo_externa.material TO vec_documentos_ensayo;

-- Material sintético ligado a la preimagen exacta de cada operación.
CREATE FUNCTION ensayo_externa.preparar(p_caso text,p_accion text,p_recurso text,p_ambito text,p_finalidad text,
 p_tipo text,p_campos jsonb,p_preimagen bytea,p_decision text) RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE h text:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"preimagen_sha256":"'||encode(sha256(p_preimagen),'hex')||'"}}','UTF8')),'hex'); c bytea; d bytea;
BEGIN
 c:=convert_to(jsonb_build_object('audiencia_consumo','vec_documentos.operacion.v1','operacion',p_accion,'efecto_ref',p_recurso,
  'huella_efecto_sha256',h,'decision_ref',p_decision,'nonce','nonce:'||p_decision,
  'emitida_en',to_char(clock_timestamp(),'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'expira_en',to_char(clock_timestamp()+interval '5 seconds','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'decision_valida_hasta',to_char(clock_timestamp()+interval '5 seconds','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,'UTF8');
 d:=convert_to(jsonb_build_object('accion',p_accion,'modulo_id','documentos','tipo_recurso',p_tipo,'finalidad',p_finalidad,
  'campos_permitidos',p_campos,'obligaciones','[]'::jsonb,'recurso_ref',p_recurso,'contexto_recurso_huella_sha256',h,
  'principal_id','per:00000000-0000-4000-8000-000000000001','perfil_activo_ref','perfil:00000000-0000-4000-8000-000000000001',
  'correlacion_ref','corr:00000000-0000-4000-8000-000000000002','decision_ref',p_decision)::text,'UTF8');
 INSERT INTO ensayo_externa.material(caso,preimagen,capacidad,decision,auth) VALUES(p_caso,p_preimagen,c,d,
  jsonb_build_object('accion',p_accion,'finalidad',p_finalidad,'recurso_ref',p_recurso,'ambito_ref',p_ambito,
   'principal_id','per:00000000-0000-4000-8000-000000000001','perfil_activo_ref','perfil:00000000-0000-4000-8000-000000000001',
   'correlacion_ref','corr:00000000-0000-4000-8000-000000000002'))
 ON CONFLICT (caso) DO UPDATE SET preimagen=EXCLUDED.preimagen,capacidad=EXCLUDED.capacidad,decision=EXCLUDED.decision,auth=EXCLUDED.auth;
END $f$;

CREATE FUNCTION ensayo_externa.preimagen_externa(p_id text,p_clave text,p_custodia text) RETURNS bytea
LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT convert_to('{"accion":"documentos.externo.registrar","id":"'||p_id||'","clave_idempotencia":"'||p_clave||'","modulo_id":"dietas","expediente_ref":"exp:00000000-0000-4000-8000-000000000001","tipo_ref":"tipo:00000000-0000-4000-8000-000000000002","version":1,"mime":"","tamano":0,"huella_sha256":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","custodio_id":"dietas.justificantes","custodia_ref":"'||p_custodia||'","politica_ref":"pol:00000000-0000-4000-8000-000000000001","version_politica":1,"huella_politica_sha256":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","proteccion":"conservacion","conservacion_hasta":"2030-01-01T00:00:00Z","estado_politica":"aprobada"}','UTF8')
$f$;

CREATE FUNCTION ensayo_externa.invocar(p_caso text) RETURNS jsonb
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE m ensayo_externa.material; r jsonb;
BEGIN
 SELECT * INTO STRICT m FROM ensayo_externa.material WHERE caso=p_caso;
 IF m.auth->>'accion'='documentos.externo.registrar' THEN
  SELECT vec_documentos.registrar_referencia_externa_v1(m.preimagen,m.auth,m.capacidad,m.decision,
   '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea) INTO r;
 ELSE
  SELECT vec_documentos.listar_expediente_v2(m.preimagen,m.auth,m.capacidad,m.decision,
   '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea) INTO r;
 END IF;
 RETURN r;
END $f$;

CREATE FUNCTION ensayo_externa.registrar() RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE r jsonb; r2 jsonb;
 id text:='doc:00000000-0000-4000-8000-0000000000e1'; clave text:='idem:00000000-0000-4000-8000-0000000000e1';
BEGIN
 PERFORM ensayo_externa.preparar('externa','documentos.externo.registrar',id,'exp:00000000-0000-4000-8000-000000000001',
  'registrar_documento_externo','documento_externo','["documento","recibo"]',
  ensayo_externa.preimagen_externa(id,clave,'justificante:dietas:0001'),'decision:00000000-0000-4000-8000-0000000000e1');
 r:=ensayo_externa.invocar('externa');
 IF r->>'custodia'<>'externa' OR r->>'custodio_id'<>'dietas.justificantes' OR r->>'custodia_ref'<>'justificante:dietas:0001'
    OR r->>'mime'<>'' OR (r->>'tamano')::bigint<>0 OR r->>'estado_firma'<>'pendiente_proveedor'
    OR r->>'numero_vec' !~ '^VEC-[0-9]{4}-[0-9]+$' OR r ? 'objeto_ref'
 THEN RAISE EXCEPTION 'registro externo incoherente: %',r; END IF;
 UPDATE ensayo_externa.material SET recibo=r WHERE caso='externa';
 -- Replay inmediato con el mismo material (sintético AD3-62): mismo recibo.
 r2:=ensayo_externa.invocar('externa');
 IF r2 IS DISTINCT FROM r THEN RAISE EXCEPTION 'replay externo cambió el recibo'; END IF;
END $f$;

CREATE FUNCTION ensayo_externa.negativos() RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 -- La misma clave con otra referencia de custodia es conflicto.
 PERFORM ensayo_externa.preparar('otra_custodia','documentos.externo.registrar','doc:00000000-0000-4000-8000-0000000000e1',
  'exp:00000000-0000-4000-8000-000000000001','registrar_documento_externo','documento_externo','["documento","recibo"]',
  ensayo_externa.preimagen_externa('doc:00000000-0000-4000-8000-0000000000e1','idem:00000000-0000-4000-8000-0000000000e1','justificante:dietas:0002'),
  'decision:00000000-0000-4000-8000-0000000000e2');
 BEGIN PERFORM ensayo_externa.invocar('otra_custodia'); RAISE EXCEPTION 'clave reutilizada aceptada';
 EXCEPTION WHEN SQLSTATE '23505' THEN NULL; END;
 -- El identificador del original VEC ya existente no puede nombrar otra cosa.
 PERFORM ensayo_externa.preparar('id_compartido','documentos.externo.registrar','doc:00000000-0000-4000-8000-000000000001',
  'exp:00000000-0000-4000-8000-000000000001','registrar_documento_externo','documento_externo','["documento","recibo"]',
  ensayo_externa.preimagen_externa('doc:00000000-0000-4000-8000-000000000001','idem:00000000-0000-4000-8000-0000000000e3','justificante:dietas:0003'),
  'decision:00000000-0000-4000-8000-0000000000e3');
 BEGIN PERFORM ensayo_externa.invocar('id_compartido'); RAISE EXCEPTION 'identificador compartido aceptado';
 EXCEPTION WHEN SQLSTATE '23505' THEN NULL; END;
 -- El mismo original externo no se anota dos veces en un expediente.
 PERFORM ensayo_externa.preparar('duplicado','documentos.externo.registrar','doc:00000000-0000-4000-8000-0000000000e6',
  'exp:00000000-0000-4000-8000-000000000001','registrar_documento_externo','documento_externo','["documento","recibo"]',
  ensayo_externa.preimagen_externa('doc:00000000-0000-4000-8000-0000000000e6','idem:00000000-0000-4000-8000-0000000000e6','justificante:dietas:0001'),
  'decision:00000000-0000-4000-8000-0000000000e6');
 BEGIN PERFORM ensayo_externa.invocar('duplicado'); RAISE EXCEPTION 'original externo duplicado aceptado';
 EXCEPTION WHEN SQLSTATE '23505' THEN NULL; END;
 -- Referencia de custodia con ruta: rechazada antes de consumir.
 PERFORM ensayo_externa.preparar('ruta','documentos.externo.registrar','doc:00000000-0000-4000-8000-0000000000e4',
  'exp:00000000-0000-4000-8000-000000000001','registrar_documento_externo','documento_externo','["documento","recibo"]',
  ensayo_externa.preimagen_externa('doc:00000000-0000-4000-8000-0000000000e4','idem:00000000-0000-4000-8000-0000000000e4','../etc/passwd'),
  'decision:00000000-0000-4000-8000-0000000000e4');
 BEGIN PERFORM ensayo_externa.invocar('ruta'); RAISE EXCEPTION 'referencia de custodia con ruta aceptada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 -- Registro externo con la finalidad del alta generada: denegado.
 PERFORM ensayo_externa.preparar('finalidad','documentos.externo.registrar','doc:00000000-0000-4000-8000-0000000000e5',
  'exp:00000000-0000-4000-8000-000000000001','alta_documento_generado','documento_externo','["documento","recibo"]',
  ensayo_externa.preimagen_externa('doc:00000000-0000-4000-8000-0000000000e5','idem:00000000-0000-4000-8000-0000000000e5','justificante:dietas:0005'),
  'decision:00000000-0000-4000-8000-0000000000e5');
 BEGIN PERFORM ensayo_externa.invocar('finalidad'); RAISE EXCEPTION 'finalidad ajena aceptada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $f$;

CREATE FUNCTION ensayo_externa.listar() RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE p bytea; r jsonb; r2 jsonb;
BEGIN
 p:=convert_to('{"accion":"documentos.expediente.listar","expediente_ref":"exp:00000000-0000-4000-8000-000000000001","cursor":"","limite":1}','UTF8');
 PERFORM ensayo_externa.preparar('lista1','documentos.expediente.listar','exp:00000000-0000-4000-8000-000000000001',
  'exp:00000000-0000-4000-8000-000000000001','listar_documentos_expediente','expediente_documental','["items","siguiente_cursor"]',p,
  'decision:00000000-0000-4000-8000-0000000000f1');
 r:=ensayo_externa.invocar('lista1');
 IF jsonb_array_length(r->'items')<>1 OR r->'items'->0->>'custodia'<>'vec'
    OR r->>'siguiente_cursor'<>'doc:00000000-0000-4000-8000-000000000001'
 THEN RAISE EXCEPTION 'primera página incoherente: %',r; END IF;
 p:=convert_to('{"accion":"documentos.expediente.listar","expediente_ref":"exp:00000000-0000-4000-8000-000000000001","cursor":"doc:00000000-0000-4000-8000-000000000001","limite":1}','UTF8');
 PERFORM ensayo_externa.preparar('lista2','documentos.expediente.listar','exp:00000000-0000-4000-8000-000000000001',
  'exp:00000000-0000-4000-8000-000000000001','listar_documentos_expediente','expediente_documental','["items","siguiente_cursor"]',p,
  'decision:00000000-0000-4000-8000-0000000000f2');
 r2:=ensayo_externa.invocar('lista2');
 IF jsonb_array_length(r2->'items')<>1 OR r2->'items'->0->>'custodia'<>'externa'
    OR r2->'items'->0->>'id'<>'doc:00000000-0000-4000-8000-0000000000e1' OR r2->>'siguiente_cursor'<>''
 THEN RAISE EXCEPTION 'segunda página incoherente: %',r2; END IF;
 -- La misma decisión de lista no se consume dos veces.
 BEGIN PERFORM ensayo_externa.invocar('lista2'); RAISE EXCEPTION 'lista repetida con la misma decisión';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $f$;

-- Tras reiniciar: decisión fresca con la misma preimagen, mismo recibo.
CREATE FUNCTION ensayo_externa.recuperar() RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE m ensayo_externa.material; r jsonb;
BEGIN
 SELECT * INTO STRICT m FROM ensayo_externa.material WHERE caso='externa';
 PERFORM ensayo_externa.preparar('externa_fresca','documentos.externo.registrar',m.auth->>'recurso_ref',m.auth->>'ambito_ref',
  'registrar_documento_externo','documento_externo','["documento","recibo"]',m.preimagen,'decision:00000000-0000-4000-8000-0000000000e9');
 r:=ensayo_externa.invocar('externa_fresca');
 IF r IS DISTINCT FROM m.recibo THEN RAISE EXCEPTION 'recuperación externa cambió el recibo: % / %',r,m.recibo; END IF;
END $f$;

GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA ensayo_externa TO vec_documentos_ensayo;
COMMIT;
