\set ON_ERROR_STOP on
-- Sólo para PostgreSQL desechable sin decisiones de circuito. Nunca en una
-- base con historia: una decisión registrada impide el DOWN.
BEGIN;
RESET ROLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000008:revision-circuito:v1',0));
DO $preimagen_dba$
BEGIN
 IF current_user IS DISTINCT FROM session_user
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regprocedure('vec_dietas.cotejar_efecto_circuito_v2(text,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_dietas.decidir_comision_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_dietas.consultar_documento_circuito_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'Dietas 000008 DOWN requiere DBA y preimagen completa' USING ERRCODE='55000'; END IF;
END $preimagen_dba$;
-- El DBA ve todas las filas pese a FORCE RLS contextual.
LOCK TABLE vec_dietas.recibo_operacion_comision IN ACCESS EXCLUSIVE MODE;
DO $historia_dba$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_dietas.recibo_operacion_comision WHERE operacion LIKE 'circuito\_%')
 THEN RAISE EXCEPTION 'Dietas 000008 DOWN protege decisiones del circuito' USING ERRCODE='55000'; END IF;
END $historia_dba$;
SET LOCAL ROLE vec_dietas_propietario;
GRANT EXECUTE ON FUNCTION
 vec_dietas.decidir_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_dietas.preleer_circuito_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_dietas_ejecutor;
DROP FUNCTION vec_dietas.decidir_comision_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_dietas.consultar_documento_circuito_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_dietas.cotejar_efecto_circuito_v2(text,bytea,bytea,bytea) RESTRICT;
COMMIT;
