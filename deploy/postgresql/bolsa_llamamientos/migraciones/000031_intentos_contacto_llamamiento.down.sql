\set ON_ERROR_STOP on
-- Revierte Bolsa 000031 solo si ningún contacto usa los resultados nuevos.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000031',0));
DO $precondicion$ BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_bolsa_llamamientos.registrar_contacto_participacion_v2(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,integer,integer,boolean,text[])') IS NULL THEN
  RAISE EXCEPTION 'estado incompatible para deshacer Bolsa 000031' USING ERRCODE='55000';
 END IF;
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contacto_participacion WHERE resultado IN('numero_erroneo','no_entregado')) THEN
  RAISE EXCEPTION 'hay historia con resultados de 000031; no se deshace' USING ERRCODE='55000';
 END IF;
END $precondicion$;
DROP FUNCTION vec_bolsa_llamamientos.registrar_contacto_participacion_v2(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,integer,integer,boolean,text[]) RESTRICT;
DROP INDEX vec_bolsa_llamamientos.contacto_participacion_intentos_llamamiento;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion DROP CONSTRAINT contacto_participacion_resultado_check;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion ADD CONSTRAINT contacto_participacion_resultado_check CHECK(resultado IN('contactado','no_contesta','buzon','acepta','rechaza','aplazado','otro','enviado','no_enviado'));
COMMIT;
