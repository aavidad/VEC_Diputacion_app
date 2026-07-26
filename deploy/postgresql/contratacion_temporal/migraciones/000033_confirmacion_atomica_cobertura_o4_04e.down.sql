BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(
 pg_catalog.hashtextextended(
  'vec_contratacion_temporal:o4_04:migraciones',0));
SELECT control FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema=13 FOR UPDATE;
DO $prevalidacion$
BEGIN
 IF NOT EXISTS(
  SELECT 1 FROM vec_contratacion_temporal.control_migracion_cobertura_o4
   WHERE control AND version_esquema=13
 ) OR pg_catalog.to_regprocedure(
  'vec_contratacion_temporal.'
  ||'leer_terminal_primario_decision_cobertura_o404e_v1(jsonb)'
 ) IS NOT NULL
 OR NOT EXISTS(
  SELECT 1 FROM pg_catalog.pg_roles
   WHERE rolname='vec_contratacion_temporal_confirmador_cobertura'
 )
 OR pg_catalog.has_schema_privilege(
  (
   SELECT oid FROM pg_catalog.pg_roles
    WHERE rolname='vec_contratacion_temporal_confirmador_cobertura'
  ),
  'vec_contratacion_temporal','USAGE'
 )
 OR pg_catalog.has_function_privilege(
  (
   SELECT oid FROM pg_catalog.pg_roles
    WHERE rolname='vec_contratacion_temporal_confirmador_cobertura'
  ),
  'vec_contratacion_temporal.'
  ||'confirmar_operacion_decision_cobertura_o404e_v1(jsonb)',
  'EXECUTE'
 )
 THEN
  RAISE EXCEPTION USING ERRCODE='55000',
   MESSAGE='estado o capa 000034 impide revertir confirmación O4-04E';
 END IF;
END
$prevalidacion$;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
 SET version_esquema=12,
 actualizada_en=pg_catalog.date_trunc(
  'microseconds',pg_catalog.clock_timestamp())
 WHERE control AND version_esquema=13;
DROP FUNCTION
 vec_contratacion_temporal.confirmar_operacion_decision_cobertura_o404e_v1(
  jsonb),
 vec_contratacion_temporal.o404e_confirmar_concesion_v1(
  jsonb,text,timestamptz);
COMMIT;
