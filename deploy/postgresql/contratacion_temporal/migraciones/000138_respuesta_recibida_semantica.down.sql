\set ON_ERROR_STOP on
-- Solo retira el punto de entrada v2. Ninguna declaración ni recibo se borra.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000138',0));
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v2(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
DROP FUNCTION vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v2(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
COMMIT;
