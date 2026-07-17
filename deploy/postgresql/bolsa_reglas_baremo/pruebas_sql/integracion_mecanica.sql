-- Prueba la mecanica atomica, no el PDP. La frontera V2 se sustituye solo
-- dentro de esta transaccion revertida y las puertas runtime siguen cerradas.
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;

CREATE OR REPLACE FUNCTION
vec_autorizacion.revalidar_decision_reglas_baremo_v1(
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_recurso_canonico bytea,
    p_operacion text,
    p_correlacion_ref text,
    p_recurso_ref text,
    p_huella_contexto_sha256 text,
    p_instante timestamptz
)
RETURNS boolean
LANGUAGE sql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $prueba$ SELECT true $prueba$;

DO $retirar_checks$
DECLARE
    restriccion record;
BEGIN
    FOR restriccion IN
        SELECT conname FROM pg_catalog.pg_constraint
         WHERE conrelid =
             'vec_autorizacion.decision_autorizacion_solicitud_ligada_v2'::regclass
           AND contype IN ('c', 'f')
    LOOP
        EXECUTE format(
            'ALTER TABLE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2 DROP CONSTRAINT %I',
            restriccion.conname
        );
    END LOOP;
END
$retirar_checks$;
SET LOCAL session_replication_role = replica;

DO $decisiones$
DECLARE
    referencia text;
    canonica bytea;
BEGIN
    FOREACH referencia IN ARRAY ARRAY[
        'decision:reglas:alta', 'decision:reglas:alterada',
        'decision:reglas:publicar', 'decision:reglas:obsoleta',
        'decision:reglas:activar', 'decision:reglas:retirar',
        'decision:reglas:consulta', 'decision:reglas:consulta-dos'
    ] LOOP
        canonica := convert_to(jsonb_build_object(
            'decision_ref', referencia,
            'principal_id', 'principal:rrhh'
        )::text, 'UTF8');
        INSERT INTO vec_autorizacion.decision_autorizacion_solicitud_ligada_v2(
            decision_ref, huella_decision_sha256, decision_canonica,
            documento_v2, documento_comun, principal_id,
            perfil_activo_ref, accion, recurso_ref, modulo_id,
            tipo_recurso, contexto_recurso_huella_sha256, finalidad,
            correlacion_ref, solicitud_huella_sha256,
            motivo_huella_sha256, motivo_canonico, motivo_catalogo_id,
            motivo_catalogo_version, motivo_catalogo_huella_sha256,
            motivo_entrada_clave, asignacion_ref, version_rol_ref,
            control_vigencia_version_rol_ref,
            control_vigencia_version_rol_revision, emitida_en,
            valida_hasta, registrada_en
        ) VALUES (
            referencia, encode(sha256(canonica), 'hex'), canonica,
            '{}'::jsonb, '{}'::jsonb, 'principal:rrhh', 'perfil:rrhh',
            'prueba', 'reglas-baremo:' || repeat('a', 64), 'bolsa',
            'version_reglas_baremo_gobernada', repeat('b', 64),
            'prueba', 'correlacion_0123456789abcdef0123456789abcdef',
            repeat('c', 64), repeat('d', 64), convert_to('motivo', 'UTF8'),
            'motivos.reglas', 1, repeat('e', 64),
            'motivo_0123456789abcdef0123456789abcdef',
            'asignacion:rrhh', 'rol:rrhh', 'rol:rrhh', 1,
            clock_timestamp() - interval '1 second',
            clock_timestamp() + interval '2 minutes', clock_timestamp()
        );
    END LOOP;
END
$decisiones$;

SET LOCAL session_replication_role = origin;
SET LOCAL ROLE vec_bolsa_reglas_baremo_propietario;

DO $mecanica$
DECLARE
    contexto bytea := convert_to(
        '{"ambitos":{},"atributos":{}}', 'UTF8'
    );
    huella_contexto text := encode(sha256(contexto), 'hex');
    huella_contenido text := encode(
        sha256(convert_to('contenido-reglas-prueba', 'UTF8')), 'hex'
    );
    bytes_1 bytea := convert_to(
        '{"estado":"borrador","revision":1}', 'UTF8'
    );
    bytes_2 bytea := convert_to(
        '{"estado":"publicada","revision":2}', 'UTF8'
    );
    bytes_3 bytea := convert_to(
        '{"estado":"activa","revision":3}', 'UTF8'
    );
    bytes_4 bytea := convert_to(
        '{"estado":"retirada","revision":4}', 'UTF8'
    );
    h1 text;
    h2 text;
    h3 text;
    h4 text;
    ahora timestamptz(6) := date_trunc('microseconds', clock_timestamp());
    operacion jsonb;
    prueba jsonb;
    decision bytea;
    recibo record;
    obtenido bytea;
BEGIN
    h1 := encode(sha256(bytes_1), 'hex');
    h2 := encode(sha256(bytes_2), 'hex');
    h3 := encode(sha256(bytes_3), 'hex');
    h4 := encode(sha256(bytes_4), 'hex');

    decision := convert_to(jsonb_build_object(
        'decision_ref', 'decision:reglas:alta',
        'principal_id', 'principal:rrhh'
    )::text, 'UTF8');
    prueba := jsonb_build_object(
        'esquema_huella',
            'vec.autorizacion.decision.reforzada.v2.solicitud-ligada',
        'decision_ref', 'decision:reglas:alta',
        'huella_decision_sha256', encode(sha256(decision), 'hex'),
        'verificada_en', to_char(ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    );
    operacion := jsonb_build_object(
        'esquema',
            'vec.bolsa.reglas-baremo.confirmacion-postgresql.v1',
        'operacion', 'alta_borrador',
        'intencion_ref', 'intencion:reglas:alta',
        'intencion_version', 1,
        'intencion_huella_sha256', encode(
            sha256(convert_to('intencion-alta', 'UTF8')), 'hex'
        ),
        'esperado_revision', NULL,
        'esperado_huella_estado_sha256', NULL,
        'contenido_ref', 'reglas:convocatoria:2026',
        'contenido_version', 1,
        'huella_contenido_sha256', huella_contenido,
        'resultado_revision', 1,
        'resultado_estado', 'borrador',
        'resultado_huella_estado_sha256', h1,
        'prueba_ref', NULL, 'prueba_version', NULL,
        'prueba_huella_sha256', NULL,
        'accion', 'bolsa.reglas_baremo.borrador.crear',
        'recurso_ref', 'reglas-baremo:' || h1,
        'correlacion_ref',
            'correlacion_0123456789abcdef0123456789abcdef',
        'huella_contexto_recurso_sha256', huella_contexto,
        'efectuar_en', to_char(ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    );
    SELECT * INTO STRICT recibo
      FROM vec_bolsa_reglas_baremo.confirmar_cambio_v1(
          operacion, prueba, decision, contexto, bytes_1
      );
    IF recibo.resultado <> 'confirmada' OR recibo.revision <> 1 THEN
        RAISE EXCEPTION 'alta no confirmada';
    END IF;
    SELECT * INTO STRICT recibo
      FROM vec_bolsa_reglas_baremo.confirmar_cambio_v1(
          operacion, prueba, decision, contexto, bytes_1
      );
    IF recibo.resultado <> 'repetida'
       OR (SELECT count(*) FROM vec_bolsa_reglas_baremo.version_reglas_baremo) <> 1
       OR (SELECT count(*) FROM vec_bolsa_reglas_baremo.uso_decision) <> 1 THEN
        RAISE EXCEPTION 'idempotencia exacta no conservada';
    END IF;

    BEGIN
        PERFORM * FROM vec_bolsa_reglas_baremo.confirmar_cambio_v1(
            jsonb_set(
                jsonb_set(
                    operacion, '{resultado_huella_estado_sha256}',
                    to_jsonb(encode(sha256(convert_to(
                        '{"estado":"borrador-alterado","revision":1}',
                        'UTF8'
                    )), 'hex'))
                ),
                '{recurso_ref}',
                to_jsonb('reglas-baremo:' || encode(sha256(convert_to(
                    '{"estado":"borrador-alterado","revision":1}',
                    'UTF8'
                )), 'hex'))
            ), prueba, decision, contexto, convert_to(
                '{"estado":"borrador-alterado","revision":1}', 'UTF8'
            )
        );
        RAISE EXCEPTION 'indice idempotente alterado aceptado';
    EXCEPTION WHEN unique_violation THEN NULL;
    END;
    BEGIN
        PERFORM * FROM vec_bolsa_reglas_baremo.confirmar_cambio_v1(
            operacion, prueba, decision, contexto,
            convert_to('{"bytes":"alterados"}', 'UTF8')
        );
        RAISE EXCEPTION 'bytes alterados aceptados con huella previa';
    EXCEPTION WHEN invalid_parameter_value THEN NULL;
    END;
    BEGIN
        PERFORM * FROM vec_bolsa_reglas_baremo.confirmar_cambio_v1(
            jsonb_set(
                operacion, '{resultado_estado}', '"activa"'::jsonb
            ), prueba, decision, contexto, bytes_1
        );
        RAISE EXCEPTION 'proyeccion de estado incompatible aceptada';
    EXCEPTION WHEN invalid_parameter_value THEN NULL;
    END;

    ahora := date_trunc('microseconds', clock_timestamp());
    decision := convert_to(jsonb_build_object(
        'decision_ref', 'decision:reglas:publicar',
        'principal_id', 'principal:rrhh'
    )::text, 'UTF8');
    prueba := jsonb_build_object(
        'esquema_huella',
            'vec.autorizacion.decision.reforzada.v2.solicitud-ligada',
        'decision_ref', 'decision:reglas:publicar',
        'huella_decision_sha256', encode(sha256(decision), 'hex'),
        'verificada_en', to_char(ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    );
    operacion := jsonb_build_object(
        'esquema',
            'vec.bolsa.reglas-baremo.confirmacion-postgresql.v1',
        'operacion', 'publicar',
        'intencion_ref', 'intencion:reglas:publicar',
        'intencion_version', 1,
        'intencion_huella_sha256', encode(
            sha256(convert_to('intencion-publicar', 'UTF8')), 'hex'
        ),
        'esperado_revision', 1,
        'esperado_huella_estado_sha256', h1,
        'contenido_ref', 'reglas:convocatoria:2026',
        'contenido_version', 1,
        'huella_contenido_sha256', huella_contenido,
        'resultado_revision', 2,
        'resultado_estado', 'publicada',
        'resultado_huella_estado_sha256', h2,
        'prueba_ref', 'prueba:reglas:publicar', 'prueba_version', 1,
        'prueba_huella_sha256', repeat('1', 64),
        'accion', 'bolsa.reglas_baremo.publicar',
        'recurso_ref', 'reglas-baremo:' || h2,
        'correlacion_ref',
            'correlacion_1123456789abcdef0123456789abcdef',
        'huella_contexto_recurso_sha256', huella_contexto,
        'efectuar_en', to_char(ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    );
    PERFORM * FROM vec_bolsa_reglas_baremo.confirmar_cambio_v1(
        operacion, prueba, decision, contexto, bytes_2
    );

    BEGIN
        ahora := date_trunc('microseconds', clock_timestamp());
        decision := convert_to(jsonb_build_object(
            'decision_ref', 'decision:reglas:obsoleta',
            'principal_id', 'principal:rrhh'
        )::text, 'UTF8');
        prueba := jsonb_build_object(
            'esquema_huella',
                'vec.autorizacion.decision.reforzada.v2.solicitud-ligada',
            'decision_ref', 'decision:reglas:obsoleta',
            'huella_decision_sha256', encode(sha256(decision), 'hex'),
            'verificada_en', to_char(ahora AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        );
        PERFORM * FROM vec_bolsa_reglas_baremo.confirmar_cambio_v1(
            jsonb_build_object(
                'esquema',
                    'vec.bolsa.reglas-baremo.confirmacion-postgresql.v1',
                'operacion', 'descartar',
                'intencion_ref', 'intencion:reglas:obsoleta',
                'intencion_version', 1,
                'intencion_huella_sha256', repeat('2', 64),
                'esperado_revision', 1,
                'esperado_huella_estado_sha256', h1,
                'contenido_ref', 'reglas:convocatoria:2026',
                'contenido_version', 1,
                'huella_contenido_sha256', huella_contenido,
                'resultado_revision', 2,
                'resultado_estado', 'descartada',
                'resultado_huella_estado_sha256', encode(sha256(
                    convert_to('{"estado":"descartada","revision":2}',
                        'UTF8')
                ), 'hex'),
                'prueba_ref', 'prueba:reglas:obsoleta',
                'prueba_version', 1, 'prueba_huella_sha256', repeat('3', 64),
                'accion', 'bolsa.reglas_baremo.descartar',
                'recurso_ref', 'reglas-baremo:' || encode(sha256(
                    convert_to('{"estado":"descartada","revision":2}',
                        'UTF8')
                ), 'hex'),
                'correlacion_ref',
                    'correlacion_2123456789abcdef0123456789abcdef',
                'huella_contexto_recurso_sha256', huella_contexto,
                'efectuar_en', to_char(ahora AT TIME ZONE 'UTC',
                    'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
            ), prueba, decision, contexto,
            convert_to('{"estado":"descartada","revision":2}', 'UTF8')
        );
        RAISE EXCEPTION 'version obsoleta aceptada';
    EXCEPTION WHEN serialization_failure THEN NULL;
    END;

    ahora := date_trunc('microseconds', clock_timestamp());
    decision := convert_to(jsonb_build_object(
        'decision_ref', 'decision:reglas:activar',
        'principal_id', 'principal:rrhh'
    )::text, 'UTF8');
    prueba := jsonb_build_object(
        'esquema_huella',
            'vec.autorizacion.decision.reforzada.v2.solicitud-ligada',
        'decision_ref', 'decision:reglas:activar',
        'huella_decision_sha256', encode(sha256(decision), 'hex'),
        'verificada_en', to_char(ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    );
    operacion := jsonb_build_object(
        'esquema',
            'vec.bolsa.reglas-baremo.confirmacion-postgresql.v1',
        'operacion', 'activar', 'intencion_ref', 'intencion:reglas:activar',
        'intencion_version', 1, 'intencion_huella_sha256', repeat('4', 64),
        'esperado_revision', 2, 'esperado_huella_estado_sha256', h2,
        'contenido_ref', 'reglas:convocatoria:2026', 'contenido_version', 1,
        'huella_contenido_sha256', huella_contenido,
        'resultado_revision', 3, 'resultado_estado', 'activa',
        'resultado_huella_estado_sha256', h3,
        'prueba_ref', 'prueba:reglas:activar', 'prueba_version', 1,
        'prueba_huella_sha256', repeat('5', 64),
        'accion', 'bolsa.reglas_baremo.activar',
        'recurso_ref', 'reglas-baremo:' || h3,
        'correlacion_ref',
            'correlacion_3123456789abcdef0123456789abcdef',
        'huella_contexto_recurso_sha256', huella_contexto,
        'efectuar_en', to_char(ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    );
    PERFORM * FROM vec_bolsa_reglas_baremo.confirmar_cambio_v1(
        operacion, prueba, decision, contexto, bytes_3
    );

    ahora := date_trunc('microseconds', clock_timestamp());
    decision := convert_to(jsonb_build_object(
        'decision_ref', 'decision:reglas:retirar',
        'principal_id', 'principal:rrhh'
    )::text, 'UTF8');
    prueba := jsonb_build_object(
        'esquema_huella',
            'vec.autorizacion.decision.reforzada.v2.solicitud-ligada',
        'decision_ref', 'decision:reglas:retirar',
        'huella_decision_sha256', encode(sha256(decision), 'hex'),
        'verificada_en', to_char(ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    );
    operacion := jsonb_build_object(
        'esquema',
            'vec.bolsa.reglas-baremo.confirmacion-postgresql.v1',
        'operacion', 'retirar', 'intencion_ref', 'intencion:reglas:retirar',
        'intencion_version', 1, 'intencion_huella_sha256', repeat('6', 64),
        'esperado_revision', 3, 'esperado_huella_estado_sha256', h3,
        'contenido_ref', 'reglas:convocatoria:2026', 'contenido_version', 1,
        'huella_contenido_sha256', huella_contenido,
        'resultado_revision', 4, 'resultado_estado', 'retirada',
        'resultado_huella_estado_sha256', h4,
        'prueba_ref', 'prueba:reglas:retirar', 'prueba_version', 1,
        'prueba_huella_sha256', repeat('7', 64),
        'accion', 'bolsa.reglas_baremo.retirar',
        'recurso_ref', 'reglas-baremo:' || h4,
        'correlacion_ref',
            'correlacion_4123456789abcdef0123456789abcdef',
        'huella_contexto_recurso_sha256', huella_contexto,
        'efectuar_en', to_char(ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    );
    PERFORM * FROM vec_bolsa_reglas_baremo.confirmar_cambio_v1(
        operacion, prueba, decision, contexto, bytes_4
    );

    ahora := date_trunc('microseconds', clock_timestamp());
    decision := convert_to(jsonb_build_object(
        'decision_ref', 'decision:reglas:consulta',
        'principal_id', 'principal:rrhh'
    )::text, 'UTF8');
    prueba := jsonb_build_object(
        'esquema_huella',
            'vec.autorizacion.decision.reforzada.v2.solicitud-ligada',
        'decision_ref', 'decision:reglas:consulta',
        'huella_decision_sha256', encode(sha256(decision), 'hex'),
        'verificada_en', to_char(ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    );
    SELECT consulta.version_canonica INTO STRICT obtenido
      FROM vec_bolsa_reglas_baremo.obtener_version_exacta_v1(
          jsonb_build_object(
              'esquema',
                  'vec.bolsa.reglas-baremo.consulta-postgresql.v1',
              'contenido_ref', 'reglas:convocatoria:2026',
              'contenido_version', 1,
              'huella_contenido_sha256', huella_contenido,
              'revision', 1, 'huella_estado_sha256', h1,
              'accion', 'bolsa.reglas_baremo.version.consultar',
              'recurso_ref', 'reglas-baremo:' || h1,
              'correlacion_ref',
                  'correlacion_5123456789abcdef0123456789abcdef',
              'huella_contexto_recurso_sha256', huella_contexto,
              'solicitada_en', to_char(ahora AT TIME ZONE 'UTC',
                  'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
          ), prueba, decision, contexto
      ) AS consulta;
    IF obtenido <> bytes_1 THEN
        RAISE EXCEPTION 'consulta historica no devolvio bytes exactos';
    END IF;
    BEGIN
        PERFORM *
          FROM vec_bolsa_reglas_baremo.obtener_version_exacta_v1(
              jsonb_build_object(
                  'esquema',
                      'vec.bolsa.reglas-baremo.consulta-postgresql.v1',
                  'contenido_ref', 'reglas:convocatoria:2026',
                  'contenido_version', 1,
                  'huella_contenido_sha256', huella_contenido,
                  'revision', 1, 'huella_estado_sha256', h1,
                  'accion', 'bolsa.reglas_baremo.version.consultar',
                  'recurso_ref', 'reglas-baremo:' || h1,
                  'correlacion_ref',
                      'correlacion_5123456789abcdef0123456789abcdef',
                  'huella_contexto_recurso_sha256', huella_contexto,
                  'solicitada_en', to_char(ahora AT TIME ZONE 'UTC',
                      'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
              ), prueba, decision, contexto
          );
        RAISE EXCEPTION 'replay de consulta aceptado';
    EXCEPTION WHEN unique_violation THEN NULL;
    END;

    IF (SELECT count(*) FROM vec_bolsa_reglas_baremo.version_reglas_baremo) <> 4
       OR (SELECT revision FROM vec_bolsa_reglas_baremo.estado_actual) <> 4
       OR (SELECT count(*) FROM vec_bolsa_reglas_baremo.outbox) <> 4
       OR (SELECT count(*) FROM vec_bolsa_reglas_baremo.uso_prueba_transicion) <> 3
       OR (SELECT count(*) FROM vec_bolsa_reglas_baremo.uso_decision) <> 5
       OR (SELECT count(*) FROM vec_bolsa_reglas_baremo.auditoria) <> 5
       OR EXISTS (
           SELECT 1 FROM vec_bolsa_reglas_baremo.auditoria
            WHERE encode(sha256(
                decode(huella_anterior_sha256, 'hex') || registro_canonico
            ), 'hex') <> huella_auditoria_sha256
       ) THEN
        RAISE EXCEPTION 'historia, consumos o cadena de auditoria invalidos';
    END IF;
    BEGIN
        UPDATE vec_bolsa_reglas_baremo.version_reglas_baremo
           SET estado = 'borrador';
        RAISE EXCEPTION 'historia mutable';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
    END;
END
$mecanica$;

ROLLBACK;
