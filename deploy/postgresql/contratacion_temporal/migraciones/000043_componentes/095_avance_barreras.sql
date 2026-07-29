-- CT-000043: avance de barreras únicamente tras cerrar el catálogo.
\if :ct000043_avanzar_barrera
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 7,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 6;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 23,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 22;

DO $barrera$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 23
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 7
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'no se pudo avanzar la barrera de prueba RRHH';
    END IF;
END
$barrera$;
\endif
