\set ON_ERROR_STOP on
-- Sólo para PostgreSQL desechable vacío. Nunca revertir historia ni usarlo en
-- la base principal; la activación operativa de 000006 exige conservar 000007.
BEGIN;
RESET ROLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000007:circuito:v1',0));
DO $preimagen_dba$
BEGIN
 IF current_user IS DISTINCT FROM session_user
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regclass('vec_dietas.borrador_comision') IS NULL
    OR to_regclass('vec_dietas.comision_revision') IS NULL
    OR to_regclass('vec_dietas.recibo_operacion_comision') IS NULL
    OR to_regclass('vec_dietas.historia_operacion_comision') IS NULL
    OR to_regclass('vec_dietas.outbox_comision') IS NULL
    OR to_regclass('vec_dietas.cola_circuito_comision') IS NULL
    OR to_regprocedure('vec_dietas.abrir_cola_revision_comision_v1()') IS NULL
    OR to_regprocedure('vec_dietas.cotejar_efecto_circuito_v1(text,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_dietas.preleer_circuito_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_dietas.decidir_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_dietas.listar_bandeja_comisiones_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'Dietas 000007 DOWN requiere DBA y preimagen completa' USING ERRCODE='55000'; END IF;
END $preimagen_dba$;

-- ACCESS EXCLUSIVE bloquea nuevas altas/revisiones durante el inventario y
-- hasta el COMMIT. El DBA ve todas las filas pese a FORCE RLS contextual.
LOCK TABLE vec_dietas.borrador_comision,vec_dietas.comision_revision,
 vec_dietas.recibo_operacion_comision,vec_dietas.historia_operacion_comision,
 vec_dietas.outbox_comision,vec_dietas.cola_circuito_comision
 IN ACCESS EXCLUSIVE MODE;
DO $historia_dba$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_dietas.borrador_comision)
    OR EXISTS (SELECT 1 FROM vec_dietas.comision_revision)
    OR EXISTS (SELECT 1 FROM vec_dietas.recibo_operacion_comision)
    OR EXISTS (SELECT 1 FROM vec_dietas.historia_operacion_comision)
    OR EXISTS (SELECT 1 FROM vec_dietas.outbox_comision)
    OR EXISTS (SELECT 1 FROM vec_dietas.cola_circuito_comision)
 THEN RAISE EXCEPTION 'Dietas 000007 DOWN protege expediente, recibo, historia, outbox o cola' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_trigger t
   WHERE t.tgrelid='vec_dietas.comision_revision'::regclass
     AND t.tgname='abrir_cola_revision' AND t.tgenabled='O' AND NOT t.tgisinternal)
    OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_language l ON l.oid=p.prolang
      WHERE l.lanname IN ('plpgsql','sql')
        AND p.oid NOT IN (
          'vec_dietas.abrir_cola_revision_comision_v1()'::regprocedure,
          'vec_dietas.cotejar_efecto_circuito_v1(text,bytea,bytea,bytea)'::regprocedure,
          'vec_dietas.preleer_circuito_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
          'vec_dietas.decidir_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
          'vec_dietas.listar_bandeja_comisiones_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
        AND p.prosrc ~ '(cola_circuito_comision|preleer_circuito_comision_v1|decidir_comision_v1|listar_bandeja_comisiones_v1|cotejar_efecto_circuito_v1|abrir_cola_revision_comision_v1)')
 THEN RAISE EXCEPTION 'Dietas 000007 DOWN detecta dependencia o trigger ajeno' USING ERRCODE='55000'; END IF;
END $historia_dba$;

SET LOCAL ROLE vec_dietas_propietario;
DROP TRIGGER abrir_cola_revision ON vec_dietas.comision_revision RESTRICT;
DROP FUNCTION vec_dietas.abrir_cola_revision_comision_v1() RESTRICT;
DROP FUNCTION vec_dietas.preleer_circuito_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_dietas.decidir_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_dietas.listar_bandeja_comisiones_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
DROP FUNCTION vec_dietas.cotejar_efecto_circuito_v1(text,bytea,bytea,bytea) RESTRICT;
DROP TABLE vec_dietas.cola_circuito_comision RESTRICT;
COMMIT;
