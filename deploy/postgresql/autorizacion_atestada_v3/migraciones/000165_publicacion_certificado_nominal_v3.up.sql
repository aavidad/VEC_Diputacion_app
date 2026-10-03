\set ON_ERROR_STOP on
-- AD165: operación V3 específica para publicar o retirar un certificado
-- nominal de firmante. El destinatario administrador queda denegado.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000165',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
  OR pg_catalog.to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
  OR pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL
  OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.publicar_certificado_firmante_ct_v2(bytea,text,text)') IS NULL
  OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_organizacion_destino_certificado_v1(text,text,text,numeric,text)') IS NULL
  OR pg_catalog.to_regprocedure('vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)') IS NULL
  OR pg_catalog.to_regprocedure('vec_autorizacion.acreditar_ambito_certificado_nominal_v1(text,text,text)') IS NULL
  OR pg_catalog.to_regprocedure('vec_autorizacion.destino_no_administrador_certificado_nominal_v1(text,text)') IS NULL
  OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
  OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_publicacion_certificado_nominal_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD165: dependencias causales incompatibles' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_autorizacion_certificado_nominal_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN
 EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO vec_autorizacion_certificado_nominal_ejecutor',pg_catalog.current_database());
END $conexion$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

-- Se inserta sólo el perfil y la audiencia nuevos. La preimagen exacta
-- post-AD159 se coteja en el clon PostgreSQL18 antes de instalar.
DO $nucleo$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text;fuente text;nuevo text;actual text;meta jsonb;acl aclitem[];deps jsonb;deps_compartidas jsonb;
 propietario oid;config text[];definidora boolean;
 esperado_def text:='7ccd6cb3e0c25a015a3a085af0be57dfd73cd3b48843317e727fb7677bedf0c8';
 esperado_fuente text:='f3d2c780bc7703d0df2c0e52016f7946c5e06e31fd6a3368a8cf89dc540939c5';
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'publicacion_certificado_nominal'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contexto_actor.certificado_nominal.publicar.v1'
 AND c->>'operacion' IN ('administracion.certificados.nominal.publicar','administracion.certificados.nominal.retirar')
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'administracion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'vinculo_certificado_nominal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_certificados_firmantes'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true'
 AND vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(
   d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
   c->>'operacion','administracion','vinculo_certificado_nominal','gestionar_certificados_firmantes',
   '[]'::jsonb,d->'vinculo_autenticacion_actor') IS TRUE)
$x$;
 excl text:=$x$p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'$x$;
 excl_nuevo text:=$x$p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento' AND p_perfil_mutacion IS DISTINCT FROM 'publicacion_certificado_nominal'$x$;
 runtime text:=E'       OR NOT (\n           (\n               p_perfil_mutacion IS DISTINCT FROM ''bolsa_llamamiento''';
 runtime_nuevo text:=E'       OR NOT (\n           (p_perfil_mutacion IS NOT DISTINCT FROM ''publicacion_certificado_nominal''\n            AND EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit\n             AND NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls))\n            AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole\n             AND m.roleid=''vec_autorizacion_certificado_nominal_ejecutor''::regrole\n             AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)\n            AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole)=1)\n           OR (\n               p_perfil_mutacion IS DISTINCT FROM ''bolsa_llamamiento''';
BEGIN
 SELECT pg_catalog.pg_get_functiondef(f),p.prosrc,pg_catalog.to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,fuente,meta,acl,propietario,config,definidora FROM pg_catalog.pg_proc p WHERE p.oid=f;
 SELECT COALESCE(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f;
 SELECT COALESCE(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO deps_compartidas FROM pg_catalog.pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database())
 AND d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f;
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(original,'UTF8')),'hex') IS DISTINCT FROM esperado_def
  OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM esperado_fuente
  OR propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
  OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(COALESCE(acl,pg_catalog.acldefault('f',propietario))) a
    WHERE a.grantee=propietario AND a.grantor=propietario AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
  OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(COALESCE(acl,pg_catalog.acldefault('f',propietario))) a
    WHERE a.grantee<>propietario OR a.grantor<>propietario OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
  OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,marca,''))<>pg_catalog.length(marca)
  OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,excl,''))<>pg_catalog.length(excl)
  OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,runtime,''))<>pg_catalog.length(runtime)
  OR pg_catalog.strpos(original,'''publicacion_certificado_nominal''')<>0
 THEN RAISE EXCEPTION 'AD165: preimagen núcleo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=pg_catalog.replace(original,runtime,runtime_nuevo);
 nuevo:=pg_catalog.replace(nuevo,excl,excl_nuevo);
 nuevo:=pg_catalog.replace(nuevo,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
  OR pg_catalog.replace(pg_catalog.replace(pg_catalog.replace(actual,extension||marca,marca),excl_nuevo,excl),runtime_nuevo,runtime) IS DISTINCT FROM original
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
 THEN RAISE EXCEPTION 'AD165: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text;nuevo text;esperado text:='531bfedaeaba478d66551c518922339682c8e1ab3a4167737f59d167f34dc831';
 audiencia text:='vec_contexto_actor.certificado_nominal.publicar.v1';
BEGIN
 SELECT pg_catalog.regexp_replace(pg_catalog.pg_get_constraintdef(c.oid,false),'\s+',' ','g') INTO STRICT d
 FROM pg_catalog.pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(d,'UTF8')),'hex') IS DISTINCT FROM esperado
  OR pg_catalog.strpos(d,'CHECK ((audiencia_consumo = ANY (ARRAY[')<>1 OR pg_catalog.right(d,4)<>'])))'
  OR pg_catalog.strpos(d,pg_catalog.quote_literal(audiencia))<>0
 THEN RAISE EXCEPTION 'AD165: preimagen audiencia incompatible' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 nuevo:=pg_catalog.left(d,pg_catalog.length(d)-4)||', '||pg_catalog.quote_literal(audiencia)||'::text])))';
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nuevo;
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_publicacion_certificado_nominal_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb;d jsonb;x record;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable' OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR p_capacidad IS NULL OR pg_catalog.octet_length(p_capacidad) NOT BETWEEN 2 AND 32768
  OR p_decision IS NULL OR pg_catalog.octet_length(p_decision) NOT BETWEEN 2 AND 524288
 THEN RAISE EXCEPTION 'AD165: transacción o material denegado' USING ERRCODE='42501'; END IF;
 c:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contexto_actor.certificado_nominal.publicar.v1'
  OR c->>'operacion' NOT IN ('administracion.certificados.nominal.publicar','administracion.certificados.nominal.retirar')
  OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'administracion'
  OR d->>'tipo_recurso' IS DISTINCT FROM 'vinculo_certificado_nominal'
  OR d->>'finalidad' IS DISTINCT FROM 'gestionar_certificados_firmantes'
  OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
  OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
  OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
  OR vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(
   d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
   c->>'operacion','administracion','vinculo_certificado_nominal','gestionar_certificados_firmantes',
   '[]'::jsonb,d->'vinculo_autenticacion_actor') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD165: decisión nominal denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'publicacion_certificado_nominal',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD165: requiere decisión actual' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_publicacion_certificado_nominal_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_publicacion_certificado_nominal_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_autorizacion_propietario;

RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.operar_certificado_nominal_v3(
 p_descriptor bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb;c jsonb;a jsonb;s jsonb;org jsonb;x record;resultado jsonb;accion text;sha text;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable' OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR p_descriptor IS NULL OR pg_catalog.octet_length(p_descriptor) NOT BETWEEN 2 AND 16384
 THEN RAISE EXCEPTION 'AD165: operación denegada' USING ERRCODE='42501'; END IF;
 d:=pg_catalog.convert_from(p_descriptor,'UTF8')::jsonb;
 c:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;
 a:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 accion:=CASE d->>'estado' WHEN 'vigente' THEN 'administracion.certificados.nominal.publicar'
   WHEN 'retirado' THEN 'administracion.certificados.nominal.retirar' ELSE NULL END;
 sha:=pg_catalog.encode(pg_catalog.sha256(p_descriptor),'hex');
 IF accion IS NULL OR c->>'operacion' IS DISTINCT FROM accion
  OR c->>'efecto_ref' IS DISTINCT FROM 'certificado-nominal:'||(d->>'certificado_der_sha256')
  OR c->>'huella_efecto_sha256' IS DISTINCT FROM sha
  OR a->>'contexto_recurso_huella_sha256' IS DISTINCT FROM sha
  OR a->>'principal_id' IS NOT DISTINCT FROM d->>'persona_ref'
 THEN RAISE EXCEPTION 'AD165: recurso denegado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_publicacion_certificado_nominal_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 s:=vec_contexto_actor_v1.fuentes_certificado_firmante_ct_v2(
  d->>'cuenta_ref',d->>'persona_ref',d->>'vinculo_cuenta_persona_ref');
 IF s IS NULL THEN RAISE EXCEPTION 'AD165: identidad destino no acreditada' USING ERRCODE='42501'; END IF;
 org:=vec_contexto_actor_v1.acreditar_organizacion_destino_certificado_v1(
  d->>'cuenta_ref',d->>'persona_ref',d->>'vinculo_cuenta_persona_ref',
  (s#>>'{vinculo_cuenta_persona,version}')::numeric,d->>'organizacion_ref');
 IF org IS NULL OR vec_autorizacion.acreditar_ambito_certificado_nominal_v1(
   a->>'version_rol_ref',a->>'asignacion_ref',org->>'organizacion_ref') IS NOT TRUE
  OR vec_autorizacion.destino_no_administrador_certificado_nominal_v1(d->>'cuenta_ref',d->>'persona_ref') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD165: destino denegado' USING ERRCODE='42501'; END IF;
 resultado:=vec_contexto_actor_v1.publicar_certificado_firmante_ct_v2(p_descriptor,x.decision_ref,x.auditoria_ref);
 IF resultado IS NULL OR vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
  OR vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(
   a->>'version_rol_ref',a->>'asignacion_ref',a->>'principal_id',a->>'perfil_activo_ref',
   accion,'administracion','vinculo_certificado_nominal','gestionar_certificados_firmantes','[]'::jsonb,a->'vinculo_autenticacion_actor') IS NOT TRUE
  OR vec_autorizacion.destino_no_administrador_certificado_nominal_v1(d->>'cuenta_ref',d->>'persona_ref') IS NOT TRUE
  OR vec_autorizacion.acreditar_ambito_certificado_nominal_v1(
   a->>'version_rol_ref',a->>'asignacion_ref',org->>'organizacion_ref') IS NOT TRUE
  OR vec_contexto_actor_v1.acreditar_organizacion_destino_certificado_v1(
   d->>'cuenta_ref',d->>'persona_ref',d->>'vinculo_cuenta_persona_ref',
   (s#>>'{vinculo_cuenta_persona,version}')::numeric,d->>'organizacion_ref') IS DISTINCT FROM org
 THEN RAISE EXCEPTION 'AD165: revalidación final denegada' USING ERRCODE='42501'; END IF;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.operar_certificado_nominal_v3(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_autorizacion_certificado_nominal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion.operar_certificado_nominal_v3(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_autorizacion_certificado_nominal_ejecutor;
COMMIT;
