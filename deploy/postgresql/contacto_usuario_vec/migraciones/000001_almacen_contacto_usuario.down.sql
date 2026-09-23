\set ON_ERROR_STOP on
-- Sólo deshacer en una base de ensayo sin historia ni migración 000002.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:migracion:1',0));
DO $pre$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
       OR to_regprocedure('vec_contacto_usuario_v1.consultar_recibo_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_contacto_usuario_v1' AND nspowner='vec_contacto_usuario_owner'::regrole)
       OR EXISTS (SELECT 1 FROM vec_contacto_usuario_v1.versiones)
       OR EXISTS (SELECT 1 FROM vec_contacto_usuario_v1.actual)
       OR EXISTS (SELECT 1 FROM vec_contacto_usuario_v1.outbox) THEN
        RAISE EXCEPTION 'contacto: DOWN sólo en ensayo vacío' USING ERRCODE='55000';
    END IF;
END $pre$;
SET LOCAL ROLE vec_contacto_usuario_owner;
DROP FUNCTION vec_contacto_usuario_v1.revalidar_consulta_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contacto_usuario_v1.consultar_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contacto_usuario_v1.registrar_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea);
DROP TABLE vec_contacto_usuario_v1.outbox;
DROP TABLE vec_contacto_usuario_v1.actual;
DROP TABLE vec_contacto_usuario_v1.versiones;
DROP SCHEMA vec_contacto_usuario_v1 RESTRICT;
COMMIT;
