\set ON_ERROR_STOP on
-- CA35: efecto privado de lote ADMIN ordinario con instante unico.
-- No altera ni concede las fachadas CA20 de actos singulares.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regrole('vec_contexto_actor_v1_propietario') IS NULL
 OR to_regrole('vec_autorizacion_propietario') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.bloquear_contexto_admin_v1(text,text,text,text,numeric,numeric,numeric,numeric)') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.crear_perfil_vinculo_admin_v1(text,text,numeric,numeric,text,text,text,numeric,text,timestamptz)') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.revocar_perfil_vinculo_admin_v1(text,text,text,text,numeric,numeric,numeric,numeric,text,numeric,text)') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.crear_perfil_vinculo_admin_lote_v1(text,text,numeric,numeric,text,text,text,numeric,text,timestamptz,timestamptz,timestamptz)') IS NOT NULL
 OR to_regprocedure('vec_contexto_actor_v1.revocar_perfil_vinculo_admin_lote_v1(text,text,text,text,numeric,numeric,numeric,numeric,text,numeric,text,timestamptz)') IS NOT NULL
 THEN RAISE EXCEPTION 'CA35: preimagen CA20 incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;

CREATE FUNCTION vec_contexto_actor_v1.crear_perfil_vinculo_admin_lote_v1(
 p_cuenta_ref text,p_persona_ref text,p_cuenta_version numeric,p_persona_version numeric,
 p_perfil_ref text,p_vinculo_ref text,p_procedencia_ref text,p_procedencia_version numeric,
 p_procedencia_huella text,p_vigente_desde timestamptz,p_vigente_hasta timestamptz,p_instante_lote timestamptz)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE c record;pe record;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR p_instante_lote IS NULL OR NOT isfinite(p_instante_lote) OR p_instante_lote<transaction_timestamp() OR p_instante_lote>clock_timestamp()
 OR p_vigente_desde IS NULL OR NOT isfinite(p_vigente_desde) OR p_vigente_desde>p_instante_lote
 OR p_vigente_hasta IS NULL OR NOT isfinite(p_vigente_hasta) OR p_vigente_hasta<=p_instante_lote
 OR vec_contexto_actor_v1.referencia_valida(p_cuenta_ref,'cta_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(p_persona_ref,'per_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref,'prf_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(p_vinculo_ref,'vca_') IS NOT TRUE
 OR p_cuenta_version IS NULL OR p_persona_version IS NULL
 THEN RAISE EXCEPTION 'CA35: alta invalida' USING ERRCODE='22023'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 SELECT cv.* INTO c FROM vec_contexto_actor_v1.proyeccion_cuenta_actual ca
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones cv USING(cuenta_ref,version)
 WHERE ca.cuenta_ref=p_cuenta_ref FOR UPDATE OF ca;
 IF NOT FOUND THEN RAISE EXCEPTION 'CA35: cuenta no vigente' USING ERRCODE='P0002'; END IF;
 SELECT pv.* INTO pe FROM vec_contexto_actor_v1.persona_actual pa
 JOIN vec_contexto_actor_v1.persona_versiones pv USING(persona_ref,version)
 WHERE pa.persona_ref=p_persona_ref FOR UPDATE OF pa;
 IF NOT FOUND THEN RAISE EXCEPTION 'CA35: persona no vigente' USING ERRCODE='P0002'; END IF;
 IF c.version<>p_cuenta_version OR pe.version<>p_persona_version THEN
  RAISE EXCEPTION 'CA35: version obsoleta' USING ERRCODE='40001'; END IF;
 IF c.estado<>'activo' OR pe.estado<>'activo'
 OR c.procedencia_autoridad<>'autoridad_maestra_acreditada' OR pe.procedencia_autoridad<>'autoridad_maestra_acreditada'
 OR p_instante_lote<GREATEST(c.vigente_desde,pe.vigente_desde)
 OR p_instante_lote>=LEAST(c.vigente_hasta,pe.vigente_hasta)
 OR p_vigente_desde<GREATEST(c.vigente_desde,pe.vigente_desde)
 OR NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.procedencias
   WHERE procedencia_ref=p_procedencia_ref AND procedencia_version=p_procedencia_version
   AND procedencia_huella_sha256=p_procedencia_huella AND procedencia_autoridad='autoridad_maestra_acreditada')
 OR NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual va
   JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv USING(vinculo_ref,version)
   WHERE vv.cuenta_ref=p_cuenta_ref AND vv.persona_ref=p_persona_ref
   AND vv.estado='activo' AND vv.procedencia_autoridad='autoridad_maestra_acreditada'
   AND p_instante_lote>=vv.vigente_desde AND p_instante_lote<vv.vigente_hasta)
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_actual WHERE perfil_ref=p_perfil_ref)
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual WHERE vinculo_ref=p_vinculo_ref)
 THEN RAISE EXCEPTION 'CA35: alta sin preimagen acreditada' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_contexto_actor_v1.perfil_versiones
 (perfil_ref,version,persona_ref,procedencia_ref,procedencia_version,procedencia_huella_sha256,
 procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES(p_perfil_ref,1,p_persona_ref,p_procedencia_ref,p_procedencia_version,p_procedencia_huella,
 'autoridad_maestra_acreditada','activo',p_vigente_desde,p_vigente_hasta);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones
 (vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref,procedencia_ref,procedencia_version,
 procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES(p_vinculo_ref,1,p_cuenta_ref,p_perfil_ref,p_persona_ref,p_procedencia_ref,p_procedencia_version,
 p_procedencia_huella,'autoridad_maestra_acreditada','activo',p_vigente_desde,p_vigente_hasta);
 INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES(p_perfil_ref,1);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES(p_vinculo_ref,1);
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.crear_perfil_vinculo_admin_lote_v1(text,text,numeric,numeric,text,text,text,numeric,text,timestamptz,timestamptz,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.crear_perfil_vinculo_admin_lote_v1(text,text,numeric,numeric,text,text,text,numeric,text,timestamptz,timestamptz,timestamptz) TO vec_autorizacion_propietario;

CREATE FUNCTION vec_contexto_actor_v1.revocar_perfil_vinculo_admin_lote_v1(
 p_cuenta_ref text,p_persona_ref text,p_perfil_ref text,p_vinculo_ref text,
 p_cuenta_version numeric,p_persona_version numeric,p_perfil_version numeric,p_vinculo_version numeric,
 p_procedencia_ref text,p_procedencia_version numeric,p_procedencia_huella text,p_instante_lote timestamptz)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE p record;v record;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR p_instante_lote IS NULL OR NOT isfinite(p_instante_lote) OR p_instante_lote<transaction_timestamp() OR p_instante_lote>clock_timestamp()
 THEN RAISE EXCEPTION 'CA35: revocacion invalida' USING ERRCODE='22023'; END IF;
 PERFORM vec_contexto_actor_v1.bloquear_contexto_admin_v1(
 p_cuenta_ref,p_persona_ref,p_perfil_ref,p_vinculo_ref,
 p_cuenta_version,p_persona_version,p_perfil_version,p_vinculo_version);
 IF p_perfil_version>=18446744073709551615::numeric OR p_vinculo_version>=18446744073709551615::numeric
 OR NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.procedencias
   WHERE procedencia_ref=p_procedencia_ref AND procedencia_version=p_procedencia_version
   AND procedencia_huella_sha256=p_procedencia_huella AND procedencia_autoridad='autoridad_maestra_acreditada')
 THEN RAISE EXCEPTION 'CA35: revocacion sin preimagen acreditada' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM vec_contexto_actor_v1.perfil_versiones WHERE perfil_ref=p_perfil_ref AND version=p_perfil_version;
 SELECT * INTO STRICT v FROM vec_contexto_actor_v1.vinculo_contexto_versiones WHERE vinculo_ref=p_vinculo_ref AND version=p_vinculo_version;
 IF (p.procedencia_ref,p.procedencia_version,p.procedencia_huella_sha256)
 IS NOT DISTINCT FROM (p_procedencia_ref,p_procedencia_version,p_procedencia_huella)
 OR (v.procedencia_ref,v.procedencia_version,v.procedencia_huella_sha256)
 IS NOT DISTINCT FROM (p_procedencia_ref,p_procedencia_version,p_procedencia_huella)
 OR p_instante_lote<GREATEST(p.vigente_desde,v.vigente_desde)
 OR p_instante_lote>=LEAST(p.vigente_hasta,v.vigente_hasta)
 THEN RAISE EXCEPTION 'CA35: revocacion no vigente' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual va
 JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv USING(vinculo_ref,version)
 WHERE vv.perfil_ref=p_perfil_ref AND vv.vinculo_ref<>p_vinculo_ref AND vv.estado='activo'
 AND p_instante_lote>=vv.vigente_desde AND p_instante_lote<vv.vigente_hasta)
 THEN RAISE EXCEPTION 'CA35: perfil con otro vinculo vigente' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_contexto_actor_v1.perfil_versiones
 (perfil_ref,version,persona_ref,procedencia_ref,procedencia_version,procedencia_huella_sha256,
 procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES(p_perfil_ref,p_perfil_version+1,p_persona_ref,p_procedencia_ref,p_procedencia_version,p_procedencia_huella,
 'autoridad_maestra_acreditada','revocado',p_instante_lote,p.vigente_hasta);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones
 (vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref,procedencia_ref,procedencia_version,
 procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES(p_vinculo_ref,p_vinculo_version+1,p_cuenta_ref,p_perfil_ref,p_persona_ref,p_procedencia_ref,
 p_procedencia_version,p_procedencia_huella,'autoridad_maestra_acreditada','revocado',p_instante_lote,v.vigente_hasta);
 UPDATE vec_contexto_actor_v1.perfil_actual SET version=p_perfil_version+1 WHERE perfil_ref=p_perfil_ref AND version=p_perfil_version;
 IF NOT FOUND THEN RAISE EXCEPTION 'CA35: CAS perfil perdido' USING ERRCODE='40001'; END IF;
 UPDATE vec_contexto_actor_v1.vinculo_contexto_actual SET version=p_vinculo_version+1 WHERE vinculo_ref=p_vinculo_ref AND version=p_vinculo_version;
 IF NOT FOUND THEN RAISE EXCEPTION 'CA35: CAS vinculo perdido' USING ERRCODE='40001'; END IF;
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.revocar_perfil_vinculo_admin_lote_v1(text,text,text,text,numeric,numeric,numeric,numeric,text,numeric,text,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.revocar_perfil_vinculo_admin_lote_v1(text,text,text,text,numeric,numeric,numeric,numeric,text,numeric,text,timestamptz) TO vec_autorizacion_propietario;
COMMIT;
