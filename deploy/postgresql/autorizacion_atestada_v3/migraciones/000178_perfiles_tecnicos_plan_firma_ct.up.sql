\set ON_ERROR_STOP on
-- AD178: perfiles técnicos fijos de recuperación R5 de 48 campos y de gobierno
-- del plan nominal de firma CT. Preimágenes medidas el 05/10/2026 sobre la
-- principal posterior a AD195/AD196 (clon H10-30 + AD194/IS16/CA36/AUT47/AD193/AD195/P36/AD196/AUT49/AUT48).
-- Una sola vez; sin DOWN. Orden: AD193 → AD195/AD196 → AD178 → AD177 → CC7.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000178',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $delta$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 -- Medidas sobre el núcleo y el CHECK posteriores a AD195/AD196 (pg_get_constraintdef(...,false)).
 esperada_def text:='728dde660bd784951e6685402a625f62dedc6d08cff35d3e259a1d9af471737a';
 esperada_src text:='717eba51bc117748907f46dbf9ad1341b53a1aeb1896745b9561eebc3f6d189c';
 esperada_audiencias text:='26497f113bb8468042bffa3fffaf846ce9da5d6db289f0d3b48a0f011703d5f0';
 esperada_post_def text:='2ccd704afe6140d604faa626631e9743edda8f785517d1136054c746cb9b1381';
 esperada_post_src text:='4729b6666065a8a3582443d803bf8535940f0eae650b27a315aca260f3183b8b';
 esperada_post_audiencias text:='e76428d2ecd1c79da88a827cf33138f5c75026ed2138858c67f93420e932eae7';
 original text;fuente text;nueva text;actual text;audiencias text;audiencias_nuevas text;
 def_sha text;src_sha text;aud_sha text;meta jsonb;deps jsonb;compartidas jsonb;
 audiencia text:='vec_contratacion_temporal.firmas_r5.recuperar.v2';
 audiencia_gobierno text:='vec_catalogos_configurables.plan_nominal_firma.gobierno.v1';
 ancla text:=$ancla$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_firmas_r5_ct_v2'$ancla$;
 extension text:=$extension$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'recuperacion_firmas_r5_ct_v2'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.documento.firmas_r5_v2.recuperar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.firmas_r5.recuperar.v2'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'expediente_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT NULL
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["ByteRange","CanonNominal","CanonNominalRef","CanonNominalSHA256","CatalogoHuella","CatalogoRef","CertificadoHuella","ClaveIdempotencia","CoincideFirmanteCandidato","CoincideFirmanteEnOtroPaso","ConMotivoDevolucion","ContenidoFirmadoHuellaSHA256","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","EntradaDocumentoHuella","EntradaDocumentoLongitud","EntradaDocumentoRef","EntradaDocumentoVersion","EvidenciaFirmasCanonica","EvidenciaFirmasHuellaSHA256","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaAnteriorRef","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","FirmanteRef","HistoriaHuella","HistoriaRevision","HistoriaSeparacionAcreditada","MaterialRootSHA256","OrdenFirmaPDF","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboAnteriorRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","RevisionHuellaSHA256","RevisionLongitud","Secuencia","SelloTiempoEstado","Via"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$extension$;
 gobierno text:=$gobierno$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'gobierno_plan_nominal_firma_ct'
 AND (c->>'operacion' IN ('vec.catalogos.crear','vec.catalogos.actualizar','vec.catalogos.publicar','vec.catalogos.retirar')) IS TRUE
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'catalogo_configurable'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT NULL
 AND (d->>'recurso_ref' ~ '^[a-z][a-z0-9._-]{2,127}:[1-9][0-9]{0,9}$') IS TRUE
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$gobierno$;
BEGIN
 IF esperada_def IS NULL OR esperada_src IS NULL OR esperada_audiencias IS NULL
 OR esperada_post_def IS NULL OR esperada_post_src IS NULL OR esperada_post_audiencias IS NULL THEN
  RAISE EXCEPTION 'AD178: PARO clave=pre_post_aprobadas actual=NULL esperado=medidas_post_AD195_AD196' USING ERRCODE='55000';
 END IF;
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR current_user<>'vec_autorizacion_atestada_v3_propietario' OR getdatabaseencoding()<>'UTF8'
 OR f IS NULL OR to_regprocedure('vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(text,text,text)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_recuperacion_firmas_r5_ct_v2_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
  RAISE EXCEPTION 'AD178: PARO clave=objetos actual=incompatible esperado=post_AD196_sin_AD178' USING ERRCODE='55000';
 END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 def_sha:=encode(sha256(convert_to(original,'UTF8')),'hex');src_sha:=encode(sha256(convert_to(fuente,'UTF8')),'hex');
 IF def_sha IS DISTINCT FROM esperada_def OR src_sha IS DISTINCT FROM esperada_src THEN
  RAISE EXCEPTION 'AD178: PARO clave=nucleo_sha256 actual=%/% esperado=%/%',def_sha,src_sha,esperada_def,esperada_src USING ERRCODE='55000';
 END IF;
 IF (meta->>'proowner')::oid IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
 OR (meta->>'prosecdef')::boolean IS NOT TRUE
 OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee=p.proowner AND a.grantor=p.proowner AND a.privilege_type='EXECUTE' AND NOT a.is_grantable) THEN
  RAISE EXCEPTION 'AD178: PARO clave=nucleo_metadatos actual=incompatible esperado=propietario_config_ACL_privados' USING ERRCODE='55000';
 END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO compartidas FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
 AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database());
 LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT audiencias FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 aud_sha:=encode(sha256(convert_to(audiencias,'UTF8')),'hex');
 IF aud_sha IS DISTINCT FROM esperada_audiencias THEN
  RAISE EXCEPTION 'AD178: PARO clave=audiencias_sha256 actual=% esperado=%',aud_sha,esperada_audiencias USING ERRCODE='55000';
 END IF;
 IF length(original)-length(replace(original,ancla,''))<>length(ancla)
 OR strpos(original,'''recuperacion_firmas_r5_ct_v2''')<>0
 OR strpos(original,'''gobierno_plan_nominal_firma_ct''')<>0 THEN
  RAISE EXCEPTION 'AD178: PARO clave=ancla_recuperacion actual=incompatible esperado=una_consulta44_sin_recuperacion48_gobierno' USING ERRCODE='55000';
 END IF;
 -- El runtime CT es el mismo de la consulta R5; no se cambia su clasificación.
 nueva:=replace(original,ancla,extension||gobierno||ancla);
 IF encode(sha256(convert_to(nueva,'UTF8')),'hex') IS DISTINCT FROM esperada_post_def THEN
  RAISE EXCEPTION 'AD178: PARO clave=post_def_sha256 actual=% esperado=%',encode(sha256(convert_to(nueva,'UTF8')),'hex'),esperada_post_def USING ERRCODE='55000';
 END IF;
 -- Mismo patrón que AD185: se envuelve el CHECK vigente y se añade un OR.
 IF left(audiencias,7)<>'CHECK (' OR right(audiencias,1)<>')'
 OR strpos(audiencias,quote_literal(audiencia))<>0 OR strpos(audiencias,quote_literal(audiencia_gobierno))<>0 THEN
  RAISE EXCEPTION 'AD178: PARO clave=forma_audiencias actual=incompatible esperado=CHECK_sin_recuperacion_gobierno' USING ERRCODE='55000';
 END IF;
 audiencias_nuevas:='CHECK (('||substr(audiencias,8,length(audiencias)-8)||') OR audiencia_consumo IN ('||quote_literal(audiencia)||','||quote_literal(audiencia_gobierno)||'))';
 EXECUTE nueva;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nueva OR replace(actual,extension||gobierno||ancla,ancla) IS DISTINCT FROM original
 OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM esperada_post_src
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb) FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())) IS DISTINCT FROM compartidas THEN
  RAISE EXCEPTION 'AD178: PARO clave=delta_metadatos actual=divergente esperado=solo_OR_recuperacion48_gobierno' USING ERRCODE='55000';
 END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||audiencias_nuevas;
 -- PostgreSQL deparsa IN como ANY: se coteja la postimagen medida, no el texto enviado.
 IF (SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,false),'UTF8')),'hex') FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated) IS DISTINCT FROM esperada_post_audiencias THEN
  RAISE EXCEPTION 'AD178: PARO clave=delta_audiencias actual=divergente esperado=solo_recuperacion_v2_gobierno_CT' USING ERRCODE='55000';
 END IF;
END $delta$;

-- Sólo consume una autorización NUEVA; K conserva la autoridad de los bytes
-- históricos. Esta fachada no lee historia nominal ni revalida el cargo antiguo.
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_recuperacion_firmas_r5_ct_v2_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE c jsonb;d jsonb;x record;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR pg_is_in_recovery() OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 2 AND 32768
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288 THEN
  RAISE EXCEPTION 'AD178: recuperación nominal denegada' USING ERRCODE='42501';
 END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'AD178: material de recuperación inválido' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.firmas_r5.recuperar.v2'
 OR c->>'operacion' IS DISTINCT FROM 'contratacion_temporal.documento.firmas_r5_v2.recuperar'
 OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'expediente_contratacion_temporal'
 OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
 OR d->>'recurso_ref' IS NULL OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
 OR d->>'contexto_recurso_huella_sha256' IS NULL OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
 OR d->'campos_permitidos' IS DISTINCT FROM '["ByteRange","CanonNominal","CanonNominalRef","CanonNominalSHA256","CatalogoHuella","CatalogoRef","CertificadoHuella","ClaveIdempotencia","CoincideFirmanteCandidato","CoincideFirmanteEnOtroPaso","ConMotivoDevolucion","ContenidoFirmadoHuellaSHA256","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","EntradaDocumentoHuella","EntradaDocumentoLongitud","EntradaDocumentoRef","EntradaDocumentoVersion","EvidenciaFirmasCanonica","EvidenciaFirmasHuellaSHA256","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaAnteriorRef","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","FirmanteRef","HistoriaHuella","HistoriaRevision","HistoriaSeparacionAcreditada","MaterialRootSHA256","OrdenFirmaPDF","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboAnteriorRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","RevisionHuellaSHA256","RevisionLongitud","Secuencia","SelloTiempoEstado","Via"]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'AD178: recuperación nominal denegada' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
 'recuperacion_firmas_r5_ct_v2',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
 p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'AD178: recuperación exige autorización nueva' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_recuperacion_firmas_r5_ct_v2_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_recuperacion_firmas_r5_ct_v2_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_propietario;
DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.consumir_recuperacion_firmas_r5_ct_v2_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND (a.grantee NOT IN (p.proowner,'vec_contratacion_temporal_propietario'::regrole)
  OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable)) THEN
  RAISE EXCEPTION 'AD178: PARO clave=ACL_fachada actual=divergente esperado=AUT_owner_y_CT_owner_EXECUTE' USING ERRCODE='55000';
 END IF;
END $acl$;

-- AUT41 recibe sólo el ámbito acreditado por un consumo de recuperación nuevo.
-- No recibe SELECT de tablas AD ni una identidad derivada del selector histórico.
CREATE FUNCTION vec_autorizacion_atestada_v3.comprobar_consumo_recuperacion_firmas_ct_v2(
 p_material_consulta bytea,p_consumo jsonb
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE r record;s jsonb;original json;material_h text;contexto_h text;
 c jsonb;d jsonb;ahora timestamptz(6);version_actual numeric;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 OR pg_is_in_recovery() THEN
  RAISE EXCEPTION 'AD178: PARO clave=transaccion actual=incompatible esperado=serializable_escritura_UTC' USING ERRCODE='42501';
 END IF;
 IF p_material_consulta IS NULL OR octet_length(p_material_consulta) NOT BETWEEN 2 AND 4096 THEN
  RAISE EXCEPTION 'AD178: PARO clave=material actual=invalido esperado=consulta_R5_V2' USING ERRCODE='42501';
 END IF;
 original:=convert_from(p_material_consulta,'UTF8')::json;s:=original::jsonb;
 IF jsonb_typeof(s) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(s))<>10
 OR NOT (s ?& ARRAY['CatalogoHuella','ClaveIdempotencia','Documento','ExpedienteRef',
   'FirmantePrincipalCandidatoRef','OrganizacionRef','PasoOrden','UnidadRef','VersionExpediente','Via'])
 OR (SELECT count(*) FROM json_each(original))<>10
 OR jsonb_typeof(s->'Via') IS DISTINCT FROM 'string'
 OR (s->>'Via' IN('certificado_vec','portafirmas_registro_rrhh')) IS NOT TRUE
 OR s->'UnidadRef' IS DISTINCT FROM 'null'::jsonb
 OR jsonb_typeof(s->'OrganizacionRef') IS DISTINCT FROM 'string'
 OR (s->>'OrganizacionRef' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$') IS NOT TRUE
 OR jsonb_typeof(s->'ExpedienteRef') IS DISTINCT FROM 'string'
 OR (s->>'ExpedienteRef' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$') IS NOT TRUE
 OR jsonb_typeof(s->'Documento') IS DISTINCT FROM 'string'
 OR (s->>'Documento' ~ '^[a-z][a-z0-9_]{1,63}$') IS NOT TRUE
 OR jsonb_typeof(s->'FirmantePrincipalCandidatoRef') IS DISTINCT FROM 'string'
 OR (s->>'FirmantePrincipalCandidatoRef' ~ '^per_[A-Za-z0-9_-]{2,159}$') IS NOT TRUE
 OR jsonb_typeof(s->'ClaveIdempotencia') IS DISTINCT FROM 'string'
 OR (s->>'ClaveIdempotencia' ~ '^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$') IS NOT TRUE
 OR jsonb_typeof(s->'CatalogoHuella') IS DISTINCT FROM 'string'
 OR (s->>'CatalogoHuella' ~ '^[0-9a-f]{64}$') IS NOT TRUE
 OR jsonb_typeof(s->'VersionExpediente') IS DISTINCT FROM 'number'
 OR (s->>'VersionExpediente' ~ '^[1-9][0-9]{0,15}$') IS NOT TRUE
 OR jsonb_typeof(s->'PasoOrden') IS DISTINCT FROM 'number'
 OR (s->>'PasoOrden' ~ '^[12]$') IS NOT TRUE THEN
  RAISE EXCEPTION 'AD178: PARO clave=material actual=invalido esperado=consulta_R5_V2_exacta' USING ERRCODE='42501';
 END IF;
 version_actual:=(s->>'VersionExpediente')::numeric;
 IF version_actual>9007199254740991 THEN
  RAISE EXCEPTION 'AD178: PARO clave=version actual=fuera_rango esperado=entera_segura' USING ERRCODE='42501';
 END IF;
 material_h:=encode(sha256(p_material_consulta),'hex');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
  '"},"atributos":{"material_sha256":"'||material_h||'"}}','UTF8')),'hex');
 IF jsonb_typeof(p_consumo) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_consumo))<>7
 OR NOT (p_consumo ?& ARRAY['decision_ref','efecto_ref','huella_efecto_sha256',
  'consumo_huella_sha256','auditoria_ref','consumida_en','consumo_nuevo'])
 OR p_consumo->'consumo_nuevo' IS DISTINCT FROM 'true'::jsonb
 OR EXISTS(SELECT 1 FROM jsonb_object_keys(p_consumo) AS k(nombre)
  WHERE k.nombre<>'consumo_nuevo' AND jsonb_typeof(p_consumo->k.nombre) IS DISTINCT FROM 'string')
 -- PostgreSQL limita {m,n} a 255: la longitud se comprueba aparte.
 OR (p_consumo->>'decision_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]*$') IS NOT TRUE OR length(p_consumo->>'decision_ref') NOT BETWEEN 3 AND 512
 OR (p_consumo->>'auditoria_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]*$') IS NOT TRUE OR length(p_consumo->>'auditoria_ref') NOT BETWEEN 3 AND 512
 OR (p_consumo->>'consumo_huella_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
 OR p_consumo->>'efecto_ref' IS DISTINCT FROM s->>'ExpedienteRef'
 OR p_consumo->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h THEN
  RAISE EXCEPTION 'AD178: PARO clave=consumo actual=invalido esperado=recuperacion_nueva_ligada' USING ERRCODE='42501';
 END IF;
 SELECT a.decision_ref,a.efecto_ref,a.huella_efecto_sha256,a.huella_decision_sha256 AS consumo_decision_sha256,
  a.consumo_huella_sha256,a.consumida_en,u.auditoria_ref,u.registrada_en,u.tipo_registro,u.version_consumo,
  t.huella_decision_sha256,t.huella_capacidad_sha256,t.decision_canonica,t.capacidad_canonica,
  a.transaccion_origen AS consumo_origen,u.transaccion_origen AS auditoria_origen INTO STRICT r
 FROM vec_autorizacion_atestada_v3.consumo_decision_v3 a
 JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 u
  ON u.decision_ref=a.decision_ref AND u.efecto_ref=a.efecto_ref AND u.huella_efecto_sha256=a.huella_efecto_sha256
 JOIN vec_autorizacion_atestada_v3.atestacion_decision_v3 t
  ON t.decision_ref=a.decision_ref AND t.efecto_ref=a.efecto_ref AND t.huella_efecto_sha256=a.huella_efecto_sha256
 WHERE a.decision_ref=p_consumo->>'decision_ref' FOR SHARE OF a,u,t;
 ahora:=clock_timestamp();
 IF r.efecto_ref IS DISTINCT FROM s->>'ExpedienteRef'
 OR r.huella_efecto_sha256 IS DISTINCT FROM contexto_h
 OR r.consumo_huella_sha256 IS DISTINCT FROM p_consumo->>'consumo_huella_sha256'
 OR r.auditoria_ref IS DISTINCT FROM p_consumo->>'auditoria_ref'
 OR r.consumida_en IS DISTINCT FROM (p_consumo->>'consumida_en')::timestamptz
 OR r.registrada_en IS DISTINCT FROM r.consumida_en OR r.consumida_en>ahora
 -- Sello AD193: ambos TopXID completos de esta transacción (xid8, con época).
 OR r.consumo_origen IS DISTINCT FROM pg_current_xact_id()
 OR r.auditoria_origen IS DISTINCT FROM pg_current_xact_id()
 OR r.tipo_registro IS DISTINCT FROM 'consumo_confirmado_v4' OR r.version_consumo IS DISTINCT FROM 4
 OR r.huella_decision_sha256 IS DISTINCT FROM encode(sha256(r.decision_canonica),'hex')
 OR r.consumo_decision_sha256 IS DISTINCT FROM r.huella_decision_sha256
 OR r.huella_capacidad_sha256 IS DISTINCT FROM encode(sha256(r.capacidad_canonica),'hex') THEN
  RAISE EXCEPTION 'AD178: PARO clave=filas actual=no_acreditadas esperado=consumo_auditoria_TX_actual' USING ERRCODE='42501';
 END IF;
 d:=convert_from(r.decision_canonica,'UTF8')::jsonb;c:=convert_from(r.capacidad_canonica,'UTF8')::jsonb;
 IF jsonb_typeof(d) IS DISTINCT FROM 'object' OR jsonb_typeof(c) IS DISTINCT FROM 'object'
 OR d->>'decision_ref' IS DISTINCT FROM r.decision_ref OR d->'concedida' IS DISTINCT FROM 'true'::jsonb
 OR d->>'codigo' IS DISTINCT FROM 'concedida'
 OR d->>'accion' IS DISTINCT FROM 'contratacion_temporal.documento.firmas_r5_v2.recuperar'
 OR c->>'operacion' IS DISTINCT FROM d->>'accion'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.firmas_r5.recuperar.v2'
 OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'expediente_contratacion_temporal'
 OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
 OR d->>'recurso_ref' IS DISTINCT FROM s->>'ExpedienteRef' OR c->>'efecto_ref' IS DISTINCT FROM s->>'ExpedienteRef'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
 OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
 OR c->>'huella_decision_sha256' IS DISTINCT FROM r.huella_decision_sha256
 OR d->'campos_permitidos' IS DISTINCT FROM '["ByteRange","CanonNominal","CanonNominalRef","CanonNominalSHA256","CatalogoHuella","CatalogoRef","CertificadoHuella","ClaveIdempotencia","CoincideFirmanteCandidato","CoincideFirmanteEnOtroPaso","ConMotivoDevolucion","ContenidoFirmadoHuellaSHA256","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","EntradaDocumentoHuella","EntradaDocumentoLongitud","EntradaDocumentoRef","EntradaDocumentoVersion","EvidenciaFirmasCanonica","EvidenciaFirmasHuellaSHA256","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaAnteriorRef","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","FirmanteRef","HistoriaHuella","HistoriaRevision","HistoriaSeparacionAcreditada","MaterialRootSHA256","OrdenFirmaPDF","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboAnteriorRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","RevisionHuellaSHA256","RevisionLongitud","Secuencia","SelloTiempoEstado","Via"]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 OR jsonb_typeof(d->'principal_id') IS DISTINCT FROM 'string' OR d->>'principal_id'=''
 OR jsonb_typeof(d->'perfil_activo_ref') IS DISTINCT FROM 'string' OR d->>'perfil_activo_ref'=''
 OR jsonb_typeof(d->'emitida_en') IS DISTINCT FROM 'string'
 OR jsonb_typeof(d->'valida_hasta') IS DISTINCT FROM 'string'
 OR (d->>'emitida_en')::timestamptz>ahora OR (d->>'valida_hasta')::timestamptz<=ahora THEN
  RAISE EXCEPTION 'AD178: PARO clave=capacidad actual=no_acreditada esperado=recuperacion_R5_48_vigente' USING ERRCODE='42501';
 END IF;
 IF (d->>'valida_hasta')::timestamptz<=clock_timestamp() THEN
  RAISE EXCEPTION 'AD178: PARO clave=vigencia actual=caducada esperado=vigente_al_retornar' USING ERRCODE='42501';
 END IF;
 RETURN jsonb_build_object('organizacion_ref',s->>'OrganizacionRef','expediente_ref',s->>'ExpedienteRef',
  'documento',s->>'Documento','version_expediente',version_actual,'decision_ref',r.decision_ref,
  'consumo_huella_sha256',r.consumo_huella_sha256,'auditoria_ref',r.auditoria_ref,
  'decision_valida_hasta',d->>'valida_hasta');
EXCEPTION WHEN data_exception OR no_data_found OR too_many_rows THEN
 RAISE EXCEPTION 'AD178: PARO clave=material_o_filas actual=incompatible esperado=consumo_ligado_actual' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.comprobar_consumo_recuperacion_firmas_ct_v2(bytea,jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.comprobar_consumo_recuperacion_firmas_ct_v2(bytea,jsonb) TO vec_autorizacion_propietario;
DO $acl_helper$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.comprobar_consumo_recuperacion_firmas_ct_v2(bytea,jsonb)'::regprocedure;
 owner_id oid:='vec_autorizacion_atestada_v3_propietario'::regrole;
 aut_id oid:='vec_autorizacion_propietario'::regrole;
BEGIN
 IF (SELECT p.proowner FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM owner_id
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND (a.grantee NOT IN(owner_id,aut_id) OR a.grantor<>owner_id
   OR a.privilege_type<>'EXECUTE' OR (a.grantee=aut_id AND a.is_grantable)))
 OR NOT has_function_privilege(aut_id,f,'EXECUTE') THEN
  RAISE EXCEPTION 'AD178: PARO clave=ACL_comprobador actual=divergente esperado=solo_AD_owner_AUT_owner' USING ERRCODE='55000';
 END IF;
END $acl_helper$;

COMMIT;
