\set ON_ERROR_STOP on
-- Fuente candidata: consulta del recibo histórico propio; no acredita replay de Guardar.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:migracion:2',0));
-- Sólo en ensayo vacío: una versión o lectura auditada impide reversión.
DO $historia$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN RAISE EXCEPTION 'contacto: DOWN requiere DBA' USING ERRCODE='42501'; END IF;
    IF EXISTS (SELECT 1 FROM vec_contacto_usuario_v1.versiones)
       OR EXISTS (SELECT 1 FROM vec_contacto_usuario_v1.actual)
       OR EXISTS (SELECT 1 FROM vec_contacto_usuario_v1.outbox) THEN
        RAISE EXCEPTION 'contacto: historia conservada; no admite DOWN' USING ERRCODE='55000';
    END IF;
END $historia$;
SET LOCAL ROLE vec_contacto_usuario_owner;
DROP FUNCTION vec_contacto_usuario_v1.consultar_version_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) RESTRICT;
DROP POLICY bloqueo_version ON vec_contacto_usuario_v1.actual;
DROP POLICY lectura_version ON vec_contacto_usuario_v1.actual;

DROP FUNCTION vec_contacto_usuario_v1.consultar_recibo_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea) RESTRICT;
COMMIT;
