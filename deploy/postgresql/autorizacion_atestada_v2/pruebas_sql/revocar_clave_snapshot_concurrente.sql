BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v2_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
INSERT INTO vec_autorizacion_atestada_v2.revocacion_clave_capacidad(
    clave_id, version, revocada_en, motivo_catalogado_ref, acto_ref
) VALUES (
    'clave-capacidad:prueba:snapshot:2', 2,
    date_trunc('microseconds', clock_timestamp() + interval '1 second'),
    'seguridad.prueba.snapshot', 'acto:revocar:clave:prueba:snapshot:2'
);
SELECT pg_sleep(3);
COMMIT;
