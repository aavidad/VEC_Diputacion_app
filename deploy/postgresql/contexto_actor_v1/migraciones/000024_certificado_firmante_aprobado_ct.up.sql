\set ON_ERROR_STOP on
-- CA24: identidad por certificado DER; no publica perfiles ni competencia.
-- Alta exclusiva mediante adjunto aprobado AD160. No instalar sobre principal.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:migracion:certificado_firmante:000024',0));
DO $preimagen$
DECLARE n text;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
  OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR pg_catalog.to_regclass('vec_contexto_actor_v1.certificado_firmante_versiones') IS NOT NULL
  OR pg_catalog.to_regclass('vec_contexto_actor_v1.certificado_firmante_actual') IS NOT NULL
  OR pg_catalog.to_regrole('vec_contexto_actor_certificado_firmante_lector_ct') IS NOT NULL
 THEN RAISE EXCEPTION 'CA24: preimagen incompatible' USING ERRCODE='55000'; END IF;
 FOREACH n IN ARRAY ARRAY['proyeccion_cuenta_actual','proyeccion_cuenta_versiones','persona_actual','persona_versiones','vinculo_contexto_actual','vinculo_contexto_versiones','control_generacion_punteros_actuales_v2','registros_contexto'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class c WHERE c.oid=pg_catalog.to_regclass('vec_contexto_actor_v1.'||n)
   AND c.relowner='vec_contexto_actor_v1_propietario'::regrole AND c.relkind='r')
  THEN RAISE EXCEPTION 'CA24: fuente CA ausente %',n USING ERRCODE='55000'; END IF;
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc f WHERE f.oid=pg_catalog.to_regprocedure('vec_autorizacion.acreditar_adjunto_plan_cargo_ct_v1(text,text,text,text)')
   AND f.proowner='vec_autorizacion_propietario'::regrole AND f.prosecdef AND f.provolatile='v'
   AND f.proconfig @> ARRAY['search_path=pg_catalog']
   AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(f.prosrc,'UTF8')),'hex')='f1c551630b9fb6303f57308eb3999ff5b0746f6587a5211c18ab7ee6f1b619bb')
  OR NOT pg_catalog.has_function_privilege('vec_contexto_actor_v1_propietario','vec_autorizacion.acreditar_adjunto_plan_cargo_ct_v1(text,text,text,text)','EXECUTE')
 THEN RAISE EXCEPTION 'CA24: adjunto aprobado AD160 ausente' USING ERRCODE='55000'; END IF;
END $preimagen$;
CREATE ROLE vec_contexto_actor_certificado_firmante_lector_ct NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN
 EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO vec_contexto_actor_certificado_firmante_lector_ct',pg_catalog.current_database());
END $conexion$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;

CREATE TABLE vec_contexto_actor_v1.certificado_firmante_versiones (
 vinculo_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_valida(vinculo_ref,'vcc_') IS TRUE),
 revision numeric(20,0) NOT NULL CHECK(revision BETWEEN 1 AND 18446744073709551615),
 certificado_der_sha256 text NOT NULL CHECK(certificado_der_sha256 ~ '^[0-9a-f]{64}$' AND certificado_der_sha256<>pg_catalog.repeat('0',64)),
 cuenta_ref text NOT NULL,cuenta_version numeric(20,0) NOT NULL,
 persona_ref text NOT NULL,persona_version numeric(20,0) NOT NULL,
 vinculo_cuenta_persona_ref text NOT NULL,vinculo_cuenta_persona_version numeric(20,0) NOT NULL,
 estado text NOT NULL CHECK(estado IN ('activo','revocado')),
 vigente_desde timestamptz(6) NOT NULL,vigente_hasta timestamptz(6) NOT NULL,
 evidencia_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_valida(evidencia_ref,'evi_') IS TRUE),
 evidencia_sha256 text NOT NULL CHECK(evidencia_sha256 ~ '^[0-9a-f]{64}$' AND evidencia_sha256<>pg_catalog.repeat('0',64)),
 plan_clave text NOT NULL CHECK(plan_clave ~ '^[0-9a-f]{32}$'),
 plan_sha256 text NOT NULL CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 aprobacion_ref text NOT NULL,recibo_ref text NOT NULL,
 descriptor_canonico bytea NOT NULL CHECK(pg_catalog.octet_length(descriptor_canonico) BETWEEN 1 AND 16384),
 documento_canonico bytea NOT NULL CHECK(pg_catalog.octet_length(documento_canonico) BETWEEN 1 AND 32768),
 huella_sha256 text NOT NULL CHECK(huella_sha256=pg_catalog.encode(pg_catalog.sha256(documento_canonico),'hex')),
 registrado_en timestamptz(6) NOT NULL,
 PRIMARY KEY(vinculo_ref,revision),UNIQUE(certificado_der_sha256,vinculo_ref,revision,huella_sha256),
 UNIQUE(plan_clave),UNIQUE(recibo_ref),
 FOREIGN KEY(cuenta_ref,cuenta_version) REFERENCES vec_contexto_actor_v1.proyeccion_cuenta_versiones(cuenta_ref,version),
 FOREIGN KEY(persona_ref,persona_version) REFERENCES vec_contexto_actor_v1.persona_versiones(persona_ref,version),
 FOREIGN KEY(vinculo_cuenta_persona_ref,vinculo_cuenta_persona_version) REFERENCES vec_contexto_actor_v1.vinculo_contexto_versiones(vinculo_ref,version),
 CHECK(vec_contexto_actor_v1.instante_valido(vigente_desde) IS TRUE AND vec_contexto_actor_v1.instante_valido(vigente_hasta) IS TRUE AND vigente_hasta>vigente_desde),
 CHECK(vec_contexto_actor_v1.instante_valido(registrado_en) IS TRUE)
);
CREATE TABLE vec_contexto_actor_v1.certificado_firmante_actual (
 certificado_der_sha256 text PRIMARY KEY,vinculo_ref text NOT NULL,revision numeric(20,0) NOT NULL,huella_sha256 text NOT NULL,
 FOREIGN KEY(certificado_der_sha256,vinculo_ref,revision,huella_sha256)
  REFERENCES vec_contexto_actor_v1.certificado_firmante_versiones(certificado_der_sha256,vinculo_ref,revision,huella_sha256)
);
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.certificado_firmante_versiones
 FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia();
CREATE TRIGGER historia_no_truncable BEFORE TRUNCATE ON vec_contexto_actor_v1.certificado_firmante_versiones
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();
CREATE TRIGGER puntero_no_truncable BEFORE TRUNCATE ON vec_contexto_actor_v1.certificado_firmante_actual
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();
CREATE TRIGGER puntero_no_borrable BEFORE DELETE ON vec_contexto_actor_v1.certificado_firmante_actual
 FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia();
DO $acl$
DECLARE n text;
BEGIN
 FOREACH n IN ARRAY ARRAY['certificado_firmante_versiones','certificado_firmante_actual'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_contexto_actor_v1.%I ENABLE ROW LEVEL SECURITY',n);
  EXECUTE pg_catalog.format('ALTER TABLE vec_contexto_actor_v1.%I FORCE ROW LEVEL SECURITY',n);
  EXECUTE pg_catalog.format('CREATE POLICY propietario_exacto ON vec_contexto_actor_v1.%I FOR ALL TO vec_contexto_actor_v1_propietario USING(current_user=''vec_contexto_actor_v1_propietario'') WITH CHECK(current_user=''vec_contexto_actor_v1_propietario'')',n);
  EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_contexto_actor_v1.%I FROM PUBLIC',n);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_contexto_actor_v1.%I FROM PUBLIC',n);
 END LOOP;
END $acl$;

-- El perfil del vca histórico NO es un atributo ni un permiso del binding.
CREATE FUNCTION vec_contexto_actor_v1.identidad_binding_certificado_vigente_ct_v1(c text,cv numeric,p text,pv numeric,v text,vv numeric)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE cuenta record;persona record;enlace record;generacion numeric;ahora timestamptz;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable' OR pg_catalog.current_setting('transaction_read_only')<>'off' THEN RETURN false; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 SELECT a.* INTO cuenta FROM vec_contexto_actor_v1.proyeccion_cuenta_actual x
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones a USING(cuenta_ref,version) WHERE x.cuenta_ref=c FOR SHARE OF x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT a.* INTO enlace FROM vec_contexto_actor_v1.vinculo_contexto_actual x
 JOIN vec_contexto_actor_v1.vinculo_contexto_versiones a USING(vinculo_ref,version) WHERE x.vinculo_ref=v FOR SHARE OF x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT a.* INTO persona FROM vec_contexto_actor_v1.persona_actual x
 JOIN vec_contexto_actor_v1.persona_versiones a USING(persona_ref,version) WHERE x.persona_ref=p FOR SHARE OF x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.generacion INTO STRICT generacion FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 x WHERE control_id=true FOR SHARE OF x;
 ahora:=pg_catalog.clock_timestamp();
 RETURN cuenta.version=cv AND persona.version=pv AND enlace.version=vv AND enlace.cuenta_ref=c AND enlace.persona_ref=p
  AND cuenta.estado='activo' AND persona.estado='activo' AND enlace.estado='activo'
  AND cuenta.procedencia_autoridad='autoridad_maestra_acreditada' AND persona.procedencia_autoridad='autoridad_maestra_acreditada' AND enlace.procedencia_autoridad='autoridad_maestra_acreditada'
  AND ahora>=cuenta.vigente_desde AND ahora<cuenta.vigente_hasta AND ahora>=persona.vigente_desde AND ahora<persona.vigente_hasta AND ahora>=enlace.vigente_desde AND ahora<enlace.vigente_hasta;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.identidad_binding_certificado_vigente_ct_v1(text,numeric,text,numeric,text,numeric) FROM PUBLIC;

CREATE FUNCTION vec_contexto_actor_v1.registrar_vinculo_certificado_aprobado_ct_v1(b bytea,k text,h text,aprobacion text,recibo text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE d jsonb;fuente jsonb;ctx jsonb;canon text;actual record;hist record;registrada timestamptz(6);doc bytea;sha text;estado text;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable' OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR b IS NULL OR pg_catalog.octet_length(b) NOT BETWEEN 1 AND 16384 THEN RETURN false; END IF;
 fuente:=vec_autorizacion.acreditar_adjunto_plan_cargo_ct_v1(k,h,aprobacion,recibo);
 IF fuente IS NULL OR pg_catalog.convert_to(fuente->>'vinculo_certificado_canonico','UTF8') IS DISTINCT FROM b THEN RETURN false; END IF;
 d:=pg_catalog.convert_from(b,'UTF8')::jsonb;
 IF pg_catalog.jsonb_typeof(d)<>'object' OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>18
  OR NOT d ?& ARRAY['esquema','vinculo_ref','revision','certificado_der_sha256','cuenta_ref','cuenta_version','persona_ref','persona_version','vinculo_cuenta_persona_ref','vinculo_cuenta_persona_version','estado','vigente_desde','vigente_hasta','evidencia_ref','evidencia_sha256','preimagen_ref','preimagen_revision','preimagen_sha256']
  OR d->>'esquema' IS DISTINCT FROM 'vec.certificado-cuenta.binding.v1'
  OR d->>'persona_ref' IS DISTINCT FROM fuente->>'persona_ref'
  OR d->>'estado' NOT IN ('activo','revocado')
  OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_each(d) e WHERE e.key NOT IN ('revision','cuenta_version','persona_version','vinculo_cuenta_persona_version','preimagen_revision') AND pg_catalog.jsonb_typeof(e.value) IS DISTINCT FROM 'string')
  OR EXISTS(SELECT 1 FROM pg_catalog.unnest(ARRAY['revision','cuenta_version','persona_version','vinculo_cuenta_persona_version']) x WHERE pg_catalog.jsonb_typeof(d->x) IS DISTINCT FROM 'number' OR d->>x !~ '^[1-9][0-9]{0,19}$' OR (d->>x)::numeric>18446744073709551615)
  OR pg_catalog.jsonb_typeof(d->'preimagen_revision') IS DISTINCT FROM 'number' OR d->>'preimagen_revision' !~ '^(0|[1-9][0-9]{0,19})$' OR (d->>'preimagen_revision')::numeric>18446744073709551615
  OR vec_contexto_actor_v1.referencia_valida(d->>'vinculo_ref','vcc_') IS NOT TRUE
  OR vec_contexto_actor_v1.referencia_valida(d->>'cuenta_ref','cta_') IS NOT TRUE
  OR vec_contexto_actor_v1.referencia_valida(d->>'persona_ref','per_') IS NOT TRUE
  OR vec_contexto_actor_v1.referencia_valida(d->>'vinculo_cuenta_persona_ref','vca_') IS NOT TRUE
  OR vec_contexto_actor_v1.referencia_valida(d->>'evidencia_ref','evi_') IS NOT TRUE
  OR d->>'certificado_der_sha256' !~ '^[0-9a-f]{64}$' OR d->>'certificado_der_sha256'=pg_catalog.repeat('0',64)
  OR d->>'evidencia_sha256' !~ '^[0-9a-f]{64}$' OR d->>'evidencia_sha256'=pg_catalog.repeat('0',64)
  OR d->>'vigente_desde' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$'
  OR d->>'vigente_hasta' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$'
  OR vec_contexto_actor_v1.instante_valido((d->>'vigente_desde')::timestamptz) IS NOT TRUE
  OR vec_contexto_actor_v1.instante_valido((d->>'vigente_hasta')::timestamptz) IS NOT TRUE
  OR (d->>'vigente_hasta')::timestamptz<=(d->>'vigente_desde')::timestamptz THEN RETURN false; END IF;
 -- Reconstitución exacta: rechaza duplicados, espacios y orden alternativo.
 canon:=pg_catalog.format('{"esquema":%s,"vinculo_ref":%s,"revision":%s,"certificado_der_sha256":%s,"cuenta_ref":%s,"cuenta_version":%s,"persona_ref":%s,"persona_version":%s,"vinculo_cuenta_persona_ref":%s,"vinculo_cuenta_persona_version":%s,"estado":%s,"vigente_desde":%s,"vigente_hasta":%s,"evidencia_ref":%s,"evidencia_sha256":%s,"preimagen_ref":%s,"preimagen_revision":%s,"preimagen_sha256":%s}',
  pg_catalog.to_json(d->>'esquema'),pg_catalog.to_json(d->>'vinculo_ref'),d->>'revision',pg_catalog.to_json(d->>'certificado_der_sha256'),pg_catalog.to_json(d->>'cuenta_ref'),d->>'cuenta_version',pg_catalog.to_json(d->>'persona_ref'),d->>'persona_version',pg_catalog.to_json(d->>'vinculo_cuenta_persona_ref'),d->>'vinculo_cuenta_persona_version',pg_catalog.to_json(d->>'estado'),pg_catalog.to_json(d->>'vigente_desde'),pg_catalog.to_json(d->>'vigente_hasta'),pg_catalog.to_json(d->>'evidencia_ref'),pg_catalog.to_json(d->>'evidencia_sha256'),pg_catalog.to_json(d->>'preimagen_ref'),d->>'preimagen_revision',pg_catalog.to_json(d->>'preimagen_sha256'));
 IF pg_catalog.convert_to(canon,'UTF8') IS DISTINCT FROM b THEN RETURN false; END IF;
 SELECT pg_catalog.convert_from(r.representacion_canonica,'UTF8')::jsonb INTO STRICT ctx FROM vec_contexto_actor_v1.registros_contexto r WHERE r.registro_contexto_ref=fuente->>'registro_destino_ref' FOR SHARE OF r;
 IF ctx->>'persona_ref' IS DISTINCT FROM d->>'persona_ref' OR ctx->>'cuenta_ref' IS DISTINCT FROM d->>'cuenta_ref'
  OR (ctx->>'cuenta_version')::numeric IS DISTINCT FROM (d->>'cuenta_version')::numeric
  OR (ctx->>'persona_version')::numeric IS DISTINCT FROM (d->>'persona_version')::numeric
  OR ctx->>'contexto_actor_ref' IS DISTINCT FROM d->>'vinculo_cuenta_persona_ref'
  OR (ctx->>'contexto_version')::numeric IS DISTINCT FROM (d->>'vinculo_cuenta_persona_version')::numeric THEN RETURN false; END IF;
 -- Orden compartido CA global -> certificado -> puntero y generaciones.
 PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:certificado_firmante:'||(d->>'certificado_der_sha256'),0));
 IF vec_contexto_actor_v1.identidad_binding_certificado_vigente_ct_v1(d->>'cuenta_ref',(d->>'cuenta_version')::numeric,d->>'persona_ref',(d->>'persona_version')::numeric,d->>'vinculo_cuenta_persona_ref',(d->>'vinculo_cuenta_persona_version')::numeric) IS NOT TRUE THEN RETURN false; END IF;
 SELECT a.*,v.estado,v.cuenta_ref,v.persona_ref INTO actual FROM vec_contexto_actor_v1.certificado_firmante_actual a
 JOIN vec_contexto_actor_v1.certificado_firmante_versiones v USING(certificado_der_sha256,vinculo_ref,revision,huella_sha256) WHERE a.certificado_der_sha256=d->>'certificado_der_sha256' FOR UPDATE OF a;
 SELECT * INTO hist FROM vec_contexto_actor_v1.certificado_firmante_versiones v WHERE v.plan_clave=k;
 IF FOUND THEN
  RETURN hist.descriptor_canonico=b AND hist.plan_sha256=h AND hist.aprobacion_ref=aprobacion AND hist.recibo_ref=recibo
   AND actual.vinculo_ref=hist.vinculo_ref AND actual.revision=hist.revision AND actual.huella_sha256=hist.huella_sha256;
 END IF;
 IF actual.vinculo_ref IS NULL THEN
  IF d->>'preimagen_ref'<>'' OR (d->>'preimagen_revision')::numeric<>0 OR d->>'preimagen_sha256'<>'' OR (d->>'revision')::numeric<>1 OR d->>'estado'<>'activo' THEN RETURN false; END IF;
 ELSE
  IF actual.vinculo_ref IS DISTINCT FROM d->>'preimagen_ref' OR actual.revision IS DISTINCT FROM (d->>'preimagen_revision')::numeric OR actual.huella_sha256 IS DISTINCT FROM d->>'preimagen_sha256'
   OR actual.vinculo_ref IS DISTINCT FROM d->>'vinculo_ref' OR actual.revision+1 IS DISTINCT FROM (d->>'revision')::numeric
   OR actual.cuenta_ref IS DISTINCT FROM d->>'cuenta_ref' OR actual.persona_ref IS DISTINCT FROM d->>'persona_ref'
   OR (actual.estado='revocado' AND d->>'estado'<>'revocado') THEN RETURN false; END IF;
 END IF;
 registrada:=pg_catalog.clock_timestamp();estado:=d->>'estado';
 IF estado='activo' AND (registrada<(d->>'vigente_desde')::timestamptz OR registrada>=(d->>'vigente_hasta')::timestamptz) THEN RETURN false; END IF;
 doc:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('esquema','vec.certificado-cuenta.registro.v1','descriptor',d,'plan_clave',k,'plan_sha256',h,'aprobacion_ref',aprobacion,'recibo_ref',recibo,'registrado_en',pg_catalog.to_char(registrada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,'UTF8');
 sha:=pg_catalog.encode(pg_catalog.sha256(doc),'hex');
 INSERT INTO vec_contexto_actor_v1.certificado_firmante_versiones VALUES(d->>'vinculo_ref',(d->>'revision')::numeric,d->>'certificado_der_sha256',d->>'cuenta_ref',(d->>'cuenta_version')::numeric,d->>'persona_ref',(d->>'persona_version')::numeric,d->>'vinculo_cuenta_persona_ref',(d->>'vinculo_cuenta_persona_version')::numeric,estado,(d->>'vigente_desde')::timestamptz,(d->>'vigente_hasta')::timestamptz,d->>'evidencia_ref',d->>'evidencia_sha256',k,h,aprobacion,recibo,b,doc,sha,registrada);
 INSERT INTO vec_contexto_actor_v1.certificado_firmante_actual AS a VALUES(d->>'certificado_der_sha256',d->>'vinculo_ref',(d->>'revision')::numeric,sha)
 ON CONFLICT(certificado_der_sha256) DO UPDATE SET vinculo_ref=EXCLUDED.vinculo_ref,revision=EXCLUDED.revision,huella_sha256=EXCLUDED.huella_sha256
 WHERE a.vinculo_ref=actual.vinculo_ref AND a.revision=actual.revision AND a.huella_sha256=actual.huella_sha256;
 IF NOT FOUND THEN RAISE EXCEPTION 'certificado_firmante_cas_rechazado' USING ERRCODE='40001'; END IF;
 RETURN true;
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception OR check_violation OR foreign_key_violation OR unique_violation THEN RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.registrar_vinculo_certificado_aprobado_ct_v1(bytea,text,text,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.registrar_vinculo_certificado_aprobado_ct_v1(bytea,text,text,text,text) TO vec_autorizacion_propietario;

CREATE FUNCTION vec_contexto_actor_v1.exigir_runtime_certificado_firmante_ct_v1()
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE l oid:=pg_catalog.to_regrole(session_user);r oid:='vec_contexto_actor_certificado_firmante_lector_ct'::regrole;f oid[];
BEGIN
 f:=ARRAY[pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_runtime_certificado_firmante_ct_v1()'),pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_vinculo_certificado_firmante_ct_v1(text)'),pg_catalog.to_regprocedure('vec_contexto_actor_v1.exportar_destino_vinculo_certificado_ct_v1(text)')];
 IF pg_catalog.current_setting('role')<>'none' OR l IS NULL OR array_position(f,NULL) IS NOT NULL
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE oid=l AND rolcanlogin AND rolinherit AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls AND rolconfig IS NULL)
  OR (SELECT count(*) FROM pg_catalog.pg_auth_members WHERE member=l)<>1
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=l AND roleid=r AND inherit_option AND NOT admin_option AND NOT set_option)
  OR EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=r)
  OR EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting WHERE setrole IN(l,r))
  OR vec_contexto_actor_v1.privilegios_efectivos_runtime_minimos(l,(SELECT oid FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database()),'vec_contexto_actor_v1'::regnamespace,f) IS NOT TRUE
 THEN RAISE EXCEPTION 'certificado_firmante_runtime_rechazado' USING ERRCODE='42501'; END IF;
 RETURN session_user;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.exigir_runtime_certificado_firmante_ct_v1() FROM PUBLIC;
CREATE FUNCTION vec_contexto_actor_v1.acreditar_runtime_certificado_firmante_ct_v1(OUT identidad_login text,OUT acreditada boolean)
RETURNS record LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN identidad_login:=vec_contexto_actor_v1.exigir_runtime_certificado_firmante_ct_v1();acreditada:=true;END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.acreditar_runtime_certificado_firmante_ct_v1() FROM PUBLIC;

CREATE FUNCTION vec_contexto_actor_v1.revalidar_vinculo_certificado_firmante_ct_v1(ref text,rev numeric,h text,cert text,persona text,cuenta text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE v record;ahora timestamptz;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable' OR pg_catalog.current_setting('transaction_read_only')<>'off' THEN RETURN false; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended('vec_contexto_actor_v1:certificado_firmante:'||cert,0));
 SELECT x.* INTO v FROM vec_contexto_actor_v1.certificado_firmante_actual a JOIN vec_contexto_actor_v1.certificado_firmante_versiones x
 USING(certificado_der_sha256,vinculo_ref,revision,huella_sha256) WHERE a.certificado_der_sha256=cert FOR SHARE OF a;
 IF NOT FOUND OR v.vinculo_ref IS DISTINCT FROM ref OR v.revision IS DISTINCT FROM rev OR v.huella_sha256 IS DISTINCT FROM h
  OR v.persona_ref IS DISTINCT FROM persona OR v.cuenta_ref IS DISTINCT FROM cuenta OR v.estado<>'activo'
  OR vec_contexto_actor_v1.identidad_binding_certificado_vigente_ct_v1(v.cuenta_ref,v.cuenta_version,v.persona_ref,v.persona_version,v.vinculo_cuenta_persona_ref,v.vinculo_cuenta_persona_version) IS NOT TRUE THEN RETURN false; END IF;
 ahora:=pg_catalog.clock_timestamp();
 RETURN ahora>=v.vigente_desde AND ahora<v.vigente_hasta AND pg_catalog.encode(pg_catalog.sha256(v.documento_canonico),'hex')=v.huella_sha256;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.revalidar_vinculo_certificado_firmante_ct_v1(text,numeric,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.revalidar_vinculo_certificado_firmante_ct_v1(text,numeric,text,text,text,text) TO vec_autorizacion_propietario;

CREATE FUNCTION vec_contexto_actor_v1.resolver_vinculo_certificado_firmante_ct_v1(cert text)
RETURNS TABLE(certificado_der_sha256 text,persona_ref text,cuenta_ref text,vinculo_ref text,revision numeric,huella_sha256 text,vigente boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE v record;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_certificado_firmante_ct_v1();
 IF cert IS NULL OR cert !~ '^[0-9a-f]{64}$' OR cert=pg_catalog.repeat('0',64) THEN RETURN; END IF;
 SELECT x.* INTO v FROM vec_contexto_actor_v1.certificado_firmante_actual a JOIN vec_contexto_actor_v1.certificado_firmante_versiones x
 USING(certificado_der_sha256,vinculo_ref,revision,huella_sha256) WHERE a.certificado_der_sha256=cert;
 IF NOT FOUND OR vec_contexto_actor_v1.revalidar_vinculo_certificado_firmante_ct_v1(v.vinculo_ref,v.revision,v.huella_sha256,cert,v.persona_ref,v.cuenta_ref) IS NOT TRUE
 THEN RETURN; END IF;
 RETURN QUERY SELECT v.certificado_der_sha256,v.persona_ref,v.cuenta_ref,v.vinculo_ref,v.revision,v.huella_sha256,true;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.resolver_vinculo_certificado_firmante_ct_v1(text) FROM PUBLIC;

-- Metadatos del registro CA existente para construir el adjunto del kit. No
-- fabrica contexto ni registra identidad de otra persona en una sesión nueva.
CREATE FUNCTION vec_contexto_actor_v1.exportar_destino_vinculo_certificado_ct_v1(registro text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE r record;d jsonb;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_certificado_firmante_ct_v1();
 SELECT * INTO STRICT r FROM vec_contexto_actor_v1.registros_contexto x WHERE x.registro_contexto_ref=registro;
 d:=pg_catalog.convert_from(r.representacion_canonica,'UTF8')::jsonb;
 IF r.autoridad_efectiva<>'autoridad_maestra_acreditada' OR pg_catalog.encode(pg_catalog.sha256(r.representacion_canonica),'hex')<>r.huella_sha256
  OR vec_contexto_actor_v1.identidad_binding_certificado_vigente_ct_v1(d->>'cuenta_ref',(d->>'cuenta_version')::numeric,d->>'persona_ref',(d->>'persona_version')::numeric,d->>'contexto_actor_ref',(d->>'contexto_version')::numeric) IS NOT TRUE
 THEN RAISE EXCEPTION 'certificado_firmante_destino_no_acreditado' USING ERRCODE='P0002'; END IF;
 RETURN pg_catalog.jsonb_build_object('cuenta_ref',d->>'cuenta_ref','cuenta_version',(d->>'cuenta_version')::numeric,'persona_ref',d->>'persona_ref','persona_version',(d->>'persona_version')::numeric,'vinculo_cuenta_persona_ref',d->>'contexto_actor_ref','vinculo_cuenta_persona_version',(d->>'contexto_version')::numeric);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.exportar_destino_vinculo_certificado_ct_v1(text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_contexto_actor_certificado_firmante_lector_ct;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_runtime_certificado_firmante_ct_v1(),vec_contexto_actor_v1.resolver_vinculo_certificado_firmante_ct_v1(text),vec_contexto_actor_v1.exportar_destino_vinculo_certificado_ct_v1(text)
 TO vec_contexto_actor_certificado_firmante_lector_ct;
COMMIT;
