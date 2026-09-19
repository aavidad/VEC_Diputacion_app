\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
DO $f$ BEGIN
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.llamamiento_operativo) THEN RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='hay llamamientos operativos; no se deshace'; END IF;
END $f$;
DROP FUNCTION vec_bolsa_llamamientos.registrar_resultado_llamamiento_operativo_v1(text,text,text,text,timestamptz) RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.abrir_llamamiento_operativo_v1(text,text,text,text,text,timestamptz,timestamptz,text) RESTRICT;
DROP FUNCTION vec_bolsa_llamamientos.contactos_participacion_v1(text) RESTRICT;
DROP TABLE vec_bolsa_llamamientos.outbox_llamamiento_operativo RESTRICT;
DROP TABLE vec_bolsa_llamamientos.auditoria_llamamiento_operativo RESTRICT;
DROP TABLE vec_bolsa_llamamientos.situacion_participacion_llamamiento RESTRICT;
DROP TABLE vec_bolsa_llamamientos.llamamiento_operativo RESTRICT;
COMMIT;
