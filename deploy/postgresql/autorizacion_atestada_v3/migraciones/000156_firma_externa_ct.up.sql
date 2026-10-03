\set ON_ERROR_STOP on
-- AD3-156. Consumo nominal del registro de firma externa de CT170.
-- Exige postimagen exacta H7 -> H8 -> AD155 -> AD151; hito1 + H3/H4 no la contiene.
-- El rol candidato rol:firma_externa_registro_ct_desarrollo:v1 pertenece al
-- registrador RRHH, nunca al firmante. Esta migración no lo publica, no lo
-- asigna y no concede competencias por petición. Queda cerrada hasta que la
-- autoridad central publique rol, huella y asignación nominal con CAS.
-- AD156 no acredita validez legal, portafirmas ni firma por autenticación.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog, pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000156',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE rol text;
BEGIN
 IF current_user IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario' THEN
  RAISE EXCEPTION 'AD3-156: PARO clave=rol_sql actual=% esperado=vec_autorizacion_atestada_v3_propietario',current_user USING ERRCODE='55000'; END IF;
 IF to_regclass('vec_autorizacion_atestada_v3.clave_capacidad_version') IS NULL THEN
  RAISE EXCEPTION 'AD3-156: PARO clave=tabla_clave_capacidad_instalada actual=false esperado=true' USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.consumir_solicitud_documental_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'AD3-156: PARO clave=consumidor_ad155_instalado actual=false esperado=true' USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_circuito_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'AD3-156: PARO clave=consumidor_ad151_instalado actual=false esperado=true' USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_externa_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
  RAISE EXCEPTION 'AD3-156: PARO clave=fachada_firma_externa_ya_instalada actual=true esperado=false' USING ERRCODE='55000'; END IF;
 FOREACH rol IN ARRAY ARRAY['vec_contratacion_temporal_propietario','vec_contratacion_temporal_ejecutor','vec_contratacion_temporal_migrador'] LOOP
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=rol AND NOT rolcanlogin
    AND NOT rolsuper AND NOT rolcreaterole AND NOT rolcreatedb AND NOT rolbypassrls)
  THEN RAISE EXCEPTION 'AD3-156: PARO clave=rol_tecnico_cerrado rol=% actual=false esperado=true',rol USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;

DO $nucleo$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; nuevo text; actual text; fuente text; meta jsonb; deps jsonb; deps_compartidas jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 -- Postimagen AD155 publicada: definición/fuente íntegras, sin reconstrucción parcial.
 esperada_def_sha256 text:='5dbdac03a2a4e52ca4cb45c57b3bf18e621091da9e818091e30a3f90e68e7330';
 esperada_fuente_sha256 text:='cdc8cb87f27360741a2d0d52e8b9d58d0a1423be8abd8ff9063f389b2c8ea75e';
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 pre151 text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'ct_circuito_consultar'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.circuito.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.circuito.consultar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'expediente_circuito_rrhh'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^expediente:[A-Za-z0-9._:/#-]{2,149}$'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["circuito"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb)
$x$;
 extension text:=$y$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'firma_externa_documento_ct'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.documento.firma_externa.registrar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.firma_externa.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'firma_externa_documento_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d->>'version_rol_ref' IS NOT DISTINCT FROM 'rol:firma_externa_registro_ct_desarrollo:v1'
 AND d->>'recurso_ref' IS NOT NULL
 AND d->>'recurso_ref' ~ '^operacion-firma-externa-ct:[A-Za-z0-9][A-Za-z0-9._-]{15,63}$'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$y$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD3-156: PARO funcion=consumir_decision_mutacion_v3_interna clave=instalada actual=false esperado=true' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO original,fuente,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 IF NOT FOUND OR original IS NULL OR fuente IS NULL OR meta IS NULL
    OR propietario IS NULL OR config IS NULL OR definidora IS NULL
 THEN RAISE EXCEPTION 'AD3-156: PARO funcion=consumir_decision_mutacion_v3_interna clave=metadatos_presentes actual=false esperado=true' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO deps_compartidas FROM pg_shdepend d
 WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
   AND d.classid='pg_proc'::regclass AND d.objid=f;
 IF length(original)-length(replace(original,pre151,''))<>length(pre151)
    OR encode(sha256(convert_to(replace(original,pre151||marca,marca),'UTF8')),'hex') IS DISTINCT FROM esperada_def_sha256 THEN
  RAISE EXCEPTION 'AD3-156: PARO funcion=consumir_decision_mutacion_v3_interna clave=pre_ad151_definicion_reconstruida_sha256 actual=% esperado=%',
   encode(sha256(convert_to(replace(original,pre151||marca,marca),'UTF8')),'hex'),esperada_def_sha256 USING ERRCODE='55000'; END IF;
 IF length(fuente)-length(replace(fuente,pre151,''))<>length(pre151)
    OR encode(sha256(convert_to(replace(fuente,pre151||marca,marca),'UTF8')),'hex') IS DISTINCT FROM esperada_fuente_sha256 THEN
  RAISE EXCEPTION 'AD3-156: PARO funcion=consumir_decision_mutacion_v3_interna clave=pre_ad151_fuente_reconstruida_sha256 actual=% esperado=%',
   encode(sha256(convert_to(replace(fuente,pre151||marca,marca),'UTF8')),'hex'),esperada_fuente_sha256 USING ERRCODE='55000'; END IF;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
         AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
         AND p.prokind='f' AND p.provolatile='v' AND p.proparallel='u'
         AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
    OR EXISTS (SELECT 1 FROM pg_database db
         CROSS JOIN LATERAL aclexplode(coalesce(db.datacl,acldefault('d',db.datdba))) a
         WHERE db.datname=current_database() AND a.grantee=0 AND a.privilege_type='TEMPORARY')
    OR EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolcanlogin
         AND has_database_privilege(r.oid,current_database(),'TEMPORARY'))
    OR NOT EXISTS (SELECT 1 FROM aclexplode(coalesce(acl,acldefault('f',propietario))) a
         WHERE a.grantee=propietario AND a.grantor=propietario
           AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
    OR EXISTS (SELECT 1 FROM aclexplode(coalesce(acl,acldefault('f',propietario))) a
         WHERE a.grantee<>propietario OR a.grantor<>propietario
            OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
    OR deps IS DISTINCT FROM jsonb_build_array(
         jsonb_build_object('classid','pg_proc'::regclass::oid,'objid',f::oid,'objsubid',0,
           'refclassid','pg_language'::regclass::oid,
           'refobjid',(SELECT oid FROM pg_language WHERE lanname='plpgsql'),
           'refobjsubid',0,'deptype','n'),
         jsonb_build_object('classid','pg_proc'::regclass::oid,'objid',f::oid,'objsubid',0,
           'refclassid','pg_namespace'::regclass::oid,
           'refobjid','vec_autorizacion_atestada_v3'::regnamespace::oid,
           'refobjsubid',0,'deptype','n'))
    OR deps_compartidas IS DISTINCT FROM jsonb_build_array(
         jsonb_build_object('dbid',(SELECT oid FROM pg_database WHERE datname=current_database()),
           'classid','pg_proc'::regclass::oid,'objid',f::oid,'objsubid',0,
           'refclassid','pg_authid'::regclass::oid,'refobjid',propietario,'deptype','o'))
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR strpos(original,'vec_contratacion_temporal_ejecutor')=0
    OR strpos(original,'documentos.firmado.custodiar')=0
    OR strpos(original,'ct_circuito_consultar')=0
    OR strpos(original,'firma_externa_documento_ct')<>0
    OR strpos(original,'contratacion_temporal.documento.firma_externa.registrar')<>0
 THEN RAISE EXCEPTION 'AD3-156: PARO funcion=consumir_decision_mutacion_v3_interna clave=entorno_acl_dependencias_post_ad151 actual=false esperado=true' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,extension||marca,marca) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
        FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
          AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps_compartidas
 THEN RAISE EXCEPTION 'AD3-156: PARO funcion=consumir_decision_mutacion_v3_interna clave=postimagen_reversible_metadatos_preservados actual=false esperado=true' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE
 d text; anterior text; nueva text;
 sufijo_ad151 text:=$s$, 'vec_contratacion_temporal.circuito.consultar.v1'::text]))$s$;
 esperada_pre_ad151_sha256 text:='c220a791d3bf62f5a87ca192373900178c7c3344f10384b1080cfef9c2f81626';
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF NOT FOUND OR d IS NULL THEN
  RAISE EXCEPTION 'AD3-156: PARO clave=check_audiencias_instalado actual=false esperado=true' USING ERRCODE='55000'; END IF;
 IF right(d,length(sufijo_ad151)) IS DISTINCT FROM sufijo_ad151 THEN
  RAISE EXCEPTION 'AD3-156: PARO clave=sufijo_audiencia_ad151 actual=% esperado=%',
   right(d,length(sufijo_ad151)),sufijo_ad151 USING ERRCODE='55000'; END IF;
 anterior:=left(d,length(d)-length(sufijo_ad151))||']))';
 IF encode(sha256(convert_to(anterior,'UTF8')),'hex') IS DISTINCT FROM esperada_pre_ad151_sha256 THEN
  RAISE EXCEPTION 'AD3-156: PARO clave=pre_ad151_check_audiencias_sha256 actual=% esperado=%',
   encode(sha256(convert_to(anterior,'UTF8')),'hex'),esperada_pre_ad151_sha256 USING ERRCODE='55000'; END IF;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'''vec_contratacion_temporal.circuito.consultar.v1''::text')=0
    OR strpos(d,'''vec_contratacion_temporal.firma_externa.v1''::text')<>0 THEN
  RAISE EXCEPTION 'AD3-156: PARO clave=audiencia_nominal_post_ad151 actual=false esperado=true' USING ERRCODE='55000'; END IF;
 nueva:=left(d,length(d)-3)||', ''vec_contratacion_temporal.firma_externa.v1''::text]))';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
 IF (SELECT pg_get_constraintdef(c.oid,true) FROM pg_constraint c
     WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND c.conname='clave_capacidad_version_audiencia_consumo_check') IS DISTINCT FROM nueva THEN
  RAISE EXCEPTION 'AD3-156: PARO clave=postimagen_audiencias actual=divergente esperado=exacta' USING ERRCODE='55000'; END IF;
END $audiencias$;

-- CT170 comprueba el contrato de negocio completo. Esta fachada solo recibe
-- sus bytes canónicos, verifica claves cerradas y liga el efecto V3 a ellos.
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_externa_ct_v3_atestada(
 p_solicitud text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' AS $f$
DECLARE s jsonb; c jsonb; d jsonb; x record; material_h text; contexto_h text; recurso text;
BEGIN
 IF p_solicitud IS NULL OR octet_length(p_solicitud) NOT BETWEEN 2 AND 65536
    OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 1 AND 65536
    OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
 THEN RAISE EXCEPTION 'AD3-156: material inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  s:=p_solicitud::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'AD3-156: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(s) IS DISTINCT FROM 'object' OR jsonb_typeof(c) IS DISTINCT FROM 'object'
    OR jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(s))<>45
    OR NOT (s ?& ARRAY[
     'Via','OrganizacionRef','ExpedienteRef','VersionExpediente','Documento','CatalogoRef','CatalogoHuella',
     'PasoRef','PasoOrden','Secuencia','HistoriaRevision','HistoriaHuella',
     'OriginalRef','OriginalVersion','OriginalHuella','FirmadoHuella',
     'CertificadoHuella','FirmanteRef','FirmantePrincipalRef','PerfilFirmanteRef','CargoFirmante',
     'UnidadFirmanteRef','PerfilActivoFirmanteRef','PuestoFirmanteRef','AmbitoFirmanteRef','AsignacionFirmanteRef',
     'AsignacionFirmanteVersion','AsignacionFirmanteHuella','VersionRolFirmanteRef','VersionRolFirmanteHuella',
     'ControlVigenciaFirmanteRef','ControlVigenciaFirmanteRevision','ControlVigenciaFirmanteHuella',
     'AsignacionVigenteDesde','AsignacionVigenteHasta',
     'ActoCompetenciaRef','DelegacionRef','PoliticaVerificacion',
     'RevocacionEstado','SelloTiempoEstado','ReferenciaPortafirmasDeclarada','FechaPortafirmasDeclarada',
     'ClaveIdempotencia','DocumentoCustodiaRef','DocumentoCustodiaVersion'])
    OR jsonb_typeof(s->'OrganizacionRef') IS DISTINCT FROM 'string'
    OR s->>'OrganizacionRef' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR jsonb_typeof(s->'ClaveIdempotencia') IS DISTINCT FROM 'string'
    OR s->>'ClaveIdempotencia' !~ '^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$'
    OR jsonb_typeof(s->'FirmantePrincipalRef') IS DISTINCT FROM 'string'
    OR s->>'FirmantePrincipalRef' !~ '^per_[A-Za-z0-9_-]{2,159}$'
 THEN RAISE EXCEPTION 'AD3-156: material de firma externa inválido' USING ERRCODE='22023'; END IF;
 material_h:=encode(sha256(convert_to(p_solicitud,'UTF8')),'hex');
 recurso:='operacion-firma-externa-ct:'||(s->>'ClaveIdempotencia');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
  '"},"atributos":{"material_sha256":"'||material_h||'"}}','UTF8')),'hex');
 IF c->>'operacion' IS DISTINCT FROM 'contratacion_temporal.documento.firma_externa.registrar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.firma_externa.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'firma_externa_documento_contratacion_temporal'
    OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
    OR d->>'version_rol_ref' IS DISTINCT FROM 'rol:firma_externa_registro_ct_desarrollo:v1'
    OR d->>'recurso_ref' IS DISTINCT FROM recurso OR c->>'efecto_ref' IS DISTINCT FROM recurso
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
    OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
    OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
    OR d->>'principal_id' IS NULL
    OR d->>'principal_id' !~ '^per_[A-Za-z0-9_-]{2,159}$'
    OR d->>'principal_id' IS NOT DISTINCT FROM s->>'FirmantePrincipalRef'
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-156: firma externa denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'firma_externa_documento_ct',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 -- También el replay de CT170 requiere autorización vigente y una auditoría nueva.
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-156: requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_externa_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_contratacion_temporal_propietario;
DO $acl$
DECLARE f regprocedure; firma text; a record;
 permitido oid:='vec_contratacion_temporal_propietario'::regrole::oid;
BEGIN
 FOREACH firma IN ARRAY ARRAY[
  'vec_autorizacion_atestada_v3.registrar_y_consumir_firma_externa_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
  f:=firma::regprocedure;
  -- Cerrar también concesiones heredadas de privilegios predeterminados.
  FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,
   LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
   WHERE p.oid=f AND x.grantee<>p.proowner LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
    CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(a.grantee)) END);
  END LOOP;
  EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_contratacion_temporal_propietario',f::text);
  IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
     OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
     OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
     OR NOT has_function_privilege(permitido,f,'EXECUTE')
     OR EXISTS(SELECT 1 FROM pg_proc p,
      LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
      WHERE p.oid=f AND (x.grantee NOT IN (p.proowner,permitido)
       OR x.privilege_type<>'EXECUTE' OR x.is_grantable))
  THEN RAISE EXCEPTION 'AD3-156: PARO funcion=registrar_y_consumir_firma_externa_ct_v3_atestada clave=acl_cerrada actual=false esperado=true' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;
COMMIT;
