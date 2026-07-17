SET SESSION AUTHORIZATION vec_ad2_emisor_prueba;
BEGIN;
SELECT count(*) FROM
    vec_autorizacion_atestada_v2.obtener_material_emisor_capacidad();
SELECT pg_sleep(3);
ROLLBACK;
