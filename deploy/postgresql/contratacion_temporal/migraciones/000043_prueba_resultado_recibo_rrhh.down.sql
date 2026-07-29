\set ON_ERROR_STOP on
\set ct000043_aplicar_acl false
\set ct000043_avanzar_barrera false
-- CT-000043: reversión conservadora. La prueba durable nunca se elimina.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_05:consultas_rrhh:migraciones', 0
));

-- La puerta produce un rechazo estable también en reentrada, antes de
-- referenciar estáticamente la tabla que el primer DOWN ya retiró.
DO $puerta$
BEGIN
    IF pg_catalog.current_setting('server_version_num') <> '180004'
       OR pg_catalog.getdatabaseencoding() IS DISTINCT FROM 'UTF8'
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .control_migracion_cobertura_o4
            WHERE control AND version_esquema = 23
       )
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .control_migracion_consultas_rrhh
            WHERE control AND version_esquema = 7
       )
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para revertir prueba RRHH';
    END IF;
END
$puerta$;

LOCK TABLE
    vec_contratacion_temporal.control_migracion_cobertura_o4,
    vec_contratacion_temporal.control_migracion_consultas_rrhh,
    vec_contratacion_temporal.registro_acceso_rrhh,
    vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2,
    vec_contratacion_temporal.alcance_acceso_rrhh,
    vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
BEGIN
    IF pg_catalog.current_setting('server_version_num') <> '180004'
       OR pg_catalog.getdatabaseencoding() IS DISTINCT FROM 'UTF8'
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .control_migracion_cobertura_o4
            WHERE control AND version_esquema = 23
       )
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .control_migracion_consultas_rrhh
            WHERE control AND version_esquema = 7
       )
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.cerrar_prueba_resultado_recibo_rrhh_v2('
           || 'vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2,'
           || 'vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2,'
           || 'vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,'
           || 'bytea,bytea,bytea,bytea,numeric,numeric,'
           || 'bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.unnest(ARRAY[
                 'contexto_cierre_prueba_rrhh_v2',
                 'contenido_cierre_prueba_rrhh_v2',
                 'evidencia_consumo_nuevo_rrhh_v3',
                 'resultado_cierre_prueba_rrhh_v2',
                 '_contexto_cierre_prueba_rrhh_v2',
                 '_contenido_cierre_prueba_rrhh_v2',
                 '_evidencia_consumo_nuevo_rrhh_v3',
                 '_resultado_cierre_prueba_rrhh_v2',
                 'prueba_resultado_recibo_rrhh_v2',
                 '_prueba_resultado_recibo_rrhh_v2'
             ]::text[]) AS esperado(nombre)
            WHERE pg_catalog.to_regtype(
                'vec_contratacion_temporal.' || esperado.nombre
            ) IS NULL
       )
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_attribute atributo
            WHERE atributo.attrelid =
                  'vec_contratacion_temporal.'
                  'registro_acceso_rrhh'::pg_catalog.regclass
              AND atributo.attnum > 0
              AND NOT atributo.attisdropped
              AND atributo.attname IN (
                  'expediente_ref_prueba_v2',
                  'version_expediente_prueba_v2'
              )
       ) <> 2
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_constraint restriccion
            WHERE restriccion.conname = ANY(ARRAY[
                  'registro_acceso_rrhh_prueba_resultado_v2_unica',
                  'registro_acceso_rrhh_prueba_cadena_v2_unica',
                  'registro_acceso_rrhh_prueba_vec_v2_unica',
                  'vinculo_identidad_acceso_rrhh_prueba_v2_unica',
                  'alcance_acceso_rrhh_prueba_v2_unica'
              ]::name[])
              AND restriccion.conrelid IN (
                  'vec_contratacion_temporal.'
                  'registro_acceso_rrhh'::pg_catalog.regclass,
                  'vec_contratacion_temporal.'
                  'vinculo_identidad_acceso_rrhh_v2'::pg_catalog.regclass,
                  'vec_contratacion_temporal.'
                  'alcance_acceso_rrhh'::pg_catalog.regclass
              )
       ) <> 5 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para revertir prueba RRHH';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .prueba_resultado_recibo_rrhh_v2
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'existe prueba durable RRHH; reversión prohibida';
    END IF;
END
$prevalidacion$;

-- Valida la misma huella literal y el mismo cierre de catálogo que el UP.
\ir 000043_componentes/085_guardia_columnas_padre.sql
\ir 000043_componentes/090_acl_catalogo_y_barrera.sql

DROP FUNCTION
vec_contratacion_temporal.cerrar_prueba_resultado_recibo_rrhh_v2(
    vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2,
    vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2,
    vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) RESTRICT;

DROP TABLE
vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
RESTRICT;

ALTER TABLE vec_contratacion_temporal.alcance_acceso_rrhh
DROP CONSTRAINT alcance_acceso_rrhh_prueba_v2_unica
RESTRICT;
ALTER TABLE
vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2
DROP CONSTRAINT vinculo_identidad_acceso_rrhh_prueba_v2_unica
RESTRICT;
ALTER TABLE vec_contratacion_temporal.registro_acceso_rrhh
DROP CONSTRAINT registro_acceso_rrhh_prueba_vec_v2_unica
RESTRICT,
DROP CONSTRAINT registro_acceso_rrhh_prueba_cadena_v2_unica
RESTRICT,
DROP CONSTRAINT registro_acceso_rrhh_prueba_resultado_v2_unica
RESTRICT,
DROP COLUMN version_expediente_prueba_v2
RESTRICT,
DROP COLUMN expediente_ref_prueba_v2
RESTRICT;

DROP TYPE
vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2
RESTRICT;
DROP TYPE
vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3
RESTRICT;
DROP TYPE
vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2
RESTRICT;
DROP TYPE
vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2
RESTRICT;

UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 6,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 7;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 22,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 23;

DO $retirada$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 22
    )
    OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 6
    )
    OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2'
    ) IS NOT NULL
    OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type tipo
         WHERE tipo.typnamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND tipo.typname = ANY(ARRAY[
               'contexto_cierre_prueba_rrhh_v2',
               'contenido_cierre_prueba_rrhh_v2',
               'evidencia_consumo_nuevo_rrhh_v3',
               'resultado_cierre_prueba_rrhh_v2',
               '_contexto_cierre_prueba_rrhh_v2',
               '_contenido_cierre_prueba_rrhh_v2',
               '_evidencia_consumo_nuevo_rrhh_v3',
               '_resultado_cierre_prueba_rrhh_v2',
               'prueba_resultado_recibo_rrhh_v2',
               '_prueba_resultado_recibo_rrhh_v2'
           ]::name[])
    )
    OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc funcion
         WHERE funcion.pronamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND funcion.proname =
               'cerrar_prueba_resultado_recibo_rrhh_v2'
    )
    OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_attribute atributo
         WHERE atributo.attrelid =
               'vec_contratacion_temporal.'
               'registro_acceso_rrhh'::pg_catalog.regclass
           AND atributo.attnum > 0
           AND NOT atributo.attisdropped
           AND atributo.attname IN (
               'expediente_ref_prueba_v2',
               'version_expediente_prueba_v2'
           )
    )
    OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_constraint restriccion
         WHERE restriccion.conname = ANY(ARRAY[
               'registro_acceso_rrhh_prueba_resultado_v2_unica',
               'registro_acceso_rrhh_prueba_cadena_v2_unica',
               'registro_acceso_rrhh_prueba_vec_v2_unica',
               'vinculo_identidad_acceso_rrhh_prueba_v2_unica',
               'alcance_acceso_rrhh_prueba_v2_unica'
           ]::name[])
           AND restriccion.conrelid IN (
               'vec_contratacion_temporal.'
               'registro_acceso_rrhh'::pg_catalog.regclass,
               'vec_contratacion_temporal.'
               'vinculo_identidad_acceso_rrhh_v2'::pg_catalog.regclass,
               'vec_contratacion_temporal.'
               'alcance_acceso_rrhh'::pg_catalog.regclass
           )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada de prueba durable RRHH incompleta';
    END IF;
END
$retirada$;
COMMIT;
\unset ct000043_aplicar_acl
\unset ct000043_avanzar_barrera
