\set ON_ERROR_STOP on
-- AD286: recuperación de la operación firmada exacta y su corte histórico.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:AD286',0));
DO $pre$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.comprobar_consumo_recuperacion_firmas_ct_v2(bytea,jsonb)');
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR pg_is_in_recovery() OR f IS NULL
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner=to_regrole('vec_autorizacion_atestada_v3_propietario') AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u' AND p.proacl=ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario','vec_autorizacion_propietario=X/vec_autorizacion_atestada_v3_propietario']::aclitem[]
   AND p.proconfig=ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='897eac68c1f79337250e825c41e500a79e45c3237908a336576f82d469e35fcb'
   AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='884828ee4f4d9769b2bf9e49ff2df0e5b5e922a09822812a84457da9b7fc9ab1')
 THEN RAISE EXCEPTION 'AD286: preimagen o ACL divergentes' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.comprobar_consumo_recuperacion_firmas_ct_v2(p_material_consulta bytea, p_consumo jsonb)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
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
 OR (s->'UnidadRef' IS DISTINCT FROM 'null'::jsonb AND (jsonb_typeof(s->'UnidadRef') IS DISTINCT FROM 'string'
  OR (s->>'UnidadRef' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$') IS NOT TRUE
  OR s->>'Via' IS DISTINCT FROM 'certificado_vec'))
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
 OR (s->>'PasoOrden' ~ '^[1-3]$') IS NOT TRUE
 OR (s->>'PasoOrden'='3' AND (s->>'Via' IS DISTINCT FROM 'certificado_vec'
  OR s->'UnidadRef' IS NOT DISTINCT FROM 'null'::jsonb)) THEN
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
 THEN
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
 -- La capacidad ya consumida aporta la decisión que AD210 contrasta con
 -- la asignación actual. La vía histórica externa conserva el hash previo.
 IF s->>'Via'='certificado_vec' THEN
  contexto_h:=vec_autorizacion_atestada_v3.huella_recurso_consulta_firmas_r5_ct_v1(
   convert_from(p_material_consulta,'UTF8'),r.decision_canonica);
 END IF;
 IF p_consumo->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
  OR r.huella_efecto_sha256 IS DISTINCT FROM contexto_h THEN
  RAISE EXCEPTION 'AD286: consumo y ámbito actual divergentes' USING ERRCODE='42501';
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
END $function$;


DO $post$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.comprobar_consumo_recuperacion_firmas_ct_v2(bytea,jsonb)');
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR pg_is_in_recovery() OR f IS NULL
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner=to_regrole('vec_autorizacion_atestada_v3_propietario') AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u' AND p.proacl=ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario','vec_autorizacion_propietario=X/vec_autorizacion_atestada_v3_propietario']::aclitem[]
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='1e45359539948ccd4e63f28abc9b13dc04f00b2affdd072f0422bb75c00656ea'
   AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='7f0494e2c2e88ac5642198767e18789148f25d304037d8a11dd78dd4045c8a3c')
 THEN RAISE EXCEPTION 'AD286: postimagen o ACL divergentes' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
