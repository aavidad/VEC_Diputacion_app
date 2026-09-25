\set ON_ERROR_STOP on
-- CT118 DOWN: solo sin historia. Con alguna firma o devolución registrada se
-- rechaza: la historia de firmas es de solo adición y se conserva.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000118', 0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $pre$
BEGIN
    IF to_regclass('vec_contratacion_temporal.firma_documento_v1') IS NULL
       OR to_regclass('vec_contratacion_temporal.firma_documento_auditoria_v1') IS NULL
       OR to_regclass('vec_contratacion_temporal.firma_documento_outbox_v1') IS NULL THEN
        RAISE EXCEPTION 'CT118 DOWN: CT118 no instalada' USING ERRCODE='55000';
    END IF;
    LOCK TABLE vec_contratacion_temporal.firma_documento_v1,
               vec_contratacion_temporal.firma_documento_auditoria_v1,
               vec_contratacion_temporal.firma_documento_outbox_v1 IN ACCESS EXCLUSIVE MODE;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1)
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.firma_documento_auditoria_v1)
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.firma_documento_outbox_v1) THEN
        RAISE EXCEPTION 'CT118 DOWN: no admitido con historia de firmas' USING ERRCODE='55000';
    END IF;
END
$pre$;
DROP FUNCTION vec_contratacion_temporal.consultar_firmas_documento_v1(text,text);
DROP FUNCTION vec_contratacion_temporal.registrar_firma_documento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_contratacion_temporal.firma_documento_outbox_v1;
DROP TABLE vec_contratacion_temporal.firma_documento_auditoria_v1;
DROP TABLE vec_contratacion_temporal.firma_documento_v1;
COMMIT;
