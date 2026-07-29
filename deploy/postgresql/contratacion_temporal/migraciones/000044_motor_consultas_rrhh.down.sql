\set ON_ERROR_STOP on
\set ct000044_aplicar_acl false
\set ct000044_avanzar_barrera false
-- CT-000044: reversión conservadora. No retira estado causal durable ni
-- acepta una estructura distinta de la huella aprobada.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_05:consultas_rrhh:migraciones', 0
));

-- La puerta no referencia estáticamente objetos que un DOWN previo retiró.
DO $puerta$
BEGIN
    IF pg_catalog.current_setting('server_version_num') <> '180004'
       OR pg_catalog.getdatabaseencoding() IS DISTINCT FROM 'UTF8'
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .control_migracion_cobertura_o4
            WHERE control AND version_esquema = 24
       )
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .control_migracion_consultas_rrhh
            WHERE control AND version_esquema = 8
       )
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.'
           'control_causal_familia_cursor_rrhh'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para revertir motor RRHH';
    END IF;
END
$puerta$;

LOCK TABLE
    vec_contratacion_temporal.control_migracion_cobertura_o4,
    vec_contratacion_temporal.control_migracion_consultas_rrhh,
    vec_contratacion_temporal.control_publicacion_rrhh,
    vec_contratacion_temporal.familia_cursor_cuadro_rrhh,
    vec_contratacion_temporal.cursor_cuadro_rrhh,
    vec_contratacion_temporal.consumo_cursor_cuadro_rrhh,
    vec_contratacion_temporal.revocacion_familia_cursor_rrhh,
    vec_contratacion_temporal.control_causal_familia_cursor_rrhh
IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
DECLARE
    v_esquema oid :=
        'vec_contratacion_temporal'::pg_catalog.regnamespace;
BEGIN
    IF NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .control_migracion_cobertura_o4
            WHERE control AND version_esquema = 24
       )
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .control_migracion_consultas_rrhh
            WHERE control AND version_esquema = 8
       )
       OR (
           SELECT pg_catalog.count(*)
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
       ) <> 11
       OR (
           SELECT pg_catalog.count(*)
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
       ) <> 16 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para revertir motor RRHH';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .control_causal_familia_cursor_rrhh
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'existe estado causal RRHH; reversión prohibida';
    END IF;
END
$prevalidacion$;

-- La misma huella cierra UP y DOWN antes de retirar un solo objeto.
\ir 000044_componentes/090_acl_catalogo_y_barrera.sql

DROP FUNCTION
vec_contratacion_temporal.motor_consultar_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
) RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.motor_consultar_detalle_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
) RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.aplicar_efectos_cursor_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,
    vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1,
    vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,
    bytea,
    vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2
) RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.preparar_salida_cursor_cuadro_rrhh_v1(
    vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,
    vec_contratacion_temporal.materializacion_cuadro_rrhh_v1
) RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.resolver_estado_cursor_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    text, text, numeric, text, text
) RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1
) RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.materializar_detalle_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    numeric
) RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.consumir_autorizacion_motor_consultas_rrhh_v1(
    text,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
) RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.acreditar_contexto_motor_consultas_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
) RESTRICT;

DROP TRIGGER avanzar_control_causal_revocacion_antes
ON vec_contratacion_temporal.revocacion_familia_cursor_rrhh;
DROP TABLE
vec_contratacion_temporal.control_causal_familia_cursor_rrhh
RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.avanzar_control_causal_revocacion_cursor_rrhh_v1()
RESTRICT;
DROP FUNCTION
vec_contratacion_temporal.validar_avance_control_causal_cursor_rrhh_v1()
RESTRICT;

DROP TYPE
vec_contratacion_temporal.resultado_motor_detalle_rrhh_v1
RESTRICT;
DROP TYPE
vec_contratacion_temporal.resultado_motor_cuadro_rrhh_v1
RESTRICT;
DROP TYPE
vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1
RESTRICT;
DROP TYPE
vec_contratacion_temporal.materializacion_detalle_rrhh_v1
RESTRICT;
DROP TYPE
vec_contratacion_temporal.materializacion_cuadro_rrhh_v1
RESTRICT;
DROP TYPE
vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1
RESTRICT;
DROP TYPE
vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
RESTRICT;

UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 7,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 8;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 23,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 24;

DO $retirada$
DECLARE
    v_esquema oid :=
        'vec_contratacion_temporal'::pg_catalog.regnamespace;
BEGIN
    IF NOT EXISTS (
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
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada incompleta del motor RRHH';
    END IF;
END
$retirada$;
COMMIT;
\unset ct000044_aplicar_acl
\unset ct000044_avanzar_barrera
