\set ON_ERROR_STOP on
\set ct000043_aplicar_acl true
\set ct000043_avanzar_barrera true
-- CT-000043 se ejecuta exclusivamente con psql 18.4; los componentes forman
-- una única transacción y no son migraciones independientes.
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
 WHERE control AND version_esquema = 22
 FOR UPDATE;
SELECT control
  FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
 WHERE control AND version_esquema = 6
 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF pg_catalog.current_setting('server_version_num') <> '180004'
       OR pg_catalog.getdatabaseencoding() IS DISTINCT FROM 'UTF8'
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.control_migracion_cobertura_o4
            WHERE control AND version_esquema = 22
       )
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
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
                  '_resultado_cierre_prueba_rrhh_v2'
              ]::name[])
       )
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.cerrar_prueba_resultado_recibo_rrhh_v2('
           || 'vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2,'
           || 'vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2,'
           || 'vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,'
           || 'bytea,bytea,bytea,bytea,numeric,numeric,'
           || 'bytea,bytea,bytea,bytea)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.canon_recibo_lectura_rrhh_v2('
           || 'vec_contratacion_temporal.evidencia_recibo_lectura_rrhh_v2)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.'
           || 'revalidar_evidencia_consumo_consulta_cuadro_rrhh_v3_atestada('
           || 'bytea,bytea,bytea,bytea,numeric,numeric,'
           || 'bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.'
           || 'revalidar_evidencia_consumo_consulta_detalle_rrhh_v3_atestada('
           || 'bytea,bytea,bytea,bytea,numeric,numeric,'
           || 'bytea,bytea,bytea,bytea)'
       ) IS NULL
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
            WHERE restriccion.conrelid IN (
                  'vec_contratacion_temporal.'
                  'registro_acceso_rrhh'::pg_catalog.regclass,
                  'vec_contratacion_temporal.'
                  'vinculo_identidad_acceso_rrhh_v2'::pg_catalog.regclass,
                  'vec_contratacion_temporal.'
                  'alcance_acceso_rrhh'::pg_catalog.regclass
              )
              AND restriccion.conname LIKE '%prueba%v2%'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para prueba durable RRHH';
    END IF;
END
$prevalidacion$;

\ir 000043_componentes/010_tipos_cierre.sql
\ir 000043_componentes/020_relaciones_y_prueba.sql
\ir 000043_componentes/030_primitiva_cierre.sql
\ir 000043_componentes/085_guardia_columnas_padre.sql
\ir 000043_componentes/090_acl_catalogo_y_barrera.sql
\ir 000043_componentes/095_avance_barreras.sql
COMMIT;
\unset ct000043_aplicar_acl
\unset ct000043_avanzar_barrera
