\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_convocatorias:migracion:000007',0));
LOCK TABLE vec_bolsa_convocatorias.lectura_version_v3 IN ACCESS EXCLUSIVE MODE;
DO $historia$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.lectura_version_v3) THEN
  RAISE EXCEPTION 'Bolsa007: no se puede retirar una lectura con historia' USING ERRCODE='55000';
 END IF;
END $historia$;
DROP FUNCTION vec_bolsa_convocatorias.obtener_version_exacta_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_bolsa_convocatorias.lectura_version_v3;
-- Se conservan ACL de V1, fuentes y funciones previas, y el consumo AD3.
COMMIT;
