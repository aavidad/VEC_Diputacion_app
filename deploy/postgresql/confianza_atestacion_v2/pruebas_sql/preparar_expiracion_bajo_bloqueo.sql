BEGIN;
SET LOCAL ROLE vec_confianza_atestacion_v2_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_confianza_atestacion_v2:gobierno:v1', 0
    )
);

DO $preparar$
DECLARE
    ahora timestamptz(6) := date_trunc('microseconds', clock_timestamp());
    publicada timestamptz(6);
    expira timestamptz(6);
    revision text := 'confianza:atestacion:v2:integracion:expiracion';
    raiz record;
    preimagen bytea;
    huella text;
BEGIN
    publicada := ahora - interval '1 second';
    expira := ahora + interval '5 seconds';
    preimagen :=
        vec_confianza_atestacion_v2.encuadrar_huella(
            'vec.configuracion-confianza-atestacion-autorizacion.v2'
        ) ||
        vec_confianza_atestacion_v2.encuadrar_huella(revision) ||
        vec_confianza_atestacion_v2.encuadrar_huella(
            vec_confianza_atestacion_v2.instante_rfc3339nano(publicada)
        ) ||
        vec_confianza_atestacion_v2.encuadrar_huella(
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
         WHERE enlace.configuracion_revision =
                   'confianza:atestacion:v2:integracion:1'
         ORDER BY version.clave_id COLLATE "C"
    LOOP
        preimagen := preimagen ||
            vec_confianza_atestacion_v2.encuadrar_huella(raiz.clave_id) ||
            vec_confianza_atestacion_v2.encuadrar_huella('EdDSA') ||
            vec_confianza_atestacion_v2.encuadrar_huella(
                raiz.huella_clave_spki_sha256
            ) ||
            vec_confianza_atestacion_v2.encuadrar_huella(
                'VEC-AD-2-COSE-EDDSA-1'
            ) ||
            vec_confianza_atestacion_v2.encuadrar_huella(
                raiz.audiencia_despliegue
            ) ||
            vec_confianza_atestacion_v2.encuadrar_huella('activa') ||
            vec_confianza_atestacion_v2.encuadrar_huella(
                vec_confianza_atestacion_v2.instante_rfc3339nano(
                    raiz.valida_desde
                )
            ) ||
            vec_confianza_atestacion_v2.encuadrar_huella(
                vec_confianza_atestacion_v2.instante_rfc3339nano(
                    raiz.valida_hasta
                )
            ) ||
            vec_confianza_atestacion_v2.encuadrar_huella('');
    END LOOP;
    huella := encode(sha256(preimagen), 'hex');

    INSERT INTO vec_confianza_atestacion_v2.acto_gobierno(
        acto_ref, secuencia, clase, emitido_en, documento_huella_sha256
    ) VALUES (
        'acto:integracion:publicacion:expiracion', 6,
        'publicacion_configuracion', ahora, repeat('6', 64)
    );
    INSERT INTO vec_confianza_atestacion_v2.acto_gobierno(
        acto_ref, secuencia, clase, emitido_en, documento_huella_sha256
    ) VALUES (
        'acto:integracion:activacion:expiracion', 7,
        'activacion_configuracion', ahora, repeat('7', 64)
    );
    INSERT INTO vec_confianza_atestacion_v2.configuracion_confianza_version(
        revision, secuencia, huella_configuracion_sha256,
        publicada_en, expira_en, numero_raices, acto_ref
    ) VALUES (
        revision, 2, huella, publicada, expira, 2,
        'acto:integracion:publicacion:expiracion'
    );
    INSERT INTO vec_confianza_atestacion_v2.configuracion_raiz(
        configuracion_revision, clave_id, version, acto_ref
    )
    SELECT revision, enlace.clave_id, enlace.version,
           'acto:integracion:publicacion:expiracion'
      FROM vec_confianza_atestacion_v2.configuracion_raiz AS enlace
     WHERE enlace.configuracion_revision =
               'confianza:atestacion:v2:integracion:1';
    INSERT INTO vec_confianza_atestacion_v2.puntero_configuracion_actual(
        orden, revision, establecida_en, acto_ref
    ) VALUES (
        2, revision, ahora, 'acto:integracion:activacion:expiracion'
    );
END
$preparar$;

COMMIT;
