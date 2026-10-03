\set ON_ERROR_STOP on
-- Clon desechable, tras la fixture principal. Prueba el primer plan con pg_temp.
BEGIN;
CREATE TEMP TABLE version_rol (valor integer);
CREATE DOMAIN pg_temp.text AS pg_catalog.text CHECK (false);
CREATE DOMAIN pg_temp.jsonb AS pg_catalog.jsonb CHECK (false);
CREATE DOMAIN pg_temp.timestamptz AS pg_catalog.timestamptz CHECK (false);
CREATE DOMAIN pg_temp.numeric AS pg_catalog.numeric CHECK (false);
CREATE DOMAIN pg_temp.bigint AS pg_catalog.int8 CHECK (false);
CREATE TYPE pg_temp.record AS (valor pg_catalog.int4);
COMMIT;
SET SESSION AUTHORIZATION vec_externo_v3_fuente_autorizacion_desarrollo;
BEGIN READ ONLY;
SELECT 1 / ((vec_autorizacion.obtener_preimagen_candidato_externo_ro_v1(
    'candidato_bolsa_historial_propio_desarrollo','rol:candidato_bolsa_historial_propio_desarrollo:v1',
    'principal:aut26:sintetico','perfil:aut26:activa') #>> '{asignacion,estado_observacion}') = 'activa')::pg_catalog.int4;
SELECT 1 / ((vec_autorizacion.obtener_checkpoint_motivos_candidato_externo_ro_v1()
    #>> '{checkpoint,coherente}') = 'true')::pg_catalog.int4;
COMMIT;
RESET SESSION AUTHORIZATION;
SELECT 'AUT26_TIPOS_TEMPORALES_OK' AS resultado;
