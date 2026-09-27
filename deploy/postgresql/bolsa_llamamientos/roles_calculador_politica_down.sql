\set ON_ERROR_STOP on
-- DBA. Solo después de Bolsa 000051 DOWN y sin LOGIN nominal miembro.
BEGIN;
SET LOCAL search_path=pg_catalog;
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:rol:calculador_politica',0));
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_calculador_politica'
                   AND NOT rolcanlogin AND NOT rolbypassrls)
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_politica_ofertas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR EXISTS(SELECT 1 FROM pg_auth_members WHERE roleid='vec_bolsa_llamamientos_calculador_politica'::regrole)
 THEN RAISE EXCEPTION 'rol calculador de política: historia, consumidor o LOGIN presente' USING ERRCODE='55000'; END IF;
END $pre$;
DO $desconectar$ BEGIN
 EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM vec_bolsa_llamamientos_calculador_politica',current_database());
END $desconectar$;
DROP ROLE vec_bolsa_llamamientos_calculador_politica;
COMMIT;
