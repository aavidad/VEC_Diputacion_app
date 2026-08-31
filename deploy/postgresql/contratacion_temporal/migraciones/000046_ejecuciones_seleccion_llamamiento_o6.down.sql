\set ON_ERROR_STOP on
-- CT-LITE-O6-REM-02: retirada solo sin una ejecución reservada o terminal.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o6:ejecuciones-seleccion-llamamiento:v1', 0
));

DO $puerta$
BEGIN
    IF pg_catalog.current_setting('server_version_num') <> '180004'
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'
       ) IS NULL
       OR EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
       )
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_proc funcion
            WHERE funcion.pronamespace =
                  'vec_contratacion_temporal'::pg_catalog.regnamespace
              AND funcion.proname = ANY(ARRAY[
                  'resolver_terminal_autorizado_seleccion_llamamiento_o6_v2',
                  'reservar_seleccion_llamamiento_o6_v1',
                  'abrir_ventana_seleccion_llamamiento_o6_v1',
                  'marcar_indeterminada_seleccion_llamamiento_o6_v1',
                  'liberar_seleccion_llamamiento_o6_v1',
                  'confirmar_seleccion_llamamiento_o6_v1',
                  'consultar_seleccion_llamamiento_o6_v1',
                  'campo_canonico_seleccion_llamamiento_o6_v1',
                  'entero_json_seleccion_llamamiento_o6_v1',
                  'referencia_json_seleccion_llamamiento_o6_v1',
                  'huella_solicitud_seleccion_llamamiento_o6_v1',
                  'solicitud_json_seleccion_llamamiento_o6_v1',
                  'solicitud_desde_texto_seleccion_llamamiento_o6_v1',
                  'recibo_json_seleccion_llamamiento_o6_v1',
                  'recibo_desde_texto_seleccion_llamamiento_o6_v1',
                  'artefacto_json_seleccion_llamamiento_o6_v1',
                  'referencia_material_seleccion_llamamiento_o6_v1',
                  'contexto_material_seleccion_llamamiento_o6_v1',
                  'huellas_materiales_seleccion_llamamiento_o6_v1',
                  'materiales_ligados_seleccion_llamamiento_o6_v1',
                  'confirmacion_canonica_seleccion_llamamiento_o6_v1',
                  'nuevo_token_fencing_seleccion_llamamiento_o6_v2'
              ]::name[])
       ) <> 22 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada O6 rechazada por historia o deriva';
    END IF;
END
$puerta$;

REVOKE EXECUTE ON FUNCTION
    vec_contratacion_temporal.resolver_terminal_autorizado_seleccion_llamamiento_o6_v2(uuid, text),
    vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(uuid, text, text),
    vec_contratacion_temporal.abrir_ventana_seleccion_llamamiento_o6_v1(uuid, text, text, text, text),
    vec_contratacion_temporal.marcar_indeterminada_seleccion_llamamiento_o6_v1(uuid, text, text, text, text),
    vec_contratacion_temporal.liberar_seleccion_llamamiento_o6_v1(uuid, text, text, text),
    vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(uuid, text, text, text, text, text),
    vec_contratacion_temporal.consultar_seleccion_llamamiento_o6_v1(uuid, text, text)
    FROM vec_contratacion_temporal_ejecutor;

DROP FUNCTION vec_contratacion_temporal.consultar_seleccion_llamamiento_o6_v1(uuid, text, text) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(uuid, text, text, text, text, text) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.liberar_seleccion_llamamiento_o6_v1(uuid, text, text, text) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.marcar_indeterminada_seleccion_llamamiento_o6_v1(uuid, text, text, text, text) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.abrir_ventana_seleccion_llamamiento_o6_v1(uuid, text, text, text, text) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(uuid, text, text) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.resolver_terminal_autorizado_seleccion_llamamiento_o6_v2(uuid, text) RESTRICT;
DROP TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 RESTRICT;
DROP FUNCTION vec_contratacion_temporal.nuevo_token_fencing_seleccion_llamamiento_o6_v2() RESTRICT;
DROP FUNCTION vec_contratacion_temporal.confirmacion_canonica_seleccion_llamamiento_o6_v1(jsonb, text, jsonb, text) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.materiales_ligados_seleccion_llamamiento_o6_v1(jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.huellas_materiales_seleccion_llamamiento_o6_v1(jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.contexto_material_seleccion_llamamiento_o6_v1(jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.referencia_material_seleccion_llamamiento_o6_v1(text, jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(jsonb, boolean) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.recibo_desde_texto_seleccion_llamamiento_o6_v1(text) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.solicitud_desde_texto_seleccion_llamamiento_o6_v1(text) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.solicitud_json_seleccion_llamamiento_o6_v1(jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.huella_solicitud_seleccion_llamamiento_o6_v1(jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.entero_json_seleccion_llamamiento_o6_v1(jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.campo_canonico_seleccion_llamamiento_o6_v1(text, text) RESTRICT;

COMMIT;
