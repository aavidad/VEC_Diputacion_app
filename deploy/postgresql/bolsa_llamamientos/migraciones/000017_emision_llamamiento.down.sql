\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000017',0));
DO $f$ BEGIN
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.llamamiento_emitido)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contacto_participacion WHERE resultado IN('enviado','no_enviado'))
 THEN RAISE EXCEPTION 'hay historia B7; no se deshace' USING ERRCODE='55000'; END IF;
END $f$;
DROP FUNCTION vec_bolsa_llamamientos.contar_llamamientos_en_curso_v1(text),vec_bolsa_llamamientos.recuperar_llamamiento_emitido_v1(text,text),vec_bolsa_llamamientos.emitir_llamamiento_v1(text,text,text,text,text,jsonb,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_bolsa_llamamientos.llamamiento_emitido;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion DROP CONSTRAINT contacto_participacion_resultado_check;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion ADD CONSTRAINT contacto_participacion_resultado_check CHECK(resultado IN('contactado','no_contesta','buzon','acepta','rechaza','aplazado','otro'));
COMMIT;
