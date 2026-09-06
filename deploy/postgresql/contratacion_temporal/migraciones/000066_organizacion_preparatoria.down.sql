\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $conservar$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.organizacion_cambio)
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.organizacion_catalogo_revision WHERE revision>1) THEN
        RAISE EXCEPTION 'conservar organización con cambios registrados' USING ERRCODE='55000';
    END IF;
END
$conservar$;
DROP FUNCTION vec_contratacion_temporal.registrar_cambio_organizacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.obtener_cambio_organizacion_v1(uuid);
DROP FUNCTION vec_contratacion_temporal.obtener_catalogo_organizacion_v1(integer);
DROP FUNCTION vec_contratacion_temporal.listar_catalogos_organizacion_v1();
DROP FUNCTION vec_contratacion_temporal.inicializar_organizacion_preparatoria_v1(text);
DROP TABLE vec_contratacion_temporal.organizacion_outbox;
DROP TABLE vec_contratacion_temporal.organizacion_cambio;
DROP TABLE vec_contratacion_temporal.organizacion_catalogo_revision;
COMMIT;
