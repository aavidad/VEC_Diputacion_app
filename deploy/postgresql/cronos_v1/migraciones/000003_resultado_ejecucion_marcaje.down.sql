\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
LOCK TABLE vec_cronos_v1.resultado_ejecucion_marcaje IN ACCESS EXCLUSIVE MODE;
DO $pre$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN
        RAISE EXCEPTION 'reversión Cronos requiere DBA' USING ERRCODE='42501';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_cronos_v1.resultado_ejecucion_marcaje) THEN
        RAISE EXCEPTION 'Cronos conserva resultados de ejecución' USING ERRCODE='55000';
    END IF;
END
$pre$;
DROP FUNCTION vec_cronos_v1.registrar_resultado_ejecucion_marcaje_v1(text,text,text,text,text,text,text,text,timestamptz);
DROP TABLE vec_cronos_v1.resultado_ejecucion_marcaje;
DO $conectar$ BEGIN EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM vec_cronos_v1_auditor', current_database()); END $conectar$;
REVOKE USAGE ON SCHEMA vec_cronos_v1 FROM vec_cronos_v1_auditor;
DROP ROLE vec_cronos_v1_auditor;
COMMIT;
