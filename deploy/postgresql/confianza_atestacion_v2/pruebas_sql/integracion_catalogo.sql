BEGIN;
SET LOCAL ROLE vec_confianza_atestacion_v2_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_confianza_atestacion_v2:gobierno:v1', 0
    )
);

DO $sembrar_material_publico$
DECLARE
    ahora timestamptz(6) := date_trunc('microseconds', clock_timestamp());
    publicada timestamptz(6);
    expira timestamptz(6);
    valida_desde timestamptz(6);
    valida_hasta timestamptz(6);
    revision text := 'confianza:atestacion:v2:integracion:1';
    clave_id text := 'clave:atestacion:v2:integracion:1';
    clave_id_segunda text := 'clave:atestacion:v2:integracion:2';
    audiencia text := 'vec-diputacion/pruebas/vec/autorizacion-v2';
    spki bytea := decode(
        '302a300506032b6570032100' ||
        'd75a980182b10ab7d54bfed3c964073a' ||
        '0ee172f3daa62325af021a68f707511a',
        'hex'
    );
    spki_segunda bytea := decode(
        '302a300506032b6570032100' ||
        '3d4017c3e843895a92b70aa74d1b7ebc' ||
        '9c982ccf2ec4968cc0cd55f12af4660c',
        'hex'
    );
    huella_spki text;
    huella_spki_segunda text;
    preimagen bytea;
    huella_configuracion text;
BEGIN
    publicada := ahora - interval '1 minute';
    expira := ahora + interval '1 hour';
    valida_desde := ahora - interval '1 hour';
    valida_hasta := ahora + interval '2 hours';
    huella_spki := encode(sha256(spki), 'hex');
    huella_spki_segunda := encode(sha256(spki_segunda), 'hex');
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
        ) ||
        vec_confianza_atestacion_v2.encuadrar_huella(clave_id) ||
        vec_confianza_atestacion_v2.encuadrar_huella('EdDSA') ||
        vec_confianza_atestacion_v2.encuadrar_huella(huella_spki) ||
        vec_confianza_atestacion_v2.encuadrar_huella(
            'VEC-AD-2-COSE-EDDSA-1'
        ) ||
        vec_confianza_atestacion_v2.encuadrar_huella(audiencia) ||
        vec_confianza_atestacion_v2.encuadrar_huella('activa') ||
        vec_confianza_atestacion_v2.encuadrar_huella(
            vec_confianza_atestacion_v2.instante_rfc3339nano(valida_desde)
        ) ||
        vec_confianza_atestacion_v2.encuadrar_huella(
            vec_confianza_atestacion_v2.instante_rfc3339nano(valida_hasta)
        ) ||
        vec_confianza_atestacion_v2.encuadrar_huella('') ||
        vec_confianza_atestacion_v2.encuadrar_huella(clave_id_segunda) ||
        vec_confianza_atestacion_v2.encuadrar_huella('EdDSA') ||
        vec_confianza_atestacion_v2.encuadrar_huella(
            huella_spki_segunda
        ) ||
        vec_confianza_atestacion_v2.encuadrar_huella(
            'VEC-AD-2-COSE-EDDSA-1'
        ) ||
        vec_confianza_atestacion_v2.encuadrar_huella(audiencia) ||
        vec_confianza_atestacion_v2.encuadrar_huella('activa') ||
        vec_confianza_atestacion_v2.encuadrar_huella(
            vec_confianza_atestacion_v2.instante_rfc3339nano(valida_desde)
        ) ||
        vec_confianza_atestacion_v2.encuadrar_huella(
            vec_confianza_atestacion_v2.instante_rfc3339nano(valida_hasta)
        ) ||
        vec_confianza_atestacion_v2.encuadrar_huella('');
    huella_configuracion := encode(sha256(preimagen), 'hex');

    INSERT INTO vec_confianza_atestacion_v2.acto_gobierno(
        acto_ref, secuencia, clase, emitido_en,
        documento_huella_sha256
    ) VALUES (
        'acto:integracion:publicacion:raiz:1', 1,
        'publicacion_raiz', ahora, repeat('1', 64)
    );
    INSERT INTO vec_confianza_atestacion_v2.acto_gobierno(
        acto_ref, secuencia, clase, emitido_en,
        documento_huella_sha256
    ) VALUES (
        'acto:integracion:publicacion:configuracion:1', 2,
        'publicacion_configuracion', ahora, repeat('2', 64)
    );
    INSERT INTO vec_confianza_atestacion_v2.acto_gobierno(
        acto_ref, secuencia, clase, emitido_en,
        documento_huella_sha256
    ) VALUES (
        'acto:integracion:activacion:raiz:1', 3,
        'activacion_raiz', ahora, repeat('3', 64)
    );
    INSERT INTO vec_confianza_atestacion_v2.acto_gobierno(
        acto_ref, secuencia, clase, emitido_en,
        documento_huella_sha256
    ) VALUES (
        'acto:integracion:activacion:raiz:2', 4,
        'activacion_raiz', ahora, repeat('4', 64)
    );
    INSERT INTO vec_confianza_atestacion_v2.acto_gobierno(
        acto_ref, secuencia, clase, emitido_en,
        documento_huella_sha256
    ) VALUES (
        'acto:integracion:activacion:configuracion:1', 5,
        'activacion_configuracion', ahora, repeat('4', 64)
    );

    INSERT INTO vec_confianza_atestacion_v2.raiz_confianza_version(
        clave_id, version, audiencia_despliegue, clave_publica_spki,
        huella_clave_spki_sha256, valida_desde, valida_hasta, acto_ref
    ) VALUES (
        clave_id, 1, audiencia, spki, huella_spki,
        valida_desde, valida_hasta,
        'acto:integracion:publicacion:raiz:1'
    );
    INSERT INTO vec_confianza_atestacion_v2.raiz_confianza_version(
        clave_id, version, audiencia_despliegue, clave_publica_spki,
        huella_clave_spki_sha256, valida_desde, valida_hasta, acto_ref
    ) VALUES (
        clave_id_segunda, 1, audiencia, spki_segunda,
        huella_spki_segunda, valida_desde, valida_hasta,
        'acto:integracion:publicacion:raiz:1'
    );
    INSERT INTO vec_confianza_atestacion_v2.configuracion_confianza_version(
        revision, secuencia, huella_configuracion_sha256,
        publicada_en, expira_en, numero_raices, acto_ref
    ) VALUES (
        revision, 1, huella_configuracion,
        publicada, expira, 2,
        'acto:integracion:publicacion:configuracion:1'
    );
    INSERT INTO vec_confianza_atestacion_v2.configuracion_raiz(
        configuracion_revision, clave_id, version, acto_ref
    ) VALUES (
        revision, clave_id, 1,
        'acto:integracion:publicacion:configuracion:1'
    );
    INSERT INTO vec_confianza_atestacion_v2.configuracion_raiz(
        configuracion_revision, clave_id, version, acto_ref
    ) VALUES (
        revision, clave_id_segunda, 1,
        'acto:integracion:publicacion:configuracion:1'
    );
    INSERT INTO vec_confianza_atestacion_v2.puntero_raiz_actual(
        clave_id, orden, version, establecida_en, acto_ref
    ) VALUES (
        clave_id, 1, 1, ahora,
        'acto:integracion:activacion:raiz:1'
    );
    INSERT INTO vec_confianza_atestacion_v2.puntero_raiz_actual(
        clave_id, orden, version, establecida_en, acto_ref
    ) VALUES (
        clave_id_segunda, 1, 1, ahora,
        'acto:integracion:activacion:raiz:2'
    );
    INSERT INTO vec_confianza_atestacion_v2.puntero_configuracion_actual(
        orden, revision, establecida_en, acto_ref
    ) VALUES (
        1, revision, ahora,
        'acto:integracion:activacion:configuracion:1'
    );

    IF vec_confianza_atestacion_v2.calcular_huella_configuracion(
           revision
       ) IS DISTINCT FROM huella_configuracion THEN
        RAISE EXCEPTION 'la huella SQL no coincide con VEC-AD-2';
    END IF;
END
$sembrar_material_publico$;

-- Regresiones de inmutabilidad y anti-rollback. Cada caso adversarial usa una
-- subtransaccion y no deja actos sinteticos residuales.
DO $regresiones$
DECLARE
    ahora timestamptz(6) := date_trunc('microseconds', clock_timestamp());
    spki_existente bytea;
    audiencia text;
    desde timestamptz(6);
    hasta timestamptz(6);
BEGIN
    SELECT clave_publica_spki, audiencia_despliegue,
           valida_desde, valida_hasta
      INTO STRICT spki_existente, audiencia, desde, hasta
     FROM vec_confianza_atestacion_v2.raiz_confianza_version
     WHERE clave_id = 'clave:atestacion:v2:integracion:1' AND version = 1;

    IF vec_confianza_atestacion_v2.instante_go_valido(
           '10000-01-01 00:00:00+00'::timestamptz
       ) OR vec_confianza_atestacion_v2.instante_go_valido(
           '0001-01-01 00:00:00 BC'::timestamptz
       ) THEN
        RAISE EXCEPTION 'se acepto un instante fuera del rango Go';
    END IF;

    BEGIN
        UPDATE vec_confianza_atestacion_v2.configuracion_confianza_version
           SET numero_raices = numero_raices
         WHERE revision = 'confianza:atestacion:v2:integracion:1';
        RAISE EXCEPTION 'UPDATE de historia aceptado';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        DELETE FROM vec_confianza_atestacion_v2.puntero_raiz_actual
         WHERE clave_id = 'clave:atestacion:v2:integracion:1';
        RAISE EXCEPTION 'DELETE de historia aceptado';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        TRUNCATE TABLE vec_confianza_atestacion_v2.puntero_raiz_actual;
        RAISE EXCEPTION 'TRUNCATE de historia aceptado';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        INSERT INTO vec_confianza_atestacion_v2.configuracion_raiz(
            configuracion_revision, clave_id, version, acto_ref
        ) VALUES (
            'confianza:atestacion:v2:integracion:1',
            'clave:atestacion:v2:integracion:1', 1,
            'acto:integracion:publicacion:configuracion:1'
        );
        RAISE EXCEPTION 'se amplio una configuracion ya activada';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;

    BEGIN
        INSERT INTO vec_confianza_atestacion_v2.acto_gobierno(
            acto_ref, secuencia, clase, emitido_en,
            documento_huella_sha256
        ) VALUES (
            'acto:integracion:alias:rechazado', 6,
            'publicacion_raiz', ahora, repeat('5', 64)
        );
        INSERT INTO vec_confianza_atestacion_v2.raiz_confianza_version(
            clave_id, version, audiencia_despliegue, clave_publica_spki,
            huella_clave_spki_sha256, valida_desde, valida_hasta, acto_ref
        ) VALUES (
            'clave:atestacion:v2:alias:rechazado', 1, audiencia,
            spki_existente, encode(sha256(spki_existente), 'hex'),
            desde, hasta, 'acto:integracion:alias:rechazado'
        );
        RAISE EXCEPTION 'alias de material SPKI aceptado';
    EXCEPTION WHEN unique_violation THEN NULL;
    END;

    BEGIN
        INSERT INTO vec_confianza_atestacion_v2.acto_gobierno(
            acto_ref, secuencia, clase, emitido_en,
            documento_huella_sha256
        ) VALUES (
            'acto:integracion:version:rechazada', 6,
            'publicacion_raiz', ahora, repeat('5', 64)
        );
        INSERT INTO vec_confianza_atestacion_v2.raiz_confianza_version(
            clave_id, version, audiencia_despliegue, clave_publica_spki,
            huella_clave_spki_sha256, valida_desde, valida_hasta, acto_ref
        ) VALUES (
            'clave:atestacion:v2:integracion:1', 1, audiencia,
            decode(
                '302a300506032b6570032100' || repeat('11', 32),
                'hex'
            ),
            encode(sha256(decode(
                '302a300506032b6570032100' || repeat('11', 32),
                'hex'
            )), 'hex'),
            desde, hasta, 'acto:integracion:version:rechazada'
        );
        RAISE EXCEPTION 'version de raiz no monotona aceptada';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;

    BEGIN
        INSERT INTO vec_confianza_atestacion_v2.acto_gobierno(
            acto_ref, secuencia, clase, emitido_en,
            documento_huella_sha256
        ) VALUES (
            'acto:integracion:raiz:futura', 6,
            'publicacion_raiz', ahora, repeat('6', 64)
        );
        INSERT INTO vec_confianza_atestacion_v2.acto_gobierno(
            acto_ref, secuencia, clase, emitido_en,
            documento_huella_sha256
        ) VALUES (
            'acto:integracion:puntero:futuro', 7,
            'activacion_raiz', ahora, repeat('7', 64)
        );
        INSERT INTO vec_confianza_atestacion_v2.raiz_confianza_version(
            clave_id, version, audiencia_despliegue, clave_publica_spki,
            huella_clave_spki_sha256, valida_desde, valida_hasta, acto_ref
        ) VALUES (
            'clave:atestacion:v2:integracion:1', 2, audiencia,
            decode(
                '302a300506032b6570032100' || repeat('22', 32),
                'hex'
            ),
            encode(sha256(decode(
                '302a300506032b6570032100' || repeat('22', 32),
                'hex'
            )), 'hex'),
            ahora + interval '30 seconds', ahora + interval '2 hours',
            'acto:integracion:raiz:futura'
        );
        INSERT INTO vec_confianza_atestacion_v2.puntero_raiz_actual(
            clave_id, orden, version, establecida_en, acto_ref
        ) VALUES (
            'clave:atestacion:v2:integracion:1', 2, 2,
            ahora + interval '1 minute',
            'acto:integracion:puntero:futuro'
        );
        RAISE EXCEPTION 'se acepto un puntero futuro';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;

    BEGIN
        INSERT INTO vec_confianza_atestacion_v2.acto_gobierno(
            acto_ref, secuencia, clase, emitido_en,
            documento_huella_sha256
        ) VALUES (
            'acto:integracion:publicacion:retroceso', 6,
            'publicacion_configuracion', ahora, repeat('6', 64)
        );
        INSERT INTO vec_confianza_atestacion_v2.configuracion_confianza_version(
            revision, secuencia, huella_configuracion_sha256,
            publicada_en, expira_en, numero_raices, acto_ref
        ) SELECT 'confianza:atestacion:v2:publicacion:retroceso', 2,
                 repeat('6', 64), publicada_en,
                 publicada_en + interval '1 hour', 1,
                 'acto:integracion:publicacion:retroceso'
            FROM vec_confianza_atestacion_v2.configuracion_confianza_version
           WHERE revision = 'confianza:atestacion:v2:integracion:1';
        RAISE EXCEPTION 'publicacion de configuracion no creciente aceptada';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;

    BEGIN
        INSERT INTO vec_confianza_atestacion_v2.acto_gobierno(
            acto_ref, secuencia, clase, emitido_en,
            documento_huella_sha256
        ) VALUES (
            'acto:integracion:puntero:rechazado', 6,
            'activacion_configuracion', ahora, repeat('5', 64)
        );
        INSERT INTO vec_confianza_atestacion_v2.puntero_configuracion_actual(
            orden, revision, establecida_en, acto_ref
        ) VALUES (
            2, 'confianza:atestacion:v2:integracion:1', ahora,
            'acto:integracion:puntero:rechazado'
        );
        RAISE EXCEPTION 'retroceso de configuracion aceptado';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;

    BEGIN
        INSERT INTO vec_confianza_atestacion_v2.acto_gobierno(
            acto_ref, secuencia, clase, emitido_en,
            documento_huella_sha256
        ) VALUES (
            'acto:integracion:configuracion:falsa', 6,
            'publicacion_configuracion', ahora, repeat('5', 64)
        );
        INSERT INTO vec_confianza_atestacion_v2.acto_gobierno(
            acto_ref, secuencia, clase, emitido_en,
            documento_huella_sha256
        ) VALUES (
            'acto:integracion:activacion:falsa', 7,
            'activacion_configuracion', ahora, repeat('6', 64)
        );
        INSERT INTO vec_confianza_atestacion_v2.configuracion_confianza_version(
            revision, secuencia, huella_configuracion_sha256,
            publicada_en, expira_en, numero_raices, acto_ref
        ) VALUES (
            'confianza:atestacion:v2:integracion:falsa', 2, repeat('f', 64),
            ahora - interval '1 minute', ahora + interval '1 hour', 1,
            'acto:integracion:configuracion:falsa'
        );
        INSERT INTO vec_confianza_atestacion_v2.configuracion_raiz(
            configuracion_revision, clave_id, version, acto_ref
        ) VALUES (
            'confianza:atestacion:v2:integracion:falsa',
            'clave:atestacion:v2:integracion:1', 1,
            'acto:integracion:configuracion:falsa'
        );
        INSERT INTO vec_confianza_atestacion_v2.puntero_configuracion_actual(
            orden, revision, establecida_en, acto_ref
        ) VALUES (
            2, 'confianza:atestacion:v2:integracion:falsa', ahora,
            'acto:integracion:activacion:falsa'
        );
        RAISE EXCEPTION 'configuracion con huella falsa activada';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
END
$regresiones$;

COMMIT;
