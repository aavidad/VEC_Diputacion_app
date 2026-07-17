BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v2_propietario;
SET LOCAL search_path = pg_catalog;
INSERT INTO vec_autorizacion_atestada_v2.clave_capacidad_version(
    clave_id, version, secreto_hmac, huella_secreto_sha256, emisor_id,
    audiencia, valida_desde, valida_hasta, acto_ref
) SELECT 'clave-capacidad:prueba:conocimiento:4', 4, secreto,
         encode(sha256(secreto), 'hex'),
         'broker-capacidad:prueba:conocimiento:4',
         'vec_autorizacion_atestada_v2.registrar_y_consumir',
         clock_timestamp() - interval '1 minute',
         clock_timestamp() + interval '1 hour',
         'acto:clave:prueba:conocimiento:4'
    FROM (SELECT decode(repeat('de', 32), 'hex') AS secreto) AS material;
COMMIT;
