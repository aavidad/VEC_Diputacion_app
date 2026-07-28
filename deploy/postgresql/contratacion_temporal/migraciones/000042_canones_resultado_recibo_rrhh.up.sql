\set ON_ERROR_STOP on
\set ct000042_aplicar_acl true
-- CT-000042 se ejecuta solo con psql 18.4; los includes son una transacción.
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
 WHERE control AND version_esquema = 21
 FOR UPDATE;
SELECT control
  FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
 WHERE control AND version_esquema = 5
 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF pg_catalog.current_setting('server_version_num') <> '180004'
       OR pg_catalog.getdatabaseencoding() IS DISTINCT FROM 'UTF8'
       OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 21
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 5
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type tipo
         WHERE tipo.typnamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND tipo.typname = ANY(ARRAY[
               'resumen_publicacion_rrhh_v1',
               'solicitud_operativa_rrhh_v1',
               'analisis_operativo_rrhh_v1',
               'comprobacion_operativa_rrhh_v1',
               'cobertura_operativa_rrhh_v1',
               'asignacion_operativa_rrhh_v1',
               'hito_expediente_rrhh_v1',
               'entrada_detalle_expediente_rrhh_v1',
               'evidencia_recibo_lectura_rrhh_v2',
               '_resumen_publicacion_rrhh_v1',
               '_solicitud_operativa_rrhh_v1',
               '_analisis_operativo_rrhh_v1',
               '_comprobacion_operativa_rrhh_v1',
               '_cobertura_operativa_rrhh_v1',
               '_asignacion_operativa_rrhh_v1',
               '_hito_expediente_rrhh_v1',
               '_entrada_detalle_expediente_rrhh_v1',
               '_evidencia_recibo_lectura_rrhh_v2'
           ]::name[])
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc funcion
         WHERE funcion.pronamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND funcion.proname = ANY(ARRAY[
               'codificar_texto_utf8_rrhh_v1',
               'decodificar_texto_utf8_rrhh_v1',
               'encuadrar_valor_rrhh_v1',
               'texto_instante_canonico_rrhh_v1',
               'canon_resumen_publicacion_rrhh_v1',
               'canon_resultado_consulta_rrhh_puro_v1',
               'canon_contenido_cuadro_rrhh_v1',
               'canon_contenido_detalle_rrhh_v1',
               'huella_material_consumo_rrhh_v3',
               'canon_recibo_lectura_rrhh_v2'
           ]::name[])
    ) OR pg_catalog.to_regtype(
        'vec_contratacion_temporal.evidencia_resultado_rrhh_v1'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar cánones RRHH';
    END IF;
END
$prevalidacion$;

\ir 000042_componentes/010_tipos_nominales.sql
\ir 000042_componentes/015_codec_utf8_inmutable.sql
\ir 000042_componentes/016_instante_canonico.sql
\ir 000042_componentes/020_auxiliares_y_canones_base.sql
\ir 000042_componentes/030_canon_detalle.sql
\ir 000042_componentes/090_acl_catalogo_y_barrera.sql
COMMIT;
\unset ct000042_aplicar_acl
