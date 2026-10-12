\set ON_ERROR_STOP on
-- B102: la anotación puede mencionar el canal teléfono sin almacenar cifras.
-- Preimagen literal de HZ20+B87; B72/B87 e historia conservadas.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000102',0));
DO $pre$
DECLARE v_funcion oid; v_tabla oid;
BEGIN
 v_tabla:=pg_catalog.to_regclass('vec_bolsa_llamamientos.contacto_participacion');
 SELECT p.oid INTO v_funcion FROM pg_catalog.pg_proc p
 JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_bolsa_llamamientos' AND p.proname='registrar_contacto_participacion_v3' AND p.pronargs=24;
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR v_tabla IS NULL OR v_funcion IS NULL
    OR (SELECT pg_catalog.md5(pg_catalog.pg_get_functiondef(v_funcion)))<>'9df6e0d7ff6dbabeb589a9364378f3c8'
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=v_funcion
      AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef
      AND p.provolatile='v')
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_constraint c WHERE c.conrelid=v_tabla
      AND c.conname='contacto_participacion_anotacion_check' AND c.contype='c'
      AND pg_catalog.md5(pg_catalog.pg_get_constraintdef(c.oid,true))='05fd5342b4d77b134dc803ffccb109a3')
 THEN RAISE EXCEPTION 'B102: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
LOCK TABLE vec_bolsa_llamamientos.contacto_participacion IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion
 DROP CONSTRAINT contacto_participacion_anotacion_check;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion
 ADD CONSTRAINT contacto_participacion_anotacion_check CHECK(
  anotacion=pg_catalog.btrim(anotacion)
  AND pg_catalog.octet_length(anotacion)<=1000
  AND (pg_catalog.octet_length(anotacion)>=1 OR
       (instante_servidor AND canal='telefono' AND llamamiento_ref IS NOT NULL AND oferta_ref IS NULL))
  AND anotacion !~* '(^|[^[:alpha:]])(dni|nie|nif|pasaporte|email)([^[:alpha:]]|$)'
  AND anotacion !~* '(^|[^[:alpha:]])tel[eé]fono[^[:alnum:]]{0,8}[+]?[0-9]'
  AND anotacion !~ '(^|[^[:alnum:]])[+]?[0-9]([[:space:].-]*[0-9]){8,14}($|[^0-9])'
  AND pg_catalog.strpos(anotacion,'@')=0);

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.registrar_contacto_participacion_v3(p_contacto_ref text, p_bolsa_ref text, p_participacion_ref text, p_llamamiento_ref text, p_canal text, p_instante timestamp with time zone, p_actor text, p_resultado text, p_anotacion text, p_clave text, p_recibo text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea, p_oferta_ref text, p_evidencia_ref text, p_evidencia_huella text)
 RETURNS TABLE(reutilizado boolean, recibo_ref text, contacto_ref text)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE consumo record; d jsonb; anterior vec_bolsa_llamamientos.contacto_participacion%ROWTYPE;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_contacto_ref IS NULL OR p_contacto_ref<>btrim(p_contacto_ref) OR octet_length(p_contacto_ref) NOT BETWEEN 1 AND 256
    OR p_bolsa_ref IS NULL OR p_bolsa_ref<>btrim(p_bolsa_ref) OR octet_length(p_bolsa_ref) NOT BETWEEN 1 AND 256
    OR p_participacion_ref IS NULL OR p_participacion_ref<>btrim(p_participacion_ref) OR octet_length(p_participacion_ref) NOT BETWEEN 1 AND 256
    OR p_canal IS NULL OR p_canal NOT IN('telefono','correo','sms','presencial','otro')
    OR p_instante IS NULL OR NOT isfinite(p_instante)
    OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_resultado IS NULL OR p_resultado NOT IN('contactado','no_contesta','buzon','acepta','rechaza','aplazado','otro','enviado','no_enviado','numero_erroneo','no_entregado','entrega_declarada')
    OR p_anotacion IS NULL OR p_anotacion<>btrim(p_anotacion) OR octet_length(p_anotacion) NOT BETWEEN 1 AND 1000
    OR (p_anotacion ~* '(^|[^[:alpha:]])(dni|nie|nif|pasaporte|email)([^[:alpha:]]|$)'
       OR p_anotacion ~* '(^|[^[:alpha:]])tel[eé]fono[^[:alnum:]]{0,8}[+]?[0-9]'
       OR p_anotacion ~ '(^|[^[:alnum:]])[+]?[0-9]([[:space:].-]*[0-9]){8,14}($|[^0-9])') OR strpos(p_anotacion,'@')<>0
    OR p_clave IS NULL OR p_clave<>btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 1 AND 256
    OR p_recibo IS NULL OR p_recibo<>btrim(p_recibo) OR octet_length(p_recibo) NOT BETWEEN 1 AND 256
    OR (p_llamamiento_ref IS NOT NULL AND (p_llamamiento_ref<>btrim(p_llamamiento_ref) OR octet_length(p_llamamiento_ref) NOT BETWEEN 1 AND 256))
    OR (p_oferta_ref IS NOT NULL AND (p_oferta_ref !~ '^oferta:[0-9a-f]{64}$' OR p_llamamiento_ref IS NOT NULL
       OR p_canal<>'correo' OR p_resultado NOT IN('enviado','no_enviado','no_entregado','entrega_declarada')))
    OR (p_resultado='entrega_declarada' AND (p_oferta_ref IS NULL OR p_evidencia_ref IS NULL OR p_evidencia_huella IS NULL
       OR octet_length(p_evidencia_ref) NOT BETWEEN 1 AND 256 OR p_evidencia_ref !~ '^[A-Za-z][A-Za-z0-9:_-]*$'
       OR strpos(p_evidencia_ref,'@')<>0
       OR p_evidencia_ref ~* '(^|[:_-])(dni|nie|nif|pasaporte|passport)([:_-]|$)'
       OR p_evidencia_ref ~* '(([0-9][:_-]?){8}|[XYZ][:_-]?([0-9][:_-]?){7})[A-Z]'
       OR p_evidencia_huella !~ '^[0-9a-f]{64}$'))
    OR (p_resultado<>'entrega_declarada' AND (p_evidencia_ref IS NOT NULL OR p_evidencia_huella IS NOT NULL)) THEN
  RAISE EXCEPTION 'B72: contacto inválido' USING ERRCODE='22023';
 END IF;
 -- Autorizar también el replay: la clave o el recibo históricos no dan acceso.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B72: contacto no autorizado' USING ERRCODE='42501'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_participacion_ref OR consumo.consumo_nuevo IS NOT TRUE
    OR consumo.decision_ref IS NULL OR consumo.auditoria_ref IS NULL
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'bolsa.contacto_participacion.registrar'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_contactos_participacion'
    OR d->>'recurso_ref' IS DISTINCT FROM p_participacion_ref
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'B72: contacto no autorizado' USING ERRCODE='42501';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e
               JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea)
               WHERE e.participacion_ref=p_participacion_ref AND c.bolsa_ref=p_bolsa_ref) THEN
  RAISE EXCEPTION 'B72: participación ajena a la bolsa' USING ERRCODE='23503';
 END IF;
 IF p_oferta_ref IS NOT NULL AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.oferta_publicada o
                                           WHERE o.oferta_ref=p_oferta_ref AND o.bolsa_ref=p_bolsa_ref) THEN
  RAISE EXCEPTION 'B72: oferta ajena a la bolsa' USING ERRCODE='23503';
 END IF;
 IF p_llamamiento_ref IS NOT NULL AND NOT EXISTS(
  SELECT 1 FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo l
  JOIN vec_bolsa_llamamientos.integracion_desarrollo o USING(operacion_ref)
  WHERE l.llamamiento_ref=p_llamamiento_ref AND l.bolsa_ref=p_bolsa_ref
   AND convert_from(o.registro_canonico,'UTF8')::jsonb#>>'{propuesta,participacion_seleccionada_ref}'=p_participacion_ref) THEN
  RAISE EXCEPTION 'B72: llamamiento ajeno' USING ERRCODE='23503';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended(
  'vec_bolsa_llamamientos:contacto:'||p_participacion_ref||chr(31)||p_clave,0));
 SELECT * INTO anterior FROM vec_bolsa_llamamientos.contacto_participacion c
  WHERE c.participacion_ref=p_participacion_ref AND c.clave_idempotencia=p_clave FOR SHARE;
 IF FOUND THEN
  IF anterior.contacto_ref IS DISTINCT FROM p_contacto_ref OR anterior.recibo_ref IS DISTINCT FROM p_recibo
     OR anterior.bolsa_ref IS DISTINCT FROM p_bolsa_ref OR anterior.llamamiento_ref IS DISTINCT FROM p_llamamiento_ref
     OR anterior.canal IS DISTINCT FROM p_canal OR anterior.instante IS DISTINCT FROM p_instante
     OR anterior.actor IS DISTINCT FROM p_actor OR anterior.resultado IS DISTINCT FROM p_resultado
     OR anterior.anotacion IS DISTINCT FROM p_anotacion OR anterior.oferta_ref IS DISTINCT FROM p_oferta_ref
     OR anterior.evidencia_ref IS DISTINCT FROM p_evidencia_ref OR anterior.evidencia_huella IS DISTINCT FROM p_evidencia_huella THEN
   RAISE EXCEPTION 'B72: clave idempotente divergente' USING ERRCODE='VBC01';
  END IF;
  RETURN QUERY SELECT true,anterior.recibo_ref,anterior.contacto_ref; RETURN;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.contacto_participacion(
  contacto_ref,bolsa_ref,participacion_ref,llamamiento_ref,canal,instante,actor,resultado,anotacion,clave_idempotencia,recibo_ref,
  oferta_ref,evidencia_ref,evidencia_huella)
 VALUES(p_contacto_ref,p_bolsa_ref,p_participacion_ref,p_llamamiento_ref,p_canal,p_instante,p_actor,p_resultado,p_anotacion,p_clave,p_recibo,
  p_oferta_ref,p_evidencia_ref,p_evidencia_huella);
 RETURN QUERY SELECT false,p_recibo,p_contacto_ref;
END $function$;

COMMIT;
