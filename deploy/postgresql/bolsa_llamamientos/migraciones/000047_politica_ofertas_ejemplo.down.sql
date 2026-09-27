\set ON_ERROR_STOP on
-- B47 DOWN solo para una instalación sin política ni ofertas nuevas.
-- Nunca ejecutar sobre historia conservada.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000047',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.politica_ofertas_version') IS NULL
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.politica_ofertas_version)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.politica_ofertas_outbox)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.oferta_publicada
              WHERE plazo ? 'politica_version')
 THEN RAISE EXCEPTION 'B47 DOWN: política u oferta con historia; reversión prohibida' USING ERRCODE='55000'; END IF;
END $pre$;
DROP TRIGGER oferta_politica_b47 ON vec_bolsa_llamamientos.oferta_publicada;
DROP FUNCTION vec_bolsa_llamamientos.verificar_politica_oferta_b47();
DROP FUNCTION vec_bolsa_llamamientos.publicar_oferta_v2(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text);
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.publicar_oferta_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_ejecutor;
DROP FUNCTION vec_bolsa_llamamientos.publicar_politica_ofertas_v1(text,bigint,jsonb,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_bolsa_llamamientos.leer_politica_ofertas_v1(text);
DROP TABLE vec_bolsa_llamamientos.politica_ofertas_outbox;
DROP TABLE vec_bolsa_llamamientos.politica_ofertas_version;
COMMIT;
