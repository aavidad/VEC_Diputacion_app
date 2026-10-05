\set ON_ERROR_STOP on
-- CT175: recuperación nominal nueva; no sustituye ni reaplica CT172.
-- AD178 y AUT41 deben estar instaladas causalmente antes de este borrador.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL TimeZone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000175',0));
DO $pre$
DECLARE nombre text; f regprocedure;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
  OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR to_regprocedure('vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
  RAISE EXCEPTION 'CT175: PARO clave=preimagen actual=incompatible esperado=PG18_CT172_sin_CT175' USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY ARRAY[
  'vec_autorizacion_atestada_v3.consumir_recuperacion_firmas_r5_ct_v2_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_autorizacion.recuperar_canon_historico_firmante_ct_v1(jsonb,bytea,jsonb)',
  'vec_contratacion_temporal.consultar_firmas_documento_v3(text,text,text,text,text)'] LOOP
  f:=to_regprocedure(nombre);
  IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE') THEN
   RAISE EXCEPTION 'CT175: PARO clave=dependencia actual=%_ausente_o_sin_EXECUTE esperado=AD178_AUT41_CT170',nombre USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH nombre IN ARRAY ARRAY['firma_documento_v1','firma_documento_revision_pdf_v2','firma_documento_custodia_v1','firma_historia_cabeza_v1'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid=to_regclass('vec_contratacion_temporal.'||nombre)
   AND c.relowner=current_user::regrole AND c.relrowsecurity AND c.relforcerowsecurity) THEN
   RAISE EXCEPTION 'CT175: PARO clave=tabla actual=%_incompatible esperado=CT_propietario_RLS',nombre USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;
CREATE FUNCTION vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v2(
 p_solicitud text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE s jsonb; c jsonb; d jsonb; consumo record; exacta record; fila record; revision record; cabeza record; firmas jsonb:='[]'::jsonb; revisiones jsonb:='[]'::jsonb; recuperaciones jsonb:='[]'::jsonb; proyeccion jsonb; selector jsonb; canon bytea;
 h text; contexto_h text; v numeric; org text; coincide boolean; separacion boolean; total integer:=0;
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
 BEGIN
  s:=p_solicitud::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'material de recuperación inválido' USING ERRCODE='22023';
 END;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(s,ARRAY['OrganizacionRef','ExpedienteRef','VersionExpediente',
   'Documento','FirmantePrincipalCandidatoRef','ClaveIdempotencia','PasoOrden','CatalogoHuella','Via','UnidadRef']) IS NOT TRUE
  OR (SELECT count(*) FROM json_each(p_solicitud::json))<>(SELECT count(*) FROM jsonb_each(s))
  OR jsonb_typeof(s->'Via') IS DISTINCT FROM 'string' OR s->>'Via' NOT IN('certificado_vec','portafirmas_registro_rrhh')
  OR s->'UnidadRef' IS DISTINCT FROM 'null'::jsonb
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
  OR jsonb_typeof(s->'PasoOrden') IS DISTINCT FROM 'number' OR s->>'PasoOrden' !~ '^[12]$' THEN
  RAISE EXCEPTION 'lectura V2 inválida' USING ERRCODE='22023'; END IF;
 h:=encode(sha256(convert_to(p_solicitud,'UTF8')),'hex');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
  '"},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 IF c->>'efecto_ref' IS DISTINCT FROM s->>'ExpedienteRef'
  OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
  OR d->>'recurso_ref' IS DISTINCT FROM s->>'ExpedienteRef'
  OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
  OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex') THEN
  RAISE EXCEPTION 'lectura V2 divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_recuperacion_firmas_r5_ct_v2_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM s->>'ExpedienteRef'
  OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_h THEN
  RAISE EXCEPTION 'consumo lectura V2 divergente' USING ERRCODE='42501'; END IF;
 SELECT a.version,vv.agregado_json->>'organizacion_ref' INTO v,org
  FROM vec_contratacion_temporal.expediente_integral_actual a JOIN vec_contratacion_temporal.expediente_version_integral vv
   ON vv.expediente_ref=a.expediente_ref AND vv.version=a.version WHERE a.expediente_ref=s->>'ExpedienteRef' FOR KEY SHARE OF a;
 IF d->>'valida_hasta' IS NULL OR (d->>'valida_hasta')::timestamptz<=clock_timestamp() THEN
  RAISE EXCEPTION 'consulta de recuperación caducada' USING ERRCODE='42501'; END IF;
 IF NOT FOUND OR org IS DISTINCT FROM s->>'OrganizacionRef' THEN
  RETURN jsonb_build_object('Encontrado',false,'ExpedienteRef',s->>'ExpedienteRef','Firmas','[]'::jsonb,'RevisionesPDF','[]'::jsonb,'Recuperaciones','[]'::jsonb,
   'HistoriaRevision',0,'HistoriaHuella',vacia,'CoincideFirmanteEnOtroPaso',false,'HistoriaSeparacionAcreditada',false); END IF;
 SELECT * INTO exacta FROM vec_contratacion_temporal.firma_documento_v1 f
  WHERE f.organizacion_ref=s->>'OrganizacionRef' AND f.expediente_ref=s->>'ExpedienteRef' AND f.clave_idempotencia=s->>'ClaveIdempotencia';
 IF FOUND THEN
  IF exacta.documento IS DISTINCT FROM s->>'Documento' OR exacta.expediente_version IS DISTINCT FROM (s->>'VersionExpediente')::numeric
   OR exacta.paso_orden IS DISTINCT FROM (s->>'PasoOrden')::integer OR exacta.catalogo_huella_sha256 IS DISTINCT FROM s->>'CatalogoHuella'
   OR exacta.via_registro IS DISTINCT FROM s->>'Via'
   OR exacta.politica_verificacion IS DISTINCT FROM 'politica:vec:firma:verificacion-autonoma:v2'
   OR NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.firma_documento_revision_pdf_v2 r WHERE r.firma_ref=exacta.firma_ref) THEN
   RAISE EXCEPTION 'recuperación lectura V2 divergente' USING ERRCODE='P1181'; END IF;
 ELSE
  IF d->>'valida_hasta' IS NULL OR (d->>'valida_hasta')::timestamptz<=clock_timestamp() THEN
   RAISE EXCEPTION 'consulta de recuperación caducada' USING ERRCODE='42501'; END IF;
  IF v IS DISTINCT FROM (s->>'VersionExpediente')::numeric THEN
   RETURN jsonb_build_object('Encontrado',false,'ExpedienteRef',s->>'ExpedienteRef','Firmas','[]'::jsonb,'RevisionesPDF','[]'::jsonb,'Recuperaciones','[]'::jsonb,
    'HistoriaRevision',0,'HistoriaHuella',vacia,'CoincideFirmanteEnOtroPaso',false,'HistoriaSeparacionAcreditada',false); END IF;
 END IF;
 -- Columnas calificadas: «revision» es también una variable de esta función.
 SELECT hc.revision,hc.huella_sha256 INTO cabeza FROM vec_contratacion_temporal.firma_historia_cabeza_v1 hc
  WHERE hc.organizacion_ref=s->>'OrganizacionRef' AND hc.expediente_ref=s->>'ExpedienteRef';
 IF NOT FOUND THEN cabeza.revision:=0; cabeza.huella_sha256:=vacia; END IF;
 IF exacta.firma_ref IS NOT NULL THEN cabeza.revision:=exacta.historia_revision_observada; cabeza.huella_sha256:=exacta.historia_huella_observada; END IF;

 -- Primero selecciona el corte propio: el helper privado no recibe filas futuras.
 -- El límite coincide con MaximoFilas del servicio Go de recuperación.
 FOR fila IN SELECT f.* FROM vec_contratacion_temporal.firma_documento_v1 f
  WHERE f.organizacion_ref=s->>'OrganizacionRef' AND f.expediente_ref=s->>'ExpedienteRef'
   AND f.documento=s->>'Documento' AND f.expediente_version<=(s->>'VersionExpediente')::numeric
   AND (exacta.firma_ref IS NULL OR f.secuencia<=exacta.secuencia)
  ORDER BY f.secuencia LIMIT 129 LOOP
  total:=total+1;
  IF total>128 THEN RAISE EXCEPTION 'historia recuperada no representable' USING ERRCODE='P1525'; END IF;
  proyeccion:=vec_contratacion_temporal.consultar_firmas_documento_v3(
   s->>'OrganizacionRef',s->>'ExpedienteRef',s->>'Documento',s->>'FirmantePrincipalCandidatoRef',fila.firma_ref);
  IF jsonb_typeof(proyeccion) IS DISTINCT FROM 'array' OR jsonb_array_length(proyeccion)<>1
   OR proyeccion->0->>'FirmaRef' IS DISTINCT FROM fila.firma_ref THEN
   RAISE EXCEPTION 'proyección histórica incoherente' USING ERRCODE='55000'; END IF;
  firmas:=firmas||proyeccion;
  SELECT * INTO revision FROM vec_contratacion_temporal.firma_documento_revision_pdf_v2 r WHERE r.firma_ref=fila.firma_ref;
  IF fila.politica_verificacion='politica:vec:firma:verificacion-autonoma:v2' THEN
   IF NOT FOUND OR revision.competencia_consumo_decision_ref IS DISTINCT FROM fila.decision_ref
    OR revision.competencia_consumo_huella_sha256 IS DISTINCT FROM fila.consumo_huella_sha256
    OR revision.competencia_auditoria_consumo_ref IS DISTINCT FROM fila.auditoria_consumo_ref THEN
    RAISE EXCEPTION 'evidencia V2 histórica no disponible' USING ERRCODE='55000'; END IF;
   selector:=jsonb_build_object('evidencia_ref',revision.competencia_evidencia_ref,
    'canon_sha256',revision.competencia_evidencia_huella_sha256,
    'efecto_original_ref',CASE fila.via_registro WHEN 'certificado_vec' THEN 'operacion-firma-vec-ct:' ELSE 'operacion-firma-externa-ct:' END||fila.clave_idempotencia,
    'organizacion_ref',fila.organizacion_ref,'unidad_ref',fila.unidad_firmante_ref,
    'expediente_ref',fila.expediente_ref,'documento_ref',fila.original_documento_ref,
    'consumo_original_decision_ref',fila.decision_ref,'consumo_original_huella_sha256',fila.consumo_huella_sha256);
   canon:=vec_autorizacion.recuperar_canon_historico_firmante_ct_v1(selector,convert_to(p_solicitud,'UTF8'),to_jsonb(consumo));
   IF canon IS NULL OR octet_length(canon) NOT BETWEEN 512 AND 32768
    OR encode(sha256(canon),'hex') IS DISTINCT FROM revision.competencia_evidencia_huella_sha256 THEN
    RAISE EXCEPTION 'canon V2 histórico incoherente' USING ERRCODE='55000'; END IF;
   recuperaciones:=recuperaciones||jsonb_build_array(jsonb_build_object('FirmaRef',fila.firma_ref,
    'MaterialRootSHA256',fila.solicitud_huella_sha256,'CanonNominal',convert_from(canon,'UTF8'),
    'CanonNominalSHA256',revision.competencia_evidencia_huella_sha256,'CanonNominalRef',revision.competencia_evidencia_ref));
   revisiones:=revisiones||jsonb_build_array(proyeccion->0||jsonb_build_object(
    'FirmanteRef',fila.firmante_ref,'CertificadoHuella',fila.certificado_huella_sha256,
    'FirmaAnteriorRef',revision.firma_anterior_ref,'ReciboAnteriorRef',revision.recibo_anterior_ref,
    'EntradaDocumentoRef',revision.entrada_documento_ref,'EntradaDocumentoVersion',revision.entrada_documento_version,
    'EntradaDocumentoHuella',revision.entrada_huella_sha256,'EntradaDocumentoLongitud',revision.entrada_longitud,
    'OrdenFirmaPDF',revision.orden_pdf,'ByteRange',revision.byte_range,'RevisionHuellaSHA256',revision.revision_huella_sha256,
    'ContenidoFirmadoHuellaSHA256',revision.contenido_firmado_huella_sha256,'RevisionLongitud',revision.revision_longitud,
    'EvidenciaFirmasCanonica',revision.evidencia_firmas_texto,'EvidenciaFirmasHuellaSHA256',revision.evidencia_firmas_huella_sha256));
  ELSIF FOUND THEN
   RAISE EXCEPTION 'revisión V2 sin acto V2' USING ERRCODE='55000';
  END IF;
 END LOOP;
 -- El corte global del recibo protege también los indicadores: una firma
 -- posterior de otro documento no modifica estos booleanos históricos.
 SELECT EXISTS(SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1 f
  WHERE f.organizacion_ref=s->>'OrganizacionRef' AND f.expediente_ref=s->>'ExpedienteRef' AND f.resultado='firmado'
   AND f.expediente_version<=(s->>'VersionExpediente')::numeric
   AND (exacta.firma_ref IS NULL OR f.historia_revision_observada<exacta.historia_revision_observada OR f.firma_ref=exacta.firma_ref)
   AND ROW(f.documento,f.paso_orden,f.catalogo_huella_sha256) IS DISTINCT FROM ROW(s->>'Documento',(s->>'PasoOrden')::integer,s->>'CatalogoHuella')
   AND f.firmante_principal_ref=s->>'FirmantePrincipalCandidatoRef'),
  NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1 f
  WHERE f.organizacion_ref=s->>'OrganizacionRef' AND f.expediente_ref=s->>'ExpedienteRef' AND f.resultado='firmado'
   AND f.expediente_version<=(s->>'VersionExpediente')::numeric
   AND (exacta.firma_ref IS NULL OR f.historia_revision_observada<exacta.historia_revision_observada OR f.firma_ref=exacta.firma_ref)
   AND ROW(f.documento,f.paso_orden,f.catalogo_huella_sha256) IS DISTINCT FROM ROW(s->>'Documento',(s->>'PasoOrden')::integer,s->>'CatalogoHuella')
   AND (f.firmante_principal_ref IS NULL OR f.catalogo_huella_sha256 IS DISTINCT FROM s->>'CatalogoHuella')) INTO coincide,separacion;
 -- Cada posición global previa debe tener una fila observable. CT170
 -- avanzó también por filas legacy sin revisión; no se presume su separación.
 -- DISTINCT impide que dos filas con la misma posición cubran una laguna.
 -- Las filas futuras (incluido legado sin posición) no alteran este corte.
 IF exacta.firma_ref IS NOT NULL AND
  (SELECT count(DISTINCT f.historia_revision_observada)
   FROM vec_contratacion_temporal.firma_documento_v1 f
   WHERE f.organizacion_ref=s->>'OrganizacionRef' AND f.expediente_ref=s->>'ExpedienteRef'
    AND f.historia_revision_observada<exacta.historia_revision_observada)
   IS DISTINCT FROM exacta.historia_revision_observada THEN
  separacion:=false;
 END IF;
 IF (d->>'valida_hasta')::timestamptz<=clock_timestamp() OR d->>'valida_hasta' IS NULL THEN
  RAISE EXCEPTION 'consulta de recuperación caducada' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('Encontrado',true,'ExpedienteRef',s->>'ExpedienteRef','Firmas',firmas,'RevisionesPDF',revisiones,
  'Recuperaciones',recuperaciones,'HistoriaRevision',cabeza.revision,'HistoriaHuella',cabeza.huella_sha256,
  'CoincideFirmanteEnOtroPaso',coincide,'HistoriaSeparacionAcreditada',separacion);
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;
DO $acl$
DECLARE f regprocedure:='vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure; x record;
BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee NOT IN(p.proowner,'vec_contratacion_temporal_ejecutor'::regrole) LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
 END LOOP;
 IF (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
  OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f
   AND (a.grantee NOT IN(p.proowner,'vec_contratacion_temporal_ejecutor'::regrole) OR a.privilege_type<>'EXECUTE' OR a.is_grantable OR a.grantor<>p.proowner)) THEN
  RAISE EXCEPTION 'CT175: PARO clave=ACL actual=incompatible esperado=propietario_ejecutor_EXECUTE' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
