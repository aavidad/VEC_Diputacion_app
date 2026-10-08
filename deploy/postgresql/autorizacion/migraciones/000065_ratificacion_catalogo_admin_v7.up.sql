\set ON_ERROR_STOP on
-- AUT65: estructura para ratificar prospectivamente los siete descriptores
-- nominales ausentes del ADMIN v7. No ejecuta el acto ni concede RBAC.
-- El plan aprobado debe declarar cada concesión y su ámbito; el consumidor
-- AUT48:96 exige ["organizacion_ref"] para el módulo administracion.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_ratificacion_catalogo_admin_v1(jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.acreditar_perfil_aplicacion_nominal_vigente_aut48(text,text,text,text,text,text,text,text,jsonb,jsonb)') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.config_ratificacion_catalogo_admin_v1') IS NOT NULL
 OR pg_catalog.to_regrole('vec_admin_ratificacion_catalogo_admin_ejecutor') IS NOT NULL
 OR (SELECT pg_catalog.count(*) FROM vec_autorizacion.catalogo_accion_nominal_v1 WHERE version_rol_ref='rol:administracion_perfiles:v7')<>13
 OR (SELECT pg_catalog.jsonb_array_length(documento->'concesiones') FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v7') IS DISTINCT FROM 20
 THEN RAISE EXCEPTION 'AUT65: PARO clave=preimagen actual=divergente esperado=ADMIN7_AD230_13de20' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE TABLE vec_autorizacion.config_ratificacion_catalogo_admin_v1 (
 login_nombre name PRIMARY KEY,
 plan_sha256 text NOT NULL UNIQUE CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 preimagen_sha256 text NOT NULL CHECK(preimagen_sha256 ~ '^[0-9a-f]{64}$'),
 catalogo_sha256 text NOT NULL CHECK(catalogo_sha256 ~ '^[0-9a-f]{64}$'),
 aprobacion_ref text NOT NULL CHECK(aprobacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
 aprobacion_sha256 text NOT NULL CHECK(aprobacion_sha256 ~ '^[0-9a-f]{64}$'),
 entorno text NOT NULL CHECK(entorno='desarrollo'),
 vigente_desde timestamptz NOT NULL,vigente_hasta timestamptz NOT NULL,
 CHECK(pg_catalog.isfinite(vigente_desde) AND pg_catalog.isfinite(vigente_hasta) AND vigente_hasta>vigente_desde)
);
ALTER TABLE vec_autorizacion.config_ratificacion_catalogo_admin_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.config_ratificacion_catalogo_admin_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.config_ratificacion_catalogo_admin_v1 TO vec_autorizacion_propietario
 USING(current_user='vec_autorizacion_propietario') WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.config_ratificacion_catalogo_admin_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.config_ratificacion_catalogo_admin_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.config_ratificacion_catalogo_admin_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.config_ratificacion_catalogo_admin_v1 FROM PUBLIC;
CREATE TABLE vec_autorizacion.registro_ratificacion_catalogo_admin_v1 (
 operacion_ref text PRIMARY KEY CHECK(operacion_ref ~ '^rca_[A-Za-z0-9_-]{22,124}$'),
 plan_sha256 text NOT NULL UNIQUE CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 plan bytea NOT NULL CHECK(pg_catalog.octet_length(plan) BETWEEN 1 AND 32768),
 login_nombre name NOT NULL,
 aprobacion_ref text NOT NULL,aprobacion_sha256 text NOT NULL CHECK(aprobacion_sha256 ~ '^[0-9a-f]{64}$'),
 auditoria_ref text NOT NULL UNIQUE,recibo jsonb NOT NULL CHECK(pg_catalog.jsonb_typeof(recibo)='object'),
 registrada_en timestamptz NOT NULL
);
ALTER TABLE vec_autorizacion.registro_ratificacion_catalogo_admin_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.registro_ratificacion_catalogo_admin_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.registro_ratificacion_catalogo_admin_v1 TO vec_autorizacion_propietario
 USING(current_user='vec_autorizacion_propietario') WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.registro_ratificacion_catalogo_admin_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.registro_ratificacion_catalogo_admin_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.registro_ratificacion_catalogo_admin_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.registro_ratificacion_catalogo_admin_v1 FROM PUBLIC;
RESET ROLE;
CREATE ROLE vec_admin_ratificacion_catalogo_admin_ejecutor NOLOGIN NOINHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO vec_admin_ratificacion_catalogo_admin_ejecutor',pg_catalog.current_database()); END $conexion$;
SET LOCAL ROLE vec_autorizacion_propietario;

CREATE FUNCTION vec_autorizacion.exigir_operador_ratificacion_catalogo_admin_v1()
RETURNS vec_autorizacion.config_ratificacion_catalogo_admin_v1 LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE l record;g record;cfg vec_autorizacion.config_ratificacion_catalogo_admin_v1;ns oid;db oid;f oid;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable' OR pg_catalog.current_setting('transaction_read_only')<>'off'
 OR pg_catalog.current_setting('TimeZone')<>'UTC' OR pg_catalog.current_setting('role')<>'none'
 THEN RAISE EXCEPTION 'AUT65: PARO clave=transaccion actual=divergente esperado=SERIALIZABLE_RW_UTC_sin_SETROLE' USING ERRCODE='25000'; END IF;
 SELECT * INTO l FROM pg_catalog.pg_roles WHERE rolname=session_user;
 SELECT * INTO g FROM pg_catalog.pg_roles WHERE rolname='vec_admin_ratificacion_catalogo_admin_ejecutor';
 ns:=pg_catalog.to_regnamespace('vec_autorizacion');
 SELECT oid INTO db FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database();
 f:=pg_catalog.to_regprocedure('vec_autorizacion.ratificar_catalogo_admin_v7(text,text)');
 IF l.oid IS NULL OR g.oid IS NULL OR NOT l.rolcanlogin OR NOT l.rolinherit OR l.rolsuper OR l.rolcreaterole OR l.rolcreatedb OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
 OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreaterole OR g.rolcreatedb OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members WHERE member=l.oid)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=l.oid AND roleid=g.oid AND inherit_option AND NOT set_option AND NOT admin_option)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=g.oid)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting WHERE setrole IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::pg_catalog.regclass AND refobjid=l.oid)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::pg_catalog.regclass AND refobjid=g.oid
  AND NOT ((dbid=db AND classid='pg_catalog.pg_namespace'::pg_catalog.regclass AND objid=ns AND deptype='a')
   OR (dbid=db AND classid='pg_catalog.pg_proc'::pg_catalog.regclass AND objid=f AND deptype='a')
   OR (dbid=0 AND classid='pg_catalog.pg_database'::pg_catalog.regclass AND objid=db AND deptype='a')))
 OR NOT COALESCE((SELECT pg_catalog.count(*)=1 AND pg_catalog.bool_and(x.privilege_type='CONNECT' AND NOT x.is_grantable)
  FROM pg_catalog.pg_database d CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(d.datacl,pg_catalog.acldefault('d',d.datdba))) x
  WHERE d.oid=db AND x.grantee=g.oid),false)
 OR NOT COALESCE((SELECT pg_catalog.count(*)=1 AND pg_catalog.bool_and(x.privilege_type='USAGE' AND NOT x.is_grantable)
  FROM pg_catalog.pg_namespace n CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(n.nspacl,pg_catalog.acldefault('n',n.nspowner))) x
  WHERE n.oid=ns AND x.grantee=g.oid),false)
 OR NOT COALESCE((SELECT pg_catalog.count(*)=1 AND pg_catalog.bool_and(x.privilege_type='EXECUTE' AND NOT x.is_grantable)
  FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) x
  WHERE p.oid=f AND x.grantee=g.oid),false)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_database d CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(d.datacl,pg_catalog.acldefault('d',d.datdba))) x WHERE d.oid=db AND x.grantee=l.oid)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(n.nspacl,pg_catalog.acldefault('n',n.nspowner))) x WHERE n.oid=ns AND x.grantee=l.oid)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee=l.oid)
 OR pg_catalog.has_schema_privilege(l.oid,ns,'CREATE')
 OR pg_catalog.has_database_privilege(l.oid,db,'CREATE,TEMP')
 THEN RAISE EXCEPTION 'AUT65: PARO clave=operador actual=no_acreditado esperado=LOGIN_minimo_exclusivo' USING ERRCODE='42501'; END IF;
 SELECT * INTO cfg FROM vec_autorizacion.config_ratificacion_catalogo_admin_v1 WHERE login_nombre=session_user FOR SHARE;
 IF NOT FOUND OR pg_catalog.clock_timestamp()<cfg.vigente_desde OR pg_catalog.clock_timestamp()>=cfg.vigente_hasta
 THEN RAISE EXCEPTION 'AUT65: PARO clave=configuracion actual=ausente_o_caducada esperado=aprobacion_externa_vigente' USING ERRCODE='42501'; END IF;
 RETURN cfg;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.exigir_operador_ratificacion_catalogo_admin_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.ratificar_catalogo_admin_v7(plan_canonico text,sha_aprobado text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='20s'
AS $f$
DECLARE cfg vec_autorizacion.config_ratificacion_catalogo_admin_v1;p jsonb;sha text;pre jsonb;pre_sha text;cat jsonb;cat_sha text;
 r record;c record;actual record;d jsonb;concesion jsonb;v_accion text;recurso text;descriptores_sha text;instante timestamptz(6);
 aud record;evento jsonb;recibo jsonb;previo record;postcat jsonb;
BEGIN
 cfg:=vec_autorizacion.exigir_operador_ratificacion_catalogo_admin_v1();
 IF plan_canonico IS NULL OR pg_catalog.octet_length(plan_canonico) NOT BETWEEN 1 AND 32768
 OR sha_aprobado IS DISTINCT FROM cfg.plan_sha256
 THEN RAISE EXCEPTION 'AUT65: PARO clave=plan actual=no_aprobado esperado=bytes_y_SHA_externos' USING ERRCODE='42501'; END IF;
 BEGIN p:=plan_canonico::jsonb; EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'AUT65: PARO clave=plan actual=JSON_invalido esperado=canonico' USING ERRCODE='22023'; END;
 sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(plan_canonico,'UTF8')),'hex');
 IF sha IS DISTINCT FROM cfg.plan_sha256 OR plan_canonico IS DISTINCT FROM p::text
 OR pg_catalog.jsonb_typeof(p) IS DISTINCT FROM 'object'
 OR NOT p ?& ARRAY['esquema','version','operacion_ref','preparado_en','caduca_en','rol_ref','rol_sha256','control_revision','control_sha256','catalogo_sha256','preimagen_sha256','descriptores']
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(p))<>12
 OR p->>'esquema' IS DISTINCT FROM 'vec.admin.ratificacion-catalogo.v1'
 OR p->>'version' IS DISTINCT FROM '0600'
 OR p->>'operacion_ref' !~ '^rca_[A-Za-z0-9_-]{22,124}$'
 OR p->>'rol_ref' IS DISTINCT FROM 'rol:administracion_perfiles:v7'
 OR p->>'rol_sha256' !~ '^[0-9a-f]{64}$'
 OR p->>'control_sha256' !~ '^[0-9a-f]{64}$'
 OR p->>'catalogo_sha256' !~ '^[0-9a-f]{64}$'
 OR p->>'preimagen_sha256' !~ '^[0-9a-f]{64}$'
 OR p->>'control_revision' !~ '^[1-9][0-9]{0,17}$'
 OR pg_catalog.jsonb_typeof(p->'descriptores') IS DISTINCT FROM 'array'
 OR pg_catalog.jsonb_array_length(p->'descriptores')<>7
 THEN RAISE EXCEPTION 'AUT65: PARO clave=plan actual=estructura_divergente esperado=0600_siete_descriptores' USING ERRCODE='22023'; END IF;
 IF p->>'preimagen_sha256' IS DISTINCT FROM cfg.preimagen_sha256 OR p->>'catalogo_sha256' IS DISTINCT FROM cfg.catalogo_sha256
 OR (p->>'preparado_en')::timestamptz>pg_catalog.clock_timestamp()
 OR (p->>'caduca_en')::timestamptz<=pg_catalog.clock_timestamp()
 OR (p->>'caduca_en')::timestamptz<=(p->>'preparado_en')::timestamptz
 OR (p->>'caduca_en')::timestamptz>cfg.vigente_hasta
 THEN RAISE EXCEPTION 'AUT65: PARO clave=aprobacion actual=divergente_o_caducada esperado=plan_config_vigentes' USING ERRCODE='42501'; END IF;
 -- La lista y sus ámbitos son bytes del plan aprobado, jamás valores inferidos.
 IF p->'descriptores' IS DISTINCT FROM (SELECT pg_catalog.jsonb_agg(x.value ORDER BY x.value->>'accion_ref') FROM pg_catalog.jsonb_array_elements(p->'descriptores') x)
 THEN RAISE EXCEPTION 'AUT65: PARO clave=descriptores actual=orden_divergente esperado=accion_asc' USING ERRCODE='22023'; END IF;
 FOR d IN SELECT x.value FROM pg_catalog.jsonb_array_elements(p->'descriptores') x LOOP
  IF pg_catalog.jsonb_typeof(d) IS DISTINCT FROM 'object'
  OR NOT d ?& ARRAY['accion_ref','concesion','dimensiones_ambito']
  OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(d))<>3
  OR d->>'accion_ref' IS DISTINCT FROM 'accion:'||(d#>>'{concesion,accion}')
  OR d->'dimensiones_ambito' IS DISTINCT FROM '["organizacion_ref"]'::jsonb
  THEN RAISE EXCEPTION 'AUT65: PARO clave=descriptor actual=incompatible esperado=fuente_y_ambito_AUT48' USING ERRCODE='22023'; END IF;
  v_accion:=d#>>'{concesion,accion}';
  SELECT x.tipo_recurso INTO recurso FROM (VALUES
   ('administracion.perfiles.aprobar','propuesta_perfil'),
   ('administracion.perfiles.historial.consultar','historial_perfil'),
   ('administracion.perfiles.otorgar','perfil'),
   ('administracion.perfiles.proponer','perfil'),
   ('administracion.perfiles.rechazar','propuesta_perfil'),
   ('administracion.perfiles.recibo.consultar','recibo_perfil'),
   ('administracion.perfiles.revocar','perfil')) x(accion,tipo_recurso) WHERE x.accion=v_accion;
  IF recurso IS NULL OR d->'concesion' IS DISTINCT FROM pg_catalog.jsonb_build_object(
   'accion',v_accion,'modulo_id','administracion','tipo_recurso',recurso,
   'finalidades',pg_catalog.jsonb_build_array('gestion_perfiles'),'garantia_minima','alto')
  THEN RAISE EXCEPTION 'AUT65: PARO clave=concesion actual=divergente esperado=AUT23_sin_campos_obligaciones' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF (SELECT pg_catalog.count(DISTINCT x.value->>'accion_ref') FROM pg_catalog.jsonb_array_elements(p->'descriptores') x)<>7
 OR (SELECT pg_catalog.jsonb_agg(x.value->>'accion_ref' ORDER BY x.value->>'accion_ref') FROM pg_catalog.jsonb_array_elements(p->'descriptores') x)
 IS DISTINCT FROM '["accion:administracion.perfiles.aprobar","accion:administracion.perfiles.historial.consultar","accion:administracion.perfiles.otorgar","accion:administracion.perfiles.proponer","accion:administracion.perfiles.rechazar","accion:administracion.perfiles.recibo.consultar","accion:administracion.perfiles.revocar"]'::jsonb
 THEN RAISE EXCEPTION 'AUT65: PARO clave=conjunto actual=divergente esperado=siete_acciones_AUT23' USING ERRCODE='22023'; END IF;
 descriptores_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to((p->'descriptores')::text,'UTF8')),'hex');
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO previo FROM vec_autorizacion.registro_ratificacion_catalogo_admin_v1 WHERE operacion_ref=p->>'operacion_ref' FOR SHARE;
 IF FOUND THEN
  IF previo.plan_sha256 IS DISTINCT FROM sha OR previo.plan IS DISTINCT FROM pg_catalog.convert_to(plan_canonico,'UTF8')
  OR previo.login_nombre IS DISTINCT FROM session_user OR previo.aprobacion_ref IS DISTINCT FROM cfg.aprobacion_ref
  OR previo.aprobacion_sha256 IS DISTINCT FROM cfg.aprobacion_sha256
  THEN RAISE EXCEPTION 'AUT65: PARO clave=replay actual=material_divergente esperado=plan_aprobacion_original' USING ERRCODE='23505'; END IF;
  SELECT * INTO r FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v7' FOR SHARE;
  SELECT x.* INTO c FROM vec_autorizacion.control_vigencia_version_rol_actual a JOIN vec_autorizacion.control_vigencia_version_rol x USING(version_rol_ref,revision)
   WHERE a.version_rol_ref='rol:administracion_perfiles:v7' FOR SHARE OF a,x;
  IF r.huella_sha256 IS DISTINCT FROM p->>'rol_sha256' OR c.estado IS DISTINCT FROM 'habilitada'
  OR c.huella_sha256 IS DISTINCT FROM p->>'control_sha256'
  OR (SELECT pg_catalog.count(*) FROM vec_autorizacion.catalogo_accion_nominal_v1 WHERE version_rol_ref='rol:administracion_perfiles:v7')<>20
  OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(p->'descriptores') x
   WHERE NOT EXISTS(SELECT 1 FROM vec_autorizacion.catalogo_accion_nominal_v1 n
    WHERE n.version_rol_ref='rol:administracion_perfiles:v7' AND n.accion_ref=x.value->>'accion_ref'
    AND n.concesion=x.value->'concesion' AND n.dimensiones_ambito=x.value->'dimensiones_ambito'
    AND n.vigente_desde=(previo.recibo->>'vigente_desde')::timestamptz AND n.fuente_huella_sha256=r.huella_sha256))
  THEN RAISE EXCEPTION 'AUT65: PARO clave=replay_estado actual=divergente esperado=ratificacion_original_viva' USING ERRCODE='40001'; END IF;
  RETURN pg_catalog.jsonb_build_object('recibo',previo.recibo,'replay',true);
 END IF;
 LOCK TABLE vec_autorizacion.catalogo_accion_nominal_v1 IN SHARE ROW EXCLUSIVE MODE;
 SELECT * INTO r FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v7' FOR SHARE;
 SELECT x.* INTO c FROM vec_autorizacion.control_vigencia_version_rol_actual a JOIN vec_autorizacion.control_vigencia_version_rol x USING(version_rol_ref,revision)
 WHERE a.version_rol_ref='rol:administracion_perfiles:v7' FOR UPDATE OF a;
 SELECT * INTO actual FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 WHERE version_rol_ref='rol:administracion_perfiles:v7' FOR SHARE;
 SELECT COALESCE(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(x) ORDER BY x.accion_ref,x.version),'[]'::jsonb) INTO cat
 FROM vec_autorizacion.catalogo_accion_nominal_v1 x WHERE x.version_rol_ref='rol:administracion_perfiles:v7';
 cat_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(cat::text,'UTF8')),'hex');
 pre:=pg_catalog.jsonb_build_object('rol',pg_catalog.to_jsonb(r),'control',pg_catalog.to_jsonb(c),'catalogo',cat);
 pre_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pre::text,'UTF8')),'hex');
 IF r.version_rol_ref IS NULL OR r.rol_id IS DISTINCT FROM 'administracion_perfiles' OR r.version IS DISTINCT FROM 7
 OR r.documento->>'estado' IS DISTINCT FROM 'publicada' OR r.huella_sha256 IS DISTINCT FROM p->>'rol_sha256'
 OR r.huella_sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
 OR c.version_rol_ref IS NULL OR c.estado IS DISTINCT FROM 'habilitada'
 OR c.revision::text IS DISTINCT FROM p->>'control_revision' OR c.huella_sha256 IS DISTINCT FROM p->>'control_sha256'
 OR c.huella_sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_control_rol_admin_v1(c.documento),'UTF8')),'hex')
 OR actual.version_rol_huella_sha256 IS DISTINCT FROM r.huella_sha256 OR actual.categoria_administrativa IS DISTINCT FROM 'aplicacion'
 OR actual.tipo_perfil IS DISTINCT FROM 'fijo_sistema'
 OR pg_catalog.jsonb_array_length(r.documento->'concesiones')<>20 OR pg_catalog.jsonb_array_length(cat)<>13
 OR cat_sha IS DISTINCT FROM p->>'catalogo_sha256' OR pre_sha IS DISTINCT FROM p->>'preimagen_sha256'
 OR EXISTS(SELECT 1 FROM vec_autorizacion.catalogo_accion_nominal_v1 n WHERE n.accion_ref IN
  (SELECT x.value->>'accion_ref' FROM pg_catalog.jsonb_array_elements(p->'descriptores') x))
 OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(cat) x WHERE x.value->>'fuente_ref'<>'rol:administracion_perfiles:v7'
  OR x.value->>'fuente_version'<>'7' OR x.value->>'fuente_huella_sha256'<>r.huella_sha256
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(r.documento->'concesiones') y WHERE y.value=x.value->'concesion'))
 OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(p->'descriptores') x
  WHERE NOT EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(r.documento->'concesiones') y WHERE y.value=x.value->'concesion'))
 THEN RAISE EXCEPTION 'AUT65: PARO clave=CAS_preimagen actual=divergente esperado=rol_control_catalogo_13de20' USING ERRCODE='40001'; END IF;
 instante:=pg_catalog.clock_timestamp();
 FOR d IN SELECT x.value FROM pg_catalog.jsonb_array_elements(p->'descriptores') x LOOP
  INSERT INTO vec_autorizacion.catalogo_accion_nominal_v1(
   accion_ref,version,fuente_ref,fuente_version,fuente_huella_sha256,version_rol_ref,concesion,dimensiones_ambito,clase_control,vigente_desde,vigente_hasta)
  VALUES(d->>'accion_ref',1,'rol:administracion_perfiles:v7',7,r.huella_sha256,'rol:administracion_perfiles:v7',
   d->'concesion',d->'dimensiones_ambito','administrador_aplicacion',instante,NULL);
 END LOOP;
 evento:=pg_catalog.jsonb_build_object('tipo_registro','ratificacion_catalogo_admin','evento_ref','evento_'||pg_catalog.substr(sha,1,32),
  'operador_login',session_user::text,'plan_sha256',sha,'preimagen_sha256',pre_sha,'catalogo_sha256',cat_sha,
  'rol_sha256',r.huella_sha256,'control_sha256',c.huella_sha256,'aprobacion_ref',cfg.aprobacion_ref,
  'aprobacion_sha256',cfg.aprobacion_sha256,'descriptores_sha256',descriptores_sha,'proceso','postgresql',
  'canal','operacion_tecnica_privada','finalidad_ref','ratificacion_catalogo_admin_v7','correlacion_ref','correlacion_'||pg_catalog.substr(sha,33,32));
 SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_ratificacion_catalogo_admin_v1(evento);
 SELECT COALESCE(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(x) ORDER BY x.accion_ref),'[]'::jsonb) INTO postcat
 FROM vec_autorizacion.catalogo_accion_nominal_v1 x WHERE x.version_rol_ref='rol:administracion_perfiles:v7';
 IF pg_catalog.jsonb_array_length(postcat)<>20 OR pg_catalog.clock_timestamp()>=(p->>'caduca_en')::timestamptz
 THEN RAISE EXCEPTION 'AUT65: PARO clave=postimagen actual=incompatible esperado=20_vigentes_y_plan_vigente' USING ERRCODE='40001'; END IF;
 recibo:=pg_catalog.jsonb_build_object('esquema','vec.admin.ratificacion-catalogo.recibo.v1','operacion_ref',p->>'operacion_ref',
  'plan_sha256',sha,'preimagen_sha256',pre_sha,'rol_ref',r.version_rol_ref,'rol_sha256',r.huella_sha256,
  'control_revision',c.revision,'control_sha256',c.huella_sha256,'catalogo_previo_sha256',cat_sha,
  'descriptores_sha256',descriptores_sha,'catalogo_posterior_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(postcat::text,'UTF8')),'hex'),
  'vigente_desde',instante,'aprobacion_ref',cfg.aprobacion_ref,'aprobacion_sha256',cfg.aprobacion_sha256,
  'operador_login',session_user::text,'auditoria_ref',aud.auditoria_ref,'auditoria_secuencia',aud.secuencia,
  'auditoria_huella_sha256',aud.huella_sha256,'confirmado_en',aud.registrada_en);
 INSERT INTO vec_autorizacion.registro_ratificacion_catalogo_admin_v1(
  operacion_ref,plan_sha256,plan,login_nombre,aprobacion_ref,aprobacion_sha256,auditoria_ref,recibo,registrada_en)
 VALUES(p->>'operacion_ref',sha,pg_catalog.convert_to(plan_canonico,'UTF8'),session_user,cfg.aprobacion_ref,
  cfg.aprobacion_sha256,aud.auditoria_ref,recibo,aud.registrada_en);
 PERFORM vec_autorizacion.exigir_operador_ratificacion_catalogo_admin_v1();
 RETURN pg_catalog.jsonb_build_object('recibo',recibo,'replay',false);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.ratificar_catalogo_admin_v7(text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_ratificacion_catalogo_admin_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion.ratificar_catalogo_admin_v7(text,text) TO vec_admin_ratificacion_catalogo_admin_ejecutor;
RESET ROLE;
DO $acl$
DECLARE f oid;t regclass;g oid:='vec_admin_ratificacion_catalogo_admin_ejecutor'::pg_catalog.regrole;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_autorizacion.exigir_operador_ratificacion_catalogo_admin_v1()'::pg_catalog.regprocedure::oid,
  'vec_autorizacion.ratificar_catalogo_admin_v7(text,text)'::pg_catalog.regprocedure::oid] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_propietario'::pg_catalog.regrole
   AND p.prosecdef AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on'])
  OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) x
   WHERE p.oid=f AND x.grantee<>p.proowner AND NOT(f='vec_autorizacion.ratificar_catalogo_admin_v7(text,text)'::pg_catalog.regprocedure::oid
    AND x.grantee=g AND x.privilege_type='EXECUTE' AND NOT x.is_grantable))
  THEN RAISE EXCEPTION 'AUT65: PARO clave=ACL_funcion actual=divergente esperado=fachada_exclusiva' USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH t IN ARRAY ARRAY[
  'vec_autorizacion.config_ratificacion_catalogo_admin_v1'::pg_catalog.regclass,
  'vec_autorizacion.registro_ratificacion_catalogo_admin_v1'::pg_catalog.regclass] LOOP
  IF EXISTS(SELECT 1 FROM pg_catalog.pg_class c WHERE c.oid=t AND (c.relowner<>'vec_autorizacion_propietario'::pg_catalog.regrole
   OR NOT c.relrowsecurity OR NOT c.relforcerowsecurity))
  OR EXISTS(SELECT 1 FROM pg_catalog.pg_class c CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(c.relacl,pg_catalog.acldefault('r',c.relowner))) x
   WHERE c.oid=t AND x.grantee<>c.relowner)
  OR EXISTS(SELECT 1 FROM pg_catalog.pg_type y JOIN pg_catalog.pg_class c ON c.reltype=y.oid
   CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(y.typacl,pg_catalog.acldefault('T',y.typowner))) x
   WHERE c.oid=t AND x.grantee<>y.typowner)
  THEN RAISE EXCEPTION 'AUT65: PARO clave=ACL_tabla actual=divergente esperado=propietario_RLS_forzada' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;
COMMIT;
