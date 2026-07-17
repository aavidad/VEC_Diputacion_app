BEGIN;
SET LOCAL ROLE vec_confianza_atestacion_v2_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
INSERT INTO vec_confianza_atestacion_v2.revocacion_raiz(
    clave_id, version, revocada_en, motivo_catalogado_ref, acto_ref
) VALUES (
    'raiz:prueba:snapshot:no-activa', 1,
    date_trunc('microseconds', clock_timestamp() + interval '1 second'),
    'seguridad.prueba.snapshot', 'acto:revocar:raiz:prueba:snapshot'
);
SELECT pg_sleep(3);
COMMIT;
