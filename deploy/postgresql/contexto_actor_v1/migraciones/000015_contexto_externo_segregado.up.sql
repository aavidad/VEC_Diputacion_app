\set ON_ERROR_STOP on
-- CTX15: snapshots externos aprobados, recibos propios y revocación segregada.
-- No importa ni modifica historia instalada CTX12/13/14. Sin DOWN con historia.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:migracion:externo:000015',0));
DO $preimagen$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.to_regclass('vec_contexto_actor_v1.contexto_externo_versiones') IS NOT NULL
    OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_candidato_externo_v1(text,text,text)') IS NULL
    OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(text,text,text)') IS NULL
    OR pg_catalog.to_regrole('vec_contexto_actor_v1_usuarios_externo') IS NOT NULL
    OR (SELECT count(*) FROM pg_catalog.pg_proc WHERE oid IN(
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()'),
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)'))
      AND proowner='vec_contexto_actor_v1_propietario'::regrole AND prosecdef AND proconfig=ARRAY['search_path=pg_catalog']::text[])<>2
    OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex') FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure('vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()')) IS DISTINCT FROM '5e390b4528407e3a49b74a33ade5a902424f5593a54cebcd3d5cb649aa6af3c7'
    OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex') FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)')) IS DISTINCT FROM 'c5b5b757a8699e58e0b3b73e0f15c3e8b85a0d17dca262b3cffc9abc9b4e7900' THEN
  RAISE EXCEPTION 'CTX15: preimagen incompatible' USING ERRCODE='55000';
 END IF;
END $preimagen$;
CREATE ROLE vec_contexto_actor_v1_usuarios_externo NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
DO $base$ BEGIN
 EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO vec_contexto_actor_v1_usuarios_externo',current_database());
END $base$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
-- Objeto cerrado: una proyección externa de la persona canónica, no un maestro.
CREATE FUNCTION vec_contexto_actor_v1.snapshot_contexto_externo_valido_v1(p jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE k text; c jsonb; prefijo text; n integer;
BEGIN
 IF p IS NULL OR pg_catalog.jsonb_typeof(p)<>'object' OR pg_catalog.octet_length(p::text)>16384
    OR NOT p ?& ARRAY['provision_ref','poblacion','estado','cuenta','persona','perfil','contexto','vinculo_candidato']
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(p))<>8
    OR (p->>'poblacion' IN('candidato','usuarios')) IS NOT TRUE OR (p->>'estado' IN('activo','revocado')) IS NOT TRUE
    OR vec_contexto_actor_v1.referencia_valida(p->>'provision_ref',CASE WHEN p->>'poblacion'='candidato' THEN 'pce_' ELSE 'pue_' END) IS NOT TRUE
    OR (p->>'poblacion'='usuarios' AND p->'vinculo_candidato' IS DISTINCT FROM 'null'::jsonb) THEN RETURN false; END IF;
 FOREACH k IN ARRAY ARRAY['cuenta','persona','perfil','contexto','vinculo_candidato'] LOOP
  IF k='vinculo_candidato' AND p->>'poblacion'='usuarios' THEN CONTINUE; END IF;
  c:=p->k;
  prefijo:=CASE k WHEN 'cuenta' THEN 'cta_' WHEN 'persona' THEN 'per_' WHEN 'perfil' THEN 'prf_' WHEN 'contexto' THEN 'vca_' ELSE 'vin_' END;
  n:=CASE WHEN k='vinculo_candidato' THEN 11 ELSE 10 END;
  IF pg_catalog.jsonb_typeof(c) IS DISTINCT FROM 'object'
     OR NOT c ?& ARRAY['referencia','version','procedencia_ref','procedencia_version','procedencia_huella_sha256','procedencia_autoridad','estado','vigente_desde','vigente_hasta']
     OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(c))<>n-1
     OR vec_contexto_actor_v1.referencia_valida(c->>'referencia',prefijo) IS NOT TRUE
     OR pg_catalog.jsonb_typeof(c->'version') IS DISTINCT FROM 'number'
     OR pg_catalog.scale((c->>'version')::numeric)<>0
     OR (c->>'version')::numeric NOT BETWEEN 1 AND 18446744073709551615::numeric
     OR pg_catalog.jsonb_typeof(c->'procedencia_version') IS DISTINCT FROM 'number'
     OR pg_catalog.scale((c->>'procedencia_version')::numeric)<>0
     OR vec_contexto_actor_v1.procedencia_valida(c->>'procedencia_ref',(c->>'procedencia_version')::numeric,c->>'procedencia_huella_sha256',c->>'procedencia_autoridad') IS NOT TRUE
     OR c->>'procedencia_autoridad' IS DISTINCT FROM 'autoridad_maestra_acreditada'
     OR c->>'estado' IS DISTINCT FROM p->>'estado'
     OR c->>'vigente_desde' IS NULL OR c->>'vigente_hasta' IS NULL
     OR c->>'vigente_desde' !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$'
     OR c->>'vigente_hasta' !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$'
     OR vec_contexto_actor_v1.instante_valido((c->>'vigente_desde')::timestamptz) IS NOT TRUE
     OR vec_contexto_actor_v1.instante_valido((c->>'vigente_hasta')::timestamptz) IS NOT TRUE
     OR (c->>'vigente_hasta')::timestamptz<=(c->>'vigente_desde')::timestamptz
     OR pg_catalog.to_char((c->>'vigente_desde')::timestamptz AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') IS DISTINCT FROM c->>'vigente_desde'
     OR pg_catalog.to_char((c->>'vigente_hasta')::timestamptz AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') IS DISTINCT FROM c->>'vigente_hasta'
     OR (k='vinculo_candidato' AND vec_contexto_actor_v1.referencia_valida(c->>'candidato_ref','can_') IS NOT TRUE) THEN RETURN false; END IF;
 END LOOP;
 RETURN true;
EXCEPTION WHEN data_exception THEN RETURN false;
END $f$;
CREATE FUNCTION vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(p_snapshot jsonb,p_version numeric)
RETURNS text LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.jsonb_build_array(p_snapshot,p_version)::text,'UTF8')),'hex')
$f$;
CREATE TABLE vec_contexto_actor_v1.contexto_externo_identidad(
 provision_ref text PRIMARY KEY,cuenta_ref text NOT NULL UNIQUE,perfil_ref text NOT NULL UNIQUE,
 persona_ref text NOT NULL,familia text NOT NULL CHECK(familia IN('candidato','usuarios')),
 contexto_ref text NOT NULL UNIQUE,
 UNIQUE(provision_ref,cuenta_ref,perfil_ref,persona_ref,familia,contexto_ref)
);
CREATE TABLE vec_contexto_actor_v1.contexto_externo_versiones(
 provision_ref text NOT NULL,version numeric(20,0) NOT NULL CHECK(version BETWEEN 1 AND 18446744073709551615::numeric),
 cuenta_ref text NOT NULL,perfil_ref text NOT NULL,persona_ref text NOT NULL,familia text NOT NULL,contexto_ref text NOT NULL,
 snapshot jsonb NOT NULL CHECK(vec_contexto_actor_v1.snapshot_contexto_externo_valido_v1(snapshot)),
 huella_sha256 text NOT NULL CHECK(huella_sha256=vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(snapshot,version)),
 PRIMARY KEY(provision_ref,version),
 FOREIGN KEY(provision_ref,cuenta_ref,perfil_ref,persona_ref,familia,contexto_ref) REFERENCES vec_contexto_actor_v1.contexto_externo_identidad(provision_ref,cuenta_ref,perfil_ref,persona_ref,familia,contexto_ref),
 CHECK(snapshot->>'provision_ref'=provision_ref AND snapshot->>'poblacion'=familia
  AND snapshot#>>'{cuenta,referencia}'=cuenta_ref AND snapshot#>>'{perfil,referencia}'=perfil_ref
  AND snapshot#>>'{persona,referencia}'=persona_ref AND snapshot#>>'{contexto,referencia}'=contexto_ref)
);
CREATE TABLE vec_contexto_actor_v1.contexto_externo_actual(
 provision_ref text PRIMARY KEY,version numeric(20,0) NOT NULL,
 FOREIGN KEY(provision_ref,version) REFERENCES vec_contexto_actor_v1.contexto_externo_versiones(provision_ref,version)
);
CREATE TABLE vec_contexto_actor_v1.control_generacion_contexto_externo_v1(
 control_id boolean PRIMARY KEY CHECK(control_id),generacion numeric(20,0) NOT NULL CHECK(generacion>=0)
);
INSERT INTO vec_contexto_actor_v1.control_generacion_contexto_externo_v1 VALUES(true,0);
CREATE TABLE vec_contexto_actor_v1.registros_contexto_externo_v2(
 LIKE vec_contexto_actor_v1.registros_contexto INCLUDING ALL,
 provision_ref text NOT NULL,provision_version numeric(20,0) NOT NULL,provision_huella_sha256 text NOT NULL,
 FOREIGN KEY(provision_ref,provision_version) REFERENCES vec_contexto_actor_v1.contexto_externo_versiones(provision_ref,version)
);
DO $rls$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['contexto_externo_identidad','contexto_externo_versiones','contexto_externo_actual','control_generacion_contexto_externo_v1','registros_contexto_externo_v2'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_contexto_actor_v1.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('ALTER TABLE vec_contexto_actor_v1.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('CREATE POLICY propietario_exacto ON vec_contexto_actor_v1.%I FOR ALL TO vec_contexto_actor_v1_propietario USING(current_user=''vec_contexto_actor_v1_propietario'') WITH CHECK(current_user=''vec_contexto_actor_v1_propietario'')',t);
  EXECUTE pg_catalog.format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_contexto_actor_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado()',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_contexto_actor_v1.%I FROM PUBLIC,vec_contexto_actor_v1_runtime,vec_contexto_actor_v1_candidato_externo,vec_contexto_actor_v1_usuarios_externo,vec_autorizacion_propietario',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_contexto_actor_v1.%I FROM PUBLIC,vec_contexto_actor_v1_runtime,vec_contexto_actor_v1_candidato_externo,vec_contexto_actor_v1_usuarios_externo,vec_autorizacion_propietario',t);
 END LOOP;
 FOREACH t IN ARRAY ARRAY['contexto_externo_identidad','contexto_externo_versiones','registros_contexto_externo_v2'] LOOP
  EXECUTE pg_catalog.format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.%I FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia()',t);
 END LOOP;
END $rls$;
CREATE FUNCTION vec_contexto_actor_v1.controlar_mutacion_contexto_externo_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:externo:mutacion:v1',0));
 UPDATE vec_contexto_actor_v1.control_generacion_contexto_externo_v1 SET generacion=generacion+1 WHERE control_id=true;
 IF NOT FOUND THEN RAISE EXCEPTION 'generación externa ausente' USING ERRCODE='55000'; END IF;
 RETURN NULL;
END $f$;
CREATE TRIGGER controlar_generacion BEFORE INSERT OR UPDATE OR DELETE ON vec_contexto_actor_v1.contexto_externo_actual FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.controlar_mutacion_contexto_externo_v1();
CREATE FUNCTION vec_contexto_actor_v1.preimagen_snapshot_contexto_externo_v1(p_provision_ref text)
RETURNS TABLE(version numeric,huella_sha256 text) LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT v.version,v.huella_sha256 FROM vec_contexto_actor_v1.contexto_externo_actual a JOIN vec_contexto_actor_v1.contexto_externo_versiones v USING(provision_ref,version) WHERE a.provision_ref=p_provision_ref
$f$;
CREATE FUNCTION vec_contexto_actor_v1.publicar_snapshot_contexto_externo_v1(p_snapshot jsonb,p_version_esperada numeric,p_huella_esperada text,p_huella_aprobada text)
RETURNS TABLE(provision_ref text,version numeric,huella_sha256 text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE anterior record; v numeric; h text; k text; c jsonb; ahora timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR vec_contexto_actor_v1.snapshot_contexto_externo_valido_v1(p_snapshot) IS NOT TRUE
    OR p_version_esperada IS NULL OR pg_catalog.scale(p_version_esperada)<>0
    OR p_version_esperada NOT BETWEEN 0 AND 18446744073709551614::numeric
    OR p_huella_aprobada IS NULL OR p_huella_aprobada !~ '^[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'snapshot externo inválido' USING ERRCODE='22023'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:externo:mutacion:v1',0));
 SELECT x.* INTO anterior FROM vec_contexto_actor_v1.contexto_externo_actual a JOIN vec_contexto_actor_v1.contexto_externo_versiones x USING(provision_ref,version) WHERE a.provision_ref=p_snapshot->>'provision_ref' FOR UPDATE OF a;
 IF (NOT FOUND AND (p_version_esperada<>0 OR p_huella_esperada IS NOT NULL))
    OR (FOUND AND (anterior.version IS DISTINCT FROM p_version_esperada OR anterior.huella_sha256 IS DISTINCT FROM p_huella_esperada)) THEN
  RAISE EXCEPTION 'snapshot externo CAS divergente' USING ERRCODE='40001'; END IF;
 ahora:=pg_catalog.clock_timestamp();
 IF p_version_esperada>0 AND p_snapshot->>'estado'='activo' AND anterior.snapshot->>'estado'<>'activo' THEN
  RAISE EXCEPTION 'snapshot externo no reactivable' USING ERRCODE='55000'; END IF;
 FOREACH k IN ARRAY ARRAY['cuenta','persona','perfil','contexto','vinculo_candidato'] LOOP
  c:=p_snapshot->k;
  IF c='null'::jsonb THEN CONTINUE; END IF;
  IF p_snapshot->>'estado'='activo' AND (ahora<(c->>'vigente_desde')::timestamptz OR ahora>=(c->>'vigente_hasta')::timestamptz) THEN
   RAISE EXCEPTION 'componente externo no vigente' USING ERRCODE='55000'; END IF;
  IF p_version_esperada>0 AND p_snapshot->>'estado'='activo' AND ahora>=(anterior.snapshot#>>ARRAY[k,'vigente_hasta'])::timestamptz THEN
   RAISE EXCEPTION 'componente externo caducado no reactivable' USING ERRCODE='55000'; END IF;
  IF p_version_esperada>0 AND (c->>'referencia' IS DISTINCT FROM anterior.snapshot#>>ARRAY[k,'referencia']
      OR (c->>'version')::numeric<(anterior.snapshot#>>ARRAY[k,'version'])::numeric
      OR ((c->>'version')::numeric=(anterior.snapshot#>>ARRAY[k,'version'])::numeric AND c IS DISTINCT FROM anterior.snapshot->k)
      OR (k='vinculo_candidato' AND c->>'candidato_ref' IS DISTINCT FROM anterior.snapshot#>>ARRAY[k,'candidato_ref'])) THEN
   RAISE EXCEPTION 'componente externo divergente' USING ERRCODE='55000'; END IF;
 END LOOP;
 -- Solo el canal de provisión interno contrasta la colisión con historia compartida.
 -- La persona canónica puede coincidir; cuenta/perfil/contexto necesitan referencias propias.
 IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones c WHERE c.cuenta_ref=p_snapshot#>>'{cuenta,referencia}')
    OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_versiones c WHERE c.perfil_ref=p_snapshot#>>'{perfil,referencia}')
    OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_versiones c WHERE c.vinculo_ref=p_snapshot#>>'{contexto,referencia}') THEN
  RAISE EXCEPTION 'referencia externa ya compartida' USING ERRCODE='55000'; END IF;
 v:=p_version_esperada+1; h:=vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(p_snapshot,v);
 IF h IS DISTINCT FROM p_huella_aprobada THEN RAISE EXCEPTION 'huella externa no aprobada' USING ERRCODE='42501'; END IF;
 IF p_version_esperada=0 THEN
  INSERT INTO vec_contexto_actor_v1.contexto_externo_identidad VALUES(p_snapshot->>'provision_ref',p_snapshot#>>'{cuenta,referencia}',p_snapshot#>>'{perfil,referencia}',p_snapshot#>>'{persona,referencia}',p_snapshot->>'poblacion',p_snapshot#>>'{contexto,referencia}');
 END IF;
 INSERT INTO vec_contexto_actor_v1.contexto_externo_versiones VALUES(p_snapshot->>'provision_ref',v,p_snapshot#>>'{cuenta,referencia}',p_snapshot#>>'{perfil,referencia}',p_snapshot#>>'{persona,referencia}',p_snapshot->>'poblacion',p_snapshot#>>'{contexto,referencia}',p_snapshot,h);
 INSERT INTO vec_contexto_actor_v1.contexto_externo_actual AS a VALUES(p_snapshot->>'provision_ref',v) ON CONFLICT ON CONSTRAINT contexto_externo_actual_pkey DO UPDATE SET version=excluded.version;
 RETURN QUERY SELECT p_snapshot->>'provision_ref',v,h;
END $f$;
CREATE FUNCTION vec_contexto_actor_v1.snapshot_contexto_externo_vigente_v1(p_cuenta text,p_perfil text,p_familia text,p_ahora timestamptz)
RETURNS SETOF vec_contexto_actor_v1.contexto_externo_versiones LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE s vec_contexto_actor_v1.contexto_externo_versiones; k text; c jsonb; n integer;
BEGIN
 PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended('vec_contexto_actor_v1:externo:mutacion:v1',0));
 PERFORM 1 FROM vec_contexto_actor_v1.control_generacion_contexto_externo_v1 WHERE control_id=true FOR SHARE;
 IF NOT FOUND OR p_ahora IS NULL THEN RETURN; END IF;
 p_ahora:=pg_catalog.clock_timestamp();
 SELECT count(*) INTO n FROM vec_contexto_actor_v1.contexto_externo_actual a JOIN vec_contexto_actor_v1.contexto_externo_versiones v USING(provision_ref,version) WHERE v.cuenta_ref=p_cuenta AND v.perfil_ref=p_perfil AND v.familia=p_familia;
 IF n<>1 THEN RETURN; END IF;
 SELECT v.* INTO STRICT s FROM vec_contexto_actor_v1.contexto_externo_actual a JOIN vec_contexto_actor_v1.contexto_externo_versiones v USING(provision_ref,version) WHERE v.cuenta_ref=p_cuenta AND v.perfil_ref=p_perfil AND v.familia=p_familia;
 IF s.snapshot->>'estado'<>'activo' THEN RETURN; END IF;
 FOREACH k IN ARRAY ARRAY['cuenta','persona','perfil','contexto','vinculo_candidato'] LOOP
  c:=s.snapshot->k; IF c='null'::jsonb THEN CONTINUE; END IF;
  IF p_ahora<(c->>'vigente_desde')::timestamptz OR p_ahora>=(c->>'vigente_hasta')::timestamptz THEN RETURN; END IF;
 END LOOP;
 RETURN NEXT s;
END $f$;

-- Canon y manifiesto con el mismo orden de claves V2; fuentes solo del snapshot.
CREATE FUNCTION vec_contexto_actor_v1.canon_snapshot_contexto_externo_v2(p jsonb,p_resuelto timestamptz)
RETURNS TABLE(representacion bytea,manifiesto bytea) LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE k text; c jsonb; etiqueta text; partes text[]:='{}'; vinculo text:=''; proc_vinculo text:=''; documento text; procedencia text;
BEGIN
 FOREACH k IN ARRAY ARRAY['cuenta','persona','perfil','contexto'] LOOP
  c:=p->k; etiqueta:=CASE WHEN k='contexto' THEN 'vinculo_ref' ELSE k||'_ref' END;
  partes:=pg_catalog.array_append(partes,pg_catalog.format('"%s":{"%s":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s}',k,etiqueta,(c->'referencia')::text,c->>'version',(c->'procedencia_ref')::text,c->>'procedencia_version',(c->'procedencia_huella_sha256')::text,(c->'procedencia_autoridad')::text));
 END LOOP;
 c:=p->'vinculo_candidato';
 IF c IS DISTINCT FROM 'null'::jsonb THEN
  vinculo:=pg_catalog.format('{"vinculo_ref":%s,"version":%s,"tipo":"candidato","referencia":%s,"estado":%s,"vigente_desde":%s,"vigente_hasta":%s}',(c->'referencia')::text,c->>'version',(c->'candidato_ref')::text,(c->'estado')::text,(c->'vigente_desde')::text,(c->'vigente_hasta')::text);
  proc_vinculo:=pg_catalog.format('{"vinculo_ref":%s,"version":%s,"tipo":"candidato","referencia":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s}',(c->'referencia')::text,c->>'version',(c->'candidato_ref')::text,(c->'procedencia_ref')::text,c->>'procedencia_version',(c->'procedencia_huella_sha256')::text,(c->'procedencia_autoridad')::text);
 END IF;
 procedencia:=pg_catalog.format('{"esquema":"vec.contexto-actor.procedencia-manifiesto.v1","autoridad_efectiva":"autoridad_maestra_acreditada",%s,"vinculos":[%s]}',pg_catalog.array_to_string(partes,','),proc_vinculo);
 documento:=pg_catalog.format('{"esquema":"vec.contexto-actor.vinculado.v2","principal_ref":%s,"metodo":"certificado","garantia":"alto","perfil_activo_ref":%s,"persona_ref":%s,"contexto_actor_ref":%s,"contexto_version":%s,"cuenta_ref":%s,"cuenta_version":%s,"persona_version":%s,"perfil_version":%s,"estado":%s,"vigente_desde":%s,"vigente_hasta":%s,"resuelto_en":%s,"vinculos":[%s]}',
  (p#>'{persona,referencia}')::text,(p#>'{perfil,referencia}')::text,(p#>'{persona,referencia}')::text,(p#>'{contexto,referencia}')::text,p#>>'{contexto,version}',(p#>'{cuenta,referencia}')::text,p#>>'{cuenta,version}',p#>>'{persona,version}',p#>>'{perfil,version}',(p#>'{contexto,estado}')::text,(p#>'{contexto,vigente_desde}')::text,(p#>'{contexto,vigente_hasta}')::text,
  pg_catalog.to_json(pg_catalog.to_char(p_resuelto AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,vinculo);
 RETURN QUERY SELECT pg_catalog.convert_to(documento,'UTF8'),pg_catalog.convert_to(procedencia,'UTF8');
END $f$;
CREATE FUNCTION vec_contexto_actor_v1.exigir_runtime_usuarios_externo_v1()
RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE l record; g record; n integer; esquema oid; base oid; funciones oid[];
BEGIN
 SELECT oid,rolcanlogin,rolsuper,rolinherit,rolcreaterole,rolcreatedb,rolreplication,rolbypassrls,rolconfig
 INTO l FROM pg_catalog.pg_roles WHERE rolname=session_user;
 SELECT oid,rolcanlogin,rolsuper,rolinherit,rolcreaterole,rolcreatedb,rolreplication,rolbypassrls,rolconfig
 INTO g FROM pg_catalog.pg_roles WHERE rolname='vec_contexto_actor_v1_usuarios_externo';
 SELECT oid INTO esquema FROM pg_catalog.pg_namespace WHERE nspname='vec_contexto_actor_v1';
 SELECT oid INTO base FROM pg_catalog.pg_database WHERE datname=current_database();
 funciones:=ARRAY[
  pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_runtime_usuarios_externo_v1()'),
  pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_contexto_usuarios_externo_v1(text,text,text,text,timestamptz)'),
  pg_catalog.to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_usuarios_externo_v1(text,text,text,text,timestamptz)')];
 SELECT count(*) INTO n FROM pg_catalog.pg_auth_members WHERE member=l.oid;
 IF l.oid IS NULL OR g.oid IS NULL OR esquema IS NULL OR base IS NULL OR array_position(funciones,NULL) IS NOT NULL
   OR l.rolcanlogin IS NOT TRUE OR l.rolsuper OR NOT l.rolinherit OR l.rolcreaterole OR l.rolcreatedb
   OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
   OR g.rolcanlogin OR g.rolsuper OR g.rolinherit OR g.rolcreaterole OR g.rolcreatedb
   OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
   OR current_setting('role')<>'none' OR n<>1
   OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=l.oid AND roleid=g.oid
       AND NOT admin_option AND inherit_option AND NOT set_option)
   OR EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.oid<>l.oid AND r.oid<>g.oid AND pg_catalog.pg_has_role(l.oid,r.oid,'MEMBER'))
   OR EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.oid<>g.oid AND pg_catalog.pg_has_role(g.oid,r.oid,'MEMBER'))
   OR EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting WHERE setrole IN(l.oid,g.oid))
   OR EXISTS(SELECT 1 FROM pg_catalog.pg_default_acl d LEFT JOIN LATERAL pg_catalog.aclexplode(coalesce(d.defaclacl,'{}'::aclitem[])) a ON true
      WHERE d.defaclrole IN(l.oid,g.oid) OR a.grantee IN(l.oid,g.oid) OR a.grantor IN(l.oid,g.oid))
   OR EXISTS(SELECT 1 FROM pg_catalog.pg_policy p WHERE l.oid=ANY(p.polroles) OR g.oid=ANY(p.polroles))
   OR EXISTS(SELECT 1 FROM pg_catalog.pg_shdepend d WHERE d.refclassid='pg_catalog.pg_authid'::regclass AND d.refobjid=l.oid)
   OR NOT coalesce((SELECT count(*)=5 AND bool_and(d.deptype='a' AND d.objsubid=0 AND (
       (d.classid='pg_catalog.pg_database'::regclass AND d.objid=base) OR
       (d.classid='pg_catalog.pg_namespace'::regclass AND d.objid=esquema) OR
       (d.classid='pg_catalog.pg_proc'::regclass AND d.objid=ANY(funciones))))
      FROM pg_catalog.pg_shdepend d WHERE d.refclassid='pg_catalog.pg_authid'::regclass AND d.refobjid=g.oid),false)
   OR NOT coalesce((SELECT count(*)=1 AND bool_and(a.privilege_type='CONNECT' AND NOT a.is_grantable)
      FROM pg_catalog.pg_database d CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(d.datacl,pg_catalog.acldefault('d',d.datdba))) a
      WHERE d.oid=base AND a.grantee=g.oid),false)
   OR NOT coalesce((SELECT count(*)=1 AND bool_and(a.privilege_type='USAGE' AND NOT a.is_grantable)
      FROM pg_catalog.pg_namespace d CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(d.nspacl,pg_catalog.acldefault('n',d.nspowner))) a
      WHERE d.oid=esquema AND a.grantee=g.oid),false)
   OR NOT coalesce((SELECT count(*)=3 AND bool_and(a.privilege_type='EXECUTE' AND NOT a.is_grantable)
      FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
      WHERE p.oid=ANY(funciones) AND a.grantee=g.oid),false)
   OR vec_contexto_actor_v1.privilegios_efectivos_runtime_minimos(l.oid,base,esquema,funciones) IS NOT TRUE THEN
  RAISE EXCEPTION 'LOGIN candidato externo ContextoActor no acreditado' USING ERRCODE='42501';
 END IF;
 RETURN session_user;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.exigir_runtime_usuarios_externo_v1() FROM PUBLIC;
CREATE FUNCTION vec_contexto_actor_v1.acreditar_runtime_usuarios_externo_v1()
RETURNS TABLE(identidad_login text,acreditada boolean)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 identidad_login:=vec_contexto_actor_v1.exigir_runtime_usuarios_externo_v1();
 acreditada:=true; RETURN NEXT;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.acreditar_runtime_usuarios_externo_v1() FROM PUBLIC;

-- La preimagen anterior fija los bytes. Se retira una única rama de fallback
-- externa del acreditador corporativo sin reescribir su auditoría de ACL.
DO $retirar_fallback$
DECLARE definicion text; marca text:=$marca$    IF pg_catalog.pg_has_role(session_user,'vec_contexto_actor_v1_candidato_externo','MEMBER') THEN
        RETURN vec_contexto_actor_v1.exigir_runtime_candidato_externo_v1();
    END IF;
$marca$;
BEGIN
 SELECT pg_catalog.pg_get_functiondef('vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()'::regprocedure) INTO definicion;
 IF (length(definicion)-length(replace(definicion,marca,'')))/length(marca)<>1 THEN
  RAISE EXCEPTION 'CTX15: rama viva divergente' USING ERRCODE='55000'; END IF;
 EXECUTE replace(definicion,marca,'');
END $retirar_fallback$;

CREATE FUNCTION vec_contexto_actor_v1.exigir_familia_contexto_externo_v1(p_familia text)
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF p_familia='candidato' THEN PERFORM vec_contexto_actor_v1.exigir_runtime_candidato_externo_v1();
 ELSIF p_familia='usuarios' THEN PERFORM vec_contexto_actor_v1.exigir_runtime_usuarios_externo_v1();
 ELSE RAISE EXCEPTION 'familia externa inválida' USING ERRCODE='42501'; END IF;
END $f$;

CREATE FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_externo_v2(p_operacion_ref text,p_registro_contexto_ref text,p_cuenta_ref text,p_perfil_ref text,p_solicitado_en timestamptz,p_familia text)
RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,
 huella_sha256 text,manifiesto_procedencia_canonico bytea,manifiesto_procedencia_huella_sha256 text,
 autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE s vec_contexto_actor_v1.contexto_externo_versiones; r record; b record; ahora timestamptz;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_familia_contexto_externo_v1(p_familia);
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'aislamiento de contexto externo inválido' USING ERRCODE='25000'; END IF;
 IF vec_contexto_actor_v1.referencia_operacion_valida(p_operacion_ref,'oca_') IS NOT TRUE
    OR vec_contexto_actor_v1.referencia_operacion_valida(p_registro_contexto_ref,'rca_') IS NOT TRUE
    OR vec_contexto_actor_v1.referencia_valida(p_cuenta_ref,'cta_') IS NOT TRUE
    OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref,'prf_') IS NOT TRUE
    OR vec_contexto_actor_v1.instante_valido(p_solicitado_en) IS NOT TRUE THEN RAISE EXCEPTION 'solicitud externa inválida' USING ERRCODE='22023'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:externo:operacion:'||p_operacion_ref,0));
 IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.registros_contexto compartido WHERE compartido.operacion_ref=p_operacion_ref OR compartido.registro_contexto_ref=p_registro_contexto_ref) THEN
  RAISE EXCEPTION 'identificador de recibo externo ya compartido' USING ERRCODE='23505'; END IF;
 SELECT * INTO s FROM vec_contexto_actor_v1.snapshot_contexto_externo_vigente_v1(p_cuenta_ref,p_perfil_ref,p_familia,pg_catalog.clock_timestamp());
 IF NOT FOUND THEN RAISE EXCEPTION 'snapshot externo no vigente' USING ERRCODE='P0002'; END IF;
 SELECT x.* INTO r FROM vec_contexto_actor_v1.registros_contexto_externo_v2 x WHERE x.operacion_ref=p_operacion_ref;
 IF FOUND THEN
  IF r.registro_contexto_ref IS DISTINCT FROM p_registro_contexto_ref OR r.cuenta_ref IS DISTINCT FROM p_cuenta_ref
     OR r.perfil_ref IS DISTINCT FROM p_perfil_ref OR r.solicitado_en IS DISTINCT FROM p_solicitado_en
     OR r.provision_ref IS DISTINCT FROM s.provision_ref OR r.provision_version IS DISTINCT FROM s.version
     OR r.provision_huella_sha256 IS DISTINCT FROM s.huella_sha256 THEN RAISE EXCEPTION 'colisión operación externa' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT r.operacion_ref,r.registro_contexto_ref,r.representacion_canonica,r.huella_sha256,r.manifiesto_procedencia_canonico,r.manifiesto_procedencia_huella_sha256,r.autoridad_efectiva,r.resuelto_en;
  RETURN;
 END IF;
 ahora:=pg_catalog.clock_timestamp();
 IF ahora<p_solicitado_en OR ahora>p_solicitado_en+interval '5 seconds' THEN RAISE EXCEPTION 'ventana externa agotada' USING ERRCODE='57014'; END IF;
 SELECT * INTO b FROM vec_contexto_actor_v1.canon_snapshot_contexto_externo_v2(s.snapshot,ahora);
 INSERT INTO vec_contexto_actor_v1.registros_contexto_externo_v2(operacion_ref,registro_contexto_ref,cuenta_ref,perfil_ref,metodo,garantia,solicitado_en,resuelto_en,representacion_canonica,huella_sha256,manifiesto_procedencia_canonico,manifiesto_procedencia_huella_sha256,autoridad_efectiva,provision_ref,provision_version,provision_huella_sha256)
 VALUES(p_operacion_ref,p_registro_contexto_ref,p_cuenta_ref,p_perfil_ref,'certificado','alto',p_solicitado_en,ahora,b.representacion,pg_catalog.encode(pg_catalog.sha256(b.representacion),'hex'),b.manifiesto,pg_catalog.encode(pg_catalog.sha256(b.manifiesto),'hex'),'autoridad_maestra_acreditada',s.provision_ref,s.version,s.huella_sha256);
 RETURN QUERY SELECT p_operacion_ref,p_registro_contexto_ref,b.representacion,pg_catalog.encode(pg_catalog.sha256(b.representacion),'hex'),b.manifiesto,pg_catalog.encode(pg_catalog.sha256(b.manifiesto),'hex'),'autoridad_maestra_acreditada'::text,ahora;
END $f$;

CREATE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_externo_v2(p_operacion_ref text,p_registro_contexto_ref text,p_cuenta_ref text,p_perfil_ref text,p_solicitado_en timestamptz,p_familia text)
RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,
 huella_sha256 text,manifiesto_procedencia_canonico bytea,manifiesto_procedencia_huella_sha256 text,
 autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE s vec_contexto_actor_v1.contexto_externo_versiones; r record; b record; ahora timestamptz;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_familia_contexto_externo_v1(p_familia);
 IF current_setting('transaction_isolation')<>'read committed' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'aislamiento de contexto externo inválido' USING ERRCODE='25000'; END IF;
 IF vec_contexto_actor_v1.referencia_operacion_valida(p_operacion_ref,'oca_') IS NOT TRUE
    OR vec_contexto_actor_v1.referencia_operacion_valida(p_registro_contexto_ref,'rca_') IS NOT TRUE
    OR vec_contexto_actor_v1.referencia_valida(p_cuenta_ref,'cta_') IS NOT TRUE
    OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref,'prf_') IS NOT TRUE
    OR vec_contexto_actor_v1.instante_valido(p_solicitado_en) IS NOT TRUE THEN RAISE EXCEPTION 'solicitud externa inválida' USING ERRCODE='22023'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:externo:operacion:'||p_operacion_ref,0));
 IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.registros_contexto compartido WHERE compartido.operacion_ref=p_operacion_ref OR compartido.registro_contexto_ref=p_registro_contexto_ref) THEN
  RAISE EXCEPTION 'identificador de recibo externo ya compartido' USING ERRCODE='23505'; END IF;
 SELECT * INTO s FROM vec_contexto_actor_v1.snapshot_contexto_externo_vigente_v1(p_cuenta_ref,p_perfil_ref,p_familia,pg_catalog.clock_timestamp());
 IF NOT FOUND THEN RAISE EXCEPTION 'snapshot externo no vigente' USING ERRCODE='P0002'; END IF;
 SELECT x.* INTO r FROM vec_contexto_actor_v1.registros_contexto_externo_v2 x WHERE x.operacion_ref=p_operacion_ref;
 IF FOUND THEN
  IF r.registro_contexto_ref IS DISTINCT FROM p_registro_contexto_ref OR r.cuenta_ref IS DISTINCT FROM p_cuenta_ref
     OR r.perfil_ref IS DISTINCT FROM p_perfil_ref OR r.solicitado_en IS DISTINCT FROM p_solicitado_en
     OR r.provision_ref IS DISTINCT FROM s.provision_ref OR r.provision_version IS DISTINCT FROM s.version
     OR r.provision_huella_sha256 IS DISTINCT FROM s.huella_sha256 THEN RAISE EXCEPTION 'colisión operación externa' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT r.operacion_ref,r.registro_contexto_ref,r.representacion_canonica,r.huella_sha256,r.manifiesto_procedencia_canonico,r.manifiesto_procedencia_huella_sha256,r.autoridad_efectiva,r.resuelto_en;
  RETURN;
 END IF;
 RETURN;
END $f$;

CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.resolver_contexto_candidato_externo_v1(p_operacion_ref text,p_registro_contexto_ref text,p_cuenta_ref text,p_perfil_ref text,p_solicitado_en timestamptz) RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,
 huella_sha256 text,manifiesto_procedencia_canonico bytea,manifiesto_procedencia_huella_sha256 text,
 autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 RETURN QUERY SELECT * FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_externo_v2(p_operacion_ref,p_registro_contexto_ref,p_cuenta_ref,p_perfil_ref,p_solicitado_en,'candidato');
END $f$;

CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_candidato_externo_v1(p_operacion_ref text,p_registro_contexto_ref text,p_cuenta_ref text,p_perfil_ref text,p_solicitado_en timestamptz) RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,
 huella_sha256 text,manifiesto_procedencia_canonico bytea,manifiesto_procedencia_huella_sha256 text,
 autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 RETURN QUERY SELECT * FROM vec_contexto_actor_v1.reconciliar_contexto_externo_v2(p_operacion_ref,p_registro_contexto_ref,p_cuenta_ref,p_perfil_ref,p_solicitado_en,'candidato');
END $f$;

CREATE FUNCTION vec_contexto_actor_v1.resolver_contexto_usuarios_externo_v1(p_operacion_ref text,p_registro_contexto_ref text,p_cuenta_ref text,p_perfil_ref text,p_solicitado_en timestamptz) RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,
 huella_sha256 text,manifiesto_procedencia_canonico bytea,manifiesto_procedencia_huella_sha256 text,
 autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 RETURN QUERY SELECT * FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_externo_v2(p_operacion_ref,p_registro_contexto_ref,p_cuenta_ref,p_perfil_ref,p_solicitado_en,'usuarios');
END $f$;

CREATE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_usuarios_externo_v1(p_operacion_ref text,p_registro_contexto_ref text,p_cuenta_ref text,p_perfil_ref text,p_solicitado_en timestamptz) RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,
 huella_sha256 text,manifiesto_procedencia_canonico bytea,manifiesto_procedencia_huella_sha256 text,
 autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 RETURN QUERY SELECT * FROM vec_contexto_actor_v1.reconciliar_contexto_externo_v2(p_operacion_ref,p_registro_contexto_ref,p_cuenta_ref,p_perfil_ref,p_solicitado_en,'usuarios');
END $f$;

CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.acreditar_candidato_externo_v1(p_persona_ref text,p_perfil_ref text,p_candidato_ref text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE s record; n integer;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RETURN false; END IF;
 SELECT count(*) INTO n FROM vec_contexto_actor_v1.contexto_externo_identidad i WHERE i.persona_ref=p_persona_ref AND i.perfil_ref=p_perfil_ref AND i.familia='candidato';
 IF n<>1 THEN RETURN false; END IF;
 SELECT v.* INTO s FROM vec_contexto_actor_v1.contexto_externo_identidad i CROSS JOIN LATERAL vec_contexto_actor_v1.snapshot_contexto_externo_vigente_v1(i.cuenta_ref,i.perfil_ref,'candidato',pg_catalog.clock_timestamp()) v WHERE i.persona_ref=p_persona_ref AND i.perfil_ref=p_perfil_ref AND i.familia='candidato';
 RETURN FOUND AND s.snapshot#>>'{vinculo_candidato,candidato_ref}' IS NOT DISTINCT FROM p_candidato_ref;
END $f$;
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(p_cuenta_ref text,p_persona_ref text,p_perfil_ref text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE s record;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RETURN false; END IF;
 SELECT * INTO s FROM vec_contexto_actor_v1.snapshot_contexto_externo_vigente_v1(p_cuenta_ref,p_perfil_ref,'usuarios',pg_catalog.clock_timestamp());
 RETURN FOUND AND s.persona_ref IS NOT DISTINCT FROM p_persona_ref;
END $f$;
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.perfil_candidato_externo_provisionado_v1(p_persona_ref text,p_perfil_ref text)
RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE s record;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RETURN false; END IF;
 SELECT v.* INTO s FROM vec_contexto_actor_v1.contexto_externo_identidad i CROSS JOIN LATERAL vec_contexto_actor_v1.snapshot_contexto_externo_vigente_v1(i.cuenta_ref,i.perfil_ref,'candidato',pg_catalog.clock_timestamp()) v WHERE i.persona_ref=p_persona_ref AND i.perfil_ref=p_perfil_ref AND i.familia='candidato';
 RETURN FOUND;
END $f$;
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.perfil_usuarios_externo_provisionado_v1(p_persona_ref text,p_perfil_ref text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE s record;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RETURN false; END IF;
 SELECT v.* INTO s FROM vec_contexto_actor_v1.contexto_externo_identidad i CROSS JOIN LATERAL vec_contexto_actor_v1.snapshot_contexto_externo_vigente_v1(i.cuenta_ref,i.perfil_ref,'usuarios',pg_catalog.clock_timestamp()) v WHERE i.persona_ref=p_persona_ref AND i.perfil_ref=p_perfil_ref AND i.familia='usuarios';
 RETURN FOUND;
END $f$;
-- Conserva el núcleo interno exacto instalado; la nueva fachada privada
-- rechaza población externa histórica antes de llamarlo.
DO $conservar_nucleo$
DECLARE definicion text; marca text:='FUNCTION vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(';
BEGIN
 SELECT pg_catalog.pg_get_functiondef('vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)'::regprocedure) INTO definicion;
 IF (length(definicion)-length(replace(definicion,marca,'')))/length(marca)<>1 THEN
  RAISE EXCEPTION 'CTX15: núcleo vivo divergente' USING ERRCODE='55000'; END IF;
 EXECUTE replace(definicion,marca,'FUNCTION vec_contexto_actor_v1.acreditar_uso_registro_contexto_interno_v2(');
END $conservar_nucleo$;

CREATE FUNCTION vec_contexto_actor_v1.acreditar_uso_registro_contexto_externo_v2(
    p_registro_contexto_ref text,
    p_contexto_actor_esquema text,
    p_contexto_actor_huella_sha256 text,
    p_manifiesto_procedencia_huella_sha256 text,
    p_autoridad_efectiva text,
    p_cuenta_ref text,
    p_cuenta_version numeric,
    p_persona_ref text,
    p_persona_version numeric,
    p_perfil_ref text,
    p_perfil_version numeric,
    p_contexto_actor_ref text,
    p_contexto_actor_version numeric,
    p_metodo text,
    p_garantia text,
    p_emitida_en timestamptz,
    p_valida_hasta timestamptz
)
RETURNS timestamptz LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE r record; s record; b record; k text; c jsonb; ahora timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'acreditación externa requiere serializable' USING ERRCODE='25000'; END IF;
 SELECT x.* INTO r FROM vec_contexto_actor_v1.registros_contexto_externo_v2 x WHERE x.registro_contexto_ref=p_registro_contexto_ref FOR SHARE OF x;
 IF NOT FOUND OR p_contexto_actor_esquema IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
    OR r.cuenta_ref IS DISTINCT FROM p_cuenta_ref OR r.perfil_ref IS DISTINCT FROM p_perfil_ref
    OR r.metodo IS DISTINCT FROM p_metodo OR r.garantia IS DISTINCT FROM p_garantia
    OR r.huella_sha256 IS DISTINCT FROM p_contexto_actor_huella_sha256
    OR r.manifiesto_procedencia_huella_sha256 IS DISTINCT FROM p_manifiesto_procedencia_huella_sha256
    OR r.autoridad_efectiva IS DISTINCT FROM p_autoridad_efectiva
    OR vec_contexto_actor_v1.instante_valido(p_emitida_en) IS NOT TRUE
    OR vec_contexto_actor_v1.instante_valido(p_valida_hasta) IS NOT TRUE
    OR p_valida_hasta<=p_emitida_en OR r.resuelto_en>p_emitida_en THEN RETURN NULL; END IF;
 SELECT v.* INTO s FROM vec_contexto_actor_v1.contexto_externo_identidad i CROSS JOIN LATERAL vec_contexto_actor_v1.snapshot_contexto_externo_vigente_v1(i.cuenta_ref,i.perfil_ref,i.familia,pg_catalog.clock_timestamp()) v WHERE i.provision_ref=r.provision_ref;
 IF NOT FOUND OR s.version IS DISTINCT FROM r.provision_version OR s.huella_sha256 IS DISTINCT FROM r.provision_huella_sha256
    OR s.persona_ref IS DISTINCT FROM p_persona_ref OR s.contexto_ref IS DISTINCT FROM p_contexto_actor_ref
    OR (s.snapshot#>>'{cuenta,version}')::numeric IS DISTINCT FROM p_cuenta_version
    OR (s.snapshot#>>'{persona,version}')::numeric IS DISTINCT FROM p_persona_version
    OR (s.snapshot#>>'{perfil,version}')::numeric IS DISTINCT FROM p_perfil_version
    OR (s.snapshot#>>'{contexto,version}')::numeric IS DISTINCT FROM p_contexto_actor_version THEN RETURN NULL; END IF;
 ahora:=pg_catalog.clock_timestamp();
 IF ahora<p_emitida_en OR ahora>=p_valida_hasta THEN RETURN NULL; END IF;
 FOREACH k IN ARRAY ARRAY['cuenta','persona','perfil','contexto','vinculo_candidato'] LOOP
  c:=s.snapshot->k; IF c='null'::jsonb THEN CONTINUE; END IF;
  IF p_emitida_en<(c->>'vigente_desde')::timestamptz OR p_valida_hasta>(c->>'vigente_hasta')::timestamptz THEN RETURN NULL; END IF;
 END LOOP;
 SELECT * INTO b FROM vec_contexto_actor_v1.canon_snapshot_contexto_externo_v2(s.snapshot,r.resuelto_en);
 IF b.representacion IS DISTINCT FROM r.representacion_canonica OR b.manifiesto IS DISTINCT FROM r.manifiesto_procedencia_canonico
    OR pg_catalog.encode(pg_catalog.sha256(b.representacion),'hex') IS DISTINCT FROM p_contexto_actor_huella_sha256
    OR pg_catalog.encode(pg_catalog.sha256(b.manifiesto),'hex') IS DISTINCT FROM p_manifiesto_procedencia_huella_sha256 THEN RETURN NULL; END IF;
 RETURN ahora;
END $f$;
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
    p_registro_contexto_ref text,
    p_contexto_actor_esquema text,
    p_contexto_actor_huella_sha256 text,
    p_manifiesto_procedencia_huella_sha256 text,
    p_autoridad_efectiva text,
    p_cuenta_ref text,
    p_cuenta_version numeric,
    p_persona_ref text,
    p_persona_version numeric,
    p_perfil_ref text,
    p_perfil_version numeric,
    p_contexto_actor_ref text,
    p_contexto_actor_version numeric,
    p_metodo text,
    p_garantia text,
    p_emitida_en timestamptz,
    p_valida_hasta timestamptz
)
RETURNS timestamptz LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE externo boolean; interno boolean;
BEGIN
 SELECT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.registros_contexto_externo_v2 r WHERE r.registro_contexto_ref=p_registro_contexto_ref),
        EXISTS(SELECT 1 FROM vec_contexto_actor_v1.registros_contexto r WHERE r.registro_contexto_ref=p_registro_contexto_ref)
 INTO externo,interno;
 IF externo AND interno THEN RETURN NULL; END IF;
 IF externo THEN RETURN vec_contexto_actor_v1.acreditar_uso_registro_contexto_externo_v2(p_registro_contexto_ref,p_contexto_actor_esquema,p_contexto_actor_huella_sha256,p_manifiesto_procedencia_huella_sha256,p_autoridad_efectiva,p_cuenta_ref,p_cuenta_version,p_persona_ref,p_persona_version,p_perfil_ref,p_perfil_version,p_contexto_actor_ref,p_contexto_actor_version,p_metodo,p_garantia,p_emitida_en,p_valida_hasta); END IF;
 -- CTX12/14 quedan como historia; su población externa no se reactiva por fallback.
 IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.registro_candidato_externo_v1 r WHERE r.registro_contexto_ref=p_registro_contexto_ref)
    OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_usuarios_externo_identidad i WHERE i.cuenta_ref=p_cuenta_ref OR i.perfil_ref=p_perfil_ref)
    OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.contexto_externo_identidad i WHERE i.cuenta_ref=p_cuenta_ref OR i.perfil_ref=p_perfil_ref) THEN RETURN NULL; END IF;
 RETURN vec_contexto_actor_v1.acreditar_uso_registro_contexto_interno_v2(p_registro_contexto_ref,p_contexto_actor_esquema,p_contexto_actor_huella_sha256,p_manifiesto_procedencia_huella_sha256,p_autoridad_efectiva,p_cuenta_ref,p_cuenta_version,p_persona_ref,p_persona_version,p_perfil_ref,p_perfil_version,p_contexto_actor_ref,p_contexto_actor_version,p_metodo,p_garantia,p_emitida_en,p_valida_hasta);
END $f$;

DO $acl$ DECLARE r record; BEGIN
 FOR r IN SELECT p.oid::regprocedure AS firma FROM pg_catalog.pg_proc p WHERE p.pronamespace='vec_contexto_actor_v1'::regnamespace AND p.proname=ANY(ARRAY['snapshot_contexto_externo_valido_v1','huella_snapshot_contexto_externo_v1','controlar_mutacion_contexto_externo_v1','preimagen_snapshot_contexto_externo_v1','publicar_snapshot_contexto_externo_v1','snapshot_contexto_externo_vigente_v1','canon_snapshot_contexto_externo_v2','exigir_runtime_usuarios_externo_v1','acreditar_runtime_usuarios_externo_v1','exigir_familia_contexto_externo_v1','resolver_y_registrar_contexto_externo_v2','reconciliar_contexto_externo_v2','resolver_contexto_usuarios_externo_v1','reconciliar_contexto_usuarios_externo_v1','acreditar_uso_registro_contexto_interno_v2','acreditar_uso_registro_contexto_externo_v2']) LOOP
 EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,vec_contexto_actor_v1_runtime,vec_contexto_actor_v1_candidato_externo,vec_contexto_actor_v1_usuarios_externo,vec_autorizacion_propietario',r.firma);
 END LOOP;
END $acl$;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_contexto_actor_v1_candidato_externo;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_runtime_candidato_externo_v1(),
 vec_contexto_actor_v1.resolver_contexto_candidato_externo_v1(text,text,text,text,timestamptz),
 vec_contexto_actor_v1.reconciliar_contexto_candidato_externo_v1(text,text,text,text,timestamptz) TO vec_contexto_actor_v1_candidato_externo;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_contexto_actor_v1_usuarios_externo;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_runtime_usuarios_externo_v1(),
 vec_contexto_actor_v1.resolver_contexto_usuarios_externo_v1(text,text,text,text,timestamptz),
 vec_contexto_actor_v1.reconciliar_contexto_usuarios_externo_v1(text,text,text,text,timestamptz) TO vec_contexto_actor_v1_usuarios_externo;
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.publicar_provision_candidato_externo_v1(
 p_provision_ref text,p_version_esperada numeric,p_huella_esperada text,p_huella_aprobada text,
 p_cuenta_ref text,p_cuenta_version numeric,p_perfil_ref text,p_perfil_version numeric,
 p_persona_ref text,p_persona_version numeric,p_contexto_ref text,p_contexto_version numeric,
 p_vinculo_candidato_ref text,p_vinculo_candidato_version numeric,p_candidato_ref text,
 p_estado text,p_desde timestamptz,p_hasta timestamptz,p_fuente_huella_sha256 text
) RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN RAISE EXCEPTION 'provisión externa histórica retirada' USING ERRCODE='55000'; END $f$;
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.publicar_perfil_usuarios_externo_v1(
 p_provision_ref text,p_version_esperada numeric,p_huella_esperada text,p_huella_aprobada text,
 p_cuenta_ref text,p_cuenta_version numeric,p_persona_ref text,p_persona_version numeric,
 p_perfil_ref text,p_perfil_version numeric,p_contexto_ref text,p_contexto_version numeric,
 p_estado text,p_desde timestamptz,p_hasta timestamptz,p_fuente_huella_sha256 text
) RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN RAISE EXCEPTION 'provisión externa histórica retirada' USING ERRCODE='55000'; END $f$;
RESET ROLE;
DO $postimagen$ DECLARE g oid; n integer; BEGIN
 FOREACH g IN ARRAY ARRAY['vec_contexto_actor_v1_candidato_externo'::regrole::oid,'vec_contexto_actor_v1_usuarios_externo'::regrole::oid] LOOP
  SELECT count(*) INTO n FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.pronamespace='vec_contexto_actor_v1'::regnamespace AND a.grantee=g;
  IF n<>3 THEN RAISE EXCEPTION 'CTX15: ACL externa incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
END $postimagen$;
COMMIT;
