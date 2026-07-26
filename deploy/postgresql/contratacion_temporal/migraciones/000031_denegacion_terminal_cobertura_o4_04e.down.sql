BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_04:migraciones',0));
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema=11 FOR UPDATE;
DO $prevalidacion$
BEGIN
  IF NOT EXISTS(
    SELECT 1 FROM vec_contratacion_temporal
      .control_migracion_cobertura_o4
     WHERE control AND version_esquema=11
  ) OR EXISTS(
    SELECT 1 FROM vec_contratacion_temporal
      .prueba_denegacion_decision_cobertura
  ) OR pg_catalog.to_regprocedure(
    'vec_contratacion_temporal.o404e_contexto_recurso_concesion_v1(jsonb)'
  ) IS NOT NULL
  OR pg_catalog.to_regprocedure(
    'vec_contratacion_temporal.o404e_semantica_propuesta_v1(jsonb)'
  ) IS NOT NULL
  OR pg_catalog.to_regprocedure(
    'vec_contratacion_temporal.o404e_concesion_ligada_v1(jsonb,jsonb)'
  ) IS NOT NULL
  OR pg_catalog.to_regprocedure(
    'vec_contratacion_temporal.o404e_transicion_dominio_ligada_v1('
    ||'jsonb,jsonb,jsonb,jsonb,timestamp with time zone)'
  ) IS NOT NULL
  OR pg_catalog.to_regprocedure(
    'vec_contratacion_temporal.o404e_revalidar_gobierno_actual_v1('
    ||'text,text,text,numeric,text,text,numeric,text,text,numeric,text)'
  ) IS NOT NULL
  THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='estado, historia o capa 000032 impide revertir rama denegada';
  END IF;
END
$prevalidacion$;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema=10,
       actualizada_en=pg_catalog.date_trunc(
         'microseconds',pg_catalog.clock_timestamp())
 WHERE control AND version_esquema=11;
DROP FUNCTION
  vec_contratacion_temporal.o404e_confirmar_denegacion_v1(
    jsonb,timestamptz);
COMMIT;
