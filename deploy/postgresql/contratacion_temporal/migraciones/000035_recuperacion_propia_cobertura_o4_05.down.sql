BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_04:migraciones',0));
SELECT control FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema=15 FOR UPDATE;
DO $prevalidacion$
BEGIN
  IF NOT EXISTS(
    SELECT 1 FROM vec_contratacion_temporal
      .control_migracion_cobertura_o4
     WHERE control AND version_esquema=15
  ) OR pg_catalog.to_regprocedure(
    'vec_contratacion_temporal.recuperar_resultado_propio_decision_cobertura_o405_v1(jsonb)'
  ) IS NULL THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='down de recuperación propia O4-05 fuera de orden';
  END IF;
  IF EXISTS(
    SELECT 1 FROM vec_contratacion_temporal
      .reserva_operacion_decision_cobertura
  ) OR EXISTS(
    SELECT 1 FROM vec_contratacion_temporal
      .confirmacion_operacion_decision_cobertura
  ) OR EXISTS(
    SELECT 1 FROM vec_contratacion_temporal
      .terminal_operacion_decision_cobertura
  ) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='historia de cobertura impide retirar recuperación propia O4-05';
  END IF;
END
$prevalidacion$;
REVOKE EXECUTE ON FUNCTION
 vec_contratacion_temporal
  .recuperar_resultado_propio_decision_cobertura_o405_v1(jsonb)
 FROM vec_contratacion_temporal_lector_resultado_cobertura;
REVOKE USAGE ON SCHEMA vec_contratacion_temporal
 FROM vec_contratacion_temporal_lector_resultado_cobertura;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
 SET version_esquema=14,
 actualizada_en=pg_catalog.date_trunc(
   'microseconds',pg_catalog.clock_timestamp())
 WHERE control AND version_esquema=15;
DROP FUNCTION vec_contratacion_temporal
 .recuperar_resultado_propio_decision_cobertura_o405_v1(jsonb);
COMMIT;
