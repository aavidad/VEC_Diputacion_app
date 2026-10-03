\set ON_ERROR_STOP on
-- AD162: consumidores nominales V2 de revisión PDF incremental. Conserva AD156/157.
-- Autoridades de cargos y catálogo se revalidan en CT172/AUT31 antes del efecto.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000162',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
  OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_externa_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
  OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
  OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_verificada_ct_v2_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD162 preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
DO $nucleo$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; meta jsonb; deps jsonb; compartidas jsonb;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 extension text:=$extension$
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'firma_vec_documento_ct_v2'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.documento.firma_vec.registrar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.firma_vec.v2'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'firma_vec_documento_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT NULL
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'firma_externa_documento_ct_v2'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.documento.firma_externa.registrar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.firma_externa.v2'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'firma_externa_documento_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT NULL
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_firmas_r5_ct_v2'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.documento.firmas_r5_v2.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.firmas_r5.consultar.v2'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'expediente_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT NULL
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["ByteRange","CatalogoHuella","CatalogoRef","CertificadoHuella","ClaveIdempotencia","CoincideFirmanteCandidato","CoincideFirmanteEnOtroPaso","ConMotivoDevolucion","ContenidoFirmadoHuellaSHA256","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","EntradaDocumentoHuella","EntradaDocumentoLongitud","EntradaDocumentoRef","EntradaDocumentoVersion","EvidenciaFirmasCanonica","EvidenciaFirmasHuellaSHA256","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaAnteriorRef","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","FirmanteRef","HistoriaHuella","HistoriaRevision","HistoriaSeparacionAcreditada","OrdenFirmaPDF","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboAnteriorRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","RevisionHuellaSHA256","RevisionLongitud","Secuencia","SelloTiempoEstado","Via"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$extension$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO STRICT original,meta FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.classid,x.objid,x.objsubid,x.refclassid,x.refobjid,x.refobjsubid,x.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend x WHERE x.classid='pg_proc'::regclass AND x.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.dbid,x.classid,x.objid,x.objsubid,x.refclassid,x.refobjid,x.deptype),'[]'::jsonb)
 INTO compartidas FROM pg_shdepend x WHERE x.classid='pg_proc'::regclass AND x.objid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM '7c3f6ea9f42633072c3512999c8cfe3c88ff722c8ec0cfc327e7454e9b4558db'
  OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'ca78b04efb396268af1b4e8d8952854fbbd18e0005f31c4d02169c1731d631f4'
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
  OR length(original)-length(replace(original,marca,''))<>length(marca)
  OR strpos(original,'vec_contratacion_temporal_ejecutor')=0
  OR strpos(original,'''firma_externa_documento_ct''')=0
  OR strpos(original,'''firma_vec_documento_ct_v2''')<>0
  OR strpos(original,'''consulta_firmas_r5_ct_v2''')<>0 THEN
  RAISE EXCEPTION 'AD162 núcleo no corresponde a preimagen nominal' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,extension||marca); EXECUTE nuevo;
 IF pg_get_functiondef(f) IS DISTINCT FROM nuevo
  OR replace(pg_get_functiondef(f),extension||marca,marca) IS DISTINCT FROM original
  OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
  OR (SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.classid,x.objid,x.objsubid,x.refclassid,x.refobjid,x.refobjsubid,x.deptype),'[]'::jsonb)
   FROM pg_depend x WHERE x.classid='pg_proc'::regclass AND x.objid=f) IS DISTINCT FROM deps
  OR (SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.dbid,x.classid,x.objid,x.objsubid,x.refclassid,x.refobjid,x.deptype),'[]'::jsonb)
   FROM pg_shdepend x WHERE x.classid='pg_proc'::regclass AND x.objid=f) IS DISTINCT FROM compartidas THEN
  RAISE EXCEPTION 'AD162 núcleo alterado fuera del delta nominal' USING ERRCODE='55000'; END IF;
END $nucleo$;
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; nueva text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT d FROM pg_constraint c
  WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
  OR strpos(d,'''vec_contratacion_temporal.firma_externa.v1''')=0
  OR strpos(d,'''vec_contratacion_temporal.firma_vec.v2''')<>0
  OR strpos(d,'''vec_contratacion_temporal.firma_externa.v2''')<>0
  OR strpos(d,'''vec_contratacion_temporal.firmas_r5.consultar.v2''')<>0 THEN
  RAISE EXCEPTION 'AD162 audiencias preimagen incompatible' USING ERRCODE='55000'; END IF;
 nueva:=left(d,length(d)-3)||', ''vec_contratacion_temporal.firma_vec.v2''::text, ''vec_contratacion_temporal.firma_externa.v2''::text, ''vec_contratacion_temporal.firmas_r5.consultar.v2''::text]))';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
 IF (SELECT pg_get_constraintdef(c.oid,true) FROM pg_constraint c
   WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check') IS DISTINCT FROM nueva THEN
  RAISE EXCEPTION 'AD162 postimagen audiencia divergente' USING ERRCODE='55000'; END IF;
END $audiencias$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_verificada_ct_v2_atestada(
 p_solicitud text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE s jsonb; c jsonb; d jsonb; x record; h text; contexto_h text; recurso text; accion text; audiencia text; tipo_recurso text; perfil text;
BEGIN
 IF p_solicitud IS NULL OR octet_length(p_solicitud) NOT BETWEEN 2 AND 65536
  OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 1 AND 65536
  OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
  RAISE EXCEPTION 'AD162 material inválido' USING ERRCODE='22023'; END IF;
 s:=p_solicitud::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 IF jsonb_typeof(s) IS DISTINCT FROM 'object' OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
  OR s->>'Via' IS NULL OR s->>'Via' NOT IN ('certificado_vec','portafirmas_registro_rrhh')
  OR s->>'PoliticaVerificacion' IS DISTINCT FROM 'politica:vec:firma:verificacion-autonoma:v2'
  OR s->>'OrganizacionRef' IS NULL OR s->>'OrganizacionRef' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  OR s->>'ClaveIdempotencia' IS NULL OR s->>'ClaveIdempotencia' !~ '^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$'
  OR s->>'FirmantePrincipalRef' IS NULL OR s->>'FirmantePrincipalRef' !~ '^per_[A-Za-z0-9_-]{2,159}$' THEN
  RAISE EXCEPTION 'AD162 material firma inválido' USING ERRCODE='22023'; END IF;
 IF s->>'Via'='certificado_vec' THEN
  accion:='contratacion_temporal.documento.firma_vec.registrar'; audiencia:='vec_contratacion_temporal.firma_vec.v2';
  tipo_recurso:='firma_vec_documento_contratacion_temporal'; perfil:='firma_vec_documento_ct_v2'; recurso:='operacion-firma-vec-ct:'||(s->>'ClaveIdempotencia');
 ELSE
  accion:='contratacion_temporal.documento.firma_externa.registrar'; audiencia:='vec_contratacion_temporal.firma_externa.v2';
  tipo_recurso:='firma_externa_documento_contratacion_temporal'; perfil:='firma_externa_documento_ct_v2'; recurso:='operacion-firma-externa-ct:'||(s->>'ClaveIdempotencia');
 END IF;
 h:=encode(sha256(convert_to(p_solicitud,'UTF8')),'hex');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
  '"},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 IF c->>'operacion' IS DISTINCT FROM accion OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
  OR d->>'accion' IS DISTINCT FROM accion OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
  OR d->>'tipo_recurso' IS DISTINCT FROM tipo_recurso OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
  OR d->>'recurso_ref' IS DISTINCT FROM recurso OR c->>'efecto_ref' IS DISTINCT FROM recurso
  OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
  OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
  OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
  OR d->>'principal_id' IS NULL OR d->>'principal_id' !~ '^per_[A-Za-z0-9_-]{2,159}$'
  OR (s->>'Via'='certificado_vec' AND d->>'principal_id' IS DISTINCT FROM s->>'FirmantePrincipalRef')
  OR (s->>'Via'='portafirmas_registro_rrhh' AND (d->>'principal_id' IS NOT DISTINCT FROM s->>'FirmantePrincipalRef'
   OR d->>'version_rol_ref' IS DISTINCT FROM 'rol:firma_externa_registro_ct_desarrollo:v1'))
  OR d->>'perfil_activo_ref' IS DISTINCT FROM s->>'PerfilActivoOperadorRef'
  OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'AD162 firma V2 denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  perfil,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD162 firma requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_firmas_r5_ct_v2_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 IF p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 1 AND 65536
  OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
  RAISE EXCEPTION 'AD162 material consulta inválido' USING ERRCODE='22023'; END IF;
 c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 IF c->>'operacion' IS DISTINCT FROM 'contratacion_temporal.documento.firmas_r5_v2.consultar'
  OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.firmas_r5.consultar.v2'
  OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
  OR d->>'tipo_recurso' IS DISTINCT FROM 'expediente_contratacion_temporal'
  OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
  OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
  OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
  OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
  OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
  OR d->'campos_permitidos' IS DISTINCT FROM '["ByteRange","CatalogoHuella","CatalogoRef","CertificadoHuella","ClaveIdempotencia","CoincideFirmanteCandidato","CoincideFirmanteEnOtroPaso","ConMotivoDevolucion","ContenidoFirmadoHuellaSHA256","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","EntradaDocumentoHuella","EntradaDocumentoLongitud","EntradaDocumentoRef","EntradaDocumentoVersion","EvidenciaFirmasCanonica","EvidenciaFirmasHuellaSHA256","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaAnteriorRef","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","FirmanteRef","HistoriaHuella","HistoriaRevision","HistoriaSeparacionAcreditada","OrdenFirmaPDF","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboAnteriorRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","RevisionHuellaSHA256","RevisionLongitud","Secuencia","SelloTiempoEstado","Via"]'::jsonb
  OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'AD162 consulta V2 denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'consulta_firmas_r5_ct_v2',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD162 consulta requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
DO $acl$
DECLARE f regprocedure; x record;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_autorizacion_atestada_v3.registrar_y_consumir_firma_verificada_ct_v2_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_autorizacion_atestada_v3.consumir_consulta_firmas_r5_ct_v2_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure] LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f);
  FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND a.grantee<>0 AND a.grantee<>p.proowner LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f,pg_get_userbyid(x.grantee)); END LOOP;
  EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_contratacion_temporal_propietario',f);
 END LOOP;
END $acl$;
COMMIT;
