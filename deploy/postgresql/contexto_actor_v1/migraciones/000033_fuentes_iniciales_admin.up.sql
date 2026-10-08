\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN
  RAISE EXCEPTION 'CA33: clave=migrador.superusuario actual=false esperado=true' USING ERRCODE='42501'; END IF;
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999 THEN
  RAISE EXCEPTION 'CA33: clave=server_version_num actual=% esperado=180000..189999',current_setting('server_version_num') USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_identidad_sesiones_v1.cotejar_recibo_fuentes_iniciales_admin_v1(text,text,text)') IS NULL THEN RAISE EXCEPTION 'CA33: clave=dependencia.vec_identidad_sesiones_v1.cotejar_recibo_fuentes_iniciales_admin_v1(text,text,text) actual=ausente esperado=presente' USING ERRCODE='55000'; END IF;
 IF to_regclass('vec_contexto_actor_v1.control_generacion_punteros_actuales_v2') IS NULL THEN RAISE EXCEPTION 'CA33: clave=dependencia.vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 actual=ausente esperado=presente' USING ERRCODE='55000'; END IF;
 IF to_regclass('vec_contexto_actor_v1.fuentes_iniciales_admin_v1') IS NOT NULL THEN RAISE EXCEPTION 'CA33: clave=vec_contexto_actor_v1.fuentes_iniciales_admin_v1 actual=presente esperado=ausente' USING ERRCODE='55000'; END IF;
END $pre$;
DO $ca20_preimagen$
DECLARE actual text;
BEGIN
 SELECT encode(pg_catalog.sha256(convert_to(prosrc,'UTF8')),'hex') INTO actual
 FROM pg_proc WHERE oid=to_regprocedure('vec_contexto_actor_v1.crear_perfil_vinculo_admin_v1(text,text,numeric,numeric,text,text,text,numeric,text,timestamptz)')
 AND proowner=to_regrole('vec_contexto_actor_v1_propietario') AND prosecdef;
 IF actual IS DISTINCT FROM '649163c68f2072983155bc479e20a98820946f7362449979824278de88d9ef5f' THEN
  RAISE EXCEPTION 'CA33: clave=CA20.prosrc_sha256 actual=% esperado=649163c68f2072983155bc479e20a98820946f7362449979824278de88d9ef5f',coalesce(actual,'ausente') USING ERRCODE='55000';
 END IF;
END $ca20_preimagen$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;

CREATE TABLE vec_contexto_actor_v1.fuentes_iniciales_admin_v1(
 operacion_ref text PRIMARY KEY,plan_sha256 text NOT NULL CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 aprobacion_ref text NOT NULL,plan jsonb NOT NULL,recibo_is jsonb NOT NULL,recibo jsonb NOT NULL,
 registrada_en timestamptz NOT NULL DEFAULT clock_timestamp()
);
-- Titularidad inicial independiente de perfiles. No concede ninguna acción.
CREATE TABLE vec_contexto_actor_v1.titularidad_cuenta_persona_v1(
 cuenta_ref text PRIMARY KEY,persona_ref text NOT NULL,
 version numeric(20,0) NOT NULL CHECK(version=1),
 fuente_ref text NOT NULL,fuente_version numeric(20,0) NOT NULL CHECK(fuente_version=1),
 fuente_sha256 text NOT NULL CHECK(fuente_sha256 ~ '^[0-9a-f]{64}$'),
 alcance_fuente text NOT NULL CHECK(alcance_fuente='sintetico_declarado'),
 operacion_ref text NOT NULL REFERENCES vec_contexto_actor_v1.fuentes_iniciales_admin_v1(operacion_ref) DEFERRABLE INITIALLY DEFERRED,
 vigente_desde timestamptz NOT NULL,vigente_hasta timestamptz NOT NULL,
 CHECK(vec_contexto_actor_v1.referencia_valida(cuenta_ref,'cta_') IS TRUE),
 CHECK(vec_contexto_actor_v1.referencia_valida(persona_ref,'per_') IS TRUE),
 CHECK(isfinite(vigente_desde) AND isfinite(vigente_hasta) AND vigente_hasta>vigente_desde),
 FOREIGN KEY(cuenta_ref,version) REFERENCES vec_contexto_actor_v1.proyeccion_cuenta_versiones(cuenta_ref,version),
 FOREIGN KEY(persona_ref,version) REFERENCES vec_contexto_actor_v1.persona_versiones(persona_ref,version),
 FOREIGN KEY(fuente_ref,fuente_version) REFERENCES vec_contexto_actor_v1.procedencias(procedencia_ref,procedencia_version)
);
DO $tables$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['fuentes_iniciales_admin_v1','titularidad_cuenta_persona_v1'] LOOP
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.%I FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia()',t);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_contexto_actor_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado()',t);
  EXECUTE format('ALTER TABLE vec_contexto_actor_v1.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_contexto_actor_v1.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_contexto_actor_v1.%I TO vec_contexto_actor_v1_propietario USING(current_user=%L) WITH CHECK(current_user=%L)',t,'vec_contexto_actor_v1_propietario','vec_contexto_actor_v1_propietario');
  EXECUTE format('REVOKE ALL ON TABLE vec_contexto_actor_v1.%I FROM PUBLIC',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_contexto_actor_v1.%I FROM PUBLIC',t);
 END LOOP;
END $tables$;

CREATE FUNCTION vec_contexto_actor_v1.preimagen_fuentes_iniciales_admin_v1(p jsonb)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE r record; pe jsonb;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'CA33: requiere SERIALIZABLE escritura' USING ERRCODE='25000'; END IF;
 PERFORM vec_identidad_sesiones_v1.validar_plan_fuentes_iniciales_admin_v1(p);
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:fuentes-iniciales:v1',0));
 -- Mismo protocolo de los triggers instalados: bloquea antes de observar
 -- cualquier puntero, conserva generaciones y revalidación MVCC existentes.
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 SELECT * INTO r FROM vec_contexto_actor_v1.fuentes_iniciales_admin_v1 WHERE operacion_ref=p->>'operacion_ref';
 IF FOUND THEN
  IF r.plan IS DISTINCT FROM p THEN RAISE EXCEPTION 'CA33: replay divergente' USING ERRCODE='40001'; END IF;
  RETURN jsonb_build_object('esquema','vec.ca.preimagen-fuentes-admin.v1','version',1,'operacion_ref',p->>'operacion_ref','recibo',r.recibo);
 END IF;
 IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.organizacion_versiones WHERE organizacion_ref=p#>>'{organizacion,organizacion_ref}')
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.persona_versiones v JOIN LATERAL jsonb_array_elements(p->'personas') personas_plan(value) ON v.persona_ref=personas_plan.value->>'persona_ref')
 THEN RAISE EXCEPTION 'CA33: CAS inicial divergente' USING ERRCODE='40001'; END IF;
 FOR pe IN SELECT p->'procedencia' UNION ALL SELECT value->'fuente_titularidad' FROM jsonb_array_elements(p->'personas') LOOP
  PERFORM pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:procedencia:v1:'||(pe->>'referencia'),0));
  IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.procedencias WHERE procedencia_ref=pe->>'referencia')
  THEN RAISE EXCEPTION 'CA33: procedencia inicial ya existente' USING ERRCODE='40001'; END IF;
 END LOOP;
 RETURN jsonb_build_object('esquema','vec.ca.preimagen-fuentes-admin.v1','version',1,'operacion_ref',p->>'operacion_ref','organizacion',NULL,'personas',jsonb_build_array(),'procedencias',jsonb_build_array());
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.preimagen_fuentes_iniciales_admin_v1(jsonb) FROM PUBLIC,vec_contexto_actor_v1_runtime;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.preimagen_fuentes_iniciales_admin_v1(jsonb) TO vec_autorizacion_propietario;

CREATE FUNCTION vec_contexto_actor_v1.confirmar_fuentes_iniciales_admin_v1(p jsonb,recibo_is jsonb,pre_sha text,operacion text,plan_sha text,aprobacion text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE pre jsonb; real_is jsonb; pe jsonb; cuenta jsonb; cta text; procedencia jsonb; ahora timestamptz; recibo jsonb; datos jsonb:='[]'::jsonb; existente record;
BEGIN
 pre:=vec_contexto_actor_v1.preimagen_fuentes_iniciales_admin_v1(p);
 IF operacion IS DISTINCT FROM p->>'operacion_ref' OR plan_sha IS NULL OR plan_sha !~ '^[0-9a-f]{64}$'
 OR aprobacion IS NULL OR octet_length(aprobacion) NOT BETWEEN 1 AND 128
 THEN RAISE EXCEPTION 'CA33: compromiso inválido' USING ERRCODE='22023'; END IF;
 real_is:=vec_identidad_sesiones_v1.cotejar_recibo_fuentes_iniciales_admin_v1(operacion,plan_sha,aprobacion);
 IF real_is IS DISTINCT FROM recibo_is THEN RAISE EXCEPTION 'CA33: recibo IS no acreditado' USING ERRCODE='55000'; END IF;
 SELECT * INTO existente FROM vec_contexto_actor_v1.fuentes_iniciales_admin_v1 WHERE operacion_ref=operacion;
 IF FOUND THEN
  IF existente.plan_sha256 IS DISTINCT FROM plan_sha OR existente.aprobacion_ref IS DISTINCT FROM aprobacion OR existente.recibo_is IS DISTINCT FROM real_is
  THEN RAISE EXCEPTION 'CA33: replay divergente' USING ERRCODE='40001'; END IF;
  RETURN existente.recibo;
 END IF;
 IF pre_sha IS DISTINCT FROM encode(pg_catalog.sha256(convert_to(pre::text,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'CA33: CAS preimagen divergente' USING ERRCODE='40001'; END IF;
 IF jsonb_array_length(real_is#>'{datos,personas}')<>2
 THEN RAISE EXCEPTION 'CA33: cardinalidad IS divergente' USING ERRCODE='55000'; END IF;
 -- Todas las colisiones de proyección/titularidad se cotejan ANTES de escribir.
 FOR pe IN SELECT value FROM jsonb_array_elements(p->'personas') LOOP
  SELECT value INTO STRICT cuenta FROM jsonb_array_elements(real_is#>'{datos,personas}') WHERE value->>'persona_ref'=pe->>'persona_ref';
  FOR cta IN SELECT cuenta->>'cuenta_ordinaria_ref' UNION ALL SELECT cuenta->>'cuenta_privilegiada_ref' LOOP
   IF vec_contexto_actor_v1.referencia_valida(cta,'cta_') IS NOT TRUE
   OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones WHERE cuenta_ref=cta)
   OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.titularidad_cuenta_persona_v1 WHERE cuenta_ref=cta)
   THEN RAISE EXCEPTION 'CA33: cuenta real ya proyectada' USING ERRCODE='40001'; END IF;
  END LOOP;
 END LOOP;
 ahora:=clock_timestamp(); procedencia:=p->'procedencia';
 FOR pe IN SELECT p->'procedencia' UNION SELECT value->'fuente_titularidad' FROM jsonb_array_elements(p->'personas') LOOP
  INSERT INTO vec_contexto_actor_v1.procedencias VALUES(pe->>'referencia',1,pe->>'huella_sha256','autoridad_maestra_acreditada');
 END LOOP;
 INSERT INTO vec_contexto_actor_v1.organizacion_versiones(organizacion_ref,version,procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES(p#>>'{organizacion,organizacion_ref}',1,procedencia->>'referencia',1,procedencia->>'huella_sha256','autoridad_maestra_acreditada','activo',ahora,(p#>>'{organizacion,vigente_hasta}')::timestamptz);
 INSERT INTO vec_contexto_actor_v1.organizacion_actual VALUES(p#>>'{organizacion,organizacion_ref}',1);
 FOR pe IN SELECT value FROM jsonb_array_elements(p->'personas') ORDER BY value->>'persona_ref' LOOP
  INSERT INTO vec_contexto_actor_v1.persona_versiones(persona_ref,version,procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
  VALUES(pe->>'persona_ref',1,procedencia->>'referencia',1,procedencia->>'huella_sha256','autoridad_maestra_acreditada','activo',ahora,(pe->>'vigente_hasta')::timestamptz);
  INSERT INTO vec_contexto_actor_v1.persona_actual VALUES(pe->>'persona_ref',1);
  SELECT value INTO STRICT cuenta FROM jsonb_array_elements(real_is#>'{datos,personas}') WHERE value->>'persona_ref'=pe->>'persona_ref';
  FOR cta IN SELECT cuenta->>'cuenta_ordinaria_ref' UNION ALL SELECT cuenta->>'cuenta_privilegiada_ref' LOOP
   INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones(cuenta_ref,version,procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
   VALUES(cta,1,procedencia->>'referencia',1,procedencia->>'huella_sha256','autoridad_maestra_acreditada','activo',ahora,(pe->>'vigente_hasta')::timestamptz);
   INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES(cta,1);
   INSERT INTO vec_contexto_actor_v1.titularidad_cuenta_persona_v1(cuenta_ref,persona_ref,version,fuente_ref,fuente_version,fuente_sha256,alcance_fuente,operacion_ref,vigente_desde,vigente_hasta)
   VALUES(cta,pe->>'persona_ref',1,pe#>>'{fuente_titularidad,referencia}',1,pe#>>'{fuente_titularidad,huella_sha256}','sintetico_declarado',operacion,ahora,(pe->>'vigente_hasta')::timestamptz);
  END LOOP;
  datos:=datos||jsonb_build_array(cuenta||jsonb_build_object('persona_version',1,'proyeccion_cuenta_version',1));
 END LOOP;
 recibo:=jsonb_build_object('esquema','vec.ca.fuentes-iniciales-admin.v1','version',1,'recibo_ref','recibo_ca_fuentes:'||replace(pg_catalog.gen_random_uuid()::text,'-',''),'operacion_ref',operacion,'plan_sha256',plan_sha,'aprobacion_ref',aprobacion,'alcance_fuente','sintetico_declarado','registrada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'datos',jsonb_build_object('organizacion_ref',p#>>'{organizacion,organizacion_ref}','organizacion_version',1,'personas',datos));
 recibo:=recibo||jsonb_build_object('huella_sha256',encode(pg_catalog.sha256(convert_to(recibo::text,'UTF8')),'hex'));
 INSERT INTO vec_contexto_actor_v1.fuentes_iniciales_admin_v1(operacion_ref,plan_sha256,aprobacion_ref,plan,recibo_is,recibo) VALUES(operacion,plan_sha,aprobacion,p,real_is,recibo);
 RETURN recibo;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.confirmar_fuentes_iniciales_admin_v1(jsonb,jsonb,text,text,text,text) FROM PUBLIC,vec_contexto_actor_v1_runtime;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.confirmar_fuentes_iniciales_admin_v1(jsonb,jsonb,text,text,text,text) TO vec_autorizacion_propietario;

-- Mantiene todas las guardas de CA20; permite la fuente inicial sin perfil.
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.crear_perfil_vinculo_admin_v1(
  p_cuenta_ref text, p_persona_ref text, p_cuenta_version numeric, p_persona_version numeric,
  p_perfil_ref text, p_vinculo_ref text, p_procedencia_ref text,
  p_procedencia_version numeric, p_procedencia_huella text, p_vigente_hasta timestamptz
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp AS $funcion$
DECLARE c record; pe record; instante timestamptz;
BEGIN
  IF current_setting('transaction_isolation') <> 'serializable'
     OR current_setting('transaction_read_only') <> 'off' THEN
    RAISE EXCEPTION 'CA20: requiere SERIALIZABLE de escritura' USING ERRCODE = '25000';
  END IF;
  IF vec_contexto_actor_v1.referencia_valida(p_cuenta_ref,'cta_') IS NOT TRUE
     OR vec_contexto_actor_v1.referencia_valida(p_persona_ref,'per_') IS NOT TRUE
     OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref,'prf_') IS NOT TRUE
     OR vec_contexto_actor_v1.referencia_valida(p_vinculo_ref,'vca_') IS NOT TRUE
     OR p_cuenta_version IS NULL OR p_persona_version IS NULL
     OR vec_contexto_actor_v1.instante_valido(p_vigente_hasta) IS NOT TRUE THEN
    RAISE EXCEPTION 'CA20: alta invalida' USING ERRCODE = '22023';
  END IF;
  PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
  SELECT cv.* INTO c FROM vec_contexto_actor_v1.proyeccion_cuenta_actual ca
    JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones cv USING (cuenta_ref,version)
    WHERE ca.cuenta_ref=p_cuenta_ref FOR UPDATE OF ca;
  IF NOT FOUND THEN RAISE EXCEPTION 'CA20: cuenta no vigente' USING ERRCODE = 'P0002'; END IF;
  SELECT pv.* INTO pe FROM vec_contexto_actor_v1.persona_actual pa
    JOIN vec_contexto_actor_v1.persona_versiones pv USING (persona_ref,version)
    WHERE pa.persona_ref=p_persona_ref FOR UPDATE OF pa;
  IF NOT FOUND THEN RAISE EXCEPTION 'CA20: persona no vigente' USING ERRCODE = 'P0002'; END IF;
  instante := pg_catalog.clock_timestamp();
  IF c.version <> p_cuenta_version OR pe.version <> p_persona_version THEN
    RAISE EXCEPTION 'CA20: version obsoleta' USING ERRCODE = '40001';
  END IF;
  IF c.estado <> 'activo' OR pe.estado <> 'activo'
     OR c.procedencia_autoridad <> 'autoridad_maestra_acreditada'
     OR pe.procedencia_autoridad <> 'autoridad_maestra_acreditada'
     OR instante < GREATEST(c.vigente_desde,pe.vigente_desde)
     OR instante >= LEAST(c.vigente_hasta,pe.vigente_hasta)
     OR p_vigente_hasta <= instante
     OR NOT EXISTS (SELECT 1 FROM vec_contexto_actor_v1.procedencias
       WHERE procedencia_ref=p_procedencia_ref AND procedencia_version=p_procedencia_version
         AND procedencia_huella_sha256=p_procedencia_huella
         AND procedencia_autoridad='autoridad_maestra_acreditada')
     OR NOT (EXISTS (
       SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual va
       JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv USING (vinculo_ref,version)
       WHERE vv.cuenta_ref=p_cuenta_ref AND vv.persona_ref=p_persona_ref
         AND vv.estado='activo' AND vv.procedencia_autoridad='autoridad_maestra_acreditada'
         AND instante>=vv.vigente_desde AND instante<vv.vigente_hasta)
       OR EXISTS (
         SELECT 1 FROM vec_contexto_actor_v1.titularidad_cuenta_persona_v1 t
         JOIN vec_contexto_actor_v1.fuentes_iniciales_admin_v1 f USING(operacion_ref)
         WHERE t.cuenta_ref=p_cuenta_ref AND t.persona_ref=p_persona_ref
           AND t.version=1 AND t.alcance_fuente='sintetico_declarado'
           AND instante>=t.vigente_desde AND instante<t.vigente_hasta
       ))
     OR EXISTS (SELECT 1 FROM vec_contexto_actor_v1.perfil_actual WHERE perfil_ref=p_perfil_ref)
     OR EXISTS (SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual WHERE vinculo_ref=p_vinculo_ref) THEN
    RAISE EXCEPTION 'CA20: alta sin preimagen acreditada' USING ERRCODE = '55000';
  END IF;
  INSERT INTO vec_contexto_actor_v1.perfil_versiones
    (perfil_ref,version,persona_ref,procedencia_ref,procedencia_version,procedencia_huella_sha256,
     procedencia_autoridad,estado,vigente_desde,vigente_hasta)
  VALUES (p_perfil_ref,1,p_persona_ref,p_procedencia_ref,p_procedencia_version,p_procedencia_huella,
          'autoridad_maestra_acreditada','activo',instante,p_vigente_hasta);
  INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones
    (vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref,procedencia_ref,procedencia_version,
     procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
  VALUES (p_vinculo_ref,1,p_cuenta_ref,p_perfil_ref,p_persona_ref,p_procedencia_ref,p_procedencia_version,
          p_procedencia_huella,'autoridad_maestra_acreditada','activo',instante,p_vigente_hasta);
  INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES (p_perfil_ref,1);
  INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES (p_vinculo_ref,1);
  RETURN true;
END $funcion$;
COMMIT;
