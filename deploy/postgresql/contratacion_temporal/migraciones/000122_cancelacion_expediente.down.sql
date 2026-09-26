\set ON_ERROR_STOP on
-- CT122 DOWN: solo sin historia. Si algún expediente tiene una versión de
-- cancelación o su evento, la reversión se niega: la historia es de solo
-- adición y un expediente cancelado no puede volver a quedar en curso.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000122',0));
LOCK TABLE vec_contratacion_temporal.expediente_version_integral IN SHARE ROW EXCLUSIVE MODE;

DO $pre$
DECLARE v_origen text;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario'
       OR to_regclass('vec_contratacion_temporal.cancelacion_expediente_v1') IS NULL THEN
        RAISE EXCEPTION 'CT122 DOWN: CT122 no instalada' USING ERRCODE='55000';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral WHERE origen_version='cancelacion_expediente_ct122')
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.outbox_expediente_integral WHERE tipo_evento='ct.expediente_cancelado.v1') THEN
        RAISE EXCEPTION 'CT122 DOWN: no admitido con historia de cancelación' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_constraintdef(oid) INTO STRICT v_origen FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
       AND conname='expediente_version_integral_origen_version_check';
    IF length(v_origen)-length(replace(v_origen,', ''cancelacion_expediente_ct122''::text',''))<>length(', ''cancelacion_expediente_ct122''::text') THEN
        RAISE EXCEPTION 'CT122 DOWN: origen de versión no localizado' USING ERRCODE='55000';
    END IF;
END
$pre$;

DROP FUNCTION vec_contratacion_temporal.consultar_cancelacion_expediente_v1(text,text);
DROP FUNCTION vec_contratacion_temporal.confirmar_cancelacion_expediente_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.preparar_cancelacion_expediente_v1(jsonb);
DROP FUNCTION vec_contratacion_temporal.misma_intencion_ct122(vec_contratacion_temporal.cancelacion_expediente_v1,text,jsonb);
DROP FUNCTION vec_contratacion_temporal.resultado_cancelacion_ct122(vec_contratacion_temporal.cancelacion_expediente_v1);
DROP TABLE vec_contratacion_temporal.cancelacion_expediente_v1;
DROP FUNCTION vec_contratacion_temporal.admision_cancelacion_ct122(jsonb,numeric,jsonb);
DROP FUNCTION vec_contratacion_temporal.validar_confirmacion_ct122(jsonb,bytea,bytea,numeric,numeric);
DROP FUNCTION vec_contratacion_temporal.validar_preparacion_ct122(jsonb);
DROP FUNCTION vec_contratacion_temporal.validar_material_cancelacion_ct122(jsonb);
DROP FUNCTION vec_contratacion_temporal.exigir_sesion_ct122(boolean);
DROP FUNCTION vec_contratacion_temporal.fases_texto_ct122(jsonb);
DROP FUNCTION vec_contratacion_temporal.agregado_dominio_ct122(jsonb,numeric);
DROP FUNCTION vec_contratacion_temporal.huella_contexto_go_ct122(jsonb,jsonb);
DROP FUNCTION vec_contratacion_temporal.mapa_go_ct122(jsonb);
DROP FUNCTION vec_contratacion_temporal.referencia_valida_ct122(text);
DROP FUNCTION vec_contratacion_temporal.texto_valido_ct122(text,integer,boolean);

DO $origen$
DECLARE v_origen text;
BEGIN
    SELECT pg_get_constraintdef(oid) INTO STRICT v_origen FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
       AND conname='expediente_version_integral_origen_version_check';
    ALTER TABLE vec_contratacion_temporal.expediente_version_integral
        DROP CONSTRAINT expediente_version_integral_origen_version_check;
    EXECUTE 'ALTER TABLE vec_contratacion_temporal.expediente_version_integral ADD CONSTRAINT expediente_version_integral_origen_version_check '
        ||replace(v_origen,', ''cancelacion_expediente_ct122''::text','');
END
$origen$;
COMMIT;
