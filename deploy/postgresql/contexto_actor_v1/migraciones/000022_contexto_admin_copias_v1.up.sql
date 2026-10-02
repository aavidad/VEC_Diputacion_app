\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:migracion:000022',0));
DO $preimagen$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.bloquear_contexto_admin_v1(text,text,text,text,numeric,numeric,numeric,numeric)') IS NULL
 OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])') IS NULL
 OR pg_catalog.to_regclass('vec_contexto_actor_v1.perfil_admin_copias_v1') IS NOT NULL
 OR pg_catalog.to_regrole('vec_identidad_sesiones_v1_admin_copias') IS NOT NULL
 THEN RAISE EXCEPTION 'CA22: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $preimagen$;
CREATE ROLE vec_identidad_sesiones_v1_admin_copias NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN
 EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO vec_identidad_sesiones_v1_admin_copias',current_database());
END $conexion$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;

-- Una asignación nominal fija; revocar añade una fila y nunca rehabilita una cuenta.
-- La provisión exige la instantánea central y la huella de aprobación por CLI.
CREATE TABLE vec_contexto_actor_v1.perfil_admin_copias_v1(
 cuenta_ref text NOT NULL, revision numeric(20,0) NOT NULL CHECK(revision IN(1,2)),
 persona_ref text NOT NULL, perfil_ref text NOT NULL, vinculo_ref text NOT NULL,
 cuenta_version numeric(20,0) NOT NULL, persona_version numeric(20,0) NOT NULL,
 perfil_version numeric(20,0) NOT NULL, vinculo_version numeric(20,0) NOT NULL,
 estado text NOT NULL CHECK(estado IN('activo','revocado')),
 audiencia text NOT NULL CHECK(audiencia ~ '^[a-z0-9][a-z0-9._:-]{3,255}$'),
 vigente_hasta timestamptz(6) NOT NULL CHECK(pg_catalog.isfinite(vigente_hasta)),
 acto_ref text NOT NULL UNIQUE CHECK(acto_ref ~ '^acto_admin:[0-9a-f]{32}$'),
 preimagen_sha256 text NOT NULL CHECK(preimagen_sha256 ~ '^[0-9a-f]{64}$'),
 aprobacion_sha256 text NOT NULL CHECK(aprobacion_sha256 ~ '^[0-9a-f]{64}$' AND aprobacion_sha256<>pg_catalog.repeat('0',64)),
 registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 PRIMARY KEY(cuenta_ref,revision),
 CHECK(vec_contexto_actor_v1.referencia_valida(cuenta_ref,'cta_') IS TRUE),
 CHECK(vec_contexto_actor_v1.referencia_valida(persona_ref,'per_') IS TRUE),
 CHECK(vec_contexto_actor_v1.referencia_valida(perfil_ref,'prf_') IS TRUE),
 CHECK(vec_contexto_actor_v1.referencia_valida(vinculo_ref,'vca_') IS TRUE),
 CHECK(cuenta_version>=1 AND persona_version>=1 AND perfil_version>=1 AND vinculo_version>=1),
 CHECK((revision=1 AND estado='activo') OR (revision=2 AND estado='revocado'))
);
CREATE UNIQUE INDEX perfil_admin_copias_unico_v1 ON vec_contexto_actor_v1.perfil_admin_copias_v1(perfil_ref) WHERE revision=1;
CREATE TRIGGER perfil_admin_copias_historia BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.perfil_admin_copias_v1 FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia();
CREATE TRIGGER perfil_admin_copias_no_truncar BEFORE TRUNCATE ON vec_contexto_actor_v1.perfil_admin_copias_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();
-- El avance de este puntero obliga a una transacción SERIALIZABLE con snapshot
-- anterior a la revocación a fallar con 40001 al bloquear la fila actual.
CREATE TABLE vec_contexto_actor_v1.perfil_admin_copias_actual_v1(
 cuenta_ref text PRIMARY KEY,
 revision numeric(20,0) NOT NULL CHECK(revision IN(1,2)),
 FOREIGN KEY(cuenta_ref,revision) REFERENCES vec_contexto_actor_v1.perfil_admin_copias_v1(cuenta_ref,revision)
);
CREATE TRIGGER perfil_admin_copias_actual_no_truncar BEFORE TRUNCATE ON vec_contexto_actor_v1.perfil_admin_copias_actual_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();
CREATE TABLE vec_contexto_actor_v1.preimagen_funciones_admin_copias_v1(nombre text PRIMARY KEY,definicion text NOT NULL,sha256 text NOT NULL,uso_identidad_previo boolean NOT NULL DEFAULT pg_catalog.has_schema_privilege('vec_identidad_sesiones_v1_propietario','vec_contexto_actor_v1','USAGE'),uso_autorizacion_previo boolean NOT NULL DEFAULT pg_catalog.has_schema_privilege('vec_autorizacion_propietario','vec_contexto_actor_v1','USAGE'));
CREATE TRIGGER preimagen_admin_copias_historia BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.preimagen_funciones_admin_copias_v1 FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia();
CREATE TRIGGER preimagen_admin_copias_no_truncar BEFORE TRUNCATE ON vec_contexto_actor_v1.preimagen_funciones_admin_copias_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();

DO $extraer$
DECLARE n text; firma text; def text; guard constant text:='PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1();';
BEGIN
 FOREACH n IN ARRAY ARRAY['resolver_y_registrar_contexto_actor_v2','reconciliar_contexto_actor_v2'] LOOP
  firma:=pg_catalog.format('vec_contexto_actor_v1.%s(text,text,text,text,text,text,timestamptz,text[])',n);
  SELECT pg_catalog.pg_get_functiondef(p.oid) INTO STRICT def FROM pg_catalog.pg_proc p
   WHERE p.oid=pg_catalog.to_regprocedure(firma) AND p.proowner='vec_contexto_actor_v1_propietario'::regrole AND p.prosecdef AND p.proconfig @> ARRAY['search_path=pg_catalog'];
  IF (pg_catalog.length(def)-pg_catalog.length(pg_catalog.replace(def,guard,'')))/pg_catalog.length(guard)<>1 THEN
   RAISE EXCEPTION 'CA22: marcador de autoridad divergente' USING ERRCODE='55000'; END IF;
  INSERT INTO vec_contexto_actor_v1.preimagen_funciones_admin_copias_v1(nombre,definicion,sha256) VALUES(n,def,pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(def,'UTF8')),'hex'));
  def:=pg_catalog.replace(def,'FUNCTION vec_contexto_actor_v1.'||n||'(','FUNCTION vec_contexto_actor_v1.'||n||'_propietaria_admin_v1(');
  def:=pg_catalog.replace(def,guard,'-- Autoridad comprobada por la fachada propietaria nominal.');
  EXECUTE def;
  EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION vec_contexto_actor_v1.%s_propietaria_admin_v1(text,text,text,text,text,text,timestamptz,text[]) FROM PUBLIC,vec_contexto_actor_v1_runtime,vec_identidad_sesiones_v1_admin_copias',n);
 END LOOP;
END $extraer$;

CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(p_operacion_ref text,p_registro_contexto_ref text,p_cuenta_ref text,p_perfil_ref text,p_metodo text,p_garantia text,p_solicitado_en timestamptz,p_proyecciones text[])
RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,huella_sha256 text,manifiesto_procedencia_canonico bytea,manifiesto_procedencia_huella_sha256 text,autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1();
 RETURN QUERY SELECT * FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2_propietaria_admin_v1(p_operacion_ref,p_registro_contexto_ref,p_cuenta_ref,p_perfil_ref,p_metodo,p_garantia,p_solicitado_en,p_proyecciones);
END $f$;
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_actor_v2(p_operacion_ref text,p_registro_contexto_ref text,p_cuenta_ref text,p_perfil_ref text,p_metodo text,p_garantia text,p_solicitado_en timestamptz,p_proyecciones text[])
RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,huella_sha256 text,manifiesto_procedencia_canonico bytea,manifiesto_procedencia_huella_sha256 text,autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1();
 RETURN QUERY SELECT * FROM vec_contexto_actor_v1.reconciliar_contexto_actor_v2_propietaria_admin_v1(p_operacion_ref,p_registro_contexto_ref,p_cuenta_ref,p_perfil_ref,p_metodo,p_garantia,p_solicitado_en,p_proyecciones);
END $f$;

CREATE FUNCTION vec_contexto_actor_v1.preimagen_perfil_admin_copias_v1(p_cuenta text,p_persona text,p_perfil text,p_vinculo text)
RETURNS TABLE(snapshot jsonb,sha256 text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE c record;pe record;pr record;v record;b record;doc jsonb;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'CA22: requiere SERIALIZABLE' USING ERRCODE='25000'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT x.* INTO STRICT c FROM vec_contexto_actor_v1.proyeccion_cuenta_actual a JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones x USING(cuenta_ref,version) WHERE a.cuenta_ref=p_cuenta FOR UPDATE OF a;
 SELECT x.* INTO STRICT pr FROM vec_contexto_actor_v1.perfil_actual a JOIN vec_contexto_actor_v1.perfil_versiones x USING(perfil_ref,version) WHERE a.perfil_ref=p_perfil FOR UPDATE OF a;
 SELECT x.* INTO STRICT v FROM vec_contexto_actor_v1.vinculo_contexto_actual a JOIN vec_contexto_actor_v1.vinculo_contexto_versiones x USING(vinculo_ref,version) WHERE a.vinculo_ref=p_vinculo FOR UPDATE OF a;
 SELECT x.* INTO STRICT pe FROM vec_contexto_actor_v1.persona_actual a JOIN vec_contexto_actor_v1.persona_versiones x USING(persona_ref,version) WHERE a.persona_ref=p_persona FOR UPDATE OF a;
 PERFORM vec_contexto_actor_v1.bloquear_contexto_admin_v1(p_cuenta,p_persona,p_perfil,p_vinculo,c.version,pe.version,pr.version,v.version);
 SELECT x.* INTO b FROM vec_contexto_actor_v1.perfil_admin_copias_actual_v1 a
 JOIN vec_contexto_actor_v1.perfil_admin_copias_v1 x USING(cuenta_ref,revision)
 WHERE a.cuenta_ref=p_cuenta FOR UPDATE OF a;
 doc:=pg_catalog.jsonb_build_object('esquema','vec.admin-copias.preimagen.v1','cuenta',to_jsonb(c),'persona',to_jsonb(pe),'perfil',to_jsonb(pr),'vinculo',to_jsonb(v),'asignacion',CASE WHEN b.cuenta_ref IS NULL THEN 'null'::jsonb ELSE to_jsonb(b) END);
 RETURN QUERY SELECT doc,pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(doc::text,'UTF8')),'hex');
END $f$;

CREATE FUNCTION vec_contexto_actor_v1.provisionar_perfil_admin_copias_v1(p_cuenta text,p_persona text,p_perfil text,p_vinculo text,p_audiencia text,p_hasta timestamptz,p_preimagen_sha256 text,p_aprobacion_sha256 text,p_acto text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE pre record;s jsonb;
BEGIN
 SELECT * INTO STRICT pre FROM vec_contexto_actor_v1.preimagen_perfil_admin_copias_v1(p_cuenta,p_persona,p_perfil,p_vinculo);
 IF pre.sha256 IS DISTINCT FROM p_preimagen_sha256 THEN RAISE EXCEPTION 'CA22: CAS divergente' USING ERRCODE='40001'; END IF;
 s:=pre.snapshot;
 IF s->'asignacion' IS DISTINCT FROM 'null'::jsonb OR p_hasta IS NULL OR NOT pg_catalog.isfinite(p_hasta) OR p_hasta<=pg_catalog.clock_timestamp()
 OR p_hasta>LEAST((s->'cuenta'->>'vigente_hasta')::timestamptz,(s->'persona'->>'vigente_hasta')::timestamptz,(s->'perfil'->>'vigente_hasta')::timestamptz,(s->'vinculo'->>'vigente_hasta')::timestamptz) THEN RAISE EXCEPTION 'CA22: provisión no vigente o no revivible' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_contexto_actor_v1.perfil_admin_copias_v1 VALUES(p_cuenta,1,p_persona,p_perfil,p_vinculo,(s->'cuenta'->>'version')::numeric,(s->'persona'->>'version')::numeric,(s->'perfil'->>'version')::numeric,(s->'vinculo'->>'version')::numeric,'activo',p_audiencia,p_hasta,p_acto,p_preimagen_sha256,p_aprobacion_sha256,pg_catalog.clock_timestamp());
 INSERT INTO vec_contexto_actor_v1.perfil_admin_copias_actual_v1 VALUES(p_cuenta,1);
 RETURN true;
END $f$;
CREATE FUNCTION vec_contexto_actor_v1.revocar_perfil_admin_copias_v1(p_cuenta text,p_persona text,p_perfil text,p_vinculo text,p_preimagen_sha256 text,p_aprobacion_sha256 text,p_acto text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE pre record;b record;
BEGIN
 SELECT * INTO STRICT pre FROM vec_contexto_actor_v1.preimagen_perfil_admin_copias_v1(p_cuenta,p_persona,p_perfil,p_vinculo);
 IF pre.sha256 IS DISTINCT FROM p_preimagen_sha256 THEN RAISE EXCEPTION 'CA22: CAS divergente' USING ERRCODE='40001'; END IF;
 SELECT x.* INTO STRICT b FROM vec_contexto_actor_v1.perfil_admin_copias_actual_v1 a
 JOIN vec_contexto_actor_v1.perfil_admin_copias_v1 x USING(cuenta_ref,revision)
 WHERE a.cuenta_ref=p_cuenta FOR UPDATE OF a;
 IF b.estado<>'activo' THEN RAISE EXCEPTION 'CA22: no revivible' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_contexto_actor_v1.perfil_admin_copias_v1 VALUES(b.cuenta_ref,2,b.persona_ref,b.perfil_ref,b.vinculo_ref,b.cuenta_version,b.persona_version,b.perfil_version,b.vinculo_version,'revocado',b.audiencia,b.vigente_hasta,p_acto,p_preimagen_sha256,p_aprobacion_sha256,pg_catalog.clock_timestamp());
 UPDATE vec_contexto_actor_v1.perfil_admin_copias_actual_v1 SET revision=2 WHERE cuenta_ref=b.cuenta_ref AND revision=1;
 IF NOT FOUND THEN RAISE EXCEPTION 'CA22: revisión actual divergente' USING ERRCODE='40001'; END IF;
 RETURN true;
END $f$;

CREATE FUNCTION vec_contexto_actor_v1.consultar_perfil_admin_copias_v1(p_cuenta text)
RETURNS TABLE(persona_ref text,perfil_ref text,vinculo_ref text,cuenta_version numeric,persona_version numeric,perfil_version numeric,vinculo_version numeric,audiencia text,vigente_hasta timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE b record;ahora timestamptz;
BEGIN
 IF current_setting('transaction_read_only')<>'off' OR current_setting('transaction_isolation')='repeatable read' THEN RETURN; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT x.* INTO b FROM vec_contexto_actor_v1.perfil_admin_copias_actual_v1 a
 JOIN vec_contexto_actor_v1.perfil_admin_copias_v1 x USING(cuenta_ref,revision)
 WHERE a.cuenta_ref=p_cuenta FOR SHARE OF a;
 IF NOT FOUND OR b.estado<>'activo' THEN RETURN; END IF;
 PERFORM vec_contexto_actor_v1.bloquear_contexto_admin_v1(p_cuenta,b.persona_ref,b.perfil_ref,b.vinculo_ref,b.cuenta_version,b.persona_version,b.perfil_version,b.vinculo_version);
 ahora:=pg_catalog.clock_timestamp();
 IF ahora>=b.vigente_hasta THEN RETURN; END IF;
 RETURN QUERY SELECT b.persona_ref,b.perfil_ref,b.vinculo_ref,b.cuenta_version,b.persona_version,b.perfil_version,b.vinculo_version,b.audiencia,b.vigente_hasta;
EXCEPTION WHEN no_data_found THEN RETURN;
END $f$;

CREATE FUNCTION vec_contexto_actor_v1.consultar_perfil_admin_copias_por_actor_v1(p_persona text,p_perfil text)
RETURNS TABLE(cuenta_ref text,persona_ref text,perfil_ref text,vinculo_ref text,cuenta_version numeric,persona_version numeric,perfil_version numeric,vinculo_version numeric,audiencia text,vigente_hasta timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE cuenta text;
BEGIN
 SELECT x.cuenta_ref INTO STRICT cuenta FROM vec_contexto_actor_v1.perfil_admin_copias_v1 x WHERE x.persona_ref=p_persona AND x.perfil_ref=p_perfil AND x.revision=1;
 RETURN QUERY SELECT cuenta,b.* FROM vec_contexto_actor_v1.consultar_perfil_admin_copias_v1(cuenta) b WHERE b.persona_ref=p_persona AND b.perfil_ref=p_perfil;
EXCEPTION WHEN no_data_found OR too_many_rows THEN RETURN;
END $f$;

CREATE FUNCTION vec_contexto_actor_v1.exigir_runtime_admin_copias_v1()
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE l record;g record;fs oid[];ns oid[];base oid;
BEGIN
 SELECT * INTO l FROM pg_catalog.pg_roles WHERE rolname=session_user;
 SELECT * INTO g FROM pg_catalog.pg_roles WHERE rolname='vec_identidad_sesiones_v1_admin_copias';
 SELECT oid INTO base FROM pg_catalog.pg_database WHERE datname=current_database();
 fs:=ARRAY[pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_runtime_admin_copias_v1()'),pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_copias_v1(text,text,text,timestamptz)'),pg_catalog.to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_admin_copias_v1(text,text,text,timestamptz)'),pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.resolver_cuenta_admin_copias_v1(text,text,text,text,text,timestamptz,timestamptz)'),pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.vincular_sesion_admin_copias_v1(text,text,text,text,text,timestamptz,timestamptz,text,text)')];
 ns:=ARRAY[pg_catalog.to_regnamespace('vec_contexto_actor_v1'),pg_catalog.to_regnamespace('vec_identidad_sesiones_v1')];
 IF l.oid IS NULL OR g.oid IS NULL OR array_position(fs,NULL) IS NOT NULL OR array_position(ns,NULL) IS NOT NULL
 OR NOT l.rolcanlogin OR NOT l.rolinherit OR l.rolsuper OR l.rolcreaterole OR l.rolcreatedb OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
 OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreaterole OR g.rolcreatedb OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
 OR current_setting('role')<>'none'
 OR (SELECT count(*) FROM pg_catalog.pg_auth_members WHERE member=l.oid)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=l.oid AND roleid=g.oid AND NOT admin_option AND inherit_option AND NOT set_option)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.oid<>g.oid AND r.oid<>l.oid AND pg_catalog.pg_has_role(l.oid,r.oid,'MEMBER'))
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=g.oid)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting WHERE setrole IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=l.oid)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_default_acl d LEFT JOIN LATERAL pg_catalog.aclexplode(COALESCE(d.defaclacl,'{}'::aclitem[])) a ON true WHERE d.defaclrole IN(l.oid,g.oid) OR a.grantee IN(l.oid,g.oid) OR a.grantor IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_policy p WHERE l.oid=ANY(p.polroles) OR g.oid=ANY(p.polroles))
 OR NOT COALESCE((SELECT count(*)=1 AND bool_and(a.privilege_type='CONNECT' AND NOT a.is_grantable) FROM pg_catalog.pg_database d CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(d.datacl,pg_catalog.acldefault('d',d.datdba))) a WHERE d.oid=base AND a.grantee=g.oid),false)
 OR NOT COALESCE((SELECT count(*)=2 AND bool_and(a.privilege_type='USAGE' AND NOT a.is_grantable) FROM pg_catalog.pg_namespace n CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(n.nspacl,pg_catalog.acldefault('n',n.nspowner))) a WHERE n.oid=ANY(ns) AND a.grantee=g.oid),false)
 OR NOT COALESCE((SELECT count(*)=5 AND bool_and(a.privilege_type='EXECUTE' AND NOT a.is_grantable) FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=ANY(fs) AND a.grantee=g.oid),false)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(n.nspacl,pg_catalog.acldefault('n',n.nspowner))) a WHERE n.oid=ANY(ns) AND a.grantee=0)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.pronamespace=ANY(ns) AND a.grantee=0 AND p.prosecdef)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_class c WHERE c.relnamespace=ANY(ns) AND c.relkind IN('r','p','v','m','S') AND (pg_catalog.has_table_privilege(l.oid,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') OR pg_catalog.has_any_column_privilege(l.oid,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))

 OR NOT COALESCE((SELECT count(*)=8 AND bool_and(deptype='a' AND objsubid=0 AND ((classid='pg_catalog.pg_database'::regclass AND objid=base) OR (classid='pg_catalog.pg_namespace'::regclass AND objid=ANY(ns)) OR (classid='pg_catalog.pg_proc'::regclass AND objid=ANY(fs)))) FROM pg_catalog.pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=g.oid),false)
 THEN RAISE EXCEPTION 'CA22: LOGIN ADMIN no acreditado' USING ERRCODE='42501'; END IF;
 RETURN session_user;
END $f$;

CREATE FUNCTION vec_contexto_actor_v1.acreditar_runtime_admin_copias_v1()
RETURNS TABLE(identidad_login text,acreditada boolean) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 RETURN QUERY SELECT vec_contexto_actor_v1.exigir_runtime_admin_copias_v1(),true;
END $f$;

CREATE FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_copias_v1(p_operacion text,p_registro text,p_cuenta text,p_solicitado timestamptz)
RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,huella_sha256 text,manifiesto_procedencia_canonico bytea,manifiesto_procedencia_huella_sha256 text,autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE b record;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_admin_copias_v1();
 IF vec_contexto_actor_v1.referencia_operacion_valida(p_operacion,'oca_') IS NOT TRUE
 THEN RAISE EXCEPTION 'CA22: operación inválida' USING ERRCODE='22023'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:operacion:v2:'||p_operacion,0));
 SELECT * INTO STRICT b FROM vec_contexto_actor_v1.consultar_perfil_admin_copias_v1(p_cuenta);
 RETURN QUERY SELECT * FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2_propietaria_admin_v1(p_operacion,p_registro,p_cuenta,b.perfil_ref,'certificado','alto',p_solicitado,'{}'::text[]);
END $f$;
CREATE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_admin_copias_v1(p_operacion text,p_registro text,p_cuenta text,p_solicitado timestamptz)
RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,huella_sha256 text,manifiesto_procedencia_canonico bytea,manifiesto_procedencia_huella_sha256 text,autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE b record;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_admin_copias_v1();
 IF vec_contexto_actor_v1.referencia_operacion_valida(p_operacion,'oca_') IS NOT TRUE
 THEN RAISE EXCEPTION 'CA22: operación inválida' USING ERRCODE='22023'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:operacion:v2:'||p_operacion,0));
 SELECT * INTO STRICT b FROM vec_contexto_actor_v1.consultar_perfil_admin_copias_v1(p_cuenta);
 RETURN QUERY SELECT * FROM vec_contexto_actor_v1.reconciliar_contexto_actor_v2_propietaria_admin_v1(p_operacion,p_registro,p_cuenta,b.perfil_ref,'certificado','alto',p_solicitado,'{}'::text[]);
END $f$;

DO $acl$
DECLARE n text;f record;
BEGIN
 FOREACH n IN ARRAY ARRAY['perfil_admin_copias_v1','perfil_admin_copias_actual_v1','preimagen_funciones_admin_copias_v1'] LOOP
 EXECUTE pg_catalog.format('ALTER TABLE vec_contexto_actor_v1.%I ENABLE ROW LEVEL SECURITY',n);
 EXECUTE pg_catalog.format('ALTER TABLE vec_contexto_actor_v1.%I FORCE ROW LEVEL SECURITY',n);
 EXECUTE pg_catalog.format('CREATE POLICY propietario_exacto ON vec_contexto_actor_v1.%I FOR ALL TO vec_contexto_actor_v1_propietario USING(current_user=%L) WITH CHECK(current_user=%L)',n,'vec_contexto_actor_v1_propietario','vec_contexto_actor_v1_propietario');
 EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_contexto_actor_v1.%I FROM PUBLIC,vec_identidad_sesiones_v1_admin_copias',n);
 EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_contexto_actor_v1.%I FROM PUBLIC,vec_identidad_sesiones_v1_admin_copias',n);
 END LOOP;
 FOR f IN SELECT p.oid::regprocedure AS firma FROM pg_catalog.pg_proc p WHERE p.pronamespace='vec_contexto_actor_v1'::regnamespace AND (p.proname LIKE '%admin_copias_v1' OR p.proname LIKE '%_propietaria_admin_v1') LOOP EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,vec_contexto_actor_v1_runtime,vec_identidad_sesiones_v1_admin_copias',f.firma); END LOOP;
END $acl$;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_identidad_sesiones_v1_admin_copias,vec_identidad_sesiones_v1_propietario,vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.preimagen_perfil_admin_copias_v1(text,text,text,text),vec_contexto_actor_v1.provisionar_perfil_admin_copias_v1(text,text,text,text,text,timestamptz,text,text,text),vec_contexto_actor_v1.revocar_perfil_admin_copias_v1(text,text,text,text,text,text,text),vec_contexto_actor_v1.consultar_perfil_admin_copias_v1(text),vec_contexto_actor_v1.consultar_perfil_admin_copias_por_actor_v1(text,text) TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.exigir_runtime_admin_copias_v1(),vec_contexto_actor_v1.consultar_perfil_admin_copias_v1(text) TO vec_identidad_sesiones_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_runtime_admin_copias_v1(),vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_copias_v1(text,text,text,timestamptz),vec_contexto_actor_v1.reconciliar_contexto_admin_copias_v1(text,text,text,timestamptz) TO vec_identidad_sesiones_v1_admin_copias;
COMMIT;
