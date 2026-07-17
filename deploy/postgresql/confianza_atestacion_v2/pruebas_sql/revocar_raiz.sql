BEGIN;
SET LOCAL ROLE vec_confianza_atestacion_v2_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_confianza_atestacion_v2:gobierno:v1', 0
    )
);

INSERT INTO vec_confianza_atestacion_v2.acto_gobierno(
    acto_ref, secuencia, clase, emitido_en, documento_huella_sha256
) VALUES (
    'acto:integracion:revocacion:raiz:1', 6, 'revocacion_raiz',
    date_trunc('microseconds', clock_timestamp()), repeat('5', 64)
);
INSERT INTO vec_confianza_atestacion_v2.revocacion_raiz(
    clave_id, version, revocada_en, motivo_catalogado_ref, acto_ref
) VALUES (
    'clave:atestacion:v2:integracion:1', 1,
    date_trunc('microseconds', clock_timestamp()),
    'motivo:seguridad:integracion',
    'acto:integracion:revocacion:raiz:1'
);
COMMIT;
