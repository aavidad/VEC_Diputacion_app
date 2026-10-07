\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:migracion:000020', 0));

-- Preimagen causal: propietario, superficie ACL y protección de las tablas
-- que consume esta migración. No depende del LOGIN o entorno del ensayo.
DO $frontera_preimagen$
DECLARE nombre text;tabla regclass;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_contexto_actor_v1_propietario' AND NOT(rolcanlogin OR rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls))
 OR NOT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname='vec_contexto_actor_v1' AND nspowner=to_regrole('vec_contexto_actor_v1_propietario'))
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a WHERE n.nspname='vec_contexto_actor_v1' AND (a.grantee=0 OR (a.privilege_type='CREATE' AND a.grantee<>n.nspowner)))
 THEN RAISE EXCEPTION 'ADMIN: namespace o propietario divergente' USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY ARRAY['persona_actual','persona_versiones','proyeccion_cuenta_actual','proyeccion_cuenta_versiones','perfil_actual','perfil_versiones','vinculo_contexto_actual','vinculo_contexto_versiones'] LOOP
  tabla:=to_regclass('vec_contexto_actor_v1.'||nombre);
  IF tabla IS NULL OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=tabla AND relowner=to_regrole('vec_contexto_actor_v1_propietario') AND relrowsecurity=false AND relforcerowsecurity=false)
  OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a WHERE c.oid=tabla AND a.grantee<>c.relowner)
  THEN RAISE EXCEPTION 'ADMIN: tabla o ACL divergente %',nombre USING ERRCODE='55000'; END IF;
 END LOOP;
END $frontera_preimagen$;

DO $preimagen$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = current_user AND rolsuper)
     OR pg_catalog.to_regrole('vec_contexto_actor_v1_propietario') IS NULL
     OR pg_catalog.to_regclass('vec_contexto_actor_v1.perfil_actual') IS NULL
     OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.impedir_revivir_perfil_v1()') IS NULL
     OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.bloquear_contexto_admin_v1(text,text,text,text,numeric,numeric,numeric,numeric)') IS NOT NULL
  THEN RAISE EXCEPTION 'CA20: preimagen incompatible' USING ERRCODE = '55000'; END IF;
END $preimagen$;

SET LOCAL ROLE vec_contexto_actor_v1_propietario;

-- Guarda transaccional de la autoridad de contexto. La barrera es la misma
-- que AUT23; el consumidor futuro debe avanzar allí su revisión antes de
-- invocar esta fachada, siempre dentro de una sola transacción.
CREATE FUNCTION vec_contexto_actor_v1.bloquear_contexto_admin_v1(
  p_cuenta_ref text, p_persona_ref text, p_perfil_ref text, p_vinculo_ref text,
  p_cuenta_version numeric, p_persona_version numeric,
  p_perfil_version numeric, p_vinculo_version numeric
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp AS $funcion$
DECLARE c record; pe record; p record; v record; instante timestamptz;
BEGIN
  IF vec_contexto_actor_v1.referencia_valida(p_cuenta_ref,'cta_') IS NOT TRUE
     OR vec_contexto_actor_v1.referencia_valida(p_persona_ref,'per_') IS NOT TRUE
     OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref,'prf_') IS NOT TRUE
     OR vec_contexto_actor_v1.referencia_valida(p_vinculo_ref,'vca_') IS NOT TRUE
     OR p_cuenta_version IS NULL OR p_persona_version IS NULL
     OR p_perfil_version IS NULL OR p_vinculo_version IS NULL THEN
    RAISE EXCEPTION 'CA20: referencias o versiones invalidas' USING ERRCODE = '22023';
  END IF;
  PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
  SELECT cv.* INTO c FROM vec_contexto_actor_v1.proyeccion_cuenta_actual ca
    JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones cv USING (cuenta_ref,version)
    WHERE ca.cuenta_ref=p_cuenta_ref FOR UPDATE OF ca;
  IF NOT FOUND THEN RAISE EXCEPTION 'CA20: contexto no vigente' USING ERRCODE = 'P0002'; END IF;
  SELECT pv.* INTO p FROM vec_contexto_actor_v1.perfil_actual pa
    JOIN vec_contexto_actor_v1.perfil_versiones pv USING (perfil_ref,version)
    WHERE pa.perfil_ref=p_perfil_ref FOR UPDATE OF pa;
  IF NOT FOUND THEN RAISE EXCEPTION 'CA20: contexto no vigente' USING ERRCODE = 'P0002'; END IF;
  SELECT vv.* INTO v FROM vec_contexto_actor_v1.vinculo_contexto_actual va
    JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv USING (vinculo_ref,version)
    WHERE va.vinculo_ref=p_vinculo_ref FOR UPDATE OF va;
  IF NOT FOUND THEN RAISE EXCEPTION 'CA20: contexto no vigente' USING ERRCODE = 'P0002'; END IF;
  SELECT pv.* INTO pe FROM vec_contexto_actor_v1.persona_actual pa
    JOIN vec_contexto_actor_v1.persona_versiones pv USING (persona_ref,version)
    WHERE pa.persona_ref=p_persona_ref FOR UPDATE OF pa;
  IF NOT FOUND THEN RAISE EXCEPTION 'CA20: contexto no vigente' USING ERRCODE = 'P0002'; END IF;
  instante := pg_catalog.clock_timestamp();
  IF c.version <> p_cuenta_version OR pe.version <> p_persona_version
     OR p.version <> p_perfil_version OR v.version <> p_vinculo_version THEN
    RAISE EXCEPTION 'CA20: version obsoleta' USING ERRCODE = '40001';
  END IF;
  IF p.persona_ref <> p_persona_ref OR v.persona_ref <> p_persona_ref
     OR v.cuenta_ref <> p_cuenta_ref OR v.perfil_ref <> p_perfil_ref
     OR c.estado <> 'activo' OR pe.estado <> 'activo'
     OR p.estado <> 'activo' OR v.estado <> 'activo'
     OR c.procedencia_autoridad <> 'autoridad_maestra_acreditada'
     OR pe.procedencia_autoridad <> 'autoridad_maestra_acreditada'
     OR p.procedencia_autoridad <> 'autoridad_maestra_acreditada'
     OR v.procedencia_autoridad <> 'autoridad_maestra_acreditada'
     OR instante < GREATEST(c.vigente_desde,pe.vigente_desde,p.vigente_desde,v.vigente_desde)
     OR instante >= LEAST(c.vigente_hasta,pe.vigente_hasta,p.vigente_hasta,v.vigente_hasta) THEN
    RAISE EXCEPTION 'CA20: contexto no vigente' USING ERRCODE = 'P0002';
  END IF;
  RETURN true;
END $funcion$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.bloquear_contexto_admin_v1(text,text,text,text,numeric,numeric,numeric,numeric) FROM PUBLIC, vec_contexto_actor_v1_runtime;

-- El alta no deduce identidad de un rol. Exige persona/cuenta ya acreditadas,
-- una procedencia maestra preexistente y referencias nuevas. La asignación V3
-- y el acto auditado pertenecen a AUT24, no a estas tablas.
CREATE FUNCTION vec_contexto_actor_v1.crear_perfil_vinculo_admin_v1(
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
     OR NOT EXISTS (
       SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual va
       JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv USING (vinculo_ref,version)
       WHERE vv.cuenta_ref=p_cuenta_ref AND vv.persona_ref=p_persona_ref
         AND vv.estado='activo' AND vv.procedencia_autoridad='autoridad_maestra_acreditada'
         AND instante>=vv.vigente_desde AND instante<vv.vigente_hasta)
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
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.crear_perfil_vinculo_admin_v1(text,text,numeric,numeric,text,text,text,numeric,text,timestamptz) FROM PUBLIC, vec_contexto_actor_v1_runtime;

CREATE FUNCTION vec_contexto_actor_v1.revocar_perfil_vinculo_admin_v1(
  p_cuenta_ref text, p_persona_ref text, p_perfil_ref text, p_vinculo_ref text,
  p_cuenta_version numeric, p_persona_version numeric,
  p_perfil_version numeric, p_vinculo_version numeric,
  p_procedencia_ref text, p_procedencia_version numeric, p_procedencia_huella text
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp AS $funcion$
DECLARE p record; v record; instante timestamptz;
BEGIN
  IF current_setting('transaction_isolation') <> 'serializable'
     OR current_setting('transaction_read_only') <> 'off' THEN
    RAISE EXCEPTION 'CA20: requiere SERIALIZABLE de escritura' USING ERRCODE = '25000';
  END IF;
  PERFORM vec_contexto_actor_v1.bloquear_contexto_admin_v1(
    p_cuenta_ref,p_persona_ref,p_perfil_ref,p_vinculo_ref,
    p_cuenta_version,p_persona_version,p_perfil_version,p_vinculo_version);
  IF p_perfil_version >= 18446744073709551615::numeric
     OR p_vinculo_version >= 18446744073709551615::numeric
     OR NOT EXISTS (SELECT 1 FROM vec_contexto_actor_v1.procedencias
       WHERE procedencia_ref=p_procedencia_ref AND procedencia_version=p_procedencia_version
         AND procedencia_huella_sha256=p_procedencia_huella
         AND procedencia_autoridad='autoridad_maestra_acreditada') THEN
    RAISE EXCEPTION 'CA20: revocacion sin preimagen acreditada' USING ERRCODE = '55000';
  END IF;
  SELECT * INTO STRICT p FROM vec_contexto_actor_v1.perfil_versiones
    WHERE perfil_ref=p_perfil_ref AND version=p_perfil_version;
  SELECT * INTO STRICT v FROM vec_contexto_actor_v1.vinculo_contexto_versiones
    WHERE vinculo_ref=p_vinculo_ref AND version=p_vinculo_version;
  IF (p.procedencia_ref,p.procedencia_version,p.procedencia_huella_sha256)
     IS NOT DISTINCT FROM (p_procedencia_ref,p_procedencia_version,p_procedencia_huella)
     OR (v.procedencia_ref,v.procedencia_version,v.procedencia_huella_sha256)
     IS NOT DISTINCT FROM (p_procedencia_ref,p_procedencia_version,p_procedencia_huella) THEN
    RAISE EXCEPTION 'CA20: revocacion requiere procedencia nueva' USING ERRCODE = '55000';
  END IF;
  instante := pg_catalog.clock_timestamp();
  IF instante >= LEAST(p.vigente_hasta,v.vigente_hasta) THEN
    RAISE EXCEPTION 'CA20: vigencia agotada' USING ERRCODE = 'P0002';
  END IF;
  IF EXISTS (
    SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual va
    JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv USING (vinculo_ref,version)
    WHERE vv.perfil_ref=p_perfil_ref AND vv.vinculo_ref<>p_vinculo_ref
      AND vv.estado='activo' AND instante>=vv.vigente_desde AND instante<vv.vigente_hasta
  ) THEN
    RAISE EXCEPTION 'CA20: perfil con otro vinculo vigente' USING ERRCODE = '55000';
  END IF;
  INSERT INTO vec_contexto_actor_v1.perfil_versiones
    (perfil_ref,version,persona_ref,procedencia_ref,procedencia_version,procedencia_huella_sha256,
     procedencia_autoridad,estado,vigente_desde,vigente_hasta)
  VALUES (p_perfil_ref,p_perfil_version+1,p_persona_ref,p_procedencia_ref,p_procedencia_version,p_procedencia_huella,
          'autoridad_maestra_acreditada','revocado',instante,p.vigente_hasta);
  INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones
    (vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref,procedencia_ref,procedencia_version,
     procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
  VALUES (p_vinculo_ref,p_vinculo_version+1,p_cuenta_ref,p_perfil_ref,p_persona_ref,p_procedencia_ref,
          p_procedencia_version,p_procedencia_huella,'autoridad_maestra_acreditada','revocado',instante,v.vigente_hasta);
  UPDATE vec_contexto_actor_v1.perfil_actual SET version=p_perfil_version+1
    WHERE perfil_ref=p_perfil_ref AND version=p_perfil_version;
  IF NOT FOUND THEN RAISE EXCEPTION 'CA20: CAS perfil perdido' USING ERRCODE = '40001'; END IF;
  UPDATE vec_contexto_actor_v1.vinculo_contexto_actual SET version=p_vinculo_version+1
    WHERE vinculo_ref=p_vinculo_ref AND version=p_vinculo_version;
  IF NOT FOUND THEN RAISE EXCEPTION 'CA20: CAS vinculo perdido' USING ERRCODE = '40001'; END IF;
  RETURN true;
END $funcion$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.revocar_perfil_vinculo_admin_v1(text,text,text,text,numeric,numeric,numeric,numeric,text,numeric,text) FROM PUBLIC, vec_contexto_actor_v1_runtime;
COMMIT;
