\set ON_ERROR_STOP on
-- CT130 DOWN: solo instalación vacía. Con una reincorporación confirmada,
-- una versión o un evento no se borra historia: la reversión falla cerrada.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000130',0));
LOCK TABLE vec_contratacion_temporal.expediente_version_integral IN SHARE ROW EXCLUSIVE MODE;

DO $pre$
DECLARE v text;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR to_regclass('vec_contratacion_temporal.reincorporacion_titular_v1') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.verificar_reincorporacion_publicada_bolsa_v1(text,text,bigint)') IS NULL THEN
  RAISE EXCEPTION 'CT130 DOWN: estado incompatible' USING ERRCODE='55000'; END IF;
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.reincorporacion_titular_v1)
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral
       WHERE origen_version='reincorporacion_titular_ct130')
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.outbox_expediente_integral
       WHERE tipo_evento='ct.reincorporacion_titular.v1') THEN
  RAISE EXCEPTION 'CT130 DOWN: historia de reincorporación conservada' USING ERRCODE='55000'; END IF;
 IF to_regclass('vec_bolsa_llamamientos.reincorporacion_titular_ct') IS NOT NULL THEN
  RAISE EXCEPTION 'CT130 DOWN: B46 depende del verificador CT' USING ERRCODE='55000'; END IF;
 SELECT pg_get_constraintdef(oid) INTO STRICT v FROM pg_constraint
  WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
    AND conname='expediente_version_integral_origen_version_check';
 IF strpos(v,', ''reincorporacion_titular_ct130''::text')=0 THEN
  RAISE EXCEPTION 'CT130 DOWN: origen de versión incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

DROP FUNCTION vec_contratacion_temporal.verificar_reincorporacion_publicada_bolsa_v1(text,text,bigint);
DROP FUNCTION vec_contratacion_temporal.leer_reincorporaciones_bolsa_v1(bigint,text,integer);
DROP FUNCTION vec_contratacion_temporal.preparar_reincorporacion_titular_v1(jsonb);
DROP FUNCTION vec_contratacion_temporal.confirmar_reincorporacion_titular_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.resultado_reincorporacion_ct130(vec_contratacion_temporal.reincorporacion_titular_v1);
DROP FUNCTION vec_contratacion_temporal.origen_reincorporacion_ct130(jsonb);
DROP FUNCTION vec_contratacion_temporal.validar_material_reincorporacion_ct130(jsonb);
DROP TABLE vec_contratacion_temporal.reincorporacion_titular_v1;

DO $origen$
DECLARE v text;
BEGIN
 SELECT pg_get_constraintdef(oid) INTO STRICT v FROM pg_constraint
  WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
    AND conname='expediente_version_integral_origen_version_check';
 ALTER TABLE vec_contratacion_temporal.expediente_version_integral DROP CONSTRAINT expediente_version_integral_origen_version_check;
 EXECUTE 'ALTER TABLE vec_contratacion_temporal.expediente_version_integral ADD CONSTRAINT expediente_version_integral_origen_version_check '
  ||replace(v,', ''reincorporacion_titular_ct130''::text','');
END $origen$;
COMMIT;
