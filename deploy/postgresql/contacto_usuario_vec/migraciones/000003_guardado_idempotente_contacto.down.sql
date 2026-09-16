\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contacto_usuario_owner;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:migracion:3',0));
LOCK TABLE vec_contacto_usuario_v1.intenciones IN ACCESS EXCLUSIVE MODE;
-- El owner debe comprobar TODA la tabla; la excepción revierte este cambio de RLS.
ALTER TABLE vec_contacto_usuario_v1.intenciones NO FORCE ROW LEVEL SECURITY;
DO $historia$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contacto_usuario_v1.intenciones) THEN
        RAISE EXCEPTION 'contacto V2: intenciones conservadas; no admite DOWN' USING ERRCODE='55000';
    END IF;
END $historia$;
DROP FUNCTION vec_contacto_usuario_v1.registrar_contacto_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DROP POLICY lectura_intencion ON vec_contacto_usuario_v1.versiones;
DROP TABLE vec_contacto_usuario_v1.intenciones RESTRICT;
COMMIT;
