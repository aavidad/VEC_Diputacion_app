\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000028', 0));
-- Con historia de ofertas no se deshace: publicar y resolver consumen
-- autorizaciones que ya constan como efecto.
DO $f$ BEGIN
 IF to_regclass('vec_bolsa_llamamientos.oferta_publicada') IS NULL THEN
  RAISE EXCEPTION 'migracion 000028 no aplicada' USING ERRCODE='55000';
 END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.oferta_publicada) THEN
  RAISE EXCEPTION 'hay historia de ofertas; no se deshace' USING ERRCODE='55000';
 END IF;
END $f$;
DROP FUNCTION vec_bolsa_llamamientos.listar_ofertas_bolsa_v1(text,timestamptz,integer);
DROP FUNCTION vec_bolsa_llamamientos.resolver_oferta_v1(text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_bolsa_llamamientos.publicar_oferta_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_bolsa_llamamientos.proyectar_oferta_v1(text,timestamptz);
DROP TABLE vec_bolsa_llamamientos.resolucion_oferta;
DROP TABLE vec_bolsa_llamamientos.disposicion_oferta;
DROP TABLE vec_bolsa_llamamientos.oferta_publicada;
COMMIT;
