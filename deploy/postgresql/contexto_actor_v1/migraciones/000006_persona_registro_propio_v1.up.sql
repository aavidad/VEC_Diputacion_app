\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:000006',0));
DO $pre$ BEGIN
 IF current_user<>'vec_contexto_actor_v1_propietario'
    OR to_regclass('vec_contexto_actor_v1.persona_actual') IS NULL
    OR to_regprocedure('vec_contexto_actor_v1.registrar_persona_registro_propio_v1(text,text,text,text,boolean,text,numeric,text,text,timestamptz,timestamptz)') IS NOT NULL
    OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_identidad_sesiones_v1_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'contexto 000006: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Sólo proyecta una cuenta ya provisionada por Identidad. No crea empleo,
-- candidatura, concesiones ni permiso de acceso. El propietario de Identidad
-- lo llama dentro de la MISMA transacción que el registro inicial.
CREATE FUNCTION vec_contexto_actor_v1.registrar_persona_registro_propio_v1(
 p_cuenta text,p_persona text,p_perfil text,p_vinculo text,p_nueva boolean,
 p_procedencia text,p_procedencia_version numeric,p_procedencia_huella text,
 p_procedencia_autoridad text,p_desde timestamptz,p_hasta timestamptz)
RETURNS TABLE(cuenta_version numeric,persona_version numeric,perfil_version numeric,vinculo_version numeric)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_persona record; v_procedencia record; v_cuenta record;
BEGIN
 IF current_user<>'vec_contexto_actor_v1_propietario'
    OR NOT pg_has_role(session_user,'vec_identidad_sesiones_v1_provisionador','MEMBER')
    OR NOT vec_contexto_actor_v1.referencia_valida(p_cuenta,'cta_')
    OR NOT vec_contexto_actor_v1.referencia_valida(p_persona,'per_')
    OR NOT vec_contexto_actor_v1.referencia_valida(p_perfil,'prf_')
    OR NOT vec_contexto_actor_v1.referencia_valida(p_vinculo,'vca_')
    OR NOT vec_contexto_actor_v1.procedencia_valida(p_procedencia,p_procedencia_version,p_procedencia_huella,p_procedencia_autoridad)
    OR p_procedencia_autoridad<>'autoridad_maestra_acreditada'
    OR p_nueva IS NULL OR p_desde IS NULL OR p_hasta IS NULL OR p_hasta<=p_desde
    OR clock_timestamp()<p_desde OR clock_timestamp()>=p_hasta
 THEN RAISE EXCEPTION 'registro propio: proyección denegada' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:registro-propio:persona:'||p_persona,0));
 SELECT * INTO v_procedencia FROM vec_contexto_actor_v1.procedencias
  WHERE procedencia_ref=p_procedencia AND procedencia_version=p_procedencia_version FOR SHARE;
 IF FOUND THEN
  IF v_procedencia.procedencia_huella_sha256<>p_procedencia_huella OR v_procedencia.procedencia_autoridad<>p_procedencia_autoridad
  THEN RAISE EXCEPTION 'registro propio: procedencia divergente' USING ERRCODE='23505'; END IF;
 ELSE
  INSERT INTO vec_contexto_actor_v1.procedencias(procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)
   VALUES(p_procedencia,p_procedencia_version,p_procedencia_huella,p_procedencia_autoridad);
 END IF;
 SELECT a.version,v.estado,v.vigente_desde,v.vigente_hasta INTO v_cuenta
  FROM vec_contexto_actor_v1.proyeccion_cuenta_actual a JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones v USING(cuenta_ref,version)
  WHERE a.cuenta_ref=p_cuenta FOR SHARE OF a;
 IF FOUND THEN
  IF v_cuenta.estado<>'activo' OR v_cuenta.vigente_desde>clock_timestamp() OR v_cuenta.vigente_hasta<=clock_timestamp()
  THEN RAISE EXCEPTION 'registro propio: cuenta revocada' USING ERRCODE='42501'; END IF;
  IF EXISTS(
   SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual a
   JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
   WHERE v.cuenta_ref=p_cuenta AND v.persona_ref<>p_persona
  ) THEN RAISE EXCEPTION 'registro propio: cuenta vinculada a otra persona' USING ERRCODE='42501'; END IF;
  cuenta_version:=v_cuenta.version;
 ELSE
  INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones(cuenta_ref,version,procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
   VALUES(p_cuenta,1,p_procedencia,p_procedencia_version,p_procedencia_huella,p_procedencia_autoridad,'activo',p_desde,p_hasta);
  INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual(cuenta_ref,version) VALUES(p_cuenta,1);
  cuenta_version:=1;
 END IF;
 SELECT a.version,v.estado,v.vigente_desde,v.vigente_hasta INTO v_persona
  FROM vec_contexto_actor_v1.persona_actual a JOIN vec_contexto_actor_v1.persona_versiones v USING(persona_ref,version)
  WHERE a.persona_ref=p_persona FOR SHARE OF a;
 IF p_nueva THEN
  IF FOUND THEN RAISE EXCEPTION 'registro propio: persona existente' USING ERRCODE='23505'; END IF;
  INSERT INTO vec_contexto_actor_v1.persona_versiones(persona_ref,version,procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
   VALUES(p_persona,1,p_procedencia,p_procedencia_version,p_procedencia_huella,p_procedencia_autoridad,'activo',p_desde,p_hasta);
  INSERT INTO vec_contexto_actor_v1.persona_actual(persona_ref,version) VALUES(p_persona,1);
  persona_version:=1;
 ELSE
  IF NOT FOUND OR v_persona.estado<>'activo' OR v_persona.vigente_desde>clock_timestamp() OR v_persona.vigente_hasta<=clock_timestamp()
  THEN RAISE EXCEPTION 'registro propio: persona ausente o revocada' USING ERRCODE='42501'; END IF;
  persona_version:=v_persona.version;
 END IF;
 INSERT INTO vec_contexto_actor_v1.perfil_versiones(perfil_ref,version,persona_ref,procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
  VALUES(p_perfil,1,p_persona,p_procedencia,p_procedencia_version,p_procedencia_huella,p_procedencia_autoridad,'activo',p_desde,p_hasta);
 INSERT INTO vec_contexto_actor_v1.perfil_actual(perfil_ref,version) VALUES(p_perfil,1);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones(vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref,procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
  VALUES(p_vinculo,1,p_cuenta,p_perfil,p_persona,p_procedencia,p_procedencia_version,p_procedencia_huella,p_procedencia_autoridad,'activo',p_desde,p_hasta);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual(vinculo_ref,version) VALUES(p_vinculo,1);
 perfil_version:=1; vinculo_version:=1;
 RETURN NEXT;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.registrar_persona_registro_propio_v1(text,text,text,text,boolean,text,numeric,text,text,timestamptz,timestamptz) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_identidad_sesiones_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.registrar_persona_registro_propio_v1(text,text,text,text,boolean,text,numeric,text,text,timestamptz,timestamptz) TO vec_identidad_sesiones_v1_propietario;

-- El maestro aún no entrega equivalencia de cuenta entre alias rotados.
-- Bajo el mismo lock de persona que usa el alta, deniega crear otra cuenta
-- si la persona ya conserva un vínculo, incluso histórico.
CREATE FUNCTION vec_contexto_actor_v1.persona_con_cuenta_registro_propio_v1(p_persona text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $c$
BEGIN
 IF current_user<>'vec_contexto_actor_v1_propietario'
    OR NOT pg_has_role(session_user,'vec_identidad_sesiones_v1_provisionador','MEMBER')
    OR NOT vec_contexto_actor_v1.referencia_valida(p_persona,'per_')
 THEN RAISE EXCEPTION 'registro propio: equivalencia de cuenta denegada' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:registro-propio:persona:'||p_persona,0));
 RETURN EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_versiones v WHERE v.persona_ref=p_persona);
END $c$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.persona_con_cuenta_registro_propio_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.persona_con_cuenta_registro_propio_v1(text) TO vec_identidad_sesiones_v1_propietario;

-- Revalidación del vínculo completo tras los locks y en el replay. Compara
-- las versiones actuales exactas: una revocación o sustitución deniega.
CREATE FUNCTION vec_contexto_actor_v1.validar_registro_propio_v1(
 p_cuenta text,p_cuenta_version numeric,p_persona text,p_persona_version numeric,
 p_perfil text,p_perfil_version numeric,p_vinculo text,p_vinculo_version numeric)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $v$
DECLARE c record; p record; f record; v record; t timestamptz:=clock_timestamp();
BEGIN
 IF current_user<>'vec_contexto_actor_v1_propietario'
    OR NOT pg_has_role(session_user,'vec_identidad_sesiones_v1_provisionador','MEMBER')
 THEN RAISE EXCEPTION 'registro propio: contexto denegado' USING ERRCODE='42501'; END IF;
 SELECT a.version,h.estado,h.vigente_desde,h.vigente_hasta INTO c
 FROM vec_contexto_actor_v1.proyeccion_cuenta_actual a
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones h USING(cuenta_ref,version)
 WHERE a.cuenta_ref=p_cuenta FOR SHARE OF a;
 IF NOT FOUND OR c.version IS DISTINCT FROM p_cuenta_version OR c.estado<>'activo' OR t<c.vigente_desde OR t>=c.vigente_hasta
 THEN RAISE EXCEPTION 'registro propio: cuenta revocada' USING ERRCODE='42501'; END IF;
 SELECT a.version,h.estado,h.vigente_desde,h.vigente_hasta INTO p
 FROM vec_contexto_actor_v1.persona_actual a JOIN vec_contexto_actor_v1.persona_versiones h USING(persona_ref,version)
 WHERE a.persona_ref=p_persona FOR SHARE OF a;
 IF NOT FOUND OR p.version IS DISTINCT FROM p_persona_version OR p.estado<>'activo' OR t<p.vigente_desde OR t>=p.vigente_hasta
 THEN RAISE EXCEPTION 'registro propio: persona revocada' USING ERRCODE='42501'; END IF;
 SELECT a.version,h.persona_ref,h.estado,h.vigente_desde,h.vigente_hasta INTO f
 FROM vec_contexto_actor_v1.perfil_actual a JOIN vec_contexto_actor_v1.perfil_versiones h USING(perfil_ref,version)
 WHERE a.perfil_ref=p_perfil FOR SHARE OF a;
 IF NOT FOUND OR f.version IS DISTINCT FROM p_perfil_version OR f.persona_ref IS DISTINCT FROM p_persona OR f.estado<>'activo' OR t<f.vigente_desde OR t>=f.vigente_hasta
 THEN RAISE EXCEPTION 'registro propio: perfil revocado' USING ERRCODE='42501'; END IF;
 SELECT a.version,h.cuenta_ref,h.perfil_ref,h.persona_ref,h.estado,h.vigente_desde,h.vigente_hasta INTO v
 FROM vec_contexto_actor_v1.vinculo_contexto_actual a JOIN vec_contexto_actor_v1.vinculo_contexto_versiones h USING(vinculo_ref,version)
 WHERE a.vinculo_ref=p_vinculo FOR SHARE OF a;
 IF NOT FOUND OR v.version IS DISTINCT FROM p_vinculo_version OR v.cuenta_ref IS DISTINCT FROM p_cuenta
    OR v.perfil_ref IS DISTINCT FROM p_perfil OR v.persona_ref IS DISTINCT FROM p_persona
    OR v.estado<>'activo' OR t<v.vigente_desde OR t>=v.vigente_hasta
 THEN RAISE EXCEPTION 'registro propio: vínculo revocado' USING ERRCODE='42501'; END IF;
END $v$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.validar_registro_propio_v1(text,numeric,text,numeric,text,numeric,text,numeric) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.validar_registro_propio_v1(text,numeric,text,numeric,text,numeric,text,numeric) TO vec_identidad_sesiones_v1_propietario;
COMMIT;
