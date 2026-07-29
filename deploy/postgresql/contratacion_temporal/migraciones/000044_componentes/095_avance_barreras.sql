-- CT-000044: avance CAS de barreras después de cerrar ACL y catálogo.
\if :ct000044_avanzar_barrera
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 8,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 7;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 24,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 23;

DO $barrera$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 24
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 8
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'no se pudo avanzar la barrera del motor RRHH';
    END IF;
END
$barrera$;
\endif
