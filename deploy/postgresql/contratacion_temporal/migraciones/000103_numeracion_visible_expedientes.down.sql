\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DROP FUNCTION vec_contratacion_temporal.siguiente_numero_visible_v1(integer);
DROP TABLE vec_contratacion_temporal.numeracion_expedientes;
COMMIT;
