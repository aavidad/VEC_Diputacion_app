\set ON_ERROR_STOP on
-- B111: metadato técnico de persona sólo en el sobre SQL del detalle RRHH.
-- Conserva la proyección pública, listas y lector exterior de B96/B99.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp; SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000111',0));
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regprocedure('vec_bolsa_llamamientos.consultar_inscripcion_v1(text,jsonb,bytea,bytea,jsonb)') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.cotejar_objetivo_revision_meritos_v1(text,text,text,text,bytea,bytea,jsonb)') IS NOT NULL
 OR to_regrole('vec_meritos_propietario') IS NULL
 THEN RAISE EXCEPTION 'B111: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $detalle$
DECLARE f oid:=to_regprocedure('vec_bolsa_llamamientos.consultar_inscripcion_v1(text,jsonb,bytea,bytea,jsonb)');
 original text;nuevo text;actual text;meta jsonb;
 declaracion text:=$d$ v_conjunto jsonb; v_ambito_solicitud jsonb; v_unidad_ref text; v_ambito_ref text;$d$;
 declaracion_nueva text:=$d$ v_conjunto jsonb; v_ambito_solicitud jsonb; v_unidad_ref text; v_ambito_ref text;
 v_persona_objetivo text;$d$;
 seleccion text:=$s$'fuente_sha256',s.ambito_fuente_sha256) INTO STRICT v_ambito_solicitud$s$;
 seleccion_nueva text:=$s$'fuente_sha256',s.ambito_fuente_sha256),s.persona_ref
    INTO STRICT v_ambito_solicitud,v_persona_objetivo$s$;
 retorno text:=$r$ RETURN jsonb_build_object('resultado',v_resultado,'proyeccion',v_proyeccion,
  'auditoria_ref',v_auditoria.auditoria_ref,
  'correlacion_ref',v_auditoria.correlacion_ref,'consultada_en',v_auditoria.registrada_en);$r$;
 retorno_nuevo text:=$r$ IF p_accion='bolsa.inscripcion.rrhh.consultar' AND v_resultado='obtenida'
  AND (v_persona_objetivo IS NULL OR v_persona_objetivo !~ '^per_[A-Za-z0-9_-]{22,128}$')
 THEN RAISE EXCEPTION 'B111: persona objetivo incompatible' USING ERRCODE='55000'; END IF;
 RETURN jsonb_build_object('resultado',v_resultado,'proyeccion',v_proyeccion,
  'auditoria_ref',v_auditoria.auditoria_ref,
  'correlacion_ref',v_auditoria.correlacion_ref,'consultada_en',v_auditoria.registrada_en)
  || CASE WHEN p_accion='bolsa.inscripcion.rrhh.consultar' AND v_resultado='obtenida'
     THEN jsonb_build_object('persona_ref_interna',v_persona_objetivo)
     ELSE '{}'::jsonb END;$r$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO original,meta FROM pg_proc p WHERE p.oid=f;
 IF original IS NULL OR encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM
  'f30b4cba82e2e09ace0d6bb0cd4c4ec1bda06c82ce30b93a45910623e5c8b64b'
 OR (length(original)-length(replace(original,declaracion,'')))/length(declaracion)<>1
 OR (length(original)-length(replace(original,seleccion,'')))/length(seleccion)<>1
 OR (length(original)-length(replace(original,retorno,'')))/length(retorno)<>1
 OR strpos(original,'persona_ref_interna')<>0
 THEN RAISE EXCEPTION 'B111: lector instalado divergente' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,declaracion,declaracion_nueva);
 nuevo:=replace(nuevo,seleccion,seleccion_nueva);
 nuevo:=replace(nuevo,retorno,retorno_nuevo);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
 OR replace(replace(replace(actual,retorno_nuevo,retorno),seleccion_nueva,seleccion),declaracion_nueva,declaracion)
  IS DISTINCT FROM original
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 THEN RAISE EXCEPTION 'B111: lector alterado fuera del sobre técnico' USING ERRCODE='55000'; END IF;
END $detalle$;

CREATE FUNCTION vec_bolsa_llamamientos.cotejar_objetivo_revision_meritos_v1(
 p_solicitud_ref text,p_persona_ref text,p_actor_ref text,p_perfil_ref text,
 p_contexto bytea,p_vinculo bytea,p_captura jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE c jsonb; v jsonb; detalle jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR session_user<>'vec_bolsa_inscripciones_rrhh_lector'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR p_solicitud_ref !~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 OR p_persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_actor_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_perfil_ref !~ '^prf_[A-Za-z0-9_-]{22,128}$'
 OR jsonb_typeof(p_captura) IS DISTINCT FROM 'object'
 OR p_captura->>'accion' IS DISTINCT FROM 'bolsa.inscripcion.rrhh.consultar'
 OR p_captura->>'recurso_ref' IS DISTINCT FROM p_solicitud_ref
 OR p_captura->>'persona_ref' IS DISTINCT FROM p_actor_ref
 THEN RETURN false; END IF;
 BEGIN c:=convert_from(p_contexto,'UTF8')::jsonb;v:=convert_from(p_vinculo,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN RETURN false; END;
 IF c->>'principal_ref' IS DISTINCT FROM p_actor_ref
 OR c->>'perfil_activo_ref' IS DISTINCT FROM p_perfil_ref
 OR v->>'principal_id' IS DISTINCT FROM p_actor_ref
 OR v->>'perfil_activo_ref' IS DISTINCT FROM p_perfil_ref
 THEN RETURN false; END IF;
 detalle:=vec_bolsa_llamamientos.consultar_inscripcion_v1(
  'bolsa.inscripcion.rrhh.consultar',jsonb_build_object('solicitud_ref',p_solicitud_ref),
  p_contexto,p_vinculo,p_captura);
 RETURN detalle->>'resultado' IS NOT DISTINCT FROM 'obtenida'
  AND detalle->>'persona_ref_interna' IS NOT DISTINCT FROM p_persona_ref
  AND coalesce(detalle->>'auditoria_ref','')<>'';
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.cotejar_objetivo_revision_meritos_v1(
 text,text,text,text,bytea,bytea,jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_meritos_propietario;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.cotejar_objetivo_revision_meritos_v1(
 text,text,text,text,bytea,bytea,jsonb) TO vec_meritos_propietario;
COMMIT;
