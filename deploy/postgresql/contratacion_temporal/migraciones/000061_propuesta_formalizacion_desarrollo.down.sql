\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000061_propuesta_formalizacion',0));
LOCK TABLE vec_contratacion_temporal.propuesta_formalizacion IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_contratacion_temporal.expediente_version_integral IN SHARE MODE;
DO $proteger$
DECLARE v_origen text;
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.propuesta_formalizacion)
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral
           WHERE origen_version='propuesta_formalizacion_o6')
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.outbox_expediente_integral
           WHERE tipo_evento='contratacion_temporal.propuesta_formalizacion_registrada')
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.actuacion_expediente_integral
           WHERE actuacion_json->>'accion_clave'='registrar_propuesta_formalizacion') THEN
        RAISE EXCEPTION 'reversión denegada: historia de propuesta conservada' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_constraintdef(oid) INTO STRICT v_origen FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
       AND conname='expediente_version_integral_origen_version_check' AND contype='c' AND convalidated;
    IF v_origen IS DISTINCT FROM $origen$CHECK ((origen_version = ANY (ARRAY['alta_o2'::text, 'analisis_o3'::text, 'cobertura_o4'::text, 'asignacion_o5'::text, 'informe_juridico_o5'::text, 'fiscalizacion_o5'::text, 'propuesta_formalizacion_o6'::text])))$origen$ THEN
        RAISE EXCEPTION 'origen integral incompatible para retirar propuesta' USING ERRCODE='55000';
    END IF;
END
$proteger$;
DROP FUNCTION vec_contratacion_temporal.registrar_propuesta_formalizacion_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_contratacion_temporal.propuesta_formalizacion;
-- Únicamente las cuatro publicaciones de instalación, sin historia de negocio.
DROP TABLE vec_contratacion_temporal.publicacion_propuesta_formalizacion_desarrollo;
ALTER TABLE vec_contratacion_temporal.expediente_version_integral
    DROP CONSTRAINT expediente_version_integral_origen_version_check;
ALTER TABLE vec_contratacion_temporal.expediente_version_integral
    ADD CONSTRAINT expediente_version_integral_origen_version_check CHECK ((origen_version = ANY (ARRAY['alta_o2'::text, 'analisis_o3'::text, 'cobertura_o4'::text, 'asignacion_o5'::text, 'informe_juridico_o5'::text, 'fiscalizacion_o5'::text])));
-- No borra consumos/auditorías V3: AD3-20 DOWN los protege por separado.
COMMIT;
