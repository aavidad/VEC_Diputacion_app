BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:o4_05:consultas_rrhh:migraciones',
        0
    )
);

SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control
   AND version_esquema = 16
 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1
         FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control
           AND version_esquema = 16
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .control_migracion_consultas_rrhh
         WHERE control
           AND version_esquema = 1
    ) OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.registrar_acceso_rrhh_interno_v1(jsonb)'
    ) IS NULL OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.control_cadena_accesos_rrhh'
    ) IS NULL OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.registro_acceso_rrhh'
    ) IS NULL OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_contratacion_temporal_consultor_rrhh'
           AND NOT rolcanlogin
           AND NOT rolsuper
           AND NOT rolcreatedb
           AND NOT rolcreaterole
           AND rolinherit
           AND NOT rolreplication
           AND NOT rolbypassrls
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'down del registro de accesos RRHH fuera de orden';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.registro_acceso_rrhh
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'historia impide retirar registro de accesos RRHH';
    END IF;
END
$prevalidacion$;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.registrar_acceso_rrhh_interno_v1(jsonb)
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;

DROP FUNCTION
    vec_contratacion_temporal.registrar_acceso_rrhh_interno_v1(jsonb);
DROP TABLE vec_contratacion_temporal.registro_acceso_rrhh;
DROP TABLE vec_contratacion_temporal.control_cadena_accesos_rrhh;
DROP TABLE
    vec_contratacion_temporal.control_migracion_consultas_rrhh;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 15,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds',
           pg_catalog.clock_timestamp()
       )
 WHERE control
   AND version_esquema = 16;

COMMIT;
