\set ON_ERROR_STOP on
-- CT224: consulta nominal del tercer paso de resolución.
-- El paso 3 exige unidad concreta y paso publicado vigente en CC10. Los pasos
-- 1 y 2 y la recuperación conservan su contrato instalado.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000224',0));
DO $pre$
DECLARE f oid:=to_regprocedure('vec_contratacion_temporal.consultar_firmas_r5_atestadas_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR pg_is_in_recovery() OR f IS NULL
  OR to_regprocedure('vec_catalogos_configurables.paso_plan_firma_con_unidad_v1(text,text,integer,text,text)') IS NULL
  OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner=to_regrole('vec_contratacion_temporal_propietario') AND p.prosecdef
   AND p.provolatile='v' AND p.proparallel='u'
   AND p.proconfig=ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[] AND p.proacl=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario','vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='3946a57a01656340fc7e3ec5ef2ea9f3695c13d42cc2ba2f63d36688d35662b3'
   AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='8c51c117fbf2671ed94ca5e4564a438e81f51d672466657dcad516c8c386b936')
 THEN RAISE EXCEPTION 'CT224: preimagen, propietario o plan distintos' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.consultar_firmas_r5_atestadas_v3(p_solicitud text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE s jsonb; c jsonb; d jsonb; consumo record; exacta record; cabeza record; firmas jsonb; revisiones jsonb;
 h text; contexto_h text; v numeric; org text; coincide boolean; separacion boolean;
 vacia text:=encode(sha256(convert_to('vec:ct:firma-historia:v1:[]','UTF8')),'hex');
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
  OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
  OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
  OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION 'lectura V2 denegada' USING ERRCODE='42501'; END IF;
 IF p_solicitud IS NULL OR octet_length(p_solicitud) NOT BETWEEN 2 AND 4096
  OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 1 AND 65536
  OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
  RAISE EXCEPTION 'lectura V2 inválida' USING ERRCODE='22023'; END IF;
 s:=p_solicitud::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(s,ARRAY['OrganizacionRef','ExpedienteRef','VersionExpediente',
   'Documento','FirmantePrincipalCandidatoRef','ClaveIdempotencia','PasoOrden','CatalogoHuella','Via','UnidadRef']) IS NOT TRUE
  OR (SELECT count(*) FROM json_each(p_solicitud::json))<>(SELECT count(*) FROM jsonb_each(s))
  OR jsonb_typeof(s->'Via') IS DISTINCT FROM 'string' OR s->>'Via' NOT IN('certificado_vec','portafirmas_registro_rrhh')
  OR (s->'UnidadRef' IS DISTINCT FROM 'null'::jsonb AND (jsonb_typeof(s->'UnidadRef') IS DISTINCT FROM 'string'
   OR s->>'UnidadRef' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' OR s->>'Via' IS DISTINCT FROM 'certificado_vec'))
  OR jsonb_typeof(s->'OrganizacionRef') IS DISTINCT FROM 'string'
  OR s->>'OrganizacionRef' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  OR jsonb_typeof(s->'ExpedienteRef') IS DISTINCT FROM 'string'
  OR s->>'ExpedienteRef' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  OR jsonb_typeof(s->'Documento') IS DISTINCT FROM 'string' OR s->>'Documento' !~ '^[a-z][a-z0-9_]{1,63}$'
  OR jsonb_typeof(s->'FirmantePrincipalCandidatoRef') IS DISTINCT FROM 'string'
  OR s->>'FirmantePrincipalCandidatoRef' !~ '^per_[A-Za-z0-9_-]{2,159}$'
  OR jsonb_typeof(s->'ClaveIdempotencia') IS DISTINCT FROM 'string'
  OR s->>'ClaveIdempotencia' !~ '^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$'
  OR jsonb_typeof(s->'CatalogoHuella') IS DISTINCT FROM 'string' OR s->>'CatalogoHuella' !~ '^[0-9a-f]{64}$'
  OR jsonb_typeof(s->'VersionExpediente') IS DISTINCT FROM 'number' OR s->>'VersionExpediente' !~ '^[1-9][0-9]{0,15}$'
  OR (s->>'VersionExpediente')::numeric>9007199254740991
  OR jsonb_typeof(s->'PasoOrden') IS DISTINCT FROM 'number' OR s->>'PasoOrden' !~ '^[1-3]$'
  OR (s->>'PasoOrden'='3' AND s->'UnidadRef' IS NOT DISTINCT FROM 'null'::jsonb) THEN
  RAISE EXCEPTION 'lectura V2 inválida' USING ERRCODE='22023'; END IF;
 h:=encode(sha256(convert_to(p_solicitud,'UTF8')),'hex');
 contexto_h:=vec_autorizacion_atestada_v3.huella_recurso_consulta_firmas_r5_ct_v1(p_solicitud,p_decision);
 IF s->'UnidadRef' IS DISTINCT FROM 'null'::jsonb AND vec_catalogos_configurables.paso_plan_firma_con_unidad_v1(
   s->>'CatalogoHuella',s->>'Documento',(s->>'PasoOrden')::integer,s->>'OrganizacionRef',s->>'UnidadRef') IS NOT TRUE THEN
  RAISE EXCEPTION 'lectura V2 unidad sin paso en el plan' USING ERRCODE='42501'; END IF;
 IF c->>'efecto_ref' IS DISTINCT FROM s->>'ExpedienteRef'
  OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
  OR d->>'recurso_ref' IS DISTINCT FROM s->>'ExpedienteRef'
  OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
  OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex') THEN
  RAISE EXCEPTION 'lectura V2 divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_consulta_firmas_r5_ct_v2_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM s->>'ExpedienteRef'
  OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_h THEN
  RAISE EXCEPTION 'consumo lectura V2 divergente' USING ERRCODE='42501'; END IF;
 SELECT a.version,vv.agregado_json->>'organizacion_ref' INTO v,org
  FROM vec_contratacion_temporal.expediente_integral_actual a JOIN vec_contratacion_temporal.expediente_version_integral vv
   ON vv.expediente_ref=a.expediente_ref AND vv.version=a.version WHERE a.expediente_ref=s->>'ExpedienteRef' FOR KEY SHARE OF a;
 IF NOT FOUND OR org IS DISTINCT FROM s->>'OrganizacionRef' THEN
  RETURN jsonb_build_object('Encontrado',false,'ExpedienteRef',s->>'ExpedienteRef','Firmas','[]'::jsonb,'RevisionesPDF','[]'::jsonb,
   'HistoriaRevision',0,'HistoriaHuella',vacia,'CoincideFirmanteEnOtroPaso',false,'HistoriaSeparacionAcreditada',false); END IF;
 SELECT * INTO exacta FROM vec_contratacion_temporal.firma_documento_v1 f
  WHERE f.organizacion_ref=s->>'OrganizacionRef' AND f.expediente_ref=s->>'ExpedienteRef' AND f.clave_idempotencia=s->>'ClaveIdempotencia';
 IF FOUND THEN
  IF exacta.documento IS DISTINCT FROM s->>'Documento' OR exacta.expediente_version IS DISTINCT FROM (s->>'VersionExpediente')::numeric
   OR exacta.paso_orden IS DISTINCT FROM (s->>'PasoOrden')::integer OR exacta.catalogo_huella_sha256 IS DISTINCT FROM s->>'CatalogoHuella'
   OR exacta.politica_verificacion IS DISTINCT FROM 'politica:vec:firma:verificacion-autonoma:v2' THEN
   RAISE EXCEPTION 'recuperación lectura V2 divergente' USING ERRCODE='P1181'; END IF;
 ELSE
  IF v IS DISTINCT FROM (s->>'VersionExpediente')::numeric THEN
   RETURN jsonb_build_object('Encontrado',false,'ExpedienteRef',s->>'ExpedienteRef','Firmas','[]'::jsonb,'RevisionesPDF','[]'::jsonb,
    'HistoriaRevision',0,'HistoriaHuella',vacia,'CoincideFirmanteEnOtroPaso',false,'HistoriaSeparacionAcreditada',false); END IF;
 END IF;
 SELECT revision,huella_sha256 INTO cabeza FROM vec_contratacion_temporal.firma_historia_cabeza_v1
  WHERE organizacion_ref=s->>'OrganizacionRef' AND expediente_ref=s->>'ExpedienteRef';
 IF NOT FOUND THEN cabeza.revision:=0; cabeza.huella_sha256:=vacia; END IF;
 IF exacta.firma_ref IS NOT NULL THEN cabeza.revision:=exacta.historia_revision_observada; cabeza.huella_sha256:=exacta.historia_huella_observada; END IF;
 IF (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_v1 f WHERE f.organizacion_ref=s->>'OrganizacionRef'
   AND f.expediente_ref=s->>'ExpedienteRef' AND f.documento=s->>'Documento')>1000 THEN
  RAISE EXCEPTION 'historia V2 no representable' USING ERRCODE='P1525'; END IF;
 -- El lector privado común produce la proyección mínima, sin identidad de tercero.
 firmas:=vec_contratacion_temporal.consultar_firmas_documento_v3(s->>'OrganizacionRef',s->>'ExpedienteRef',s->>'Documento',s->>'FirmantePrincipalCandidatoRef',NULL);
 -- Recuperar una operación histórica no revela actos posteriores al propio recibo.
 SELECT coalesce(jsonb_agg(x.value ORDER BY f.secuencia),'[]'::jsonb) INTO firmas
 FROM jsonb_array_elements(firmas) x JOIN vec_contratacion_temporal.firma_documento_v1 f ON f.firma_ref=x.value->>'FirmaRef'
 WHERE f.expediente_version<=(s->>'VersionExpediente')::numeric
   AND (exacta.firma_ref IS NULL OR f.secuencia<=exacta.secuencia);
 SELECT coalesce(jsonb_agg(x.value||jsonb_build_object('FirmanteRef',f.firmante_ref,'CertificadoHuella',f.certificado_huella_sha256,
   'FirmaAnteriorRef',r.firma_anterior_ref,'ReciboAnteriorRef',r.recibo_anterior_ref,
   'EntradaDocumentoRef',r.entrada_documento_ref,'EntradaDocumentoVersion',r.entrada_documento_version,
   'EntradaDocumentoHuella',r.entrada_huella_sha256,'EntradaDocumentoLongitud',r.entrada_longitud,
   'OrdenFirmaPDF',r.orden_pdf,'ByteRange',r.byte_range,'RevisionHuellaSHA256',r.revision_huella_sha256,
   'ContenidoFirmadoHuellaSHA256',r.contenido_firmado_huella_sha256,'RevisionLongitud',r.revision_longitud,
   'EvidenciaFirmasCanonica',r.evidencia_firmas_texto,'EvidenciaFirmasHuellaSHA256',r.evidencia_firmas_huella_sha256)
   ORDER BY f.secuencia),'[]'::jsonb) INTO revisiones
 FROM jsonb_array_elements(firmas) x JOIN vec_contratacion_temporal.firma_documento_v1 f ON f.firma_ref=x.value->>'FirmaRef'
 JOIN vec_contratacion_temporal.firma_documento_revision_pdf_v2 r ON r.firma_ref=f.firma_ref;
 SELECT EXISTS(SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1 f
  WHERE f.organizacion_ref=s->>'OrganizacionRef' AND f.expediente_ref=s->>'ExpedienteRef' AND f.resultado='firmado'
   AND ROW(f.documento,f.paso_orden,f.catalogo_huella_sha256) IS DISTINCT FROM ROW(s->>'Documento',(s->>'PasoOrden')::integer,s->>'CatalogoHuella')
   AND f.firmante_principal_ref=s->>'FirmantePrincipalCandidatoRef'),
  NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1 f
  WHERE f.organizacion_ref=s->>'OrganizacionRef' AND f.expediente_ref=s->>'ExpedienteRef' AND f.resultado='firmado'
   AND ROW(f.documento,f.paso_orden,f.catalogo_huella_sha256) IS DISTINCT FROM ROW(s->>'Documento',(s->>'PasoOrden')::integer,s->>'CatalogoHuella')
   AND (f.firmante_principal_ref IS NULL OR f.catalogo_huella_sha256 IS DISTINCT FROM s->>'CatalogoHuella')) INTO coincide,separacion;
 RETURN jsonb_build_object('Encontrado',true,'ExpedienteRef',s->>'ExpedienteRef','Firmas',firmas,'RevisionesPDF',revisiones,
  'HistoriaRevision',cabeza.revision,'HistoriaHuella',cabeza.huella_sha256,
  'CoincideFirmanteEnOtroPaso',coincide,'HistoriaSeparacionAcreditada',separacion);
EXCEPTION WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN
 RAISE EXCEPTION 'consulta V2 transitoria' USING ERRCODE='P1525';
WHEN data_exception THEN
 RAISE EXCEPTION 'material de consulta V2 inválido' USING ERRCODE='22023';
END $function$;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_firmas_r5_atestadas_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_firmas_r5_atestadas_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;

DO $post$
DECLARE f oid:=to_regprocedure('vec_contratacion_temporal.consultar_firmas_r5_atestadas_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF f IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner=to_regrole('vec_contratacion_temporal_propietario') AND p.prosecdef
   AND p.provolatile='v' AND p.proparallel='u'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[] AND p.proacl=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario','vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='d71de0dbe2a8677c4d24f7058886a3309ab1262bd0af6b9e6f170933a1210fd4'
   AND (length(p.prosrc)-length(replace(p.prosrc,'^[1-3]$','')))=length('^[1-3]$')
   AND (length(p.prosrc)-length(replace(p.prosrc,$marker$s->>'PasoOrden'='3' AND s->'UnidadRef' IS NOT DISTINCT FROM 'null'::jsonb$marker$,'')))
       =length($marker$s->>'PasoOrden'='3' AND s->'UnidadRef' IS NOT DISTINCT FROM 'null'::jsonb$marker$))
 THEN RAISE EXCEPTION 'CT224: posimagen, propietario o ACL distintos' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
