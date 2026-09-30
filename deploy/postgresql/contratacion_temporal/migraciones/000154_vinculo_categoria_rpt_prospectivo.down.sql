\set ON_ERROR_STOP on
-- Solo para una instalación sin actos ni eventos. Nunca ejecutar sobre historia.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000154',0));
DO $pre$
BEGIN
 IF to_regclass('vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1') IS NULL
    OR to_regclass('vec_contratacion_temporal.evento_vinculo_categoria_rpt_ct_v1') IS NULL
 THEN RAISE EXCEPTION 'CT-154: DOWN prohibido con historia o preimagen ajena' USING ERRCODE='55000'; END IF;
END $pre$;
-- Los escritores no toman el advisory de migración. ACCESS EXCLUSIVE toma
-- primero el vínculo y después el evento, igual que sus escrituras, y se
-- mantiene hasta COMMIT; el conteo y los DROP observan el mismo límite.
LOCK TABLE vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1,
 vec_contratacion_temporal.evento_vinculo_categoria_rpt_ct_v1 IN ACCESS EXCLUSIVE MODE;
DO $historia$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1)
    OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.evento_vinculo_categoria_rpt_ct_v1)
 THEN RAISE EXCEPTION 'CT-154: DOWN prohibido con historia' USING ERRCODE='55000'; END IF;
END $historia$;
DROP FUNCTION vec_contratacion_temporal.registrar_vinculo_categoria_rpt_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.consultar_vinculo_categoria_rpt_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.anclaje_vinculo_categoria_rpt_ct154(text,text);
DROP TABLE vec_contratacion_temporal.evento_vinculo_categoria_rpt_ct_v1;
DROP TABLE vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1;
COMMIT;
