\set ON_ERROR_STOP on
-- AD3-158. Dos consumos nominales para el original firmable de Documentos13.
-- Reserva y confirmación usan el mismo id opaco, acciones y preimágenes
-- diferentes. El replay conserva la revalidación viva de AD62/113.
-- No publica roles ni concesiones: sin las dos acciones gobernadas en una
-- versión publicada y asignada, el núcleo V3 deniega la operación.
-- Preimagen causal: H7 -> H8 -> AD155 -> AD151 -> AD156 -> AD157 (fachada cerrada),
-- con AD113/136 para el consumidor documental. HITO1 + H3/H4 no basta.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000158',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $pre$
DECLARE rol text;
BEGIN
 IF current_user IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario' THEN
  RAISE EXCEPTION 'AD3-158: PARO clave=rol_sql actual=% esperado=vec_autorizacion_atestada_v3_propietario',current_user USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.consumir_solicitud_documental_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'AD3-158: PARO clave=consumidor_ad155_instalado actual=false esperado=true' USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_circuito_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'AD3-158: PARO clave=consumidor_ad151_instalado actual=false esperado=true' USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_externa_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'AD3-158: PARO clave=consumidor_ad156_instalado actual=false esperado=true' USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'AD3-158: PARO clave=fachada_ad157_instalada actual=false esperado=true' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=
   'vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
   AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND p.prokind='f' AND p.provolatile='v' AND p.proparallel='u' AND p.prosecdef
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='48d812c39c5a26c4e905f26575acd04bc6d11017e2ecbac4c3b20eb09ae04144'
   AND NOT EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
     WHERE a.grantee NOT IN (p.proowner,'vec_contratacion_temporal_propietario'::regrole)
       OR a.privilege_type<>'EXECUTE' OR a.is_grantable)) THEN
  RAISE EXCEPTION 'AD3-158: PARO clave=fachada_ad157_fuente_acl actual=divergente esperado=3be925094' USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'AD3-158: PARO clave=consumidor_documentos_replay_instalado actual=false esperado=true' USING ERRCODE='55000'; END IF;
 FOREACH rol IN ARRAY ARRAY['vec_documentos_propietario','vec_documentos_ejecutor','vec_documentos_migrador'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=rol AND NOT rolcanlogin
     AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls) THEN
   RAISE EXCEPTION 'AD3-158: PARO clave=rol_documentos_cerrado rol=% actual=false esperado=true',rol USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;

-- No se añade audiencia: Documentos60 ya gobierna vec_documentos.operacion.v1.
-- El check completo se reconstruye hacia la preimagen registrada por AD151.
DO $audiencias$
DECLARE d text; anterior text; sufijo156 text:=$s$, 'vec_contratacion_temporal.firma_externa.v1'::text]))$s$;
 sufijo151 text:=$s$, 'vec_contratacion_temporal.circuito.consultar.v1'::text]))$s$;
 esperado text:='c220a791d3bf62f5a87ca192373900178c7c3344f10384b1080cfef9c2f81626';
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO d FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF NOT FOUND OR d IS NULL THEN RAISE EXCEPTION 'AD3-158: PARO clave=check_audiencias actual=ausente esperado=validado' USING ERRCODE='55000'; END IF;
 IF right(d,length(sufijo156)) IS DISTINCT FROM sufijo156 THEN
  RAISE EXCEPTION 'AD3-158: PARO clave=sufijo_audiencia_ad156 actual=% esperado=%',right(d,length(sufijo156)),sufijo156 USING ERRCODE='55000'; END IF;
 anterior:=left(d,length(d)-length(sufijo156))||']))';
 IF right(anterior,length(sufijo151)) IS DISTINCT FROM sufijo151 THEN
  RAISE EXCEPTION 'AD3-158: PARO clave=sufijo_audiencia_ad151 actual=% esperado=%',right(anterior,length(sufijo151)),sufijo151 USING ERRCODE='55000'; END IF;
 anterior:=left(anterior,length(anterior)-length(sufijo151))||']))';
 IF encode(sha256(convert_to(anterior,'UTF8')),'hex') IS DISTINCT FROM esperado THEN
  RAISE EXCEPTION 'AD3-158: PARO clave=check_audiencias_pre_ad151_sha256 actual=% esperado=%',
   encode(sha256(convert_to(anterior,'UTF8')),'hex'),esperado USING ERRCODE='55000'; END IF;
 IF strpos(d,'''vec_documentos.operacion.v1''::text')=0 THEN
  RAISE EXCEPTION 'AD3-158: PARO clave=audiencia_documentos actual=ausente esperado=vec_documentos.operacion.v1' USING ERRCODE='55000'; END IF;
END $audiencias$;

DO $nucleo$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; fuente text; nuevo text; actual text; meta jsonb; deps jsonb; deps_compartidas jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 esperada_def_sha256 text:='5dbdac03a2a4e52ca4cb45c57b3bf18e621091da9e818091e30a3f90e68e7330';
 esperada_fuente_sha256 text:='cdc8cb87f27360741a2d0d52e8b9d58d0a1423be8abd8ff9063f389b2c8ea75e';
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 pre151 text:=$p151$           OR (
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
$p151$;
 pre156 text:=$p156$           OR (
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
$p156$;
 extension text:=$e$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'operacion_documentos_comunes'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_documentos.operacion.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM d->>'accion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'documentos'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_original_firmable'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'custodiar_original_firmable'
 AND nullif(d->>'version_rol_ref','') IS NOT NULL
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^ref:[0-9a-f]{64}$'
 AND d->>'recurso_ref' IS DISTINCT FROM ('ref:'||repeat('0',64))
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND ((d->>'accion' IS NOT DISTINCT FROM 'documentos.original_firmable.reservar'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["reserva","intento"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.original_firmable.confirmar'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["documento","recibo"]'::jsonb)))
$e$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD3-158: PARO clave=nucleo_v3 actual=ausente esperado=instalado' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
  INTO original,fuente,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 IF NOT FOUND OR original IS NULL OR fuente IS NULL OR meta IS NULL OR propietario IS NULL OR config IS NULL OR definidora IS NULL THEN
  RAISE EXCEPTION 'AD3-158: PARO clave=metadatos_nucleo actual=ausentes esperado=presentes' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
  INTO deps_compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
   AND d.classid='pg_proc'::regclass AND d.objid=f;
 IF length(original)-length(replace(original,pre151||pre156||marca,''))<>length(pre151||pre156||marca)
    OR encode(sha256(convert_to(replace(original,pre151||pre156||marca,marca),'UTF8')),'hex') IS DISTINCT FROM esperada_def_sha256 THEN
  RAISE EXCEPTION 'AD3-158: PARO clave=nucleo_pre_ad151_definicion_sha256 actual=% esperado=%',
   encode(sha256(convert_to(replace(original,pre151||pre156||marca,marca),'UTF8')),'hex'),esperada_def_sha256 USING ERRCODE='55000'; END IF;
 IF length(fuente)-length(replace(fuente,pre151||pre156||marca,''))<>length(pre151||pre156||marca)
    OR encode(sha256(convert_to(replace(fuente,pre151||pre156||marca,marca),'UTF8')),'hex') IS DISTINCT FROM esperada_fuente_sha256 THEN
  RAISE EXCEPTION 'AD3-158: PARO clave=nucleo_pre_ad151_fuente_sha256 actual=% esperado=%',
   encode(sha256(convert_to(replace(fuente,pre151||pre156||marca,marca),'UTF8')),'hex'),esperada_fuente_sha256 USING ERRCODE='55000'; END IF;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.prokind='f' AND p.provolatile='v'
       AND p.proparallel='u' AND p.prosecdef)
    OR EXISTS(SELECT 1 FROM pg_database db CROSS JOIN LATERAL aclexplode(coalesce(db.datacl,acldefault('d',db.datdba))) a
       WHERE db.datname=current_database() AND a.grantee=0 AND a.privilege_type='TEMPORARY')
    OR EXISTS(SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolcanlogin
       AND has_database_privilege(r.oid,current_database(),'TEMPORARY'))
    OR NOT EXISTS(SELECT 1 FROM aclexplode(coalesce(acl,acldefault('f',propietario))) a
       WHERE a.grantee=propietario AND a.grantor=propietario
         AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
    OR EXISTS(SELECT 1 FROM aclexplode(coalesce(acl,acldefault('f',propietario))) a
       WHERE a.grantee<>propietario OR a.grantor<>propietario OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
    OR deps IS DISTINCT FROM jsonb_build_array(
       jsonb_build_object('classid','pg_proc'::regclass::oid,'objid',f::oid,'objsubid',0,'refclassid','pg_language'::regclass::oid,
         'refobjid',(SELECT oid FROM pg_language WHERE lanname='plpgsql'),'refobjsubid',0,'deptype','n'),
       jsonb_build_object('classid','pg_proc'::regclass::oid,'objid',f::oid,'objsubid',0,'refclassid','pg_namespace'::regclass::oid,
         'refobjid','vec_autorizacion_atestada_v3'::regnamespace::oid,'refobjsubid',0,'deptype','n'))
    OR deps_compartidas IS DISTINCT FROM jsonb_build_array(jsonb_build_object(
       'dbid',(SELECT oid FROM pg_database WHERE datname=current_database()),'classid','pg_proc'::regclass::oid,
       'objid',f::oid,'objsubid',0,'refclassid','pg_authid'::regclass::oid,'refobjid',propietario,'deptype','o'))
    OR strpos(original,'operacion_documentos_comunes')=0
    OR strpos(original,'documentos.firmado.custodiar')=0
    OR strpos(original,'documentos.original_firmable.reservar')<>0
    OR strpos(original,'documentos.original_firmable.confirmar')<>0 THEN
  RAISE EXCEPTION 'AD3-158: PARO clave=nucleo_acl_estructura_nominal actual=divergente esperado=post_ad156' USING ERRCODE='55000'; END IF;
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
        AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps_compartidas THEN
  RAISE EXCEPTION 'AD3-158: PARO clave=nucleo_postimagen_metadatos actual=divergente esperado=preservados' USING ERRCODE='55000'; END IF;
END $nucleo$;

DO $wrapper$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; nuevo text; actual text; meta jsonb; deps jsonb; deps_compartidas jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 esperada_def_sha256 text:='3e4412dbac79bcce6d0425f4c0fddfbb32602ed2169b9d694c2292fbb83a5d4f';
 ancla text:=$a$      AND d->>'finalidad'='registrar_documento_externo' AND d->'campos_permitidos'='["documento","recibo"]'::jsonb)
     OR (accion='documentos.firmado.custodiar' AND d->>'tipo_recurso'='documento_firmado'
      AND d->>'finalidad'='custodiar_documento_firmado' AND d->'campos_permitidos'='["documento_firmado.custodia","evidencia_custodia"]'::jsonb))$a$;
 ancla_nueva text:=$b$      AND d->>'finalidad'='registrar_documento_externo' AND d->'campos_permitidos'='["documento","recibo"]'::jsonb)
     OR (accion='documentos.firmado.custodiar' AND d->>'tipo_recurso'='documento_firmado'
      AND d->>'finalidad'='custodiar_documento_firmado' AND d->'campos_permitidos'='["documento_firmado.custodia","evidencia_custodia"]'::jsonb)
     OR (accion='documentos.original_firmable.reservar' AND d->>'tipo_recurso'='documento_original_firmable'
      AND d->>'finalidad'='custodiar_original_firmable' AND d->'campos_permitidos'='["reserva","intento"]'::jsonb)
     OR (accion='documentos.original_firmable.confirmar' AND d->>'tipo_recurso'='documento_original_firmable'
      AND d->>'finalidad'='custodiar_original_firmable' AND d->'campos_permitidos'='["documento","recibo"]'::jsonb))$b$;
 replay text:=$r$  IF accion NOT IN ('documentos.generado.alta','documentos.notificacion.preparar','documentos.externo.registrar','documentos.firmado.custodiar')$r$;
 replay_nuevo text:=$rn$  IF accion NOT IN ('documentos.generado.alta','documentos.notificacion.preparar','documentos.externo.registrar','documentos.firmado.custodiar','documentos.original_firmable.reservar','documentos.original_firmable.confirmar')$rn$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD3-158: PARO clave=fachada_documentos_replay actual=ausente esperado=instalada' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
  INTO original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
  INTO deps_compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
   AND d.classid='pg_proc'::regclass AND d.objid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM esperada_def_sha256 THEN
  RAISE EXCEPTION 'AD3-158: PARO clave=fachada_documentos_definicion_sha256 actual=% esperado=%',
   encode(sha256(convert_to(original,'UTF8')),'hex'),esperada_def_sha256 USING ERRCODE='55000'; END IF;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.prokind='f' AND p.provolatile='v'
       AND p.proparallel='u' AND p.prosecdef)
    OR length(original)-length(replace(original,ancla,''))<>length(ancla)
    OR length(original)-length(replace(original,replay,''))<>length(replay)
    OR strpos(original,'documentos.original_firmable.reservar')<>0
    OR strpos(original,'documentos.original_firmable.confirmar')<>0
    OR NOT EXISTS(SELECT 1 FROM aclexplode(coalesce(acl,acldefault('f',propietario))) a
       WHERE a.grantee=propietario AND a.grantor=propietario AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
    OR NOT EXISTS(SELECT 1 FROM aclexplode(coalesce(acl,acldefault('f',propietario))) a
       WHERE a.grantee='vec_documentos_propietario'::regrole AND a.grantor=propietario
         AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
    OR EXISTS(SELECT 1 FROM aclexplode(coalesce(acl,acldefault('f',propietario))) a
       WHERE a.grantee NOT IN (propietario,'vec_documentos_propietario'::regrole)
          OR a.grantor<>propietario OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
    OR deps IS DISTINCT FROM jsonb_build_array(
       jsonb_build_object('classid','pg_proc'::regclass::oid,'objid',f::oid,'objsubid',0,
         'refclassid','pg_language'::regclass::oid,'refobjid',(SELECT oid FROM pg_language WHERE lanname='plpgsql'),
         'refobjsubid',0,'deptype','n'),
       jsonb_build_object('classid','pg_proc'::regclass::oid,'objid',f::oid,'objsubid',0,
         'refclassid','pg_namespace'::regclass::oid,'refobjid','vec_autorizacion_atestada_v3'::regnamespace::oid,
         'refobjsubid',0,'deptype','n'))
    OR jsonb_array_length(deps_compartidas)<>2
    OR NOT (deps_compartidas @> jsonb_build_array(
       jsonb_build_object('dbid',(SELECT oid FROM pg_database WHERE datname=current_database()),
         'classid','pg_proc'::regclass::oid,'objid',f::oid,'objsubid',0,'refclassid','pg_authid'::regclass::oid,
         'refobjid','vec_documentos_propietario'::regrole::oid,'deptype','a'),
       jsonb_build_object('dbid',(SELECT oid FROM pg_database WHERE datname=current_database()),
         'classid','pg_proc'::regclass::oid,'objid',f::oid,'objsubid',0,'refclassid','pg_authid'::regclass::oid,
         'refobjid',propietario,'deptype','o'))) THEN
  RAISE EXCEPTION 'AD3-158: PARO clave=fachada_documentos_acl_estructura actual=divergente esperado=post_ad113' USING ERRCODE='55000'; END IF;
 nuevo:=replace(replace(original,ancla,ancla_nueva),replay,replay_nuevo);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(replace(actual,ancla_nueva,ancla),replay_nuevo,replay) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
       FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
       FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
        AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps_compartidas THEN
  RAISE EXCEPTION 'AD3-158: PARO clave=fachada_documentos_postimagen actual=divergente esperado=preservada' USING ERRCODE='55000'; END IF;
END $wrapper$;
COMMIT;
