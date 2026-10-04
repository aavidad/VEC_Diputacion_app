\set ON_ERROR_STOP on
-- AD190: consumo nominal V3 propio del lote ordinario ADMIN.
-- Borrador cerrado hasta incorporar huellas reales del core y CHECK post-AD189.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000190',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE f oid:=to_regprocedure('vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)');
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef AND p.prorettype='boolean'::regtype)
 OR has_function_privilege('vec_autorizacion_atestada_v3_propietario',f,'EXECUTE') IS NOT TRUE
 OR to_regrole('vec_admin_perfiles_lote_ejecutor') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_lote_ordinario_admin_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD190: gate AUT45 ausente o preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_admin_perfiles_lote_ejecutor NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $db$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_admin_perfiles_lote_ejecutor',current_database());END $db$;
SET LOCAL ROLE vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion.json_cadena_canonica_go_admin_v1(text) TO vec_autorizacion_atestada_v3_propietario;
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

-- Serializador de recurso compartido por el consumidor y la fachada AUT44.
-- Corresponde a json.Marshal({ambitos:{org,unidad},atributos:{Hsol}}).
CREATE FUNCTION vec_autorizacion_atestada_v3.recurso_lote_ordinario_admin_v1(p_solicitud text)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE m jsonb;solicitud_sha text;canon text;org text;unidad text;persona text;
BEGIN
 IF p_solicitud IS NULL OR octet_length(p_solicitud) NOT BETWEEN 2 AND 65536
 THEN RAISE EXCEPTION 'AD190: solicitud de lote invalida' USING ERRCODE='22023'; END IF;
 m:=p_solicitud::jsonb;org:=m->>'OrganizacionRef';
 unidad:=m#>>'{Cambios,0,Objetivo,UnidadRef}';
 persona:=m#>>'{Cambios,0,Objetivo,PersonaRef}';
 IF m->>'Esquema' IS DISTINCT FROM 'administracion_perfiles_lote:v3'
 OR org IS NULL OR org !~ '^[a-z][a-z0-9_:-]{2,127}$'
 OR unidad IS NULL OR unidad !~ '^[a-z][a-z0-9_:-]{2,127}$'
 OR persona IS NULL OR persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
 THEN RAISE EXCEPTION 'AD190: recurso de lote invalido' USING ERRCODE='22023'; END IF;
 solicitud_sha:=encode(sha256(convert_to(p_solicitud,'UTF8')),'hex');
 canon:='{"ambitos":{"organizacion_ref":'||vec_autorizacion.json_cadena_canonica_go_admin_v1(org)
  ||',"unidad_ref":'||vec_autorizacion.json_cadena_canonica_go_admin_v1(unidad)
  ||'},"atributos":{"solicitud_sha256":'||vec_autorizacion.json_cadena_canonica_go_admin_v1(solicitud_sha)||'}}';
 RETURN jsonb_build_object('recurso_ref',persona,'organizacion_ref',org,'unidad_ref',unidad,
  'solicitud_sha256',solicitud_sha,'contexto_canonico',canon,
  'contexto_sha256',encode(sha256(convert_to(canon,'UTF8')),'hex'));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.recurso_lote_ordinario_admin_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.recurso_lote_ordinario_admin_v1(text) TO vec_autorizacion_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.login_lote_ordinario_admin_valido_v1()
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT current_setting('role')='none' AND EXISTS(
 SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid JOIN pg_roles g ON g.oid=m.roleid
 WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
 AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
 AND l.rolconfig IS NULL AND g.oid=to_regrole('vec_admin_perfiles_lote_ejecutor')
 AND NOT(g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
 AND g.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
 AND (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1
 AND NOT EXISTS(SELECT 1 FROM pg_auth_members x WHERE x.member=g.oid)
 AND NOT EXISTS(SELECT 1 FROM pg_db_role_setting x WHERE x.setrole IN(l.oid,g.oid))
 AND NOT EXISTS(SELECT 1 FROM pg_shdepend x WHERE x.refclassid='pg_catalog.pg_authid'::regclass AND x.refobjid=l.oid));
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.login_lote_ordinario_admin_valido_v1() FROM PUBLIC;

DO $nucleo$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text;fuente text;nuevo text;actual text;meta jsonb;deps jsonb;compartidas jsonb;h text;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 runtime_marca text:=E'       OR NOT (\n           (p_perfil_mutacion IN (''usuarios_admin_listar'',''usuarios_admin_consultar'')';
 runtime_nuevo text:=E'       OR NOT (\n           (p_perfil_mutacion IS NOT DISTINCT FROM ''admin_perfiles_lote_ordinario''\n            AND vec_autorizacion_atestada_v3.login_lote_ordinario_admin_valido_v1() IS TRUE)\n           OR (p_perfil_mutacion IN (''usuarios_admin_listar'',''usuarios_admin_consultar'')';
 excl_marca text:=$x$p_perfil_mutacion IS DISTINCT FROM 'usuarios_admin_consultar'$x$;
 excl_nuevo text:=excl_marca||' AND p_perfil_mutacion IS DISTINCT FROM ''admin_perfiles_lote_ordinario''';
 extension190 text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'admin_perfiles_lote_ordinario'
 AND c->>'operacion' IS NOT DISTINCT FROM 'administracion.perfiles.aplicar_lote_ordinario'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_autorizacion.administracion_perfiles.lote_ordinario.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'administracion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_perfiles'
 AND d->>'garantia_minima' IS NOT DISTINCT FROM 'alto'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^per_[A-Za-z0-9_-]{22,128}$'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true'
 AND vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(
  d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
  'administracion.perfiles.aplicar_lote_ordinario','administracion','persona','gestion_perfiles',
  '[]'::jsonb,d->'vinculo_autenticacion_actor') IS TRUE)
$x$;
BEGIN
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 h:=encode(sha256(convert_to(original,'UTF8')),'hex');
 -- No se acepta una huella inferida de archivos: captura PostgreSQL real post-AD189.
 IF h IS DISTINCT FROM '536ea653143147e0cfb2d3277948530acb8d4d1643e44f4786462fe5ad1379fe' THEN
  RAISE EXCEPTION 'AD190: PARO clave=nucleo_def actual=%',h USING ERRCODE='55000'; END IF;
 h:=encode(sha256(convert_to(fuente,'UTF8')),'hex');
 IF h IS DISTINCT FROM 'b7eb48be035e854928c9685c916f9139166a197e3cae731510b3597f0fe40a45' THEN
  RAISE EXCEPTION 'AD190: PARO clave=nucleo_src actual=%',h USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee=p.proowner AND a.grantor=p.proowner AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
 OR length(original)-length(replace(original,marca,''))<>length(marca)
 OR length(original)-length(replace(original,runtime_marca,''))<>length(runtime_marca)
 OR length(original)-length(replace(original,excl_marca,''))<>length(excl_marca)
 OR strpos(original,'''admin_perfiles_lote_ordinario''')<>0
 THEN RAISE EXCEPTION 'AD190: PARO clave=preimagen_core' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO compartidas FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
 AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database());
 nuevo:=replace(original,marca,extension190||marca);
 nuevo:=replace(nuevo,excl_marca,excl_nuevo);
 nuevo:=replace(nuevo,runtime_marca,runtime_nuevo);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
 OR replace(replace(replace(actual,extension190||marca,marca),excl_nuevo,excl_marca),runtime_nuevo,runtime_marca) IS DISTINCT FROM original
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
     FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
     FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())) IS DISTINCT FROM compartidas
 THEN RAISE EXCEPTION 'AD190: PARO clave=postimagen_core' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencia$
DECLARE anterior text;h text;nuevo text;
BEGIN
 SELECT pg_get_constraintdef(oid,false) INTO STRICT anterior FROM pg_constraint
 WHERE conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND conname='clave_capacidad_version_audiencia_consumo_check' AND contype='c' AND convalidated;
 h:=encode(sha256(convert_to(anterior,'UTF8')),'hex');
 IF h IS DISTINCT FROM '62ad0be0944790785a298a8387dca798b03e392d3208b5d59adf06793a6f4236'
 OR left(anterior,7)<>'CHECK (' OR right(anterior,1)<>')'
 THEN RAISE EXCEPTION 'AD190: PARO clave=audiencia_CHECK actual=%',h USING ERRCODE='55000'; END IF;
 nuevo:='CHECK (('||substr(anterior,8,length(anterior)-8)||') OR audiencia_consumo IS NOT DISTINCT FROM ''vec_autorizacion.administracion_perfiles.lote_ordinario.v1'')';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nuevo;
END $audiencia$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_lote_ordinario_admin_v3_atestada(
 p_solicitud text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE c jsonb;d jsonb;x record;recurso jsonb;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 512 AND 32768
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
 OR vec_autorizacion_atestada_v3.login_lote_ordinario_admin_valido_v1() IS NOT TRUE
 THEN RAISE EXCEPTION 'AD190: material o login denegado' USING ERRCODE='42501'; END IF;
 c:=convert_from(p_capacidad,'UTF8')::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 recurso:=vec_autorizacion_atestada_v3.recurso_lote_ordinario_admin_v1(p_solicitud);
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_autorizacion.administracion_perfiles.lote_ordinario.v1'
 OR c->>'operacion' IS DISTINCT FROM 'administracion.perfiles.aplicar_lote_ordinario'
 OR d->>'accion' IS DISTINCT FROM c->>'operacion'
 OR d->>'modulo_id' IS DISTINCT FROM 'administracion'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'persona'
 OR d->>'finalidad' IS DISTINCT FROM 'gestion_perfiles'
 OR d->>'garantia_minima' IS DISTINCT FROM 'alto'
 OR d->>'recurso_ref' IS DISTINCT FROM recurso->>'recurso_ref'
 OR c->>'efecto_ref' IS DISTINCT FROM d->>'recurso_ref'
 OR c->>'huella_efecto_sha256' IS DISTINCT FROM recurso->>'contexto_sha256'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM recurso->>'contexto_sha256'
 OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '["auditar"]'::jsonb
 OR d->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'administracion_privilegiada'
 OR d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'true'
 OR vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(
  d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
  'administracion.perfiles.aplicar_lote_ordinario','administracion','persona','gestion_perfiles',
  '[]'::jsonb,d->'vinculo_autenticacion_actor') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD190: decision nominal denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'admin_perfiles_lote_ordinario',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.decision_ref IS DISTINCT FROM d->>'decision_ref'
 OR x.efecto_ref IS DISTINCT FROM d->>'recurso_ref'
 OR x.huella_efecto_sha256 IS DISTINCT FROM recurso->>'contexto_sha256'
 OR x.auditoria_ref IS NULL OR x.auditoria_ref !~ '^aud_v3_[0-9a-f]{32}$'
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
  JOIN vec_autorizacion_atestada_v3.consumo_decision_v3 s USING(decision_ref)
  JOIN vec_autorizacion_atestada_v3.atestacion_decision_v3 t USING(decision_ref)
  WHERE a.auditoria_ref=x.auditoria_ref AND a.tipo_registro='consumo_confirmado_v3' AND a.version_consumo=3
  AND a.efecto_ref=x.efecto_ref AND a.huella_efecto_sha256=x.huella_efecto_sha256
  AND a.actor_ref=d->>'principal_id' AND a.perfil_activo_ref=d->>'perfil_activo_ref'
  AND a.finalidad_ref='gestion_perfiles' AND a.registrada_en=x.consumida_en
  AND s.consumida_en=x.consumida_en AND s.consumo_huella_sha256=x.consumo_huella_sha256
  AND t.capacidad_canonica=p_capacidad AND t.decision_canonica=p_decision
  AND t.motivo_canonico=p_motivo AND t.contexto_actor_canonico=p_contexto
  AND t.payload_vec_ad_3=p_payload AND t.sobre_cose_sign1=p_sobre
  AND t.evidencia_verificacion=p_evidencia AND t.raiz_publica_spki=p_raiz)
 OR vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
  p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
 OR vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(
  d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
  'administracion.perfiles.aplicar_lote_ordinario','administracion','persona','gestion_perfiles',
  '[]'::jsonb,d->'vinculo_autenticacion_actor') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD190: acuse o revalidacion denegados' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,
  x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_lote_ordinario_admin_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_lote_ordinario_admin_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_autorizacion_propietario;
COMMIT;
