\set ON_ERROR_STOP on
-- Retirar la API nueva no reabre la lectura sin autorización/auditoría.
-- No toca configuración ni auditoría T13 conservadas. No usar para restaurar
-- un runtime antiguo que todavía invoque el lector sin material V3.
BEGIN;
SET LOCAL ROLE vec_administracion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended(
    'vec_administracion.dependencias.configuracion_correo.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended(
    'vec_administracion:configuracion_correo:000002',0));
DROP FUNCTION vec_administracion.consultar_configuracion_correo_v2(
    bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
REVOKE ALL ON FUNCTION vec_administracion.leer_configuracion_correo_v1()
    FROM PUBLIC,vec_administracion_ejecutor,vec_administracion_migrador;
COMMIT;
