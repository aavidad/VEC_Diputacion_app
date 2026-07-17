BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL ROLE vec_bolsa_reglas_baremo_propietario;
DO $candidato$
DECLARE
    ahora timestamptz(6) := date_trunc('microseconds', clock_timestamp());
    contexto bytea := convert_to(
        '{"ambitos":{},"atributos":{}}', 'UTF8'
    );
    bytes_esperados bytea := convert_to(
        '{"estado":"borrador","revision":1,"prueba":"concurrencia"}',
        'UTF8'
    );
    bytes_resultado bytea := convert_to(
        '{"estado":"publicada","revision":2,"ganador":"a"}', 'UTF8'
    );
    decision bytea := convert_to(jsonb_build_object(
        'decision_ref', 'decision:reglas:concurrencia:a',
        'principal_id', 'principal:rrhh'
    )::text, 'UTF8');
    huella_resultado text;
BEGIN
    huella_resultado := encode(sha256(bytes_resultado), 'hex');
    PERFORM * FROM vec_bolsa_reglas_baremo.confirmar_cambio_v1(
        jsonb_build_object(
            'esquema',
                'vec.bolsa.reglas-baremo.confirmacion-postgresql.v1',
            'operacion', 'publicar',
            'intencion_ref', 'intencion:reglas:concurrencia:a',
            'intencion_version', 1,
            'intencion_huella_sha256', repeat('2', 64),
            'esperado_revision', 1,
            'esperado_huella_estado_sha256',
                encode(sha256(bytes_esperados), 'hex'),
            'contenido_ref', 'reglas:concurrencia',
            'contenido_version', 1,
            'huella_contenido_sha256', encode(sha256(convert_to(
                'contenido-concurrencia', 'UTF8'
            )), 'hex'),
            'resultado_revision', 2, 'resultado_estado', 'publicada',
            'resultado_huella_estado_sha256', huella_resultado,
            'prueba_ref', 'prueba:reglas:concurrencia:a',
            'prueba_version', 1, 'prueba_huella_sha256', repeat('3', 64),
            'accion', 'bolsa.reglas_baremo.publicar',
            'recurso_ref', 'reglas-baremo:' || huella_resultado,
            'correlacion_ref',
                'correlacion_7123456789abcdef0123456789abcdef',
            'huella_contexto_recurso_sha256', encode(sha256(contexto), 'hex'),
            'efectuar_en', to_char(ahora AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        ),
        jsonb_build_object(
            'esquema_huella',
                'vec.autorizacion.decision.reforzada.v2.solicitud-ligada',
            'decision_ref', 'decision:reglas:concurrencia:a',
            'huella_decision_sha256', encode(sha256(decision), 'hex'),
            'verificada_en', to_char(ahora AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        ), decision, contexto, bytes_resultado
    );
END
$candidato$;
COMMIT;
