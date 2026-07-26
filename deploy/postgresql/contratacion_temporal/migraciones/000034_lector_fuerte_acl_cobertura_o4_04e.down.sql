BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(
 pg_catalog.hashtextextended(
  'vec_contratacion_temporal:o4_04:migraciones',0));
SELECT control FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema=14 FOR UPDATE;
DO $prevalidacion$
BEGIN
 IF NOT EXISTS(
  SELECT 1 FROM vec_contratacion_temporal.control_migracion_cobertura_o4
   WHERE control AND version_esquema=14
 ) OR EXISTS(
  SELECT 1 FROM vec_contratacion_temporal
   .confirmacion_operacion_decision_cobertura
   WHERE carga_huella_sha256 IS NOT NULL
 ) OR EXISTS(
  SELECT 1 FROM vec_contratacion_temporal
   .prueba_denegacion_decision_cobertura
 ) THEN
  RAISE EXCEPTION USING ERRCODE='55000',
   MESSAGE='estado o historia O4-04E impide revertir lector fuerte';
 END IF;
END
$prevalidacion$;
REVOKE EXECUTE ON FUNCTION
 vec_contratacion_temporal.confirmar_operacion_decision_cobertura_o404e_v1(
  jsonb),
 vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404e_v1(
  jsonb)
FROM vec_contratacion_temporal_confirmador_cobertura;
REVOKE USAGE ON SCHEMA vec_contratacion_temporal
 FROM vec_contratacion_temporal_confirmador_cobertura;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
 SET version_esquema=13,
 actualizada_en=pg_catalog.date_trunc(
  'microseconds',pg_catalog.clock_timestamp())
 WHERE control AND version_esquema=14;
DROP FUNCTION
 vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404e_v1(
  jsonb);
COMMIT;
