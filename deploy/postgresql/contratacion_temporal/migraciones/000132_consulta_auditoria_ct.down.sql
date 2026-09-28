\set ON_ERROR_STOP on
-- Solo para una instalación sin historia. No ejecutar sobre bases conservadas.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
DROP FUNCTION vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(
    text,text,text,timestamptz,timestamptz,integer,timestamptz,text,text,text,text,
    bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
COMMIT;
