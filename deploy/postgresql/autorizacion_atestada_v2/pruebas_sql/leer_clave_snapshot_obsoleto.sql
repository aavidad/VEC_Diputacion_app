\set VERBOSITY verbose
SET SESSION AUTHORIZATION vec_ad2_emisor_prueba;
BEGIN ISOLATION LEVEL SERIALIZABLE;
-- Fija el snapshot mientras la revocación concurrente aún no ha hecho COMMIT.
SELECT count(*) FROM pg_catalog.pg_class;
SELECT * FROM
    vec_autorizacion_atestada_v2.obtener_material_emisor_capacidad();
COMMIT;
