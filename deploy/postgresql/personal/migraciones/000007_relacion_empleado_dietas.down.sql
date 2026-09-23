\set ON_ERROR_STOP on
BEGIN;
-- La retirada debe ver toda la historia aun con FORCE RLS. El canal de
-- migración es superusuario y la desactivación se hace antes de cambiar de rol.
SET LOCAL row_security=off;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000007:relacion-dietas:v1',0));
DO $$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper) THEN
   RAISE EXCEPTION 'Personal 000007 DOWN requiere autoridad de migración con inspección RLS completa' USING ERRCODE='42501';
 END IF;
END $$;
LOCK TABLE vec_personal.relacion_empleado_dietas IN ACCESS EXCLUSIVE MODE;
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM vec_personal.relacion_empleado_dietas)
    OR EXISTS (
      SELECT 1 FROM pg_proc p JOIN pg_language l ON l.oid=p.prolang
       WHERE l.lanname='plpgsql'
         AND p.oid<>'vec_personal.revalidar_relacion_dietas_v1(text,text,text,text,text,text,bigint,text,text,bigint,date)'::regprocedure
         AND p.prosrc LIKE '%revalidar_relacion_dietas_v1%'
    ) THEN
   RAISE EXCEPTION 'Personal 000007 DOWN protege relación/historia o consumidor Dietas' USING ERRCODE='55000';
 END IF;
END $$;
REVOKE EXECUTE ON FUNCTION vec_personal.resolver_relacion_dietas_v1(text,text,text,date),vec_personal.revalidar_relacion_dietas_v1(text,text,text,text,text,text,bigint,text,text,bigint,date) FROM PUBLIC,vec_dietas_propietario,vec_dietas_ejecutor;
REVOKE USAGE ON SCHEMA vec_personal FROM vec_dietas_propietario,vec_dietas_ejecutor;
SET LOCAL ROLE vec_personal_migrador;
SET LOCAL ROLE vec_personal_propietario;
DROP FUNCTION vec_personal.revalidar_relacion_dietas_v1(text,text,text,text,text,text,bigint,text,text,bigint,date) RESTRICT;
DROP FUNCTION vec_personal.resolver_relacion_dietas_v1(text,text,text,date) RESTRICT;
DROP TABLE vec_personal.relacion_empleado_dietas RESTRICT;
DROP FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1() RESTRICT;
COMMIT;
