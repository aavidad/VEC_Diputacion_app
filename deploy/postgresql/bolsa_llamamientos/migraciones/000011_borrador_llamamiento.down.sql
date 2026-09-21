\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000011',0));
LOCK TABLE vec_bolsa_llamamientos.borrador_llamamiento_interno,vec_bolsa_llamamientos.borrador_llamamiento_historia,vec_bolsa_llamamientos.borrador_llamamiento_auditoria,vec_bolsa_llamamientos.borrador_llamamiento_outbox,vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento IN ACCESS EXCLUSIVE MODE;
DO $f$ BEGIN
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.borrador_llamamiento_interno) OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.borrador_llamamiento_historia) OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.borrador_llamamiento_auditoria) OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.borrador_llamamiento_outbox) OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento) THEN RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='se conserva historia o bitácora de borrador llamamiento; reversión denegada'; END IF;
END $f$;
REVOKE EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text) FROM vec_bolsa_llamamientos_registrador_frontera;
REVOKE USAGE ON SCHEMA vec_bolsa_llamamientos FROM vec_bolsa_llamamientos_registrador_frontera;
DROP FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text);
DROP FUNCTION vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento();
DROP FUNCTION vec_bolsa_llamamientos.consultar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_bolsa_llamamientos.exigir_runtime_borrador_llamamiento();
DROP TABLE vec_bolsa_llamamientos.borrador_llamamiento_outbox;
DROP TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento;
DROP TABLE vec_bolsa_llamamientos.borrador_llamamiento_auditoria;
DROP TABLE vec_bolsa_llamamientos.borrador_llamamiento_historia;
DROP TABLE vec_bolsa_llamamientos.borrador_llamamiento_interno;
DROP FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_rechazar_mutacion();
DROP FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_contenido_valido(jsonb);
COMMIT;
