\set ON_ERROR_STOP on
-- AUT49: registro gobernado de perfiles asignables (clase ordinario).
-- Un operador técnico con LOGIN propio y una fila de configuración que el DBA
-- liga a la huella de un plan aprobado registra versiones de rol ORDINARIAS ya
-- publicadas en rol_administrable_exacto_v1. Así el administrador podrá
-- asignarlas después desde la pantalla. Instalar esta estructura no registra
-- ningún perfil, no publica concesiones ni crea asignaciones. Nunca admite el
-- rol de administración, Sistemas, roles sensibles, Intervención ni perfiles
-- fijos. La auditoría común (AD196) se escribe en la misma transacción.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) OR current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999 THEN RAISE EXCEPTION 'AUT49: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF to_regclass('vec_autorizacion.rol_administrable_exacto_v1') IS NULL OR to_regclass('vec_autorizacion.rol_sensible_exacto') IS NULL
 OR to_regclass('vec_autorizacion.perfil_fijo_categoria_nominal_v1') IS NULL OR to_regclass('vec_autorizacion.asignacion_perfil_externa') IS NULL
 OR to_regprocedure('vec_autorizacion.rechazar_mutacion_inmutable()') IS NULL OR to_regprocedure('vec_autorizacion.ambitos_positivos_validos(jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion.resolver_rol_administrable_v1(text)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_perfiles_asignables_admin_v1(jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_perfiles_asignables_admin_v1(jsonb)') IS NULL
 OR to_regclass('vec_autorizacion.config_perfiles_asignables_admin_v1') IS NOT NULL OR to_regrole('vec_admin_perfiles_asignables_ejecutor') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT49: PARO clave=dependencias actual=divergente esperado=AUT24_AD196_sin_AUT49' USING ERRCODE='55000'; END IF;
END $pre$;

SET LOCAL ROLE vec_autorizacion_propietario;
-- Una fila por LOGIN técnico, escrita por el DBA tras la aprobación externa.
CREATE TABLE vec_autorizacion.config_perfiles_asignables_admin_v1(
 login_nombre name PRIMARY KEY,
 plan_sha256 text NOT NULL UNIQUE CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 aprobacion_ref text NOT NULL CHECK(aprobacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
 aprobacion_sha256 text NOT NULL CHECK(aprobacion_sha256 ~ '^[0-9a-f]{64}$'),
 entorno text NOT NULL CHECK(entorno='desarrollo'),
 vigente_desde timestamptz NOT NULL,vigente_hasta timestamptz NOT NULL,
 CHECK(isfinite(vigente_desde) AND isfinite(vigente_hasta) AND vigente_hasta>vigente_desde AND vigente_hasta-vigente_desde<=interval '1 day')
);
-- Historia del registro: plan exacto, recibo y auditoría. Sirve al replay.
CREATE TABLE vec_autorizacion.registro_perfiles_asignables_admin_v1(
 operacion_ref text PRIMARY KEY CHECK(operacion_ref ~ '^rpa_[A-Za-z0-9_-]{22,124}$'),
 plan_sha256 text NOT NULL UNIQUE CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 plan bytea NOT NULL CHECK(octet_length(plan) BETWEEN 1 AND 65536 AND encode(sha256(plan),'hex')=plan_sha256),
 login_nombre name NOT NULL,
 auditoria_ref text NOT NULL,
 recibo jsonb NOT NULL CHECK(jsonb_typeof(recibo)='object'),
 registrada_en timestamptz NOT NULL CHECK(isfinite(registrada_en))
);
DO $tablas$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['config_perfiles_asignables_admin_v1','registro_perfiles_asignables_admin_v1'] LOOP
  EXECUTE format('ALTER TABLE vec_autorizacion.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_autorizacion.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_autorizacion.%I FOR ALL TO vec_autorizacion_propietario USING(current_user=''vec_autorizacion_propietario'') WITH CHECK(current_user=''vec_autorizacion_propietario'')',t);
  EXECUTE format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_autorizacion.%I FROM PUBLIC',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_autorizacion.%I FROM PUBLIC',t);
 END LOOP;
END $tablas$;
RESET ROLE;
CREATE ROLE vec_admin_perfiles_asignables_ejecutor NOLOGIN NOINHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_admin_perfiles_asignables_ejecutor',current_database()); END $conexion$;
SET LOCAL ROLE vec_autorizacion_propietario;

-- El LOGIN debe ser mínimo y exclusivo, como en AUT42/AUT45: miembro único del
-- grupo con INHERIT y sin SET/ADMIN, sin ajustes propios ni permisos directos.
CREATE FUNCTION vec_autorizacion.exigir_operador_perfiles_asignables_admin_v1()
RETURNS vec_autorizacion.config_perfiles_asignables_admin_v1 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE l record;g record;cfg vec_autorizacion.config_perfiles_asignables_admin_v1;ns oid;db oid;f oid;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' OR current_setting('role')<>'none' THEN RAISE EXCEPTION 'AUT49: PARO clave=transaccion actual=divergente esperado=SERIALIZABLE_RW_UTC_sin_SETROLE' USING ERRCODE='25000'; END IF;
 SELECT * INTO l FROM pg_roles WHERE rolname=session_user;SELECT * INTO g FROM pg_roles WHERE rolname='vec_admin_perfiles_asignables_ejecutor';
 ns:=to_regnamespace('vec_autorizacion');SELECT oid INTO db FROM pg_database WHERE datname=current_database();f:=to_regprocedure('vec_autorizacion.registrar_perfiles_asignables_admin_v1(text,text)');
 IF l.oid IS NULL OR g.oid IS NULL OR NOT l.rolcanlogin OR NOT l.rolinherit OR l.rolsuper OR l.rolcreaterole OR l.rolcreatedb OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
 OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreaterole OR g.rolcreatedb OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
 OR (SELECT count(*) FROM pg_auth_members WHERE member=l.oid)<>1 OR NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=l.oid AND roleid=g.oid AND inherit_option AND NOT set_option AND NOT admin_option)
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=g.oid) OR EXISTS(SELECT 1 FROM pg_db_role_setting WHERE setrole IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=l.oid)
 OR EXISTS(SELECT 1 FROM pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=g.oid AND NOT((dbid=db AND classid='pg_catalog.pg_namespace'::regclass AND objid=ns AND deptype='a') OR(dbid=db AND classid='pg_catalog.pg_proc'::regclass AND objid=f AND deptype='a') OR(dbid=0 AND classid='pg_catalog.pg_database'::regclass AND objid=db AND deptype='a')))
 OR NOT COALESCE((SELECT count(*)=1 AND bool_and(x.privilege_type='CONNECT' AND NOT x.is_grantable) FROM pg_database d CROSS JOIN LATERAL aclexplode(COALESCE(d.datacl,acldefault('d',d.datdba))) x WHERE d.oid=db AND x.grantee=g.oid),false)
 OR NOT COALESCE((SELECT count(*)=1 AND bool_and(x.privilege_type='USAGE' AND NOT x.is_grantable) FROM pg_namespace n CROSS JOIN LATERAL aclexplode(COALESCE(n.nspacl,acldefault('n',n.nspowner))) x WHERE n.oid=ns AND x.grantee=g.oid),false)
 OR NOT COALESCE((SELECT count(*)=1 AND bool_and(x.privilege_type='EXECUTE' AND NOT x.is_grantable) FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee=g.oid),false)
 OR EXISTS(SELECT 1 FROM pg_database d CROSS JOIN LATERAL aclexplode(COALESCE(d.datacl,acldefault('d',d.datdba))) x WHERE d.oid=db AND x.grantee=l.oid)
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(COALESCE(n.nspacl,acldefault('n',n.nspowner))) x WHERE n.oid=ns AND x.grantee=l.oid)
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee=l.oid)
 OR has_schema_privilege(l.oid,ns,'CREATE') OR has_database_privilege(l.oid,db,'CREATE,TEMP') THEN RAISE EXCEPTION 'AUT49: PARO clave=operador actual=no_acreditado esperado=LOGIN_minimo_exclusivo' USING ERRCODE='42501'; END IF;
 SELECT * INTO cfg FROM vec_autorizacion.config_perfiles_asignables_admin_v1 WHERE login_nombre=session_user FOR SHARE;
 IF NOT FOUND OR clock_timestamp()<cfg.vigente_desde OR clock_timestamp()>=cfg.vigente_hasta THEN RAISE EXCEPTION 'AUT49: PARO clave=configuracion actual=ausente_o_caducada esperado=aprobacion_externa_vigente' USING ERRCODE='42501'; END IF;
 RETURN cfg;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.exigir_operador_perfiles_asignables_admin_v1() FROM PUBLIC;

-- Un perfil del plan es asignable sólo si es una versión ORDINARIA publicada y
-- habilitada, con su huella y control exactos. La clasificación positiva es la
-- aprobación de Alberto sobre la lista exacta del plan; estas exclusiones son la
-- defensa técnica: administración, Sistemas, roles sensibles o fijos,
-- Intervención y fiscalización, aspirantes y usuarios externos nunca entran,
-- aunque el plan los incluya. Comparan por rol_id para cubrir cualquier versión.
CREATE FUNCTION vec_autorizacion.validar_perfil_asignable_admin_v1(t jsonb,p_registro boolean)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' AS $f$
DECLARE r record;c record;dur numeric;
BEGIN
 IF jsonb_typeof(t) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(t))<>9
 OR NOT t ?& ARRAY['version_rol_ref','version_rol_sha256','control_revision','control_sha256','unidad_requerida','ambitos_fijos','vigente_desde','vigente_hasta','duracion_propuesta_segundos']
 OR jsonb_typeof(t->'version_rol_ref') IS DISTINCT FROM 'string' OR t->>'version_rol_ref' !~ '^rol:[a-z0-9][a-z0-9_:.-]{0,190}:v[1-9][0-9]{0,8}$'
 OR jsonb_typeof(t->'version_rol_sha256') IS DISTINCT FROM 'string' OR t->>'version_rol_sha256' !~ '^[0-9a-f]{64}$'
 OR jsonb_typeof(t->'control_revision') IS DISTINCT FROM 'number' OR t->>'control_revision' !~ '^[1-9][0-9]{0,18}$'
 OR jsonb_typeof(t->'control_sha256') IS DISTINCT FROM 'string' OR t->>'control_sha256' !~ '^[0-9a-f]{64}$'
 OR jsonb_typeof(t->'unidad_requerida') IS DISTINCT FROM 'boolean'
 OR jsonb_typeof(t->'ambitos_fijos') IS DISTINCT FROM 'array' OR jsonb_array_length(t->'ambitos_fijos')>16
 OR vec_autorizacion.ambitos_positivos_validos(jsonb_build_object('ambitos',t->'ambitos_fijos')) IS NOT TRUE
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(t->'ambitos_fijos') a WHERE a->>'clave'='unidad_ref')
 OR jsonb_typeof(t->'vigente_desde') IS DISTINCT FROM 'string' OR jsonb_typeof(t->'vigente_hasta') IS DISTINCT FROM 'string'
 OR t->>'vigente_desde' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$' OR t->>'vigente_hasta' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$'
 OR jsonb_typeof(t->'duracion_propuesta_segundos') IS DISTINCT FROM 'number' OR t->>'duracion_propuesta_segundos' !~ '^[1-9][0-9]{1,7}$'
 THEN RAISE EXCEPTION 'AUT49: PARO clave=perfil actual=invalido esperado=campos_exactos' USING ERRCODE='22023'; END IF;
 dur:=(t->>'duracion_propuesta_segundos')::numeric;
 IF dur NOT BETWEEN 60 AND 31536000 OR (t->>'vigente_desde')::timestamptz>=(t->>'vigente_hasta')::timestamptz
 OR NOT isfinite((t->>'vigente_desde')::timestamptz) OR NOT isfinite((t->>'vigente_hasta')::timestamptz)
 OR (p_registro AND (t->>'vigente_hasta')::timestamptz<=clock_timestamp())
 THEN RAISE EXCEPTION 'AUT49: PARO clave=vigencia actual=invalida esperado=intervalo_finito_futuro' USING ERRCODE='22023'; END IF;
 SELECT * INTO r FROM vec_autorizacion.version_rol WHERE version_rol_ref=t->>'version_rol_ref' FOR SHARE;
 IF NOT FOUND OR r.huella_sha256 IS DISTINCT FROM t->>'version_rol_sha256' OR r.documento->>'estado' IS DISTINCT FROM 'publicada'
 OR jsonb_typeof(r.documento->'concesiones') IS DISTINCT FROM 'array' OR jsonb_array_length(r.documento->'concesiones')<1
 THEN RAISE EXCEPTION 'AUT49: PARO clave=version_rol actual=divergente esperado=publicada_huella_exacta' USING ERRCODE='P0V01'; END IF;
 SELECT x.* INTO c FROM vec_autorizacion.control_vigencia_version_rol_actual ca JOIN vec_autorizacion.control_vigencia_version_rol x USING(version_rol_ref,revision) WHERE ca.version_rol_ref=r.version_rol_ref FOR SHARE OF ca;
 IF NOT FOUND OR c.estado IS DISTINCT FROM 'habilitada' OR c.revision::text IS DISTINCT FROM t->>'control_revision' OR c.huella_sha256 IS DISTINCT FROM t->>'control_sha256'
 THEN RAISE EXCEPTION 'AUT49: PARO clave=control actual=divergente esperado=habilitado_revision_exacta' USING ERRCODE='P0V01'; END IF;
 IF r.rol_id IN('administracion_perfiles','operador_plataforma')
 OR EXISTS(SELECT 1 FROM vec_autorizacion.rol_sensible_exacto WHERE version_rol_ref=r.version_rol_ref)
 OR EXISTS(SELECT 1 FROM vec_autorizacion.rol_sensible_exacto s JOIN vec_autorizacion.version_rol v USING(version_rol_ref) WHERE v.rol_id=r.rol_id)
 OR EXISTS(SELECT 1 FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 f JOIN vec_autorizacion.version_rol v USING(version_rol_ref) WHERE v.rol_id=r.rol_id)
 OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_externa e JOIN vec_autorizacion.version_rol v USING(version_rol_ref) WHERE v.rol_id=r.rol_id)
 -- Roles de aspirantes y de usuarios externos: nunca se reparten a personal interno.
 OR r.rol_id ~ '(^candidato_|extern)'
 -- Intervención y fiscalización siguen el circuito con doble control.
 OR r.rol_id ~ '^intervencion' OR r.documento->>'nombre' ~* '(fiscaliz|intervenc)'
 -- Ninguna versión del mismo rol puede figurar ya con otra clase.
 OR EXISTS(SELECT 1 FROM vec_autorizacion.rol_administrable_exacto_v1 a JOIN vec_autorizacion.version_rol v USING(version_rol_ref) WHERE v.rol_id=r.rol_id AND a.clase<>'ordinario')
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(r.documento->'concesiones') x
  WHERE x->>'modulo_id' IN('administracion','intervencion','aspirantes') OR x->>'accion' LIKE 'administracion.%' OR x->>'accion' LIKE '%fiscalizacion%'
  OR EXISTS(SELECT 1 FROM jsonb_array_elements_text(CASE WHEN jsonb_typeof(x->'finalidades')='array' THEN x->'finalidades' ELSE '[]'::jsonb END) fi WHERE fi LIKE '%fiscaliz%'))
 THEN RAISE EXCEPTION 'AUT49: PARO clave=clase actual=no_ordinaria esperado=rol_ordinario_interno' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('version_rol_ref',r.version_rol_ref,'version_rol_sha256',r.huella_sha256);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.validar_perfil_asignable_admin_v1(jsonb,boolean) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.aplicar_perfiles_asignables_admin_v1(plan_canonico text,sha_aprobado text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' AS $f$
DECLARE cfg vec_autorizacion.config_perfiles_asignables_admin_v1;p jsonb;sha text;t jsonb;v jsonb;lista jsonb:='[]';previo record;lista_sha text;
 e jsonb;aud record;recibo jsonb;instante timestamptz;
BEGIN
 cfg:=vec_autorizacion.exigir_operador_perfiles_asignables_admin_v1();
 IF plan_canonico IS NULL OR octet_length(plan_canonico) NOT BETWEEN 1 AND 65536 OR sha_aprobado IS DISTINCT FROM cfg.plan_sha256 THEN RAISE EXCEPTION 'AUT49: PARO clave=plan actual=divergente esperado=plan_privado_aprobado' USING ERRCODE='42501'; END IF;
 sha:=encode(sha256(convert_to(plan_canonico,'UTF8')),'hex');
 IF sha IS DISTINCT FROM cfg.plan_sha256 THEN RAISE EXCEPTION 'AUT49: PARO clave=plan_sha actual=divergente esperado=aprobado' USING ERRCODE='42501'; END IF;
 p:=plan_canonico::jsonb;
 -- El texto aprobado debe ser la forma canónica de jsonb: sin claves repetidas ni
 -- variantes de formato que hagan leer al aprobador un valor distinto del ejecutado.
 IF plan_canonico IS DISTINCT FROM p::text THEN RAISE EXCEPTION 'AUT49: PARO clave=plan actual=no_canonico esperado=jsonb_text' USING ERRCODE='22023'; END IF;
 IF jsonb_typeof(p) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(p))<>5 OR NOT p ?& ARRAY['esquema','operacion_ref','preparado_en','caduca_en','perfiles']
 OR p->>'esquema' IS DISTINCT FROM 'vec.admin.perfiles-asignables.plan.v1' OR jsonb_typeof(p->'operacion_ref') IS DISTINCT FROM 'string' OR p->>'operacion_ref' !~ '^rpa_[A-Za-z0-9_-]{22,124}$'
 OR jsonb_typeof(p->'perfiles') IS DISTINCT FROM 'array' OR jsonb_array_length(p->'perfiles') NOT BETWEEN 1 AND 32
 OR (SELECT count(DISTINCT x->>'version_rol_ref') FROM jsonb_array_elements(p->'perfiles') x)<>jsonb_array_length(p->'perfiles')
 OR jsonb_typeof(p->'preparado_en') IS DISTINCT FROM 'string' OR jsonb_typeof(p->'caduca_en') IS DISTINCT FROM 'string'
 OR p->>'preparado_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$' OR p->>'caduca_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$'
 OR (p->>'preparado_en')::timestamptz>clock_timestamp() OR (p->>'caduca_en')::timestamptz<=(p->>'preparado_en')::timestamptz
 OR (p->>'caduca_en')::timestamptz>(p->>'preparado_en')::timestamptz+interval '1 day'
 THEN RAISE EXCEPTION 'AUT49: PARO clave=plan actual=invalido esperado=plan_cerrado_1_32' USING ERRCODE='22023'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO previo FROM vec_autorizacion.registro_perfiles_asignables_admin_v1 WHERE operacion_ref=p->>'operacion_ref' FOR SHARE;
 IF FOUND THEN
  -- Replay: mismo plan exacto; el recibo original sólo se devuelve si los
  -- perfiles registrados siguen ahí con su huella y su versión publicada.
  IF previo.plan IS DISTINCT FROM convert_to(plan_canonico,'UTF8') OR previo.plan_sha256 IS DISTINCT FROM sha
  THEN RAISE EXCEPTION 'AUT49: PARO clave=replay actual=material_distinto esperado=plan_original' USING ERRCODE='23505'; END IF;
  IF EXISTS(SELECT 1 FROM jsonb_array_elements(previo.recibo->'perfiles') x LEFT JOIN vec_autorizacion.rol_administrable_exacto_v1 a ON a.version_rol_ref=x->>'version_rol_ref'
   LEFT JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref
   WHERE a.version_rol_ref IS NULL OR a.clase<>'ordinario' OR a.huella_sha256 IS DISTINCT FROM x->>'version_rol_sha256' OR r.huella_sha256 IS DISTINCT FROM a.huella_sha256)
  THEN RAISE EXCEPTION 'AUT49: PARO clave=replay_perfiles actual=divergente esperado=registro_original' USING ERRCODE='P0V01'; END IF;
  RETURN jsonb_build_object('recibo',previo.recibo,'replay',true);
 END IF;
 IF (p->>'caduca_en')::timestamptz<=clock_timestamp() THEN RAISE EXCEPTION 'AUT49: PARO clave=plan_caducado actual=caducado esperado=vigente' USING ERRCODE='42501'; END IF;
 FOR t IN SELECT value FROM jsonb_array_elements(p->'perfiles') ORDER BY value->>'version_rol_ref' LOOP
  v:=vec_autorizacion.validar_perfil_asignable_admin_v1(t,true);
  IF EXISTS(SELECT 1 FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=t->>'version_rol_ref') THEN RAISE EXCEPTION 'AUT49: PARO clave=ya_registrado actual=presente esperado=ausente' USING ERRCODE='23505'; END IF;
  lista:=lista||jsonb_build_array(v);
 END LOOP;
 lista_sha:=encode(sha256(convert_to(lista::text,'UTF8')),'hex');
 e:=jsonb_build_object('tipo_registro','perfiles_asignables_admin','evento_ref','evento_'||substr(sha,1,32),'operador_login',session_user::text,'plan_sha256',sha,
  'operacion_ref',p->>'operacion_ref','perfiles_sha256',lista_sha,'perfiles_numero',jsonb_array_length(lista)::text,'aprobacion_sha256',cfg.aprobacion_sha256,
  'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','perfiles_asignables_admin','correlacion_ref','correlacion_'||substr(sha,33,32));
 SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_perfiles_asignables_admin_v1(e);
 FOR t IN SELECT value FROM jsonb_array_elements(p->'perfiles') ORDER BY value->>'version_rol_ref' LOOP
  INSERT INTO vec_autorizacion.rol_administrable_exacto_v1(version_rol_ref,clase,huella_sha256,vigente_desde,vigente_hasta,unidad_requerida,audiencia_administrativa,ambitos_fijos,duracion_propuesta)
  VALUES(t->>'version_rol_ref','ordinario',t->>'version_rol_sha256',(t->>'vigente_desde')::timestamptz,(t->>'vigente_hasta')::timestamptz,(t->>'unidad_requerida')::boolean,
   'vec_autorizacion.administracion_perfiles.lote_ordinario.v1',t->'ambitos_fijos',make_interval(secs=>(t->>'duracion_propuesta_segundos')::double precision));
 END LOOP;
 recibo:=jsonb_build_object('esquema','vec.admin.perfiles-asignables.recibo.v1','operacion_ref',p->>'operacion_ref','plan_sha256',sha,'aprobacion_ref',cfg.aprobacion_ref,'aprobacion_sha256',cfg.aprobacion_sha256,'perfiles',lista,'perfiles_sha256',lista_sha,
  'auditoria_ref',aud.auditoria_ref,'auditoria_secuencia',aud.secuencia,'auditoria_huella_sha256',aud.huella_sha256,'confirmado_en',aud.registrada_en);
 INSERT INTO vec_autorizacion.registro_perfiles_asignables_admin_v1 VALUES(p->>'operacion_ref',sha,convert_to(plan_canonico,'UTF8'),session_user,aud.auditoria_ref,recibo,aud.registrada_en);
 -- Revalidación final bajo la misma transacción: configuración aún vigente.
 PERFORM vec_autorizacion.exigir_operador_perfiles_asignables_admin_v1();
 IF clock_timestamp()>=(p->>'caduca_en')::timestamptz THEN RAISE EXCEPTION 'AUT49: PARO clave=vigencia_final actual=caducada esperado=plan_vigente' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('recibo',recibo,'replay',false);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.aplicar_perfiles_asignables_admin_v1(text,text) FROM PUBLIC;

-- Fachada única del operador. Todo intento, también el replay o el rechazo,
-- deja un registro común. El efecto y su intento permitido comparten subbloque;
-- si algo falla se revierten ambos y sólo queda la negativa gestionada.
CREATE FUNCTION vec_autorizacion.registrar_perfiles_asignables_admin_v1(plan_canonico text,sha_aprobado text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='5s' SET statement_timeout='30s' AS $f$
DECLARE respuesta jsonb;estado text:='permitido';codigo text;motivo text;aud record;sol text;evento text;corr text;solsha text;
BEGIN
 sol:='solicitud_perfiles_asignables:'||replace(gen_random_uuid()::text,'-','');evento:='evento_'||replace(gen_random_uuid()::text,'-','');corr:='correlacion_'||replace(gen_random_uuid()::text,'-','');
 solsha:=encode(sha256(convert_to(jsonb_build_object('plan',plan_canonico,'sha_aprobado',sha_aprobado)::text,'UTF8')),'hex');
 BEGIN
  respuesta:=vec_autorizacion.aplicar_perfiles_asignables_admin_v1(plan_canonico,sha_aprobado);
  motivo:=CASE WHEN (respuesta->>'replay')::boolean THEN 'perfiles_asignables_replay' ELSE 'perfiles_asignables_registrado' END;
  SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_intento_perfiles_asignables_admin_v1(jsonb_build_object('tipo_registro','intento_perfiles_asignables_admin','evento_ref',evento,'operador_login',session_user::text,'solicitud_sha256',solsha,'accion','registrar_perfiles_asignables_admin_v1','recurso_ref',sol,'resultado',estado,'motivo_ref',motivo,'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','perfiles_asignables_admin','correlacion_ref',corr));
  PERFORM vec_autorizacion.exigir_operador_perfiles_asignables_admin_v1();
 EXCEPTION WHEN OTHERS THEN respuesta:=NULL;codigo:=SQLSTATE;estado:=CASE WHEN codigo IN('42501','22023','22P02','22007','22008','P0V01','23505','25000') THEN 'denegado' ELSE 'error' END;
  motivo:=CASE WHEN estado='denegado' THEN 'perfiles_asignables_denegado' ELSE 'perfiles_asignables_error' END;
  codigo:=CASE WHEN estado='denegado' THEN 'perfiles_asignables_rechazado' ELSE 'perfiles_asignables_no_disponible' END;
 END;
 IF estado<>'permitido' THEN
  SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_intento_perfiles_asignables_admin_v1(jsonb_build_object('tipo_registro','intento_perfiles_asignables_admin','evento_ref',evento,'operador_login',session_user::text,'solicitud_sha256',solsha,'accion','registrar_perfiles_asignables_admin_v1','recurso_ref',sol,'resultado',estado,'motivo_ref',motivo,'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','perfiles_asignables_admin','correlacion_ref',corr));
 END IF;
 RETURN jsonb_build_object('estado',estado,'codigo',codigo,'recibo',respuesta->'recibo','replay',COALESCE((respuesta->>'replay')::boolean,false),
  'auditoria_intento',jsonb_build_object('auditoria_ref',aud.auditoria_ref,'secuencia',aud.secuencia,'huella_sha256',aud.huella_sha256,'correlacion_ref',aud.correlacion_ref,'registrada_en',aud.registrada_en));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.registrar_perfiles_asignables_admin_v1(text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_perfiles_asignables_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion.registrar_perfiles_asignables_admin_v1(text,text) TO vec_admin_perfiles_asignables_ejecutor;
RESET ROLE;

DO $acl$
DECLARE g oid:=to_regrole('vec_admin_perfiles_asignables_ejecutor');
BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid IN(to_regprocedure('vec_autorizacion.exigir_operador_perfiles_asignables_admin_v1()'),to_regprocedure('vec_autorizacion.validar_perfil_asignable_admin_v1(jsonb,boolean)'),to_regprocedure('vec_autorizacion.aplicar_perfiles_asignables_admin_v1(text,text)'))
  AND a.grantee<>p.proowner)
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=to_regprocedure('vec_autorizacion.registrar_perfiles_asignables_admin_v1(text,text)') AND a.grantee NOT IN(p.proowner,g))
 OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(COALESCE(c.relacl,acldefault('r',c.relowner))) a
  WHERE c.oid IN('vec_autorizacion.config_perfiles_asignables_admin_v1'::regclass,'vec_autorizacion.registro_perfiles_asignables_admin_v1'::regclass) AND a.grantee<>c.relowner)
 OR EXISTS(SELECT 1 FROM pg_proc WHERE oid IN(to_regprocedure('vec_autorizacion.exigir_operador_perfiles_asignables_admin_v1()'),to_regprocedure('vec_autorizacion.validar_perfil_asignable_admin_v1(jsonb,boolean)'),to_regprocedure('vec_autorizacion.aplicar_perfiles_asignables_admin_v1(text,text)'),to_regprocedure('vec_autorizacion.registrar_perfiles_asignables_admin_v1(text,text)'))
  AND (proowner<>'vec_autorizacion_propietario'::regrole OR NOT prosecdef))
 THEN RAISE EXCEPTION 'AUT49: PARO clave=ACL actual=ampliada esperado=propietario_y_grupo_operador' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
