\set ON_ERROR_STOP on
-- Solo revierte un ensayo vacío. La historia de consultas no se destruye.
BEGIN;
SET LOCAL row_security=off;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000014:competencias-asignacion-dietas:v1',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
 THEN RAISE EXCEPTION 'Personal 000014 DOWN requiere migrador superusuario' USING ERRCODE='42501'; END IF;
END $pre$;
LOCK TABLE vec_personal.recibo_competencias_asignacion_dietas,
 vec_personal.evidencia_competencias_asignacion_dietas IN ACCESS EXCLUSIVE MODE;
DO $historia$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_personal.recibo_competencias_asignacion_dietas)
    OR EXISTS(SELECT 1 FROM vec_personal.evidencia_competencias_asignacion_dietas)
 THEN RAISE EXCEPTION 'Personal 000014 DOWN protege recibos e historia'
  USING ERRCODE='55000'; END IF;
END $historia$;
REVOKE EXECUTE ON FUNCTION vec_personal.consultar_competencias_asignacion_dietas_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_personal_d7_ejecutor,vec_personal_ejecutor;
SET LOCAL ROLE vec_personal_propietario;
DROP FUNCTION vec_personal.consultar_competencias_asignacion_dietas_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP POLICY competencias_actor_candidato ON vec_personal.asignacion_dietas;
DROP TABLE vec_personal.evidencia_competencias_asignacion_dietas RESTRICT;
DROP TABLE vec_personal.recibo_competencias_asignacion_dietas RESTRICT;
COMMIT;
