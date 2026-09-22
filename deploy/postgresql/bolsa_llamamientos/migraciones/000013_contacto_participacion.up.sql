\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000013',0));
DO $b$ DECLARE accion text; ruta text; BEGIN
 SELECT pg_get_constraintdef(oid,true) INTO STRICT accion FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass AND conname='bitacora_intento_borrador_llamamiento_accion_check';
 SELECT pg_get_constraintdef(oid,true) INTO STRICT ruta FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass AND conname='bitacora_intento_borrador_llamamiento_ruta_clase_check';
 IF strpos(accion,'''cambiar_situacion''::text')=0 OR strpos(accion,'''registrar_contacto''::text')<>0 OR strpos(ruta,'''situacion''::text')=0 OR strpos(ruta,'''contactos''::text')<>0 THEN RAISE EXCEPTION 'bitácora incompatible con B3' USING ERRCODE='55000'; END IF;
END $b$;
ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento DROP CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check;
ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento ADD CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check CHECK(accion IN('crear','consultar','cambiar_situacion','registrar_contacto'));
ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento DROP CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check;
ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento ADD CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check CHECK(ruta_clase IN('coleccion','detalle','situacion','contactos'));
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(p_correlacion_ref text,p_accion text,p_ruta_clase text,p_actor_ref text,p_resultado text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_ref text; BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento();
 IF p_correlacion_ref IS NULL OR p_correlacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$' OR p_accion NOT IN('crear','consultar','cambiar_situacion','registrar_contacto') OR p_ruta_clase NOT IN('coleccion','detalle','situacion','contactos') OR (p_actor_ref IS NOT NULL AND p_actor_ref !~ '^per_[A-Za-z0-9_-]{22,128}$') OR p_resultado NOT IN('autenticacion_requerida','acceso_denegado','recurso_no_disponible','infraestructura_no_disponible','resultado_indeterminado') THEN RAISE EXCEPTION 'intento de frontera Bolsa inválido' USING ERRCODE='22023'; END IF;
 v_ref:='intento:'||translate(encode(sha256(convert_to(p_correlacion_ref||':'||p_accion||':'||p_ruta_clase||':'||coalesce(p_actor_ref,'')||':'||p_resultado||':'||clock_timestamp()::text,'UTF8')),'hex'),'0123456789','ghijklmnop');
 INSERT INTO vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento(intento_ref,correlacion_ref,accion,ruta_clase,actor_ref,resultado,registrada_en) VALUES(v_ref,p_correlacion_ref,p_accion,p_ruta_clase,p_actor_ref,p_resultado,clock_timestamp());
END $f$;
DO $p$ BEGIN
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_contacto_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL OR to_regclass('vec_bolsa_llamamientos.situacion_participacion') IS NULL OR to_regclass('vec_bolsa_llamamientos.contacto_participacion') IS NOT NULL THEN RAISE EXCEPTION 'precondición B3 incompatible' USING ERRCODE='55000'; END IF;
END $p$;
CREATE TABLE vec_bolsa_llamamientos.contacto_participacion(
 contacto_ref text PRIMARY KEY,
 bolsa_ref text NOT NULL,
 participacion_ref text NOT NULL,
 llamamiento_ref text,
 canal text NOT NULL CHECK(canal IN('telefono','correo','sms','presencial','otro')),
 instante timestamptz(6) NOT NULL,
 actor text NOT NULL CHECK(actor ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 resultado text NOT NULL CHECK(resultado IN('contactado','no_contesta','buzon','acepta','rechaza','aplazado','otro')),
 anotacion text NOT NULL CHECK(anotacion=btrim(anotacion) AND octet_length(anotacion) BETWEEN 1 AND 1000 AND anotacion !~* '(^|[^[:alpha:]])(dni|nie|nif|pasaporte|email|teléfono)([^[:alpha:]]|$)' AND strpos(anotacion,'@')=0),
 clave_idempotencia text NOT NULL,
 recibo_ref text NOT NULL UNIQUE,
 registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(participacion_ref,clave_idempotencia)
);
CREATE INDEX contacto_participacion_participacion_fecha ON vec_bolsa_llamamientos.contacto_participacion(participacion_ref,instante DESC,contacto_ref DESC);
CREATE INDEX contacto_participacion_bolsa_fecha ON vec_bolsa_llamamientos.contacto_participacion(bolsa_ref,instante DESC,contacto_ref DESC);
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion FORCE ROW LEVEL SECURITY;
CREATE POLICY contacto_participacion_solo_propietario ON vec_bolsa_llamamientos.contacto_participacion TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.contacto_participacion FROM PUBLIC;
CREATE TRIGGER contacto_participacion_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.contacto_participacion FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

CREATE FUNCTION vec_bolsa_llamamientos.registrar_contacto_participacion_v1(p_contacto_ref text,p_bolsa_ref text,p_participacion_ref text,p_llamamiento_ref text,p_canal text,p_instante timestamptz,p_actor text,p_resultado text,p_anotacion text,p_clave text,p_recibo text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(reutilizado boolean,recibo_ref text,contacto_ref text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE consumo record; d jsonb; anterior vec_bolsa_llamamientos.contacto_participacion%ROWTYPE;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR p_contacto_ref IS NULL OR p_bolsa_ref IS NULL OR p_participacion_ref IS NULL OR p_canal NOT IN('telefono','correo','sms','presencial','otro') OR p_instante IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$' OR p_resultado NOT IN('contactado','no_contesta','buzon','acepta','rechaza','aplazado','otro') OR p_anotacion IS NULL OR p_anotacion<>btrim(p_anotacion) OR octet_length(p_anotacion) NOT BETWEEN 1 AND 1000 OR p_clave IS NULL OR p_clave<>btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 1 AND 256 OR p_recibo IS NULL THEN RAISE EXCEPTION 'contacto inválido' USING ERRCODE='22023'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea) WHERE e.participacion_ref=p_participacion_ref AND c.bolsa_ref=p_bolsa_ref) THEN RAISE EXCEPTION 'participación ajena' USING ERRCODE='23503'; END IF;
 IF p_llamamiento_ref IS NOT NULL AND NOT EXISTS(
  SELECT 1 FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo l
  JOIN vec_bolsa_llamamientos.integracion_desarrollo o USING(operacion_ref)
  WHERE l.llamamiento_ref=p_llamamiento_ref AND l.bolsa_ref=p_bolsa_ref
   AND convert_from(o.registro_canonico,'UTF8')::jsonb#>>'{propuesta,participacion_seleccionada_ref}'=p_participacion_ref
 ) THEN RAISE EXCEPTION 'llamamiento ajeno' USING ERRCODE='23503'; END IF;
 SELECT * INTO consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 d:=convert_from(p_decision,'UTF8')::jsonb;
 IF consumo.efecto_ref IS DISTINCT FROM p_participacion_ref OR consumo.consumo_nuevo IS NOT TRUE OR d->>'principal_id' IS DISTINCT FROM p_actor OR d->>'accion' IS DISTINCT FROM 'bolsa.contacto_participacion.registrar' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa' OR d->>'finalidad' IS DISTINCT FROM 'gestion_contactos_participacion' OR d->>'recurso_ref' IS DISTINCT FROM p_participacion_ref OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN RAISE EXCEPTION 'contacto no autorizado' USING ERRCODE='42501'; END IF;
 SELECT * INTO anterior FROM vec_bolsa_llamamientos.contacto_participacion c WHERE c.participacion_ref=p_participacion_ref AND c.clave_idempotencia=p_clave FOR SHARE;
 IF FOUND THEN
  IF anterior.bolsa_ref<>p_bolsa_ref OR anterior.llamamiento_ref IS DISTINCT FROM p_llamamiento_ref OR anterior.canal<>p_canal OR anterior.instante<>p_instante OR anterior.actor<>p_actor OR anterior.resultado<>p_resultado OR anterior.anotacion<>p_anotacion THEN RAISE EXCEPTION 'clave idempotente divergente' USING ERRCODE='VBC01'; END IF;
  RETURN QUERY SELECT true,anterior.recibo_ref,anterior.contacto_ref; RETURN;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.contacto_participacion(contacto_ref,bolsa_ref,participacion_ref,llamamiento_ref,canal,instante,actor,resultado,anotacion,clave_idempotencia,recibo_ref) VALUES(p_contacto_ref,p_bolsa_ref,p_participacion_ref,p_llamamiento_ref,p_canal,p_instante,p_actor,p_resultado,p_anotacion,p_clave,p_recibo);
 RETURN QUERY SELECT false,p_recibo,p_contacto_ref;
END $f$;
CREATE FUNCTION vec_bolsa_llamamientos.listar_contactos_participacion_v1(p_bolsa_ref text,p_participacion_ref text,p_cursor text,p_limite integer,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(contacto_ref text,bolsa_ref text,participacion_ref text,llamamiento_ref text,canal text,instante timestamptz,actor text,resultado text,anotacion text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE consumo record; d jsonb; BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR p_bolsa_ref IS NULL OR p_limite NOT BETWEEN 1 AND 100 THEN RAISE EXCEPTION 'consulta contacto inválida' USING ERRCODE='22023'; END IF;
 SELECT * INTO consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_contacto_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 d:=convert_from(p_decision,'UTF8')::jsonb;
 IF consumo.efecto_ref IS DISTINCT FROM coalesce(p_participacion_ref,p_bolsa_ref) OR consumo.consumo_nuevo IS NOT TRUE OR d->>'accion' IS DISTINCT FROM 'bolsa.contacto_participacion.consultar' OR d->>'finalidad' IS DISTINCT FROM 'consulta_contactos_participacion' OR d->>'recurso_ref' IS DISTINCT FROM coalesce(p_participacion_ref,p_bolsa_ref) OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN RAISE EXCEPTION 'consulta contacto no autorizada' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT c.contacto_ref,c.bolsa_ref,c.participacion_ref,c.llamamiento_ref,c.canal,c.instante,c.actor,c.resultado,c.anotacion FROM vec_bolsa_llamamientos.contacto_participacion c WHERE c.bolsa_ref=p_bolsa_ref AND (p_participacion_ref IS NULL OR c.participacion_ref=p_participacion_ref) AND (p_cursor IS NULL OR (c.instante,c.contacto_ref)<(SELECT x.instante,x.contacto_ref FROM vec_bolsa_llamamientos.contacto_participacion x WHERE x.contacto_ref=p_cursor)) ORDER BY c.instante DESC,c.contacto_ref DESC LIMIT p_limite;
END
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_contacto_participacion_v1(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_contactos_participacion_v1(text,text,text,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_contacto_participacion_v1(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_contactos_participacion_v1(text,text,text,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
