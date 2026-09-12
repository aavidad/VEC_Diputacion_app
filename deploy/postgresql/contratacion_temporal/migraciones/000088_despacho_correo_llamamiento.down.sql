\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $down$
BEGIN
    RAISE EXCEPTION 'CT88: DOWN denegado; conserva reservas e historial de resultados de correo' USING ERRCODE='55000';
END
$down$;
COMMIT;
