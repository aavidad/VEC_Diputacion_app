-- O4-04D/1: retira tablas y cánones después de bajar 000024.
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
 WHERE control AND version_esquema = 3
 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.control_migracion_cobertura_o4
            WHERE control AND version_esquema = 3
       )
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.consumo_cobertura_lote'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.consumo_cobertura_evidencia'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.o404d_material_lote_v1(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.prevalidar_bloquear_lote_consumo_c1_cobertura_o404d_v1(jsonb)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.persistir_lote_consumo_c1_cobertura_o404d_v1(jsonb,jsonb)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada del esquema C1 O4-04D fuera de orden';
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
            MESSAGE = 'retirada del esquema C1 O4-04D con historia durable';
    END IF;
END
$prevalidacion$;

DROP FUNCTION vec_contratacion_temporal.o404d_material_lote_v1(jsonb);
DROP FUNCTION
vec_contratacion_temporal.o404d_material_evidencia_v1(jsonb);
DROP FUNCTION
vec_contratacion_temporal.o404d_prueba_canon_v1(jsonb, text);
DROP FUNCTION
vec_contratacion_temporal.o404d_dependencia_lector_o404c_v1(jsonb);

DROP TABLE vec_contratacion_temporal.consumo_cobertura_evidencia;
DROP TABLE vec_contratacion_temporal.consumo_cobertura_lote;

COMMIT;
