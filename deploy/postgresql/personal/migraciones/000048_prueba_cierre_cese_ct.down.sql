\set ON_ERROR_STOP on
-- Sólo copia desechable sin efectos B115; nunca revertir historia instalada.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $pre$
BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
    WHERE n.nspname='vec_bolsa_llamamientos' AND p.proname='aplicar_cese_nativo_ct_v1')
 THEN RAISE EXCEPTION 'Personal48: consumidor B115 presente' USING ERRCODE='55000'; END IF;
END $pre$;
DROP FUNCTION vec_personal.probar_cierre_cese_ct_v1(bytea);
-- USAGE del propietario Bolsa puede ser previo: se conserva.
COMMIT;
