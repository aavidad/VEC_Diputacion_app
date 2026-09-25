\set ON_ERROR_STOP on
-- CT116 DOWN: solo sin historia de modificaciones.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000116',0));
LOCK TABLE vec_contratacion_temporal.expediente_version_integral IN SHARE ROW EXCLUSIVE MODE;

DO $pre$
DECLARE v_origen text;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario'
       OR to_regclass('vec_contratacion_temporal.modificacion_nombramiento_v1') IS NULL THEN
        RAISE EXCEPTION 'CT116 DOWN: estado incompatible' USING ERRCODE='55000';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral WHERE origen_version='modificacion_nombramiento_ct116')
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.outbox_expediente_integral WHERE tipo_evento='ct.modificacion.v1') THEN
        RAISE EXCEPTION 'CT116 DOWN: no admitido con historia de modificaciones' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_constraintdef(oid) INTO STRICT v_origen FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
       AND conname='expediente_version_integral_origen_version_check';
    IF strpos(v_origen,', ''modificacion_nombramiento_ct116''::text')=0 THEN
        RAISE EXCEPTION 'CT116 DOWN: origen de versión no localizado' USING ERRCODE='55000';
    END IF;
END
$pre$;

DROP FUNCTION vec_contratacion_temporal.confirmar_modificacion_nombramiento_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.preparar_modificacion_nombramiento_v1(jsonb);
DROP FUNCTION vec_contratacion_temporal.resultado_modificacion_ct116(vec_contratacion_temporal.modificacion_nombramiento_v1);
DROP FUNCTION vec_contratacion_temporal.proyeccion_modificacion_ct116(jsonb,jsonb,text,jsonb);
DROP FUNCTION vec_contratacion_temporal.validar_material_modificacion_ct116(jsonb);
DROP TABLE vec_contratacion_temporal.modificacion_nombramiento_v1;

DO $origen$
DECLARE v_origen text;
BEGIN
    SELECT pg_get_constraintdef(oid) INTO STRICT v_origen FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
       AND conname='expediente_version_integral_origen_version_check';
    ALTER TABLE vec_contratacion_temporal.expediente_version_integral
        DROP CONSTRAINT expediente_version_integral_origen_version_check;
    EXECUTE 'ALTER TABLE vec_contratacion_temporal.expediente_version_integral ADD CONSTRAINT expediente_version_integral_origen_version_check '
        ||replace(v_origen,', ''modificacion_nombramiento_ct116''::text','');
END
$origen$;
COMMIT;
