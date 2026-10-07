\set ON_ERROR_STOP on
-- CA37: proyección de una única identidad interna ordinaria sintética.
-- La titularidad tiene historia propia: no reutiliza la fuente de altas ADMIN.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:identidad-interna-sintetica:migracion:v1',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 OR to_regprocedure('vec_identidad_sesiones_v1.validar_plan_identidad_interna_sintetica_v1(jsonb)') IS NULL
 OR to_regprocedure('vec_identidad_sesiones_v1.cotejar_recibo_identidad_interna_sintetica_v1(text,text,text)') IS NULL
 OR to_regclass('vec_contexto_actor_v1.efectos_identidad_interna_sintetica_v1') IS NOT NULL
 OR to_regclass('vec_contexto_actor_v1.titularidad_cuenta_persona_v1') IS NULL
 OR EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid='vec_contexto_actor_v1.titularidad_cuenta_persona_v1'::regclass AND attname IN('tipo_operacion','operacion_identidad_ref') AND NOT attisdropped)
 OR to_regprocedure('vec_contexto_actor_v1.preimagen_identidad_interna_sintetica_v1(jsonb)') IS NOT NULL
 OR to_regprocedure('vec_contexto_actor_v1.confirmar_identidad_interna_sintetica_v1(jsonb,jsonb,text,text,text,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'CA37: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;

CREATE TABLE vec_contexto_actor_v1.efectos_identidad_interna_sintetica_v1(
 operacion_ref text PRIMARY KEY CHECK(operacion_ref ~ '^piis_[A-Za-z0-9_-]{22,123}$'),
 plan_sha256 text NOT NULL CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 aprobacion_ref text NOT NULL CHECK(octet_length(aprobacion_ref) BETWEEN 1 AND 128),
 plan jsonb NOT NULL,recibo_is jsonb NOT NULL,recibo jsonb NOT NULL,
 registrada_en timestamptz NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(operacion_ref,plan_sha256,aprobacion_ref)
);
-- La tabla original sigue siendo la autoridad semántica única. Las filas
-- existentes mantienen su FK ADMIN; las nuevas identifican expresamente su
-- efecto CA37 y nunca se hacen pasar por una fuente de altas iniciales.
ALTER TABLE vec_contexto_actor_v1.titularidad_cuenta_persona_v1
 ADD COLUMN tipo_operacion text NOT NULL DEFAULT 'fuentes_iniciales_admin_v1',
 ADD COLUMN operacion_identidad_ref text;
ALTER TABLE vec_contexto_actor_v1.titularidad_cuenta_persona_v1
 ALTER COLUMN operacion_ref DROP NOT NULL,
 ADD CONSTRAINT titularidad_cuenta_persona_tipo_operacion_ck CHECK(
  (tipo_operacion='fuentes_iniciales_admin_v1' AND operacion_ref IS NOT NULL AND operacion_identidad_ref IS NULL)
  OR (tipo_operacion='identidad_interna_sintetica_v1' AND operacion_ref IS NULL AND operacion_identidad_ref IS NOT NULL)),
 ADD CONSTRAINT titularidad_cuenta_persona_identidad_fk FOREIGN KEY(operacion_identidad_ref)
  REFERENCES vec_contexto_actor_v1.efectos_identidad_interna_sintetica_v1(operacion_ref) DEFERRABLE INITIALLY DEFERRED;
DO $tablas$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['efectos_identidad_interna_sintetica_v1'] LOOP
  EXECUTE format('ALTER TABLE vec_contexto_actor_v1.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_contexto_actor_v1.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_contexto_actor_v1.%I TO vec_contexto_actor_v1_propietario USING(current_user=%L) WITH CHECK(current_user=%L)',t,'vec_contexto_actor_v1_propietario','vec_contexto_actor_v1_propietario');
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.%I FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia()',t);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_contexto_actor_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado()',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_contexto_actor_v1.%I FROM PUBLIC',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_contexto_actor_v1.%I FROM PUBLIC',t);
 END LOOP;
END $tablas$;

CREATE FUNCTION vec_contexto_actor_v1.preimagen_identidad_interna_sintetica_v1(p jsonb)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE efecto record;org record;ahora timestamptz;ev jsonb;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION 'CA37: requiere SERIALIZABLE escritura' USING ERRCODE='25000'; END IF;
 PERFORM vec_identidad_sesiones_v1.validar_plan_identidad_interna_sintetica_v1(p);
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:identidad-interna-sintetica:v1',0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:identidad-interna-sintetica:v1:'||(p->>'operacion_ref'),0));
 SELECT * INTO efecto FROM vec_contexto_actor_v1.efectos_identidad_interna_sintetica_v1 WHERE operacion_ref=p->>'operacion_ref';
 IF FOUND THEN
  IF efecto.plan IS DISTINCT FROM p THEN RAISE EXCEPTION 'CA37: replay divergente' USING ERRCODE='40001'; END IF;
  RETURN jsonb_build_object('esquema','vec.ca.preimagen-identidad-interna-sintetica.v1','version',1,'operacion_ref',p->>'operacion_ref','recibo',efecto.recibo);
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:persona:v1:'||(p#>>'{persona,persona_ref}'),0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:procedencia:v1:'||(p#>>'{procedencia,referencia}'),0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:procedencia:v1:'||(p#>>'{persona,fuente_titularidad,referencia}'),0));
 SELECT ov.* INTO org FROM vec_contexto_actor_v1.organizacion_actual oa
 JOIN vec_contexto_actor_v1.organizacion_versiones ov USING(organizacion_ref,version)
 WHERE oa.organizacion_ref=p#>>'{organizacion,organizacion_ref}' FOR SHARE OF oa,ov;
 ahora:=clock_timestamp();
 IF NOT FOUND OR org.version<>(p#>>'{organizacion,version_esperada}')::numeric OR org.estado<>'activo'
 OR ahora<org.vigente_desde OR ahora>=org.vigente_hasta OR org.procedencia_autoridad<>'autoridad_maestra_acreditada'
 OR org.procedencia_huella_sha256 IS DISTINCT FROM p#>>'{organizacion,procedencia_huella_sha256}'
 OR org.vigente_hasta IS DISTINCT FROM (p#>>'{organizacion,vigente_hasta}')::timestamptz
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.persona_versiones WHERE persona_ref=p#>>'{persona,persona_ref}')
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.procedencias WHERE procedencia_ref IN(p#>>'{procedencia,referencia}',p#>>'{persona,fuente_titularidad,referencia}'))
 OR ((p#>>'{procedencia,referencia}')=(p#>>'{persona,fuente_titularidad,referencia}')
     AND (p#>>'{procedencia,huella_sha256}')<>(p#>>'{persona,fuente_titularidad,huella_sha256}'))
 THEN RAISE EXCEPTION 'CA37: CAS inicial divergente' USING ERRCODE='40001'; END IF;
 ev:=jsonb_build_object('esquema','vec.ca.preimagen-identidad-interna-sintetica.v1','version',1,'operacion_ref',p->>'operacion_ref',
  'organizacion_ref',org.organizacion_ref,'organizacion_version',org.version,'organizacion_procedencia_ref',org.procedencia_ref,
  'organizacion_procedencia_version',org.procedencia_version,'organizacion_procedencia_huella_sha256',org.procedencia_huella_sha256,
  'organizacion_procedencia_autoridad',org.procedencia_autoridad,'organizacion_estado',org.estado,
  'organizacion_vigente_desde',to_char(org.vigente_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'organizacion_vigente_hasta',to_char(org.vigente_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'persona_ref',p#>>'{persona,persona_ref}','persona_version',0,
  'persona_vigente_hasta',p#>>'{persona,vigente_hasta}',
  'persona_procedencia_ref',p#>>'{procedencia,referencia}','persona_procedencia_version',p#>>'{procedencia,version}',
  'persona_procedencia_huella_sha256',p#>>'{procedencia,huella_sha256}',
  'titularidad_fuente_ref',p#>>'{persona,fuente_titularidad,referencia}','titularidad_fuente_version',p#>>'{persona,fuente_titularidad,version}',
  'titularidad_fuente_huella_sha256',p#>>'{persona,fuente_titularidad,huella_sha256}',
  'cuenta_ordinaria_ref',NULL,'proyeccion_cuenta_version',0);
 RETURN ev;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.preimagen_identidad_interna_sintetica_v1(jsonb) FROM PUBLIC,vec_contexto_actor_v1_runtime;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.preimagen_identidad_interna_sintetica_v1(jsonb) TO vec_autorizacion_propietario;

CREATE FUNCTION vec_contexto_actor_v1.confirmar_identidad_interna_sintetica_v1(p jsonb,recibo_is jsonb,pre_sha text,operacion text,plan_sha text,aprobacion text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE pre jsonb;real_is jsonb;existente record;cuenta text;ahora timestamptz;recibo jsonb;
BEGIN
 pre:=vec_contexto_actor_v1.preimagen_identidad_interna_sintetica_v1(p);
 IF operacion IS DISTINCT FROM p->>'operacion_ref' OR plan_sha IS NULL OR plan_sha !~ '^[0-9a-f]{64}$'
 OR aprobacion IS NULL OR octet_length(aprobacion) NOT BETWEEN 1 AND 128 THEN
  RAISE EXCEPTION 'CA37: compromiso inválido' USING ERRCODE='22023'; END IF;
 real_is:=vec_identidad_sesiones_v1.cotejar_recibo_identidad_interna_sintetica_v1(operacion,plan_sha,aprobacion);
 IF real_is IS DISTINCT FROM recibo_is THEN RAISE EXCEPTION 'CA37: recibo IS no acreditado' USING ERRCODE='55000'; END IF;
 SELECT * INTO existente FROM vec_contexto_actor_v1.efectos_identidad_interna_sintetica_v1 WHERE operacion_ref=operacion;
 IF FOUND THEN
  IF existente.plan_sha256 IS DISTINCT FROM plan_sha OR existente.aprobacion_ref IS DISTINCT FROM aprobacion OR existente.recibo_is IS DISTINCT FROM real_is THEN
   RAISE EXCEPTION 'CA37: replay divergente' USING ERRCODE='40001'; END IF;
  RETURN existente.recibo;
 END IF;
 IF pre_sha IS DISTINCT FROM encode(sha256(convert_to(pre::text,'UTF8')),'hex') THEN
  RAISE EXCEPTION 'CA37: CAS preimagen divergente' USING ERRCODE='40001'; END IF;
 cuenta:=real_is#>>'{datos,cuenta_ordinaria_ref}';
 IF real_is->>'esquema' IS DISTINCT FROM 'vec.is.identidad-interna-sintetica.v1' OR real_is->>'version' IS DISTINCT FROM '1'
 OR real_is->>'operacion_ref' IS DISTINCT FROM operacion OR real_is->>'plan_sha256' IS DISTINCT FROM plan_sha
 OR real_is->>'aprobacion_ref' IS DISTINCT FROM aprobacion OR real_is->>'alcance_fuente' IS DISTINCT FROM 'sintetico_declarado'
 OR real_is#>>'{datos,persona_ref}' IS DISTINCT FROM p#>>'{persona,persona_ref}' OR real_is#>>'{datos,version_titularidad}' IS DISTINCT FROM '1'
 OR jsonb_typeof(real_is->'datos') IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(real_is->'datos'))<>3
 OR vec_contexto_actor_v1.referencia_valida(cuenta,'cta_') IS NOT TRUE OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones WHERE cuenta_ref=cuenta)
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.titularidad_cuenta_persona_v1 WHERE cuenta_ref=cuenta)
 THEN RAISE EXCEPTION 'CA37: recibo IS o cuenta divergente' USING ERRCODE='55000'; END IF;
 ahora:=clock_timestamp();
 -- AUT57 acredita privadamente la fuente maestra limitada al clon de
 -- desarrollo; el alcance sintético permanece explícito en titularidad/recibo.
 INSERT INTO vec_contexto_actor_v1.procedencias VALUES(p#>>'{procedencia,referencia}',1,p#>>'{procedencia,huella_sha256}','autoridad_maestra_acreditada');
 IF p#>>'{persona,fuente_titularidad,referencia}' IS DISTINCT FROM p#>>'{procedencia,referencia}' THEN
  INSERT INTO vec_contexto_actor_v1.procedencias VALUES(p#>>'{persona,fuente_titularidad,referencia}',1,p#>>'{persona,fuente_titularidad,huella_sha256}','autoridad_maestra_acreditada');
 END IF;
 INSERT INTO vec_contexto_actor_v1.persona_versiones(persona_ref,version,procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES(p#>>'{persona,persona_ref}',1,p#>>'{procedencia,referencia}',1,p#>>'{procedencia,huella_sha256}','autoridad_maestra_acreditada','activo',ahora,(p#>>'{persona,vigente_hasta}')::timestamptz);
 INSERT INTO vec_contexto_actor_v1.persona_actual VALUES(p#>>'{persona,persona_ref}',1);
 INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones(cuenta_ref,version,procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES(cuenta,1,p#>>'{procedencia,referencia}',1,p#>>'{procedencia,huella_sha256}','autoridad_maestra_acreditada','activo',ahora,(p#>>'{persona,vigente_hasta}')::timestamptz);
 INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES(cuenta,1);
 INSERT INTO vec_contexto_actor_v1.titularidad_cuenta_persona_v1(cuenta_ref,persona_ref,version,fuente_ref,fuente_version,fuente_sha256,alcance_fuente,operacion_ref,tipo_operacion,operacion_identidad_ref,vigente_desde,vigente_hasta)
 VALUES(cuenta,p#>>'{persona,persona_ref}',1,p#>>'{persona,fuente_titularidad,referencia}',1,p#>>'{persona,fuente_titularidad,huella_sha256}','sintetico_declarado',NULL,'identidad_interna_sintetica_v1',operacion,ahora,(p#>>'{persona,vigente_hasta}')::timestamptz);
 recibo:=jsonb_build_object('esquema','vec.ca.identidad-interna-sintetica.v1','version',1,
  'recibo_ref','recibo_ca_identidad:'||replace(gen_random_uuid()::text,'-',''),'operacion_ref',operacion,'plan_sha256',plan_sha,
  'aprobacion_ref',aprobacion,'alcance_fuente','sintetico_declarado','registrada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'datos',jsonb_build_object('organizacion_ref',p#>>'{organizacion,organizacion_ref}','organizacion_version',(p#>>'{organizacion,version_esperada}')::numeric,
   'persona_ref',p#>>'{persona,persona_ref}','persona_version',1,'cuenta_ordinaria_ref',cuenta,'proyeccion_cuenta_version',1));
 recibo:=recibo||jsonb_build_object('huella_sha256',encode(sha256(convert_to(recibo::text,'UTF8')),'hex'));
 INSERT INTO vec_contexto_actor_v1.efectos_identidad_interna_sintetica_v1(operacion_ref,plan_sha256,aprobacion_ref,plan,recibo_is,recibo)
 VALUES(operacion,plan_sha,aprobacion,p,real_is,recibo);
 RETURN recibo;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.confirmar_identidad_interna_sintetica_v1(jsonb,jsonb,text,text,text,text) FROM PUBLIC,vec_contexto_actor_v1_runtime;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.confirmar_identidad_interna_sintetica_v1(jsonb,jsonb,text,text,text,text) TO vec_autorizacion_propietario;

-- Fachada histórica para AUT57: coteja el vínculo tal como estaba al instante
-- del recibo. No concede un perfil ni acredita vigencia actual; ese consumidor
-- deberá revalidar los punteros/estado actuales mediante su propia frontera.
CREATE FUNCTION vec_contexto_actor_v1.cotejar_titularidad_cuenta_persona_canonica_v1(p_cuenta text,p_persona text,p_instante timestamptz)
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE n integer;
BEGIN
 IF vec_contexto_actor_v1.referencia_valida(p_cuenta,'cta_') IS NOT TRUE OR vec_contexto_actor_v1.referencia_valida(p_persona,'per_') IS NOT TRUE
 OR vec_contexto_actor_v1.instante_valido(p_instante) IS NOT TRUE THEN RETURN false; END IF;
 SELECT count(*) INTO n FROM vec_contexto_actor_v1.titularidad_cuenta_persona_v1 t
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones cv ON cv.cuenta_ref=t.cuenta_ref AND cv.version=t.version
 JOIN vec_contexto_actor_v1.persona_versiones pv ON pv.persona_ref=t.persona_ref AND pv.version=t.version
 LEFT JOIN vec_contexto_actor_v1.efectos_identidad_interna_sintetica_v1 ei ON ei.operacion_ref=t.operacion_identidad_ref
 WHERE t.cuenta_ref=p_cuenta AND t.persona_ref=p_persona AND t.version=1
  AND t.tipo_operacion IN('fuentes_iniciales_admin_v1','identidad_interna_sintetica_v1')
  AND p_instante>=t.vigente_desde AND p_instante<t.vigente_hasta
  AND cv.estado='activo' AND pv.estado='activo'
  AND p_instante>=cv.vigente_desde AND p_instante<cv.vigente_hasta
  AND p_instante>=pv.vigente_desde AND p_instante<pv.vigente_hasta
  AND ((t.tipo_operacion='fuentes_iniciales_admin_v1' AND t.operacion_ref IS NOT NULL)
    OR (t.tipo_operacion='identidad_interna_sintetica_v1' AND ei.operacion_ref IS NOT NULL
     AND ei.recibo#>>'{datos,cuenta_ordinaria_ref}'=t.cuenta_ref AND ei.recibo#>>'{datos,persona_ref}'=t.persona_ref));
 RETURN n=1;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.cotejar_titularidad_cuenta_persona_canonica_v1(text,text,timestamptz) FROM PUBLIC,vec_contexto_actor_v1_runtime;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.cotejar_titularidad_cuenta_persona_canonica_v1(text,text,timestamptz) TO vec_autorizacion_propietario;
COMMIT;
