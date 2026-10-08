\set ON_ERROR_STOP on
-- SÓLO CLON DESECHABLE, como superusuario. Nunca en la principal ni en cidonia.
-- Comprueba el camino positivo de los dos comprobadores nuevos con filas
-- sintéticas selladas con la transacción actual. Para insertarlas se retiran
-- los CHECK y disparadores de las tres tablas DENTRO de esta transacción, que
-- termina en ROLLBACK. No acredita el recorrido causal con productores reales:
-- sólo que los validadores aceptan lo que deben aceptar.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='30s';
SET LOCAL session_replication_role=replica;
DO $preparar$
DECLARE r record;
BEGIN
 IF NOT (SELECT rolsuper FROM pg_roles WHERE rolname=current_user) THEN
  RAISE EXCEPTION 'prueba sólo para superusuario en un clon'; END IF;
 FOR r IN SELECT c.conrelid::regclass AS t,c.conname FROM pg_constraint c
  WHERE c.conrelid IN ('vec_autorizacion_atestada_v3.consumo_decision_v3'::regclass,
   'vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass,
   'vec_autorizacion_atestada_v3.atestacion_decision_v3'::regclass) AND c.contype IN ('c','f') LOOP
  EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I',r.t,r.conname);
 END LOOP;
END $preparar$;

CREATE TEMP TABLE sintetico(nombre text PRIMARY KEY, valor jsonb) ON COMMIT DROP;

-- Inserta consumo, auditoría v4 y atestación ligados para una decisión.
CREATE FUNCTION pg_temp.sembrar(p_ref text,p_decision jsonb,p_capacidad jsonb,p_efecto text,p_huella text,p_tipo text)
RETURNS jsonb LANGUAGE plpgsql AS $f$
DECLARE dc bytea:=convert_to(p_decision::text,'UTF8'); cc bytea; ahora timestamptz(6):=clock_timestamp();
 hd text; cap jsonb; hc text:=encode(sha256(convert_to('consumo:'||p_ref,'UTF8')),'hex');
BEGIN
 hd:=encode(sha256(dc),'hex');
 cap:=p_capacidad||jsonb_build_object('huella_decision_sha256',hd);
 cc:=convert_to(cap::text,'UTF8');
 INSERT INTO vec_autorizacion_atestada_v3.atestacion_decision_v3(decision_ref,huella_decision_sha256,decision_canonica,
  motivo_canonico,contexto_actor_canonico,payload_vec_ad_3,sobre_cose_sign1,evidencia_verificacion,raiz_publica_spki,
  capacidad_canonica,huella_capacidad_sha256,efecto_ref,huella_efecto_sha256,registrada_en)
 VALUES(p_ref,hd,dc,'\x00','\x00','\x00','\x00','\x00','\x00',cc,encode(sha256(cc),'hex'),p_efecto,p_huella,ahora);
 INSERT INTO vec_autorizacion_atestada_v3.consumo_decision_v3(decision_ref,huella_decision_sha256,nonce,efecto_ref,
  huella_efecto_sha256,consumo_huella_sha256,consumida_en,transaccion_origen)
 VALUES(p_ref,hd,'n-'||p_ref,p_efecto,p_huella,hc,ahora,pg_current_xact_id());
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(auditoria_ref,secuencia,decision_ref,efecto_ref,
  huella_efecto_sha256,anterior_sha256,huella_sha256,registrada_en,tipo_registro,version_consumo,actor_ref,
  perfil_activo_ref,finalidad_ref,proceso,canal,transaccion_origen)
 VALUES('aud-'||p_ref,900000000+abs(hashtext(p_ref))%99999999,p_ref,p_efecto,p_huella,repeat('0',64),encode(sha256(convert_to('auditoria:'||p_ref,'UTF8')),'hex'),ahora,p_tipo,4,
  p_decision->>'principal_id',p_decision->>'perfil_activo_ref',p_decision->>'finalidad','prueba','sql',pg_current_xact_id());
 RETURN jsonb_build_object('decision_ref',p_ref,'efecto_ref',p_efecto,'huella_efecto_sha256',p_huella,
  'consumo_huella_sha256',hc,'auditoria_ref','aud-'||p_ref,'consumida_en',ahora,'consumo_nuevo',true);
END $f$;

DO $sembrar$
DECLARE material text; contexto text; d jsonb; recibo jsonb;
 campos jsonb:='["ByteRange","CanonNominal","CanonNominalRef","CanonNominalSHA256","CatalogoHuella","CatalogoRef","CertificadoHuella","ClaveIdempotencia","CoincideFirmanteCandidato","CoincideFirmanteEnOtroPaso","ConMotivoDevolucion","ContenidoFirmadoHuellaSHA256","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","EntradaDocumentoHuella","EntradaDocumentoLongitud","EntradaDocumentoRef","EntradaDocumentoVersion","EvidenciaFirmasCanonica","EvidenciaFirmasHuellaSHA256","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaAnteriorRef","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","FirmanteRef","HistoriaHuella","HistoriaRevision","HistoriaSeparacionAcreditada","MaterialRootSHA256","OrdenFirmaPDF","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboAnteriorRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","RevisionHuellaSHA256","RevisionLongitud","Secuencia","SelloTiempoEstado","Via"]';
BEGIN
 -- Recuperación R5: material de diez claves y contexto de organización.
 material:='{"CatalogoHuella":"'||repeat('a',64)||'","ClaveIdempotencia":"prueba-ad178-000001","Documento":"contrato","ExpedienteRef":"exp-sintetico-1","FirmantePrincipalCandidatoRef":"per_sintetico","OrganizacionRef":"org-sintetica","PasoOrden":1,"UnidadRef":null,"VersionExpediente":3,"Via":"certificado_vec"}';
 contexto:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"org-sintetica"},"atributos":{"material_sha256":"'||encode(sha256(convert_to(material,'UTF8')),'hex')||'"}}','UTF8')),'hex');
 d:=jsonb_build_object('decision_ref','decision:sintetica-recuperacion','concedida',true,'codigo','concedida',
  'accion','contratacion_temporal.documento.firmas_r5_v2.recuperar','modulo_id','contratacion_temporal',
  'tipo_recurso','expediente_contratacion_temporal','finalidad','gestionar_contratacion_temporal',
  'recurso_ref','exp-sintetico-1','contexto_recurso_huella_sha256',contexto,
  'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'),
  'campos_permitidos',campos,'obligaciones','[]'::jsonb,'principal_id','per_sintetico','perfil_activo_ref','perfil:sintetico',
  'emitida_en',to_char(clock_timestamp()-interval '1 minute','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
  'valida_hasta',to_char(clock_timestamp()+interval '10 minutes','YYYY-MM-DD"T"HH24:MI:SS"Z"'));
 recibo:=pg_temp.sembrar('decision:sintetica-recuperacion',d,jsonb_build_object('operacion',d->>'accion',
  'audiencia_consumo','vec_contratacion_temporal.firmas_r5.recuperar.v2','efecto_ref','exp-sintetico-1','huella_efecto_sha256',contexto),
  'exp-sintetico-1',contexto,'consumo_confirmado_v4');
 INSERT INTO sintetico VALUES('rec_material',to_jsonb(material)),('rec_recibo',recibo);
 -- Gobierno del plan: recurso catalogo_id:version y superficie privilegiada.
 contexto:=repeat('e',64);
 d:=jsonb_build_object('decision_ref','decision:sintetica-gobierno','concedida',true,'codigo','concedida',
  'accion','vec.catalogos.publicar','modulo_id','contratacion_temporal','tipo_recurso','catalogo_configurable',
  'finalidad','gestionar_contratacion_temporal','recurso_ref','ct.plan.firma:1','contexto_recurso_huella_sha256',contexto,
  'vinculo_autenticacion_actor',jsonb_build_object('superficie','administracion_privilegiada','cuenta_privilegiada',true),
  'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,'principal_id','per_sintetico','perfil_activo_ref','perfil:sintetico',
  'version_rol_ref','rol:sintetico:1','asignacion_ref','asig:sintetica',
  'valida_hasta',to_char(clock_timestamp()+interval '10 minutes','YYYY-MM-DD"T"HH24:MI:SS"Z"'));
 recibo:=pg_temp.sembrar('decision:sintetica-gobierno',d,jsonb_build_object('operacion','vec.catalogos.publicar',
  'audiencia_consumo','vec_catalogos_configurables.plan_nominal_firma.gobierno.v1','efecto_ref','ct.plan.firma:1','huella_efecto_sha256',contexto),
  'ct.plan.firma:1',contexto,'consumo_confirmado_v4');
 INSERT INTO sintetico VALUES('gob_recibo',recibo);
END $sembrar$;
GRANT SELECT ON sintetico TO vec_autorizacion_propietario,vec_catalogos_configurables_propietario;

-- 1. Recuperación: el comprobador acepta y devuelve sus ocho claves.
SET LOCAL ROLE vec_autorizacion_propietario;
DO $recuperacion$
DECLARE m bytea; r jsonb; x jsonb;
BEGIN
 SELECT convert_to(valor#>>'{}','UTF8') INTO m FROM sintetico WHERE nombre='rec_material';
 SELECT valor INTO r FROM sintetico WHERE nombre='rec_recibo';
 x:=vec_autorizacion_atestada_v3.comprobar_consumo_recuperacion_firmas_ct_v2(m,r);
 IF (SELECT count(*) FROM jsonb_object_keys(x))<>8 OR x->>'expediente_ref'<>'exp-sintetico-1'
  OR x->>'version_expediente'<>'3' OR x->>'decision_ref'<>'decision:sintetica-recuperacion' THEN
  RAISE EXCEPTION 'positivo recuperación divergente: %',x; END IF;
 -- decision_ref de 600 caracteres: se rechaza por forma, no por la regex.
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.comprobar_consumo_recuperacion_firmas_ct_v2(m,r||jsonb_build_object('decision_ref',repeat('d',600)));
  RAISE EXCEPTION 'decision_ref de 600 aceptada';
 EXCEPTION WHEN insufficient_privilege THEN
  IF SQLERRM NOT LIKE '%clave=consumo%' THEN RAISE EXCEPTION 'causa inesperada: %',SQLERRM; END IF;
 END;
END $recuperacion$;
RESET ROLE;

-- 2. Un sello de otra transacción deniega aunque todo lo demás coincida.
UPDATE vec_autorizacion_atestada_v3.consumo_decision_v3 SET transaccion_origen=NULL
 WHERE decision_ref='decision:sintetica-recuperacion';
SET LOCAL ROLE vec_autorizacion_propietario;
DO $sello$
DECLARE m bytea; r jsonb;
BEGIN
 SELECT convert_to(valor#>>'{}','UTF8') INTO m FROM sintetico WHERE nombre='rec_material';
 SELECT valor INTO r FROM sintetico WHERE nombre='rec_recibo';
 PERFORM vec_autorizacion_atestada_v3.comprobar_consumo_recuperacion_firmas_ct_v2(m,r);
 RAISE EXCEPTION 'sello NULL aceptado';
EXCEPTION WHEN insufficient_privilege THEN
 IF SQLERRM NOT LIKE '%clave=filas%' THEN RAISE EXCEPTION 'causa inesperada: %',SQLERRM; END IF;
END $sello$;
RESET ROLE;

-- 3. Gobierno: pasa todas sus comprobaciones y se detiene en la categoría
-- Aplicación de AUT, que hoy no admite vec.catalogos.* (falta la extensión de K).
SET LOCAL ROLE vec_catalogos_configurables_propietario;
DO $gobierno$
DECLARE r jsonb;
BEGIN
 SELECT valor INTO r FROM sintetico WHERE nombre='gob_recibo';
 PERFORM vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1(r);
 RAISE EXCEPTION 'gobierno aceptado sin categoría Aplicación';
EXCEPTION WHEN insufficient_privilege THEN
 IF SQLERRM<>'AD177 categoría Aplicación no acreditada' THEN RAISE EXCEPTION 'causa inesperada: %',SQLERRM; END IF;
END $gobierno$;
RESET ROLE;
ROLLBACK;
SELECT 'AD177-AD178-POSITIVO-SINTETICO-OK; filas sintéticas deshechas, sin recorrido causal';
