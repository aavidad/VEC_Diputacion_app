\set ON_ERROR_STOP on
-- No se retira una historia de consultas propias. DOWN sólo es posible vacío,
-- bajo inspección total RLS y sin consumidor dependiente.
BEGIN;
SET LOCAL row_security=off;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000008:consulta-relacion-propia-dietas:v1',0));
DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper) THEN RAISE EXCEPTION 'Personal 000008 DOWN requiere migrador superusuario' USING ERRCODE='42501'; END IF; END $$;
LOCK TABLE vec_personal.recibo_consulta_relacion_propia_dietas,vec_personal.evidencia_consulta_relacion_propia_dietas IN ACCESS EXCLUSIVE MODE;
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM vec_personal.recibo_consulta_relacion_propia_dietas) OR EXISTS(SELECT 1 FROM vec_personal.evidencia_consulta_relacion_propia_dietas)
    OR EXISTS(SELECT 1 FROM pg_proc p JOIN pg_language l ON l.oid=p.prolang WHERE l.lanname='plpgsql' AND p.oid<>'vec_personal.consultar_relaciones_propias_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure AND p.prosrc LIKE '%consultar_relaciones_propias_dietas_v1%') THEN
   RAISE EXCEPTION 'Personal 000008 DOWN protege historia o consumidor' USING ERRCODE='55000';
 END IF;
END $$;
REVOKE EXECUTE ON FUNCTION vec_personal.consultar_relaciones_propias_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_dietas_ejecutor;
REVOKE USAGE ON SCHEMA vec_personal FROM vec_dietas_ejecutor;
SET LOCAL ROLE vec_personal_migrador;
SET LOCAL ROLE vec_personal_propietario;
DROP FUNCTION vec_personal.consultar_relaciones_propias_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP TABLE vec_personal.evidencia_consulta_relacion_propia_dietas RESTRICT;
DROP TABLE vec_personal.recibo_consulta_relacion_propia_dietas RESTRICT;
COMMIT;
