\set ON_ERROR_STOP on
-- AUT51: mantenimiento privado Rol6→Rol7 del perfil fijo de Aplicación para el
-- gobierno del plan nominal de firma (AD177). Añade sólo vec.catalogos.{crear,
-- actualizar,publicar,retirar} sobre catalogo_configurable de Contratación
-- temporal y generaliza a la versión vigente la puerta del lote de AUT45.
-- Instalar la estructura no publica Rol7: lo hace la CLI con plan aprobado.
BEGIN;
SET LOCAL search_path=pg_catalog;SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) OR current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999 THEN RAISE EXCEPTION 'AUT51: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501';END IF;
 IF to_regclass('vec_autorizacion.config_mantenimiento_lote_admin_v1') IS NULL
 OR to_regprocedure('vec_autorizacion.acreditar_perfil_aplicacion_nominal_vigente_aut48(text,text,text,text,text,text,text,text,jsonb,jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_mantenimiento_perfil_fijo_admin_v1(jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion.ajustar_inicio_asignacion_mantenimiento_v1(jsonb)') IS NULL
 OR to_regclass('vec_autorizacion.config_mantenimiento_plan_firma_admin_v1') IS NOT NULL
 OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_admin_mantenimiento_plan_firma_ejecutor')
 THEN RAISE EXCEPTION 'AUT51: PARO clave=dependencias actual=divergente esperado=AUT45_AUT48_AD183_sin_AUT51' USING ERRCODE='55000';END IF;
END $pre$;
-- La puerta del lote que se sustituye debe ser exactamente la de AUT45.
DO $guarda$ DECLARE actual text;BEGIN
 SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') INTO actual FROM pg_proc WHERE oid=to_regprocedure('vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)') AND proowner='vec_autorizacion_propietario'::regrole AND prosecdef;
 IF actual IS DISTINCT FROM 'f6cbcb22c9be6267b74e17dcf7bdb9b83ba49018c5a15b44c575270db9c6eea9' THEN RAISE EXCEPTION 'AUT51: PARO clave=acreditar_perfil_aplicacion_lote_ordinario_v1.prosrc actual=% esperado=f6cbcb22c9be6267b74e17dcf7bdb9b83ba49018c5a15b44c575270db9c6eea9',COALESCE(actual,'ausente') USING ERRCODE='55000';END IF;
END $guarda$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.concesiones_plan_firma_admin_v1() RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT $datos$[{"accion":"vec.catalogos.crear","modulo_id":"contratacion_temporal","tipo_recurso":"catalogo_configurable","finalidades":["gestionar_contratacion_temporal"],"garantia_minima":"alto","campos_permitidos":[],"obligaciones":[]},{"accion":"vec.catalogos.actualizar","modulo_id":"contratacion_temporal","tipo_recurso":"catalogo_configurable","finalidades":["gestionar_contratacion_temporal"],"garantia_minima":"alto","campos_permitidos":[],"obligaciones":[]},{"accion":"vec.catalogos.publicar","modulo_id":"contratacion_temporal","tipo_recurso":"catalogo_configurable","finalidades":["gestionar_contratacion_temporal"],"garantia_minima":"alto","campos_permitidos":[],"obligaciones":[]},{"accion":"vec.catalogos.retirar","modulo_id":"contratacion_temporal","tipo_recurso":"catalogo_configurable","finalidades":["gestionar_contratacion_temporal"],"garantia_minima":"alto","campos_permitidos":[],"obligaciones":[]}]$datos$::jsonb
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.concesiones_plan_firma_admin_v1() FROM PUBLIC;
CREATE TABLE vec_autorizacion.config_mantenimiento_plan_firma_admin_v1(
 login_nombre name PRIMARY KEY,
 plan_sha256 text NOT NULL UNIQUE CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 preimagen_sha256 text NOT NULL CHECK(preimagen_sha256 ~ '^[0-9a-f]{64}$'),
 catalogo_sha256 text NOT NULL CHECK(catalogo_sha256 ~ '^[0-9a-f]{64}$'),
 aprobacion_ref text NOT NULL CHECK(aprobacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
 aprobacion_sha256 text NOT NULL CHECK(aprobacion_sha256 ~ '^[0-9a-f]{64}$'),
 entorno text NOT NULL CHECK(entorno='desarrollo'),
 vigente_desde timestamptz NOT NULL,vigente_hasta timestamptz NOT NULL,
 CHECK(isfinite(vigente_desde) AND isfinite(vigente_hasta) AND vigente_hasta>vigente_desde)
);
ALTER TABLE vec_autorizacion.config_mantenimiento_plan_firma_admin_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.config_mantenimiento_plan_firma_admin_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.config_mantenimiento_plan_firma_admin_v1 TO vec_autorizacion_propietario USING(current_user='vec_autorizacion_propietario') WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.config_mantenimiento_plan_firma_admin_v1 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.config_mantenimiento_plan_firma_admin_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.config_mantenimiento_plan_firma_admin_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.config_mantenimiento_plan_firma_admin_v1 FROM PUBLIC;
RESET ROLE;
CREATE ROLE vec_admin_mantenimiento_plan_firma_ejecutor NOLOGIN NOINHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_admin_mantenimiento_plan_firma_ejecutor',current_database()); END $conexion$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.exigir_operador_mantenimiento_plan_firma_admin_v1()
RETURNS vec_autorizacion.config_mantenimiento_plan_firma_admin_v1 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE l record;g record;cfg vec_autorizacion.config_mantenimiento_plan_firma_admin_v1;ns oid;db oid;f oid;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' OR current_setting('role')<>'none' THEN RAISE EXCEPTION 'AUT51: PARO clave=transaccion actual=divergente esperado=SERIALIZABLE_RW_UTC_sin_SETROLE' USING ERRCODE='25000'; END IF;
 SELECT * INTO l FROM pg_roles WHERE rolname=session_user;SELECT * INTO g FROM pg_roles WHERE rolname='vec_admin_mantenimiento_plan_firma_ejecutor';
 ns:=to_regnamespace('vec_autorizacion');SELECT oid INTO db FROM pg_database WHERE datname=current_database();f:=to_regprocedure('vec_autorizacion.mantener_version_perfil_fijo_plan_firma_admin_v1(text,text)');
 IF l.oid IS NULL OR g.oid IS NULL OR NOT l.rolcanlogin OR NOT l.rolinherit OR l.rolsuper OR l.rolcreaterole OR l.rolcreatedb OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
 OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreaterole OR g.rolcreatedb OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
 OR (SELECT count(*) FROM pg_auth_members WHERE member=l.oid)<>1 OR NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=l.oid AND roleid=g.oid AND inherit_option AND NOT set_option AND NOT admin_option)
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=g.oid) OR EXISTS(SELECT 1 FROM pg_db_role_setting WHERE setrole IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=l.oid)
 OR EXISTS(SELECT 1 FROM pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=g.oid AND NOT((dbid=db AND classid='pg_catalog.pg_namespace'::regclass AND objid=ns AND deptype='a') OR(dbid=db AND classid='pg_catalog.pg_proc'::regclass AND objid=f AND deptype='a') OR(dbid=0 AND classid='pg_catalog.pg_database'::regclass AND objid=db AND deptype='a')))
 -- Dependencias conservan el objeto al añadir GRANT OPTION; se coteja la ACL.
 OR NOT COALESCE((SELECT count(*)=1 AND bool_and(x.privilege_type='CONNECT' AND NOT x.is_grantable) FROM pg_database d CROSS JOIN LATERAL aclexplode(COALESCE(d.datacl,acldefault('d',d.datdba))) x WHERE d.oid=db AND x.grantee=g.oid),false)
 OR NOT COALESCE((SELECT count(*)=1 AND bool_and(x.privilege_type='USAGE' AND NOT x.is_grantable) FROM pg_namespace n CROSS JOIN LATERAL aclexplode(COALESCE(n.nspacl,acldefault('n',n.nspowner))) x WHERE n.oid=ns AND x.grantee=g.oid),false)
 OR NOT COALESCE((SELECT count(*)=1 AND bool_and(x.privilege_type='EXECUTE' AND NOT x.is_grantable) FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee=g.oid),false)
 OR EXISTS(SELECT 1 FROM pg_database d CROSS JOIN LATERAL aclexplode(COALESCE(d.datacl,acldefault('d',d.datdba))) x WHERE d.oid=db AND x.grantee=l.oid)
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(COALESCE(n.nspacl,acldefault('n',n.nspowner))) x WHERE n.oid=ns AND x.grantee=l.oid)
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee=l.oid)
 OR has_schema_privilege(l.oid,ns,'CREATE') OR has_database_privilege(l.oid,db,'CREATE,TEMP') THEN RAISE EXCEPTION 'AUT51: PARO clave=operador actual=no_acreditado esperado=LOGIN_minimo_exclusivo' USING ERRCODE='42501'; END IF;
 SELECT * INTO cfg FROM vec_autorizacion.config_mantenimiento_plan_firma_admin_v1 WHERE login_nombre=session_user FOR SHARE;
 IF NOT FOUND OR clock_timestamp()<cfg.vigente_desde OR clock_timestamp()>=cfg.vigente_hasta THEN RAISE EXCEPTION 'AUT51: PARO clave=configuracion actual=ausente_o_caducada esperado=aprobacion_externa_vigente' USING ERRCODE='42501'; END IF;
 RETURN cfg;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.exigir_operador_mantenimiento_plan_firma_admin_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.documento_asignacion_destino_mantenimiento_plan_firma_v1(a jsonb,p jsonb,operador text)
RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT vec_autorizacion.ajustar_inicio_asignacion_mantenimiento_v1(jsonb_set(jsonb_set(jsonb_set(jsonb_set(a,'{version}','4'),'{version_rol_ref}','"rol:administracion_perfiles:v7"'),'{emitida_por}',to_jsonb('mantenimiento_operador:'||operador)),'{emitida_en}',p#>'{rol_destino_doc,publicada_en}'))
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.documento_asignacion_destino_mantenimiento_plan_firma_v1(jsonb,jsonb,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.preimagen_mantenimiento_plan_firma_admin_v1(p jsonb)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE r record;c record;meta record;gob record;a record;ptr record;t jsonb;asigs jsonb:='[]';cat jsonb;amb jsonb;ca_viva boolean;efectivos jsonb;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'AUT51: PARO clave=transaccion actual=divergente esperado=SERIALIZABLE_RW' USING ERRCODE='25000'; END IF;
 IF jsonb_typeof(p) IS DISTINCT FROM 'object' OR NOT p ?& ARRAY['version','operacion_ref','preparado_en','caduca_en','rol_origen_sha256','control_revision_esperada','control_huella_sha256','catalogo_sha256','rol_destino_doc','asignaciones']
 OR (SELECT count(*) FROM jsonb_object_keys(p))<>10 OR p->>'version' IS DISTINCT FROM '3'
 OR p->>'operacion_ref' !~ '^pmf_[A-Za-z0-9_-]{22,124}$' OR jsonb_typeof(p->'asignaciones') IS DISTINCT FROM 'array' OR jsonb_array_length(p->'asignaciones')<>2
 OR (p->>'preparado_en')::timestamptz>clock_timestamp() OR (p->>'caduca_en')::timestamptz<=clock_timestamp() OR (p->>'caduca_en')::timestamptz<=(p->>'preparado_en')::timestamptz
 THEN RAISE EXCEPTION 'AUT51: PARO clave=plan actual=invalido esperado=plan_cerrado_vigente_dos_APP' USING ERRCODE='22023'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 efectivos:=vec_autorizacion.administradores_aplicacion_efectivos_internos_v3();
 SELECT * INTO STRICT r FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v6' FOR SHARE;
 SELECT x.* INTO STRICT c FROM vec_autorizacion.control_vigencia_version_rol_actual ca JOIN vec_autorizacion.control_vigencia_version_rol x USING(version_rol_ref,revision) WHERE ca.version_rol_ref=r.version_rol_ref FOR UPDATE OF ca;
 SELECT * INTO STRICT meta FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 WHERE version_rol_ref=r.version_rol_ref FOR SHARE;
 SELECT * INTO STRICT gob FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=r.version_rol_ref FOR SHARE;
 SELECT COALESCE(jsonb_agg(to_jsonb(x) ORDER BY x.accion_ref,x.version),'[]') INTO cat FROM vec_autorizacion.catalogo_accion_nominal_v1 x WHERE x.version_rol_ref=r.version_rol_ref;
 IF r.huella_sha256 IS DISTINCT FROM p->>'rol_origen_sha256' OR r.documento->>'estado'<>'publicada' OR r.huella_sha256 IS DISTINCT FROM encode(pg_catalog.sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
 OR c.estado<>'habilitada' OR c.revision::text IS DISTINCT FROM p->>'control_revision_esperada' OR c.huella_sha256 IS DISTINCT FROM p->>'control_huella_sha256' OR c.huella_sha256 IS DISTINCT FROM encode(pg_catalog.sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(c.documento),'UTF8')),'hex')
 OR meta.categoria_administrativa<>'aplicacion' OR meta.tipo_perfil<>'fijo_sistema' OR meta.version_rol_huella_sha256<>r.huella_sha256 OR meta.fuente_ref<>r.version_rol_ref OR meta.fuente_version<>6 OR meta.fuente_huella_sha256<>r.huella_sha256
 OR gob.clase<>'administrador' OR gob.huella_sha256<>r.huella_sha256
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(cat) x WHERE x->>'fuente_ref'<>r.version_rol_ref OR x->>'fuente_version'<>'6' OR x->>'fuente_huella_sha256'<>r.huella_sha256 OR x->>'clase_control'<>'administrador_aplicacion' OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(r.documento->'concesiones') v WHERE v=x->'concesion'))
 OR encode(pg_catalog.sha256(convert_to(cat::text,'UTF8')),'hex') IS DISTINCT FROM p->>'catalogo_sha256'
 OR EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v7')
 OR jsonb_array_length(efectivos)<>2
 THEN RAISE EXCEPTION 'AUT51: PARO clave=fuente actual=divergente esperado=rol6_catalogo_y_dos_APP_vivos' USING ERRCODE='40001'; END IF;
 IF (p->'rol_destino_doc') IS DISTINCT FROM jsonb_set(jsonb_set(jsonb_set(jsonb_set(r.documento,'{version}','7'),'{concesiones}',r.documento->'concesiones'||vec_autorizacion.concesiones_plan_firma_admin_v1()),'{publicada_por}',p#>'{rol_destino_doc,publicada_por}'),'{publicada_en}',p#>'{rol_destino_doc,publicada_en}')
 OR p#>>'{rol_destino_doc,publicada_en}' IS DISTINCT FROM p->>'preparado_en' OR vec_autorizacion.concesiones_positivas_validas(p->'rol_destino_doc') IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT51: PARO clave=rol_destino actual=divergente esperado=clone6_mas_plan_firma_cerrado' USING ERRCODE='22023'; END IF;
 FOR t IN SELECT value FROM jsonb_array_elements(p->'asignaciones') ORDER BY value->>'perfil_ref' LOOP
  IF jsonb_typeof(t) IS DISTINCT FROM 'object' OR NOT t ?& ARRAY['perfil_ref','asignacion_origen_ref','asignacion_origen_sha256','persona_ref','cuenta_ref','vinculo_ref','cuenta_version','persona_version','perfil_version','vinculo_version','ambitos_fuente']
  OR (SELECT count(*) FROM jsonb_object_keys(t))<>11 THEN RAISE EXCEPTION 'AUT51: PARO clave=objetivo actual=invalido esperado=campos_exactos_de_asignacion' USING ERRCODE='22023'; END IF;
  SELECT x.* INTO STRICT a FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref) WHERE q.perfil_activo_ref=t->>'perfil_ref' FOR UPDATE OF q;
  IF a.asignacion_ref IS DISTINCT FROM t->>'asignacion_origen_ref' OR a.huella_sha256 IS DISTINCT FROM t->>'asignacion_origen_sha256' OR a.version<>3 OR a.version_rol_ref<>r.version_rol_ref OR a.principal_id IS DISTINCT FROM t->>'persona_ref' OR a.documento->>'estado'<>'activa'
  OR a.huella_sha256 IS DISTINCT FROM encode(pg_catalog.sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(a.documento),'UTF8')),'hex')
  OR clock_timestamp()<(a.documento->>'vigente_desde')::timestamptz OR (p->>'caduca_en')::timestamptz>=(a.documento->>'vigente_hasta')::timestamptz
  OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil h WHERE h.asignacion_id=a.asignacion_id AND h.documento->>'estado'='revocada')
  THEN RAISE EXCEPTION 'AUT51: PARO clave=asignacion actual=divergente esperado=APP6_v3_viva_sin_revocacion' USING ERRCODE='40001'; END IF;
  ca_viva:=vec_contexto_actor_v1.bloquear_contexto_admin_v1(t->>'cuenta_ref',t->>'persona_ref',t->>'perfil_ref',t->>'vinculo_ref',(t->>'cuenta_version')::numeric,(t->>'persona_version')::numeric,(t->>'perfil_version')::numeric,(t->>'vinculo_version')::numeric);
  amb:=vec_autorizacion.cotejar_ambitos_bootstrap_central_admin_v3(t->'ambitos_fuente',(a.documento->>'vigente_hasta')::timestamptz);
  IF ca_viva IS NOT TRUE OR amb->'ambitos' IS DISTINCT FROM a.documento->'ambitos' OR jsonb_array_length(a.documento->'ambitos')<>2 THEN RAISE EXCEPTION 'AUT51: PARO clave=CA_ambitos actual=divergente esperado=CA_y_org_unidad_propietarias' USING ERRCODE='42501'; END IF;
  asigs:=asigs||jsonb_build_array(jsonb_build_object('asignacion',to_jsonb(a),'fuentes_ambito',amb,'contexto',t));
 END LOOP;
 IF (SELECT count(DISTINCT x.value#>>'{asignacion,principal_id}') FROM jsonb_array_elements(asigs) x)<>2 OR (SELECT count(DISTINCT x.value#>>'{asignacion,perfil_activo_ref}') FROM jsonb_array_elements(asigs) x)<>2 THEN RAISE EXCEPTION 'AUT51: PARO clave=dos_APP actual=duplicadas esperado=dos_personas_perfiles_distintos' USING ERRCODE='22023'; END IF;
 IF (SELECT jsonb_agg(jsonb_build_object('asignacion_ref',x.value#>>'{asignacion,asignacion_ref}','persona_ref',x.value#>>'{asignacion,principal_id}','perfil_ref',x.value#>>'{asignacion,perfil_activo_ref}') ORDER BY x.value#>>'{asignacion,perfil_activo_ref}') FROM jsonb_array_elements(asigs) x)
 IS DISTINCT FROM (SELECT jsonb_agg(jsonb_build_object('asignacion_ref',x.value->>'asignacion_ref','persona_ref',x.value->>'persona_ref','perfil_ref',x.value->>'perfil_ref') ORDER BY x.value->>'perfil_ref') FROM jsonb_array_elements(efectivos) x)
 THEN RAISE EXCEPTION 'AUT51: PARO clave=APP_objetivos actual=conjunto_distinto esperado=dos_APP_efectivas_exactas' USING ERRCODE='40001'; END IF;
 RETURN jsonb_build_object('esquema','vec.admin.mantenimiento-fijo.preimagen.v3','rol',to_jsonb(r),'control',to_jsonb(c),'categoria',to_jsonb(meta),'gobierno',to_jsonb(gob),'catalogo',cat,'asignaciones',asigs);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.preimagen_mantenimiento_plan_firma_admin_v1(jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.aplicar_mantenimiento_plan_firma_admin_v1(plan_canonico text,sha_aprobado text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' AS $f$
DECLARE cfg vec_autorizacion.config_mantenimiento_plan_firma_admin_v1;p jsonb;sha text;pre jsonb;pre_sha text;r6 record;r7 record;target jsonb;target_sha text;control jsonb;control_sha text;c jsonb;meta record;gob record;t jsonb;old_a record;new_a record;doc jsonb;ref text;cat jsonb;events jsonb:='[]';aud record;e jsonb;corr text;recibo jsonb;replay boolean:=false;instante timestamptz;origenes jsonb:='[]';destinos jsonb:='[]';i int:=0;sello record;
BEGIN
 cfg:=vec_autorizacion.exigir_operador_mantenimiento_plan_firma_admin_v1();
 IF plan_canonico IS NULL OR octet_length(plan_canonico) NOT BETWEEN 1 AND 65536 OR sha_aprobado IS DISTINCT FROM cfg.plan_sha256 THEN RAISE EXCEPTION 'AUT51: PARO clave=plan actual=divergente esperado=plan_privado_aprobado' USING ERRCODE='42501'; END IF;
 sha:=encode(pg_catalog.sha256(convert_to(plan_canonico,'UTF8')),'hex');p:=plan_canonico::jsonb;
 IF sha IS DISTINCT FROM cfg.plan_sha256 OR p->>'catalogo_sha256' IS DISTINCT FROM cfg.catalogo_sha256 OR p#>>'{rol_destino_doc,publicada_por}' IS DISTINCT FROM 'mantenimiento_operador:'||session_user::text
 OR (p->>'caduca_en')::timestamptz<=clock_timestamp() THEN RAISE EXCEPTION 'AUT51: PARO clave=aprobacion actual=divergente esperado=operador_plan_catalogo_vigentes' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO r7 FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v7' FOR SHARE;replay:=FOUND;
 SELECT * INTO STRICT r6 FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v6' FOR SHARE;
 target:=p->'rol_destino_doc';target_sha:=encode(pg_catalog.sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(target),'UTF8')),'hex');
 IF NOT replay THEN
  pre:=vec_autorizacion.preimagen_mantenimiento_plan_firma_admin_v1(p);pre_sha:=encode(pg_catalog.sha256(convert_to(pre::text,'UTF8')),'hex');
  IF pre_sha IS DISTINCT FROM cfg.preimagen_sha256 THEN RAISE EXCEPTION 'AUT51: PARO clave=preimagen actual=% esperado=%',pre_sha,cfg.preimagen_sha256 USING ERRCODE='40001'; END IF;
 ELSE
  pre_sha:=cfg.preimagen_sha256;
  IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.control_vigencia_version_rol_actual q JOIN vec_autorizacion.control_vigencia_version_rol x USING(version_rol_ref,revision) WHERE q.version_rol_ref='rol:administracion_perfiles:v7' AND x.estado='habilitada') THEN RAISE EXCEPTION 'AUT51: PARO clave=replay_control actual=no_vigente esperado=rol7_habilitado' USING ERRCODE='40001'; END IF;
  IF r7.huella_sha256 IS DISTINCT FROM target_sha OR r7.documento IS DISTINCT FROM target OR r7.documento->>'estado'<>'publicada' OR r6.huella_sha256 IS DISTINCT FROM p->>'rol_origen_sha256' THEN RAISE EXCEPTION 'AUT51: PARO clave=replay_rol actual=divergente esperado=publicacion_original_viva' USING ERRCODE='40001'; END IF;
 END IF;
 FOR t IN SELECT value FROM jsonb_array_elements(p->'asignaciones') ORDER BY value->>'perfil_ref' LOOP
  SELECT * INTO STRICT old_a FROM vec_autorizacion.asignacion_perfil WHERE asignacion_ref=t->>'asignacion_origen_ref';
  IF old_a.huella_sha256 IS DISTINCT FROM t->>'asignacion_origen_sha256' OR old_a.version_rol_ref<>'rol:administracion_perfiles:v6' OR old_a.version<>3 OR old_a.principal_id IS DISTINCT FROM t->>'persona_ref' OR old_a.perfil_activo_ref IS DISTINCT FROM t->>'perfil_ref' THEN RAISE EXCEPTION 'AUT51: PARO clave=origen actual=divergente esperado=asignacion_original6' USING ERRCODE='40001'; END IF;
  doc:=vec_autorizacion.documento_asignacion_destino_mantenimiento_plan_firma_v1(old_a.documento,p,session_user::text);ref:='asignacion:'||old_a.asignacion_id||':v4';
  origenes:=origenes||jsonb_build_array(jsonb_build_object('ref',old_a.asignacion_ref,'sha',old_a.huella_sha256));destinos:=destinos||jsonb_build_array(jsonb_build_object('ref',ref,'sha',encode(pg_catalog.sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(doc),'UTF8')),'hex'),'documento',doc));
  IF replay THEN
   SELECT x.* INTO STRICT new_a FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref) WHERE q.perfil_activo_ref=t->>'perfil_ref' FOR SHARE OF q;
   SELECT * INTO STRICT sello FROM vec_autorizacion.sello_efecto_admin_tx_v1 WHERE asignacion_ref=ref;
   IF new_a.asignacion_ref IS DISTINCT FROM ref OR new_a.documento IS DISTINCT FROM doc OR new_a.huella_sha256 IS DISTINCT FROM destinos#>>'{-1,sha}' OR new_a.documento->>'estado'<>'activa' OR clock_timestamp()>=(new_a.documento->>'vigente_hasta')::timestamptz OR sello.operacion_ref IS DISTINCT FROM p->>'operacion_ref' THEN RAISE EXCEPTION 'AUT51: PARO clave=replay_asignacion actual=revocada_o_divergente esperado=destino_original_sin_rescate' USING ERRCODE='40001'; END IF;
   IF vec_contexto_actor_v1.bloquear_contexto_admin_v1(t->>'cuenta_ref',t->>'persona_ref',t->>'perfil_ref',t->>'vinculo_ref',(t->>'cuenta_version')::numeric,(t->>'persona_version')::numeric,(t->>'perfil_version')::numeric,(t->>'vinculo_version')::numeric) IS NOT TRUE
   OR vec_autorizacion.cotejar_ambitos_bootstrap_central_admin_v3(t->'ambitos_fuente',(new_a.documento->>'vigente_hasta')::timestamptz)->'ambitos' IS DISTINCT FROM new_a.documento->'ambitos' THEN
    RAISE EXCEPTION 'AUT51: PARO clave=replay_CA_ambitos actual=divergente esperado=CA_y_ambitos_originales_vivos' USING ERRCODE='42501';
   END IF;
  END IF;
 END LOOP;
 corr:='correlacion_'||substr(sha,33,32);
 e:=jsonb_build_object('tipo_registro','mantenimiento_perfil_fijo_admin','evento_ref','evento_'||substr(sha,1,32),'operador_login',session_user::text,'plan_sha256',sha,'preimagen_sha256',pre_sha,'catalogo_sha256',cfg.catalogo_sha256,
  'rol_origen_ref','rol:administracion_perfiles:v6','rol_origen_sha256',r6.huella_sha256,'rol_destino_ref','rol:administracion_perfiles:v7','rol_destino_sha256',target_sha,
  'asignacion_1_origen_ref',origenes#>>'{0,ref}','asignacion_1_origen_sha256',origenes#>>'{0,sha}','asignacion_1_destino_ref',destinos#>>'{0,ref}','asignacion_1_destino_sha256',destinos#>>'{0,sha}',
  'asignacion_2_origen_ref',origenes#>>'{1,ref}','asignacion_2_origen_sha256',origenes#>>'{1,sha}','asignacion_2_destino_ref',destinos#>>'{1,ref}','asignacion_2_destino_sha256',destinos#>>'{1,sha}',
  'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','mantenimiento_perfil_fijo_admin','correlacion_ref',corr);
 SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_mantenimiento_perfil_fijo_admin_v1(e);
 IF replay THEN
  IF EXISTS(SELECT 1 FROM jsonb_array_elements(destinos) x JOIN vec_autorizacion.sello_efecto_admin_tx_v1 z ON z.asignacion_ref=x.value->>'ref' WHERE z.auditoria_ref IS DISTINCT FROM aud.auditoria_ref OR z.operacion_ref IS DISTINCT FROM p->>'operacion_ref') THEN RAISE EXCEPTION 'AUT51: PARO clave=acuse actual=divergente esperado=confirmacion_comun_original' USING ERRCODE='40001'; END IF;
 ELSE
  instante:=(target->>'publicada_en')::timestamptz;
  INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento) VALUES('rol:administracion_perfiles:v7','administracion_perfiles',7,target_sha,instante,target);
  control:=jsonb_build_object('version_rol_ref','rol:administracion_perfiles:v7','revision',1,'estado','habilitada','actualizado_por','mantenimiento_operador:'||session_user::text,'actualizado_en',target->>'publicada_en');control_sha:=encode(pg_catalog.sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(control),'UTF8')),'hex');
  INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento,creada_en) VALUES('rol:administracion_perfiles:v7',1,'habilitada',control_sha,instante,control,clock_timestamp());
  INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual VALUES('rol:administracion_perfiles:v7',1,instante,'mantenimiento_operador:'||session_user::text,p->>'operacion_ref');
  INSERT INTO vec_autorizacion.rol_sensible_exacto VALUES('rol:administracion_perfiles:v7','administrador',target_sha);
  INSERT INTO vec_autorizacion.perfil_fijo_categoria_nominal_v1 VALUES('rol:administracion_perfiles:v7',target_sha,'aplicacion','fijo_sistema','rol:administracion_perfiles:v7',7,target_sha,instante);
  SELECT * INTO STRICT gob FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref='rol:administracion_perfiles:v6';
  INSERT INTO vec_autorizacion.rol_administrable_exacto_v1 SELECT 'rol:administracion_perfiles:v7',gob.clase,target_sha,gob.vigente_desde,gob.vigente_hasta,gob.unidad_requerida,gob.audiencia_administrativa,gob.ambitos_fijos,gob.duracion_propuesta;
  FOR c IN SELECT to_jsonb(x) FROM vec_autorizacion.catalogo_accion_nominal_v1 x WHERE x.version_rol_ref='rol:administracion_perfiles:v6' ORDER BY x.accion_ref LOOP
   INSERT INTO vec_autorizacion.catalogo_accion_nominal_v1 VALUES(c->>'accion_ref',(c->>'version')::int+1,'rol:administracion_perfiles:v7',7,target_sha,'rol:administracion_perfiles:v7',c->'concesion',c->'dimensiones_ambito','administrador_aplicacion',instante,(c->>'vigente_hasta')::timestamptz);
  END LOOP;
  FOR c IN SELECT value FROM jsonb_array_elements(vec_autorizacion.concesiones_plan_firma_admin_v1()) LOOP
   INSERT INTO vec_autorizacion.catalogo_accion_nominal_v1 VALUES('accion:'||(c->>'accion'),1,'rol:administracion_perfiles:v7',7,target_sha,'rol:administracion_perfiles:v7',c,'["organizacion_ref","unidad_ref"]','administrador_aplicacion',instante,NULL);
  END LOOP;
  FOR t IN SELECT value FROM jsonb_array_elements(destinos) LOOP
   doc:=t->'documento';
   INSERT INTO vec_autorizacion.asignacion_perfil(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento) VALUES(t->>'ref',doc->>'asignacion_id',4,doc->>'perfil_activo_ref',doc->>'principal_id','rol:administracion_perfiles:v7',t->>'sha',(doc->>'emitida_en')::timestamptz,doc);
   INSERT INTO vec_autorizacion.sello_efecto_admin_tx_v1 VALUES(t->>'ref',txid_current(),p->>'operacion_ref',aud.auditoria_ref);
   UPDATE vec_autorizacion.asignacion_perfil_actual SET asignacion_ref=t->>'ref',actualizada_en=clock_timestamp(),actualizada_por='mantenimiento_operador:'||session_user::text,acto_ref=p->>'operacion_ref' WHERE perfil_activo_ref=doc->>'perfil_activo_ref' AND asignacion_ref='asignacion:'||(doc->>'asignacion_id')||':v3';
   IF NOT FOUND THEN RAISE EXCEPTION 'AUT51: PARO clave=CAS_final actual=divergente esperado=puntero_v3_original' USING ERRCODE='40001'; END IF;
  END LOOP;
 END IF;
 PERFORM vec_autorizacion.exigir_operador_mantenimiento_plan_firma_admin_v1();IF clock_timestamp()>=(p->>'caduca_en')::timestamptz THEN RAISE EXCEPTION 'AUT51: PARO clave=vigencia_final actual=caducada esperado=plan_vigente' USING ERRCODE='42501';END IF;
 recibo:=jsonb_build_object('esquema','vec.admin.mantenimiento-fijo.v3','operacion_ref',p->>'operacion_ref','plan_sha256',sha,'rol_origen_ref','rol:administracion_perfiles:v6','rol_destino_ref','rol:administracion_perfiles:v7','rol_destino_sha256',target_sha,'asignaciones',(SELECT jsonb_agg(x.value-'documento') FROM jsonb_array_elements(destinos) x),'auditoria_ref',aud.auditoria_ref,'auditoria_secuencia',aud.secuencia,'auditoria_huella_sha256',aud.huella_sha256,'confirmado_en',aud.registrada_en);
 RETURN jsonb_build_object('recibo',recibo,'replay',replay);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.aplicar_mantenimiento_plan_firma_admin_v1(text,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.mantener_version_perfil_fijo_plan_firma_admin_v1(plan_canonico text,sha_aprobado text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' AS $f$
DECLARE respuesta jsonb;estado text:='permitido';codigo text;motivo text;aud record;sol text;evento text;corr text;solsha text;
BEGIN
 sol:='solicitud_mantenimiento:'||replace(gen_random_uuid()::text,'-','');evento:='evento_'||replace(gen_random_uuid()::text,'-','');corr:='correlacion_'||replace(gen_random_uuid()::text,'-','');solsha:=encode(pg_catalog.sha256(convert_to(jsonb_build_object('plan',plan_canonico,'sha_aprobado',sha_aprobado)::text,'UTF8')),'hex');
 BEGIN
  respuesta:=vec_autorizacion.aplicar_mantenimiento_plan_firma_admin_v1(plan_canonico,sha_aprobado);motivo:=CASE WHEN (respuesta->>'replay')::boolean THEN 'mantenimiento_replay' ELSE 'mantenimiento_registrado' END;
 SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_intento_mantenimiento_perfil_fijo_admin_v1(jsonb_build_object('tipo_registro','intento_mantenimiento_perfil_fijo_admin','evento_ref',evento,'operador_login',session_user::text,'solicitud_sha256',solsha,'accion','mantener_version_perfil_fijo_admin_v1','recurso_ref',sol,'resultado',estado,'motivo_ref',motivo,'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','mantenimiento_perfil_fijo_admin','correlacion_ref',corr));
  -- El append toma la barrera de cadena también en replay. La ventana se
  -- vuelve a cotejar bajo esa barrera y dentro del mismo subbloque del efecto.
  PERFORM vec_autorizacion.exigir_operador_mantenimiento_plan_firma_admin_v1();
  IF clock_timestamp()>=(plan_canonico::jsonb->>'caduca_en')::timestamptz OR EXISTS(SELECT 1 FROM jsonb_array_elements(respuesta#>'{recibo,asignaciones}') x JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=x.value->>'ref' WHERE clock_timestamp()<(a.documento->>'vigente_desde')::timestamptz OR clock_timestamp()>=(a.documento->>'vigente_hasta')::timestamptz) THEN
   RAISE EXCEPTION 'AUT51: PARO clave=vigencia_tras_append actual=caducada esperado=plan_vigente' USING ERRCODE='42501';
  END IF;
 EXCEPTION WHEN OTHERS THEN respuesta:=NULL;codigo:=SQLSTATE;estado:=CASE WHEN codigo IN('42501','22023','40001','23505','25000') THEN 'denegado' ELSE 'error' END;motivo:=CASE WHEN estado='denegado' THEN 'mantenimiento_denegado' ELSE 'mantenimiento_error' END;codigo:=CASE WHEN estado='denegado' THEN 'mantenimiento_rechazado' ELSE 'mantenimiento_no_disponible' END;
 END;
 IF estado<>'permitido' THEN
 SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_intento_mantenimiento_perfil_fijo_admin_v1(jsonb_build_object('tipo_registro','intento_mantenimiento_perfil_fijo_admin','evento_ref',evento,'operador_login',session_user::text,'solicitud_sha256',solsha,'accion','mantener_version_perfil_fijo_admin_v1','recurso_ref',sol,'resultado',estado,'motivo_ref',motivo,'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','mantenimiento_perfil_fijo_admin','correlacion_ref',corr));
 END IF;
 RETURN jsonb_build_object('estado',estado,'codigo',codigo,'recibo',respuesta->'recibo','replay',COALESCE((respuesta->>'replay')::boolean,false),'auditoria_intento',jsonb_build_object('auditoria_ref',aud.auditoria_ref,'secuencia',aud.secuencia,'huella_sha256',aud.huella_sha256,'correlacion_ref',aud.correlacion_ref,'registrada_en',aud.registrada_en));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.mantener_version_perfil_fijo_plan_firma_admin_v1(text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_mantenimiento_plan_firma_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion.mantener_version_perfil_fijo_plan_firma_admin_v1(text,text) TO vec_admin_mantenimiento_plan_firma_ejecutor;
RESET ROLE;

-- Metadatos de la puerta del lote antes de sustituirla (salvo el cuerpo).
CREATE TEMP TABLE aut51_puerta_lote_antes ON COMMIT DROP AS
 SELECT p.oid,p.proowner,p.prosecdef,p.provolatile,p.proconfig,p.proacl,p.proargtypes,p.prorettype
 FROM pg_proc p WHERE p.oid='vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)'::regprocedure;
-- Puerta del lote a la versión vigente (como AUT48): la versión es la de la
-- asignación actual y su concesión de lote debe ser exacta en rol y catálogo.
-- Sin esto, Rol7 dejaría sin acreditar el lote ordinario (AD190/AUT44).
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(
 version_ref text,p_asignacion_ref text,principal_ref text,perfil_ref text,
 accion text,modulo text,tipo text,finalidad text,campos jsonb,autenticacion jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE r record;ct record;asig record;meta record;ahora timestamptz;concesion_lote jsonb;
BEGIN
 concesion_lote:=vec_autorizacion.concesiones_lote_ordinario_admin_v1()->0;
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR (version_ref ~ '^rol:administracion_perfiles:v[1-9][0-9]{0,8}$') IS NOT TRUE
  OR accion IS DISTINCT FROM concesion_lote->>'accion' OR modulo IS DISTINCT FROM concesion_lote->>'modulo_id'
  OR tipo IS DISTINCT FROM concesion_lote->>'tipo_recurso' OR finalidad IS DISTINCT FROM concesion_lote#>>'{finalidades,0}'
  OR campos IS DISTINCT FROM '[]'::jsonb
  OR pg_catalog.jsonb_typeof(autenticacion)<>'object'
  OR vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(autenticacion) IS NOT TRUE
  THEN RETURN false; END IF;
 SELECT v.* INTO r FROM vec_autorizacion.version_rol v WHERE v.version_rol_ref=version_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT c.* INTO ct FROM vec_autorizacion.control_vigencia_version_rol_actual a
 JOIN vec_autorizacion.control_vigencia_version_rol c USING(version_rol_ref,revision)
 WHERE a.version_rol_ref=version_ref FOR SHARE OF a,c;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO asig FROM vec_autorizacion.asignacion_perfil_actual a
 JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
 WHERE a.perfil_activo_ref=perfil_ref AND a.asignacion_ref=p_asignacion_ref FOR SHARE OF a,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO meta FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 x WHERE x.version_rol_ref=version_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 ahora:=pg_catalog.clock_timestamp();
 RETURN r.rol_id='administracion_perfiles' AND r.version_rol_ref='rol:administracion_perfiles:v'||r.version::text AND r.documento->>'estado'='publicada'
  AND r.huella_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
  AND ct.estado='habilitada' AND ct.huella_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_control_rol_admin_v1(ct.documento),'UTF8')),'hex')
  AND meta.version_rol_huella_sha256=r.huella_sha256 AND meta.categoria_administrativa='aplicacion' AND meta.tipo_perfil='fijo_sistema'
  AND meta.fuente_ref=version_ref AND meta.fuente_version=r.version AND meta.fuente_huella_sha256=r.huella_sha256
  AND asig.version_rol_ref=version_ref AND asig.principal_id=principal_ref AND asig.perfil_activo_ref=perfil_ref
  AND asig.documento->>'estado'='activa'
  AND ahora>=(asig.documento->>'vigente_desde')::timestamptz
  AND ahora<(asig.documento->>'vigente_hasta')::timestamptz
  AND EXISTS(SELECT 1 FROM vec_autorizacion.catalogo_accion_nominal_v1 catalogo
    WHERE catalogo.version_rol_ref=version_ref AND catalogo.accion_ref='accion:'||accion
     AND catalogo.fuente_ref=version_ref AND catalogo.fuente_version=r.version
     AND catalogo.fuente_huella_sha256=r.huella_sha256
     AND catalogo.clase_control='administrador_aplicacion'
     AND catalogo.dimensiones_ambito='["organizacion_ref","unidad_ref"]'::jsonb
     AND ahora>=catalogo.vigente_desde AND (catalogo.vigente_hasta IS NULL OR ahora<catalogo.vigente_hasta)
     AND catalogo.concesion=concesion_lote)
  AND EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(r.documento->'concesiones') c WHERE c=concesion_lote);
END $f$;
RESET ROLE;
-- Poscondición: CREATE OR REPLACE conserva OID, propietario, SECURITY DEFINER,
-- volatilidad, configuración, ACL y firma de la puerta del lote.
DO $post$ BEGIN
 IF (SELECT count(*) FROM pg_proc p JOIN aut51_puerta_lote_antes a USING(oid)
     WHERE p.proowner=a.proowner AND p.prosecdef=a.prosecdef AND p.provolatile=a.provolatile
       AND p.proconfig IS NOT DISTINCT FROM a.proconfig AND p.proacl IS NOT DISTINCT FROM a.proacl
       AND p.proargtypes=a.proargtypes AND p.prorettype=a.prorettype AND p.prosecdef
       AND p.proconfig=ARRAY['search_path=pg_catalog'])<>1
 THEN RAISE EXCEPTION 'AUT51: PARO clave=puerta_lote_metadatos actual=divergente esperado=OID_propietario_ACL_configuracion_conservados' USING ERRCODE='55000';END IF;
END $post$;
COMMIT;
