\set ON_ERROR_STOP on
-- SÓLO CLON DESECHABLE, como superusuario. Nunca en la principal ni en cidonia.
-- Camino positivo CT175 -> AUT41 -> comprobador AD178 reales. Dentro de una
-- transacción que termina en ROLLBACK se retiran CHECK/FK de las tablas
-- sembradas y se sustituye SÓLO la fachada de consumo AD178 por un doble que
-- siembra consumo, auditoría v4 y atestación selladas con esta transacción.
-- La llamada se hace como LOGIN del runtime CT (SET SESSION AUTHORIZATION).
-- Para el caso «sin firma exacta» basta cambiar ClaveIdempotencia en la
-- solicitud: también debe devolver una recuperación. No acredita el recorrido
-- causal con una decisión atestada real.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='60s';
SET LOCAL session_replication_role=replica;
DO $preparar$
DECLARE r record;
BEGIN
 FOR r IN SELECT c.conrelid::regclass AS t,c.conname FROM pg_constraint c
  WHERE c.conrelid IN ('vec_autorizacion_atestada_v3.consumo_decision_v3'::regclass,
   'vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass,
   'vec_autorizacion_atestada_v3.atestacion_decision_v3'::regclass,
   'vec_autorizacion.evidencia_competencia_firmante_ct_v1'::regclass,
   'vec_contratacion_temporal.firma_documento_v1'::regclass,
   'vec_contratacion_temporal.firma_documento_revision_pdf_v2'::regclass,
   'vec_contratacion_temporal.expediente_integral_actual'::regclass,
   'vec_contratacion_temporal.expediente_version_integral'::regclass) AND c.contype IN ('c','f') LOOP
  EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I',r.t,r.conname);
 END LOOP;
END $preparar$;

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_recuperacion_firmas_r5_ct_v2_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' AS $f$
DECLARE c jsonb:=convert_from(p_capacidad,'UTF8')::jsonb; d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
 ahora timestamptz(6):=clock_timestamp(); hd text:=encode(sha256(p_decision),'hex');
 ref text:=d->>'decision_ref'; hc text:=encode(sha256(convert_to('consumo:'||(d->>'decision_ref'),'UTF8')),'hex');
BEGIN
 INSERT INTO vec_autorizacion_atestada_v3.atestacion_decision_v3(decision_ref,huella_decision_sha256,decision_canonica,
  motivo_canonico,contexto_actor_canonico,payload_vec_ad_3,sobre_cose_sign1,evidencia_verificacion,raiz_publica_spki,
  capacidad_canonica,huella_capacidad_sha256,efecto_ref,huella_efecto_sha256,registrada_en)
 VALUES(ref,hd,p_decision,'\x00','\x00','\x00','\x00','\x00','\x00',p_capacidad,encode(sha256(p_capacidad),'hex'),
  c->>'efecto_ref',c->>'huella_efecto_sha256',ahora);
 INSERT INTO vec_autorizacion_atestada_v3.consumo_decision_v3(decision_ref,huella_decision_sha256,nonce,efecto_ref,
  huella_efecto_sha256,consumo_huella_sha256,consumida_en,transaccion_origen)
 VALUES(ref,hd,'n-'||ref,c->>'efecto_ref',c->>'huella_efecto_sha256',hc,ahora,pg_current_xact_id());
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(auditoria_ref,secuencia,decision_ref,efecto_ref,
  huella_efecto_sha256,anterior_sha256,huella_sha256,registrada_en,tipo_registro,version_consumo,actor_ref,
  perfil_activo_ref,finalidad_ref,proceso,canal,transaccion_origen)
 VALUES('aud-'||ref,900000000+abs(hashtext(ref))%99999999,ref,c->>'efecto_ref',c->>'huella_efecto_sha256',repeat('0',64),
  encode(sha256(convert_to('auditoria:'||ref,'UTF8')),'hex'),ahora,'consumo_confirmado_v4',4,
  d->>'principal_id',d->>'perfil_activo_ref',d->>'finalidad','prueba','sql',pg_current_xact_id());
 RETURN QUERY SELECT ref,c->>'efecto_ref',c->>'huella_efecto_sha256',hc,'aud-'||ref,ahora,true;
END $f$;

CREATE TEMP TABLE caso(nombre text PRIMARY KEY, valor text) ON COMMIT DROP;
GRANT SELECT ON caso TO PUBLIC;

DO $sembrar$
DECLARE org text:='org-sintetica'; exp text:='exp-sintetico-175'; clave text:='prueba-ct175-pos-0001';
 efecto text; canon bytea; canon_h text; ev_ref text; firma text:='firma-ct:'||gen_random_uuid()::text;
 solicitud text; h text; ctx text; d jsonb; c jsonb;
 campos jsonb:='["ByteRange","CanonNominal","CanonNominalRef","CanonNominalSHA256","CatalogoHuella","CatalogoRef","CertificadoHuella","ClaveIdempotencia","CoincideFirmanteCandidato","CoincideFirmanteEnOtroPaso","ConMotivoDevolucion","ContenidoFirmadoHuellaSHA256","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","EntradaDocumentoHuella","EntradaDocumentoLongitud","EntradaDocumentoRef","EntradaDocumentoVersion","EvidenciaFirmasCanonica","EvidenciaFirmasHuellaSHA256","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaAnteriorRef","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","FirmanteRef","HistoriaHuella","HistoriaRevision","HistoriaSeparacionAcreditada","MaterialRootSHA256","OrdenFirmaPDF","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboAnteriorRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","RevisionHuellaSHA256","RevisionLongitud","Secuencia","SelloTiempoEstado","Via"]';
BEGIN
 efecto:='operacion-firma-vec-ct:'||clave;
 canon:=convert_to(jsonb_build_object('esquema','vec.competencia-firmante.historica.v1',
  'recurso',jsonb_build_object('organizacion_ref',org,'unidad_ref','unidad-sintetica','expediente_ref',exp,'documento_ref','doc-original-1'),
  'relleno',repeat('x',600))::text,'UTF8');
 canon_h:=encode(sha256(canon),'hex');
 ev_ref:='evidencia:competencia-firmante-ct:'||encode(sha256(convert_to(efecto,'UTF8')||canon),'hex');
 INSERT INTO vec_autorizacion.evidencia_competencia_firmante_ct_v1(evidencia_ref,efecto_ref,esquema,canonico,huella_sha256,
  organizacion_ref,unidad_ref,expediente_ref,documento_ref,persona_ref,certificado_der_sha256,organizacion_destino,
  organizacion_destino_huella_sha256,catalogo_politicas_revision,catalogo_politicas_huella_sha256,consumo_decision_ref,
  consumo_huella_sha256,auditoria_consumo_ref,auditoria_consumo_huella_sha256,registrador_principal_ref,registrador_perfil_ref)
 VALUES(ev_ref,efecto,'vec.competencia-firmante.historica.v1',canon,canon_h,org,'unidad-sintetica',exp,'doc-original-1',
  'per_x',repeat('1',64),'{}',repeat('2',64),1,repeat('3',64),'decision:firma-original',repeat('4',64),'aud-firma-original',repeat('5',64),'per_x','perfil:x');
 INSERT INTO vec_contratacion_temporal.expediente_integral_actual VALUES(exp,3,clock_timestamp(),'op-1');
 INSERT INTO vec_contratacion_temporal.expediente_version_integral VALUES(exp,3,jsonb_build_object('organizacion_ref',org),
  repeat('6',64),'\x00',repeat('6',64),'flujo',1,repeat('6',64),'fase','estado','origen','op-1',clock_timestamp());
 INSERT INTO vec_contratacion_temporal.firma_documento_v1(firma_ref,organizacion_ref,expediente_ref,expediente_version,documento,
  secuencia,clave_idempotencia,solicitud_huella_sha256,catalogo_ref,catalogo_huella_sha256,paso_ref,paso_orden,resultado,
  politica_verificacion,actor_ref,perfil_ref,decision_ref,consumo_huella_sha256,auditoria_consumo_ref,recibo_ref,registrada_en,
  via_registro,historia_revision_observada,historia_huella_observada,original_documento_ref,unidad_firmante_ref,firmante_principal_ref,firmante_ref)
 VALUES(firma,org,exp,3,'contrato',1,clave,repeat('7',64),'cat',repeat('a',64),'paso-1',1,'firmado',
  'politica:vec:firma:verificacion-autonoma:v2','per_actor','perfil:a','decision:firma-original',repeat('4',64),'aud-firma-original',
  'recibo:1',clock_timestamp(),'certificado_vec',0,repeat('8',64),'doc-original-1','unidad-sintetica','per_sintetico','per_sintetico');
 INSERT INTO vec_contratacion_temporal.firma_documento_revision_pdf_v2 VALUES(firma,1,NULL,NULL,'doc-entrada',1,repeat('9',64),100,1,
  ARRAY[0,1,2,3]::bigint[],repeat('9',64),repeat('9',64),200,'{}','{}',repeat('9',64),clock_timestamp(),'rol','cuenta','vinculo',1,repeat('9',64),
  ev_ref,canon_h,'vec.competencia-firmante.historica.v1','decision:firma-original',repeat('4',64),'aud-firma-original','\x00',repeat('9',64));
 solicitud:='{"OrganizacionRef":"'||org||'","ExpedienteRef":"'||exp||'","VersionExpediente":3,"Documento":"contrato","FirmantePrincipalCandidatoRef":"per_sintetico","ClaveIdempotencia":"'||clave||'","PasoOrden":1,"CatalogoHuella":"'||repeat('a',64)||'","Via":"certificado_vec","UnidadRef":null}';
 h:=encode(sha256(convert_to(solicitud,'UTF8')),'hex');
 ctx:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||org||'"},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 d:=jsonb_build_object('decision_ref','decision:sintetica-ct175','concedida',true,'codigo','concedida',
  'accion','contratacion_temporal.documento.firmas_r5_v2.recuperar','modulo_id','contratacion_temporal',
  'tipo_recurso','expediente_contratacion_temporal','finalidad','gestionar_contratacion_temporal',
  'recurso_ref',exp,'contexto_recurso_huella_sha256',ctx,
  'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'),
  'campos_permitidos',campos,'obligaciones','[]'::jsonb,'principal_id','per_sintetico','perfil_activo_ref','perfil:sintetico',
  'emitida_en',to_char(clock_timestamp()-interval '1 minute','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
  'valida_hasta',to_char(clock_timestamp()+interval '10 minutes','YYYY-MM-DD"T"HH24:MI:SS"Z"'));
 c:=jsonb_build_object('efecto_ref',exp,'huella_efecto_sha256',ctx,'huella_decision_sha256',encode(sha256(convert_to(d::text,'UTF8')),'hex'),
  'operacion','contratacion_temporal.documento.firmas_r5_v2.recuperar','audiencia_consumo','vec_contratacion_temporal.firmas_r5.recuperar.v2');
 INSERT INTO caso VALUES('solicitud',solicitud),('c',c::text),('d',d::text),('canon_h',canon_h),('firma',firma);
END $sembrar$;

SET SESSION AUTHORIZATION vec_ct_o207_runtime;
DO $llamar$
DECLARE r jsonb; s text; c bytea; d bytea; canon_h text; firma text;
BEGIN
 SELECT valor INTO s FROM caso WHERE nombre='solicitud';
 SELECT convert_to(valor,'UTF8') INTO c FROM caso WHERE nombre='c';
 SELECT convert_to(valor,'UTF8') INTO d FROM caso WHERE nombre='d';
 SELECT valor INTO canon_h FROM caso WHERE nombre='canon_h';
 SELECT valor INTO firma FROM caso WHERE nombre='firma';
 r:=vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v2(s,c,d,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 RAISE NOTICE 'resultado: Encontrado=% firmas=% revisiones=% recuperaciones=% canonSHA=% sep=% coincide=%',
  r->'Encontrado',jsonb_array_length(r->'Firmas'),jsonb_array_length(r->'RevisionesPDF'),jsonb_array_length(r->'Recuperaciones'),
  r#>>'{Recuperaciones,0,CanonNominalSHA256}',r->'HistoriaSeparacionAcreditada',r->'CoincideFirmanteEnOtroPaso';
 IF r->>'Encontrado'<>'true' OR jsonb_array_length(r->'Recuperaciones')<>1
  OR r#>>'{Recuperaciones,0,CanonNominalSHA256}'<>canon_h OR r#>>'{Recuperaciones,0,FirmaRef}'<>firma THEN
  RAISE EXCEPTION 'positivo CT175 divergente: %',r; END IF;
END $llamar$;
RESET SESSION AUTHORIZATION;
SET LOCAL session_replication_role=replica;

-- Negativo: canon manipulado (la huella almacenada ya no casa) -> AUT41 55000.
UPDATE vec_autorizacion.evidencia_competencia_firmante_ct_v1 SET canonico=canonico||'\x20'::bytea;
SET SESSION AUTHORIZATION vec_ct_o207_runtime;
DO $neg$
DECLARE s text; c bytea; d bytea;
BEGIN
 SELECT valor INTO s FROM caso WHERE nombre='solicitud';
 SELECT convert_to(valor,'UTF8') INTO c FROM caso WHERE nombre='c';
 SELECT convert_to(replace(valor,'decision:sintetica-ct175','decision:sintetica-ct175b'),'UTF8') INTO d FROM caso WHERE nombre='d';
 c:=convert_to(jsonb_set(convert_from(c,'UTF8')::jsonb,'{huella_decision_sha256}',to_jsonb(encode(sha256(d),'hex')))::text,'UTF8');
 PERFORM vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v2(s,c,d,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 RAISE EXCEPTION 'canon manipulado aceptado';
EXCEPTION WHEN SQLSTATE '55000' THEN RAISE NOTICE 'negativo canon manipulado: % (55000)',SQLERRM;
END $neg$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
SELECT 'CT175-AUT41-POSITIVO-SINTETICO-OK';
