\set ON_ERROR_STOP on
\set ct000044_aplicar_acl true
\set ct000044_avanzar_barrera true
-- CT-000044: motor privado y atómico de consultas RRHH. Sus componentes
-- forman una sola migración y nunca se aplican por separado en producción.
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
 WHERE control AND version_esquema = 23
 FOR UPDATE;
SELECT control
  FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
 WHERE control AND version_esquema = 7
 FOR UPDATE;

DO $prevalidacion$
DECLARE
    v_esquema oid :=
        'vec_contratacion_temporal'::pg_catalog.regnamespace;
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
           'vec_contratacion_temporal.'
           'control_causal_familia_cursor_rrhh'
       ) IS NOT NULL
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_type tipo
            WHERE tipo.typnamespace = v_esquema
              AND tipo.typname = ANY(ARRAY[
                  'material_autorizacion_consulta_rrhh_v3',
                  'estado_cursor_entrada_cuadro_rrhh_v1',
                  'materializacion_cuadro_rrhh_v1',
                  'materializacion_detalle_rrhh_v1',
                  'salida_cursor_cuadro_rrhh_v1',
                  'resultado_motor_cuadro_rrhh_v1',
                  'resultado_motor_detalle_rrhh_v1',
                  '_material_autorizacion_consulta_rrhh_v3',
                  '_estado_cursor_entrada_cuadro_rrhh_v1',
                  '_materializacion_cuadro_rrhh_v1',
                  '_materializacion_detalle_rrhh_v1',
                  '_salida_cursor_cuadro_rrhh_v1',
                  '_resultado_motor_cuadro_rrhh_v1',
                  '_resultado_motor_detalle_rrhh_v1',
                  'control_causal_familia_cursor_rrhh',
                  '_control_causal_familia_cursor_rrhh'
              ]::name[])
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc funcion
            WHERE funcion.pronamespace = v_esquema
              AND funcion.proname = ANY(ARRAY[
                  'acreditar_contexto_motor_consultas_rrhh_v1',
                  'consumir_autorizacion_motor_consultas_rrhh_v1',
                  'materializar_detalle_rrhh_v1',
                  'materializar_cuadro_rrhh_v1',
                  'validar_avance_control_causal_cursor_rrhh_v1',
                  'avanzar_control_causal_revocacion_cursor_rrhh_v1',
                  'resolver_estado_cursor_cuadro_rrhh_v1',
                  'preparar_salida_cursor_cuadro_rrhh_v1',
                  'aplicar_efectos_cursor_cuadro_rrhh_v1',
                  'motor_consultar_cuadro_rrhh_v1',
                  'motor_consultar_detalle_rrhh_v1'
              ]::name[])
       )
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.'
           'cerrar_prueba_resultado_recibo_rrhh_v2('
           || 'vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2,'
           || 'vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2,'
           || 'vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,'
           || 'bytea,bytea,bytea,bytea,numeric,numeric,'
           || 'bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc funcion
            WHERE funcion.oid =
                  'vec_contratacion_temporal.'
                  'cerrar_prueba_resultado_recibo_rrhh_v2('
                  'vec_contratacion_temporal.'
                  'contexto_cierre_prueba_rrhh_v2,'
                  'vec_contratacion_temporal.'
                  'contenido_cierre_prueba_rrhh_v2,'
                  'vec_contratacion_temporal.'
                  'evidencia_consumo_nuevo_rrhh_v3,'
                  'bytea,bytea,bytea,bytea,numeric,numeric,'
                  'bytea,bytea,bytea,bytea)'::pg_catalog.regprocedure
              AND pg_catalog.strpos(
                      funcion.prosrc, 'v_version_expediente IS NULL'
                  ) > 0
              AND pg_catalog.strpos(
                      funcion.prosrc, 'version_observada <> 0'
                  ) > 0
       )
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.'
           'registrar_y_consumir_consulta_cuadro_rrhh_v3_atestada('
           || 'bytea,bytea,bytea,bytea,numeric,numeric,'
           || 'bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.'
           'registrar_y_consumir_consulta_detalle_rrhh_v3_atestada('
           || 'bytea,bytea,bytea,bytea,numeric,numeric,'
           || 'bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.control_publicacion_rrhh'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.familia_cursor_cuadro_rrhh'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.cursor_cuadro_rrhh'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.consumo_cursor_cuadro_rrhh'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.revocacion_familia_cursor_rrhh'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para motor de consultas RRHH';
    END IF;
END
$prevalidacion$;

\ir 000044_componentes/010_tipos_resultado.sql
\ir 000044_componentes/020_guardas_y_contexto.sql
\ir 000044_componentes/030_materializacion_detalle.sql
\ir 000044_componentes/040_materializacion_cuadro.sql
\ir 000044_componentes/050_control_causal_y_cursores.sql
\ir 000044_componentes/055_efectos_cursor.sql
\ir 000044_componentes/060_motor_atomico_y_efectos.sql
\ir 000044_componentes/090_acl_catalogo_y_barrera.sql
\ir 000044_componentes/095_avance_barreras.sql
COMMIT;
\unset ct000044_aplicar_acl
\unset ct000044_avanzar_barrera
