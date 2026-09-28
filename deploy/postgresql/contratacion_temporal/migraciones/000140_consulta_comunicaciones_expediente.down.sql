\set ON_ERROR_STOP on
-- Documental. No ejecutar sobre una base con historia de lecturas AD3-105.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000140',0));
DROP FUNCTION vec_contratacion_temporal.consultar_comunicaciones_expediente_rrhh_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.estado_respuesta_comunicacion_ct140(text);
COMMIT;
