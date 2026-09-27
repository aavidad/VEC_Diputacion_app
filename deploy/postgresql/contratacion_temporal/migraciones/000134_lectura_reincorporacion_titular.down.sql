\set ON_ERROR_STOP on
-- CT134 DOWN solo es válido antes de producir historia de lecturas.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000134',0));
LOCK TABLE vec_contratacion_temporal.lectura_reincorporacion_titular_v1 IN ACCESS EXCLUSIVE MODE;
DO $pre$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR to_regprocedure('vec_contratacion_temporal.leer_antecedente_reincorporacion_titular_atestada_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.lectura_reincorporacion_titular_v1) THEN
  RAISE EXCEPTION 'CT134 DOWN: historia o estado incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
DROP FUNCTION vec_contratacion_temporal.leer_antecedente_reincorporacion_titular_atestada_v1(
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_contratacion_temporal.lectura_reincorporacion_titular_v1;
COMMIT;
