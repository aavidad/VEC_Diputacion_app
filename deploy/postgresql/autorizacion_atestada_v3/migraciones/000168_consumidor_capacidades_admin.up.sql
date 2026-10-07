\set ON_ERROR_STOP on
-- AD168: capacidades del perfil nominal de Aplicación. No abre actos ni otras lecturas.
-- Denegación/error requieren el contrato de auditoría común de frontera de L.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000168',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR pg_catalog.to_regrole('vec_admin_perfiles_lector') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_capacidades_admin_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD168: PARO clave=dependencias actual=incompatible esperado=AUT34_y_postAD166' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $nucleo$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text;fuente text;nuevo text;actual text;meta jsonb;acl aclitem[];deps jsonb;deps_compartidas jsonb;
 propietario oid;config text[];definidora boolean;
 esperado_def text:='3e73c94210c2ad0ecb4461afae9155a676464f62a4dd0e107417131e3564cddb';
 esperado_fuente text:='25519603cb49c6582eb400cfe78c5679e3bc024373466009aff164d74ff0e0f5';
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 excl_nuevo text:=$x$p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento' AND p_perfil_mutacion IS DISTINCT FROM 'publicacion_certificado_nominal' AND p_perfil_mutacion IS DISTINCT FROM 'publicacion_cargo_competencial'$x$;
 extension168 text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'capacidades_admin'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_autorizacion.administracion_perfiles.lectura.capacidades.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'administracion.perfiles.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'administracion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'perfil'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_perfiles'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["actos_disponibles","perfil_ref","preimagen","version","vinculo_ref"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true'
 AND vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(
   d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
   c->>'operacion','administracion','perfil','gestion_perfiles',
   '["actos_disponibles","perfil_ref","preimagen","version","vinculo_ref"]'::jsonb,d->'vinculo_autenticacion_actor') IS TRUE)
$x$;
 excl168 text:=excl_nuevo||' AND p_perfil_mutacion IS DISTINCT FROM ''capacidades_admin''';
 runtime_marca168 text:=E'       OR NOT (\n           (p_perfil_mutacion IS NOT DISTINCT FROM ''publicacion_cargo_competencial''';
 runtime168 text:=E'       OR NOT (\n           (p_perfil_mutacion IS NOT DISTINCT FROM ''capacidades_admin''\n            AND EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit\n             AND NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls))\n            AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole\n             AND m.roleid=''vec_admin_perfiles_lector''::regrole\n             AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)\n            AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=''vec_admin_perfiles_lector''::regrole)\n            AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole)=1)\n           OR (p_perfil_mutacion IS NOT DISTINCT FROM ''publicacion_cargo_competencial''';
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
  RAISE EXCEPTION 'AD168: PARO clave=def_postAD166 actual=% esperado=%',actual_sha,esperado_def USING ERRCODE='55000'; END IF;
 actual_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(fuente,'UTF8')),'hex');
 IF actual_sha IS DISTINCT FROM esperado_fuente THEN
  RAISE EXCEPTION 'AD168: PARO clave=src_postAD166 actual=% esperado=%',actual_sha,esperado_fuente USING ERRCODE='55000'; END IF;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
  OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(COALESCE(acl,pg_catalog.acldefault('f',propietario))) a
    WHERE a.grantee=propietario AND a.grantor=propietario AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
  OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(COALESCE(acl,pg_catalog.acldefault('f',propietario))) a
    WHERE a.grantee<>propietario OR a.grantor<>propietario OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
  OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,marca,''))<>pg_catalog.length(marca)
  OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,excl_nuevo,''))<>pg_catalog.length(excl_nuevo)
  OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,runtime_marca168,''))<>pg_catalog.length(runtime_marca168)
  OR pg_catalog.strpos(original,'''capacidades_admin''')<>0
 THEN RAISE EXCEPTION 'AD168: PARO clave=preimagen_nucleo actual=incompatible esperado=AD165_exacto_y_ACL_privada' USING ERRCODE='55000'; END IF;
 nuevo:=pg_catalog.replace(original,runtime_marca168,runtime168);
 nuevo:=pg_catalog.replace(nuevo,excl_nuevo,excl168);
 nuevo:=pg_catalog.replace(nuevo,marca,extension168||marca);
 EXECUTE nuevo;
 SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
  OR pg_catalog.replace(pg_catalog.replace(pg_catalog.replace(actual,extension168||marca,marca),excl168,excl_nuevo),runtime168,runtime_marca168) IS DISTINCT FROM original
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
 THEN RAISE EXCEPTION 'AD168: PARO clave=postimagen_nucleo actual=divergente esperado=cambio_y_ACL_exactos' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text;nuevo text;esperado text:='6aadac0ec1307e2ce7f07527a185daf8974c0c8a413a2d5149250db092cf3f8b';
 audiencia text:='vec_autorizacion.administracion_perfiles.lectura.capacidades.v1';
 actual_sha text;
BEGIN
 SELECT pg_catalog.regexp_replace(pg_catalog.pg_get_constraintdef(c.oid,false),'\s+',' ','g') INTO STRICT d
 FROM pg_catalog.pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 actual_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(d,'UTF8')),'hex');
 IF actual_sha IS DISTINCT FROM esperado THEN
  RAISE EXCEPTION 'AD168: PARO clave=CHECK_postAD166 actual=% esperado=%',actual_sha,esperado USING ERRCODE='55000'; END IF;
 IF pg_catalog.strpos(d,'CHECK ((audiencia_consumo = ANY (ARRAY[')<>1 OR pg_catalog.right(d,4)<>'])))'
  OR pg_catalog.strpos(d,pg_catalog.quote_literal(audiencia))<>0
 THEN RAISE EXCEPTION 'AD168: PARO clave=CHECK_audiencia actual=incompatible esperado=AD165_sin_cargo' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 nuevo:=pg_catalog.left(d,pg_catalog.length(d)-4)||', '||pg_catalog.quote_literal(audiencia)||'::text])))';
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nuevo;
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_capacidades_admin_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb;d jsonb;x record;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable' OR pg_catalog.current_setting('transaction_read_only')<>'off'
 OR p_capacidad IS NULL OR pg_catalog.octet_length(p_capacidad) NOT BETWEEN 2 AND 32768
 OR p_decision IS NULL OR pg_catalog.octet_length(p_decision) NOT BETWEEN 2 AND 524288
 THEN RAISE EXCEPTION 'AD168: material denegado' USING ERRCODE='42501'; END IF;
 c:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_autorizacion.administracion_perfiles.lectura.capacidades.v1'
 OR c->>'operacion' IS DISTINCT FROM 'administracion.perfiles.consultar'
 OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'administracion'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'perfil' OR d->>'finalidad' IS DISTINCT FROM 'gestion_perfiles'
 OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR d->>'correlacion_ref' IS NULL OR d->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 OR d->'campos_permitidos' IS DISTINCT FROM '["actos_disponibles","perfil_ref","preimagen","version","vinculo_ref"]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 OR vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(
  d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
  'administracion.perfiles.consultar','administracion','perfil','gestion_perfiles',
  '["actos_disponibles","perfil_ref","preimagen","version","vinculo_ref"]'::jsonb,d->'vinculo_autenticacion_actor') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD168: decisión nominal denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'capacidades_admin',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 -- El permitido sólo sale tras acreditar el asiento común y su atestación.
 -- Una lectura nueva exige decisión nueva, incluso cuando se repite el recurso.
 IF x.consumo_nuevo IS NOT TRUE OR x.decision_ref IS DISTINCT FROM d->>'decision_ref'
 OR x.auditoria_ref IS NULL OR x.auditoria_ref !~ '^aud_v3_[0-9a-f]{32}$'
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
  JOIN vec_autorizacion_atestada_v3.consumo_decision_v3 s USING(decision_ref)
  JOIN vec_autorizacion_atestada_v3.atestacion_decision_v3 t USING(decision_ref)
  WHERE a.auditoria_ref=x.auditoria_ref AND a.decision_ref=x.decision_ref
   AND a.efecto_ref=x.efecto_ref AND a.huella_efecto_sha256=x.huella_efecto_sha256
   AND a.registrada_en=x.consumida_en AND s.consumida_en=x.consumida_en
   AND s.consumo_huella_sha256=x.consumo_huella_sha256
   AND t.decision_canonica=p_decision AND t.capacidad_canonica=p_capacidad)
 OR vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
 OR vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(
  d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
  'administracion.perfiles.consultar','administracion','perfil','gestion_perfiles',
  '["actos_disponibles","perfil_ref","preimagen","version","vinculo_ref"]'::jsonb,d->'vinculo_autenticacion_actor') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD168: acuse común o revalidación denegados' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_capacidades_admin_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_capacidades_admin_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_autorizacion_propietario;
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.consultar_capacidades_admin_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE m jsonb;d jsonb;c jsonb;claves text[];x record;huella text;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable' OR pg_catalog.current_setting('transaction_read_only')<>'off'
 OR p_material IS NULL OR pg_catalog.octet_length(p_material) NOT BETWEEN 2 AND 16384
 OR p_capacidad IS NULL OR pg_catalog.octet_length(p_capacidad) NOT BETWEEN 2 AND 32768
 OR p_decision IS NULL OR pg_catalog.octet_length(p_decision) NOT BETWEEN 2 AND 524288
 THEN RAISE EXCEPTION 'AD168: consulta denegada' USING ERRCODE='42501'; END IF;
 m:=p_material::jsonb;d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;c:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;
 SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") INTO claves FROM pg_catalog.jsonb_object_keys(m) k;
 huella:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_material,'UTF8')),'hex');
 IF claves IS DISTINCT FROM ARRAY['actor_perfil_ref','actor_persona_ref','asignacion_perfil_ref','consulta','correlacion_ref','esquema','limite','operacion_ref']::text[]
 OR pg_catalog.jsonb_path_exists(m,'$.** ? (@ == null)')
 OR m->>'esquema' IS DISTINCT FROM 'administracion_perfiles_lectura_v1' OR m->>'consulta' IS DISTINCT FROM 'capacidades'
 OR pg_catalog.jsonb_typeof(m->'limite') IS DISTINCT FROM 'number' OR m->>'limite' !~ '^[1-9][0-9]{0,2}$'
 OR m->>'actor_persona_ref' IS NULL OR m->>'actor_persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR m->>'actor_perfil_ref' IS NULL OR m->>'actor_perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
 OR m->>'actor_persona_ref' IS DISTINCT FROM d->>'principal_id'
 OR m->>'actor_perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
 OR m->>'asignacion_perfil_ref' IS DISTINCT FROM d->>'asignacion_ref'
 OR m->>'correlacion_ref' IS NULL OR m->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 OR m->>'correlacion_ref' IS DISTINCT FROM d->>'correlacion_ref'
 OR m->>'operacion_ref' IS DISTINCT FROM m->>'actor_perfil_ref'
 OR c->>'efecto_ref' IS DISTINCT FROM m->>'operacion_ref' OR c->>'huella_efecto_sha256' IS DISTINCT FROM huella
 THEN RAISE EXCEPTION 'AD168: consulta nominal denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_capacidades_admin_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 -- Ninguna otra fachada está publicada: no anunciar sus acciones por el rol.
 RETURN pg_catalog.jsonb_build_object('version','1','actor_persona_ref',d->>'principal_id','acciones','[]'::jsonb,
  'perfil_activo_ref',d->>'perfil_activo_ref','asignacion_perfil_ref',d->>'asignacion_ref',
  'correlacion_ref',d->>'correlacion_ref','auditoria_ref',x.auditoria_ref,'registrada_en',x.consumida_en);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.consultar_capacidades_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.consultar_capacidades_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_admin_perfiles_lector;
COMMIT;
