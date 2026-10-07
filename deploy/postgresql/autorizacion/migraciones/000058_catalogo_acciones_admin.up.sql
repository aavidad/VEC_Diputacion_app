\set ON_ERROR_STOP on
-- AUT58: admite una instantánea completa del catálogo de acciones sólo bajo
-- aprobación externa fijada por el DBA a un LOGIN técnico exclusivo. El plan
-- compromete paquete de fuentes, catálogo, clasificación y preimagen CAS.
-- Instalar esta migración no configura un operador ni publica una instantánea.
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
 OR pg_catalog.to_regclass('vec_autorizacion.version_rol') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.control_vigencia_version_rol_actual') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.perfil_fijo_categoria_nominal_v1') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.rol_sensible_exacto') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_catalogo_acciones_admin_v1(jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_catalogo_acciones_admin_v1(jsonb)') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.config_catalogo_acciones_admin_v1') IS NOT NULL
 OR pg_catalog.to_regrole('vec_admin_catalogo_acciones_ejecutor') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT58: preimagen o AD219 ausente' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE ROLE vec_admin_catalogo_acciones_ejecutor NOLOGIN NOINHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_admin_catalogo_acciones_lector NOLOGIN NOINHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOREPLICATION NOBYPASSRLS;
DO $db$ BEGIN
 EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO vec_admin_catalogo_acciones_ejecutor,vec_admin_catalogo_acciones_lector',pg_catalog.current_database());
END $db$;
SET LOCAL ROLE vec_autorizacion_propietario;

-- La fila la escribe el DBA después de comprobar la aprobación externa y el
-- paquete exacto de módulos. Puede revocarla o acortar su ventana. El LOGIN
-- runtime carece de permisos de tabla y no puede configurar su propio plan.
CREATE TABLE vec_autorizacion.config_catalogo_acciones_admin_v1(
 login_nombre name PRIMARY KEY,
 plan_sha256 text NOT NULL UNIQUE CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 aprobacion_ref text NOT NULL CHECK(aprobacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
 aprobacion_sha256 text NOT NULL CHECK(aprobacion_sha256 ~ '^[0-9a-f]{64}$'),
 aprobador_ref text NOT NULL CHECK(aprobador_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
 paquete_ref text NOT NULL CHECK(paquete_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
 paquete_version integer NOT NULL CHECK(paquete_version BETWEEN 1 AND 2147483647),
 paquete_sha256 text NOT NULL CHECK(paquete_sha256 ~ '^[0-9a-f]{64}$'),
 destino_ref text NOT NULL CHECK(destino_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
 entorno text NOT NULL CHECK(entorno IN ('desarrollo','produccion')),
 estado text NOT NULL CHECK(estado IN ('admitida','revocada')),
 vigente_desde timestamptz(6) NOT NULL,
 vigente_hasta timestamptz(6) NOT NULL,
 CHECK(pg_catalog.isfinite(vigente_desde) AND pg_catalog.isfinite(vigente_hasta)
  AND vigente_hasta>vigente_desde AND vigente_hasta-vigente_desde<=interval '1 day')
);

-- Sólo la cabeza muta mediante CAS; todas las versiones anteriores quedan
-- inmutables y recuperables por referencia, versión y huella exactas.
CREATE TABLE vec_autorizacion.cabeza_catalogo_acciones_admin_v1(
 catalogo_ref text PRIMARY KEY CHECK(catalogo_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
 version integer NOT NULL CHECK(version BETWEEN 1 AND 2147483647),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 operacion_ref text NOT NULL CHECK(operacion_ref ~ '^caa_[A-Za-z0-9_-]{22,123}$'),
 actualizada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_autorizacion.registro_catalogo_acciones_admin_v1(
 catalogo_ref text NOT NULL CHECK(catalogo_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
 version integer NOT NULL CHECK(version BETWEEN 1 AND 2147483647),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 canon bytea NOT NULL CHECK(pg_catalog.octet_length(canon) BETWEEN 1 AND 16777216
  AND pg_catalog.encode(pg_catalog.sha256(canon),'hex')=huella_sha256),
 paquete_ref text NOT NULL,paquete_version integer NOT NULL,paquete_sha256 text NOT NULL,
 paquete_canon bytea NOT NULL CHECK(pg_catalog.octet_length(paquete_canon) BETWEEN 1 AND 16777216
  AND pg_catalog.encode(pg_catalog.sha256(paquete_canon),'hex')=paquete_sha256),
 censo_sha256 text NOT NULL CHECK(censo_sha256 ~ '^[0-9a-f]{64}$'),
 operacion_ref text NOT NULL UNIQUE CHECK(operacion_ref ~ '^caa_[A-Za-z0-9_-]{22,123}$'),
 plan_sha256 text NOT NULL CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 aprobacion_ref text NOT NULL,aprobacion_sha256 text NOT NULL,aprobador_ref text NOT NULL,
 login_nombre name NOT NULL,auditoria_ref text NOT NULL,
 recibo jsonb NOT NULL CHECK(pg_catalog.jsonb_typeof(recibo)='object'),
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(catalogo_ref,version),UNIQUE(catalogo_ref,version,huella_sha256)
);
CREATE TABLE vec_autorizacion.operacion_catalogo_acciones_admin_v1(
 operacion_ref text PRIMARY KEY CHECK(operacion_ref ~ '^caa_[A-Za-z0-9_-]{22,123}$'),
 plan_sha256 text NOT NULL UNIQUE CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 catalogo_ref text NOT NULL,version integer NOT NULL,huella_sha256 text NOT NULL,
 auditoria_ref text NOT NULL,recibo jsonb NOT NULL CHECK(pg_catalog.jsonb_typeof(recibo)='object'),
 registrada_en timestamptz(6) NOT NULL,
 FOREIGN KEY(catalogo_ref,version,huella_sha256)
  REFERENCES vec_autorizacion.registro_catalogo_acciones_admin_v1(catalogo_ref,version,huella_sha256)
);
ALTER TABLE vec_autorizacion.cabeza_catalogo_acciones_admin_v1 ADD CONSTRAINT cabeza_catalogo_registro_fk
 FOREIGN KEY(catalogo_ref,version,huella_sha256)
 REFERENCES vec_autorizacion.registro_catalogo_acciones_admin_v1(catalogo_ref,version,huella_sha256)
 DEFERRABLE INITIALLY DEFERRED;

DO $tablas$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['config_catalogo_acciones_admin_v1','cabeza_catalogo_acciones_admin_v1',
  'registro_catalogo_acciones_admin_v1','operacion_catalogo_acciones_admin_v1'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('CREATE POLICY propietario_exacto ON vec_autorizacion.%I FOR ALL TO vec_autorizacion_propietario USING(current_user=''vec_autorizacion_propietario'') WITH CHECK(current_user=''vec_autorizacion_propietario'')',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_autorizacion.%I FROM PUBLIC',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_autorizacion.%I FROM PUBLIC',t);
  IF t IN ('registro_catalogo_acciones_admin_v1','operacion_catalogo_acciones_admin_v1') THEN
   EXECUTE pg_catalog.format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
   EXECUTE pg_catalog.format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
  END IF;
 END LOOP;
END $tablas$;

-- ACL y pertenencia se comprueban de nuevo en cada operación; la configuración
-- se bloquea contra revocación concurrente hasta el COMMIT del efecto.
CREATE FUNCTION vec_autorizacion.exigir_operador_catalogo_acciones_admin_v1()
RETURNS vec_autorizacion.config_catalogo_acciones_admin_v1 LANGUAGE plpgsql SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' AS $f$
DECLARE l record;g record;cfg vec_autorizacion.config_catalogo_acciones_admin_v1;ns oid;db oid;f oid;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off'
 OR pg_catalog.current_setting('TimeZone')<>'UTC'
 OR pg_catalog.current_setting('role')<>'none' THEN
  RAISE EXCEPTION 'AUT58: transacción inválida' USING ERRCODE='25000'; END IF;
 SELECT * INTO l FROM pg_catalog.pg_roles WHERE rolname=session_user;
 SELECT * INTO g FROM pg_catalog.pg_roles WHERE rolname='vec_admin_catalogo_acciones_ejecutor';
 ns:=pg_catalog.to_regnamespace('vec_autorizacion');
 SELECT oid INTO db FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database();
 f:=pg_catalog.to_regprocedure('vec_autorizacion.registrar_catalogo_acciones_admin_v1(text,text)');
 IF l.oid IS NULL OR g.oid IS NULL OR NOT l.rolcanlogin OR NOT l.rolinherit OR l.rolsuper OR l.rolcreatedb
 OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
 OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls
 OR g.rolconfig IS NOT NULL
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members WHERE member=l.oid)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=l.oid AND roleid=g.oid
  AND inherit_option AND NOT set_option AND NOT admin_option)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=g.oid)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting WHERE setrole IN(l.oid,g.oid))
 OR pg_catalog.has_database_privilege(l.oid,db,'CREATE,TEMP')
 OR pg_catalog.has_schema_privilege(l.oid,ns,'CREATE')
 OR NOT pg_catalog.has_function_privilege(l.oid,f,'EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
  WHERE n.nspname LIKE 'vec\_%' ESCAPE '\' AND c.relkind IN('r','p','v','m','f')
  AND (c.relowner=l.oid OR c.relowner=g.oid OR pg_catalog.has_table_privilege(l.oid,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')))
 THEN RAISE EXCEPTION 'AUT58: LOGIN técnico no acreditado' USING ERRCODE='42501'; END IF;
 SELECT * INTO cfg FROM vec_autorizacion.config_catalogo_acciones_admin_v1
  WHERE login_nombre=session_user FOR SHARE;
 IF NOT FOUND OR cfg.estado<>'admitida' OR pg_catalog.clock_timestamp()<cfg.vigente_desde
 OR pg_catalog.clock_timestamp()>=cfg.vigente_hasta THEN
  RAISE EXCEPTION 'AUT58: aprobación ausente, revocada o vencida' USING ERRCODE='42501'; END IF;
 RETURN cfg;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.exigir_operador_catalogo_acciones_admin_v1() FROM PUBLIC;

-- VersionRol en Go omite las dos listas opcionales vacías al calcular su
-- huella. La autoridad central puede conservar [] en el documento SQL.
-- Se normaliza sólo esta representación, sin cambiar una concesión ni su
-- garantía, antes del cotejo exacto con la instantánea aprobada.
CREATE FUNCTION vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(p_documento jsonb)
RETURNS jsonb LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
SET search_path=pg_catalog,pg_temp AS $f$
 SELECT pg_catalog.jsonb_set(p_documento,'{concesiones}',
  (SELECT pg_catalog.jsonb_agg(
   CASE WHEN gc ? 'obligaciones' AND gc->'obligaciones' IN ('[]'::jsonb,'null'::jsonb)
    THEN gc-'obligaciones' ELSE gc END ORDER BY n)
   FROM pg_catalog.jsonb_array_elements(p_documento->'concesiones') WITH ORDINALITY AS x(g,n)
   CROSS JOIN LATERAL (SELECT CASE WHEN g ? 'campos_permitidos'
    AND g->'campos_permitidos' IN ('[]'::jsonb,'null'::jsonb)
    THEN g-'campos_permitidos' ELSE g END AS gc) normalizada))
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.aplicar_catalogo_acciones_admin_v1(plan_canonico text,sha_aprobado text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='5s' SET statement_timeout='30s' AS $f$
DECLARE cfg vec_autorizacion.config_catalogo_acciones_admin_v1;
 p jsonb; paquete jsonb;catalogo jsonb;fuente jsonb;entrada jsonb;
 sha text;paquete_sha text;catalogo_sha text;censo_sha text;flatten jsonb;
 canon bytea;paquete_bytes bytea;ahora timestamptz(6);prev record;aud record;recibo jsonb;
 esperado_version integer;esperado_sha text;actual record;numero_entradas integer;numero_perfiles integer;campo text;
BEGIN
 cfg:=vec_autorizacion.exigir_operador_catalogo_acciones_admin_v1();
 IF plan_canonico IS NULL OR pg_catalog.octet_length(plan_canonico) NOT BETWEEN 1 AND 33554432
 OR sha_aprobado !~ '^[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'AUT58: plan o huella inválidos' USING ERRCODE='22023'; END IF;
 sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(plan_canonico,'UTF8')),'hex');
 IF sha IS DISTINCT FROM sha_aprobado OR sha IS DISTINCT FROM cfg.plan_sha256 THEN
  RAISE EXCEPTION 'AUT58: plan no aprobado' USING ERRCODE='42501'; END IF;
 p:=plan_canonico::jsonb;
 IF pg_catalog.jsonb_typeof(p) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'AUT58: plan no es un objeto' USING ERRCODE='22023'; END IF;
 FOREACH campo IN ARRAY ARRAY['esquema','operacion_ref','aprobacion_ref','aprobacion_sha256',
  'paquete_canon','paquete_ref','paquete_version','paquete_sha256','catalogo_canon',
  'catalogo_ref','catalogo_version','catalogo_sha256','esperado_version','esperado_sha256',
  'preparado_en','caduca_en','entorno'] LOOP
  IF pg_catalog.jsonb_typeof(p->campo) IS DISTINCT FROM 'string' THEN
   RAISE EXCEPTION 'AUT58: campo del plan ausente o no textual' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF pg_catalog.jsonb_typeof(p) IS DISTINCT FROM 'object'
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(p))<>17
 OR (p ?& ARRAY['esquema','operacion_ref','aprobacion_ref','aprobacion_sha256','paquete_canon',
  'paquete_ref','paquete_version','paquete_sha256','catalogo_canon','catalogo_ref',
  'catalogo_version','catalogo_sha256','esperado_version','esperado_sha256',
  'preparado_en','caduca_en','entorno']) IS NOT TRUE
 OR p->>'esquema' IS DISTINCT FROM 'vec.admin.catalogo-acciones.plan.v1'
 OR p->>'operacion_ref' !~ '^caa_[A-Za-z0-9_-]{22,123}$'
 OR p->>'aprobacion_ref' IS DISTINCT FROM cfg.aprobacion_ref
 OR p->>'aprobacion_sha256' IS DISTINCT FROM cfg.aprobacion_sha256
 OR p->>'paquete_ref' IS DISTINCT FROM cfg.paquete_ref
 OR p->>'paquete_version' IS DISTINCT FROM cfg.paquete_version::text
 OR p->>'paquete_sha256' IS DISTINCT FROM cfg.paquete_sha256
 OR p->>'catalogo_ref' IS DISTINCT FROM cfg.destino_ref
 OR p->>'catalogo_version' !~ '^[1-9][0-9]{0,8}$'
 OR p->>'catalogo_sha256' !~ '^[0-9a-f]{64}$'
 OR p->>'esperado_version' !~ '^(0|[1-9][0-9]{0,8})$'
 OR p->>'esperado_sha256' !~ '^[0-9a-f]{64}$'
 OR p->>'entorno' IS DISTINCT FROM cfg.entorno
 OR pg_catalog.jsonb_typeof(p->'paquete_canon') IS DISTINCT FROM 'string'
 OR pg_catalog.jsonb_typeof(p->'catalogo_canon') IS DISTINCT FROM 'string'
 THEN RAISE EXCEPTION 'AUT58: plan divergente de la aprobación' USING ERRCODE='42501'; END IF;
 ahora:=pg_catalog.clock_timestamp();
 IF p->>'preparado_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$'
 OR p->>'caduca_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$'
 OR (p->>'preparado_en')::timestamptz(6)>ahora
 OR ahora>=(p->>'caduca_en')::timestamptz(6)
 OR (p->>'caduca_en')::timestamptz(6)>(p->>'preparado_en')::timestamptz(6)+interval '1 day'
 OR (p->>'caduca_en')::timestamptz(6)>cfg.vigente_hasta
 THEN RAISE EXCEPTION 'AUT58: plan caducado o fuera de ventana' USING ERRCODE='42501'; END IF;
 IF pg_catalog.octet_length(p->>'paquete_canon') NOT BETWEEN 1 AND 16777216
 OR pg_catalog.octet_length(p->>'catalogo_canon') NOT BETWEEN 1 AND 16777216 THEN
  RAISE EXCEPTION 'AUT58: paquete o catálogo excede límite' USING ERRCODE='22023'; END IF;
 paquete_bytes:=pg_catalog.convert_to(p->>'paquete_canon','UTF8');
 canon:=pg_catalog.convert_to(p->>'catalogo_canon','UTF8');
 paquete_sha:=pg_catalog.encode(pg_catalog.sha256(paquete_bytes),'hex');
 catalogo_sha:=pg_catalog.encode(pg_catalog.sha256(canon),'hex');
 IF paquete_sha IS DISTINCT FROM cfg.paquete_sha256 OR catalogo_sha IS DISTINCT FROM p->>'catalogo_sha256' THEN
  RAISE EXCEPTION 'AUT58: paquete o catálogo alterado' USING ERRCODE='42501'; END IF;
 paquete:=(p->>'paquete_canon')::jsonb;
 catalogo:=(p->>'catalogo_canon')::jsonb;
 IF pg_catalog.jsonb_typeof(paquete) IS DISTINCT FROM 'object'
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(paquete))<>5
 OR (paquete ?& ARRAY['esquema','referencia','version','fuentes','perfiles']) IS NOT TRUE
 OR paquete->>'esquema' IS DISTINCT FROM 'vec.admin.catalogo-acciones.paquete.v1'
 OR paquete->>'referencia' IS DISTINCT FROM cfg.paquete_ref
 OR paquete->>'version' IS DISTINCT FROM cfg.paquete_version::text
 OR pg_catalog.jsonb_typeof(paquete->'fuentes') IS DISTINCT FROM 'array'
 OR pg_catalog.jsonb_array_length(paquete->'fuentes') NOT BETWEEN 1 AND 512
 OR pg_catalog.jsonb_typeof(paquete->'perfiles') IS DISTINCT FROM 'array'
 OR pg_catalog.jsonb_array_length(paquete->'perfiles') NOT BETWEEN 1 AND 512
 OR pg_catalog.jsonb_typeof(catalogo) IS DISTINCT FROM 'object'
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(catalogo))<>9
 OR (catalogo ?& ARRAY['referencia','version','fuente_ref','fuente_version','fuente_huella_sha256',
  'vigente_desde','vigente_hasta','entradas','perfiles']) IS NOT TRUE
 OR catalogo->>'referencia' IS DISTINCT FROM cfg.destino_ref
 OR catalogo->>'version' IS DISTINCT FROM p->>'catalogo_version'
 OR catalogo->>'fuente_ref' IS DISTINCT FROM cfg.paquete_ref
 OR catalogo->>'fuente_version' IS DISTINCT FROM cfg.paquete_version::text
 OR catalogo->>'fuente_huella_sha256' IS DISTINCT FROM cfg.paquete_sha256
 OR catalogo->'perfiles' IS DISTINCT FROM paquete->'perfiles'
 OR pg_catalog.jsonb_typeof(catalogo->'entradas') IS DISTINCT FROM 'array'
 OR pg_catalog.jsonb_array_length(catalogo->'entradas') NOT BETWEEN 1 AND 512
 THEN RAISE EXCEPTION 'AUT58: paquete o instantánea incompletos' USING ERRCODE='22023'; END IF;
 numero_entradas:=pg_catalog.jsonb_array_length(catalogo->'entradas');
 numero_perfiles:=pg_catalog.jsonb_array_length(catalogo->'perfiles');
 IF pg_catalog.jsonb_typeof(catalogo->'vigente_desde') IS DISTINCT FROM 'string'
 OR catalogo->>'vigente_desde' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$'
 OR pg_catalog.jsonb_typeof(catalogo->'vigente_hasta') IS DISTINCT FROM 'string'
 OR catalogo->>'vigente_hasta' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$'
 OR (catalogo->>'vigente_desde')::timestamptz(6)>ahora
 OR (catalogo->>'vigente_hasta')::timestamptz(6)<=ahora
 THEN RAISE EXCEPTION 'AUT58: instantánea sin vigencia actual' USING ERRCODE='42501'; END IF;
 -- Todas las entradas son aportadas por una fuente declarada en el paquete
 -- comprometido; ninguna fuente vacía ni entrada extra puede pasar.
 FOR fuente IN SELECT value FROM pg_catalog.jsonb_array_elements(paquete->'fuentes') LOOP
  IF pg_catalog.jsonb_typeof(fuente) IS DISTINCT FROM 'object'
  OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(fuente))<>5
  OR (fuente ?& ARRAY['modulo_id','referencia','version','huella_sha256','entradas']) IS NOT TRUE
  OR pg_catalog.jsonb_typeof(fuente->'modulo_id') IS DISTINCT FROM 'string'
  OR pg_catalog.jsonb_typeof(fuente->'referencia') IS DISTINCT FROM 'string'
  OR pg_catalog.jsonb_typeof(fuente->'version') IS DISTINCT FROM 'number'
  OR pg_catalog.jsonb_typeof(fuente->'huella_sha256') IS DISTINCT FROM 'string'
  OR fuente->>'modulo_id' !~ '^[a-z][a-z0-9._:-]{0,127}$'
  OR fuente->>'referencia' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
  OR fuente->>'version' !~ '^[1-9][0-9]{0,8}$'
  OR fuente->>'huella_sha256' !~ '^[0-9a-f]{64}$'
  OR pg_catalog.jsonb_typeof(fuente->'entradas') IS DISTINCT FROM 'array'
  OR pg_catalog.jsonb_array_length(fuente->'entradas') NOT BETWEEN 1 AND 512
  THEN RAISE EXCEPTION 'AUT58: fuente propietaria inválida' USING ERRCODE='22023'; END IF;
  FOR entrada IN SELECT value FROM pg_catalog.jsonb_array_elements(fuente->'entradas') LOOP
   IF pg_catalog.jsonb_typeof(entrada) IS DISTINCT FROM 'object'
   OR pg_catalog.jsonb_typeof(entrada->'fuente_ref') IS DISTINCT FROM 'string'
   OR pg_catalog.jsonb_typeof(entrada->'fuente_version') IS DISTINCT FROM 'number'
   OR pg_catalog.jsonb_typeof(entrada->'fuente_huella_sha256') IS DISTINCT FROM 'string'
   OR pg_catalog.jsonb_typeof(entrada#>'{concesion,modulo_id}') IS DISTINCT FROM 'string'
   OR entrada->>'fuente_ref' IS DISTINCT FROM fuente->>'referencia'
   OR entrada->>'fuente_version' IS DISTINCT FROM fuente->>'version'
   OR entrada->>'fuente_huella_sha256' IS DISTINCT FROM fuente->>'huella_sha256'
   OR entrada#>>'{concesion,modulo_id}' IS DISTINCT FROM fuente->>'modulo_id'
   THEN RAISE EXCEPTION 'AUT58: descriptor fuera de su fuente' USING ERRCODE='42501'; END IF;
  END LOOP;
 END LOOP;
 SELECT pg_catalog.jsonb_agg(e.value ORDER BY f.n,e.n) INTO flatten
 FROM pg_catalog.jsonb_array_elements(paquete->'fuentes') WITH ORDINALITY AS f(value,n)
 CROSS JOIN LATERAL pg_catalog.jsonb_array_elements(f.value->'entradas') WITH ORDINALITY AS e(value,n);
 IF flatten IS DISTINCT FROM catalogo->'entradas' THEN
  RAISE EXCEPTION 'AUT58: entradas omitidas, añadidas o reordenadas' USING ERRCODE='42501'; END IF;
 censo_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to((catalogo->'perfiles')::text,'UTF8')),'hex');
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:catalogo-acciones:'||cfg.destino_ref,0));
 SELECT * INTO prev FROM vec_autorizacion.operacion_catalogo_acciones_admin_v1
  WHERE operacion_ref=p->>'operacion_ref';
 IF FOUND THEN
  IF prev.plan_sha256 IS DISTINCT FROM sha OR prev.catalogo_ref IS DISTINCT FROM cfg.destino_ref
  OR prev.version IS DISTINCT FROM (p->>'catalogo_version')::integer
  OR prev.huella_sha256 IS DISTINCT FROM catalogo_sha THEN
   RAISE EXCEPTION 'AUT58: operación repetida con material diferente' USING ERRCODE='23505'; END IF;
  RETURN pg_catalog.jsonb_build_object('recibo',prev.recibo,'replay',true);
 END IF;
 -- Censo actual completo: una única versión máxima por RolID, con el control
 -- vigente exacto. El paquete aprobado aporta el tipo; sólo las fuentes
 -- centrales positivas pueden demostrar una condición fija ya existente.
 IF numero_perfiles IS DISTINCT FROM (
  SELECT pg_catalog.count(*) FROM (SELECT DISTINCT ON (v.rol_id) v.rol_id
   FROM vec_autorizacion.version_rol v ORDER BY v.rol_id,v.version DESC) x)
 OR EXISTS(
  SELECT 1 FROM (SELECT DISTINCT ON (v.rol_id) v.rol_id,v.version_rol_ref,v.documento
   FROM vec_autorizacion.version_rol v ORDER BY v.rol_id,v.version DESC) v
  LEFT JOIN vec_autorizacion.control_vigencia_version_rol_actual a USING(version_rol_ref)
  LEFT JOIN vec_autorizacion.control_vigencia_version_rol c
   ON c.version_rol_ref=a.version_rol_ref AND c.revision=a.revision
  WHERE c.documento IS NULL OR NOT EXISTS(
   SELECT 1 FROM pg_catalog.jsonb_array_elements(catalogo->'perfiles') perfil
   WHERE perfil#>>'{rol,rol_id}'=v.rol_id
   AND perfil->'rol'=vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(v.documento)
   AND perfil->'control_vigencia'=c.documento
   AND perfil->>'tipo_perfil' IN ('fijo_sistema','administrable')
   AND (perfil->>'tipo_perfil'='fijo_sistema' OR NOT EXISTS(
    SELECT 1 FROM vec_autorizacion.version_rol anterior
    WHERE anterior.rol_id=v.rol_id AND (
     EXISTS(SELECT 1 FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 f
      WHERE f.version_rol_ref=anterior.version_rol_ref)
     OR EXISTS(SELECT 1 FROM vec_autorizacion.rol_sensible_exacto s
      WHERE s.version_rol_ref=anterior.version_rol_ref))))))
 OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(catalogo->'perfiles') perfil
  WHERE pg_catalog.jsonb_typeof(perfil) IS DISTINCT FROM 'object'
  OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(perfil))<>3
  OR NOT (perfil ?& ARRAY['rol','control_vigencia','tipo_perfil']))
 THEN RAISE EXCEPTION 'AUT58: censo de roles incompleto o fijo reclasificado' USING ERRCODE='42501'; END IF;
 esperado_version:=(p->>'esperado_version')::integer;
 esperado_sha:=p->>'esperado_sha256';
 IF (esperado_version=0 AND esperado_sha<>pg_catalog.repeat('0',64))
 OR (esperado_version>0 AND esperado_sha=pg_catalog.repeat('0',64))
 OR (p->>'catalogo_version')::integer<>esperado_version+1 THEN
  RAISE EXCEPTION 'AUT58: preimagen CAS incoherente' USING ERRCODE='22023'; END IF;
 SELECT * INTO actual FROM vec_autorizacion.cabeza_catalogo_acciones_admin_v1
  WHERE catalogo_ref=cfg.destino_ref FOR UPDATE;
 IF esperado_version=0 THEN
  IF FOUND THEN RAISE EXCEPTION 'AUT58: cabeza ya publicada' USING ERRCODE='40001'; END IF;
 ELSE
  IF NOT FOUND OR actual.version<>esperado_version OR actual.huella_sha256<>esperado_sha THEN
   RAISE EXCEPTION 'AUT58: cabeza CAS divergente' USING ERRCODE='40001'; END IF;
 END IF;
 SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_catalogo_acciones_admin_v1(
  pg_catalog.jsonb_build_object('tipo_registro','catalogo_acciones_admin','evento_ref','evento_'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-',''),
   'operador_login',session_user::text,'operacion_ref',p->>'operacion_ref','plan_sha256',sha,
   'catalogo_ref',cfg.destino_ref,'catalogo_version',p->>'catalogo_version','catalogo_sha256',catalogo_sha,
   'paquete_ref',cfg.paquete_ref,'paquete_version',cfg.paquete_version::text,'paquete_sha256',paquete_sha,
   'censo_sha256',censo_sha,'entradas_numero',numero_entradas::text,'perfiles_numero',numero_perfiles::text,
   'aprobacion_ref',cfg.aprobacion_ref,'aprobacion_sha256',cfg.aprobacion_sha256,
   'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','catalogo_acciones_admin',
   'correlacion_ref','correlacion_'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','')));
 recibo:=pg_catalog.jsonb_build_object('esquema','vec.admin.catalogo-acciones.recibo.v1',
  'operacion_ref',p->>'operacion_ref','plan_sha256',sha,'catalogo_ref',cfg.destino_ref,
  'catalogo_version',p->>'catalogo_version','catalogo_sha256',catalogo_sha,
  'paquete_ref',cfg.paquete_ref,'paquete_version',cfg.paquete_version,
  'paquete_sha256',paquete_sha,'censo_sha256',censo_sha,'aprobacion_ref',cfg.aprobacion_ref,
  'aprobacion_sha256',cfg.aprobacion_sha256,'aprobador_ref',cfg.aprobador_ref,
  'auditoria_ref',aud.auditoria_ref,'auditoria_secuencia',aud.secuencia,
  'auditoria_huella_sha256',aud.huella_sha256,'confirmado_en',aud.registrada_en);
 INSERT INTO vec_autorizacion.registro_catalogo_acciones_admin_v1(
  catalogo_ref,version,huella_sha256,canon,paquete_ref,paquete_version,paquete_sha256,paquete_canon,
  censo_sha256,operacion_ref,plan_sha256,aprobacion_ref,aprobacion_sha256,aprobador_ref,login_nombre,
  auditoria_ref,recibo,registrada_en)
 VALUES(cfg.destino_ref,(p->>'catalogo_version')::integer,catalogo_sha,canon,cfg.paquete_ref,cfg.paquete_version,
  paquete_sha,paquete_bytes,censo_sha,p->>'operacion_ref',sha,cfg.aprobacion_ref,cfg.aprobacion_sha256,
  cfg.aprobador_ref,session_user,aud.auditoria_ref,recibo,aud.registrada_en);
 IF esperado_version=0 THEN
  INSERT INTO vec_autorizacion.cabeza_catalogo_acciones_admin_v1 VALUES(
   cfg.destino_ref,(p->>'catalogo_version')::integer,catalogo_sha,p->>'operacion_ref',aud.registrada_en);
 ELSE
  UPDATE vec_autorizacion.cabeza_catalogo_acciones_admin_v1
   SET version=(p->>'catalogo_version')::integer,huella_sha256=catalogo_sha,
   operacion_ref=p->>'operacion_ref',actualizada_en=aud.registrada_en
   WHERE catalogo_ref=cfg.destino_ref AND version=esperado_version AND huella_sha256=esperado_sha;
  IF NOT FOUND THEN RAISE EXCEPTION 'AUT58: CAS concurrente' USING ERRCODE='40001'; END IF;
 END IF;
 INSERT INTO vec_autorizacion.operacion_catalogo_acciones_admin_v1
 VALUES(p->>'operacion_ref',sha,cfg.destino_ref,(p->>'catalogo_version')::integer,
  catalogo_sha,aud.auditoria_ref,recibo,aud.registrada_en);
 PERFORM vec_autorizacion.exigir_operador_catalogo_acciones_admin_v1();
 IF pg_catalog.clock_timestamp()>=(p->>'caduca_en')::timestamptz(6) THEN
  RAISE EXCEPTION 'AUT58: plan vencido antes del efecto' USING ERRCODE='42501'; END IF;
 RETURN pg_catalog.jsonb_build_object('recibo',recibo,'replay',false);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.aplicar_catalogo_acciones_admin_v1(text,text) FROM PUBLIC;

-- Un intento permitido y la publicación comparten subtransacción: si falla
-- la auditoría, ninguna instantánea ni cabeza sobrevive. Un rechazo deja su
-- intento en la cadena común después de deshacer el subbloque del efecto.
CREATE FUNCTION vec_autorizacion.registrar_catalogo_acciones_admin_v1(plan_canonico text,sha_aprobado text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='5s' SET statement_timeout='40s' AS $f$
DECLARE respuesta jsonb;aud record;estado text:='permitido';motivo text;codigo text;
 solicitud_sha text;evento text;corr text;sol text;
BEGIN
 evento:='evento_'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','');
 corr:='correlacion_'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','');
 sol:='solicitud_catalogo_acciones_admin:'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','');
 solicitud_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  pg_catalog.jsonb_build_object('operacion','registrar_catalogo_acciones_admin_v1',
   'plan',CASE WHEN pg_catalog.octet_length(plan_canonico)<=33554432 THEN plan_canonico ELSE '' END,
   'sha_aprobado',CASE WHEN pg_catalog.octet_length(sha_aprobado)<=128 THEN sha_aprobado ELSE '' END)::text,'UTF8')),'hex');
 BEGIN
  respuesta:=vec_autorizacion.aplicar_catalogo_acciones_admin_v1(plan_canonico,sha_aprobado);
  motivo:=CASE WHEN (respuesta->>'replay')::boolean THEN 'catalogo_acciones_replay' ELSE 'catalogo_acciones_registrado' END;
  SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_intento_catalogo_acciones_admin_v1(
   pg_catalog.jsonb_build_object('tipo_registro','intento_catalogo_acciones_admin','evento_ref',evento,
    'operador_login',session_user::text,'solicitud_sha256',solicitud_sha,
    'accion','registrar_catalogo_acciones_admin_v1','recurso_ref',sol,'resultado','permitido',
    'motivo_ref',motivo,'proceso','postgresql','canal','operacion_tecnica_privada',
    'finalidad_ref','catalogo_acciones_admin','correlacion_ref',corr));
  PERFORM vec_autorizacion.exigir_operador_catalogo_acciones_admin_v1();
 EXCEPTION WHEN OTHERS THEN
  respuesta:=NULL;
  estado:=CASE WHEN SQLSTATE IN ('42501','22023','22P02','22007','22008','23505','40001','25000')
   THEN 'denegado' ELSE 'error' END;
  motivo:=CASE WHEN estado='denegado' THEN 'catalogo_acciones_denegado' ELSE 'catalogo_acciones_error' END;
  codigo:=CASE WHEN estado='denegado' THEN 'catalogo_acciones_rechazado' ELSE 'catalogo_acciones_no_disponible' END;
 END;
 IF estado<>'permitido' THEN
  SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_intento_catalogo_acciones_admin_v1(
   pg_catalog.jsonb_build_object('tipo_registro','intento_catalogo_acciones_admin','evento_ref',evento,
    'operador_login',session_user::text,'solicitud_sha256',solicitud_sha,
    'accion','registrar_catalogo_acciones_admin_v1','recurso_ref',sol,'resultado',estado,
    'motivo_ref',motivo,'proceso','postgresql','canal','operacion_tecnica_privada',
    'finalidad_ref','catalogo_acciones_admin','correlacion_ref',corr));
 END IF;
 RETURN pg_catalog.jsonb_build_object('estado',estado,'codigo',codigo,
  'recibo',respuesta->'recibo','replay',coalesce((respuesta->>'replay')::boolean,false),
  'auditoria_intento',pg_catalog.jsonb_build_object('auditoria_ref',aud.auditoria_ref,
   'secuencia',aud.secuencia,'huella_sha256',aud.huella_sha256,
   'correlacion_ref',aud.correlacion_ref,'registrada_en',aud.registrada_en));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.registrar_catalogo_acciones_admin_v1(text,text) FROM PUBLIC;

-- Lectura de preimagen exacta y completa para el puerto Go. No transforma
-- JSONB: devuelve los bytes canónicos originales que aprobó el circuito.
CREATE FUNCTION vec_autorizacion.resolver_catalogo_acciones_administracion_v1(
 p_ref text,p_version integer,p_sha text)
RETURNS bytea LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE r record; o record; c jsonb;
BEGIN
 IF p_ref IS NULL OR p_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_version NOT BETWEEN 1 AND 2147483647 OR p_sha !~ '^[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'AUT58: selector de catálogo inválido' USING ERRCODE='22023'; END IF;
 SELECT * INTO r FROM vec_autorizacion.registro_catalogo_acciones_admin_v1
 WHERE catalogo_ref=p_ref AND version=p_version AND huella_sha256=p_sha;
 IF NOT FOUND THEN RAISE EXCEPTION 'AUT58: catálogo no publicado' USING ERRCODE='P0002'; END IF;
 SELECT * INTO o FROM vec_autorizacion.operacion_catalogo_acciones_admin_v1
 WHERE operacion_ref=r.operacion_ref;
 c:=pg_catalog.convert_from(r.canon,'UTF8')::jsonb;
 IF o.operacion_ref IS NULL OR o.plan_sha256 IS DISTINCT FROM r.plan_sha256
 OR o.auditoria_ref IS DISTINCT FROM r.auditoria_ref
 OR o.catalogo_ref IS DISTINCT FROM r.catalogo_ref
 OR o.version IS DISTINCT FROM r.version OR o.huella_sha256 IS DISTINCT FROM r.huella_sha256
 OR pg_catalog.encode(pg_catalog.sha256(r.canon),'hex') IS DISTINCT FROM r.huella_sha256
 OR pg_catalog.encode(pg_catalog.sha256(r.paquete_canon),'hex') IS DISTINCT FROM r.paquete_sha256
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to((c->'perfiles')::text,'UTF8')),'hex') IS DISTINCT FROM r.censo_sha256
 OR c->>'referencia' IS DISTINCT FROM r.catalogo_ref OR c->>'version' IS DISTINCT FROM r.version::text
 OR c->>'fuente_ref' IS DISTINCT FROM r.paquete_ref
 OR c->>'fuente_version' IS DISTINCT FROM r.paquete_version::text
 OR c->>'fuente_huella_sha256' IS DISTINCT FROM r.paquete_sha256
 OR pg_catalog.jsonb_typeof(c->'entradas') IS DISTINCT FROM 'array'
 OR pg_catalog.jsonb_typeof(c->'perfiles') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'AUT58: catálogo o auditoría divergente' USING ERRCODE='42501'; END IF;
 RETURN r.canon;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.resolver_catalogo_acciones_administracion_v1(text,integer,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_catalogo_acciones_ejecutor,vec_admin_catalogo_acciones_lector;
GRANT EXECUTE ON FUNCTION vec_autorizacion.registrar_catalogo_acciones_admin_v1(text,text)
 TO vec_admin_catalogo_acciones_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion.resolver_catalogo_acciones_administracion_v1(text,integer,text)
 TO vec_admin_catalogo_acciones_ejecutor,vec_admin_catalogo_acciones_lector;
RESET ROLE;

DO $acl$
DECLARE e oid:=pg_catalog.to_regrole('vec_admin_catalogo_acciones_ejecutor');
 l oid:=pg_catalog.to_regrole('vec_admin_catalogo_acciones_lector');
BEGIN
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_class c
  JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
  CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(c.relacl,pg_catalog.acldefault('r',c.relowner))) x
  WHERE n.nspname='vec_autorizacion' AND c.relname IN(
   'config_catalogo_acciones_admin_v1','cabeza_catalogo_acciones_admin_v1',
   'registro_catalogo_acciones_admin_v1','operacion_catalogo_acciones_admin_v1')
   AND x.grantee<>c.relowner)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
  CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x
  WHERE p.oid IN (pg_catalog.to_regprocedure('vec_autorizacion.exigir_operador_catalogo_acciones_admin_v1()'),
   pg_catalog.to_regprocedure('vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(jsonb)'),
   pg_catalog.to_regprocedure('vec_autorizacion.aplicar_catalogo_acciones_admin_v1(text,text)'))
  AND x.grantee<>p.proowner)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
  CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x
  WHERE p.oid=pg_catalog.to_regprocedure('vec_autorizacion.registrar_catalogo_acciones_admin_v1(text,text)')
  AND x.grantee NOT IN(p.proowner,e))
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
  CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x
  WHERE p.oid=pg_catalog.to_regprocedure('vec_autorizacion.resolver_catalogo_acciones_administracion_v1(text,integer,text)')
  AND x.grantee NOT IN(p.proowner,e,l))
 THEN RAISE EXCEPTION 'AUT58: ACL abierta' USING ERRCODE='42501'; END IF;
END $acl$;
COMMIT;
