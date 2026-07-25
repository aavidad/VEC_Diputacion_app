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
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.consultar_operacion_decision_cobertura_confirmada_v1(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404c_v1(jsonb)'
       ) IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada de lectores O4-04C fuera de orden';
    END IF;
END
$prevalidacion$;

REVOKE EXECUTE ON FUNCTION
vec_contratacion_temporal.consultar_operacion_decision_cobertura_confirmada_v1(
    jsonb
)
FROM vec_contratacion_temporal_ejecutor;

DROP FUNCTION
vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404c_v1(
    jsonb
);
DROP FUNCTION
vec_contratacion_temporal.consultar_operacion_decision_cobertura_confirmada_v1(
    jsonb
);

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 2,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 3;

COMMIT;
