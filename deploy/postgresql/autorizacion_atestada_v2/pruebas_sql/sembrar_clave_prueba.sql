BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v2_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

INSERT INTO vec_autorizacion_atestada_v2.clave_capacidad_version(
    clave_id, version, secreto_hmac, huella_secreto_sha256, emisor_id,
    audiencia, valida_desde, valida_hasta, acto_ref
) VALUES (
    'clave:capacidad:atestada:v2:prueba:1', 1,
    decode(repeat('42', 32), 'hex'),
    encode(sha256(decode(repeat('42', 32), 'hex')), 'hex'),
    'broker:atestacion:v2:prueba',
    'vec_autorizacion_atestada_v2.registrar_y_consumir',
    clock_timestamp() - interval '1 hour',
    clock_timestamp() + interval '2 hours',
    'acto:clave:capacidad:atestada:v2:prueba:1'
);
INSERT INTO vec_autorizacion_atestada_v2.puntero_clave_capacidad(
    orden, clave_id, version, establecida_en, acto_ref
) VALUES (
    1, 'clave:capacidad:atestada:v2:prueba:1', 1,
    clock_timestamp(), 'acto:activar:capacidad:atestada:v2:prueba:1'
);
COMMIT;
