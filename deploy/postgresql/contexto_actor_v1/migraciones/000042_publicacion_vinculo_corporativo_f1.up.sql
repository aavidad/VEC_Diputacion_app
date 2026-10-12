\set ON_ERROR_STOP on
-- CA42: alta inicial corporativa F1 desde una fila exacta aprobada por DBA.
-- Mantiene CA9/CA40 y rechaza otra identidad/perfil en el puntero actual.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_acto_vinculo_corporativo_f1(jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_vinculo_corporativo_f1(jsonb)') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(bytea)') IS NULL
 OR to_regclass('vec_contexto_actor_v1.fuente_aprobada_vinculo_f1') IS NOT NULL
 OR to_regrole('vec_contexto_actor_corporativo_rrhh_publicador') IS NOT NULL
 THEN RAISE EXCEPTION 'CA42: PARO clave=preimagen actual=incompatible esperado=CA40_AD258' USING ERRCODE='55000'; END IF;
END $pre$;
SELECT pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:migracion:acreditacion_uso:v2',0));
SELECT pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:organizacion-corporativa-rrhh:v1',0));
SELECT pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:vinculo-corporativo-rrhh:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:proyeccion-corporativa-automatica:v1',0));
LOCK TABLE vec_contexto_actor_v1.procedencias,
 vec_contexto_actor_v1.proyeccion_cuenta_versiones,
 vec_contexto_actor_v1.persona_versiones,
 vec_contexto_actor_v1.perfil_versiones,
 vec_contexto_actor_v1.vinculo_contexto_versiones IN SHARE MODE;
LOCK TABLE vec_contexto_actor_v1.organizacion_versiones,
 vec_contexto_actor_v1.organizacion_actual IN SHARE MODE;
LOCK TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones,
 vec_contexto_actor_v1.vinculo_corporativo_actual IN SHARE MODE;
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
CREATE ROLE vec_contexto_actor_corporativo_rrhh_publicador NOLOGIN NOINHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_contexto_actor_corporativo_rrhh_publicador',current_database()); END $conexion$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;

CREATE TABLE vec_contexto_actor_v1.fuente_aprobada_vinculo_f1 (
 operacion_ref text PRIMARY KEY CHECK(operacion_ref ~ '^pubv_[A-Za-z0-9_-]{22,123}$'),
 login_nombre name NOT NULL,
 cuenta_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_valida(cuenta_ref,'cta_')),
 persona_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_valida(persona_ref,'per_')),
 perfil_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_valida(perfil_ref,'prf_')),
 vinculo_contexto_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_valida(vinculo_contexto_ref,'vca_')),
 vinculo_corporativo_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_valida(vinculo_corporativo_ref,'vcr_')),
 organizacion_ref text NOT NULL CHECK(vec_contexto_actor_v1.organizacion_ref_valida(organizacion_ref)),
 procedencia_ref text NOT NULL,
 procedencia_version numeric(20,0) NOT NULL,
 procedencia_huella_sha256 text NOT NULL,
 procedencia_autoridad text NOT NULL CHECK(procedencia_autoridad='autoridad_maestra_acreditada'),
 fuente_huella_sha256 text NOT NULL CHECK(fuente_huella_sha256 ~ '^[0-9a-f]{64}$'),
 aprobacion_ref text NOT NULL CHECK(aprobacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
 aprobacion_huella_sha256 text NOT NULL CHECK(aprobacion_huella_sha256 ~ '^[0-9a-f]{64}$'),
 vigente_desde timestamptz(6) NOT NULL CHECK(vec_contexto_actor_v1.instante_valido(vigente_desde)),
 vigente_hasta timestamptz(6) NOT NULL CHECK(vec_contexto_actor_v1.instante_valido(vigente_hasta)),
 CHECK(vigente_hasta>vigente_desde),
 CHECK(vec_contexto_actor_v1.procedencia_valida(procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)),
 FOREIGN KEY(procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)
  REFERENCES vec_contexto_actor_v1.procedencias(procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)
);
CREATE TABLE vec_contexto_actor_v1.prueba_publicacion_vinculo_f1 (
 operacion_ref text PRIMARY KEY REFERENCES vec_contexto_actor_v1.fuente_aprobada_vinculo_f1(operacion_ref),
 plan_sha256 text NOT NULL CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 vinculo_corporativo_ref text NOT NULL,
 version numeric(20,0) NOT NULL CHECK(version=1),
 auditoria_ref text NOT NULL UNIQUE,
 auditoria_huella_sha256 text NOT NULL CHECK(auditoria_huella_sha256 ~ '^[0-9a-f]{64}$'),
 evento_ref text NOT NULL UNIQUE CHECK(evento_ref ~ '^evento_[0-9a-f]{32}$'),
 registrada_en timestamptz(6) NOT NULL,
 FOREIGN KEY(vinculo_corporativo_ref,version) REFERENCES vec_contexto_actor_v1.vinculo_corporativo_versiones(vinculo_corporativo_ref,version)
);
CREATE TABLE vec_contexto_actor_v1.buzon_publicacion_vinculo_f1 (
 evento_ref text PRIMARY KEY REFERENCES vec_contexto_actor_v1.prueba_publicacion_vinculo_f1(evento_ref) DEFERRABLE INITIALLY DEFERRED,
 operacion_ref text NOT NULL UNIQUE REFERENCES vec_contexto_actor_v1.prueba_publicacion_vinculo_f1(operacion_ref) DEFERRABLE INITIALLY DEFERRED,
 auditoria_ref text NOT NULL,
 plan_sha256 text NOT NULL CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 creada_en timestamptz(6) NOT NULL
);
DO $tablas$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['fuente_aprobada_vinculo_f1','prueba_publicacion_vinculo_f1','buzon_publicacion_vinculo_f1'] LOOP
  EXECUTE format('ALTER TABLE vec_contexto_actor_v1.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_contexto_actor_v1.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_contexto_actor_v1.%I TO vec_contexto_actor_v1_propietario USING(current_user=%L) WITH CHECK(current_user=%L)',t,'vec_contexto_actor_v1_propietario','vec_contexto_actor_v1_propietario');
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.%I FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia()',t);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_contexto_actor_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado()',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_contexto_actor_v1.%I FROM PUBLIC,vec_contexto_actor_v1_runtime',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_contexto_actor_v1.%I FROM PUBLIC,vec_contexto_actor_v1_runtime',t);
 END LOOP;
END $tablas$;

CREATE FUNCTION vec_contexto_actor_v1.preimagen_publicacion_vinculo_f1(
 p_operacion_ref text,p_cuenta_ref text,p_persona_ref text,p_perfil_ref text,
 p_vinculo_contexto_ref text,p_vinculo_corporativo_ref text,p_organizacion_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE s vec_contexto_actor_v1.fuente_aprobada_vinculo_f1; l record; g record;
 c record; x record; p record; v record; o record; vc record; pr record;
 ahora timestamptz(6); plan jsonb; sha text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('role')<>'none' OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'CA42: PARO clave=transaccion actual=incompatible esperado=serializable_rw_UTC' USING ERRCODE='25000'; END IF;
 SELECT * INTO l FROM pg_roles WHERE rolname=session_user;
 SELECT * INTO g FROM pg_roles WHERE rolname='vec_contexto_actor_corporativo_rrhh_publicador';
 IF l.rolname IS NULL OR NOT l.rolcanlogin OR NOT l.rolinherit OR l.rolsuper OR l.rolcreatedb OR l.rolcreaterole
 OR l.rolreplication OR l.rolbypassrls OR l.rolvaliduntil IS NOT NULL
 OR g.rolname IS NULL OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole
 OR g.rolreplication OR g.rolbypassrls OR g.rolvaliduntil IS NOT NULL
 OR NOT pg_has_role(session_user,g.rolname,'member')
 OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=l.oid)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=l.oid AND m.roleid=g.oid
  AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option
  AND m.grantor=(SELECT datdba FROM pg_database WHERE datname=current_database()))
 OR has_database_privilege(l.oid,current_database(),'CREATE,TEMPORARY')
 OR has_schema_privilege(l.oid,'vec_contexto_actor_v1','CREATE')
 OR EXISTS(SELECT 1 FROM pg_db_role_setting WHERE setrole IN (l.oid,g.oid))
 OR pg_has_role(session_user,'vec_contexto_actor_v1_runtime','member')
 OR has_schema_privilege(l.oid,'vec_autorizacion_atestada_v3','USAGE')
 OR EXISTS(SELECT 1 FROM pg_class t WHERE t.relnamespace='vec_contexto_actor_v1'::regnamespace
  AND t.relkind IN ('r','p') AND has_table_privilege(l.oid,t.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'))
 OR EXISTS(SELECT 1 FROM pg_proc f WHERE f.pronamespace='vec_contexto_actor_v1'::regnamespace
  AND f.oid NOT IN (
   'vec_contexto_actor_v1.comprobar_publicacion_vinculo_corporativo_f1(text,text,text,text,text,text,text)'::regprocedure,
   'vec_contexto_actor_v1.publicar_vinculo_corporativo_desde_fuente_v1(text,text,text,text,text,text,text,text)'::regprocedure)
  AND has_function_privilege(l.oid,f.oid,'EXECUTE'))
 THEN RAISE EXCEPTION 'CA42: PARO clave=operador actual=incompatible esperado=LOGIN_publicador_exclusivo' USING ERRCODE='42501'; END IF;
 IF (p_operacion_ref ~ '^pubv_[A-Za-z0-9_-]{22,123}$') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(p_cuenta_ref,'cta_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(p_persona_ref,'per_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref,'prf_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(p_vinculo_contexto_ref,'vca_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(p_vinculo_corporativo_ref,'vcr_') IS NOT TRUE
 OR vec_contexto_actor_v1.organizacion_ref_valida(p_organizacion_ref) IS NOT TRUE
 THEN RAISE EXCEPTION 'CA42: PARO clave=referencias actual=incompatible esperado=opacas_validas' USING ERRCODE='22023'; END IF;
 -- A→B→C→D y E antes de punteros, igual que C2.3 y los triggers V2.
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:migracion:acreditacion_uso:v2',0));
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:organizacion-corporativa-rrhh:v1',0));
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:vinculo-corporativo-rrhh:v1',0));
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:proyeccion-corporativa-automatica:v1',0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:publicacion-f1:operacion:'||p_operacion_ref,0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:publicacion-f1:organizacion:'||p_organizacion_ref,0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:publicacion-f1:vinculo:'||p_vinculo_corporativo_ref,0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 SELECT * INTO s FROM vec_contexto_actor_v1.fuente_aprobada_vinculo_f1
 WHERE operacion_ref=p_operacion_ref FOR SHARE;
 IF NOT FOUND OR s.login_nombre IS DISTINCT FROM session_user::name
 OR s.cuenta_ref IS DISTINCT FROM p_cuenta_ref OR s.persona_ref IS DISTINCT FROM p_persona_ref
 OR s.perfil_ref IS DISTINCT FROM p_perfil_ref OR s.vinculo_contexto_ref IS DISTINCT FROM p_vinculo_contexto_ref
 OR s.vinculo_corporativo_ref IS DISTINCT FROM p_vinculo_corporativo_ref
 OR s.organizacion_ref IS DISTINCT FROM p_organizacion_ref
 THEN RAISE EXCEPTION 'CA42: PARO clave=fuente actual=no_acreditada esperado=exacta' USING ERRCODE='42501'; END IF;
 SELECT ov.*,oa.version AS actual_version INTO o
 FROM vec_contexto_actor_v1.organizacion_actual oa
 JOIN vec_contexto_actor_v1.organizacion_versiones ov ON ov.organizacion_ref=oa.organizacion_ref AND ov.version=oa.version
 WHERE oa.organizacion_ref=s.organizacion_ref FOR SHARE OF oa,ov;
 SELECT cv.*,ca.version AS actual_version INTO c FROM vec_contexto_actor_v1.proyeccion_cuenta_actual ca
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones cv ON cv.cuenta_ref=ca.cuenta_ref AND cv.version=ca.version
 WHERE ca.cuenta_ref=s.cuenta_ref FOR SHARE OF ca,cv;
 SELECT xv.*,xa.version AS actual_version INTO x FROM vec_contexto_actor_v1.persona_actual xa
 JOIN vec_contexto_actor_v1.persona_versiones xv ON xv.persona_ref=xa.persona_ref AND xv.version=xa.version
 WHERE xa.persona_ref=s.persona_ref FOR SHARE OF xa,xv;
 SELECT pv.*,pa.version AS actual_version INTO p FROM vec_contexto_actor_v1.perfil_actual pa
 JOIN vec_contexto_actor_v1.perfil_versiones pv ON pv.perfil_ref=pa.perfil_ref AND pv.version=pa.version
 WHERE pa.perfil_ref=s.perfil_ref FOR SHARE OF pa,pv;
 SELECT vv.*,va.version AS actual_version INTO v FROM vec_contexto_actor_v1.vinculo_contexto_actual va
 JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv ON vv.vinculo_ref=va.vinculo_ref AND vv.version=va.version
 WHERE va.vinculo_ref=s.vinculo_contexto_ref FOR SHARE OF va,vv;
 SELECT vv.*,va.version AS actual_version INTO vc FROM vec_contexto_actor_v1.vinculo_corporativo_actual va
 JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones vv ON vv.vinculo_corporativo_ref=va.vinculo_corporativo_ref AND vv.version=va.version
 WHERE va.cuenta_ref=s.cuenta_ref AND va.superficie='interna_corporativa' AND va.uso='consulta_rrhh' FOR SHARE OF va,vv;
 SELECT * INTO pr FROM vec_contexto_actor_v1.procedencias
 WHERE procedencia_ref=s.procedencia_ref AND procedencia_version=s.procedencia_version FOR SHARE;
 ahora:=clock_timestamp();
 IF o.organizacion_ref IS NULL OR c.cuenta_ref IS NULL OR x.persona_ref IS NULL
 OR p.perfil_ref IS NULL OR v.vinculo_ref IS NULL OR pr.procedencia_ref IS NULL
 OR c.estado<>'activo' OR x.estado<>'activo' OR p.estado<>'activo' OR v.estado<>'activo' OR o.estado<>'activo'
 OR c.procedencia_autoridad<>'autoridad_maestra_acreditada'
 OR x.procedencia_autoridad<>'autoridad_maestra_acreditada'
 OR p.procedencia_autoridad<>'autoridad_maestra_acreditada'
 OR v.procedencia_autoridad<>'autoridad_maestra_acreditada'
 OR o.procedencia_autoridad<>'autoridad_maestra_acreditada'
 OR pr.procedencia_autoridad<>'autoridad_maestra_acreditada'
 OR p.persona_ref IS DISTINCT FROM s.persona_ref
 OR v.cuenta_ref IS DISTINCT FROM s.cuenta_ref OR v.persona_ref IS DISTINCT FROM s.persona_ref
 OR v.perfil_ref IS DISTINCT FROM s.perfil_ref
 OR (pr.procedencia_ref,pr.procedencia_version,pr.procedencia_huella_sha256,pr.procedencia_autoridad)
 IS DISTINCT FROM (s.procedencia_ref,s.procedencia_version,s.procedencia_huella_sha256,s.procedencia_autoridad)
 OR ahora<greatest(s.vigente_desde,c.vigente_desde,x.vigente_desde,p.vigente_desde,v.vigente_desde,o.vigente_desde)
 OR ahora>=least(s.vigente_hasta,c.vigente_hasta,x.vigente_hasta,p.vigente_hasta,v.vigente_hasta,o.vigente_hasta)
 OR s.vigente_desde<greatest(c.vigente_desde,x.vigente_desde,p.vigente_desde,v.vigente_desde,o.vigente_desde)
 OR s.vigente_hasta>least(c.vigente_hasta,x.vigente_hasta,p.vigente_hasta,v.vigente_hasta,o.vigente_hasta)
 THEN RAISE EXCEPTION 'CA42: PARO clave=fuentes actual=incompatible esperado=activas_vigentes_exactas' USING ERRCODE='42501'; END IF;
 IF vc.vinculo_corporativo_ref IS NOT NULL AND
  (vc.vinculo_corporativo_ref IS DISTINCT FROM s.vinculo_corporativo_ref
   OR vc.perfil_ref IS DISTINCT FROM s.perfil_ref OR vc.persona_ref IS DISTINCT FROM s.persona_ref
   OR vc.organizacion_ref IS DISTINCT FROM s.organizacion_ref OR vc.estado<>'activo'
   OR vc.vinculo_contexto_ref IS DISTINCT FROM s.vinculo_contexto_ref
   OR vc.procedencia_ref IS DISTINCT FROM s.procedencia_ref
   OR vc.procedencia_version IS DISTINCT FROM s.procedencia_version
   OR vc.procedencia_huella_sha256 IS DISTINCT FROM s.procedencia_huella_sha256
   OR vc.procedencia_autoridad IS DISTINCT FROM s.procedencia_autoridad
   OR ahora<vc.vigente_desde OR ahora>=vc.vigente_hasta
   OR vc.cuenta_version IS DISTINCT FROM c.version OR vc.persona_version IS DISTINCT FROM x.version
   OR vc.perfil_version IS DISTINCT FROM p.version OR vc.vinculo_contexto_version IS DISTINCT FROM v.version
   OR vc.organizacion_version IS DISTINCT FROM o.version)
 THEN RAISE EXCEPTION 'CA42: PARO clave=puntero actual=otro_perfil_o_vinculo esperado=ausente_o_replay_exacto' USING ERRCODE='40001'; END IF;
 IF vc.vinculo_corporativo_ref IS NULL AND EXISTS(
  SELECT 1 FROM vec_contexto_actor_v1.vinculo_corporativo_versiones
  WHERE vinculo_corporativo_ref=s.vinculo_corporativo_ref)
 THEN RAISE EXCEPTION 'CA42: PARO clave=historia actual=existente esperado=alta_inicial' USING ERRCODE='40001'; END IF;
 plan:=jsonb_build_object('esquema','vec.ca.publicacion-vinculo-f1.v1','operacion_ref',s.operacion_ref,
  'login',s.login_nombre,'cuenta_ref',s.cuenta_ref,'cuenta_version',c.version,
  'persona_ref',s.persona_ref,'persona_version',x.version,'perfil_ref',s.perfil_ref,'perfil_version',p.version,
  'vinculo_contexto_ref',s.vinculo_contexto_ref,'vinculo_contexto_version',v.version,
  'vinculo_corporativo_ref',s.vinculo_corporativo_ref,'organizacion_ref',s.organizacion_ref,'organizacion_version',o.version,
  'organizacion_procedencia_ref',o.procedencia_ref,'organizacion_procedencia_version',o.procedencia_version,
  'organizacion_procedencia_huella_sha256',o.procedencia_huella_sha256,
  'procedencia_ref',s.procedencia_ref,'procedencia_version',s.procedencia_version,
  'procedencia_huella_sha256',s.procedencia_huella_sha256,'fuente_huella_sha256',s.fuente_huella_sha256,
  'aprobacion_ref',s.aprobacion_ref,'aprobacion_huella_sha256',s.aprobacion_huella_sha256,
  'vigente_desde',s.vigente_desde,'vigente_hasta',s.vigente_hasta,'version',1);
 sha:=encode(sha256(convert_to(plan::text,'UTF8')),'hex');
 IF vc.vinculo_corporativo_ref IS NOT NULL AND NOT EXISTS(
  SELECT 1 FROM vec_contexto_actor_v1.prueba_publicacion_vinculo_f1 prf
  WHERE prf.operacion_ref=s.operacion_ref AND prf.plan_sha256=sha
   AND prf.vinculo_corporativo_ref=s.vinculo_corporativo_ref AND prf.version=1)
 THEN RAISE EXCEPTION 'CA42: PARO clave=replay actual=divergente esperado=prueba_original' USING ERRCODE='40001'; END IF;
 RETURN plan||jsonb_build_object('plan_sha256',sha,'estado',CASE WHEN vc.vinculo_corporativo_ref IS NULL THEN 'publicable' ELSE 'replay' END);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.preimagen_publicacion_vinculo_f1(text,text,text,text,text,text,text)
 FROM PUBLIC,vec_contexto_actor_v1_runtime;

CREATE FUNCTION vec_contexto_actor_v1.auditar_denegacion_publicacion_vinculo_f1(
 p_modo text,p_operacion_ref text,p_cuenta_ref text,p_persona_ref text,p_perfil_ref text,
 p_vinculo_contexto_ref text,p_vinculo_corporativo_ref text,p_organizacion_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE solicitud text;sha text;evento text;corr text;a record;
BEGIN
 IF p_modo NOT IN ('comprobar','publicar')
 OR octet_length(concat_ws('',p_operacion_ref,p_cuenta_ref,p_persona_ref,p_perfil_ref,
  p_vinculo_contexto_ref,p_vinculo_corporativo_ref,p_organizacion_ref))>2048
 THEN RAISE EXCEPTION 'CA42: PARO clave=intento actual=incompatible esperado=acotado' USING ERRCODE='22023'; END IF;
 solicitud:=jsonb_build_array(p_modo,p_operacion_ref,p_cuenta_ref,p_persona_ref,p_perfil_ref,
  p_vinculo_contexto_ref,p_vinculo_corporativo_ref,p_organizacion_ref)::text;
 sha:=encode(sha256(convert_to(solicitud,'UTF8')),'hex');
 evento:='evento_'||substr(encode(sha256(convert_to('vec.ca.intento-vinculo-f1:'||txid_current()::text||':'||sha,'UTF8')),'hex'),1,32);
 corr:='correlacion_'||substr(sha,33,32);
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.registrar_intento_vinculo_corporativo_f1(
  jsonb_build_object('evento_ref',evento,'operador_login',session_user::text,
   'solicitud_sha256',sha,'modo',p_modo,'correlacion_ref',corr));
 RETURN jsonb_build_object('codigo','CA42','estado','denegado','clave','fuente_o_puntero',
  'observado','incompatible','esperado','fuente_vigente_exacta',
  'auditoria_ref',a.auditoria_ref,'auditoria_huella_sha256',a.huella_sha256);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.auditar_denegacion_publicacion_vinculo_f1(text,text,text,text,text,text,text,text)
 FROM PUBLIC,vec_contexto_actor_v1_runtime;

CREATE FUNCTION vec_contexto_actor_v1.comprobar_publicacion_vinculo_corporativo_f1(
 p_operacion_ref text,p_cuenta_ref text,p_persona_ref text,p_perfil_ref text,
 p_vinculo_contexto_ref text,p_vinculo_corporativo_ref text,p_organizacion_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE p jsonb;
 a record;evento text;corr text;
BEGIN
 BEGIN
 p:=vec_contexto_actor_v1.preimagen_publicacion_vinculo_f1(p_operacion_ref,p_cuenta_ref,p_persona_ref,p_perfil_ref,p_vinculo_contexto_ref,p_vinculo_corporativo_ref,p_organizacion_ref);
 evento:='evento_'||substr(encode(sha256(convert_to('vec.ca.comprobacion-vinculo-f1:'||p_operacion_ref||':'||(p->>'plan_sha256')||':'||(p->>'estado'),'UTF8')),'hex'),1,32);
 corr:='correlacion_'||substr(p->>'plan_sha256',33,32);
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.registrar_acto_vinculo_corporativo_f1(
  jsonb_build_object('evento_ref',evento,'operador_login',session_user::text,'plan_sha256',p->>'plan_sha256',
   'aprobacion_ref',p->>'aprobacion_ref','aprobacion_sha256',p->>'aprobacion_huella_sha256',
   'procedencia_sha256',p->>'procedencia_huella_sha256','version','1','correlacion_ref',corr,
   'tipo','comprobacion','estado',p->>'estado'));
 RETURN jsonb_build_object('codigo','CA42','estado',p->>'estado','huella',p->>'plan_sha256',
  'observado',CASE WHEN p->>'estado'='replay' THEN '1' ELSE '0' END,'esperado','0',
  'auditoria_ref',a.auditoria_ref,'auditoria_huella_sha256',a.huella_sha256);
 EXCEPTION WHEN SQLSTATE '42501' OR SQLSTATE '40001' THEN
  RETURN vec_contexto_actor_v1.auditar_denegacion_publicacion_vinculo_f1('comprobar',
   p_operacion_ref,p_cuenta_ref,p_persona_ref,p_perfil_ref,p_vinculo_contexto_ref,p_vinculo_corporativo_ref,p_organizacion_ref);
 END;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.comprobar_publicacion_vinculo_corporativo_f1(text,text,text,text,text,text,text)
 FROM PUBLIC,vec_contexto_actor_v1_runtime;

CREATE FUNCTION vec_contexto_actor_v1.publicar_vinculo_corporativo_desde_fuente_v1(
 p_operacion_ref text,p_cuenta_ref text,p_persona_ref text,p_perfil_ref text,
 p_vinculo_contexto_ref text,p_vinculo_corporativo_ref text,p_organizacion_ref text,p_plan_sha256 text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE p jsonb; a record; previo vec_contexto_actor_v1.prueba_publicacion_vinculo_f1;
 evento text; corr text;
BEGIN
 BEGIN
 p:=vec_contexto_actor_v1.preimagen_publicacion_vinculo_f1(p_operacion_ref,p_cuenta_ref,p_persona_ref,p_perfil_ref,p_vinculo_contexto_ref,p_vinculo_corporativo_ref,p_organizacion_ref);
 IF p_plan_sha256 !~ '^[0-9a-f]{64}$' OR p_plan_sha256 IS DISTINCT FROM p->>'plan_sha256'
 THEN RAISE EXCEPTION 'CA42: PARO clave=plan actual=deriva esperado=huella_comprobada' USING ERRCODE='40001'; END IF;
 SELECT * INTO previo FROM vec_contexto_actor_v1.prueba_publicacion_vinculo_f1 WHERE operacion_ref=p_operacion_ref FOR SHARE;
 IF FOUND THEN
  IF p->>'estado'<>'replay' OR previo.plan_sha256 IS DISTINCT FROM p_plan_sha256
   OR previo.vinculo_corporativo_ref IS DISTINCT FROM p_vinculo_corporativo_ref OR previo.version<>1
  THEN RAISE EXCEPTION 'CA42: PARO clave=replay actual=divergente esperado=original' USING ERRCODE='40001'; END IF;
  RETURN jsonb_build_object('codigo','CA42','estado','replay','huella',previo.plan_sha256,
   'auditoria_ref',previo.auditoria_ref,'auditoria_huella_sha256',previo.auditoria_huella_sha256,'evento_ref',previo.evento_ref);
 END IF;
 IF p->>'estado'<>'publicable' THEN RAISE EXCEPTION 'CA42: PARO clave=puntero actual=ocupado esperado=alta_inicial' USING ERRCODE='40001'; END IF;
 evento:='evento_'||substr(encode(sha256(convert_to('vec.ca.publicacion-vinculo-f1:'||p_operacion_ref,'UTF8')),'hex'),1,32);
 corr:='correlacion_'||substr(p_plan_sha256,33,32);
 INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones(
  vinculo_corporativo_ref,version,cuenta_ref,cuenta_version,persona_ref,persona_version,
  perfil_ref,perfil_version,vinculo_contexto_ref,vinculo_contexto_version,organizacion_ref,organizacion_version,
  organizacion_procedencia_ref,organizacion_procedencia_version,organizacion_procedencia_huella_sha256,
  organizacion_procedencia_autoridad,superficie,uso,procedencia_ref,procedencia_version,
  procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES(p->>'vinculo_corporativo_ref',1,p->>'cuenta_ref',(p->>'cuenta_version')::numeric,
  p->>'persona_ref',(p->>'persona_version')::numeric,p->>'perfil_ref',(p->>'perfil_version')::numeric,
  p->>'vinculo_contexto_ref',(p->>'vinculo_contexto_version')::numeric,p->>'organizacion_ref',(p->>'organizacion_version')::numeric,
  p->>'organizacion_procedencia_ref',(p->>'organizacion_procedencia_version')::numeric,
  p->>'organizacion_procedencia_huella_sha256','autoridad_maestra_acreditada',
  'interna_corporativa','consulta_rrhh',p->>'procedencia_ref',(p->>'procedencia_version')::numeric,
  p->>'procedencia_huella_sha256','autoridad_maestra_acreditada','activo',(p->>'vigente_desde')::timestamptz,(p->>'vigente_hasta')::timestamptz);
 INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual(cuenta_ref,superficie,uso,vinculo_corporativo_ref,version)
 VALUES(p->>'cuenta_ref','interna_corporativa','consulta_rrhh',p->>'vinculo_corporativo_ref',1);
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.registrar_acto_vinculo_corporativo_f1(
  jsonb_build_object('evento_ref',evento,'operador_login',session_user::text,'plan_sha256',p_plan_sha256,
   'aprobacion_ref',p->>'aprobacion_ref','aprobacion_sha256',p->>'aprobacion_huella_sha256',
   'procedencia_sha256',p->>'procedencia_huella_sha256','version','1','correlacion_ref',corr,
   'tipo','publicacion','estado','publicado'));
 INSERT INTO vec_contexto_actor_v1.prueba_publicacion_vinculo_f1(
  operacion_ref,plan_sha256,vinculo_corporativo_ref,version,auditoria_ref,auditoria_huella_sha256,evento_ref,registrada_en)
 VALUES(p_operacion_ref,p_plan_sha256,p_vinculo_corporativo_ref,1,a.auditoria_ref,a.huella_sha256,evento,a.registrada_en);
 INSERT INTO vec_contexto_actor_v1.buzon_publicacion_vinculo_f1(evento_ref,operacion_ref,auditoria_ref,plan_sha256,creada_en)
 VALUES(evento,p_operacion_ref,a.auditoria_ref,p_plan_sha256,a.registrada_en);
 RETURN jsonb_build_object('codigo','CA42','estado','publicado','huella',p_plan_sha256,
  'auditoria_ref',a.auditoria_ref,'auditoria_huella_sha256',a.huella_sha256,'evento_ref',evento);
 EXCEPTION WHEN SQLSTATE '42501' OR SQLSTATE '40001' THEN
  RETURN vec_contexto_actor_v1.auditar_denegacion_publicacion_vinculo_f1('publicar',
   p_operacion_ref,p_cuenta_ref,p_persona_ref,p_perfil_ref,p_vinculo_contexto_ref,p_vinculo_corporativo_ref,p_organizacion_ref);
 END;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.publicar_vinculo_corporativo_desde_fuente_v1(text,text,text,text,text,text,text,text)
 FROM PUBLIC,vec_contexto_actor_v1_runtime;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_contexto_actor_corporativo_rrhh_publicador;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.comprobar_publicacion_vinculo_corporativo_f1(text,text,text,text,text,text,text)
 TO vec_contexto_actor_corporativo_rrhh_publicador;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.publicar_vinculo_corporativo_desde_fuente_v1(text,text,text,text,text,text,text,text)
 TO vec_contexto_actor_corporativo_rrhh_publicador;
RESET ROLE;
COMMIT;
