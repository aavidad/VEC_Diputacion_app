-- O4-04D/2: retira operaciones y devuelve la barrera v4 a v3.
-- Nunca destruye historia C1.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:o4_04:migraciones', 0
    )
);

SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema = 4
 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.control_migracion_cobertura_o4
            WHERE control AND version_esquema = 4
       )
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.consumo_cobertura_lote'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.consumo_cobertura_evidencia'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.prevalidar_bloquear_lote_consumo_c1_cobertura_o404d_v1(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.persistir_lote_consumo_c1_cobertura_o404d_v1(jsonb,jsonb)'
       ) IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada de operaciones C1 O4-04D fuera de orden';
    END IF;
    IF EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.consumo_cobertura_lote
       )
       OR EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.consumo_cobertura_evidencia
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada de operaciones C1 O4-04D con historia durable';
    END IF;
END
$prevalidacion$;

DROP POLICY propietario_total
    ON vec_contratacion_temporal.consumo_cobertura_evidencia;
DROP POLICY propietario_total
    ON vec_contratacion_temporal.consumo_cobertura_lote;

DROP TRIGGER bloquear_truncado
    ON vec_contratacion_temporal.consumo_cobertura_evidencia;
DROP TRIGGER bloquear_mutacion
    ON vec_contratacion_temporal.consumo_cobertura_evidencia;
DROP TRIGGER bloquear_truncado
    ON vec_contratacion_temporal.consumo_cobertura_lote;
DROP TRIGGER bloquear_mutacion
    ON vec_contratacion_temporal.consumo_cobertura_lote;

ALTER TABLE vec_contratacion_temporal.consumo_cobertura_evidencia
    NO FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.consumo_cobertura_evidencia
    DISABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.consumo_cobertura_lote
    NO FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.consumo_cobertura_lote
    DISABLE ROW LEVEL SECURITY;

DROP FUNCTION
vec_contratacion_temporal.persistir_lote_consumo_c1_cobertura_o404d_v1(
    jsonb, jsonb
);
DROP FUNCTION
vec_contratacion_temporal.prevalidar_bloquear_lote_consumo_c1_cobertura_o404d_v1(
    jsonb
);

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 3,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 4;

COMMIT;
