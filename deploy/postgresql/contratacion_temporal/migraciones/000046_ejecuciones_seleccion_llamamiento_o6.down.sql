\set ON_ERROR_STOP on
-- CT-LITE-O6-03: retirada solo sin una ejecución reservada o terminal.
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
                  'resolver_terminal_seleccion_llamamiento_o6_v1',
                  'reservar_seleccion_llamamiento_o6_v1',
                  'abrir_ventana_seleccion_llamamiento_o6_v1',
                  'marcar_indeterminada_seleccion_llamamiento_o6_v1',
                  'liberar_seleccion_llamamiento_o6_v1',
                  'confirmar_seleccion_llamamiento_o6_v1',
                  'consultar_seleccion_llamamiento_o6_v1'
              ]::name[])
       ) <> 7 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada O6 rechazada por historia o deriva';
    END IF;
END
$puerta$;

REVOKE EXECUTE ON FUNCTION
    vec_contratacion_temporal.resolver_terminal_seleccion_llamamiento_o6_v1(uuid),
    vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(uuid, text, jsonb),
    vec_contratacion_temporal.abrir_ventana_seleccion_llamamiento_o6_v1(uuid, text, text, jsonb, text),
    vec_contratacion_temporal.marcar_indeterminada_seleccion_llamamiento_o6_v1(uuid, text, text, jsonb, text),
    vec_contratacion_temporal.liberar_seleccion_llamamiento_o6_v1(uuid, text, text, jsonb),
    vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(uuid, text, text, jsonb, jsonb, text),
    vec_contratacion_temporal.consultar_seleccion_llamamiento_o6_v1(uuid, text, jsonb)
    FROM vec_contratacion_temporal_ejecutor;

DROP FUNCTION vec_contratacion_temporal.consultar_seleccion_llamamiento_o6_v1(uuid, text, jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(uuid, text, text, jsonb, jsonb, text) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.liberar_seleccion_llamamiento_o6_v1(uuid, text, text, jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.marcar_indeterminada_seleccion_llamamiento_o6_v1(uuid, text, text, jsonb, text) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.abrir_ventana_seleccion_llamamiento_o6_v1(uuid, text, text, jsonb, text) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(uuid, text, jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.resolver_terminal_seleccion_llamamiento_o6_v1(uuid) RESTRICT;
DROP TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 RESTRICT;

COMMIT;
