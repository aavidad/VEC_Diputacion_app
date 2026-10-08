\set ON_ERROR_STOP on
-- AD222: registro técnico preacreditación de START/session en la cadena AD207.
-- No crea LOGIN, no asigna perfiles y no concede autorizaciones V3.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000222',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE actual text; columnas integer;acl_esquema text;acl_asiento text;
 acl_reserva text;acl_presentacion text;
BEGIN
 IF pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3') IS NULL
 OR pg_catalog.to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.auditoria_consumo_v3') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5()') IS NULL
 THEN RAISE EXCEPTION 'AD222: PARO clave=objetos_base actual=ausente esperado=POST221'
 USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_constraintdef(c.oid,false),'UTF8')),'hex')
 INTO actual FROM pg_catalog.pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::pg_catalog.regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF actual IS DISTINCT FROM '541a3ca7f64d0c406a52398d565c84890e8df5695e5cf4907b972ee4db53fec8'
 THEN RAISE EXCEPTION 'AD222: PARO clave=CHECK_POST221 actual=% esperado=541a3ca7f64d0c406a52398d565c84890e8df5695e5cf4907b972ee4db53fec8',
      COALESCE(actual,'ausente') USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.count(*) INTO columnas FROM pg_catalog.pg_attribute a
 WHERE a.attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::pg_catalog.regclass
 AND a.attnum>0 AND NOT a.attisdropped;
 IF columnas IS DISTINCT FROM 69
 THEN RAISE EXCEPTION 'AD222: PARO clave=columnas_POST221 actual=% esperado=69',
      COALESCE(columnas::text,'ausente') USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
   COALESCE(n.nspacl::text,'<NULL>'),'UTF8')),'hex') INTO acl_esquema
 FROM pg_catalog.pg_namespace n WHERE n.nspname='vec_autorizacion_atestada_v3'
 AND n.nspowner='vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole;
 IF acl_esquema IS DISTINCT FROM 'e1b03ec0d3c80ef6c28015d38f6bd52108ed1120154ffca674eb3d201c78fa8b'
 THEN RAISE EXCEPTION 'AD222: PARO clave=ACL_esquema actual=% esperado=e1b03ec0d3c80ef6c28015d38f6bd52108ed1120154ffca674eb3d201c78fa8b',
   COALESCE(acl_esquema,'ausente') USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
   COALESCE(c.relacl::text,'<NULL>'),'UTF8')),'hex') INTO acl_asiento
 FROM pg_catalog.pg_class c
 WHERE c.oid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::pg_catalog.regclass
 AND c.relowner='vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole
 AND c.relrowsecurity AND c.relforcerowsecurity;
 IF acl_asiento IS DISTINCT FROM '8ab60881072b62721284dd0081f2a14de0bbf755fa88c55ddd64a319bb23a734'
 THEN RAISE EXCEPTION 'AD222: PARO clave=ACL_asiento actual=% esperado=8ab60881072b62721284dd0081f2a14de0bbf755fa88c55ddd64a319bb23a734',
   COALESCE(acl_asiento,'ausente') USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
   COALESCE(p.proacl::text,'<NULL>'),'UTF8')),'hex') INTO acl_reserva
 FROM pg_catalog.pg_proc p
 WHERE p.oid='vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5()'::pg_catalog.regprocedure
 AND p.proowner='vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole
 AND NOT p.prosecdef AND p.provolatile='v';
 IF acl_reserva IS DISTINCT FROM '656e51356341f2453b258181ad199b4ef82a0b5402ea469402eeccb8c5f78698'
 THEN RAISE EXCEPTION 'AD222: PARO clave=ACL_reserva_AD207 actual=% esperado=656e51356341f2453b258181ad199b4ef82a0b5402ea469402eeccb8c5f78698',
   COALESCE(acl_reserva,'ausente') USING ERRCODE='55000'; END IF;
 IF (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc p
   WHERE p.pronamespace='vec_autorizacion_atestada_v3'::pg_catalog.regnamespace
   AND p.proname='registrar_auditoria_presentacion_certificado_v1')<>1
 THEN RAISE EXCEPTION 'AD222: PARO clave=funcion_AD221 actual=cardinalidad_incompatible esperado=1'
 USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
   COALESCE(p.proacl::text,'<NULL>'),'UTF8')),'hex') INTO acl_presentacion
 FROM pg_catalog.pg_proc p
 WHERE p.pronamespace='vec_autorizacion_atestada_v3'::pg_catalog.regnamespace
 AND p.proname='registrar_auditoria_presentacion_certificado_v1'
 AND p.proowner='vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole
 AND p.prosecdef AND p.provolatile='v';
 IF acl_presentacion IS DISTINCT FROM 'fa29f600dd3a17a1f35c11175417ab51ed164154b05709e15e8908561884a93a'
 THEN RAISE EXCEPTION 'AD222: PARO clave=ACL_presentacion_AD221 actual=% esperado=fa29f600dd3a17a1f35c11175417ab51ed164154b05709e15e8908561884a93a',
   COALESCE(acl_presentacion,'ausente') USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR pg_catalog.pg_is_in_recovery()
 OR pg_catalog.to_regrole('vec_identidad_frontera_tecnica_ejecutor') IS NOT NULL
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.config_frontera_identidad_tecnica_v1') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.acreditar_frontera_identidad_tecnica_v1()') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_frontera_identidad_tecnica_v1(jsonb)') IS NOT NULL
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.auditoria_presentacion_certificado_v1') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5()') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.encuadrar_mac(text)') IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_trigger t
   WHERE t.tgrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::pg_catalog.regclass
   AND t.tgname='encolar_sellado_ad207' AND t.tgenabled='O')
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_attribute a
   WHERE a.attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::pg_catalog.regclass
   AND a.attname='pre_identidad_tecnica_detalle' AND NOT a.attisdropped)
 THEN RAISE EXCEPTION 'AD222: PARO clave=preimagen actual=incompatible esperado=POST221_PG18_AD207_sin222'
   USING ERRCODE='55000'; END IF;
END $pre$;

CREATE ROLE vec_identidad_frontera_tecnica_ejecutor
 NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
-- El nombre de la base cambia en clones; el único DDL dinámico adicional es
-- este GRANT al nombre actual. No instala LOGIN ni configura una vía favorable.
DO $connect$ BEGIN EXECUTE pg_catalog.format(
 'GRANT CONNECT ON DATABASE %I TO vec_identidad_frontera_tecnica_ejecutor',
 pg_catalog.current_database()); END $connect$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD COLUMN pre_identidad_tecnica_detalle jsonb;

-- La única reconstrucción dinámica necesaria amplía el CHECK vigente.
-- Su preimagen POST221 está anclada arriba; ningún predicado histórico cambia.
DO $familia$
DECLARE anterior text;nueva text;nulas text;
 propias constant text[]:=ARRAY[
  'auditoria_ref','secuencia','anterior_sha256','huella_sha256',
  'registrada_en','tipo_registro','evento_ref','evento_material_sha256',
  'operador_login','accion','modulo_id','recurso_ref','finalidad_ref',
  'resultado','motivo_ref','proceso','canal','correlacion_ref',
  'pre_identidad_tecnica_detalle'];
BEGIN
 SELECT pg_catalog.pg_get_constraintdef(c.oid,false) INTO STRICT anterior
 FROM pg_catalog.pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::pg_catalog.regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF pg_catalog.left(anterior,7)<>'CHECK (' OR pg_catalog.right(anterior,1)<>')'
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(anterior,'UTF8')),'hex')
 IS DISTINCT FROM '541a3ca7f64d0c406a52398d565c84890e8df5695e5cf4907b972ee4db53fec8'
 THEN RAISE EXCEPTION 'AD222: PARO clave=CHECK_POST221_despues_LOCK actual=incompatible esperado=541a3ca7f64d0c406a52398d565c84890e8df5695e5cf4907b972ee4db53fec8'
 USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.string_agg(pg_catalog.format('%I IS NULL',a.attname),' AND ' ORDER BY a.attnum)
 INTO STRICT nulas FROM pg_catalog.pg_attribute a
 WHERE a.attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::pg_catalog.regclass
 AND a.attnum>0 AND NOT a.attisdropped AND NOT a.attname=ANY(propias);
 IF nulas IS NULL OR nulas NOT LIKE '%decision_ref IS NULL%'
 OR nulas NOT LIKE '%actor_ref IS NULL%' OR nulas NOT LIKE '%perfil_activo_ref IS NULL%'
 OR nulas NOT LIKE '%presentacion_certificado_detalle IS NULL%'
 THEN RAISE EXCEPTION 'AD222: PARO clave=columnas_ajenas actual=incompatible esperado=decision_actor_perfil_presentacion_NULL'
 USING ERRCODE='55000'; END IF;
 nueva:='CHECK ((pre_identidad_tecnica_detalle IS NULL AND ('||
   pg_catalog.substr(anterior,8,pg_catalog.length(anterior)-8)||
   ')) OR (tipo_registro IS NOT DISTINCT FROM ''pre_identidad_tecnica_v1'' AND '||
   nulas||' AND pre_identidad_tecnica_detalle IS NOT NULL'||
   ' AND evento_ref IS NOT NULL AND evento_material_sha256 IS NOT NULL'||
   ' AND operador_login IS NOT NULL AND accion IS NOT NULL'||
   ' AND modulo_id IS NOT DISTINCT FROM ''identidad'''||
   ' AND recurso_ref IS NOT NULL AND finalidad_ref IS NOT DISTINCT FROM ''preacreditacion_identidad'''||
   ' AND resultado IS NOT NULL AND motivo_ref IS NOT NULL'||
   ' AND proceso IS NOT NULL AND canal IS NOT DISTINCT FROM ''identidad_http_interno_preacreditacion'''||
   ' AND correlacion_ref IS NOT NULL))';
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||nueva;
END $familia$;

ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD CONSTRAINT auditoria_pre_identidad_tecnica_formato_v1 CHECK (
 tipo_registro <> 'pre_identidad_tecnica_v1' OR COALESCE((
   evento_ref ~ '^evento_[0-9a-f]{32}$'
   AND evento_material_sha256 ~ '^[0-9a-f]{64}$'
   AND correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'
   AND operador_login IS NOT NULL AND accion='registrar_pre_identidad_tecnica_v1'
   AND modulo_id='identidad' AND finalidad_ref='preacreditacion_identidad'
   AND canal='identidad_http_interno_preacreditacion'
   AND proceso ~ '^[a-z][a-z0-9._-]{1,79}$'
   AND recurso_ref='solicitud_sesion:'||pg_catalog.substr(pg_catalog.encode(
     pg_catalog.sha256(pg_catalog.convert_to(
       E'vec.identidad.preacreditacion.solicitud.v1\n'||correlacion_ref,'UTF8')),'hex'),1,32)
   AND pg_catalog.jsonb_typeof(pre_identidad_tecnica_detalle)='object'
   AND pre_identidad_tecnica_detalle ?& ARRAY['fase','metodo_esperado','ruta','superficie']
   AND pre_identidad_tecnica_detalle-ARRAY['fase','metodo_esperado','ruta','superficie']='{}'::jsonb
   AND pg_catalog.jsonb_typeof(pre_identidad_tecnica_detalle->'fase')='string'
   AND pg_catalog.jsonb_typeof(pre_identidad_tecnica_detalle->'metodo_esperado')='string'
   AND pg_catalog.jsonb_typeof(pre_identidad_tecnica_detalle->'ruta')='string'
   AND pg_catalog.jsonb_typeof(pre_identidad_tecnica_detalle->'superficie')='string'
   AND pre_identidad_tecnica_detalle->>'fase'='preacreditacion'
   AND pre_identidad_tecnica_detalle->>'superficie'='interna_corporativa'
   AND ((pre_identidad_tecnica_detalle->>'ruta'='/api/vec/session'
         AND pre_identidad_tecnica_detalle->>'metodo_esperado'='GET')
     OR (pre_identidad_tecnica_detalle->>'ruta'='/api/vec/session/start'
         AND pre_identidad_tecnica_detalle->>'metodo_esperado'='POST'))
   AND ((resultado='denegado' AND motivo_ref IN
      ('certificado_requerido','autenticacion_requerida','acceso_denegado',
       'metodo_no_permitido','recurso_no_encontrado','solicitud_invalida'))
     OR (resultado='error' AND motivo_ref IN
      ('servicio_no_disponible','respuesta_incompatible')))
 ),false));

CREATE TABLE vec_autorizacion_atestada_v3.config_frontera_identidad_tecnica_v1(
 login_nombre name PRIMARY KEY,
 proceso text NOT NULL CHECK(proceso ~ '^[a-z][a-z0-9._-]{1,79}$'),
 canal text NOT NULL CHECK(canal='identidad_http_interno_preacreditacion'),
 superficie text NOT NULL CHECK(superficie='interna_corporativa'),
 vigente_desde timestamptz(6) NOT NULL,
 vigente_hasta timestamptz(6) NOT NULL,
 CHECK(pg_catalog.isfinite(vigente_desde) AND pg_catalog.isfinite(vigente_hasta)
   AND vigente_hasta>vigente_desde)
);
ALTER TABLE vec_autorizacion_atestada_v3.config_frontera_identidad_tecnica_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion_atestada_v3.config_frontera_identidad_tecnica_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion_atestada_v3.config_frontera_identidad_tecnica_v1
 FOR ALL TO vec_autorizacion_atestada_v3_propietario
 USING (current_user='vec_autorizacion_atestada_v3_propietario')
 WITH CHECK (current_user='vec_autorizacion_atestada_v3_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE
 ON vec_autorizacion_atestada_v3.config_frontera_identidad_tecnica_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE
 ON vec_autorizacion_atestada_v3.config_frontera_identidad_tecnica_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado();
REVOKE ALL ON TABLE vec_autorizacion_atestada_v3.config_frontera_identidad_tecnica_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion_atestada_v3.config_frontera_identidad_tecnica_v1 FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.codigos_frontera_identidad_tecnica_v1()
RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $funcion$
 SELECT $datos$[
  {"motivo_ref":"certificado_requerido","resultado":"denegado"},
  {"motivo_ref":"autenticacion_requerida","resultado":"denegado"},
  {"motivo_ref":"acceso_denegado","resultado":"denegado"},
  {"motivo_ref":"metodo_no_permitido","resultado":"denegado"},
  {"motivo_ref":"recurso_no_encontrado","resultado":"denegado"},
  {"motivo_ref":"solicitud_invalida","resultado":"denegado"},
  {"motivo_ref":"servicio_no_disponible","resultado":"error"},
  {"motivo_ref":"respuesta_incompatible","resultado":"error"}
 ]$datos$::jsonb
$funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.codigos_frontera_identidad_tecnica_v1() FROM PUBLIC;

-- Sólo session_user acreditado por DBA entra. El grupo NOLOGIN tiene como
-- únicas ACL CONNECT en esta base, USAGE del esquema y EXEC de dos fachadas.
CREATE FUNCTION vec_autorizacion_atestada_v3.exigir_frontera_identidad_tecnica_v1()
RETURNS vec_autorizacion_atestada_v3.config_frontera_identidad_tecnica_v1
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on
SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE l record;g record;c vec_autorizacion_atestada_v3.config_frontera_identidad_tecnica_v1;
 ns oid;db oid;f1 oid;f2 oid;
BEGIN
 SELECT * INTO l FROM pg_catalog.pg_roles WHERE rolname=session_user;
 SELECT * INTO g FROM pg_catalog.pg_roles WHERE rolname='vec_identidad_frontera_tecnica_ejecutor';
 ns:=pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3');
 SELECT oid INTO db FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database();
 f1:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.acreditar_frontera_identidad_tecnica_v1()');
 f2:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_frontera_identidad_tecnica_v1(jsonb)');
 IF pg_catalog.current_setting('role')<>'none'
 OR l.oid IS NULL OR g.oid IS NULL
 OR NOT l.rolcanlogin OR NOT l.rolinherit OR l.rolsuper OR l.rolcreaterole
 OR l.rolcreatedb OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
 OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreaterole
 OR g.rolcreatedb OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members WHERE member=l.oid)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members
   WHERE member=l.oid AND roleid=g.oid AND inherit_option
   AND NOT set_option AND NOT admin_option)
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members WHERE roleid=g.oid)<>1
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=g.oid)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting WHERE setrole IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_shdepend
   WHERE refclassid='pg_catalog.pg_authid'::pg_catalog.regclass AND refobjid=l.oid)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_shdepend
   WHERE refclassid='pg_catalog.pg_authid'::pg_catalog.regclass AND refobjid=g.oid
   AND NOT ((dbid=db AND classid='pg_catalog.pg_namespace'::pg_catalog.regclass
             AND objid=ns AND deptype='a')
     OR (dbid=db AND classid='pg_catalog.pg_proc'::pg_catalog.regclass
             AND objid IN(f1,f2) AND deptype='a')
     OR (dbid=0 AND classid='pg_catalog.pg_database'::pg_catalog.regclass
             AND objid=db AND deptype='a')))
 OR NOT COALESCE((SELECT pg_catalog.count(*)=1
    AND pg_catalog.bool_and(a.privilege_type='CONNECT' AND NOT a.is_grantable)
    FROM pg_catalog.pg_database d
    CROSS JOIN LATERAL pg_catalog.aclexplode(
      COALESCE(d.datacl,pg_catalog.acldefault('d',d.datdba))) a
    WHERE d.oid=db AND a.grantee=g.oid),false)
 OR NOT COALESCE((SELECT pg_catalog.count(*)=1
    AND pg_catalog.bool_and(a.privilege_type='USAGE' AND NOT a.is_grantable)
    FROM pg_catalog.pg_namespace n
    CROSS JOIN LATERAL pg_catalog.aclexplode(
      COALESCE(n.nspacl,pg_catalog.acldefault('n',n.nspowner))) a
    WHERE n.oid=ns AND a.grantee=g.oid),false)
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc p
    WHERE p.oid IN(f1,f2)
    AND p.proowner='vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole
    AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
    AND EXISTS(SELECT 1 FROM pg_catalog.aclexplode(
      COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
      WHERE a.grantee=g.oid AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
    AND NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(
      COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
      WHERE a.grantee NOT IN(p.proowner,g.oid)
         OR a.privilege_type<>'EXECUTE' OR a.is_grantable))<>2
 OR pg_catalog.has_schema_privilege(l.oid,'vec_autorizacion_atestada_v3','CREATE')
 OR pg_catalog.has_database_privilege(l.oid,pg_catalog.current_database(),'CREATE,TEMP')
 THEN RAISE EXCEPTION 'AD222: operador_no_acreditado' USING ERRCODE='42501'; END IF;
 SELECT * INTO c FROM vec_autorizacion_atestada_v3.config_frontera_identidad_tecnica_v1
 WHERE login_nombre=session_user FOR SHARE;
 IF NOT FOUND OR pg_catalog.clock_timestamp()<c.vigente_desde
 OR pg_catalog.clock_timestamp()>=c.vigente_hasta
 OR c.superficie IS DISTINCT FROM 'interna_corporativa'
 OR c.canal IS DISTINCT FROM 'identidad_http_interno_preacreditacion'
 THEN RAISE EXCEPTION 'AD222: configuracion_no_vigente' USING ERRCODE='42501'; END IF;
 RETURN c;
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.exigir_frontera_identidad_tecnica_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.acreditar_frontera_identidad_tecnica_v1()
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on
SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE c vec_autorizacion_atestada_v3.config_frontera_identidad_tecnica_v1;
BEGIN
 c:=vec_autorizacion_atestada_v3.exigir_frontera_identidad_tecnica_v1();
 RETURN pg_catalog.jsonb_build_object(
  'operador_login',session_user::text,'proceso',c.proceso,'canal',c.canal,
  'superficie',c.superficie,
  'rutas',pg_catalog.jsonb_build_array(
   pg_catalog.jsonb_build_object('metodo_esperado','GET','ruta','/api/vec/session'),
   pg_catalog.jsonb_build_object('metodo_esperado','POST','ruta','/api/vec/session/start')),
  'codigos',vec_autorizacion_atestada_v3.codigos_frontera_identidad_tecnica_v1());
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.acreditar_frontera_identidad_tecnica_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_frontera_identidad_tecnica_v1(p_evento jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,material_sha256 text,
 correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE
 c vec_autorizacion_atestada_v3.config_frontera_identidad_tecnica_v1;
 v_orden constant text[]:=ARRAY[
  'tipo_registro','evento_ref','operador_login','fase','metodo_esperado',
  'ruta','accion','recurso_ref','resultado','motivo_ref','proceso','canal',
  'superficie','finalidad_ref','correlacion_ref'];
 v_claves text[];v_clave text;v_material bytea;v_material_sha text;
 v_existente record;v_secuencia numeric;v_anterior text;v_instante timestamptz(6);
 v_ref text;v_huella text;v_detalle jsonb;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off'
 OR pg_catalog.current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD222: transaccion_incompatible' USING ERRCODE='25000'; END IF;
 c:=vec_autorizacion_atestada_v3.exigir_frontera_identidad_tecnica_v1();
 IF pg_catalog.jsonb_typeof(p_evento) IS DISTINCT FROM 'object'
 OR pg_catalog.octet_length(p_evento::text)>8192
 THEN RAISE EXCEPTION 'AD222: evento_invalido' USING ERRCODE='22023'; END IF;
 SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") INTO v_claves
 FROM pg_catalog.jsonb_object_keys(p_evento) k;
 IF v_claves IS DISTINCT FROM
   (SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C")
    FROM pg_catalog.unnest(v_orden) k)
 THEN RAISE EXCEPTION 'AD222: ABI_evento_incompatible' USING ERRCODE='22023'; END IF;
 FOREACH v_clave IN ARRAY v_orden LOOP
  IF pg_catalog.jsonb_typeof(p_evento->v_clave) IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(p_evento->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD222: valor_evento_invalido' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_evento->>'tipo_registro' IS DISTINCT FROM 'pre_identidad_tecnica_v1'
 OR p_evento->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR p_evento->>'operador_login' IS DISTINCT FROM session_user::text
 OR pg_catalog.octet_length(p_evento->>'operador_login')>63
 OR p_evento->>'fase' IS DISTINCT FROM 'preacreditacion'
 OR NOT ((p_evento->>'ruta'='/api/vec/session'
           AND p_evento->>'metodo_esperado'='GET')
      OR (p_evento->>'ruta'='/api/vec/session/start'
           AND p_evento->>'metodo_esperado'='POST'))
 OR p_evento->>'accion' IS DISTINCT FROM 'registrar_pre_identidad_tecnica_v1'
 OR p_evento->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 OR p_evento->>'recurso_ref' IS DISTINCT FROM 'solicitud_sesion:'||
   pg_catalog.substr(pg_catalog.encode(pg_catalog.sha256(
    pg_catalog.convert_to(E'vec.identidad.preacreditacion.solicitud.v1\n'||
       (p_evento->>'correlacion_ref'),'UTF8')),'hex'),1,32)
 OR p_evento->>'proceso' IS DISTINCT FROM c.proceso
 OR p_evento->>'canal' IS DISTINCT FROM c.canal
 OR p_evento->>'superficie' IS DISTINCT FROM c.superficie
 OR p_evento->>'finalidad_ref' IS DISTINCT FROM 'preacreditacion_identidad'
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(
    vec_autorizacion_atestada_v3.codigos_frontera_identidad_tecnica_v1()) x
   WHERE x->>'motivo_ref'=p_evento->>'motivo_ref'
     AND x->>'resultado'=p_evento->>'resultado')
 THEN RAISE EXCEPTION 'AD222: semantica_incompatible' USING ERRCODE='22023'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac(
   'vec.auditoria.pre-identidad-tecnica.v1');
 FOREACH v_clave IN ARRAY v_orden LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_evento->>v_clave);
 END LOOP;
 v_material_sha:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
   'vec_autorizacion_atestada_v3:evento-pre-identidad:'||(p_evento->>'evento_ref'),0));
 SELECT a.tipo_registro,a.auditoria_ref,a.secuencia,a.evento_material_sha256,
        a.correlacion_ref,a.registrada_en
 INTO v_existente FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
 WHERE a.evento_ref=p_evento->>'evento_ref';
 IF FOUND THEN
  IF v_existente.tipo_registro IS DISTINCT FROM 'pre_identidad_tecnica_v1'
  OR v_existente.evento_material_sha256 IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'AD222: replay_distinto' USING ERRCODE='23505'; END IF;
  PERFORM vec_autorizacion_atestada_v3.exigir_frontera_identidad_tecnica_v1();
  RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,
    v_existente.evento_material_sha256,v_existente.correlacion_ref,v_existente.registrada_en;
  RETURN;
 END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 IF v_secuencia>=9007199254740991::numeric
 THEN RAISE EXCEPTION 'AD222: secuencia_agotada' USING ERRCODE='22003'; END IF;
 PERFORM vec_autorizacion_atestada_v3.exigir_frontera_identidad_tecnica_v1();
 v_secuencia:=v_secuencia+1;v_instante:=pg_catalog.clock_timestamp();
 v_ref:='aud_v3_pit_'||pg_catalog.substr(p_evento->>'evento_ref',8,32);
 v_huella:=pg_catalog.encode(pg_catalog.sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac(
   'vec.auditoria.eslabon.pre-identidad-tecnica.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(
   v_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 v_detalle:=pg_catalog.jsonb_build_object(
   'fase',p_evento->>'fase','metodo_esperado',p_evento->>'metodo_esperado',
   'ruta',p_evento->>'ruta','superficie',p_evento->>'superficie');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,
  tipo_registro,evento_ref,evento_material_sha256,operador_login,
  pre_identidad_tecnica_detalle,accion,modulo_id,recurso_ref,finalidad_ref,
  resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,
  'pre_identidad_tecnica_v1',p_evento->>'evento_ref',v_material_sha,
  session_user::name,v_detalle,'registrar_pre_identidad_tecnica_v1','identidad',
  p_evento->>'recurso_ref','preacreditacion_identidad',
  p_evento->>'resultado',p_evento->>'motivo_ref',c.proceso,c.canal,
  p_evento->>'correlacion_ref');
 -- AD207 encola este mismo asiento. No se devuelve el eslabón aún sin sellar.
 RETURN QUERY SELECT v_ref,v_secuencia,v_material_sha,p_evento->>'correlacion_ref',v_instante;
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_frontera_identidad_tecnica_v1(jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_identidad_frontera_tecnica_ejecutor;
GRANT EXECUTE ON FUNCTION
 vec_autorizacion_atestada_v3.acreditar_frontera_identidad_tecnica_v1(),
 vec_autorizacion_atestada_v3.registrar_frontera_identidad_tecnica_v1(jsonb)
 TO vec_identidad_frontera_tecnica_ejecutor;
COMMIT;
