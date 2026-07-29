-- CT-000045: avance CAS solo después de cerrar funciones, ACL y catálogo.
\if :ct000045_avanzar_barrera
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 9,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 8;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 25,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 24;

DO $barrera$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 25
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 9
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE =
                'no se pudo avanzar la barrera de fachadas nominales RRHH';
    END IF;
END
$barrera$;
\endif
