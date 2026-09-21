\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000012', 0));
DO $f$ BEGIN IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion) THEN RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='hay situaciones registradas; no se deshace'; END IF; END $f$;
DROP TRIGGER constitucion_entrada_inicia_situacion_participacion ON vec_bolsa_llamamientos.constitucion_entrada;
DROP FUNCTION vec_bolsa_llamamientos.iniciar_situacion_participacion_v1() RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.leer_situacion_participacion_v1(text) RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.registrar_situacion_participacion_v1(text,text,timestamptz,timestamptz,text,text,text,text,timestamptz) RESTRICT;
DROP TABLE vec_bolsa_llamamientos.situacion_participacion RESTRICT;
COMMIT;
