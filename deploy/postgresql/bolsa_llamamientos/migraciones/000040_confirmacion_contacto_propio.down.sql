\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000040', 0));
-- Con confirmaciones de las personas no se deshace: cada una consumió una
-- autorización que ya consta como efecto.
DO $f$ BEGIN
 IF to_regclass('vec_bolsa_llamamientos.confirmacion_contacto_participacion') IS NULL THEN
  RAISE EXCEPTION 'migracion 000040 no aplicada' USING ERRCODE='55000';
 END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.confirmacion_contacto_participacion) THEN
  RAISE EXCEPTION 'hay confirmaciones de contacto; no se deshace' USING ERRCODE='55000';
 END IF;
END $f$;
DROP FUNCTION vec_bolsa_llamamientos.leer_confirmacion_contacto_participacion_v1(text,bigint);
DROP FUNCTION vec_bolsa_llamamientos.leer_contacto_candidato_v1(text,timestamptz);
DROP FUNCTION vec_bolsa_llamamientos.confirmar_contacto_propio_v1(text,text,bigint,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_bolsa_llamamientos.registrar_confirmacion_contacto_interna_v1(text,text,text,bigint,text,text,timestamptz,text);
DROP FUNCTION vec_bolsa_llamamientos.participacion_contacto_candidato_v1(text,text);
DROP TABLE vec_bolsa_llamamientos.confirmacion_contacto_participacion;
COMMIT;
