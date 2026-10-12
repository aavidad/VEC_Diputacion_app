\set ON_ERROR_STOP on
-- AUT77: dos altas gobernadas de preferencias propias externas y CAS nominal sobre el RolID Bolsa.
-- AUT16 conserva v1; este acto nominal publica exclusivamente el rol externo v2.
-- AD248 instala el consumidor V3 de este ejecutor antes de abrir los actos.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
DECLARE firma text;tabla text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AUT77: requiere PostgreSQL 18 y superusuario migrador' USING ERRCODE='55000';END IF;
 IF to_regrole('vec_admin_version_rol_bolsa_ejecutor') IS NULL
 THEN RAISE EXCEPTION 'AUT77: falta ejecutor AD227' USING ERRCODE='55000';END IF;
 FOREACH firma IN ARRAY ARRAY[
  'vec_autorizacion.canon_version_rol_bolsa_v1(jsonb,text)',
  'vec_autorizacion.validar_plan_version_rol_bolsa_v1(jsonb,boolean)',
  'vec_autorizacion.aplicar_version_rol_bolsa_v1(boolean,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_autorizacion.acreditar_version_rol_bolsa_v1(jsonb)',
  'vec_autorizacion.administradores_version_rol_bolsa_v1()',
  'vec_autorizacion.solicitud_replay_version_rol_bolsa_v1(bytea,text)',
  'vec_autorizacion.resultado_propuesta_acceso_version_rol_bolsa_v1(jsonb,text)',
  'vec_autorizacion.registrar_fallo_version_rol_bolsa_v1(text,text,text)',
  'vec_autorizacion.resolver_rol_administrable_v1(text)',
  'vec_autorizacion.cotejar_lectura_preferencias_bolsas_actual_v1(jsonb,text,text)',
  'vec_contexto_actor_v1.acreditar_candidato_externo_v1(text,text,text)',
  'vec_autorizacion.resolver_catalogo_acciones_administracion_v2(text,integer,text)',
  'vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_autorizacion_atestada_v3.registrar_consumir_admin_perfiles_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
  IF to_regprocedure(firma) IS NULL
  THEN RAISE EXCEPTION 'AUT77: falta función preexistente %',firma USING ERRCODE='55000';END IF;
 END LOOP;
 FOREACH firma IN ARRAY ARRAY[
  'vec_autorizacion.canon_version_rol_bolsa_externa_v1(jsonb,text)',
  'vec_autorizacion.resolver_base_version_rol_bolsa_externa_v1()',
  'vec_autorizacion.registrar_fallo_version_rol_bolsa_externa_v1(text,text,text)',
  'vec_autorizacion.validar_plan_version_rol_bolsa_externa_v1(jsonb,boolean)',
  'vec_autorizacion.comprobar_postimagen_version_rol_bolsa_externa_v1(text)',
  'vec_autorizacion.aplicar_version_rol_bolsa_externa_v1(boolean,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_autorizacion.proponer_version_rol_bolsa_externa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_autorizacion.cerrar_version_rol_bolsa_externa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
  IF to_regprocedure(firma) IS NOT NULL
  THEN RAISE EXCEPTION 'AUT77: función propia ya presente %',firma USING ERRCODE='55000';END IF;
 END LOOP;
 FOREACH tabla IN ARRAY ARRAY['propuesta_version_rol_bolsa_externa_v1',
  'cierre_version_rol_bolsa_externa_v1','outbox_version_rol_bolsa_externa_v1',
  'sello_efecto_admin_externo_tx_v1'] LOOP
  IF to_regclass('vec_autorizacion.'||tabla) IS NOT NULL
  THEN RAISE EXCEPTION 'AUT77: tabla propia ya presente %',tabla USING ERRCODE='55000';END IF;
 END LOOP;
 IF to_regrole('vec_admin_version_rol_bolsa_externa_ejecutor') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT77: ejecutor externo ya presente' USING ERRCODE='55000';END IF;
END $pre$;

-- Barrera técnica prospectiva: dos definiciones literales de la preimagen HZ21.
-- No cambian firmas, OID, propietario, ACL ni el comportamiento v1.
DO $tecnica_pre$
DECLARE v record;p pg_proc%ROWTYPE;
BEGIN
 IF (SELECT encode(sha256(convert_to(validador.prosrc,'UTF8')),'hex')
  FROM pg_proc validador WHERE validador.oid=to_regprocedure('vec_autorizacion.rol_candidato_externo_acotado_v1(jsonb)'))
  IS DISTINCT FROM '4212e0388ee883840011ea7874079e24eac5c947b2e8cc3c2f71921e613965da'
 THEN RAISE EXCEPTION 'AUT77: validador técnico externo divergente' USING ERRCODE='55000';END IF;
 FOR v IN SELECT * FROM (VALUES
  ('vec_autorizacion.publicar_rol_candidato_externo_v1(bytea,text,bytea,text,numeric,text,text,text)',
   'b58b39f2c35341a545640bfde366f07a89ee65831306950c7f3ea342ee846ff0',
   '01630f279ce8392c4e50cac14b10c63134c5fb68763b771dec3b61dc2fd149cb','vec.aut77.rol_meta'),
  ('vec_autorizacion.publicar_asignacion_candidato_externo_v1(bytea,text,bigint,text,text,text)',
   '4c1801f62da2beb36fea79199fbf3006572ca1f1444c3030729e14458e873811',
   '3301caa2a0d62bc247164000dc1f709186b08bb57da939e7f689311b0d07722d','vec.aut77.asg_meta')
 ) AS x(firma,fuente_sha,def_sha,guc) LOOP
  SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure(v.firma);
  IF NOT FOUND OR p.proowner<>to_regrole('vec_autorizacion_propietario')
  OR p.prosecdef IS NOT TRUE OR p.provolatile<>'v' OR p.proparallel<>'u'
  OR p.proisstrict IS TRUE
  OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[]
  OR p.proacl IS DISTINCT FROM ARRAY[
   'vec_autorizacion_propietario=X/vec_autorizacion_propietario'::aclitem,
   'vec_autorizacion_publicador_candidato_externo=X/vec_autorizacion_propietario'::aclitem]
  OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM v.fuente_sha
  OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM v.def_sha
  THEN RAISE EXCEPTION 'AUT77: publicador técnico instalado divergente %',v.firma USING ERRCODE='55000';END IF;
  PERFORM set_config(v.guc,(to_jsonb(p)-'prosrc')::text,true);
 END LOOP;
END $tecnica_pre$;
CREATE ROLE vec_admin_version_rol_bolsa_externa_ejecutor
 NOLOGIN NOINHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOREPLICATION NOBYPASSRLS;
DO $conexion$
BEGIN
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_admin_version_rol_bolsa_externa_ejecutor',current_database());
END $conexion$;
SET LOCAL ROLE vec_autorizacion_propietario;

CREATE TABLE vec_autorizacion.propuesta_version_rol_bolsa_externa_v1(
 propuesta_ref text PRIMARY KEY CHECK(propuesta_ref ~ '^propuesta_admin:[0-9a-f]{32}$'),
 material bytea NOT NULL CHECK(octet_length(material) BETWEEN 1 AND 131072),
 material_sha256 text NOT NULL CHECK(material_sha256=encode(sha256(material),'hex')),
 solicitud bytea NOT NULL CHECK(octet_length(solicitud) BETWEEN 1 AND 196608),
 proponente_persona_ref text NOT NULL,proponente_perfil_ref text NOT NULL,asignacion_ref text NOT NULL,
 catalogo_ref text NOT NULL,catalogo_version integer NOT NULL,catalogo_sha256 text NOT NULL,
 version_rol_ref text NOT NULL,
 creada_en timestamptz(6) NOT NULL,caduca_en timestamptz(6) NOT NULL,
 auditoria_ref text NOT NULL,resultado jsonb NOT NULL,
 CHECK(isfinite(creada_en) AND isfinite(caduca_en) AND caduca_en>creada_en AND caduca_en<=creada_en+interval '1 day'),
 FOREIGN KEY(catalogo_ref,catalogo_version,catalogo_sha256)
 REFERENCES vec_autorizacion.registro_catalogo_acciones_admin_v1(catalogo_ref,version,huella_sha256)
);
CREATE TABLE vec_autorizacion.cierre_version_rol_bolsa_externa_v1(
 propuesta_ref text PRIMARY KEY REFERENCES vec_autorizacion.propuesta_version_rol_bolsa_externa_v1(propuesta_ref),
 operacion_ref text NOT NULL UNIQUE CHECK(operacion_ref ~ '^cierre_admin:[0-9a-f]{32}$'),
 solicitud bytea NOT NULL CHECK(octet_length(solicitud) BETWEEN 1 AND 65536),
 version_rol_ref text NOT NULL UNIQUE REFERENCES vec_autorizacion.version_rol(version_rol_ref),
 rol_sha256 text NOT NULL CHECK(rol_sha256 ~ '^[0-9a-f]{64}$'),
 control_sha256 text NOT NULL CHECK(control_sha256 ~ '^[0-9a-f]{64}$'),
 aprobador_persona_ref text NOT NULL,aprobador_perfil_ref text NOT NULL,asignacion_ref text NOT NULL,
 auditoria_ref text NOT NULL,resultado jsonb NOT NULL,confirmado_en timestamptz(6) NOT NULL CHECK(isfinite(confirmado_en))
);
CREATE TABLE vec_autorizacion.outbox_version_rol_bolsa_externa_v1(
 operacion_ref text PRIMARY KEY,evento text NOT NULL CHECK(evento IN('version_propuesta','version_publicada')),
 auditoria_ref text NOT NULL,creada_en timestamptz(6) NOT NULL CHECK(isfinite(creada_en))
);
CREATE TABLE vec_autorizacion.sello_efecto_admin_externo_tx_v1(
 asignacion_ref text PRIMARY KEY REFERENCES vec_autorizacion.asignacion_perfil_externa(asignacion_ref),
 transaccion bigint NOT NULL,operacion_ref text NOT NULL,auditoria_ref text NOT NULL
);
ALTER TABLE vec_autorizacion.sello_efecto_admin_externo_tx_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.sello_efecto_admin_externo_tx_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.sello_efecto_admin_externo_tx_v1
 TO vec_autorizacion_propietario USING(current_user='vec_autorizacion_propietario')
 WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.sello_efecto_admin_externo_tx_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.sello_efecto_admin_externo_tx_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.sello_efecto_admin_externo_tx_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.sello_efecto_admin_externo_tx_v1 FROM PUBLIC;
ALTER TABLE vec_autorizacion.propuesta_version_rol_bolsa_externa_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.propuesta_version_rol_bolsa_externa_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.propuesta_version_rol_bolsa_externa_v1
 TO vec_autorizacion_propietario USING(current_user='vec_autorizacion_propietario')
 WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.propuesta_version_rol_bolsa_externa_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.propuesta_version_rol_bolsa_externa_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.propuesta_version_rol_bolsa_externa_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.propuesta_version_rol_bolsa_externa_v1 FROM PUBLIC;

ALTER TABLE vec_autorizacion.cierre_version_rol_bolsa_externa_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.cierre_version_rol_bolsa_externa_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.cierre_version_rol_bolsa_externa_v1
 TO vec_autorizacion_propietario USING(current_user='vec_autorizacion_propietario')
 WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.cierre_version_rol_bolsa_externa_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.cierre_version_rol_bolsa_externa_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.cierre_version_rol_bolsa_externa_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.cierre_version_rol_bolsa_externa_v1 FROM PUBLIC;

ALTER TABLE vec_autorizacion.outbox_version_rol_bolsa_externa_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.outbox_version_rol_bolsa_externa_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.outbox_version_rol_bolsa_externa_v1
 TO vec_autorizacion_propietario USING(current_user='vec_autorizacion_propietario')
 WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.outbox_version_rol_bolsa_externa_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.outbox_version_rol_bolsa_externa_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.outbox_version_rol_bolsa_externa_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.outbox_version_rol_bolsa_externa_v1 FROM PUBLIC;

-- Lectura nominal de la base, sin incorporarla artificialmente al censo del
-- catálogo. El contrato V3 de efecto se consume después, en proponer/cerrar.
CREATE FUNCTION vec_autorizacion.resolver_base_version_rol_bolsa_externa_v1()
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE r record;c record;
BEGIN
 IF current_user<>'vec_autorizacion_propietario'
 OR current_setting('role')<>'none'
 OR vec_autorizacion_atestada_v3.login_version_rol_bolsa_externa_valido_v1() IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT77: lectura base denegada' USING ERRCODE='42501';END IF;
 SELECT * INTO STRICT r FROM vec_autorizacion.version_rol
 WHERE version_rol_ref='rol:candidato_bolsa_portal_historial_propio_desarrollo:v1' FOR SHARE;
 SELECT h.* INTO STRICT c FROM vec_autorizacion.control_vigencia_version_rol_actual p
 JOIN vec_autorizacion.control_vigencia_version_rol h USING(version_rol_ref,revision)
 WHERE p.version_rol_ref=r.version_rol_ref FOR SHARE OF p,h;
 IF r.rol_id<>'candidato_bolsa_portal_historial_propio_desarrollo'
 OR r.version<>1 OR r.documento->>'estado'<>'publicada' OR c.estado<>'habilitada'
 OR r.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(
  vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
 OR c.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(
  vec_autorizacion.canon_control_rol_admin_v1(c.documento),'UTF8')),'hex')
 THEN RAISE EXCEPTION 'AUT77: base externa divergente' USING ERRCODE='40001';END IF;
 RETURN jsonb_build_object('rol',vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(r.documento),
  'control_vigencia',c.documento,'tipo_perfil','administrable');
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.resolver_base_version_rol_bolsa_externa_v1() FROM PUBLIC;


CREATE FUNCTION vec_autorizacion.canon_version_rol_bolsa_externa_v1(p jsonb,t text)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE s text:='';selecciones text:='';salida text:='';e jsonb;partes text[]:=ARRAY[]::text[];claves text[];
BEGIN
 IF jsonb_typeof(p) IS DISTINCT FROM 'object' OR octet_length(p::text)>131072
 THEN RAISE EXCEPTION 'AUT77: canon invalido' USING ERRCODE='22023';END IF;
 CASE t
 WHEN 'material' THEN
  claves:=ARRAY['OperacionRef','ProponentePersonaRef','PerfilActivoRef','AsignacionPerfilRef','Plan'];
  partes:=ARRAY[
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'OperacionRef'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'ProponentePersonaRef'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'PerfilActivoRef'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'AsignacionPerfilRef'),
   vec_autorizacion.canon_version_rol_bolsa_externa_v1(p->'Plan','plan')];
 WHEN 'plan' THEN
  claves:=ARRAY['operacion','catalogo_ref','catalogo_version','catalogo_huella_sha256',
   'version_rol_objetivo_ref','base','definicion_nueva','selecciones','asignaciones','motivo','referencia_acto'];
  IF p->>'operacion' IS DISTINCT FROM 'versionar'
   OR jsonb_typeof(p->'selecciones') IS DISTINCT FROM 'array'
   OR jsonb_array_length(p->'selecciones')<>2
   OR jsonb_typeof(p->'asignaciones') IS DISTINCT FROM 'array'
   OR jsonb_array_length(p->'asignaciones') NOT BETWEEN 1 AND 16
   OR (p->>'catalogo_version') !~ '^[1-9][0-9]{0,8}$'
  THEN RAISE EXCEPTION 'AUT77: plan invalido' USING ERRCODE='22023';END IF;
  FOR e IN SELECT value FROM jsonb_array_elements(p->'selecciones') WITH ORDINALITY x(value,n) ORDER BY n LOOP
   IF selecciones<>'' THEN selecciones:=selecciones||',';END IF;
   selecciones:=selecciones||vec_autorizacion.canon_gobierno_rol_nuevo_v1(e,'seleccion');
  END LOOP;
  FOR e IN SELECT value FROM jsonb_array_elements(p->'asignaciones') WITH ORDINALITY x(value,n) ORDER BY n LOOP
   IF s<>'' THEN s:=s||',';END IF;
   s:=s||vec_autorizacion.canon_version_rol_bolsa_externa_v1(e,'asignacion');
  END LOOP;
  s:='['||s||']';
  partes:=ARRAY[
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'operacion'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'catalogo_ref'),
   p->>'catalogo_version',
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'catalogo_huella_sha256'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'version_rol_objetivo_ref'),
   vec_autorizacion.canon_version_rol_bolsa_externa_v1(p->'base','base'),
   vec_autorizacion.canon_version_rol_bolsa_externa_v1(p->'definicion_nueva','definicion'),
   '['||selecciones||']',
   s,
   vec_autorizacion.canon_gobierno_rol_nuevo_v1(p->'motivo','motivo'),
   CASE WHEN coalesce(p->>'referencia_acto','')='' THEN NULL
    ELSE vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'referencia_acto') END];
 WHEN 'base' THEN
  claves:=ARRAY['rol','control_vigencia','tipo_perfil'];
  partes:=ARRAY[
   vec_autorizacion.canon_version_rol_admin_v1(p->'rol'),
   vec_autorizacion.canon_control_rol_admin_v1(p->'control_vigencia'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'tipo_perfil')];
 WHEN 'definicion' THEN
  claves:=ARRAY['rol_id','version','nombre','concesiones'];
  IF jsonb_typeof(p->'concesiones') IS DISTINCT FROM 'array'
   OR jsonb_array_length(p->'concesiones') NOT BETWEEN 2 AND 512
   OR (p->>'version') !~ '^[1-9][0-9]{0,8}$'
  THEN RAISE EXCEPTION 'AUT77: definicion invalida' USING ERRCODE='22023';END IF;
  FOR e IN SELECT value FROM jsonb_array_elements(p->'concesiones') WITH ORDINALITY x(value,n) ORDER BY n LOOP
   IF s<>'' THEN s:=s||',';END IF;
   s:=s||vec_autorizacion.objeto_canonico_go_admin_v1(e,'concesion');
  END LOOP;
  partes:=ARRAY[
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'rol_id'),
   p->>'version',
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'nombre'),
   '['||s||']'];
 WHEN 'asignacion' THEN
  claves:=ARRAY['asignacion_ref','huella_sha256','documento'];
  partes:=ARRAY[
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'asignacion_ref'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'huella_sha256'),
   vec_autorizacion.canon_asignacion_perfil_admin_v1(p->'documento')];
 ELSE RAISE EXCEPTION 'AUT77: tipo canon desconocido' USING ERRCODE='22023';
 END CASE;
 IF t='plan' AND partes[11] IS NULL THEN
  IF p ? 'referencia_acto' THEN
   RAISE EXCEPTION 'AUT77: referencia de acto vacía' USING ERRCODE='22023';END IF;
  claves:=claves[1:10];partes:=partes[1:10];
 END IF;
 IF (SELECT count(*) FROM jsonb_object_keys(p))<>cardinality(claves)
 OR NOT (p ?& claves)
 OR EXISTS(SELECT 1 FROM jsonb_object_keys(p) x WHERE NOT x=ANY(claves))
 OR array_position(partes,NULL) IS NOT NULL
 THEN RAISE EXCEPTION 'AUT77: campos canon divergentes' USING ERRCODE='22023';END IF;
 FOR i IN 1..cardinality(claves) LOOP
  IF partes[i] IS NULL THEN CONTINUE;END IF;
  IF salida<>'' THEN salida:=salida||',';END IF;
  salida:=salida||vec_autorizacion.json_cadena_canonica_go_admin_v1(claves[i])||':'||partes[i];
 END LOOP;
 RETURN '{'||salida||'}';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.canon_version_rol_bolsa_externa_v1(jsonb,text) FROM PUBLIC;


CREATE FUNCTION vec_autorizacion.validar_plan_version_rol_bolsa_externa_v1(p jsonb,p_primera boolean)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on AS $f$
DECLARE catalogo jsonb;entrada jsonb;seleccion jsonb;base jsonb;nueva jsonb;
 cabeza record;rol record;control record;puntero_control record;asignacion record;
 t jsonb;original jsonb;vistas text[]:=ARRAY[]::text[];
 acciones text[]:=ARRAY['bolsa.preferencias_bolsas.consultar_propias',
  'bolsa.preferencias_bolsas.ordenar_propias'];
 tipos text[]:=ARRAY['participaciones_candidato','participaciones_candidato'];
 finalidades text[]:=ARRAY['gestion_preferencias_bolsas_propias',
  'gestion_preferencias_bolsas_propias'];
 hojas text[];normalizadas jsonb:='[]'::jsonb;concesion jsonb;ahora timestamptz;
 i integer;campos_actuales jsonb;campos_esperados jsonb;ambitos jsonb;
 limite_entradas timestamptz:='infinity'::timestamptz;
BEGIN
 PERFORM vec_autorizacion.canon_version_rol_bolsa_externa_v1(p,'plan');
 IF p_primera IS NULL OR p->>'operacion' IS DISTINCT FROM 'versionar'
 OR p#>>'{definicion_nueva,rol_id}' IS DISTINCT FROM 'candidato_bolsa_portal_historial_propio_desarrollo'
 OR p#>>'{base,rol,rol_id}' IS DISTINCT FROM p#>>'{definicion_nueva,rol_id}'
 OR (p#>>'{definicion_nueva,version}')::bigint<>(p#>>'{base,rol,version}')::bigint+1
 OR p->>'version_rol_objetivo_ref' IS DISTINCT FROM
  'rol:'||(p#>>'{definicion_nueva,rol_id}')||':v'||(p#>>'{definicion_nueva,version}')
 OR p#>>'{base,tipo_perfil}' IS DISTINCT FROM 'administrable'
 OR p#>>'{base,rol,version}' IS DISTINCT FROM '1'
 OR p#>>'{definicion_nueva,version}' IS DISTINCT FROM '2'
 THEN RAISE EXCEPTION 'AUT77: version o base invalida' USING ERRCODE='22023';END IF;
 base:=p->'base';nueva:=p->'definicion_nueva';ahora:=clock_timestamp();
 IF p_primera THEN
  SELECT * INTO cabeza FROM vec_autorizacion.cabeza_catalogo_acciones_admin_v1
   WHERE catalogo_ref=p->>'catalogo_ref' FOR SHARE;
  IF NOT FOUND OR cabeza.version::text IS DISTINCT FROM p->>'catalogo_version'
  OR cabeza.huella_sha256 IS DISTINCT FROM p->>'catalogo_huella_sha256'
  THEN RAISE EXCEPTION 'AUT77: cabeza de catalogo divergente' USING ERRCODE='40001';END IF;
 END IF;
 -- El catálogo gobernado acredita las dos entradas exactas.
 SELECT convert_from(x.catalogo_canon,'UTF8')::jsonb INTO STRICT catalogo
 FROM vec_autorizacion.resolver_catalogo_acciones_administracion_v2(
  p->>'catalogo_ref',(p->>'catalogo_version')::integer,p->>'catalogo_huella_sha256') x;
 IF p_primera AND (ahora<(catalogo->>'vigente_desde')::timestamptz
 OR (catalogo->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
  AND ahora>=(catalogo->>'vigente_hasta')::timestamptz))
 THEN RAISE EXCEPTION 'AUT77: catalogo no vigente' USING ERRCODE='42501';END IF;
 FOR i IN 0..1 LOOP
  seleccion:=p->'selecciones'->i;
  PERFORM vec_autorizacion.canon_gobierno_rol_nuevo_v1(seleccion,'seleccion');
  SELECT value INTO entrada FROM jsonb_array_elements(catalogo->'entradas')
   WHERE value->>'referencia'=seleccion->>'entrada_ref';
  IF NOT FOUND THEN RAISE EXCEPTION 'AUT77: descriptor ausente' USING ERRCODE='42501';END IF;
  concesion:=vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(
   jsonb_build_object('concesiones',jsonb_build_array(entrada->'concesion')))#>'{concesiones,0}';
  IF entrada->>'version' IS DISTINCT FROM seleccion->>'entrada_version'
  OR encode(sha256(convert_to(vec_autorizacion.canon_gobierno_rol_nuevo_v1(entrada,'entrada'),'UTF8')),'hex')
    IS DISTINCT FROM seleccion->>'entrada_huella_sha256'
  OR entrada#>>'{concesion,accion}' IS DISTINCT FROM acciones[i+1]
  OR entrada#>>'{concesion,modulo_id}' IS DISTINCT FROM 'bolsa'
  OR entrada#>>'{concesion,tipo_recurso}' IS DISTINCT FROM tipos[i+1]
  OR entrada#>'{concesion,finalidades}' IS DISTINCT FROM jsonb_build_array(finalidades[i+1])
  OR entrada#>>'{concesion,garantia_minima}' IS DISTINCT FROM 'alto'
  OR NOT (coalesce(entrada#>'{concesion,obligaciones}','[]'::jsonb)='[]'::jsonb
    OR (i=0 AND entrada#>'{concesion,obligaciones}'='["auditar"]'::jsonb))
  OR entrada->'dimensiones_ambito' IS DISTINCT FROM '["candidato_ref"]'::jsonb
  OR entrada->>'clase_control' IS DISTINCT FROM 'ordinario'
  OR concesion IS DISTINCT FROM nueva->'concesiones'->
    (jsonb_array_length(base#>'{rol,concesiones}')+i)
  THEN RAISE EXCEPTION 'AUT77: descriptor o concesion divergente %',acciones[i+1] USING ERRCODE='42501';END IF;
  hojas:=CASE WHEN i=0 THEN ARRAY['version','orden_guardado','bolsas_actuales',
   'bolsas_nuevas','bolsas_retiradas','revision_requerida','recibo_ref','registrado_en']::text[]
   ELSE ARRAY[]::text[] END;
  IF coalesce(entrada#>'{concesion,campos_permitidos}','[]'::jsonb) IS DISTINCT FROM to_jsonb(hojas)
  THEN RAISE EXCEPTION 'AUT77: campos fuera de contrato %',acciones[i+1] USING ERRCODE='42501';END IF;
  IF p_primera AND (ahora<(entrada->>'vigente_desde')::timestamptz
   OR (entrada->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
    AND ahora>=(entrada->>'vigente_hasta')::timestamptz))
  THEN RAISE EXCEPTION 'AUT77: descriptor caducado %',acciones[i+1] USING ERRCODE='42501';END IF;
  IF entrada->>'vigente_hasta'<>'0001-01-01T00:00:00Z' THEN
   limite_entradas:=LEAST(limite_entradas,(entrada->>'vigente_hasta')::timestamptz);
  END IF;
  normalizadas:=normalizadas||jsonb_build_array(concesion);
 END LOOP;
 IF catalogo->>'vigente_hasta'<>'0001-01-01T00:00:00Z' THEN
  limite_entradas:=LEAST(limite_entradas,(catalogo->>'vigente_hasta')::timestamptz);
 END IF;
 SELECT * INTO rol FROM vec_autorizacion.version_rol
  WHERE version_rol_ref='rol:candidato_bolsa_portal_historial_propio_desarrollo:v'||(base#>>'{rol,version}')
  FOR SHARE;
 IF NOT FOUND OR rol.rol_id IS DISTINCT FROM 'candidato_bolsa_portal_historial_propio_desarrollo'
 OR rol.documento->>'estado' IS DISTINCT FROM 'publicada'
 OR rol.documento->>'nombre' IS DISTINCT FROM 'areaPersonal.miBolsa.rolPortal'
 OR rol.documento->>'publicada_por' IS DISTINCT FROM 'seguridad:desarrollo:no-autoritativa'
 OR rol.version<>1
 OR rol.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(rol.documento),'UTF8')),'hex')
 OR vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(rol.documento) IS DISTINCT FROM base->'rol'
 OR (p_primera AND EXISTS(SELECT 1 FROM vec_autorizacion.version_rol z
   WHERE z.rol_id=rol.rol_id AND z.version>rol.version))
 THEN RAISE EXCEPTION 'AUT77: preimagen del rol divergente' USING ERRCODE='40001';END IF;
 SELECT x.* INTO control FROM vec_autorizacion.control_vigencia_version_rol x
 WHERE x.version_rol_ref=rol.version_rol_ref
 AND x.revision=(base#>>'{control_vigencia,revision}')::numeric FOR SHARE;
 IF NOT FOUND OR control.estado IS DISTINCT FROM 'habilitada'
 OR control.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(control.documento),'UTF8')),'hex')
 OR control.documento IS DISTINCT FROM base->'control_vigencia'
 THEN RAISE EXCEPTION 'AUT77: control o perfil catalogado divergente' USING ERRCODE='40001';END IF;
 -- El cierre nuevo retiene el puntero base hasta COMMIT. Una retirada o
 -- avance concurrente no puede intercalarse entre cotejo y efecto.
 IF p_primera THEN
  SELECT q.revision INTO puntero_control
  FROM vec_autorizacion.control_vigencia_version_rol_actual q
  WHERE q.version_rol_ref=control.version_rol_ref FOR SHARE;
  IF NOT FOUND OR puntero_control.revision IS DISTINCT FROM control.revision
  THEN RAISE EXCEPTION 'AUT77: control base ya no actual' USING ERRCODE='40001';END IF;
 END IF;
 IF nueva->>'nombre' IS DISTINCT FROM base#>>'{rol,nombre}'
 OR nueva->'concesiones' IS DISTINCT FROM (base#>'{rol,concesiones}')||normalizadas
 OR vec_autorizacion.concesiones_positivas_validas(nueva) IS NOT TRUE
 OR jsonb_array_length(base#>'{rol,concesiones}')<>7
 OR NOT ((base#>'{rol,concesiones}') @> '[{"accion":"bolsa.historial_propio.consultar"},
  {"accion":"bolsa.participaciones_propias.consultar"},
  {"accion":"bolsa.participaciones_propias.solicitar_pausa"},
  {"accion":"bolsa.participaciones_propias.solicitar_reactivacion"},
  {"accion":"bolsa.participaciones_propias.responder_llamamiento"},
  {"accion":"bolsa.participaciones_propias.manifestar_disposicion"},
  {"accion":"bolsa.participaciones_propias.confirmar_contacto"}]'::jsonb)
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(base#>'{rol,concesiones}') x
   WHERE x->>'accion'=ANY(acciones))
 OR (p_primera AND EXISTS(SELECT 1 FROM vec_autorizacion.version_rol z
   WHERE z.version_rol_ref=p->>'version_rol_objetivo_ref'))
 THEN RAISE EXCEPTION 'AUT77: delta de dos altas divergente' USING ERRCODE='40001';END IF;
 FOR t IN SELECT value FROM jsonb_array_elements(p->'asignaciones') WITH ORDINALITY x(value,n) ORDER BY n LOOP
  IF t->>'asignacion_ref'=ANY(vistas)
  THEN RAISE EXCEPTION 'AUT77: asignacion duplicada' USING ERRCODE='22023';END IF;
  vistas:=array_append(vistas,t->>'asignacion_ref');original:=t->'documento';
  ambitos:=original->'ambitos';
  IF t->>'asignacion_ref' IS DISTINCT FROM 'asignacion:'||(original->>'asignacion_id')||':v'||(original->>'version')
  OR t->>'huella_sha256' IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(original),'UTF8')),'hex')
  OR original->>'version_rol_ref' IS DISTINCT FROM 'rol:candidato_bolsa_portal_historial_propio_desarrollo:v1'
  OR original->>'estado' IS DISTINCT FROM 'activa'
  OR vec_autorizacion.ambitos_positivos_validos(original) IS NOT TRUE
  OR jsonb_typeof(ambitos) IS DISTINCT FROM 'array' OR jsonb_array_length(ambitos)<>1
  OR ambitos->0->>'clave' IS DISTINCT FROM 'candidato_ref'
  OR jsonb_typeof(ambitos#>'{0,valores}') IS DISTINCT FROM 'array'
  OR jsonb_array_length(ambitos#>'{0,valores}')<>1
  OR (ambitos#>>'{0,valores,0}' ~ '^can_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE
  OR (p_primera AND vec_contexto_actor_v1.acreditar_candidato_externo_v1(
   original->>'principal_id',original->>'perfil_activo_ref',ambitos#>>'{0,valores,0}') IS NOT TRUE)
  THEN RAISE EXCEPTION 'AUT77: asignacion o ambitos invalidos' USING ERRCODE='22023';END IF;
  SELECT x.* INTO asignacion FROM vec_autorizacion.asignacion_perfil_externa x
   WHERE x.asignacion_ref=t->>'asignacion_ref' FOR SHARE;
  IF NOT FOUND OR asignacion.huella_sha256 IS DISTINCT FROM t->>'huella_sha256'
  OR asignacion.documento IS DISTINCT FROM original
  OR asignacion.principal_id IS DISTINCT FROM original->>'principal_id'
  OR asignacion.perfil_activo_ref IS DISTINCT FROM original->>'perfil_activo_ref'
  OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol z
   WHERE z.version_rol_ref=asignacion.version_rol_ref AND z.rol_id=rol.rol_id AND z.version<=rol.version)
  THEN RAISE EXCEPTION 'AUT77: asignacion historica divergente' USING ERRCODE='40001';END IF;
  IF p_primera AND (NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual_externa q
    WHERE q.perfil_activo_ref=asignacion.perfil_activo_ref AND q.asignacion_ref=asignacion.asignacion_ref)
   OR ahora<(asignacion.documento->>'vigente_desde')::timestamptz
   OR ahora>=(asignacion.documento->>'vigente_hasta')::timestamptz
   OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_externa rev
     WHERE rev.asignacion_id=asignacion.asignacion_id AND rev.documento->>'estado'='revocada'))
  THEN RAISE EXCEPTION 'AUT77: asignacion no actual o vigente' USING ERRCODE='40001';END IF;
 END LOOP;
 RETURN jsonb_build_object('vigente_hasta',CASE WHEN isfinite(limite_entradas)
  THEN to_char(limite_entradas AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
  ELSE '0001-01-01T00:00:00Z' END);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.validar_plan_version_rol_bolsa_externa_v1(jsonb,boolean) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.comprobar_postimagen_version_rol_bolsa_externa_v1(p_propuesta text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE c record;r record;cv record;t jsonb;a record;s record;
BEGIN
 SELECT * INTO STRICT c FROM vec_autorizacion.cierre_version_rol_bolsa_externa_v1 WHERE propuesta_ref=p_propuesta;
 SELECT * INTO STRICT r FROM vec_autorizacion.version_rol WHERE version_rol_ref=c.version_rol_ref FOR SHARE;
 SELECT x.* INTO STRICT cv FROM vec_autorizacion.control_vigencia_version_rol x
 WHERE x.version_rol_ref=c.version_rol_ref AND x.revision=1 FOR SHARE;
 IF r.huella_sha256 IS DISTINCT FROM c.rol_sha256 OR r.documento IS DISTINCT FROM c.resultado#>'{recibo,version_rol}'
 OR r.documento->>'estado' IS DISTINCT FROM 'publicada'
 OR cv.revision<>1 OR cv.estado<>'habilitada' OR cv.huella_sha256 IS DISTINCT FROM c.control_sha256
 OR cv.documento IS DISTINCT FROM c.resultado#>'{recibo,control_posterior}'
 THEN RAISE EXCEPTION 'AUT77: replay sin postimagen vigente' USING ERRCODE='40001';END IF;
 FOR t IN SELECT value FROM jsonb_array_elements(c.resultado#>'{recibo,asignaciones}') x LOOP
  SELECT * INTO STRICT a FROM vec_autorizacion.asignacion_perfil_externa
   WHERE asignacion_ref='asignacion:'||(t#>>'{posterior,asignacion_id}')||':v'||(t#>>'{posterior,version}') FOR SHARE;
  SELECT * INTO STRICT s FROM vec_autorizacion.sello_efecto_admin_externo_tx_v1
   WHERE asignacion_ref=a.asignacion_ref FOR SHARE;
  IF a.documento IS DISTINCT FROM t->'posterior'
  OR a.huella_sha256 IS DISTINCT FROM t->>'posterior_sha256'
  OR s.operacion_ref IS DISTINCT FROM c.operacion_ref
  OR s.auditoria_ref IS DISTINCT FROM c.auditoria_ref
  THEN RAISE EXCEPTION 'AUT77: replay sin asignación histórica' USING ERRCODE='40001';END IF;
 END LOOP;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.comprobar_postimagen_version_rol_bolsa_externa_v1(text) FROM PUBLIC;


CREATE FUNCTION vec_autorizacion.aplicar_version_rol_bolsa_externa_v1(
 p_cierre boolean,p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC'
SET lock_timeout='2s' AS $f$
DECLARE m jsonb;mat jsonb;plan jsonb;d jsonb;c jsonb;motivo jsonb;entrada jsonb;efectivos jsonb;
 prop vec_autorizacion.propuesta_version_rol_bolsa_externa_v1;prev vec_autorizacion.cierre_version_rol_bolsa_externa_v1;
 x record;a record;amb jsonb;ambcanon text;ctx text;huella text;material_sha text;
 persona text;perfil text;asignacion text;recurso text;op text;accion text;audiencia text;
 ahora timestamptz(6);caduca timestamptz(6);fecha text;rol jsonb;control jsonb;rol_sha text;control_sha text;
 resultado jsonb;recibo jsonb;acto text;recibo_ref text;claves text[];replay boolean:=false;
 t jsonb;ant record;puntero record;doc jsonb;destino_ref text;destino_sha text;asigs jsonb:='[]'::jsonb;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC' OR current_setting('role')<>'none'
 OR p_cierre IS NULL OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 196608
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
 OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 2 AND 32768
 OR p_motivo IS NULL OR octet_length(p_motivo) NOT BETWEEN 2 AND 65536
 THEN RAISE EXCEPTION 'AUT77: contexto invalido' USING ERRCODE='42501';END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;c:=convert_from(p_capacidad,'UTF8')::jsonb;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_path_exists(m,'$.** ? (@ == null)')
 THEN RAISE EXCEPTION 'AUT77: envelope invalido' USING ERRCODE='22023';END IF;
 SELECT array_agg(k ORDER BY k COLLATE "C") INTO claves FROM jsonb_object_keys(m) k;
 IF NOT p_cierre THEN
  IF claves IS DISTINCT FROM ARRAY['correlacion_ref','esquema','material_canon','material_sha256','plan_sha256']::text[]
  OR m->>'esquema' IS DISTINCT FROM 'administracion_version_rol_bolsa_externa_propuesta_v1'
  OR jsonb_typeof(m->'material_canon') IS DISTINCT FROM 'string' OR octet_length(m->>'material_canon') NOT BETWEEN 1 AND 131072
  THEN RAISE EXCEPTION 'AUT77: propuesta invalida' USING ERRCODE='22023';END IF;
  mat:=(m->>'material_canon')::jsonb;plan:=mat->'Plan';
  IF vec_autorizacion.canon_version_rol_bolsa_externa_v1(mat,'material') IS DISTINCT FROM m->>'material_canon'
  OR encode(sha256(convert_to(m->>'material_canon','UTF8')),'hex') IS DISTINCT FROM m->>'material_sha256'
  OR encode(sha256(convert_to(vec_autorizacion.canon_version_rol_bolsa_externa_v1(plan,'plan'),'UTF8')),'hex') IS DISTINCT FROM m->>'plan_sha256'
  OR mat->>'OperacionRef' !~ '^propuesta_admin:[0-9a-f]{32}$'
  THEN RAISE EXCEPTION 'AUT77: canon o huella divergente' USING ERRCODE='22023';END IF;
  persona:=mat->>'ProponentePersonaRef';perfil:=mat->>'PerfilActivoRef';asignacion:=mat->>'AsignacionPerfilRef';
  recurso:=plan->>'version_rol_objetivo_ref';op:=mat->>'OperacionRef';motivo:=plan->'motivo';
  accion:='administracion.perfiles.version_bolsa.proponer';audiencia:='vec_autorizacion.versionar_rol_bolsa_externa.propuesta.v1';
 ELSE
  IF claves IS DISTINCT FROM ARRAY['actor_perfil_ref','actor_persona_ref','asignacion_ref','correlacion_ref','decision','esquema','motivo','operacion_ref','propuesta_huella_sha256','propuesta_ref']::text[]
  OR m->>'esquema' IS DISTINCT FROM 'administracion_version_rol_bolsa_externa_cierre_v1'
  OR m->>'decision' IS DISTINCT FROM 'aprobada' OR m->>'operacion_ref' !~ '^cierre_admin:[0-9a-f]{32}$'
  OR m->>'propuesta_ref' !~ '^propuesta_admin:[0-9a-f]{32}$' OR m->>'propuesta_huella_sha256' !~ '^[0-9a-f]{64}$'
  THEN RAISE EXCEPTION 'AUT77: cierre invalido' USING ERRCODE='22023';END IF;
  persona:=m->>'actor_persona_ref';perfil:=m->>'actor_perfil_ref';asignacion:=m->>'asignacion_ref';
  recurso:=m->>'propuesta_ref';op:=m->>'operacion_ref';motivo:=m->'motivo';
  accion:='administracion.perfiles.version_bolsa.aprobar';audiencia:='vec_autorizacion.versionar_rol_bolsa_externa.cierre.v1';
 END IF;
 PERFORM vec_autorizacion.canon_gobierno_rol_nuevo_v1(motivo,'motivo');
 IF (persona ~ '^per_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE OR (perfil ~ '^prf_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE
 OR m->>'correlacion_ref' IS NULL OR m->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 OR (convert_from(p_motivo,'UTF8')::jsonb)->'referencia' IS DISTINCT FROM motivo
 OR d->>'principal_id' IS DISTINCT FROM persona OR d->>'perfil_activo_ref' IS DISTINCT FROM perfil
 OR d->>'asignacion_ref' IS DISTINCT FROM asignacion OR d->>'accion' IS DISTINCT FROM accion
 OR d->>'correlacion_ref' IS DISTINCT FROM m->>'correlacion_ref'
 OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
 OR vec_autorizacion.acreditar_version_rol_bolsa_v1(d) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT77: actor o decision ajenos' USING ERRCODE='42501';END IF;
 SELECT q.* INTO STRICT a FROM vec_autorizacion.asignacion_perfil q WHERE q.asignacion_ref=asignacion;
 IF jsonb_typeof(a.documento->'ambitos') IS DISTINCT FROM 'array'
 OR (SELECT count(*) FROM jsonb_array_elements(a.documento->'ambitos'))<>2
 OR (SELECT count(DISTINCT b->>'clave') FROM jsonb_array_elements(a.documento->'ambitos') b)<>2
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(a.documento->'ambitos') b
  WHERE b->>'clave' NOT IN('organizacion_ref','unidad_ref')
  OR (CASE WHEN jsonb_typeof(b->'valores')='array' THEN jsonb_array_length(b->'valores')=1 ELSE false END) IS NOT TRUE)
 THEN RAISE EXCEPTION 'AUT77: ambitos no unitarios' USING ERRCODE='42501';END IF;
 SELECT jsonb_object_agg(b->>'clave',b#>>'{valores,0}') INTO amb FROM jsonb_array_elements(a.documento->'ambitos') b;
 IF (SELECT count(*) FROM jsonb_object_keys(amb))<>2 OR NOT amb ?& ARRAY['organizacion_ref','unidad_ref']
 THEN RAISE EXCEPTION 'AUT77: ambitos divergentes' USING ERRCODE='42501';END IF;
 ambcanon:='{"organizacion_ref":'||vec_autorizacion.json_cadena_canonica_go_admin_v1(amb->>'organizacion_ref')
  ||',"unidad_ref":'||vec_autorizacion.json_cadena_canonica_go_admin_v1(amb->>'unidad_ref')||'}';
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 ctx:='{"ambitos":'||ambcanon||',"atributos":{"material_sha256":"'||material_sha||'"}}';
 huella:=encode(sha256(convert_to(ctx,'UTF8')),'hex');
 IF d->>'recurso_ref' IS DISTINCT FROM recurso OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM huella
 OR c->>'efecto_ref' IS DISTINCT FROM recurso OR c->>'huella_efecto_sha256' IS DISTINCT FROM huella
 THEN RAISE EXCEPTION 'AUT77: recurso no ligado al material' USING ERRCODE='42501';END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_externa_v3_atestada(
 p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.efecto_ref IS DISTINCT FROM recurso OR x.huella_efecto_sha256 IS DISTINCT FROM huella
 THEN RAISE EXCEPTION 'AUT77: consumo divergente' USING ERRCODE='42501';END IF;
 efectivos:=vec_autorizacion.administradores_version_rol_bolsa_v1();
 IF (SELECT count(DISTINCT v->>'persona_ref') FROM jsonb_array_elements(efectivos) v)<2
 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) v WHERE v->>'persona_ref'=persona AND v->>'perfil_ref'=perfil AND v->>'asignacion_ref'=asignacion)
 THEN RAISE EXCEPTION 'AUT77: dos ADMIN actuales necesarios' USING ERRCODE='42501';END IF;
 IF p_cierre THEN
  SELECT * INTO STRICT prop FROM vec_autorizacion.propuesta_version_rol_bolsa_externa_v1 WHERE propuesta_ref=recurso FOR SHARE;
  mat:=convert_from(prop.material,'UTF8')::jsonb;plan:=mat->'Plan';
  IF prop.material_sha256 IS DISTINCT FROM m->>'propuesta_huella_sha256'
  OR prop.proponente_persona_ref=persona OR prop.proponente_perfil_ref=perfil
  OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) v WHERE v->>'persona_ref'=prop.proponente_persona_ref
   AND v->>'perfil_ref'=prop.proponente_perfil_ref AND v->>'asignacion_ref'=prop.asignacion_ref)
  THEN RAISE EXCEPTION 'AUT77: cierre no independiente o proponente revocado' USING ERRCODE='42501';END IF;
 ELSE
  SELECT * INTO prop FROM vec_autorizacion.propuesta_version_rol_bolsa_externa_v1 WHERE propuesta_ref=op FOR SHARE;
  IF FOUND THEN
   IF vec_autorizacion.solicitud_replay_version_rol_bolsa_v1(prop.solicitud,p_material) IS NOT TRUE
   THEN RAISE EXCEPTION 'AUT77: replay de propuesta divergente' USING ERRCODE='23505';END IF;
   replay:=true;
  END IF;
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:version-rol-bolsa:'||(plan#>>'{definicion_nueva,rol_id}'),0));
 IF p_cierre THEN
  SELECT * INTO prev FROM vec_autorizacion.cierre_version_rol_bolsa_externa_v1 WHERE propuesta_ref=prop.propuesta_ref;
  IF FOUND THEN
   IF prev.operacion_ref IS DISTINCT FROM op
   OR vec_autorizacion.solicitud_replay_version_rol_bolsa_v1(prev.solicitud,p_material) IS NOT TRUE
   THEN RAISE EXCEPTION 'AUT77: propuesta cerrada con otro material' USING ERRCODE='23505';END IF;
   PERFORM vec_autorizacion.comprobar_postimagen_version_rol_bolsa_externa_v1(prop.propuesta_ref);
   replay:=true;
  END IF;
 END IF;
 entrada:=vec_autorizacion.validar_plan_version_rol_bolsa_externa_v1(plan,NOT replay);
 IF p_cierre THEN
  IF replay THEN
   resultado:=prev.resultado||jsonb_build_object('auditoria_acceso_ref',x.auditoria_ref);
  ELSE
   ahora:=clock_timestamp();
   IF ahora>=prop.caduca_en OR EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref=prop.version_rol_ref)
   THEN RAISE EXCEPTION 'AUT77: propuesta caducada o versión ocupada' USING ERRCODE='40001';END IF;
   fecha:=vec_autorizacion.fecha_canonica_go_admin_v1(to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
   acto:='acto_admin:'||substr(op,14,32);recibo_ref:='recibo_admin:'||substr(op,14,32);
   rol:=plan->'definicion_nueva'||jsonb_build_object('estado','publicada','publicada_por',persona,'publicada_en',fecha,'retirada_en','0001-01-01T00:00:00Z');
   rol_sha:=encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(rol),'UTF8')),'hex');
   control:=jsonb_build_object('version_rol_ref',prop.version_rol_ref,'revision',1,'estado','habilitada','actualizado_por',persona,'actualizado_en',fecha);
   control_sha:=encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(control),'UTF8')),'hex');
   INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
    VALUES(prop.version_rol_ref,rol->>'rol_id',(rol->>'version')::bigint,rol_sha,ahora,rol);
   INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
    VALUES(prop.version_rol_ref,1,'habilitada',control_sha,ahora,control);
   INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
    VALUES(prop.version_rol_ref,1,ahora,persona,acto);
   FOR t IN SELECT value FROM jsonb_array_elements(plan->'asignaciones') WITH ORDINALITY o(value,n) ORDER BY n LOOP
    SELECT * INTO STRICT puntero FROM vec_autorizacion.asignacion_perfil_actual_externa
     WHERE perfil_activo_ref=t#>>'{documento,perfil_activo_ref}' FOR UPDATE;
    IF puntero.asignacion_ref IS DISTINCT FROM t->>'asignacion_ref'
    THEN RAISE EXCEPTION 'AUT77: puntero CAS divergente' USING ERRCODE='40001';END IF;
    SELECT * INTO STRICT ant FROM vec_autorizacion.asignacion_perfil_externa
     WHERE asignacion_ref=t->>'asignacion_ref' FOR SHARE;
    IF ant.documento IS DISTINCT FROM t->'documento'
    OR ant.huella_sha256 IS DISTINCT FROM t->>'huella_sha256'
    OR ant.documento->>'estado' IS DISTINCT FROM 'activa'
    OR ahora<(ant.documento->>'vigente_desde')::timestamptz
    OR ahora>=(ant.documento->>'vigente_hasta')::timestamptz
    THEN RAISE EXCEPTION 'AUT77: asignación CAS divergente' USING ERRCODE='40001';END IF;
    destino_ref:='asignacion:'||ant.asignacion_id||':v'||(ant.version+1)::text;
    doc:=ant.documento||jsonb_build_object('version',ant.version+1,'version_rol_ref',prop.version_rol_ref,
     'emitida_por',persona,'emitida_en',fecha,'vigente_desde',fecha);
    destino_sha:=encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(doc),'UTF8')),'hex');
    INSERT INTO vec_autorizacion.asignacion_perfil_externa(
     asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
    VALUES(destino_ref,ant.asignacion_id,ant.version+1,ant.perfil_activo_ref,ant.principal_id,
     prop.version_rol_ref,destino_sha,ahora,doc);
    INSERT INTO vec_autorizacion.sello_efecto_admin_externo_tx_v1
     VALUES(destino_ref,txid_current(),op,x.auditoria_ref);
    UPDATE vec_autorizacion.asignacion_perfil_actual_externa SET asignacion_ref=destino_ref,
     actualizada_en=ahora,actualizada_por=persona,acto_ref=acto
     WHERE perfil_activo_ref=ant.perfil_activo_ref AND asignacion_ref=ant.asignacion_ref;
    IF NOT FOUND THEN RAISE EXCEPTION 'AUT77: puntero CAS perdido' USING ERRCODE='40001';END IF;
    asigs:=asigs||jsonb_build_array(jsonb_build_object('anterior',ant.documento,'posterior',doc,
     'anterior_sha256',ant.huella_sha256,'posterior_sha256',destino_sha));
   END LOOP;
   recibo:=jsonb_build_object('acto_ref',acto,'recibo_ref',recibo_ref,'actor_persona_ref',persona,'perfil_activo_ref',perfil,
    'asignacion_perfil_ref',asignacion,'correlacion_ref',m->>'correlacion_ref','motivo',motivo,'auditoria_ref',x.auditoria_ref,'version_rol',rol,'control_posterior',control,'asignaciones',asigs);
   resultado:=jsonb_build_object('operacion_ref',op,'material_canon',convert_from(prop.material,'UTF8'),'propuesta_huella_sha256',prop.material_sha256,
    'decision','aprobada','confirmado_en',ahora,'auditoria_acceso_ref',x.auditoria_ref,'recibo',recibo);
   INSERT INTO vec_autorizacion.cierre_version_rol_bolsa_externa_v1 VALUES(prop.propuesta_ref,op,convert_to(p_material,'UTF8'),prop.version_rol_ref,rol_sha,control_sha,persona,perfil,asignacion,x.auditoria_ref,resultado,ahora);
   INSERT INTO vec_autorizacion.outbox_version_rol_bolsa_externa_v1 VALUES(op,'version_publicada',x.auditoria_ref,ahora);
  END IF;
 ELSE
  IF replay THEN
   IF EXISTS(SELECT 1 FROM vec_autorizacion.cierre_version_rol_bolsa_externa_v1 WHERE propuesta_ref=op) THEN
    PERFORM vec_autorizacion.comprobar_postimagen_version_rol_bolsa_externa_v1(op);
   END IF;
   resultado:=vec_autorizacion.resultado_propuesta_acceso_version_rol_bolsa_v1(prop.resultado,x.auditoria_ref);
  ELSE
   IF EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref=plan->>'version_rol_objetivo_ref')
   THEN RAISE EXCEPTION 'AUT77: versión objetivo ocupada' USING ERRCODE='40001';END IF;
   ahora:=clock_timestamp();
   caduca:=LEAST(ahora+interval '1 day',(a.documento->>'vigente_hasta')::timestamptz,
    NULLIF(entrada->>'vigente_hasta','0001-01-01T00:00:00Z')::timestamptz);
   resultado:=jsonb_build_object('material_canon',m->>'material_canon','huella_sha256',m->>'material_sha256',
    'caduca_en',caduca,'auditoria_acceso_ref',x.auditoria_ref);
   INSERT INTO vec_autorizacion.propuesta_version_rol_bolsa_externa_v1 VALUES(op,convert_to(m->>'material_canon','UTF8'),m->>'material_sha256',convert_to(p_material,'UTF8'),
    persona,perfil,asignacion,plan->>'catalogo_ref',(plan->>'catalogo_version')::integer,plan->>'catalogo_huella_sha256',recurso,ahora,caduca,x.auditoria_ref,resultado);
   INSERT INTO vec_autorizacion.outbox_version_rol_bolsa_externa_v1 VALUES(op,'version_propuesta',x.auditoria_ref,ahora);
  END IF;
 END IF;
 -- La autorización queda revalidada bajo los bloqueos y la barrera de auditoría.
 IF vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
 OR vec_autorizacion.acreditar_version_rol_bolsa_v1(d) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT77: autorizacion final no vigente' USING ERRCODE='42501';END IF;
 PERFORM vec_autorizacion.validar_plan_version_rol_bolsa_externa_v1(plan,NOT replay AND NOT p_cierre);
 efectivos:=vec_autorizacion.administradores_version_rol_bolsa_v1();
 IF (SELECT count(DISTINCT v->>'persona_ref') FROM jsonb_array_elements(efectivos) v)<2
 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) v WHERE v->>'persona_ref'=persona AND v->>'perfil_ref'=perfil AND v->>'asignacion_ref'=asignacion)
 OR (p_cierre AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) v WHERE v->>'persona_ref'=prop.proponente_persona_ref AND v->>'perfil_ref'=prop.proponente_perfil_ref AND v->>'asignacion_ref'=prop.asignacion_ref))
 OR (p_cierre AND NOT replay AND clock_timestamp()>=prop.caduca_en)
 THEN RAISE EXCEPTION 'AUT77: doble control final no vigente' USING ERRCODE='42501';END IF;
 RETURN resultado||jsonb_build_object('estado','permitido','replay',replay);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.aplicar_version_rol_bolsa_externa_v1(boolean,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;


CREATE FUNCTION vec_autorizacion.registrar_fallo_version_rol_bolsa_externa_v1(
 p_material text,p_accion text,p_sqlstate text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on AS $f$
DECLARE m jsonb;h text;corr text;estado text;codigo text;a record;
BEGIN
 h:=encode(sha256(convert_to(coalesce(p_material,''),'UTF8')),'hex');
 corr:='correlacion_'||substr(h,1,32);
 BEGIN
  m:=p_material::jsonb;
  IF jsonb_typeof(m)='object' AND m->>'correlacion_ref' ~ '^correlacion_[0-9a-f]{32}$'
  THEN corr:=m->>'correlacion_ref';END IF;
 EXCEPTION WHEN OTHERS THEN NULL;
 END;
 estado:=CASE WHEN p_sqlstate IN('42501','22023','23505','P0002') THEN 'denegado' ELSE 'error' END;
 codigo:='version_rol_bolsa_'||estado;
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.registrar_intento_version_rol_bolsa_externa_v1(
  jsonb_build_object('tipo_registro','intento_version_rol_bolsa',
   'evento_ref','evento_'||replace(gen_random_uuid()::text,'-',''),
   'operador_login',session_user::text,'solicitud_sha256',h,
   'accion',p_accion,'recurso_ref','solicitud_version_rol_bolsa:'||substr(h,1,32),
   'resultado',estado,'motivo_ref',codigo,'proceso','postgresql',
   'canal','operacion_tecnica_privada','finalidad_ref','gobierno_definiciones_perfiles',
   'correlacion_ref',corr));
 RETURN jsonb_build_object('estado',estado,'codigo',codigo,'auditoria_intento',
  jsonb_build_object('auditoria_ref',a.auditoria_ref,'secuencia',a.secuencia,
   'huella_sha256',a.huella_sha256,'correlacion_ref',a.correlacion_ref,
   'registrada_en',a.registrada_en));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.registrar_fallo_version_rol_bolsa_externa_v1(text,text,text) FROM PUBLIC;


CREATE FUNCTION vec_autorizacion.proponer_version_rol_bolsa_externa_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE resultado jsonb;fallo text;
BEGIN
 BEGIN
  resultado:=vec_autorizacion.aplicar_version_rol_bolsa_externa_v1(false,p_material,p_capacidad,p_decision,
   p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS fallo=RETURNED_SQLSTATE;
  RETURN vec_autorizacion.registrar_fallo_version_rol_bolsa_externa_v1(p_material,
   'administracion.perfiles.version_bolsa.proponer',fallo);
 END;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.proponer_version_rol_bolsa_externa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.cerrar_version_rol_bolsa_externa_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE resultado jsonb;fallo text;
BEGIN
 BEGIN
  resultado:=vec_autorizacion.aplicar_version_rol_bolsa_externa_v1(true,p_material,p_capacidad,p_decision,
   p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS fallo=RETURNED_SQLSTATE;
  RETURN vec_autorizacion.registrar_fallo_version_rol_bolsa_externa_v1(p_material,
   'administracion.perfiles.version_bolsa.aprobar',fallo);
 END;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.cerrar_version_rol_bolsa_externa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;


CREATE OR REPLACE FUNCTION vec_autorizacion.publicar_rol_candidato_externo_v1(p_rol_canonico bytea, p_rol_huella text, p_control_canonico bytea, p_control_huella text, p_revision_esperada numeric, p_control_huella_esperada text, p_actualizada_por text, p_acto_ref text)
 RETURNS TABLE(version_rol_ref text, version bigint, revision numeric, huella_rol text, huella_control text)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE d jsonb; c jsonb; ref text; id text; v bigint; rev numeric;
        existente record; actual record; x jsonb; acciones text[]; esperadas text[];
BEGIN
 IF vec_autorizacion.publicador_candidato_externo_interno_valido_v1() IS NOT TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_rol_canonico IS NULL OR pg_catalog.octet_length(p_rol_canonico) NOT BETWEEN 1 AND 65536
    OR p_control_canonico IS NULL OR pg_catalog.octet_length(p_control_canonico) NOT BETWEEN 1 AND 16384
    OR p_rol_huella !~ '^[0-9a-f]{64}$' OR p_control_huella !~ '^[0-9a-f]{64}$'
    OR pg_catalog.encode(pg_catalog.sha256(p_rol_canonico),'hex') IS DISTINCT FROM p_rol_huella
    OR pg_catalog.encode(pg_catalog.sha256(p_control_canonico),'hex') IS DISTINCT FROM p_control_huella
    OR p_revision_esperada IS NULL OR p_revision_esperada<0 OR p_revision_esperada<>trunc(p_revision_esperada)
    OR (p_revision_esperada=0 AND p_control_huella_esperada IS NOT NULL)
    OR (p_revision_esperada>0 AND p_control_huella_esperada !~ '^[0-9a-f]{64}$')
    OR vec_autorizacion.texto_positivo_valido(p_actualizada_por,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(p_acto_ref,512) IS NOT TRUE
 THEN RAISE EXCEPTION 'publicación de rol externo rechazada' USING ERRCODE='42501'; END IF;
 BEGIN
  d:=pg_catalog.convert_from(p_rol_canonico,'UTF8')::jsonb;
  c:=pg_catalog.convert_from(p_control_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'documento de rol externo inválido' USING ERRCODE='22023'; END;
 id:=d->>'rol_id'; v:=(d->>'version')::bigint;
 ref:='rol:'||id||':v'||v::text; rev:=(c->>'revision')::numeric;
 IF id='candidato_bolsa_portal_historial_propio_desarrollo' AND v IS DISTINCT FROM 1
 THEN RAISE EXCEPTION 'AUT77: versión portal reservada al acto ADMIN' USING ERRCODE='42501';END IF;
 IF id NOT IN ('candidato_bolsa_historial_propio_desarrollo','candidato_bolsa_portal_historial_propio_desarrollo')
    OR vec_autorizacion.rol_candidato_externo_acotado_v1(d) IS NOT TRUE
    OR v IS NULL OR v<1 OR v=9223372036854775807
    OR d->>'estado' IS DISTINCT FROM 'publicada'
    OR d->>'nombre' IS DISTINCT FROM (CASE WHEN id='candidato_bolsa_portal_historial_propio_desarrollo'
       THEN 'areaPersonal.miBolsa.rolPortal' ELSE 'Consulta propia de bolsa en desarrollo' END)
    OR d->>'publicada_por' IS DISTINCT FROM 'seguridad:desarrollo:no-autoritativa'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>(7+CASE WHEN d ? 'retirada_en' THEN 1 ELSE 0 END)
    OR (d ? 'retirada_en' AND d->'retirada_en' IS DISTINCT FROM 'null'::jsonb AND d->'retirada_en' IS DISTINCT FROM '"0001-01-01T00:00:00Z"'::jsonb)
    OR d ?| ARRAY['retirada_por','retirada_ref','motivo_retirada_codigo']
    OR c->>'version_rol_ref' IS DISTINCT FROM ref
    OR c->>'estado' IS DISTINCT FROM 'habilitada'
    OR rev IS DISTINCT FROM p_revision_esperada+1
    OR c->>'actualizado_por' IS DISTINCT FROM p_actualizada_por
    OR c ?| ARRAY['acto_ref','motivo_codigo']
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(c))<>5
    OR pg_catalog.jsonb_typeof(d->'concesiones') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'rol externo inválido' USING ERRCODE='42501'; END IF;
 esperadas:=ARRAY['bolsa.historial_propio.consultar','bolsa.participaciones_propias.consultar'];
 IF id='candidato_bolsa_portal_historial_propio_desarrollo' THEN
  esperadas:=esperadas||ARRAY[
   'bolsa.participaciones_propias.solicitar_pausa',
   'bolsa.participaciones_propias.solicitar_reactivacion',
   'bolsa.participaciones_propias.responder_llamamiento',
   'bolsa.participaciones_propias.manifestar_disposicion',
   'bolsa.participaciones_propias.confirmar_contacto'];
 END IF;
 SELECT pg_catalog.array_agg(e->>'accion' ORDER BY e->>'accion') INTO acciones
 FROM pg_catalog.jsonb_array_elements(d->'concesiones') e;
 IF acciones IS DISTINCT FROM (SELECT pg_catalog.array_agg(a ORDER BY a) FROM pg_catalog.unnest(esperadas) a)
 THEN RAISE EXCEPTION 'concesiones de rol externo incompatibles' USING ERRCODE='42501'; END IF;
 FOR x IN SELECT value FROM pg_catalog.jsonb_array_elements(d->'concesiones') LOOP
  IF pg_catalog.jsonb_typeof(x) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(x) k WHERE k NOT IN
      ('accion','modulo_id','tipo_recurso','finalidades','campos_permitidos','obligaciones','garantia_minima'))<>0
    OR NOT x ?& ARRAY['accion','modulo_id','tipo_recurso','finalidades','garantia_minima']
    OR x->>'modulo_id' IS DISTINCT FROM 'bolsa' OR x->>'garantia_minima' IS DISTINCT FROM 'alto'
    OR x->>'tipo_recurso' IS DISTINCT FROM (CASE WHEN x->>'accion'='bolsa.participaciones_propias.manifestar_disposicion' THEN 'oferta_bolsa' ELSE 'participaciones_candidato' END)
    OR x->'finalidades' IS DISTINCT FROM pg_catalog.to_jsonb(ARRAY[CASE
      WHEN x->>'accion'='bolsa.historial_propio.consultar' THEN 'consulta_historial_propio'
      WHEN x->>'accion'='bolsa.participaciones_propias.consultar' THEN 'consulta_participaciones_propias'
      ELSE 'gestion_participaciones_propias' END])
    OR (x->>'accion'='bolsa.participaciones_propias.consultar'
      AND x->'campos_permitidos' IS DISTINCT FROM '["participaciones_candidato_minimizadas"]'::jsonb)
    OR (x->>'accion'='bolsa.historial_propio.consultar'
      AND x->'campos_permitidos' IS DISTINCT FROM '["contratos_propios","llamamientos_propios","renuncias_propias"]'::jsonb)
    OR (x->>'accion' NOT IN ('bolsa.historial_propio.consultar','bolsa.participaciones_propias.consultar')
      AND COALESCE(x->'campos_permitidos','[]'::jsonb) IS DISTINCT FROM '[]'::jsonb)
    OR COALESCE(x->'obligaciones','[]'::jsonb) IS DISTINCT FROM '[]'::jsonb
  THEN RAISE EXCEPTION 'concesión externa no acotada' USING ERRCODE='42501'; END IF;
 END LOOP;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:rol_candidato_externo:'||id,0));
 SELECT r.version_rol_ref,r.rol_id,r.version,r.huella_sha256,r.documento INTO existente
 FROM vec_autorizacion.version_rol r WHERE r.version_rol_ref=ref FOR SHARE;
 IF FOUND THEN
  IF existente.rol_id IS DISTINCT FROM id OR existente.version IS DISTINCT FROM v
     OR existente.huella_sha256 IS DISTINCT FROM p_rol_huella OR existente.documento IS DISTINCT FROM d
  THEN RAISE EXCEPTION 'rol externo publicado divergente' USING ERRCODE='40001'; END IF;
 ELSE
  INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
  VALUES(ref,id,v,p_rol_huella,(d->>'publicada_en')::timestamptz,d);
 END IF;
 SELECT a.revision,h.huella_sha256 INTO actual
 FROM vec_autorizacion.control_vigencia_version_rol_actual a
 JOIN vec_autorizacion.control_vigencia_version_rol h USING(version_rol_ref,revision)
 WHERE a.version_rol_ref=ref FOR UPDATE OF a;
 IF (NOT FOUND AND (p_revision_esperada<>0 OR p_control_huella_esperada IS NOT NULL))
    OR (FOUND AND (actual.revision IS DISTINCT FROM p_revision_esperada
      OR actual.huella_sha256 IS DISTINCT FROM p_control_huella_esperada))
 THEN RAISE EXCEPTION 'control de rol externo: CAS divergente' USING ERRCODE='40001'; END IF;
 INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
 VALUES(ref,rev,'habilitada',p_control_huella,(c->>'actualizado_en')::timestamptz,c);
 IF p_revision_esperada=0 THEN
  INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
  VALUES(ref,rev,(c->>'actualizado_en')::timestamptz,p_actualizada_por,p_acto_ref);
 ELSE
  UPDATE vec_autorizacion.control_vigencia_version_rol_actual AS puntero SET revision=rev,
   actualizada_en=(c->>'actualizado_en')::timestamptz,actualizada_por=p_actualizada_por,acto_ref=p_acto_ref
  WHERE puntero.version_rol_ref=ref AND puntero.revision=p_revision_esperada;
  IF NOT FOUND THEN RAISE EXCEPTION 'control de rol externo cambiado' USING ERRCODE='40001'; END IF;
 END IF;
 INSERT INTO vec_autorizacion.publicacion_candidato_externo_evento(
  tipo,acto_ref,objeto_ref,version,huella_previa,huella_nueva,publicada_por)
 VALUES('rol_control',p_acto_ref,ref,rev,p_control_huella_esperada,p_control_huella,p_actualizada_por);
 RETURN QUERY SELECT ref,v,rev,p_rol_huella,p_control_huella;
END $function$;

CREATE OR REPLACE FUNCTION vec_autorizacion.publicar_asignacion_candidato_externo_v1(p_documento_canonico bytea, p_huella_sha256 text, p_version_esperada bigint, p_huella_esperada text, p_actualizada_por text, p_acto_ref text)
 RETURNS TABLE(asignacion_ref text, version bigint, huella_sha256 text)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE d jsonb; r record; anterior record; ahora timestamptz:=pg_catalog.clock_timestamp();
        id text; perfil text; principal text; rol text; nueva_version bigint; ref text;
BEGIN
 IF vec_autorizacion.publicador_candidato_externo_interno_valido_v1() IS NOT TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_documento_canonico IS NULL OR pg_catalog.octet_length(p_documento_canonico) NOT BETWEEN 1 AND 65536
    OR p_huella_sha256 !~ '^[0-9a-f]{64}$'
    OR pg_catalog.encode(pg_catalog.sha256(p_documento_canonico),'hex') IS DISTINCT FROM p_huella_sha256
    OR p_version_esperada IS NULL OR p_version_esperada<0
    OR (p_version_esperada=0 AND p_huella_esperada IS NOT NULL)
    OR (p_version_esperada>0 AND p_huella_esperada !~ '^[0-9a-f]{64}$')
    OR vec_autorizacion.texto_positivo_valido(p_actualizada_por,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(p_acto_ref,512) IS NOT TRUE
 THEN RAISE EXCEPTION 'publicación externa rechazada' USING ERRCODE='42501'; END IF;
 BEGIN d:=pg_catalog.convert_from(p_documento_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'documento externo inválido' USING ERRCODE='22023'; END;
 id:=d->>'asignacion_id'; perfil:=d->>'perfil_activo_ref'; principal:=d->>'principal_id'; rol:=d->>'version_rol_ref';
 nueva_version:=p_version_esperada+1; ref:='asignacion:'||id||':v'||nueva_version::text;
 IF rol LIKE 'rol:candidato_bolsa_portal_historial_propio_desarrollo:v%'
 AND rol IS DISTINCT FROM 'rol:candidato_bolsa_portal_historial_propio_desarrollo:v1'
 THEN RAISE EXCEPTION 'AUT77: asignación portal reservada al acto ADMIN' USING ERRCODE='42501';END IF;
 IF vec_autorizacion.texto_positivo_valido(id,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(perfil,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(principal,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(rol,512) IS NOT TRUE
    OR (d->>'version')::bigint IS DISTINCT FROM nueva_version
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>(11+CASE WHEN d ? 'revocada_en' THEN 1 ELSE 0 END)
    OR NOT d ?& ARRAY['asignacion_id','version','perfil_activo_ref','principal_id','version_rol_ref',
      'estado','ambitos','vigente_desde','vigente_hasta','emitida_por','emitida_en']
    OR d->>'estado' IS DISTINCT FROM 'activa'
    OR d ?| ARRAY['revocada_por','revocacion_ref']
    OR (d ? 'revocada_en' AND d->'revocada_en' IS DISTINCT FROM 'null'::jsonb AND d->'revocada_en' IS DISTINCT FROM '"0001-01-01T00:00:00Z"'::jsonb)
    OR pg_catalog.jsonb_array_length(d->'ambitos')<>1
    OR pg_catalog.jsonb_typeof(d->'ambitos'->0) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d->'ambitos'->0))<>2
    OR NOT (d->'ambitos'->0) ?& ARRAY['clave','valores']
    OR d->'ambitos'->0->>'clave' IS DISTINCT FROM 'candidato_ref'
    OR pg_catalog.jsonb_array_length(d->'ambitos'->0->'valores')<>1
    OR d->'ambitos'->0->'valores'->>0 !~ '^can_[A-Za-z0-9_-]+$'
    OR d->>'emitida_por' IS DISTINCT FROM p_actualizada_por
    OR d->>'vigente_desde' IS NULL OR d->>'vigente_hasta' IS NULL
    OR (d->>'vigente_desde')::timestamptz>ahora
    OR (d->>'vigente_hasta')::timestamptz<=ahora
    OR vec_contexto_actor_v1.acreditar_candidato_externo_v1(principal,perfil,d->'ambitos'->0->'valores'->>0) IS NOT TRUE
 THEN RAISE EXCEPTION 'asignación externa inválida o sin provisión' USING ERRCODE='42501'; END IF;
 SELECT vr.rol_id,vr.huella_sha256,vr.documento INTO r FROM vec_autorizacion.version_rol vr
 JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=vr.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
 WHERE vr.version_rol_ref=rol AND c.estado='habilitada' FOR SHARE OF ca;
 IF NOT FOUND OR r.rol_id NOT IN ('candidato_bolsa_historial_propio_desarrollo','candidato_bolsa_portal_historial_propio_desarrollo')
    OR vec_autorizacion.rol_candidato_externo_acotado_v1(r.documento) IS NOT TRUE
 THEN RAISE EXCEPTION 'rol externo no aprobado' USING ERRCODE='42501'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:asignacion_candidato_externo:'||perfil,0));
 SELECT a.asignacion_ref,a.asignacion_id,a.version,a.principal_id,a.huella_sha256
 INTO anterior FROM vec_autorizacion.asignacion_perfil_actual_externa aa
 JOIN vec_autorizacion.asignacion_perfil_externa a ON a.asignacion_ref=aa.asignacion_ref
 WHERE aa.perfil_activo_ref=perfil FOR UPDATE OF aa;
 IF (NOT FOUND AND (p_version_esperada<>0 OR p_huella_esperada IS NOT NULL))
    OR (FOUND AND (anterior.version<>p_version_esperada OR anterior.huella_sha256 IS DISTINCT FROM p_huella_esperada
      OR anterior.asignacion_id IS DISTINCT FROM id OR anterior.principal_id IS DISTINCT FROM principal))
 THEN RAISE EXCEPTION 'asignación externa: CAS divergente' USING ERRCODE='40001'; END IF;
 IF EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_externa vigente
  JOIN vec_autorizacion.version_rol vr ON vr.version_rol_ref=vigente.version_rol_ref
  WHERE vigente.asignacion_ref=anterior.asignacion_ref
   AND vr.rol_id='candidato_bolsa_portal_historial_propio_desarrollo' AND vr.version>=2)
 THEN RAISE EXCEPTION 'AUT77: perfil portal gobernado fuera del publicador técnico' USING ERRCODE='42501';END IF;
 INSERT INTO vec_autorizacion.asignacion_perfil_externa(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES(ref,id,nueva_version,perfil,principal,rol,p_huella_sha256,(d->>'emitida_en')::timestamptz,d);
 IF p_version_esperada=0 THEN
  INSERT INTO vec_autorizacion.asignacion_perfil_actual_externa(perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
  VALUES(perfil,ref,ahora,p_actualizada_por,p_acto_ref);
 ELSE
  UPDATE vec_autorizacion.asignacion_perfil_actual_externa AS puntero
   SET asignacion_ref=ref,actualizada_en=ahora,actualizada_por=p_actualizada_por,acto_ref=p_acto_ref
  WHERE puntero.perfil_activo_ref=perfil AND puntero.asignacion_ref=anterior.asignacion_ref;
  IF NOT FOUND THEN RAISE EXCEPTION 'asignación externa: puntero cambiado' USING ERRCODE='40001'; END IF;
 END IF;
 INSERT INTO vec_autorizacion.publicacion_candidato_externo_evento(
  tipo,acto_ref,objeto_ref,version,huella_previa,huella_nueva,publicada_por)
 VALUES('asignacion',p_acto_ref,ref,nueva_version,p_huella_esperada,p_huella_sha256,p_actualizada_por);
 RETURN QUERY SELECT ref,nueva_version,p_huella_sha256;
END $function$;

DO $tecnica_post$
DECLARE v record;p pg_proc%ROWTYPE;
BEGIN
 FOR v IN SELECT * FROM (VALUES
  ('vec_autorizacion.publicar_rol_candidato_externo_v1(bytea,text,bytea,text,numeric,text,text,text)',
   '958adcfd570ec730964e5bc39c170ac6a0013a1a49a1d12e437c47191bdadd8b','vec.aut77.rol_meta'),
  ('vec_autorizacion.publicar_asignacion_candidato_externo_v1(bytea,text,bigint,text,text,text)',
   '197a1fd1227eb9d0f82b5aaa31bc108cd71aeb05251dc405acbd323775b91f9b','vec.aut77.asg_meta')
 ) AS x(firma,fuente_sha,guc) LOOP
  SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure(v.firma);
  IF NOT FOUND OR (to_jsonb(p)-'prosrc') IS DISTINCT FROM current_setting(v.guc)::jsonb
  OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM v.fuente_sha
  THEN RAISE EXCEPTION 'AUT77: cierre técnico divergente %',v.firma USING ERRCODE='55000';END IF;
 END LOOP;
END $tecnica_post$;

GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_version_rol_bolsa_externa_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion.proponer_version_rol_bolsa_externa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_autorizacion.cerrar_version_rol_bolsa_externa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_autorizacion.resolver_base_version_rol_bolsa_externa_v1(),
 vec_autorizacion.resolver_rol_administrable_v1(text)
 TO vec_admin_version_rol_bolsa_externa_ejecutor;
RESET ROLE;
COMMIT;
