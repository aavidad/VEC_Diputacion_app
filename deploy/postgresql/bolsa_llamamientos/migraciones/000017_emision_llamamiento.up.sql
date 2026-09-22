\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000017',0));
DO $pre$ BEGIN
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL OR to_regclass('vec_bolsa_llamamientos.contacto_participacion') IS NULL OR to_regclass('vec_bolsa_llamamientos.datos_contacto_participacion') IS NULL THEN RAISE EXCEPTION 'dependencias B7 ausentes' USING ERRCODE='55000'; END IF;
END $pre$;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion DROP CONSTRAINT contacto_participacion_resultado_check;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion ADD CONSTRAINT contacto_participacion_resultado_check CHECK(resultado IN('contactado','no_contesta','buzon','acepta','rechaza','aplazado','otro','enviado','no_enviado'));
CREATE TABLE vec_bolsa_llamamientos.llamamiento_emitido(
 llamamiento_ref text PRIMARY KEY,
 recibo_ref text NOT NULL UNIQUE,
 bolsa_ref text NOT NULL,
 actor_ref text NOT NULL,
 clave_idempotencia text NOT NULL,
 participaciones jsonb NOT NULL,
 configuracion jsonb NOT NULL,
 huella_comando_sha256 text NOT NULL,
 estado text NOT NULL CHECK(estado='emitido_pendiente_respuesta'),
 emitido_en timestamptz(6) NOT NULL,
 decision_ref text NOT NULL UNIQUE,
 UNIQUE(bolsa_ref,clave_idempotencia),
 CHECK(llamamiento_ref ~ '^llamamiento:[0-9a-f]{64}$' AND recibo_ref ~ '^recibo:llamamiento:[0-9a-f]{64}$'),
 CHECK(actor_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 CHECK(jsonb_typeof(participaciones)='array' AND jsonb_array_length(participaciones) BETWEEN 1 AND 100),
 CHECK(jsonb_typeof(configuracion)='object'),
 CHECK(huella_comando_sha256 ~ '^[0-9a-f]{64}$')
);
ALTER TABLE vec_bolsa_llamamientos.llamamiento_emitido ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.llamamiento_emitido FORCE ROW LEVEL SECURITY;
CREATE POLICY llamamiento_emitido_solo_propietario ON vec_bolsa_llamamientos.llamamiento_emitido TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.llamamiento_emitido FROM PUBLIC;
CREATE TRIGGER llamamiento_emitido_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.llamamiento_emitido FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE FUNCTION vec_bolsa_llamamientos.emitir_llamamiento_v1(p_llamamiento text,p_recibo text,p_bolsa text,p_actor text,p_clave text,p_participaciones jsonb,p_configuracion jsonb,p_contactos jsonb,p_emitido timestamptz,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(emision jsonb,reutilizada boolean) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE previo record; consumo record; d jsonb; v_huella text; v_total int; v_validas int; v_ordenadas int; v_contactos jsonb; BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR p_llamamiento !~ '^llamamiento:[0-9a-f]{64}$' OR p_recibo !~ '^recibo:llamamiento:[0-9a-f]{64}$' OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$' OR p_clave IS NULL OR p_clave<>btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256 OR jsonb_typeof(p_participaciones)<>'array' OR jsonb_array_length(p_participaciones) NOT BETWEEN 1 AND 100 OR NOT(SELECT bool_and(jsonb_typeof(value)='string') FROM jsonb_array_elements(p_participaciones)) OR jsonb_typeof(p_contactos)<>'array' OR jsonb_array_length(p_contactos)<>jsonb_array_length(p_participaciones) OR NOT(SELECT bool_and(jsonb_typeof(c.value)='object' AND (SELECT count(*)=3 FROM jsonb_object_keys(c.value)) AND c.value->>'participacion_ref'=p_participaciones->>(c.ordinality-1)::int AND c.value->>'resultado' IN('enviado','no_enviado') AND c.value->>'recibo_ref' ~ '^recibo:contacto:[0-9a-f]{64}$') FROM jsonb_array_elements(p_contactos) WITH ORDINALITY c(value,ordinality)) OR jsonb_typeof(p_configuracion)<>'object' OR p_configuracion ?& ARRAY['referencia','descripcion','categoria','centro','modalidad','fecha_inicio','plazo','plantilla_version','asunto','cuerpo'] IS NOT TRUE OR NOT(SELECT count(*)=10 AND bool_and(jsonb_typeof(value)='string' AND octet_length(value#>>'{}') BETWEEN 2 AND 4000 AND value#>>'{}'=btrim(value#>>'{}')) FROM jsonb_each(p_configuracion)) OR p_emitido IS NULL THEN RAISE EXCEPTION 'emisión B7 inválida' USING ERRCODE='22023'; END IF;
 v_huella:=encode(sha256(convert_to(p_bolsa||chr(31)||p_participaciones::text||chr(31)||p_configuracion::text,'UTF8')),'hex');
 SELECT * INTO previo FROM vec_bolsa_llamamientos.llamamiento_emitido WHERE bolsa_ref=p_bolsa AND clave_idempotencia=p_clave;
 IF FOUND THEN
  IF previo.actor_ref<>p_actor OR previo.huella_comando_sha256<>v_huella THEN RAISE EXCEPTION 'clave B7 divergente' USING ERRCODE='VBE01'; END IF;
  SELECT coalesce(jsonb_agg(jsonb_build_object('participacion_ref',c.participacion_ref,'resultado',c.resultado,'recibo_ref',c.recibo_ref) ORDER BY x.ordinality),'[]'::jsonb) INTO v_contactos FROM jsonb_array_elements_text(previo.participaciones) WITH ORDINALITY x(ref,ordinality) JOIN vec_bolsa_llamamientos.contacto_participacion c ON c.participacion_ref=x.ref AND c.llamamiento_ref=previo.llamamiento_ref;
  RETURN QUERY SELECT jsonb_build_object('llamamiento_ref',previo.llamamiento_ref,'recibo_ref',previo.recibo_ref,'bolsa_ref',previo.bolsa_ref,'estado',previo.estado,'participaciones',previo.participaciones,'configuracion',previo.configuracion,'emitido_en',previo.emitido_en,'contactos',v_contactos),true; RETURN;
 END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'decisión B7 inválida' USING ERRCODE='42501'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_bolsa OR consumo.consumo_nuevo IS NOT TRUE OR d->>'principal_id' IS DISTINCT FROM p_actor OR d->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1' OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida' THEN RAISE EXCEPTION 'emisión B7 no autorizada' USING ERRCODE='42501'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion c JOIN vec_bolsa_llamamientos.bolsa_constituida b ON b.bolsa_ref=c.bolsa_ref AND b.version=c.version_bolsa WHERE c.bolsa_ref=p_bolsa AND b.estado='vigente' AND b.vigente_desde<=p_emitido AND (b.vigente_hasta IS NULL OR p_emitido<b.vigente_hasta)) THEN RAISE EXCEPTION 'bolsa no vigente' USING ERRCODE='23503'; END IF;
 v_total:=jsonb_array_length(p_participaciones);
 SELECT count(*) INTO v_validas FROM jsonb_array_elements_text(p_participaciones) x(ref) JOIN vec_bolsa_llamamientos.constitucion c ON c.bolsa_ref=p_bolsa JOIN vec_bolsa_llamamientos.constitucion_entrada e ON e.instantanea_ref=c.instantanea_ref AND e.version_instantanea=c.version_instantanea AND e.participacion_ref=x.ref;
 SELECT count(*) INTO v_ordenadas FROM (SELECT e.orden,lag(e.orden) OVER(ORDER BY x.n) anterior FROM jsonb_array_elements_text(p_participaciones) WITH ORDINALITY x(ref,n) JOIN vec_bolsa_llamamientos.constitucion c ON c.bolsa_ref=p_bolsa JOIN vec_bolsa_llamamientos.constitucion_entrada e ON e.instantanea_ref=c.instantanea_ref AND e.version_instantanea=c.version_instantanea AND e.participacion_ref=x.ref) q WHERE anterior IS NULL OR orden>anterior;
 IF v_validas<>v_total OR v_ordenadas<>v_total OR (SELECT count(DISTINCT value) FROM jsonb_array_elements_text(p_participaciones))<>v_total THEN RAISE EXCEPTION 'participaciones ajenas, repetidas o desordenadas' USING ERRCODE='22023'; END IF;
 INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido VALUES(p_llamamiento,p_recibo,p_bolsa,p_actor,p_clave,p_participaciones,p_configuracion,v_huella,'emitido_pendiente_respuesta',p_emitido,consumo.decision_ref);
 INSERT INTO vec_bolsa_llamamientos.contacto_participacion(contacto_ref,bolsa_ref,participacion_ref,llamamiento_ref,canal,instante,actor,resultado,anotacion,clave_idempotencia,recibo_ref)
 SELECT 'contacto:'||encode(sha256(convert_to(p_bolsa||chr(31)||p_clave||chr(31)||(c.value->>'participacion_ref'),'UTF8')),'hex'),p_bolsa,c.value->>'participacion_ref',p_llamamiento,'correo',p_emitido,p_actor,c.value->>'resultado','Emisión de llamamiento por plantilla '||(p_configuracion->>'plantilla_version'),p_clave||':correo:'||c.ordinality,c.value->>'recibo_ref' FROM jsonb_array_elements(p_contactos) WITH ORDINALITY c(value,ordinality);
 RETURN QUERY SELECT jsonb_build_object('llamamiento_ref',p_llamamiento,'recibo_ref',p_recibo,'bolsa_ref',p_bolsa,'estado','emitido_pendiente_respuesta','participaciones',p_participaciones,'configuracion',p_configuracion,'emitido_en',p_emitido,'contactos',p_contactos),false;
END $f$;
CREATE FUNCTION vec_bolsa_llamamientos.recuperar_llamamiento_emitido_v1(p_bolsa text,p_clave text) RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT jsonb_build_object('llamamiento_ref',l.llamamiento_ref,'recibo_ref',l.recibo_ref,'bolsa_ref',l.bolsa_ref,'estado',l.estado,'participaciones',l.participaciones,'configuracion',l.configuracion,'emitido_en',l.emitido_en,'contactos',coalesce((SELECT jsonb_agg(jsonb_build_object('participacion_ref',c.participacion_ref,'resultado',c.resultado,'recibo_ref',c.recibo_ref) ORDER BY x.ordinality) FROM jsonb_array_elements_text(l.participaciones) WITH ORDINALITY x(ref,ordinality) JOIN vec_bolsa_llamamientos.contacto_participacion c ON c.participacion_ref=x.ref AND c.llamamiento_ref=l.llamamiento_ref),'[]'::jsonb)) FROM vec_bolsa_llamamientos.llamamiento_emitido l WHERE l.bolsa_ref=p_bolsa AND l.clave_idempotencia=p_clave
$f$;
CREATE FUNCTION vec_bolsa_llamamientos.contar_llamamientos_en_curso_v1(p_bolsa text) RETURNS bigint LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$ SELECT count(*) FROM vec_bolsa_llamamientos.llamamiento_emitido WHERE bolsa_ref=p_bolsa AND estado='emitido_pendiente_respuesta' $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.emitir_llamamiento_v1(text,text,text,text,text,jsonb,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_bolsa_llamamientos.recuperar_llamamiento_emitido_v1(text,text),vec_bolsa_llamamientos.contar_llamamientos_en_curso_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.emitir_llamamiento_v1(text,text,text,text,text,jsonb,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_bolsa_llamamientos.recuperar_llamamiento_emitido_v1(text,text),vec_bolsa_llamamientos.contar_llamamientos_en_curso_v1(text) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
