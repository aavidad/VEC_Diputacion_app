BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
DROP FUNCTION vec_contratacion_temporal.leer_expediente_seleccion_v1(text,text,bigint);
COMMIT;
