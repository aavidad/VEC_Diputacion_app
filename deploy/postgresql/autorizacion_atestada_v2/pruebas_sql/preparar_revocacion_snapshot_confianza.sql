BEGIN;
SET LOCAL ROLE vec_confianza_atestacion_v2_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
DO $preparar$
DECLARE
    secuencia_actual numeric(20, 0);
    spki bytea := decode(
        '302a300506032b6570032100' || repeat('ab', 32), 'hex'
    );
    audiencia text;
BEGIN
    SELECT max(secuencia) INTO secuencia_actual
      FROM vec_confianza_atestacion_v2.acto_gobierno;
    SELECT audiencia_despliegue INTO STRICT audiencia
      FROM vec_confianza_atestacion_v2.raiz_confianza_version
     ORDER BY clave_id COLLATE "C", version LIMIT 1;
    INSERT INTO vec_confianza_atestacion_v2.acto_gobierno(
        acto_ref, secuencia, clase, emitido_en, documento_huella_sha256
    ) VALUES
        ('acto:publicar:raiz:prueba:snapshot', secuencia_actual + 1,
         'publicacion_raiz', clock_timestamp(), repeat('a', 64)),
        ('acto:revocar:raiz:prueba:snapshot', secuencia_actual + 2,
         'revocacion_raiz', clock_timestamp(), repeat('b', 64));
    INSERT INTO vec_confianza_atestacion_v2.raiz_confianza_version(
        clave_id, version, audiencia_despliegue, clave_publica_spki,
        huella_clave_spki_sha256, valida_desde, valida_hasta, acto_ref
    ) VALUES (
        'raiz:prueba:snapshot:no-activa', 1, audiencia, spki,
        encode(sha256(spki), 'hex'),
        date_trunc('microseconds', clock_timestamp() - interval '1 minute'),
        date_trunc('microseconds', clock_timestamp() + interval '1 hour'),
        'acto:publicar:raiz:prueba:snapshot'
    );
END
$preparar$;
COMMIT;
