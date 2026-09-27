\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000048', 0));
DO $precondicion$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'consulta de auditoria Bolsa ausente' USING ERRCODE='55000';
 END IF;
END $precondicion$;
DROP FUNCTION vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
COMMIT;
