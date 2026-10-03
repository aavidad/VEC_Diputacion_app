\set ON_ERROR_STOP on
-- AD166: publicación nominal de cargos bajo la autoridad V3 y auditoría común.
-- Requiere AUT33 y AD165; no publica perfiles, asignaciones ni permisos.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000166',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
  OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR pg_catalog.to_regrole('vec_personal_propietario') IS NULL
  OR pg_catalog.to_regrole('vec_personal_ejecutor') IS NULL
  OR pg_catalog.to_regprocedure('vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)') IS NULL
  OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_publicacion_certificado_nominal_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
  OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_publicacion_cargo_competencial_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD166: PARO clave=dependencias actual=incompatible esperado=AUT33_AD165_y_Personal' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

-- Preimagen exacta post-AD165 acreditada en PostgreSQL 18 del clon local.
-- Sólo se insertan el perfil operativo y la audiencia propios de Personal.
DO $nucleo$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text;fuente text;nuevo text;actual text;meta jsonb;acl aclitem[];deps jsonb;deps_compartidas jsonb;
 propietario oid;config text[];definidora boolean;
 esperado_def text:='684229f6e5d3c3a3f4e7b849cc03b55ee275d58cb0b40a5cdeeff39edd9c7beb';
 esperado_fuente text:='7b04ba29e1ed6943647b445985e951f70130ce323c1acc2d2ea4eafe00f80549';
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 excl_nuevo text:=$x$p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento' AND p_perfil_mutacion IS DISTINCT FROM 'publicacion_certificado_nominal'$x$;
 extension166 text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'publicacion_cargo_competencial'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.cargo_competencial.publicar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.cargo_competencial.publicar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'cargo_competencial'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'administrar_cargos_competenciales'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["cargo","enlace","huella_sha256","recibo","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true'
 AND vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(
   d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
   c->>'operacion','personal','cargo_competencial','administrar_cargos_competenciales',
   '["cargo","enlace","huella_sha256","recibo","version"]'::jsonb,d->'vinculo_autenticacion_actor') IS TRUE)
$x$;
 excl166 text:=excl_nuevo||' AND p_perfil_mutacion IS DISTINCT FROM ''publicacion_cargo_competencial''';
 runtime_marca166 text:=E'       OR NOT (\n           (p_perfil_mutacion IS NOT DISTINCT FROM ''publicacion_certificado_nominal''';
 runtime166 text:=E'       OR NOT (\n           (p_perfil_mutacion IS NOT DISTINCT FROM ''publicacion_cargo_competencial''\n            AND EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit\n             AND NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls))\n            AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole\n             AND m.roleid=''vec_personal_ejecutor''::regrole\n             AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)\n            AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=''vec_personal_ejecutor''::regrole)\n            AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole)=1)\n           OR (p_perfil_mutacion IS NOT DISTINCT FROM ''publicacion_certificado_nominal''';
 actual_sha text;

BEGIN
 SELECT pg_catalog.pg_get_functiondef(f),p.prosrc,pg_catalog.to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,fuente,meta,acl,propietario,config,definidora FROM pg_catalog.pg_proc p WHERE p.oid=f;
 SELECT COALESCE(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f;
 SELECT COALESCE(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO deps_compartidas FROM pg_catalog.pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database())
 AND d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f;
 actual_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(original,'UTF8')),'hex');
 IF actual_sha IS DISTINCT FROM esperado_def THEN
  RAISE EXCEPTION 'AD166: PARO clave=def_postAD165 actual=% esperado=%',actual_sha,esperado_def USING ERRCODE='55000'; END IF;
 actual_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(fuente,'UTF8')),'hex');
 IF actual_sha IS DISTINCT FROM esperado_fuente THEN
  RAISE EXCEPTION 'AD166: PARO clave=src_postAD165 actual=% esperado=%',actual_sha,esperado_fuente USING ERRCODE='55000'; END IF;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
  OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(COALESCE(acl,pg_catalog.acldefault('f',propietario))) a
    WHERE a.grantee=propietario AND a.grantor=propietario AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
  OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(COALESCE(acl,pg_catalog.acldefault('f',propietario))) a
    WHERE a.grantee<>propietario OR a.grantor<>propietario OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
  OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,marca,''))<>pg_catalog.length(marca)
  OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,excl_nuevo,''))<>pg_catalog.length(excl_nuevo)
  OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,runtime_marca166,''))<>pg_catalog.length(runtime_marca166)
  OR pg_catalog.strpos(original,'''publicacion_cargo_competencial''')<>0
 THEN RAISE EXCEPTION 'AD166: PARO clave=preimagen_nucleo actual=incompatible esperado=AD165_exacto_y_ACL_privada' USING ERRCODE='55000'; END IF;
 nuevo:=pg_catalog.replace(original,runtime_marca166,runtime166);
 nuevo:=pg_catalog.replace(nuevo,excl_nuevo,excl166);
 nuevo:=pg_catalog.replace(nuevo,marca,extension166||marca);
 EXECUTE nuevo;
 SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
  OR pg_catalog.replace(pg_catalog.replace(pg_catalog.replace(actual,extension166||marca,marca),excl166,excl_nuevo),runtime166,runtime_marca166) IS DISTINCT FROM original
  OR (SELECT pg_catalog.to_jsonb(p)-'prosrc' FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
  OR (SELECT p.proacl FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM acl
  OR (SELECT p.proowner FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM propietario
  OR (SELECT p.proconfig FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM config
  OR (SELECT p.prosecdef FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM definidora
  OR (SELECT COALESCE(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
      FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
  OR (SELECT COALESCE(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
      FROM pg_catalog.pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database())
       AND d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps_compartidas
 THEN RAISE EXCEPTION 'AD166: PARO clave=postimagen_nucleo actual=divergente esperado=cambio_y_ACL_exactos' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text;nuevo text;esperado text:='89fb71c30c516e0da102b81154716e36d17a4bcad7fe261076f70f6ed758ff37';
 audiencia text:='vec_personal.cargo_competencial.publicar.v1';
 actual_sha text;
BEGIN
 SELECT pg_catalog.regexp_replace(pg_catalog.pg_get_constraintdef(c.oid,false),'\s+',' ','g') INTO STRICT d
 FROM pg_catalog.pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 actual_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(d,'UTF8')),'hex');
 IF actual_sha IS DISTINCT FROM esperado THEN
  RAISE EXCEPTION 'AD166: PARO clave=CHECK_postAD165 actual=% esperado=%',actual_sha,esperado USING ERRCODE='55000'; END IF;
 IF pg_catalog.strpos(d,'CHECK ((audiencia_consumo = ANY (ARRAY[')<>1 OR pg_catalog.right(d,4)<>'])))'
  OR pg_catalog.strpos(d,pg_catalog.quote_literal(audiencia))<>0
 THEN RAISE EXCEPTION 'AD166: PARO clave=CHECK_audiencia actual=incompatible esperado=AD165_sin_cargo' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 nuevo:=pg_catalog.left(d,pg_catalog.length(d)-4)||', '||pg_catalog.quote_literal(audiencia)||'::text])))';
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nuevo;
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_publicacion_cargo_competencial_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb;d jsonb;x record;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable' OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR p_capacidad IS NULL OR pg_catalog.octet_length(p_capacidad) NOT BETWEEN 2 AND 32768
  OR p_decision IS NULL OR pg_catalog.octet_length(p_decision) NOT BETWEEN 2 AND 524288
 THEN RAISE EXCEPTION 'AD166: transacción o material denegado' USING ERRCODE='42501'; END IF;
 c:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.cargo_competencial.publicar.v1'
  OR c->>'operacion' IS DISTINCT FROM 'personal.cargo_competencial.publicar'
  OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'personal'
  OR d->>'tipo_recurso' IS DISTINCT FROM 'cargo_competencial'
  OR d->>'finalidad' IS DISTINCT FROM 'administrar_cargos_competenciales'
  OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
  OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
  OR d->'campos_permitidos' IS DISTINCT FROM '["cargo","enlace","huella_sha256","recibo","version"]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
  OR vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(
   d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
   c->>'operacion','personal','cargo_competencial','administrar_cargos_competenciales',
   '["cargo","enlace","huella_sha256","recibo","version"]'::jsonb,d->'vinculo_autenticacion_actor') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD166: decisión nominal denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'publicacion_cargo_competencial',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD166: requiere decisión actual' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_publicacion_cargo_competencial_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_personal_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_publicacion_cargo_competencial_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_personal_propietario;
COMMIT;
