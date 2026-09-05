\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000057_consulta_justificante_respuesta_recibida',0));
-- Solo se retira la nueva entrada. No hay tablas nuevas: se conservan CT56,
-- todos los recibos y la auditoría/consumos de acceso del propietario V3.
-- Sin CASCADE: una dependencia posterior impide una retirada destructiva.
DROP FUNCTION vec_contratacion_temporal.consultar_justificante_respuesta_recibida_rrhh_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
COMMIT;
