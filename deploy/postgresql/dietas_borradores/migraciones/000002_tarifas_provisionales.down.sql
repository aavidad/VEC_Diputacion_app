\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000002:tarifas:v1',0));
DO $pre$
BEGIN
 IF current_user<>'vec_dietas_propietario'
    OR to_regclass('vec_dietas.version_tarifa_provisional') IS NULL
    OR to_regclass('vec_dietas.importe_dieta_provisional') IS NULL
    OR to_regclass('vec_dietas.importe_km_provisional') IS NULL
    OR (SELECT count(*) FROM vec_dietas.version_tarifa_provisional)<>1
    OR (SELECT count(*) FROM vec_dietas.importe_dieta_provisional)<>3
    OR (SELECT count(*) FROM vec_dietas.importe_km_provisional)<>2
    OR EXISTS (SELECT 1 FROM vec_dietas.version_tarifa_provisional WHERE version_ref<>'provisional:rd462:20260923') THEN
   RAISE EXCEPTION 'Dietas 000002 DOWN: catálogo alterado o ausente' USING ERRCODE='55000';
 END IF;
END $pre$;
DROP TABLE vec_dietas.importe_km_provisional;
DROP TABLE vec_dietas.importe_dieta_provisional;
DROP TABLE vec_dietas.version_tarifa_provisional;
COMMIT;
