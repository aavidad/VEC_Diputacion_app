\set ON_ERROR_STOP on
-- CT170, clon sintético: comprobaciones sin efectos persistentes. Ejecutar
-- después de AD3-156 y CT170; todo el recorrido termina en ROLLBACK.
DO $pre$
BEGIN
 IF to_regprocedure('vec_contratacion_temporal.registrar_firma_verificada_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_firmas_documento_v3(text,text,text,text,text)') IS NULL
    OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor',
      'vec_contratacion_temporal.registrar_firma_verificada_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.firma_documento_v1','SELECT,INSERT,UPDATE,DELETE')
    OR has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.firma_historia_cabeza_v1','SELECT,INSERT,UPDATE,DELETE')
    OR has_function_privilege('vec_contratacion_temporal_ejecutor',
      'vec_contratacion_temporal.nueva_huella_firma_historia_v1(text,text,text,text,integer,text)','EXECUTE')
    OR has_function_privilege('public',
      'vec_contratacion_temporal.registrar_firma_verificada_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('vec_contratacion_temporal_ejecutor',
      'vec_contratacion_temporal.consultar_firmas_documento_v3(text,text,text,text,text)','EXECUTE')
    OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor',
      'vec_contratacion_temporal.consultar_firmas_r5_atestadas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'CT170: ACL/preimagen incorrecta'; END IF;
 IF (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_v1 WHERE via_registro IS NOT NULL) <> 0 THEN
   RAISE NOTICE 'CT170: existen firmas externas previas en el clon';
 END IF;
END $pre$;

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
SELECT set_config('ct170.org',:'organizacion',true),
       set_config('ct170.exp',:'expediente',true),
       set_config('ct170.ver',:'version',true),
       set_config('ct170.ver_actual',:'version_actual',true);
SELECT set_config('ct170.inicial_revision',coalesce((SELECT revision
 FROM vec_contratacion_temporal.firma_historia_cabeza_v1
 WHERE organizacion_ref=:'organizacion' AND expediente_ref=:'expediente'),0)::text,true);
DO $pre_fixture$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1
  WHERE organizacion_ref=current_setting('ct170.org') AND expediente_ref=current_setting('ct170.exp')
    AND documento IN ('ct170_prueba','ct170_otro')) THEN
  RAISE EXCEPTION 'CT170: nombres de documento sintético ya usados'; END IF;
END $pre_fixture$;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
UPDATE vec_contratacion_temporal.expediente_integral_actual SET
 version=:'version'::numeric,actualizada_en=date_trunc('microseconds',clock_timestamp()),
 operacion_ref='operacion:ct170:puntero-sintetico'
WHERE expediente_ref=:'expediente';
RESET ROLE;
CREATE ROLE vec_ct170_prueba LOGIN;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct170_prueba;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_externa_ct_v3_atestada(
 p_solicitud text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 RETURN QUERY SELECT 'decision:ct170:prueba',d->>'recurso_ref',d->>'contexto_recurso_huella_sha256',
   encode(sha256(convert_to(random()::text,'UTF8')),'hex'),'auditoria:ct170:prueba:'||md5(random()::text),
   clock_timestamp(),true;
END $f$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(
 p_solicitud text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 RETURN QUERY SELECT 'decision:ct170:prueba',d->>'recurso_ref',d->>'contexto_recurso_huella_sha256',
   encode(sha256(convert_to(random()::text,'UTF8')),'hex'),'auditoria:ct170:prueba:'||md5(random()::text),
   clock_timestamp(),true;
END $f$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_firmas_r5_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 IF current_setting('ct170.ad159',true)='deny' THEN
   RAISE EXCEPTION 'CT170: lectura revocada' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT 'decision:ct170:lectura',d->>'recurso_ref',d->>'contexto_recurso_huella_sha256',
   encode(sha256(convert_to(random()::text,'UTF8')),'hex'),'auditoria:ct170:lectura:'||md5(random()::text),
   clock_timestamp(),true;
END $f$;
RESET ROLE;
-- Solo este ROLLBACK concede lectura cruda al LOGIN sintético para comprobar
-- la proyección interna; CT170 instalada mantiene la ACL privada.
SET LOCAL ROLE vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_firmas_documento_v3(text,text,text,text,text)
 TO vec_ct170_prueba;
GRANT SELECT ON vec_contratacion_temporal.firma_historia_cabeza_v1 TO vec_ct170_prueba;
CREATE POLICY prueba_lectura_cabeza ON vec_contratacion_temporal.firma_historia_cabeza_v1
 TO vec_ct170_prueba USING (true);
RESET ROLE;

CREATE FUNCTION pg_temp.solicitud(p_clave text,p_firmante text,p_secuencia integer,
 p_via text DEFAULT 'portafirmas_registro_rrhh') RETURNS text LANGUAGE sql AS $f$
SELECT (CASE WHEN p_via='certificado_vec' THEN x-'ReferenciaPortafirmasDeclarada'-'FechaPortafirmasDeclarada'
         ELSE x END)::text FROM (SELECT jsonb_build_object(
 'Via',p_via,'OrganizacionRef',current_setting('ct170.org'),
 'ExpedienteRef',current_setting('ct170.exp'),'VersionExpediente',current_setting('ct170.ver')::numeric,
 'Documento','ct170_prueba','CatalogoRef','vec.contratacion_temporal.circuito_firma:1',
 'CatalogoHuella',repeat('a',64),'PasoRef','vec.contratacion_temporal.circuito_firma:1:ct170_prueba.p1',
 'PasoOrden',1,'Secuencia',p_secuencia,
 'HistoriaRevision',coalesce((SELECT revision FROM vec_contratacion_temporal.firma_historia_cabeza_v1
  WHERE organizacion_ref=current_setting('ct170.org') AND expediente_ref=current_setting('ct170.exp')),0),
 'HistoriaHuella',coalesce((SELECT huella_sha256 FROM vec_contratacion_temporal.firma_historia_cabeza_v1
  WHERE organizacion_ref=current_setting('ct170.org') AND expediente_ref=current_setting('ct170.exp')),
  encode(sha256(convert_to('vec:ct:firma-historia:v1:[]','UTF8')),'hex')),
 'OriginalRef','documento:ct170:original:prueba','OriginalVersion',1,
 'OriginalHuella',repeat('b',64),'FirmadoHuella',repeat('c',64),'CertificadoHuella',repeat('d',64),
 'FirmanteRef','ref:'||repeat('d',64),'FirmantePrincipalRef',p_firmante,
 'PerfilFirmanteRef','prf_ct170_firmante_prueba','CargoFirmante','Jefatura de prueba',
 'UnidadFirmanteRef','unidad:ct170:prueba','PerfilActivoFirmanteRef','prf_ct170_firmante_prueba',
 'PuestoFirmanteRef',NULL,'AmbitoFirmanteRef',NULL,
 'AsignacionFirmanteRef','asignacion:ct170:prueba','AsignacionFirmanteVersion',1,
 'AsignacionFirmanteHuella',repeat('e',64),
 'VersionRolFirmanteRef','rol:ct170:firmante:prueba','VersionRolFirmanteHuella',repeat('f',64),
 'ControlVigenciaFirmanteRef','rol:ct170:firmante:prueba','ControlVigenciaFirmanteRevision',1,
 'ControlVigenciaFirmanteHuella',repeat('1',64),
 'AsignacionVigenteDesde','2026-01-01T00:00:00Z',
 'AsignacionVigenteHasta','2027-01-01T00:00:00Z',
 'ActoCompetenciaRef',NULL,'DelegacionRef',NULL,
 'PoliticaVerificacion','politica:vec:firma:verificacion-autonoma:v1',
 'RevocacionEstado','vigente','SelloTiempoEstado','no_presente',
 'ReferenciaPortafirmasDeclarada','referencia declarada de prueba',
 'FechaPortafirmasDeclarada','2026-10-02T16:00:00Z','ClaveIdempotencia',p_clave,
 'DocumentoCustodiaRef','documento:ct170:firmado:'||p_via,'DocumentoCustodiaVersion',1) AS x) q
$f$;
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
   THEN p_solicitud::jsonb->>'PerfilActivoFirmanteRef' ELSE 'prf_ct170_registrador_prueba' END)::text,'UTF8')
$f$;
CREATE FUNCTION pg_temp.registrar(p_solicitud text,p_actor text) RETURNS jsonb LANGUAGE sql AS $f$
SELECT vec_contratacion_temporal.registrar_firma_verificada_v1(p_solicitud,'\x',pg_temp.decision(p_solicitud,p_actor),
 '\x','\x',1,1,'\x','\x','\x','\x')
$f$;
CREATE FUNCTION pg_temp.material_consulta(p_clave text DEFAULT 'clave-ct170-prueba-000006',
 p_version numeric DEFAULT NULL,p_documento text DEFAULT 'ct170_prueba',p_org text DEFAULT NULL)
RETURNS text LANGUAGE sql AS $f$
SELECT '{"OrganizacionRef":'||to_json(coalesce(p_org,current_setting('ct170.org')))::text||
 ',"ExpedienteRef":'||to_json(current_setting('ct170.exp'))::text||
 ',"VersionExpediente":'||coalesce(p_version,current_setting('ct170.ver')::numeric)::text||
 ',"Documento":'||to_json(p_documento)::text||
 ',"FirmantePrincipalCandidatoRef":"per_ct170_firmante_prueba","ClaveIdempotencia":'||to_json(p_clave)::text||
 ',"PasoOrden":1,"CatalogoHuella":'||to_json(repeat('a',64))::text||'}'
$f$;
CREATE FUNCTION pg_temp.decision_consulta(p_material text) RETURNS bytea LANGUAGE sql AS $f$
SELECT convert_to(jsonb_build_object(
 'accion','contratacion_temporal.documento.firmas_r5.consultar',
 'modulo_id','contratacion_temporal','tipo_recurso','expediente_contratacion_temporal',
 'finalidad','gestionar_contratacion_temporal','recurso_ref',p_material::jsonb->>'ExpedienteRef',
 'contexto_recurso_huella_sha256',encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||
  (p_material::jsonb->>'OrganizacionRef')||'"},"atributos":{"material_sha256":"'||
  encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}','UTF8')),'hex'),
 'campos_permitidos','["CatalogoHuella","CatalogoRef","ClaveIdempotencia","CoincideFirmanteCandidato","CoincideFirmanteEnOtroPaso","ConMotivoDevolucion","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","HistoriaHuella","HistoriaRevision","HistoriaSeparacionAcreditada","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","Secuencia","SelloTiempoEstado","Via"]'::jsonb,
 'obligaciones','[]'::jsonb)::text,'UTF8')
$f$;
CREATE FUNCTION pg_temp.capacidad_consulta(p_material text,p_decision bytea) RETURNS bytea LANGUAGE sql AS $f$
SELECT convert_to(jsonb_build_object(
 'operacion','contratacion_temporal.documento.firmas_r5.consultar',
 'audiencia_consumo','vec_contratacion_temporal.firmas_r5.consultar.v1',
 'efecto_ref',p_material::jsonb->>'ExpedienteRef',
 'huella_efecto_sha256',encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||
  (p_material::jsonb->>'OrganizacionRef')||'"},"atributos":{"material_sha256":"'||
  encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}','UTF8')),'hex'),
 'huella_decision_sha256',encode(sha256(p_decision),'hex'),
 'relleno',repeat('x',512))::text,'UTF8')
$f$;
CREATE FUNCTION pg_temp.consultar_r5(p_material text) RETURNS jsonb LANGUAGE sql AS $f$
SELECT vec_contratacion_temporal.consultar_firmas_r5_atestadas_v1(p_material,
 pg_temp.capacidad_consulta(p_material,pg_temp.decision_consulta(p_material)),
 pg_temp.decision_consulta(p_material),'\x78','\x78',1,1,'\x78','\x78','\x78',convert_to(repeat('z',44),'UTF8'))
$f$;
CREATE FUNCTION pg_temp.debe_fallar(p_sql text,p_codigo text,p_caso text) RETURNS void LANGUAGE plpgsql AS $f$
BEGIN
 BEGIN EXECUTE p_sql;
 EXCEPTION WHEN others THEN
   IF SQLSTATE <> p_codigo THEN RAISE EXCEPTION 'CT170 %: % esperado %, obtenido %',p_caso,SQLERRM,p_codigo,SQLSTATE; END IF;
   RAISE NOTICE 'OK %',p_caso; RETURN;
 END;
 RAISE EXCEPTION 'CT170 %: no se rechazó',p_caso;
END $f$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA pg_temp TO PUBLIC;
SET SESSION AUTHORIZATION vec_ct170_prueba;
DO $recorrido$
DECLARE s text; r jsonb; repetido jsonb; sv text; rv jsonb; consulta jsonb;
 s_otra_fuente text; s_otro_documento text; r_otro_documento jsonb; s_cabeza_antigua text;
BEGIN
 s:=pg_temp.solicitud('clave-ct170-prueba-000001','per_ct170_firmante_prueba',1);
 r:=pg_temp.registrar(s,'per_ct170_registrador_prueba');
 IF r->>'YaRegistrada' <> 'false' OR r->>'SolicitudHuella' <> encode(sha256(convert_to(s,'UTF8')),'hex') THEN
   RAISE EXCEPTION 'CT170: recibo inicial inválido'; END IF;
 repetido:=pg_temp.registrar(s,'per_ct170_registrador_prueba');
 IF repetido->>'YaRegistrada' <> 'true' OR repetido->>'ReciboRef' <> r->>'ReciboRef'
    OR repetido->>'RegistradaEn' <> r->>'RegistradaEn' THEN
   RAISE EXCEPTION 'CT170: replay no preserva recibo/fecha'; END IF;
 IF NOT EXISTS (SELECT 1 FROM jsonb_array_elements(vec_contratacion_temporal.consultar_firmas_documento_v3(
   current_setting('ct170.org'),current_setting('ct170.exp'),'ct170_prueba','per_ct170_firmante_prueba',NULL)) x
   WHERE x->>'FirmaRef'=r->>'FirmaRef' AND x->>'Via'='portafirmas_registro_rrhh'
     AND x->>'FirmantePrincipalAcreditado'='true' AND NOT (x ? 'FirmantePrincipalRef')) THEN
   RAISE EXCEPTION 'CT170: consulta sin vía externa'; END IF;
 sv:=pg_temp.solicitud('clave-ct170-prueba-000002','per_ct170_firmante_prueba',2,'certificado_vec');
 rv:=pg_temp.registrar(sv,'per_ct170_firmante_prueba');
 IF rv->>'YaRegistrada'<>'false' OR rv->>'Secuencia'<>'2'
    OR (pg_temp.registrar(sv,'per_ct170_firmante_prueba'))->>'ReciboRef'<>rv->>'ReciboRef' THEN
   RAISE EXCEPTION 'CT170: vía VEC o replay inválido'; END IF;
 IF NOT EXISTS (SELECT 1 FROM jsonb_array_elements(vec_contratacion_temporal.consultar_firmas_documento_v3(
   current_setting('ct170.org'),current_setting('ct170.exp'),'ct170_prueba','per_ct170_firmante_prueba',NULL)) x
   WHERE x->>'FirmaRef'=rv->>'FirmaRef' AND x->>'Via'='certificado_vec'
     AND x->>'FirmantePrincipalAcreditado'='true' AND NOT (x ? 'FirmantePrincipalRef')) THEN
   RAISE EXCEPTION 'CT170: consulta sin vía VEC'; END IF;
 s_otra_fuente:=(pg_temp.solicitud('clave-ct170-prueba-000003','per_ct170_firmante_prueba',3,'certificado_vec')::jsonb ||
   jsonb_build_object('PasoOrden',2,'PasoRef','vec.contratacion_temporal.circuito_firma:1:ct170_prueba.p2',
    'OriginalRef','documento:ct170:otra-fuente','DocumentoCustodiaRef','documento:ct170:firmado:otra'))::text;
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.registrar(%L,%L)',s_otra_fuente,
   'per_ct170_firmante_prueba'),'P1184','mismo SHA pero otra referencia original');
 consulta:=pg_temp.consultar_r5(pg_temp.material_consulta());
 IF consulta->>'Encontrado'<>'true' OR consulta->>'ExpedienteRef'<>current_setting('ct170.exp')
    OR (consulta->>'HistoriaRevision')::numeric<>current_setting('ct170.inicial_revision')::numeric+2
    OR consulta->>'HistoriaHuella'!~'^[0-9a-f]{64}$'
    OR consulta->>'CoincideFirmanteEnOtroPaso'<>'false'
    OR consulta->>'HistoriaSeparacionAcreditada'<>'true'
    OR jsonb_array_length(consulta->'Firmas')<>2
    OR EXISTS (SELECT 1 FROM jsonb_array_elements(consulta->'Firmas') x
       WHERE x ? 'FirmantePrincipalRef' OR jsonb_typeof(x->'FirmantePrincipalAcreditado')<>'boolean'
          OR jsonb_typeof(x->'CoincideFirmanteCandidato')<>'boolean'
          OR x->>'CoincideFirmanteCandidato'<>'true'
          OR x->>'HistoriaHuella'!~'^[0-9a-f]{64}$') THEN
   RAISE EXCEPTION 'CT170: lector atestado filtrado o proyección inválidos'; END IF;
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.registrar(%L,%L)',
   pg_temp.solicitud('clave-ct170-prueba-000001','per_ct170_otro_firmante',1),
   'per_ct170_registrador_prueba'),'P1181','clave con otro material');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.registrar(%L,%L)',s,
   'per_ct170_firmante_prueba'),'42501','registrador igual al firmante');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.registrar(%L,%L)',
   pg_temp.solicitud('clave-ct170-prueba-000002','per_ct170_firmante_prueba',2,'portafirmas_registro_rrhh'),
   'per_ct170_registrador_prueba'),'P1181','clave de otra vía');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.registrar(%L,%L)',sv,
   'per_ct170_registrador_prueba'),'42501','VEC requiere firmante actor');
 PERFORM set_config('ct170.aut30','deny',true);
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.registrar(%L,%L)',s,
   'per_ct170_registrador_prueba'),'42501','competencia revocada en replay');
 PERFORM set_config('ct170.aut30','',true);
 s_cabeza_antigua:=(pg_temp.solicitud('clave-ct170-prueba-000005','per_ct170_firmante_prueba',3,'certificado_vec')::jsonb ||
   jsonb_build_object('DocumentoCustodiaRef','documento:ct170:firmado:quinto'))::text;
 s_otro_documento:=(pg_temp.solicitud('clave-ct170-prueba-000004','per_ct170_firmante_prueba',1,'certificado_vec')::jsonb ||
   jsonb_build_object('Documento','ct170_otro','PasoRef','vec.contratacion_temporal.circuito_firma:1:ct170_otro.p1',
     'DocumentoCustodiaRef','documento:ct170:firmado:otro'))::text;
 r_otro_documento:=pg_temp.registrar(s_otro_documento,'per_ct170_firmante_prueba');
 IF r_otro_documento->>'YaRegistrada'<>'false' THEN RAISE EXCEPTION 'CT170: otro documento no registrado'; END IF;
 consulta:=pg_temp.consultar_r5(pg_temp.material_consulta());
 IF consulta->>'CoincideFirmanteEnOtroPaso'<>'true'
    OR consulta->>'HistoriaSeparacionAcreditada'<>'true' THEN
   RAISE EXCEPTION 'CT170: hechos de separación entre documentos inválidos'; END IF;
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.registrar(%L,%L)',s_cabeza_antigua,
   'per_ct170_firmante_prueba'),'P1701','cabeza global cambia por otro documento');
 consulta:=pg_temp.consultar_r5(pg_temp.material_consulta('clave-ct170-prueba-000001'));
 IF consulta->>'Encontrado'<>'true' OR jsonb_array_length(consulta->'Firmas')<>1
    OR consulta#>>'{Firmas,0,FirmaRef}'<>r->>'FirmaRef'
    OR (consulta->>'HistoriaRevision')::numeric<>current_setting('ct170.inicial_revision')::numeric
    OR consulta->>'CoincideFirmanteEnOtroPaso'<>'false'
    OR consulta->>'HistoriaSeparacionAcreditada'<>'false' THEN
   RAISE EXCEPTION 'CT170: replay de la misma versión reveló firmas posteriores'; END IF;
 PERFORM set_config('ct170.firma',r->>'FirmaRef',true);
 PERFORM set_config('ct170.solicitud_primera',s,true);
 PERFORM set_config('ct170.recibo_primero',r->>'ReciboRef',true);
 PERFORM set_config('ct170.firma_vec',rv->>'FirmaRef',true);
 PERFORM set_config('ct170.firma_otro',r_otro_documento->>'FirmaRef',true);
 RAISE NOTICE 'CT170: dos vías, replay, separación, consulta, auditoría y outbox OK';
END $recorrido$;
RESET SESSION AUTHORIZATION;
DO $persistencia$
DECLARE f text:=current_setting('ct170.firma'); fv text:=current_setting('ct170.firma_vec');
 fo text:=current_setting('ct170.firma_otro');
BEGIN
 IF (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_v1 WHERE firma_ref IN (f,fv,fo))<>3
    OR (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_custodia_v1 WHERE firma_ref IN (f,fv,fo))<>3
    OR (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_auditoria_v1 WHERE firma_ref IN (f,fv,fo))<>3
    OR (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_outbox_v1 WHERE firma_ref IN (f,fv,fo))<>3 THEN
   RAISE EXCEPTION 'CT170: efecto parcial o duplicado'; END IF;
 IF (SELECT revision FROM vec_contratacion_temporal.firma_historia_cabeza_v1
     WHERE organizacion_ref=current_setting('ct170.org') AND expediente_ref=current_setting('ct170.exp'))
       <>current_setting('ct170.inicial_revision')::numeric+3
    OR (SELECT historia_revision_observada FROM vec_contratacion_temporal.firma_documento_v1
        WHERE firma_ref=f)<>current_setting('ct170.inicial_revision')::numeric
    OR (SELECT historia_revision_observada FROM vec_contratacion_temporal.firma_documento_v1
        WHERE firma_ref=fv)<>current_setting('ct170.inicial_revision')::numeric+1
    OR (SELECT historia_revision_observada FROM vec_contratacion_temporal.firma_documento_v1
        WHERE firma_ref=fo)<>current_setting('ct170.inicial_revision')::numeric+2 THEN
   RAISE EXCEPTION 'CT170: cabeza global o CAS de filas inválidos'; END IF;
 RAISE NOTICE 'CT170: doce filas atómicas y únicas';
END $persistencia$;
-- Restaura el puntero a la versión ya existente. El ROLLBACK conserva
-- intacta la historia; la firma ahora es histórica para la consulta.
SET LOCAL ROLE vec_contratacion_temporal_propietario;
UPDATE vec_contratacion_temporal.expediente_integral_actual SET
 version=current_setting('ct170.ver_actual')::numeric,
 actualizada_en=date_trunc('microseconds',clock_timestamp()),
 operacion_ref='operacion:ct170:puntero-sintetico'
WHERE expediente_ref=current_setting('ct170.exp');
RESET ROLE;
SET SESSION AUTHORIZATION vec_ct170_prueba;
DO $historico$
DECLARE m text:=pg_temp.material_consulta('clave-ct170-prueba-000001'); q jsonb; repetido jsonb;
BEGIN
 q:=pg_temp.consultar_r5(m);
 IF q->>'Encontrado'<>'true' OR jsonb_array_length(q->'Firmas')<>1
    OR q#>>'{Firmas,0,FirmaRef}'<>current_setting('ct170.firma')
    OR (q->>'HistoriaRevision')::numeric<>current_setting('ct170.inicial_revision')::numeric
    OR q->>'HistoriaHuella' IS DISTINCT FROM current_setting('ct170.solicitud_primera')::jsonb->>'HistoriaHuella' THEN
   RAISE EXCEPTION 'CT170: replay histórico filtró mal o reveló cabeza posterior'; END IF;
 repetido:=pg_temp.registrar(current_setting('ct170.solicitud_primera'),'per_ct170_registrador_prueba');
 IF repetido->>'YaRegistrada'<>'true' OR repetido->>'ReciboRef'<>current_setting('ct170.recibo_primero') THEN
   RAISE EXCEPTION 'CT170: replay con versión anterior no recuperó recibo'; END IF;
 q:=pg_temp.consultar_r5(pg_temp.material_consulta());
 IF q->>'Encontrado'<>'false' OR jsonb_array_length(q->'Firmas')<>0
    OR (q->>'HistoriaRevision')::numeric<>0 THEN
   RAISE EXCEPTION 'CT170: clave histórica desconocida mostró datos'; END IF;
 q:=pg_temp.consultar_r5(pg_temp.material_consulta('clave-ct170-prueba-000001',NULL,'ct170_otro'));
 IF q->>'Encontrado'<>'false' OR jsonb_array_length(q->'Firmas')<>0 THEN
   RAISE EXCEPTION 'CT170: clave de otro documento mostró datos'; END IF;
 q:=pg_temp.consultar_r5(pg_temp.material_consulta('clave-ct170-prueba-000001',NULL,'ct170_prueba',
   'organizacion:ct170:ajena'));
 IF q->>'Encontrado'<>'false' OR jsonb_array_length(q->'Firmas')<>0 THEN
   RAISE EXCEPTION 'CT170: organización ajena mostró datos'; END IF;
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar_r5(%L)',
   pg_temp.material_consulta('clave-ct170-prueba-000001',current_setting('ct170.ver')::numeric+2)),
   'P1182','versión futura');
 PERFORM set_config('ct170.ad159','deny',true);
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar_r5(%L)',m),'42501','lectura revocada');
 PERFORM set_config('ct170.ad159','',true);
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.registrar(%L,%L)',
   pg_temp.solicitud('clave-ct170-prueba-000007','per_ct170_firmante_prueba',3,'certificado_vec'),
   'per_ct170_firmante_prueba'),'P1182','alta nueva con versión obsoleta');
 RAISE NOTICE 'CT170: replay histórico exacto, privacidad, revocación y OCC OK';
END $historico$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
