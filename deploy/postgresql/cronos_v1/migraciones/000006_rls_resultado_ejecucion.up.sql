\set ON_ERROR_STOP on
-- Correctivo de 000003: el sumidero de resultados también requiere RLS
-- forzada. La función nominal auditora sigue siendo el único escritor.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:000006',0));
DO $pre$
BEGIN
 IF to_regprocedure('vec_cronos_v1.registrar_resultado_ejecucion_marcaje_v1(text,text,text,text,text,text,text,text,timestamptz)') IS NULL
  OR (SELECT relrowsecurity OR relforcerowsecurity FROM pg_class WHERE oid='vec_cronos_v1.resultado_ejecucion_marcaje'::regclass) THEN
   RAISE EXCEPTION 'preimagen de resultado Cronos incompatible' USING ERRCODE='55000';
 END IF;
END $pre$;
ALTER TABLE vec_cronos_v1.resultado_ejecucion_marcaje ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.resultado_ejecucion_marcaje FORCE ROW LEVEL SECURITY;
CREATE POLICY registrar_resultado_auditor_nominal ON vec_cronos_v1.resultado_ejecucion_marcaje
 FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (accion='cronos.marcaje.propio.registrar'
  AND resultado IN ('fallo_confirmado','resultado_indeterminado')
  AND causa IN ('persistencia','conflicto','recibo_invalido','rollback','commit'));
CREATE POLICY leer_resultado_para_recibo_nominal ON vec_cronos_v1.resultado_ejecucion_marcaje
 FOR SELECT TO vec_cronos_v1_propietario
 USING (accion='cronos.marcaje.propio.registrar');
REVOKE ALL ON vec_cronos_v1.resultado_ejecucion_marcaje FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_auditor;
COMMIT;
