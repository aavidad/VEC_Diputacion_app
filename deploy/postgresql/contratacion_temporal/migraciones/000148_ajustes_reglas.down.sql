\set ON_ERROR_STOP on
-- CT-148 DOWN solo para ensayo sin historia. Las versiones de ajustes, sus
-- cambios, recibos, consumos y eventos no se eliminan en una base conservada.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000148',0));
DO $vacia$
BEGIN
 IF to_regclass('vec_contratacion_temporal.regla_ajuste_version_v1') IS NULL
 THEN RAISE EXCEPTION 'CT-148 DOWN: CT-148 no instalada' USING ERRCODE='55000'; END IF;
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.regla_ajuste_version_v1)
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.regla_ajuste_cambio_v1)
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.regla_ajuste_outbox_v1)
 THEN RAISE EXCEPTION 'CT-148: DOWN denegado con historia' USING ERRCODE='55000'; END IF;
END $vacia$;
DROP FUNCTION vec_contratacion_temporal.operar_ajustes_reglas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.leer_ajustes_reglas_en_v1(text,timestamptz);
DROP TABLE vec_contratacion_temporal.regla_ajuste_outbox_v1;
DROP TABLE vec_contratacion_temporal.regla_ajuste_cambio_v1;
DROP TABLE vec_contratacion_temporal.regla_ajuste_version_v1;
DROP FUNCTION vec_contratacion_temporal.ajustes_reglas_forma_valida_v1(jsonb);
COMMIT;
