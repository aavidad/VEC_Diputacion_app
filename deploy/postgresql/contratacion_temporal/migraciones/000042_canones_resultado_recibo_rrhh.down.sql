\set ON_ERROR_STOP on
\set ct000042_aplicar_acl false
-- CT-000042: reversión atómica; reutiliza la misma huella literal del UP.
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
       ) OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .control_migracion_consultas_rrhh
            WHERE control AND version_esquema = 6
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para revertir cánones RRHH';
    END IF;
END
$prevalidacion$;

-- La huella compartida espera las barreras previas 21/5 y las devuelve a
-- 22/6. Todo ocurre dentro de esta transacción y no normaliza ACL en DOWN.
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 21
 WHERE control AND version_esquema = 22;
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 5
 WHERE control AND version_esquema = 6;
\ir 000042_componentes/090_acl_catalogo_y_barrera.sql

DROP FUNCTION
vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
    timestamptz,
    vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1
) RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.canon_contenido_cuadro_rrhh_v1(
    timestamptz,
    vec_contratacion_temporal.resumen_publicacion_rrhh_v1[],
    boolean, bytea
) RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.canon_recibo_lectura_rrhh_v2(
    vec_contratacion_temporal.evidencia_recibo_lectura_rrhh_v2
) RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.canon_resultado_consulta_rrhh_puro_v1(
    vec_contratacion_temporal.evidencia_resultado_rrhh_v1
) RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.huella_material_consumo_rrhh_v3(
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.canon_resumen_publicacion_rrhh_v1(
    vec_contratacion_temporal.resumen_publicacion_rrhh_v1
) RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.encuadrar_valor_rrhh_v1(bytea)
RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(timestamptz)
RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.decodificar_texto_utf8_rrhh_v1(bytea)
RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(text)
RESTRICT;

DROP TYPE
vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1
RESTRICT;
DROP TYPE
vec_contratacion_temporal.evidencia_recibo_lectura_rrhh_v2
RESTRICT;
DROP TYPE vec_contratacion_temporal.cobertura_operativa_rrhh_v1
RESTRICT;
DROP TYPE vec_contratacion_temporal.analisis_operativo_rrhh_v1
RESTRICT;
DROP TYPE vec_contratacion_temporal.asignacion_operativa_rrhh_v1
RESTRICT;
DROP TYPE vec_contratacion_temporal.solicitud_operativa_rrhh_v1
RESTRICT;
DROP TYPE vec_contratacion_temporal.hito_expediente_rrhh_v1
RESTRICT;
DROP TYPE vec_contratacion_temporal.comprobacion_operativa_rrhh_v1
RESTRICT;
DROP TYPE vec_contratacion_temporal.resumen_publicacion_rrhh_v1
RESTRICT;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 21
 WHERE control AND version_esquema = 22;
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 5
 WHERE control AND version_esquema = 6;

DO $retirada$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 21
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 5
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc funcion
         WHERE funcion.pronamespace =
               'vec_contratacion_temporal'::pg_catalog.regnamespace
           AND funcion.proname = ANY(ARRAY[
               'codificar_texto_utf8_rrhh_v1',
               'decodificar_texto_utf8_rrhh_v1',
               'texto_instante_canonico_rrhh_v1',
               'encuadrar_valor_rrhh_v1',
               'canon_resumen_publicacion_rrhh_v1',
               'canon_resultado_consulta_rrhh_puro_v1',
               'canon_recibo_lectura_rrhh_v2',
               'huella_material_consumo_rrhh_v3',
               'canon_contenido_cuadro_rrhh_v1',
               'canon_contenido_detalle_rrhh_v1'
           ]::name[])
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
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada de cánones RRHH incompleta';
    END IF;
END
$retirada$;
COMMIT;
\unset ct000042_aplicar_acl
