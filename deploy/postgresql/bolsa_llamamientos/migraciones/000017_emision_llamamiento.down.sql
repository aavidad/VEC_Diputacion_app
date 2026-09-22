\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000017',0));
DO $f$ BEGIN
	 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.llamamiento_emitido)
	    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contacto_participacion WHERE resultado IN('enviado','no_enviado'))
	    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento WHERE accion IN('consultar_datos_contacto','emitir_llamamiento','recuperar_llamamiento') OR ruta_clase='emisiones')
 THEN RAISE EXCEPTION 'hay historia B7; no se deshace' USING ERRCODE='55000'; END IF;
END $f$;
DROP FUNCTION vec_bolsa_llamamientos.contar_llamamientos_en_curso_v1(text),vec_bolsa_llamamientos.recuperar_llamamiento_emitido_v1(text,text),vec_bolsa_llamamientos.registrar_contactos_llamamiento_v1(text,text,text,jsonb),vec_bolsa_llamamientos.reservar_llamamiento_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_bolsa_llamamientos.llamamiento_emitido;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion DROP CONSTRAINT contacto_participacion_resultado_check;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion ADD CONSTRAINT contacto_participacion_resultado_check CHECK(resultado IN('contactado','no_contesta','buzon','acepta','rechaza','aplazado','otro'));
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(p_correlacion_ref text,p_accion text,p_ruta_clase text,p_actor_ref text,p_resultado text) RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_ref text; BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento();
 IF p_correlacion_ref IS NULL OR p_correlacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$' OR p_accion NOT IN('crear','consultar','cambiar_situacion','registrar_contacto') OR p_ruta_clase NOT IN('coleccion','detalle','situacion','contactos') OR (p_actor_ref IS NOT NULL AND p_actor_ref !~ '^per_[A-Za-z0-9_-]{22,128}$') OR p_resultado NOT IN('autenticacion_requerida','acceso_denegado','recurso_no_disponible','infraestructura_no_disponible','resultado_indeterminado') THEN RAISE EXCEPTION 'intento de frontera Bolsa inválido' USING ERRCODE='22023'; END IF;
 v_ref:='intento:'||translate(encode(sha256(convert_to(p_correlacion_ref||':'||p_accion||':'||p_ruta_clase||':'||coalesce(p_actor_ref,'')||':'||p_resultado||':'||clock_timestamp()::text,'UTF8')),'hex'),'0123456789','ghijklmnop');
 INSERT INTO vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento(intento_ref,correlacion_ref,accion,ruta_clase,actor_ref,resultado,registrada_en) VALUES(v_ref,p_correlacion_ref,p_accion,p_ruta_clase,p_actor_ref,p_resultado,clock_timestamp());
END $f$;
DO $bitacora$
DECLARE accion text; ruta text;
BEGIN
 SELECT pg_get_constraintdef(oid,true) INTO STRICT accion FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass AND conname='bitacora_intento_borrador_llamamiento_accion_check' AND contype='c';
 SELECT pg_get_constraintdef(oid,true) INTO STRICT ruta FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass AND conname='bitacora_intento_borrador_llamamiento_ruta_clase_check' AND contype='c';
	 accion:=replace(replace(replace(accion, ', ''consultar_datos_contacto''::text', ''), ', ''emitir_llamamiento''::text', ''), ', ''recuperar_llamamiento''::text', '');
 ruta:=replace(ruta, ', ''emisiones''::text', '');
 ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento DROP CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check;
 EXECUTE 'ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento ADD CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check '||accion;
 ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento DROP CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check;
 EXECUTE 'ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento ADD CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check '||ruta;
END $bitacora$;
COMMIT;
