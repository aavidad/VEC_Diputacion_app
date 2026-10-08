\set ON_ERROR_STOP on
-- CA32. Faceta cifrada de Persona, sin nombre claro ni autoridad de identidad.
-- Dependencia nominal AD aún pendiente de reserva/consenso: no instalar hasta
-- disponer de las dos fachadas revisadas. No presta acciones de otras facetas.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:000032',0));
DO $pre$
DECLARE firma text;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN
  RAISE EXCEPTION 'CA32: PARO clave=superusuario actual=false esperado=true' USING ERRCODE='55000'; END IF;
 IF to_regclass('vec_contexto_actor_v1.denominacion_persona_version_v1') IS NOT NULL
 OR to_regrole('vec_persona_denominacion_ejecutor') IS NOT NULL THEN
  RAISE EXCEPTION 'CA32: PARO clave=pieza_instalada actual=true esperado=false' USING ERRCODE='55000'; END IF;
 FOREACH firma IN ARRAY ARRAY[
 'vec_autorizacion_atestada_v3.registrar_y_consumir_denominacion_persona_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
 'vec_autorizacion_atestada_v3.cotejar_consumo_denominacion_persona_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)'] LOOP
  IF to_regprocedure(firma) IS NULL THEN
   RAISE EXCEPTION 'CA32: PARO clave=dependencia_nominal_AD funcion=% actual=false esperado=true',firma USING ERRCODE='55000'; END IF;
 END LOOP;
 IF to_regclass('vec_contexto_actor_v1.persona_actual') IS NULL
 OR to_regclass('vec_contexto_actor_v1.procedencias') IS NULL THEN
  RAISE EXCEPTION 'CA32: PARO clave=autoridad_persona_procedencia actual=false esperado=true' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;

CREATE FUNCTION vec_contexto_actor_v1.referencia_denominacion_v1(v text)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT v IS NOT NULL AND octet_length(v) BETWEEN 3 AND 128 AND v ~ '^[A-Za-z0-9_:-]+$'
$f$;
CREATE FUNCTION vec_contexto_actor_v1.base64_denominacion_v1(v bytea)
RETURNS text LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT replace(encode(v,'base64'),chr(10),'')
$f$;

-- Datos gobernados de configuración; sin claves, LOGIN, semillas ni permisos.
-- Cada tupla es inmutable y explicita la norma y las versiones del KMS.
CREATE TABLE vec_contexto_actor_v1.config_denominacion_persona_v1(
 configuracion_ref text PRIMARY KEY CHECK(vec_contexto_actor_v1.referencia_denominacion_v1(configuracion_ref)),
 ambito_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_denominacion_v1(ambito_ref)),
 organizacion_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_denominacion_v1(organizacion_ref)),
 unidad_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_denominacion_v1(unidad_ref)),
 norma_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_denominacion_v1(norma_ref)),
 norma_sha256 text NOT NULL CHECK(norma_sha256 ~ '^[0-9a-f]{64}$'),
 clave_cifrado_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_denominacion_v1(clave_cifrado_ref)),
 clave_indice_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_denominacion_v1(clave_indice_ref)),
 max_bytes integer NOT NULL CHECK(max_bytes BETWEEN 1 AND 4096),
 max_tokens integer NOT NULL CHECK(max_tokens BETWEEN 1 AND 32),
 vigente_desde timestamptz NOT NULL CHECK(vec_contexto_actor_v1.instante_valido(vigente_desde)),
 vigente_hasta timestamptz NOT NULL CHECK(vec_contexto_actor_v1.instante_valido(vigente_hasta) AND vigente_hasta>vigente_desde),
 CHECK(clave_cifrado_ref<>clave_indice_ref),
 UNIQUE(ambito_ref,norma_ref,norma_sha256,clave_cifrado_ref,clave_indice_ref)
);
CREATE TABLE vec_contexto_actor_v1.denominacion_persona_version_v1(
 persona_ref text NOT NULL,version numeric(16,0) NOT NULL CHECK(version BETWEEN 1 AND 9007199254740991),
 version_esperada numeric(16,0) NOT NULL CHECK(version_esperada=version-1),
 persona_version numeric(20,0) NOT NULL,
 procedencia_ref text NOT NULL,procedencia_version numeric(20,0) NOT NULL,
 procedencia_sha256 text NOT NULL,procedencia_autoridad text NOT NULL,
 configuracion_ref text NOT NULL REFERENCES vec_contexto_actor_v1.config_denominacion_persona_v1,
 sobre_canonico bytea NOT NULL,sobre_sha256 text NOT NULL,
 material_canonico bytea NOT NULL,material_sha256 text NOT NULL,
 decision_ref text NOT NULL,auditoria_ref text NOT NULL,recibo jsonb NOT NULL,
 registrada_en timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(persona_ref,version),
 FOREIGN KEY(persona_ref,persona_version) REFERENCES vec_contexto_actor_v1.persona_versiones(persona_ref,version),
 FOREIGN KEY(procedencia_ref,procedencia_version,procedencia_sha256,procedencia_autoridad)
 REFERENCES vec_contexto_actor_v1.procedencias(procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad),
 CHECK(sobre_sha256=encode(sha256(sobre_canonico),'hex')),
 CHECK(material_sha256=encode(sha256(material_canonico),'hex'))
);
CREATE TABLE vec_contexto_actor_v1.denominacion_persona_actual_v1(
 persona_ref text PRIMARY KEY,version numeric(16,0) NOT NULL,
 FOREIGN KEY(persona_ref,version) REFERENCES vec_contexto_actor_v1.denominacion_persona_version_v1
);
CREATE TABLE vec_contexto_actor_v1.outbox_denominacion_persona_v1(
 persona_ref text NOT NULL,version numeric(16,0) NOT NULL,
 evento text NOT NULL CHECK(evento='denominacion_persona_publicada'),
 auditoria_ref text NOT NULL,registrada_en timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(persona_ref,version),
 FOREIGN KEY(persona_ref,version) REFERENCES vec_contexto_actor_v1.denominacion_persona_version_v1
);
-- Acuse operativo de la lectura: no es una tabla de auditoría ni otro append.
-- El único evento nominal es el consumo común que devuelve AD.
CREATE TABLE vec_contexto_actor_v1.lectura_denominacion_persona_v1(
 decision_ref text PRIMARY KEY,material_canonico bytea NOT NULL,
 consumo jsonb NOT NULL,resultado jsonb NOT NULL
);
CREATE FUNCTION vec_contexto_actor_v1.avanzar_denominacion_persona_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 IF TG_OP='DELETE' OR (TG_OP='INSERT' AND NEW.version<>1)
 OR (TG_OP='UPDATE' AND (NEW.persona_ref IS DISTINCT FROM OLD.persona_ref OR NEW.version IS DISTINCT FROM OLD.version+1))
 THEN RAISE EXCEPTION 'CA32: avance rechazado' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $f$;
CREATE TRIGGER avance BEFORE INSERT OR UPDATE OR DELETE ON vec_contexto_actor_v1.denominacion_persona_actual_v1 FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.avanzar_denominacion_persona_v1();
DO $historia$
DECLARE nombre text;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['config_denominacion_persona_v1','denominacion_persona_version_v1','denominacion_persona_actual_v1','outbox_denominacion_persona_v1','lectura_denominacion_persona_v1'] LOOP
  EXECUTE format('ALTER TABLE vec_contexto_actor_v1.%I ENABLE ROW LEVEL SECURITY',nombre);
  EXECUTE format('ALTER TABLE vec_contexto_actor_v1.%I FORCE ROW LEVEL SECURITY',nombre);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_contexto_actor_v1.%I FOR ALL TO vec_contexto_actor_v1_propietario USING(current_user=%L) WITH CHECK(current_user=%L)',nombre,'vec_contexto_actor_v1_propietario','vec_contexto_actor_v1_propietario');
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_contexto_actor_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado()',nombre);
  IF nombre<>'denominacion_persona_actual_v1' THEN
   EXECUTE format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.%I FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia()',nombre);
  END IF;
  EXECUTE format('REVOKE ALL ON TABLE vec_contexto_actor_v1.%I FROM PUBLIC',nombre);
  EXECUTE format('REVOKE ALL ON TYPE vec_contexto_actor_v1.%I FROM PUBLIC',nombre);
 END LOOP;
END $historia$;

CREATE FUNCTION vec_contexto_actor_v1.canon_sobre_denominacion_persona_v1(s jsonb)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE i jsonb:=s->'Indice';t jsonb;anterior bytea;actual bytea;tokens text:='';k text;v text;
BEGIN
 IF jsonb_typeof(s) IS DISTINCT FROM 'object' OR jsonb_typeof(i) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(s))<>7 OR (SELECT count(*) FROM jsonb_object_keys(i))<>5
 OR NOT s ?& ARRAY['Esquema','PersonaRef','ClaveRef','Version','Nonce','Cifrado','Indice']
 OR NOT i ?& ARRAY['AmbitoRef','NormaRef','NormaSHA256','ClaveRef','Tokens']
 OR s->>'Esquema' IS DISTINCT FROM 'vec.persona.denominacion.aead.v1'
 OR vec_contexto_actor_v1.referencia_valida(s->>'PersonaRef','per_') IS NOT TRUE OR length(s->>'PersonaRef')>128
 OR jsonb_typeof(s->'Version') IS DISTINCT FROM 'number' OR s->>'Version' !~ '^[1-9][0-9]{0,15}$'
 OR (s->>'Version')::numeric>9007199254740991
 OR jsonb_typeof(i->'NormaSHA256') IS DISTINCT FROM 'string' OR i->>'NormaSHA256' !~ '^[0-9a-f]{64}$'
 OR jsonb_typeof(i->'Tokens') IS DISTINCT FROM 'array' OR jsonb_array_length(i->'Tokens') NOT BETWEEN 1 AND 32
 THEN RAISE EXCEPTION 'CA32: sobre inválido' USING ERRCODE='22023'; END IF;
 FOREACH k IN ARRAY ARRAY['ClaveRef'] LOOP
  IF jsonb_typeof(s->k) IS DISTINCT FROM 'string' OR vec_contexto_actor_v1.referencia_denominacion_v1(s->>k) IS NOT TRUE THEN RAISE EXCEPTION 'CA32: referencia inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['AmbitoRef','NormaRef','ClaveRef'] LOOP
  IF jsonb_typeof(i->k) IS DISTINCT FROM 'string' OR vec_contexto_actor_v1.referencia_denominacion_v1(i->>k) IS NOT TRUE THEN RAISE EXCEPTION 'CA32: referencia inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['Nonce','Cifrado'] LOOP
  v:=s->>k;
  IF jsonb_typeof(s->k) IS DISTINCT FROM 'string' OR v IS NULL OR v!~'^[A-Za-z0-9+/]+={0,2}$'
  OR vec_contexto_actor_v1.base64_denominacion_v1(decode(v,'base64')) IS DISTINCT FROM v
  OR (k='Nonce' AND octet_length(decode(v,'base64'))<>12)
  OR (k='Cifrado' AND octet_length(decode(v,'base64')) NOT BETWEEN 17 AND 4112)
  THEN RAISE EXCEPTION 'CA32: cifrado inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOR t IN SELECT value FROM jsonb_array_elements(i->'Tokens') LOOP
  v:=t#>>'{}';actual:=decode(v,'base64');
  IF jsonb_typeof(t) IS DISTINCT FROM 'string' OR octet_length(actual)<>32
  OR vec_contexto_actor_v1.base64_denominacion_v1(actual) IS DISTINCT FROM v
  OR (anterior IS NOT NULL AND anterior>=actual) THEN RAISE EXCEPTION 'CA32: índice inválido' USING ERRCODE='22023'; END IF;
  tokens:=tokens||CASE WHEN tokens='' THEN '' ELSE ',' END||to_jsonb(v)::text;anterior:=actual;
 END LOOP;
 RETURN '{"Esquema":"vec.persona.denominacion.aead.v1","PersonaRef":'||to_jsonb(s->>'PersonaRef')::text||',"ClaveRef":'||to_jsonb(s->>'ClaveRef')::text||',"Version":'||(s->>'Version')||',"Nonce":'||to_jsonb(s->>'Nonce')::text||',"Cifrado":'||to_jsonb(s->>'Cifrado')::text||',"Indice":{"AmbitoRef":'||to_jsonb(i->>'AmbitoRef')::text||',"NormaRef":'||to_jsonb(i->>'NormaRef')::text||',"NormaSHA256":'||to_jsonb(i->>'NormaSHA256')::text||',"ClaveRef":'||to_jsonb(i->>'ClaveRef')::text||',"Tokens":['||tokens||']}}';
END $f$;
CREATE FUNCTION vec_contexto_actor_v1.validar_material_denominacion_persona_v1(p_material text,p_publicar boolean)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE m jsonb;a jsonb;can text;sobre text;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 65536 OR p_publicar IS NULL THEN RAISE EXCEPTION 'CA32: material inválido' USING ERRCODE='22023'; END IF;
 m:=p_material::jsonb;a:=m->'ambitos';
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_typeof(a) IS DISTINCT FROM 'object'
 OR jsonb_path_exists(m,'$.** ? (@ == null)') OR (SELECT count(*) FROM jsonb_object_keys(a))<>2
 OR NOT a ?& ARRAY['organizacion_ref','unidad_ref']
 OR vec_contexto_actor_v1.referencia_denominacion_v1(a->>'organizacion_ref') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_denominacion_v1(a->>'unidad_ref') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(m->>'persona_ref','per_') IS NOT TRUE OR length(m->>'persona_ref')>128
 THEN RAISE EXCEPTION 'CA32: material inválido' USING ERRCODE='22023'; END IF;
 can:='{"esquema":'||to_jsonb(m->>'esquema')::text||',"persona_ref":'||to_jsonb(m->>'persona_ref')::text;
 IF p_publicar THEN
  IF (SELECT count(*) FROM jsonb_object_keys(m))<>10 OR NOT m ?& ARRAY['esquema','persona_ref','version_esperada','procedencia_ref','procedencia_version','procedencia_sha256','procedencia_autoridad','sobre_sha256','sobre','ambitos']
  OR m->>'esquema' IS DISTINCT FROM 'vec.persona.denominacion.publicar.v1'
  OR jsonb_typeof(m->'version_esperada') IS DISTINCT FROM 'number' OR m->>'version_esperada' !~ '^(0|[1-9][0-9]{0,15})$' OR (m->>'version_esperada')::numeric>=9007199254740991
  OR jsonb_typeof(m->'procedencia_version') IS DISTINCT FROM 'number' OR m->>'procedencia_version' !~ '^[1-9][0-9]{0,15}$' OR (m->>'procedencia_version')::numeric>9007199254740991
  OR vec_contexto_actor_v1.referencia_valida(m->>'procedencia_ref','prc_') IS NOT TRUE OR length(m->>'procedencia_ref')>128
  OR m->>'procedencia_sha256' !~ '^[0-9a-f]{64}$' OR m->>'procedencia_autoridad' NOT IN('no_autoritativa','autoridad_maestra_acreditada')
  THEN RAISE EXCEPTION 'CA32: publicación inválida' USING ERRCODE='22023'; END IF;
  sobre:=vec_contexto_actor_v1.canon_sobre_denominacion_persona_v1(m->'sobre');
  IF m->>'persona_ref' IS DISTINCT FROM m#>>'{sobre,PersonaRef}' OR (m->>'version_esperada')::numeric+1 IS DISTINCT FROM (m#>>'{sobre,Version}')::numeric
  OR m->>'sobre_sha256' IS DISTINCT FROM encode(sha256(convert_to(sobre,'UTF8')),'hex') THEN RAISE EXCEPTION 'CA32: sobre divergente' USING ERRCODE='22023'; END IF;
  can:=can||',"version_esperada":'||(m->>'version_esperada')||',"procedencia_ref":'||to_jsonb(m->>'procedencia_ref')::text||',"procedencia_version":'||(m->>'procedencia_version')||',"procedencia_sha256":'||to_jsonb(m->>'procedencia_sha256')::text||',"procedencia_autoridad":'||to_jsonb(m->>'procedencia_autoridad')::text||',"sobre_sha256":'||to_jsonb(m->>'sobre_sha256')::text||',"sobre":'||sobre;
 ELSE
  IF (SELECT count(*) FROM jsonb_object_keys(m))<>4 OR NOT m ?& ARRAY['esquema','persona_ref','version','ambitos']
  OR m->>'esquema' IS DISTINCT FROM 'vec.persona.denominacion.leer.v1'
  OR jsonb_typeof(m->'version') IS DISTINCT FROM 'number' OR m->>'version' !~ '^[1-9][0-9]{0,15}$' OR (m->>'version')::numeric>9007199254740991 THEN RAISE EXCEPTION 'CA32: lectura inválida' USING ERRCODE='22023'; END IF;
  can:=can||',"version":'||(m->>'version');
 END IF;
 can:=can||',"ambitos":{"organizacion_ref":'||to_jsonb(a->>'organizacion_ref')::text||',"unidad_ref":'||to_jsonb(a->>'unidad_ref')::text||'}}';
 IF can IS DISTINCT FROM p_material THEN RAISE EXCEPTION 'CA32: canon divergente' USING ERRCODE='22023'; END IF;
 RETURN m;
END $f$;

CREATE FUNCTION vec_contexto_actor_v1.bloquear_persona_denominacion_v1(p_ref text)
RETURNS numeric LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE p record;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' THEN RAISE EXCEPTION 'CA32: transacción incompatible' USING ERRCODE='25000'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 SELECT v.* INTO STRICT p FROM vec_contexto_actor_v1.persona_actual a JOIN vec_contexto_actor_v1.persona_versiones v USING(persona_ref,version) WHERE a.persona_ref=p_ref FOR SHARE OF a;
 IF p.estado<>'activo' OR clock_timestamp()<p.vigente_desde OR clock_timestamp()>=p.vigente_hasta THEN RAISE EXCEPTION 'CA32: Persona no vigente' USING ERRCODE='42501'; END IF;
 RETURN p.version;
END $f$;
CREATE FUNCTION vec_contexto_actor_v1.configurar_sobre_denominacion_persona_v1(s jsonb,a jsonb)
RETURNS vec_contexto_actor_v1.config_denominacion_persona_v1 LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE c vec_contexto_actor_v1.config_denominacion_persona_v1;
BEGIN
 PERFORM vec_contexto_actor_v1.canon_sobre_denominacion_persona_v1(s);
 SELECT * INTO STRICT c FROM vec_contexto_actor_v1.config_denominacion_persona_v1 WHERE ambito_ref=s#>>'{Indice,AmbitoRef}' AND norma_ref=s#>>'{Indice,NormaRef}' AND norma_sha256=s#>>'{Indice,NormaSHA256}' AND clave_cifrado_ref=s->>'ClaveRef' AND clave_indice_ref=s#>>'{Indice,ClaveRef}' FOR SHARE;
 IF c.organizacion_ref IS DISTINCT FROM a->>'organizacion_ref' OR c.unidad_ref IS DISTINCT FROM a->>'unidad_ref'
 OR clock_timestamp()<c.vigente_desde OR clock_timestamp()>=c.vigente_hasta
 OR octet_length(decode(s->>'Cifrado','base64'))>c.max_bytes+16 OR jsonb_array_length(s#>'{Indice,Tokens}')>c.max_tokens
 THEN RAISE EXCEPTION 'CA32: configuración no vigente' USING ERRCODE='42501'; END IF;
 RETURN c;
END $f$;

CREATE FUNCTION vec_contexto_actor_v1.exigir_runtime_denominacion_persona_v1()
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE l record;g record;fs oid[];ns oid:=to_regnamespace('vec_contexto_actor_v1');db oid;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' OR current_setting('role')<>'none' THEN RAISE EXCEPTION 'CA32: runtime no acreditado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT l FROM pg_roles WHERE rolname=session_user;
 SELECT * INTO STRICT g FROM pg_roles WHERE rolname='vec_persona_denominacion_ejecutor';
 SELECT oid INTO STRICT db FROM pg_database WHERE datname=current_database();
 fs:=ARRAY[to_regprocedure('vec_contexto_actor_v1.acreditar_runtime_denominacion_persona_v1()'),to_regprocedure('vec_contexto_actor_v1.publicar_denominacion_persona_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),to_regprocedure('vec_contexto_actor_v1.leer_denominacion_persona_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),to_regprocedure('vec_contexto_actor_v1.validar_acuse_denominacion_persona_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)'),to_regprocedure('vec_contexto_actor_v1.revalidar_lectura_denominacion_persona_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)')];
 IF NOT l.rolcanlogin OR NOT l.rolinherit OR l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
 OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
 OR (SELECT count(*)FROM pg_auth_members WHERE member=l.oid)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=l.oid AND roleid=g.oid AND NOT admin_option AND inherit_option AND NOT set_option)
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=g.oid)
 OR EXISTS(SELECT 1 FROM pg_db_role_setting WHERE setrole IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_shdepend WHERE refclassid='pg_authid'::regclass AND refobjid=l.oid)
 OR array_position(fs,NULL) IS NOT NULL
 OR NOT coalesce((SELECT count(*)=7 AND bool_and(deptype='a' AND objsubid=0 AND ((classid='pg_database'::regclass AND objid=db)OR(classid='pg_namespace'::regclass AND objid=ns)OR(classid='pg_proc'::regclass AND objid=ANY(fs))))FROM pg_shdepend WHERE refclassid='pg_authid'::regclass AND refobjid=g.oid),false)
 -- pg_shdepend enumera objetos, no el privilegio ni su grant option.
 OR NOT coalesce((SELECT count(*)=1 AND bool_and(a.privilege_type='CONNECT' AND NOT a.is_grantable)FROM pg_database d CROSS JOIN LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba)))a WHERE d.oid=db AND a.grantee=g.oid),false)
 OR NOT coalesce((SELECT count(*)=1 AND bool_and(a.privilege_type='USAGE' AND NOT a.is_grantable)FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner)))a WHERE n.oid=ns AND a.grantee=g.oid),false)
 OR NOT coalesce((SELECT count(*)=5 AND count(DISTINCT p.oid)=5 AND bool_and(a.privilege_type='EXECUTE' AND NOT a.is_grantable)FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner)))a WHERE p.oid=ANY(fs) AND a.grantee=g.oid),false)
 OR EXISTS(SELECT 1 FROM pg_database d CROSS JOIN LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba)))a WHERE d.oid=db AND a.grantee=l.oid)
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner)))a WHERE n.oid=ns AND a.grantee=l.oid)
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner)))a WHERE p.oid=ANY(fs) AND a.grantee=l.oid)
 OR has_schema_privilege(l.oid,ns,'CREATE') OR has_database_privilege(l.oid,db,'CREATE,TEMP')
 OR EXISTS(SELECT 1 FROM pg_default_acl d LEFT JOIN LATERAL aclexplode(coalesce(d.defaclacl,'{}'::aclitem[]))a ON true WHERE d.defaclrole IN(l.oid,g.oid) OR a.grantee IN(l.oid,g.oid)OR a.grantor IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_policy WHERE l.oid=ANY(polroles) OR g.oid=ANY(polroles))
 OR EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec\_%' ESCAPE '\' AND c.relkind IN('r','p','v','m','S') AND (has_table_privilege(l.oid,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')OR has_any_column_privilege(l.oid,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))
 OR to_regprocedure('vec_autorizacion.validar_administrador_denominacion_persona_v1(jsonb,jsonb)') IS NULL
 THEN RAISE EXCEPTION 'CA32: runtime no acreditado' USING ERRCODE='42501'; END IF;
END $f$;
CREATE FUNCTION vec_contexto_actor_v1.acreditar_runtime_denominacion_persona_v1()
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN PERFORM vec_contexto_actor_v1.exigir_runtime_denominacion_persona_v1();RETURN true;END $f$;

CREATE FUNCTION vec_contexto_actor_v1.publicar_denominacion_persona_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET row_security=on AS $f$
DECLARE m jsonb;v numeric;cfg record;anterior record;actual numeric:=0;consumo record;recibo jsonb;sobre text;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_denominacion_persona_v1();
 m:=vec_contexto_actor_v1.validar_material_denominacion_persona_v1(p_material,true);
 v:=vec_contexto_actor_v1.bloquear_persona_denominacion_v1(m->>'persona_ref');
 SELECT * INTO STRICT cfg FROM vec_contexto_actor_v1.configurar_sobre_denominacion_persona_v1(m->'sobre',m->'ambitos');
 PERFORM 1 FROM vec_contexto_actor_v1.procedencias WHERE procedencia_ref=m->>'procedencia_ref' AND procedencia_version=(m->>'procedencia_version')::numeric AND procedencia_huella_sha256=m->>'procedencia_sha256' AND procedencia_autoridad=m->>'procedencia_autoridad' FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'CA32: procedencia divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_denominacion_persona_v3_atestada(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.decision_ref IS NULL OR consumo.auditoria_ref IS NULL THEN RAISE EXCEPTION 'CA32: consumo incompleto' USING ERRCODE='55000'; END IF;
 SELECT * INTO anterior FROM vec_contexto_actor_v1.denominacion_persona_version_v1 WHERE persona_ref=m->>'persona_ref' AND version=(m->>'version_esperada')::numeric+1;
 IF FOUND THEN
  IF anterior.material_canonico IS DISTINCT FROM convert_to(p_material,'UTF8') THEN RAISE EXCEPTION 'CA32: replay divergente' USING ERRCODE='23505'; END IF;
  -- El efecto conserva recibo, sobre y nonce originales; el consumo actual
  -- debe ser nuevo para auditar la recuperación funcional de la publicación.
  IF consumo.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'CA32: replay sin acceso nuevo' USING ERRCODE='42501'; END IF;
  PERFORM vec_contexto_actor_v1.bloquear_persona_denominacion_v1(m->>'persona_ref');
  PERFORM vec_contexto_actor_v1.configurar_sobre_denominacion_persona_v1(m->'sobre',m->'ambitos');
  RETURN anterior.recibo;
 END IF;
 IF consumo.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'CA32: publicación sin consumo nuevo' USING ERRCODE='42501'; END IF;
 SELECT version INTO actual FROM vec_contexto_actor_v1.denominacion_persona_actual_v1 WHERE persona_ref=m->>'persona_ref' FOR UPDATE;
 actual:=coalesce(actual,0);
 IF actual IS DISTINCT FROM (m->>'version_esperada')::numeric THEN RAISE EXCEPTION 'CA32: CAS obsoleto' USING ERRCODE='40001'; END IF;
 PERFORM vec_contexto_actor_v1.bloquear_persona_denominacion_v1(m->>'persona_ref');
 PERFORM vec_contexto_actor_v1.configurar_sobre_denominacion_persona_v1(m->'sobre',m->'ambitos');
 sobre:=vec_contexto_actor_v1.canon_sobre_denominacion_persona_v1(m->'sobre');
 recibo:=jsonb_build_object('persona_ref',m->>'persona_ref','procedencia_ref',m->>'procedencia_ref','sobre_sha256',m->>'sobre_sha256','auditoria_ref',consumo.auditoria_ref,'version',actual+1);
 INSERT INTO vec_contexto_actor_v1.denominacion_persona_version_v1(persona_ref,version,version_esperada,persona_version,procedencia_ref,procedencia_version,procedencia_sha256,procedencia_autoridad,configuracion_ref,sobre_canonico,sobre_sha256,material_canonico,material_sha256,decision_ref,auditoria_ref,recibo)
 VALUES(m->>'persona_ref',actual+1,actual,v,m->>'procedencia_ref',(m->>'procedencia_version')::numeric,m->>'procedencia_sha256',m->>'procedencia_autoridad',cfg.configuracion_ref,convert_to(sobre,'UTF8'),m->>'sobre_sha256',convert_to(p_material,'UTF8'),encode(sha256(convert_to(p_material,'UTF8')),'hex'),consumo.decision_ref,consumo.auditoria_ref,recibo);
 IF actual=0 THEN INSERT INTO vec_contexto_actor_v1.denominacion_persona_actual_v1 VALUES(m->>'persona_ref',1);
 ELSE UPDATE vec_contexto_actor_v1.denominacion_persona_actual_v1 SET version=actual+1 WHERE persona_ref=m->>'persona_ref' AND version=actual;
  IF NOT FOUND THEN RAISE EXCEPTION 'CA32: CAS perdido' USING ERRCODE='40001'; END IF;
 END IF;
 INSERT INTO vec_contexto_actor_v1.outbox_denominacion_persona_v1(persona_ref,version,evento,auditoria_ref)VALUES(m->>'persona_ref',actual+1,'denominacion_persona_publicada',consumo.auditoria_ref);
 RETURN recibo;
END $f$;
CREATE FUNCTION vec_contexto_actor_v1.leer_denominacion_persona_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET row_security=on AS $f$
DECLARE m jsonb;v record;consumo record;prev record;salida jsonb;d jsonb;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_denominacion_persona_v1();
 m:=vec_contexto_actor_v1.validar_material_denominacion_persona_v1(p_material,false);
 PERFORM vec_contexto_actor_v1.bloquear_persona_denominacion_v1(m->>'persona_ref');
 SELECT x.* INTO STRICT v FROM vec_contexto_actor_v1.denominacion_persona_actual_v1 a JOIN vec_contexto_actor_v1.denominacion_persona_version_v1 x USING(persona_ref,version)WHERE a.persona_ref=m->>'persona_ref' AND a.version=(m->>'version')::numeric FOR SHARE OF a;
 PERFORM vec_contexto_actor_v1.configurar_sobre_denominacion_persona_v1(convert_from(v.sobre_canonico,'UTF8')::jsonb,m->'ambitos');
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_denominacion_persona_v3_atestada(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.decision_ref IS NULL OR consumo.auditoria_ref IS NULL OR consumo.consumo_huella_sha256 IS NULL THEN RAISE EXCEPTION 'CA32: consumo incompleto' USING ERRCODE='55000'; END IF;
 SELECT * INTO prev FROM vec_contexto_actor_v1.lectura_denominacion_persona_v1 WHERE decision_ref=consumo.decision_ref;
 IF FOUND THEN
  IF prev.material_canonico IS DISTINCT FROM convert_to(p_material,'UTF8') OR (prev.consumo-'consumo_nuevo') IS DISTINCT FROM (to_jsonb(consumo)-'consumo_nuevo') THEN RAISE EXCEPTION 'CA32: acuse divergente' USING ERRCODE='23505'; END IF;
  PERFORM vec_contexto_actor_v1.bloquear_persona_denominacion_v1(m->>'persona_ref');
  PERFORM vec_contexto_actor_v1.configurar_sobre_denominacion_persona_v1(convert_from(v.sobre_canonico,'UTF8')::jsonb,m->'ambitos');
  RETURN prev.resultado;
 END IF;
 IF consumo.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'CA32: lectura sin consumo nuevo' USING ERRCODE='42501'; END IF;
 PERFORM vec_contexto_actor_v1.bloquear_persona_denominacion_v1(m->>'persona_ref');
 PERFORM vec_contexto_actor_v1.configurar_sobre_denominacion_persona_v1(convert_from(v.sobre_canonico,'UTF8')::jsonb,m->'ambitos');
 d:=convert_from(p_decision,'UTF8')::jsonb;
 salida:=jsonb_build_object('sobre',convert_from(v.sobre_canonico,'UTF8')::jsonb,'acuse',jsonb_build_object('consumo_ref','consumo:'||consumo.consumo_huella_sha256,'auditoria_ref',consumo.auditoria_ref,'decision_ref',consumo.decision_ref,'correlacion_ref',d->>'correlacion_ref','recurso_ref',m->>'persona_ref','persona_ref',m->>'persona_ref','sobre_sha256',v.sobre_sha256,'version',v.version));
 INSERT INTO vec_contexto_actor_v1.lectura_denominacion_persona_v1 VALUES(consumo.decision_ref,convert_to(p_material,'UTF8'),to_jsonb(consumo),salida);
 RETURN salida;
END $f$;
-- Comprobación interna del consumo ORIGINAL confirmado. No devuelve un nombre
-- ni transforma el acuse estructural de Go en una capacidad independiente.
CREATE FUNCTION vec_contexto_actor_v1.cotejar_lectura_denominacion_persona_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET row_security=on AS $f$
DECLARE m jsonb;r record;v record;c jsonb;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_denominacion_persona_v1();m:=vec_contexto_actor_v1.validar_material_denominacion_persona_v1(p_material,false);c:=convert_from(p_capacidad,'UTF8')::jsonb;
 SELECT * INTO STRICT r FROM vec_contexto_actor_v1.lectura_denominacion_persona_v1 WHERE decision_ref=c->>'decision_ref';
 IF r.material_canonico IS DISTINCT FROM convert_to(p_material,'UTF8') OR vec_autorizacion_atestada_v3.cotejar_consumo_denominacion_persona_v3_atestada(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,r.consumo) IS NOT TRUE THEN RAISE EXCEPTION 'CA32: consumo no vigente' USING ERRCODE='42501'; END IF;
 PERFORM vec_contexto_actor_v1.bloquear_persona_denominacion_v1(m->>'persona_ref');
 SELECT x.* INTO STRICT v FROM vec_contexto_actor_v1.denominacion_persona_actual_v1 a JOIN vec_contexto_actor_v1.denominacion_persona_version_v1 x USING(persona_ref,version)WHERE a.persona_ref=m->>'persona_ref' AND a.version=(m->>'version')::numeric FOR SHARE OF a;
 IF v.sobre_sha256 IS DISTINCT FROM r.resultado#>>'{acuse,sobre_sha256}' THEN RAISE EXCEPTION 'CA32: snapshot divergente' USING ERRCODE='42501'; END IF;
 PERFORM vec_contexto_actor_v1.configurar_sobre_denominacion_persona_v1(convert_from(v.sobre_canonico,'UTF8')::jsonb,m->'ambitos');
 RETURN r.resultado;
END $f$;
CREATE FUNCTION vec_contexto_actor_v1.validar_acuse_denominacion_persona_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_acuse text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE r jsonb;
BEGIN r:=vec_contexto_actor_v1.cotejar_lectura_denominacion_persona_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);RETURN p_acuse IS NOT NULL AND r->'acuse'=p_acuse::jsonb;END $f$;
CREATE FUNCTION vec_contexto_actor_v1.revalidar_lectura_denominacion_persona_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_sobre_original text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE r jsonb;
BEGIN r:=vec_contexto_actor_v1.cotejar_lectura_denominacion_persona_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);RETURN p_sobre_original IS NOT NULL AND vec_contexto_actor_v1.canon_sobre_denominacion_persona_v1(r->'sobre')=p_sobre_original;END $f$;
CREATE FUNCTION vec_contexto_actor_v1.version_actual_denominacion_persona_v1(p_persona_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE v record;
BEGIN
 PERFORM vec_contexto_actor_v1.bloquear_persona_denominacion_v1(p_persona_ref);
 SELECT x.* INTO v FROM vec_contexto_actor_v1.denominacion_persona_actual_v1 a JOIN vec_contexto_actor_v1.denominacion_persona_version_v1 x USING(persona_ref,version) WHERE a.persona_ref=p_persona_ref FOR SHARE OF a;
 IF NOT FOUND THEN RETURN null;END IF;
 RETURN jsonb_build_object('persona_ref',v.persona_ref,'version',v.version,'sobre_sha256',v.sobre_sha256);
END $f$;
DO $cerrar$
DECLARE f regprocedure;
BEGIN
 FOR f IN SELECT p.oid::regprocedure FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_contexto_actor_v1' AND p.proname=ANY(ARRAY['referencia_denominacion_v1','base64_denominacion_v1','avanzar_denominacion_persona_v1','canon_sobre_denominacion_persona_v1','validar_material_denominacion_persona_v1','bloquear_persona_denominacion_v1','configurar_sobre_denominacion_persona_v1','exigir_runtime_denominacion_persona_v1','acreditar_runtime_denominacion_persona_v1','publicar_denominacion_persona_v1','leer_denominacion_persona_v1','cotejar_lectura_denominacion_persona_v1','validar_acuse_denominacion_persona_v1','revalidar_lectura_denominacion_persona_v1','version_actual_denominacion_persona_v1'])LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f);
 END LOOP;
END $cerrar$;
-- Sólo el propietario AUT puede proyectar la versión para su lista/ficha
-- previamente autorizada y auditada. No se publica a un grupo genérico.
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.version_actual_denominacion_persona_v1(text)TO vec_autorizacion_propietario;
RESET ROLE;
CREATE ROLE vec_persona_denominacion_ejecutor NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $conexion$BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_persona_denominacion_ejecutor',current_database());END $conexion$;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_persona_denominacion_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_runtime_denominacion_persona_v1(),vec_contexto_actor_v1.publicar_denominacion_persona_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_contexto_actor_v1.leer_denominacion_persona_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_contexto_actor_v1.validar_acuse_denominacion_persona_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text),vec_contexto_actor_v1.revalidar_lectura_denominacion_persona_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)TO vec_persona_denominacion_ejecutor;
COMMIT;
