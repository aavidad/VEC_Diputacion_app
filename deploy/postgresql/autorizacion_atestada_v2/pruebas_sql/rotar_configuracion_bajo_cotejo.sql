BEGIN;
SET LOCAL ROLE vec_confianza_atestacion_v2_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '500ms';
SET LOCAL timezone = 'UTC';
INSERT INTO vec_confianza_atestacion_v2.puntero_configuracion_actual(
    orden, revision, establecida_en, acto_ref
) VALUES (
    2, 'confianza:atestacion:v2:prueba:rotada', clock_timestamp(),
    'acto:prueba:activar:configuracion:rotada'
);
ROLLBACK;
