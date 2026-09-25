\set ON_ERROR_STOP on
-- Bolsa 000031. Intentos telefónicos del llamamiento (Reglamento de bolsas,
-- art. 8.2.a) y rebote del correo de aviso anotado por RRHH.
-- Añade los resultados «número erróneo» y «correo no entregado» al histórico
-- de contactos B3 y una función v2 que, además de lo que hace la v1, puede
-- controlar los intentos de un llamamiento con los parámetros que le pasa la
-- aplicación desde el catálogo de reglas. La v1 no cambia.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000031',0));
DO $precondicion$ DECLARE r text; BEGIN
 IF to_regclass('vec_bolsa_llamamientos.contacto_participacion') IS NOT NULL THEN
  SELECT pg_get_constraintdef(oid,true) INTO r FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.contacto_participacion'::regclass AND conname='contacto_participacion_resultado_check' AND contype='c';
 END IF;
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR r IS NULL OR strpos(r,'''no_enviado''')=0 OR strpos(r,'''numero_erroneo''')<>0 OR strpos(r,'''no_entregado''')<>0
    OR to_regprocedure('vec_bolsa_llamamientos.registrar_contacto_participacion_v1(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.registrar_contacto_participacion_v2(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,integer,integer,boolean,text[])') IS NOT NULL THEN
  RAISE EXCEPTION 'precondición Bolsa 000031 incompatible' USING ERRCODE='55000';
 END IF;
END $precondicion$;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion DROP CONSTRAINT contacto_participacion_resultado_check;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion ADD CONSTRAINT contacto_participacion_resultado_check CHECK(resultado IN('contactado','no_contesta','buzon','acepta','rechaza','aplazado','otro','enviado','no_enviado','numero_erroneo','no_entregado'));
CREATE INDEX contacto_participacion_intentos_llamamiento ON vec_bolsa_llamamientos.contacto_participacion(participacion_ref,llamamiento_ref,instante) WHERE canal='telefono' AND llamamiento_ref IS NOT NULL;

-- p_maximo_intentos NULL: registro sin control, como la v1 pero con los
-- resultados nuevos. Con control, solo teléfono y ligado a un llamamiento:
-- bajo un cerrojo por participación y llamamiento se cuentan los intentos
-- previos sin contacto; agotados (VBC03) o antes de la separación mínima si el
-- catálogo manda impedir (VBC02) no se registra nada y la autorización
-- consumida se deshace con la transacción. Devuelve el resumen previo para que
-- la aplicación calcule avisos y la propuesta de baja.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_contacto_participacion_v2(p_contacto_ref text,p_bolsa_ref text,p_participacion_ref text,p_llamamiento_ref text,p_canal text,p_instante timestamptz,p_actor text,p_resultado text,p_anotacion text,p_clave text,p_recibo text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_maximo_intentos integer,p_separacion_segundos integer,p_impedir_separacion boolean,p_resultados_sin_contacto text[])
RETURNS TABLE(reutilizado boolean,recibo_ref text,contacto_ref text,previos_sin_contacto integer,previo_contactado boolean,previo_ultimo timestamptz) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE consumo record; d jsonb; anterior vec_bolsa_llamamientos.contacto_participacion%ROWTYPE; v_control boolean; v_sin integer:=0; v_contactado boolean:=false; v_ultimo timestamptz;
 v_resultados constant text[]:=ARRAY['contactado','no_contesta','buzon','acepta','rechaza','aplazado','otro','numero_erroneo','no_entregado'];
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR p_contacto_ref IS NULL OR p_bolsa_ref IS NULL OR p_participacion_ref IS NULL OR p_canal IS NULL OR p_canal NOT IN('telefono','correo','sms','presencial','otro') OR p_instante IS NULL OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$' OR p_resultado IS NULL OR NOT p_resultado=ANY(v_resultados) OR p_anotacion IS NULL OR p_anotacion<>btrim(p_anotacion) OR octet_length(p_anotacion) NOT BETWEEN 1 AND 1000 OR p_clave IS NULL OR p_clave<>btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 1 AND 256 OR p_recibo IS NULL THEN RAISE EXCEPTION 'contacto inválido' USING ERRCODE='22023'; END IF;
 v_control:=p_maximo_intentos IS NOT NULL;
 IF (v_control AND (p_canal<>'telefono' OR p_llamamiento_ref IS NULL OR p_maximo_intentos NOT BETWEEN 1 AND 100 OR p_separacion_segundos IS NULL OR p_separacion_segundos NOT BETWEEN 0 AND 604800 OR p_impedir_separacion IS NULL OR p_resultados_sin_contacto IS NULL OR cardinality(p_resultados_sin_contacto) NOT BETWEEN 1 AND 16 OR array_position(p_resultados_sin_contacto,NULL) IS NOT NULL OR NOT p_resultados_sin_contacto<@v_resultados))
    OR (NOT v_control AND (p_separacion_segundos IS NOT NULL OR p_impedir_separacion IS NOT NULL OR p_resultados_sin_contacto IS NOT NULL)) THEN
  RAISE EXCEPTION 'control de intentos inválido' USING ERRCODE='22023';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea) WHERE e.participacion_ref=p_participacion_ref AND c.bolsa_ref=p_bolsa_ref) THEN RAISE EXCEPTION 'participación ajena' USING ERRCODE='23503'; END IF;
 IF p_llamamiento_ref IS NOT NULL AND NOT EXISTS(
  SELECT 1 FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo l
  JOIN vec_bolsa_llamamientos.integracion_desarrollo o USING(operacion_ref)
  WHERE l.llamamiento_ref=p_llamamiento_ref AND l.bolsa_ref=p_bolsa_ref
   AND convert_from(o.registro_canonico,'UTF8')::jsonb#>>'{propuesta,participacion_seleccionada_ref}'=p_participacion_ref
 ) THEN RAISE EXCEPTION 'llamamiento ajeno' USING ERRCODE='23503'; END IF;
 IF v_control THEN
  PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:intentos:'||p_participacion_ref||chr(31)||p_llamamiento_ref,0));
  SELECT count(*) FILTER (WHERE c.resultado=ANY(p_resultados_sin_contacto))::integer,coalesce(bool_or(NOT c.resultado=ANY(p_resultados_sin_contacto)),false),max(c.instante)
    INTO v_sin,v_contactado,v_ultimo
    FROM vec_bolsa_llamamientos.contacto_participacion c
   WHERE c.participacion_ref=p_participacion_ref AND c.llamamiento_ref=p_llamamiento_ref AND c.canal='telefono' AND c.contacto_ref<>p_contacto_ref;
 END IF;
 SELECT * INTO consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 d:=convert_from(p_decision,'UTF8')::jsonb;
 IF consumo.efecto_ref IS DISTINCT FROM p_participacion_ref OR consumo.consumo_nuevo IS NOT TRUE OR d->>'principal_id' IS DISTINCT FROM p_actor OR d->>'accion' IS DISTINCT FROM 'bolsa.contacto_participacion.registrar' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa' OR d->>'finalidad' IS DISTINCT FROM 'gestion_contactos_participacion' OR d->>'recurso_ref' IS DISTINCT FROM p_participacion_ref OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN RAISE EXCEPTION 'contacto no autorizado' USING ERRCODE='42501'; END IF;
 SELECT * INTO anterior FROM vec_bolsa_llamamientos.contacto_participacion c WHERE c.participacion_ref=p_participacion_ref AND c.clave_idempotencia=p_clave FOR SHARE;
 IF FOUND THEN
  IF anterior.bolsa_ref<>p_bolsa_ref OR anterior.llamamiento_ref IS DISTINCT FROM p_llamamiento_ref OR anterior.canal<>p_canal OR anterior.instante<>p_instante OR anterior.actor<>p_actor OR anterior.resultado<>p_resultado OR anterior.anotacion<>p_anotacion THEN RAISE EXCEPTION 'clave idempotente divergente' USING ERRCODE='VBC01'; END IF;
  RETURN QUERY SELECT true,anterior.recibo_ref,anterior.contacto_ref,v_sin,v_contactado,v_ultimo; RETURN;
 END IF;
 IF v_control AND NOT v_contactado THEN
  IF v_sin>=p_maximo_intentos THEN RAISE EXCEPTION 'intentos de contacto agotados' USING ERRCODE='VBC03'; END IF;
  IF p_impedir_separacion AND v_ultimo IS NOT NULL AND abs(extract(epoch FROM p_instante-v_ultimo))<p_separacion_segundos THEN RAISE EXCEPTION 'intento antes de la separación mínima' USING ERRCODE='VBC02'; END IF;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.contacto_participacion(contacto_ref,bolsa_ref,participacion_ref,llamamiento_ref,canal,instante,actor,resultado,anotacion,clave_idempotencia,recibo_ref) VALUES(p_contacto_ref,p_bolsa_ref,p_participacion_ref,p_llamamiento_ref,p_canal,p_instante,p_actor,p_resultado,p_anotacion,p_clave,p_recibo);
 RETURN QUERY SELECT false,p_recibo,p_contacto_ref,v_sin,v_contactado,v_ultimo;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_contacto_participacion_v2(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,integer,integer,boolean,text[]) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_contacto_participacion_v2(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,integer,integer,boolean,text[]) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
