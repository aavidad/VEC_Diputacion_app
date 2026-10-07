\set ON_ERROR_STOP on
-- Solo para ensayo reversible sin historia B72. Nunca ejecutar sobre una
-- base que conserve contactos asociados a ofertas o evidencias de entrega.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000072',0));
DO $precondicion$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.contacto_participacion') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.registrar_contacto_participacion_v3(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,text)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.listar_contactos_participacion_v2(text,text,text,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)') IS NULL THEN
  RAISE EXCEPTION 'B72: estado incompatible para deshacer' USING ERRCODE='55000';
 END IF;
END $precondicion$;
LOCK TABLE vec_bolsa_llamamientos.contacto_participacion IN ACCESS EXCLUSIVE MODE;
DO $historia$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contacto_participacion
            WHERE oferta_ref IS NOT NULL OR evidencia_ref IS NOT NULL
               OR evidencia_huella IS NOT NULL OR resultado='entrega_declarada') THEN
  RAISE EXCEPTION 'B72: existe historia de ofrecimientos; no se deshace' USING ERRCODE='55000';
 END IF;
END $historia$;
-- Restaura exclusivamente la guarda añadida por B72, antes de retirar las
-- columnas. La comprobación de historia anterior ya impide perder datos.
DO $restaurar_replay_legacy$
DECLARE f regprocedure; original text; cuerpo text; esperada text; actual text; metadata jsonb;
 funciones regprocedure[]:=ARRAY[
  'vec_bolsa_llamamientos.registrar_contacto_participacion_v1(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_bolsa_llamamientos.registrar_contacto_participacion_v2(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,integer,integer,boolean,text[])'::regprocedure];
 marca text:=$marca$ IF FOUND THEN
  IF anterior.bolsa_ref<>p_bolsa_ref$marca$;
 guarda text:=$guarda$ IF FOUND THEN
  IF anterior.oferta_ref IS NOT NULL THEN
   RAISE EXCEPTION 'B72: contacto de oferta requiere versión tres' USING ERRCODE='VBC01';
  END IF;
  IF anterior.bolsa_ref<>p_bolsa_ref$guarda$;
BEGIN
 FOREACH f IN ARRAY funciones LOOP
  SELECT pg_get_functiondef(p.oid),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,cuerpo,metadata
   FROM pg_proc p WHERE p.oid=f;
  IF (length(cuerpo)-length(replace(cuerpo,guarda,'')))<>length(guarda)
     OR (SELECT proowner FROM pg_proc WHERE oid=f)<>'vec_bolsa_llamamientos_propietario'::regrole
     OR NOT (SELECT prosecdef FROM pg_proc WHERE oid=f) THEN
   RAISE EXCEPTION 'B72: postimagen de replay incompatible para deshacer' USING ERRCODE='55000';
  END IF;
  esperada:=replace(original,guarda,marca);
  EXECUTE esperada;
  SELECT pg_get_functiondef(p.oid) INTO STRICT actual FROM pg_proc p WHERE p.oid=f;
  IF actual IS DISTINCT FROM esperada OR replace(actual,marca,guarda) IS DISTINCT FROM original
     OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata THEN
   RAISE EXCEPTION 'B72: restauración alteró metadata o cuerpo fuera de la guarda' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $restaurar_replay_legacy$;
DROP FUNCTION vec_bolsa_llamamientos.registrar_contacto_participacion_v3(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,text) RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.listar_contactos_participacion_v2(text,text,text,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text) RESTRICT;
DROP INDEX vec_bolsa_llamamientos.contacto_participacion_oferta_fecha;
DROP INDEX vec_bolsa_llamamientos.contacto_participacion_oferta_participacion_fecha;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion
 DROP CONSTRAINT contacto_participacion_oferta_check,
 DROP CONSTRAINT contacto_participacion_evidencia_check,
 DROP CONSTRAINT contacto_participacion_resultado_check,
 DROP COLUMN oferta_ref RESTRICT,
 DROP COLUMN evidencia_ref RESTRICT,
 DROP COLUMN evidencia_huella RESTRICT;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion
 ADD CONSTRAINT contacto_participacion_resultado_check CHECK(resultado IN(
  'contactado','no_contesta','buzon','acepta','rechaza','aplazado','otro',
  'enviado','no_enviado','numero_erroneo','no_entregado'));
COMMIT;
