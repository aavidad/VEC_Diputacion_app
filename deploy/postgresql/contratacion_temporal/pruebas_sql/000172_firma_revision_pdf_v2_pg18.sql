\set ON_ERROR_STOP on
-- CT172: invariantes SQL en un clon sintético, dentro de ROLLBACK.
-- El doble transaccional AD162/AUT31 SOLO aísla el algoritmo de cadena.
-- Esta prueba no acredita autorización V3, credenciales, criptografía ni eficacia legal.
-- El ensayo causal y recorrido criptográfico con autoridades reales son otra puerta.
DO $acl$
BEGIN
 IF NOT has_function_privilege('vec_contratacion_temporal_ejecutor',
   'vec_contratacion_temporal.registrar_firma_verificada_v2(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
  OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor',
   'vec_contratacion_temporal.consultar_firmas_r5_atestadas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
  OR has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.firma_documento_revision_pdf_v2','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
  OR EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
    CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE n.nspname='vec_contratacion_temporal' AND p.proname IN ('registrar_firma_verificada_v2','consultar_firmas_r5_atestadas_v2',
       'impedir_degradacion_firma_pdf_v2','comprobar_hija_firma_pdf_v2') AND a.grantee=0)
  OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid='vec_contratacion_temporal.firma_documento_revision_pdf_v2'::regclass
   AND relrowsecurity AND relforcerowsecurity) THEN
  RAISE EXCEPTION 'CT172 ACL/RLS abierta'; END IF;
END $acl$;
SELECT a.expediente_ref AS expediente,a.version-1 AS version,a.version AS version_actual,
       v.agregado_json->>'organizacion_ref' AS organizacion
 FROM vec_contratacion_temporal.expediente_integral_actual a
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=a.expediente_ref AND v.version=a.version-1
 JOIN vec_contratacion_temporal.expediente_version_integral actual
   ON actual.expediente_ref=a.expediente_ref AND actual.version=a.version
 WHERE a.version>1 AND v.agregado_json->>'organizacion_ref'=actual.agregado_json->>'organizacion_ref'
   AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1 f
     WHERE f.expediente_ref=a.expediente_ref)
 ORDER BY a.version DESC,a.expediente_ref LIMIT 1 \gset

BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SELECT set_config('ct172.org',:'organizacion',true),
       set_config('ct172.exp',:'expediente',true),
       set_config('ct172.ver',:'version',true),
       set_config('ct172.ver_actual',:'version_actual',true);
SELECT set_config('ct172.inicial_revision',coalesce((SELECT revision
 FROM vec_contratacion_temporal.firma_historia_cabeza_v1
 WHERE organizacion_ref=:'organizacion' AND expediente_ref=:'expediente'),0)::text,true);
DO $pre_fixture$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1
  WHERE organizacion_ref=current_setting('ct172.org') AND expediente_ref=current_setting('ct172.exp')
    AND documento IN ('ct172_prueba','ct172_otro')) THEN
  RAISE EXCEPTION 'CT172: nombres de documento sintético ya usados'; END IF;
END $pre_fixture$;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
UPDATE vec_contratacion_temporal.expediente_integral_actual SET
 version=:'version'::numeric,actualizada_en=date_trunc('microseconds',clock_timestamp()),
 operacion_ref='operacion:ct172:puntero-sintetico'
WHERE expediente_ref=:'expediente';
RESET ROLE;
CREATE ROLE vec_ct172_prueba LOGIN;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct172_prueba;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_verificada_ct_v2_atestada(
 p_solicitud text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 RETURN QUERY SELECT 'decision:ct172:prueba',d->>'recurso_ref',d->>'contexto_recurso_huella_sha256',
  encode(sha256(convert_to(random()::text,'UTF8')),'hex'),'auditoria:ct172:prueba:'||md5(random()::text),clock_timestamp(),true;
END $f$;
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion.revalidar_competencia_firmante_ct_v2(p_solicitud text)
RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN RETURN current_setting('ct172.aut31',true) IS DISTINCT FROM 'deny'; END $f$;
RESET ROLE;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
GRANT SELECT ON vec_contratacion_temporal.firma_historia_cabeza_v1 TO vec_ct172_prueba;
CREATE POLICY prueba_lectura_cabeza ON vec_contratacion_temporal.firma_historia_cabeza_v1 TO vec_ct172_prueba USING(true);
RESET ROLE;
CREATE FUNCTION pg_temp.solicitud_base(p_clave text,p_firmante text,p_secuencia integer,
 p_via text DEFAULT 'portafirmas_registro_rrhh') RETURNS text LANGUAGE sql AS $f$
SELECT (CASE WHEN p_via='certificado_vec' THEN x-'ReferenciaPortafirmasDeclarada'-'FechaPortafirmasDeclarada'
         ELSE x END)::text FROM (SELECT jsonb_build_object(
 'Via',p_via,'OrganizacionRef',current_setting('ct172.org'),
 'ExpedienteRef',current_setting('ct172.exp'),'VersionExpediente',current_setting('ct172.ver')::numeric,
 'Documento','ct172_prueba','CatalogoRef','vec.contratacion_temporal.circuito_firma:1',
 'CatalogoHuella',repeat('a',64),'PasoRef','vec.contratacion_temporal.circuito_firma:1:ct172_prueba.p1',
 'PasoOrden',1,'Secuencia',p_secuencia,
 'HistoriaRevision',coalesce((SELECT revision FROM vec_contratacion_temporal.firma_historia_cabeza_v1
  WHERE organizacion_ref=current_setting('ct172.org') AND expediente_ref=current_setting('ct172.exp')),0),
 'HistoriaHuella',coalesce((SELECT huella_sha256 FROM vec_contratacion_temporal.firma_historia_cabeza_v1
  WHERE organizacion_ref=current_setting('ct172.org') AND expediente_ref=current_setting('ct172.exp')),
  encode(sha256(convert_to('vec:ct:firma-historia:v1:[]','UTF8')),'hex')),
 'OriginalRef','documento:ct172:original:prueba','OriginalVersion',1,
 'OriginalHuella',repeat('b',64),'FirmadoHuella',repeat('c',64),'CertificadoHuella',repeat('d',64),
 'FirmanteRef','ref:'||repeat('d',64),'FirmantePrincipalRef',p_firmante,
 'PerfilFirmanteRef','prf_ct172_firmante_prueba','CargoFirmante','ct_cargo_jefatura',
 'UnidadFirmanteRef','unidad:ct172:prueba','PerfilActivoFirmanteRef','prf_ct172_firmante_prueba',
 'PuestoFirmanteRef',NULL,'AmbitoFirmanteRef',NULL,
 'AsignacionFirmanteRef','asignacion:ct172:prueba','AsignacionFirmanteVersion',1,
 'AsignacionFirmanteHuella',repeat('e',64),
 'VersionRolFirmanteRef','rol:ct172:firmante:prueba','VersionRolFirmanteHuella',repeat('f',64),
 'ControlVigenciaFirmanteRef','rol:ct172:firmante:prueba','ControlVigenciaFirmanteRevision',1,
 'ControlVigenciaFirmanteHuella',repeat('1',64),
 'AsignacionVigenteDesde','2026-01-01T00:00:00Z',
 'AsignacionVigenteHasta','2027-01-01T00:00:00Z',
 'ActoCompetenciaRef',NULL,'DelegacionRef',NULL,
 'PoliticaVerificacion','politica:vec:firma:verificacion-autonoma:v1',
 'RevocacionEstado','vigente','SelloTiempoEstado','no_presente',
 'ReferenciaPortafirmasDeclarada','referencia declarada de prueba',
 'FechaPortafirmasDeclarada','2026-10-02T16:00:00Z','ClaveIdempotencia',p_clave,
 'DocumentoCustodiaRef','documento:ct172:firmado:'||p_via,'DocumentoCustodiaVersion',1) AS x) q
$f$;

CREATE FUNCTION pg_temp.evidencia(p_orden integer,p_sha text,p_cert text,p_entrada integer,p_longitud integer) RETURNS jsonb LANGUAGE sql AS $f$
 SELECT jsonb_build_object('Orden',p_orden,'ByteRange',jsonb_build_array(0,p_entrada+50,p_longitud-50,50),
  'RevisionHuellaSHA256',p_sha,'ContenidoFirmadoHuellaSHA256',repeat('8',64),'RevisionLongitud',p_longitud,
  'CubreDocumentoCompletoHastaAqui',true,'FirmanteRef','ref:'||p_cert,'CertificadoHuellaSHA256',p_cert,
  'IntegridadEstado','valida','CadenaEstado','valida','CertificadoEstado','vigente','RevocacionEstado','vigente',
  'SelloTiempoEstado','no_presente','TipoFirma','aprobacion','NivelDocMDP',NULL,
  'CambiosDesdeAnterior',jsonb_build_object('Estado','permitidos','Detalle',jsonb_build_array('firma_anadida')))
$f$;
CREATE FUNCTION pg_temp.solicitud(p_clave text,p_orden integer,p_anterior jsonb DEFAULT NULL,p_primera text DEFAULT NULL) RETURNS text LANGUAGE plpgsql AS $f$
DECLARE s jsonb; e jsonb;
BEGIN
 s:=pg_temp.solicitud_base(p_clave,CASE WHEN p_orden=1 THEN 'per_ct172_firmante_prueba' ELSE 'per_ct172_segundo_firmante' END,p_orden)::jsonb;
 e:=CASE WHEN p_orden=1 THEN jsonb_build_array(pg_temp.evidencia(1,repeat('c',64),repeat('d',64),1000,1200))
  ELSE jsonb_build_array(p_primera::jsonb#>'{EvidenciaFirmasCanonica,0}',pg_temp.evidencia(2,repeat('2',64),repeat('3',64),1200,1400)) END;
 s:=s||jsonb_build_object('PoliticaVerificacion','politica:vec:firma:verificacion-autonoma:v2','CatalogoVersion',1,
  'RolIDFirmante','ct_cargo_jefatura','PerfilActivoOperadorRef','prf_ct172_registrador_prueba','CuentaFirmanteRef','cuenta:ct172:prueba','VinculoCredencialFirmanteRef','vinculo:ct172:prueba',
  'VinculoCredencialFirmanteRevision',1,'VinculoCredencialFirmanteHuella',repeat('4',64),
  'PasoOrden',p_orden,'OrdenFirmaPDF',p_orden,'PasoRef','vec.contratacion_temporal.circuito_firma:1:ct172_prueba.p'||p_orden,
  'FirmaAnteriorRef',p_anterior->>'FirmaRef','ReciboAnteriorRef',p_anterior->>'ReciboRef',
  'EntradaDocumentoRef',CASE WHEN p_orden=1 THEN s->>'OriginalRef' ELSE p_anterior->>'DocumentoCustodiaRef' END,
  'EntradaDocumentoVersion',1,'EntradaDocumentoHuella',CASE WHEN p_orden=1 THEN s->>'OriginalHuella' ELSE repeat('c',64) END,
  'EntradaDocumentoLongitud',CASE WHEN p_orden=1 THEN 1000 ELSE 1200 END,
  'ByteRange',e->(p_orden-1)->'ByteRange','RevisionLongitud',e->(p_orden-1)->'RevisionLongitud',
  'RevisionHuellaSHA256',e->(p_orden-1)->'RevisionHuellaSHA256','FirmadoHuella',e->(p_orden-1)->'RevisionHuellaSHA256',
  'ContenidoFirmadoHuellaSHA256',repeat('8',64),'CertificadoHuella',e->(p_orden-1)->'CertificadoHuellaSHA256',
  'FirmanteRef',e->(p_orden-1)->'FirmanteRef','EvidenciaFirmasCanonica',e,
  'EvidenciaFirmasHuellaSHA256',encode(sha256(convert_to(e::text,'UTF8')),'hex'),
  'DocumentoCustodiaRef','documento:ct172:firmado:paso:'||p_orden);
 RETURN s::text;
END $f$;
CREATE FUNCTION pg_temp.decision(p_solicitud text,p_actor text) RETURNS bytea LANGUAGE sql AS $f$
SELECT convert_to(jsonb_build_object('accion',CASE WHEN p_solicitud::jsonb->>'Via'='certificado_vec'
   THEN 'contratacion_temporal.documento.firma_vec.registrar' ELSE 'contratacion_temporal.documento.firma_externa.registrar' END,
 'modulo_id','contratacion_temporal','tipo_recurso',CASE WHEN p_solicitud::jsonb->>'Via'='certificado_vec'
   THEN 'firma_vec_documento_contratacion_temporal' ELSE 'firma_externa_documento_contratacion_temporal' END,
 'finalidad','gestionar_contratacion_temporal','version_rol_ref',
   CASE WHEN p_solicitud::jsonb->>'Via'='certificado_vec' THEN p_solicitud::jsonb->>'VersionRolFirmanteRef'
        ELSE 'rol:firma_externa_registro_ct_desarrollo:v1' END,
 'recurso_ref',(CASE WHEN p_solicitud::jsonb->>'Via'='certificado_vec'
   THEN 'operacion-firma-vec-ct:' ELSE 'operacion-firma-externa-ct:' END)||(p_solicitud::jsonb->>'ClaveIdempotencia'),
 'contexto_recurso_huella_sha256',encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||
 (p_solicitud::jsonb->>'OrganizacionRef')||'"},"atributos":{"material_sha256":"'||
 encode(sha256(convert_to(p_solicitud,'UTF8')),'hex')||'"}}','UTF8')),'hex'),
 'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,'principal_id',p_actor,
 'perfil_activo_ref',CASE WHEN p_solicitud::jsonb->>'Via'='certificado_vec'
   THEN p_solicitud::jsonb->>'PerfilActivoFirmanteRef' ELSE 'prf_ct172_registrador_prueba' END)::text,'UTF8')
$f$;

CREATE FUNCTION pg_temp.registrar(p_solicitud text,p_observada timestamptz DEFAULT '2026-10-03T00:00:00Z') RETURNS jsonb LANGUAGE sql AS $f$
 SELECT vec_contratacion_temporal.registrar_firma_verificada_v2(p_solicitud,p_observada,'\x',pg_temp.decision(p_solicitud,'per_ct172_registrador_prueba'),
 '\x','\x',1,1,'\x','\x','\x','\x')
$f$;
CREATE FUNCTION pg_temp.debe_fallar(p_sql text,p_codigo text,p_caso text) RETURNS void LANGUAGE plpgsql AS $f$
BEGIN
 BEGIN EXECUTE p_sql;
 EXCEPTION WHEN others THEN
   IF SQLSTATE <> p_codigo THEN RAISE EXCEPTION 'CT172 %: % esperado %, obtenido %',p_caso,SQLERRM,p_codigo,SQLSTATE; END IF;
   RAISE NOTICE 'OK %',p_caso; RETURN;
 END;
 RAISE EXCEPTION 'CT172 %: no se rechazó',p_caso;
END $f$;

GRANT USAGE ON SCHEMA pg_temp TO vec_ct172_prueba;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA pg_temp TO vec_ct172_prueba;
SET SESSION AUTHORIZATION vec_ct172_prueba;
DO $cadena$
DECLARE s1 text; s2 text; bad text; r1 jsonb; r2 jsonb; replay jsonb;
BEGIN
 s1:=pg_temp.solicitud('clave-ct172-paso-000001',1); r1:=pg_temp.registrar(s1);
 replay:=pg_temp.registrar(s1,'2026-10-03T00:10:00Z');
 IF r1->>'YaRegistrada'<>'false' OR replay->>'YaRegistrada'<>'true'
  OR r1->>'FirmaRef' IS DISTINCT FROM replay->>'FirmaRef' OR r1->>'ReciboRef' IS DISTINCT FROM replay->>'ReciboRef'
  OR r1->>'RegistradaEn' IS DISTINCT FROM replay->>'RegistradaEn' THEN
  RAISE EXCEPTION 'CT172 observación posterior duplicó acto'; END IF;
 s2:=pg_temp.solicitud('clave-ct172-paso-000002',2,r1,s1);
 bad:=(s2::jsonb||jsonb_build_object('EntradaDocumentoRef','documento:ct172:otra-custodia'))::text;
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.registrar(%L)',bad),'P1184','otra custodia de entrada');
 bad:=(s2::jsonb||jsonb_build_object('EntradaDocumentoLongitud',1199))::text;
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.registrar(%L)',bad),'P1184','longitud antecedente divergente');
 bad:=(s2::jsonb||jsonb_build_object('ReciboAnteriorRef','recibo-firma-ct:00000000-0000-4000-8000-000000000001'))::text;
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.registrar(%L)',bad),'P1184','recibo antecedente sustituido');
 bad:=(s2::jsonb||jsonb_build_object('OriginalRef','documento:ct172:otra-raiz'))::text;
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.registrar(%L)',bad),'P1184','original raíz sustituido');
 bad:=(s2::jsonb||jsonb_build_object('HistoriaRevision',0))::text;
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.registrar(%L)',bad),'P1701','CAS global antiguo');
 r2:=pg_temp.registrar(s2); replay:=pg_temp.registrar(s2,'2026-10-03T00:11:00Z');
 IF r2->>'ReciboRef' IS DISTINCT FROM replay->>'ReciboRef' OR r2->>'RegistradaEn' IS DISTINCT FROM replay->>'RegistradaEn'
  OR replay->>'YaRegistrada'<>'true' THEN RAISE EXCEPTION 'CT172 replay paso2 duplicado'; END IF;
 PERFORM set_config('ct172.aut31','deny',true);
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.registrar(%L)',s2),'42501','competencia revocada en replay');
 PERFORM set_config('ct172.aut31','',true);
 PERFORM set_config('ct172.firma1',r1->>'FirmaRef',true);PERFORM set_config('ct172.firma2',r2->>'FirmaRef',true);
 RAISE NOTICE 'CT172 guardas SQL cadena/replay/CAS OK; autorización/crípto no acreditadas';
END $cadena$;
RESET SESSION AUTHORIZATION;
DO $historia$
DECLARE f1 text:=current_setting('ct172.firma1');f2 text:=current_setting('ct172.firma2');
BEGIN
 IF (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_v1 WHERE firma_ref IN(f1,f2))<>2
  OR (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_revision_pdf_v2 WHERE firma_ref IN(f1,f2))<>2
  OR (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_custodia_v1 WHERE firma_ref IN(f1,f2))<>2
  OR (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_auditoria_v1 WHERE firma_ref IN(f1,f2))<>2
  OR (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_outbox_v1 WHERE firma_ref IN(f1,f2))<>2
  OR (SELECT comprobada_en FROM vec_contratacion_temporal.firma_documento_revision_pdf_v2 WHERE firma_ref=f1)<>'2026-10-03T00:00:00Z'::timestamptz
  OR (SELECT firma_anterior_ref FROM vec_contratacion_temporal.firma_documento_revision_pdf_v2 WHERE firma_ref=f2) IS DISTINCT FROM f1 THEN
  RAISE EXCEPTION 'CT172 efecto parcial/duplicado o revisión histórica sobrescrita'; END IF;
END $historia$;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SELECT pg_temp.debe_fallar('TRUNCATE vec_contratacion_temporal.firma_documento_revision_pdf_v2','55000','historia revisión no truncable');
RESET ROLE;
ROLLBACK;
