\set ON_ERROR_STOP on
-- Sólo para un ensayo sin uso. Nunca revertir recibos, versiones ni consumidores.
BEGIN;
SET LOCAL row_security=off;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000012:asignacion-dietas:v1',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
 THEN RAISE EXCEPTION 'Personal 000012 DOWN requiere migrador superusuario' USING ERRCODE='42501'; END IF;
END $pre$;
LOCK TABLE vec_personal.asignacion_dietas,
 vec_personal.recibo_asignacion_dietas,
 vec_personal.evidencia_asignacion_dietas IN ACCESS EXCLUSIVE MODE;
DO $historia$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_personal_d7_ejecutor'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
    OR EXISTS(SELECT 1 FROM pg_auth_members m
       WHERE m.member=to_regrole('vec_personal_d7_ejecutor')
          OR m.roleid=to_regrole('vec_personal_d7_ejecutor'))
    OR EXISTS(SELECT 1 FROM vec_personal.recibo_asignacion_dietas)
    OR EXISTS(SELECT 1 FROM vec_personal.evidencia_asignacion_dietas)
    OR EXISTS(SELECT 1 FROM vec_personal.asignacion_dietas WHERE version>1)
    OR to_regclass('vec_personal.auditoria_frontera_asignacion_dietas') IS NOT NULL
    OR to_regprocedure('vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text)') IS NOT NULL
    OR to_regclass('vec_personal.recibo_competencias_asignacion_dietas') IS NOT NULL
    OR to_regprocedure('vec_personal.consultar_competencias_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regclass('vec_personal.solicitud_rectificacion_dietas') IS NOT NULL
    OR EXISTS(
      SELECT 1 FROM pg_proc p JOIN pg_language l ON l.oid=p.prolang
      WHERE p.pronamespace='vec_personal'::regnamespace
        AND l.lanname IN ('plpgsql','sql')
        AND p.proname NOT IN (
          'revalidar_asignacion_dietas_v1',
          'ejecutar_asignacion_dietas_interna_v1',
          'registrar_asignacion_dietas_inicial_v1',
          'consultar_asignacion_dietas_v1',
          'corregir_asignacion_dietas_v1',
          'corregir_grupo_dieta_v1')
        AND p.prosrc LIKE '%asignacion_dietas%')
    OR EXISTS(
      SELECT 1 FROM pg_proc p
      JOIN pg_language l ON l.oid=p.prolang
      WHERE l.lanname IN ('plpgsql','sql')
        AND p.pronamespace NOT IN (
          'vec_personal'::regnamespace,
          'vec_autorizacion_atestada_v3'::regnamespace)
        AND p.prosrc LIKE '%revalidar_asignacion_dietas_v1%')
 THEN RAISE EXCEPTION 'Personal 000012 DOWN protege historia o consumidor'
      USING ERRCODE='55000'; END IF;
END $historia$;
REVOKE EXECUTE ON FUNCTION
 vec_personal.registrar_asignacion_dietas_inicial_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.corregir_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.corregir_grupo_dieta_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date)
 FROM PUBLIC,vec_dietas_ejecutor,vec_dietas_propietario,
  vec_personal_ejecutor,vec_personal_d7_ejecutor;
SET LOCAL ROLE vec_personal_propietario;
DROP FUNCTION vec_personal.registrar_asignacion_dietas_inicial_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_personal.corregir_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_personal.corregir_grupo_dieta_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_personal.ejecutar_asignacion_dietas_interna_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date) RESTRICT;
DROP TABLE vec_personal.evidencia_asignacion_dietas RESTRICT;
DROP TABLE vec_personal.recibo_asignacion_dietas RESTRICT;
RESET ROLE;
REVOKE USAGE ON SCHEMA vec_personal FROM vec_personal_d7_ejecutor;
DROP ROLE vec_personal_d7_ejecutor;
COMMIT;
