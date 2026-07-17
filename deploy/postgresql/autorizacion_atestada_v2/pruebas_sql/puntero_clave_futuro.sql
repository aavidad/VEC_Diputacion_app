-- El gobierno normal rechaza punteros futuros.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v2_propietario;
SET LOCAL search_path = pg_catalog;
DO $rechazar_futuro$
DECLARE
    rechazo boolean := false;
    secreto bytea := decode(repeat('cd', 32), 'hex');
BEGIN
    INSERT INTO vec_autorizacion_atestada_v2.clave_capacidad_version(
        clave_id, version, secreto_hmac, huella_secreto_sha256, emisor_id,
        audiencia, valida_desde, valida_hasta, acto_ref
    ) VALUES (
        'clave-capacidad:prueba:futura:3', 3, secreto,
        encode(sha256(secreto), 'hex'), 'broker-capacidad:prueba:futura:3',
        'vec_autorizacion_atestada_v2.registrar_y_consumir',
        date_trunc('microseconds', clock_timestamp() - interval '1 minute'),
        date_trunc('microseconds', clock_timestamp() + interval '2 hours'),
        'acto:clave:prueba:futura:3'
    );
    BEGIN
        INSERT INTO vec_autorizacion_atestada_v2.puntero_clave_capacidad(
            orden, clave_id, version, establecida_en, acto_ref
        ) VALUES (
            2, 'clave-capacidad:prueba:futura:3', 3,
            date_trunc('microseconds', clock_timestamp() + interval '1 hour'),
            'acto:activar:clave:prueba:futura:3'
        );
    EXCEPTION WHEN SQLSTATE '55000' THEN
        rechazo := true;
    END;
    IF NOT rechazo THEN
        RAISE EXCEPTION 'se acepto un puntero futuro';
    END IF;
END
$rechazar_futuro$;
ROLLBACK;

-- Fixture deliberadamente imposible, insertado sin triggers y revertido, que
-- demuestra que el lector elige el máximo puntero elegible y no el máximo
-- absoluto seguido de un filtro que dejaría el catálogo sin resultado.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL session_replication_role = replica;
INSERT INTO vec_autorizacion_atestada_v2.clave_capacidad_version(
    clave_id, version, secreto_hmac, huella_secreto_sha256, emisor_id,
    audiencia, valida_desde, valida_hasta, acto_ref
) SELECT 'clave-capacidad:prueba:futura:3', 3, secreto,
         encode(sha256(secreto), 'hex'),
         'broker-capacidad:prueba:futura:3',
         'vec_autorizacion_atestada_v2.registrar_y_consumir',
         date_trunc('microseconds', clock_timestamp() - interval '1 minute'),
         date_trunc('microseconds', clock_timestamp() + interval '2 hours'),
         'acto:clave:prueba:futura:3'
    FROM (SELECT decode(repeat('cd', 32), 'hex') AS secreto) AS material;
INSERT INTO vec_autorizacion_atestada_v2.puntero_clave_capacidad(
    orden, clave_id, version, establecida_en, acto_ref
) VALUES (
    2, 'clave-capacidad:prueba:futura:3', 3,
    date_trunc('microseconds', clock_timestamp() + interval '1 hour'),
    'acto:activar:clave:prueba:futura:3'
);
SET LOCAL session_replication_role = origin;
CREATE FUNCTION pg_temp.comprobar_maximo_elegible()
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
AS $prueba$
DECLARE
    material record;
BEGIN
    SELECT * INTO material FROM
        vec_autorizacion_atestada_v2.obtener_material_emisor_capacidad();
    IF NOT FOUND OR material.clave_id IS DISTINCT FROM
           'clave:capacidad:atestada:v2:prueba:1' THEN
        RAISE EXCEPTION 'el lector no eligio el maximo puntero elegible';
    END IF;
END
$prueba$;
SET SESSION AUTHORIZATION vec_ad2_emisor_prueba;
SELECT pg_temp.comprobar_maximo_elegible();
RESET SESSION AUTHORIZATION;
ROLLBACK;
