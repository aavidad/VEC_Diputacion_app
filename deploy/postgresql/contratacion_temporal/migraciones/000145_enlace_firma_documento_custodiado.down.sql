\set ON_ERROR_STOP on
-- CT145 DOWN: solo sin enlaces registrados. Devuelve la ejecución de las v1 a
-- la aplicación y retira las v2 y la tabla del enlace. Con algún enlace se
-- rechaza: la historia es de solo adición y se conserva.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000145', 0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $pre$
BEGIN
    IF to_regclass('vec_contratacion_temporal.firma_documento_custodia_v1') IS NULL THEN
        RAISE EXCEPTION 'CT145 DOWN: CT145 no instalada' USING ERRCODE='55000';
    END IF;
    LOCK TABLE vec_contratacion_temporal.firma_documento_custodia_v1 IN ACCESS EXCLUSIVE MODE;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.firma_documento_custodia_v1) THEN
        RAISE EXCEPTION 'CT145 DOWN: no admitido con enlaces registrados' USING ERRCODE='55000';
    END IF;
END
$pre$;
DROP FUNCTION vec_contratacion_temporal.consultar_firmas_documento_v2(text,text);
DROP FUNCTION vec_contratacion_temporal.registrar_firma_documento_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_contratacion_temporal.firma_documento_custodia_v1;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_firma_documento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_firmas_documento_v1(text,text)
    TO vec_contratacion_temporal_ejecutor;
COMMIT;
