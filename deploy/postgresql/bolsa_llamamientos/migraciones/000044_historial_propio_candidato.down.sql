\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000044',0));
-- No altera historia: la función no crea tablas. Las autorizaciones consumidas
-- permanecen en AD3 y su DOWN rechaza historia de esta audiencia.
DROP FUNCTION vec_bolsa_llamamientos.consultar_historial_mi_bolsa_v1(text,timestamptz,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
COMMIT;
