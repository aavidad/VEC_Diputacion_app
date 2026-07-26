BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_04:migraciones',0
  )
);
DO $control_presente$
BEGIN
  IF pg_catalog.to_regclass(
    'vec_contratacion_temporal.control_migracion_cobertura_o4'
  ) IS NULL THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='control CT ausente al retirar wrapper exacto O4-04E';
  END IF;
END
$control_presente$;
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control
 FOR UPDATE;
DO $barrera$
BEGIN
  IF NOT EXISTS(
    SELECT 1
      FROM vec_contratacion_temporal.control_migracion_cobertura_o4
     WHERE control AND version_esquema=7
  ) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='barrera CT incompatible al retirar wrapper exacto';
  END IF;
END
$barrera$;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $prevalidacion$
BEGIN
  IF EXISTS(
    SELECT 1 FROM pg_catalog.pg_proc p
    JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
     WHERE n.nspname='vec_contratacion_temporal'
       AND p.proname IN(
         'o404e_confirmar_denegacion_v1',
         'o404e_confirmar_concesion_v1',
         'confirmar_operacion_decision_cobertura_o404e_v1')
  ) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='llamadores CT impiden retirar wrapper exacto';
  END IF;
END
$prevalidacion$;
DROP FUNCTION
  vec_autorizacion.registrar_decision_cobertura_contexto_exacto_o404e_v1(
    bytea,bytea,numeric,numeric,jsonb),
  vec_autorizacion.o404e_claves_exactas_v1(jsonb,text[]),
  vec_autorizacion.o404e_mapa_json_go_v1(jsonb),
  vec_autorizacion.o404e_texto_json_go_v1(text);
COMMIT;
