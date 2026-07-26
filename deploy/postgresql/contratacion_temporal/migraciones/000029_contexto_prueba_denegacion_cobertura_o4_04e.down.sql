BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';

SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_04:migraciones',0
  )
);
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema=9
 FOR UPDATE;

DO $prevalidacion$
BEGIN
  IF NOT EXISTS (
    SELECT 1
      FROM vec_contratacion_temporal.control_migracion_cobertura_o4
     WHERE control AND version_esquema=9
  ) OR EXISTS (
    SELECT 1
      FROM vec_contratacion_temporal.prueba_denegacion_decision_cobertura
  ) OR EXISTS (
    SELECT 1
      FROM vec_contratacion_temporal
        .confirmacion_operacion_decision_cobertura
     WHERE carga_huella_sha256 IS NOT NULL
  ) OR pg_catalog.to_regprocedure(
    'vec_contratacion_temporal.o404e_cerrar_terminal_v1('
    ||'jsonb,jsonb,jsonb,timestamp with time zone)'
  ) IS NOT NULL
  OR pg_catalog.to_regprocedure(
    'vec_contratacion_temporal.o404e_leer_terminal_interno_v1(text,text)'
  ) IS NOT NULL
  OR pg_catalog.to_regprocedure(
    'vec_contratacion_temporal.o404e_iguales_constante_v1(text,text)'
  ) IS NOT NULL
  THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='historia o capa 000030 impide revertir contexto O4-04E';
  END IF;
END
$prevalidacion$;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema=8,
       actualizada_en=
         pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp())
 WHERE control AND version_esquema=9;
DROP TABLE
  vec_contratacion_temporal.prueba_denegacion_decision_cobertura;
ALTER TABLE
  vec_contratacion_temporal.terminal_operacion_decision_cobertura
DROP CONSTRAINT terminal_ambito_decision_o404e_unico;
ALTER TABLE
  vec_contratacion_temporal.confirmacion_operacion_decision_cobertura
DROP CONSTRAINT confirmacion_carga_huella_o404e_valida,
DROP COLUMN carga_huella_sha256;
DROP FUNCTION
  vec_contratacion_temporal.o404e_motivo_denegacion_canon_v1(jsonb),
  vec_contratacion_temporal.o404e_contexto_recurso_denegacion_v1(jsonb),
  vec_contratacion_temporal.o404e_mapa_json_go_v1(jsonb),
  vec_contratacion_temporal.o404e_texto_json_go_v1(text);
COMMIT;
