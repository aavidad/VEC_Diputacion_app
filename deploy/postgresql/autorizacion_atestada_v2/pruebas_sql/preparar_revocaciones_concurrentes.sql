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
) VALUES
    ('acto:prueba:revocar:raiz:atestada', 900, 'revocacion_raiz',
     clock_timestamp(), repeat('d', 64)),
    ('acto:prueba:revocar:configuracion:atestada', 901,
     'revocacion_configuracion', clock_timestamp(), repeat('e', 64)),
    ('acto:prueba:publicar:configuracion:rotada', 902,
     'publicacion_configuracion', clock_timestamp(), repeat('a', 64)),
    ('acto:prueba:activar:configuracion:rotada', 903,
     'activacion_configuracion', clock_timestamp(), repeat('b', 64));

DO $preparar_configuracion_rotada$
DECLARE
    revision_actual text;
    revision_nueva text := 'confianza:atestacion:v2:prueba:rotada';
    publicada timestamptz(6) :=
        date_trunc('microseconds', clock_timestamp() - interval '1 second');
    expira timestamptz(6) :=
        date_trunc('microseconds', clock_timestamp() + interval '1 hour');
    raiz record;
    preimagen bytea;
    numero integer := 0;
    huella text;
BEGIN
    SELECT revision INTO STRICT revision_actual
      FROM vec_confianza_atestacion_v2.puntero_configuracion_actual
     ORDER BY orden DESC LIMIT 1;
    preimagen :=
        vec_confianza_atestacion_v2.encuadrar_huella(
            'vec.configuracion-confianza-atestacion-autorizacion.v2'
        ) || vec_confianza_atestacion_v2.encuadrar_huella(revision_nueva) ||
        vec_confianza_atestacion_v2.encuadrar_huella(
            vec_confianza_atestacion_v2.instante_rfc3339nano(publicada)
        ) || vec_confianza_atestacion_v2.encuadrar_huella(
            vec_confianza_atestacion_v2.instante_rfc3339nano(expira)
        );
    FOR raiz IN
        SELECT version.clave_id, version.huella_clave_spki_sha256,
               version.audiencia_despliegue, version.valida_desde,
               version.valida_hasta
          FROM vec_confianza_atestacion_v2.configuracion_raiz AS enlace
          JOIN vec_confianza_atestacion_v2.raiz_confianza_version AS version
            ON version.clave_id = enlace.clave_id
           AND version.version = enlace.version
         WHERE enlace.configuracion_revision = revision_actual
         ORDER BY version.clave_id COLLATE "C"
    LOOP
        numero := numero + 1;
        preimagen := preimagen ||
            vec_confianza_atestacion_v2.encuadrar_huella(raiz.clave_id) ||
            vec_confianza_atestacion_v2.encuadrar_huella('EdDSA') ||
            vec_confianza_atestacion_v2.encuadrar_huella(
                raiz.huella_clave_spki_sha256
            ) || vec_confianza_atestacion_v2.encuadrar_huella(
                'VEC-AD-2-COSE-EDDSA-1'
            ) || vec_confianza_atestacion_v2.encuadrar_huella(
                raiz.audiencia_despliegue
            ) || vec_confianza_atestacion_v2.encuadrar_huella('activa') ||
            vec_confianza_atestacion_v2.encuadrar_huella(
                vec_confianza_atestacion_v2.instante_rfc3339nano(
                    raiz.valida_desde
                )
            ) || vec_confianza_atestacion_v2.encuadrar_huella(
                vec_confianza_atestacion_v2.instante_rfc3339nano(
                    raiz.valida_hasta
                )
            ) || vec_confianza_atestacion_v2.encuadrar_huella('');
    END LOOP;
    huella := encode(sha256(preimagen), 'hex');
    INSERT INTO vec_confianza_atestacion_v2.configuracion_confianza_version(
        revision, secuencia, huella_configuracion_sha256, publicada_en,
        expira_en, numero_raices, acto_ref
    ) VALUES (
        revision_nueva, 2, huella, publicada, expira, numero,
        'acto:prueba:publicar:configuracion:rotada'
    );
    INSERT INTO vec_confianza_atestacion_v2.configuracion_raiz(
        configuracion_revision, clave_id, version, acto_ref
    )
    SELECT revision_nueva, enlace.clave_id, enlace.version,
           'acto:prueba:publicar:configuracion:rotada'
      FROM vec_confianza_atestacion_v2.configuracion_raiz AS enlace
     WHERE enlace.configuracion_revision = revision_actual;
    IF vec_confianza_atestacion_v2.calcular_huella_configuracion(
           revision_nueva
       ) IS DISTINCT FROM huella THEN
        RAISE EXCEPTION 'fixture de configuracion rotada incoherente';
    END IF;
END
$preparar_configuracion_rotada$;
COMMIT;
