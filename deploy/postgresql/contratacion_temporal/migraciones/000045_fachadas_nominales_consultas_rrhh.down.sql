\set ON_ERROR_STOP on
\set ct000045_aplicar_acl false
\set ct000045_avanzar_barrera false
-- CT-000045: retirada conservadora; no borra historia ni cursores.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_05:consultas_rrhh:migraciones', 0
));
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema = 25
 FOR UPDATE;
SELECT control
  FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
 WHERE control AND version_esquema = 9
 FOR UPDATE;

DO $puerta$
BEGIN
    IF pg_catalog.current_setting('server_version_num') <> '180004'
       OR pg_catalog.getdatabaseencoding() IS DISTINCT FROM 'UTF8'
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .control_migracion_cobertura_o4
            WHERE control AND version_esquema = 25
       )
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .control_migracion_consultas_rrhh
            WHERE control AND version_esquema = 9
       )
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_proc funcion
            WHERE funcion.pronamespace =
                  'vec_contratacion_temporal'::pg_catalog.regnamespace
              AND funcion.proname = ANY(ARRAY[
                  'consultar_cuadro_rrhh_atestado_v1',
                  'consultar_detalle_rrhh_atestado_v1'
              ]::name[])
       ) <> 2 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE =
                'estado incompatible para revertir fachadas nominales RRHH';
    END IF;
END
$puerta$;

-- Recalcula la misma ACL y huella antes de retirar el primer objeto.
\ir 000045_componentes/090_acl_catalogo.sql

DROP FUNCTION
vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) RESTRICT;

REVOKE USAGE ON SCHEMA vec_contratacion_temporal
FROM vec_contratacion_temporal_consultor_rrhh;

UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 8,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 9;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 24,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 25;

DO $retirada$
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
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc funcion
         WHERE funcion.pronamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND funcion.proname = ANY(ARRAY[
               'consultar_cuadro_rrhh_atestado_v1',
               'consultar_detalle_rrhh_atestado_v1'
           ]::name[])
    ) OR pg_catalog.has_schema_privilege(
        'vec_contratacion_temporal_consultor_rrhh',
        'vec_contratacion_temporal', 'USAGE'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada de fachadas nominales RRHH incompleta';
    END IF;
END
$retirada$;

COMMIT;
