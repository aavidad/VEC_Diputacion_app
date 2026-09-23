\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
-- Comprobación de historia completa, sin que RLS pueda ocultarla.
LOCK TABLE vec_cronos_v1.marcaje_original,vec_cronos_v1.marcaje_acceso IN ACCESS EXCLUSIVE MODE;
DO $pre$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN
        RAISE EXCEPTION 'reversión Cronos requiere DBA' USING ERRCODE='42501';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_cronos_v1.marcaje_original)
       OR EXISTS (SELECT 1 FROM vec_cronos_v1.marcaje_acceso) THEN
        RAISE EXCEPTION 'Cronos conserva historia' USING ERRCODE='55000';
    END IF;
END
$pre$;
DROP FUNCTION vec_cronos_v1.registrar_marcaje_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
COMMIT;
