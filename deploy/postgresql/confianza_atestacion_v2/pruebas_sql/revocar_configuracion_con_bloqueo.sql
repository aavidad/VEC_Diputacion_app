BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_confianza_atestacion_v2:gobierno:v1', 0
    )
);
SELECT pg_catalog.pg_advisory_lock(721702027);
SELECT pg_catalog.pg_sleep(2);
SET LOCAL ROLE vec_confianza_atestacion_v2_propietario;
INSERT INTO vec_confianza_atestacion_v2.acto_gobierno(
    acto_ref, secuencia, clase, emitido_en, documento_huella_sha256
) VALUES (
    'acto:integracion:revocacion:configuracion:1', 7,
    'revocacion_configuracion',
    date_trunc('microseconds', clock_timestamp()), repeat('7', 64)
);
INSERT INTO vec_confianza_atestacion_v2.revocacion_configuracion(
    revision, revocada_en, motivo_catalogado_ref, acto_ref
) VALUES (
    'confianza:atestacion:v2:integracion:1',
    date_trunc('microseconds', clock_timestamp()),
    'motivo:seguridad:integracion',
    'acto:integracion:revocacion:configuracion:1'
);
COMMIT;
SELECT pg_catalog.pg_advisory_unlock(721702027);
