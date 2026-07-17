BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v2_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
INSERT INTO vec_autorizacion_atestada_v2.clave_capacidad_version(
    clave_id, version, secreto_hmac, huella_secreto_sha256, emisor_id,
    audiencia, valida_desde, valida_hasta, acto_ref
) SELECT 'clave-capacidad:prueba:snapshot:2', 2, secreto,
         encode(sha256(secreto), 'hex'),
         'broker-capacidad:prueba:snapshot:2',
         'vec_autorizacion_atestada_v2.registrar_y_consumir',
         date_trunc('microseconds', clock_timestamp() - interval '1 minute'),
         date_trunc('microseconds', clock_timestamp() + interval '1 hour'),
         'acto:clave:prueba:snapshot:2'
    FROM (SELECT public.gen_random_bytes(32) AS secreto) AS material;
COMMIT;
