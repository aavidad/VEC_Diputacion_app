-- Preparacion efimera para dos escritores sobre el mismo CAS. Sustituye la
-- frontera V2 por un doble mecanico; el contenedor se destruye tras la prueba.
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

SET session_replication_role = replica;
DO $decisiones$
DECLARE
    referencia text;
    canonica bytea;
BEGIN
    FOREACH referencia IN ARRAY ARRAY[
        'decision:reglas:concurrencia:alta',
        'decision:reglas:concurrencia:a',
        'decision:reglas:concurrencia:b'
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
RESET session_replication_role;

BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL ROLE vec_bolsa_reglas_baremo_propietario;
DO $alta$
DECLARE
    ahora timestamptz(6) := date_trunc('microseconds', clock_timestamp());
    contexto bytea := convert_to(
        '{"ambitos":{},"atributos":{}}', 'UTF8'
    );
    bytes_resultado bytea := convert_to(
        '{"estado":"borrador","revision":1,"prueba":"concurrencia"}',
        'UTF8'
    );
    huella_resultado text;
    decision bytea;
BEGIN
    huella_resultado := encode(sha256(bytes_resultado), 'hex');
    decision := convert_to(jsonb_build_object(
        'decision_ref', 'decision:reglas:concurrencia:alta',
        'principal_id', 'principal:rrhh'
    )::text, 'UTF8');
    PERFORM * FROM vec_bolsa_reglas_baremo.confirmar_cambio_v1(
        jsonb_build_object(
            'esquema',
                'vec.bolsa.reglas-baremo.confirmacion-postgresql.v1',
            'operacion', 'alta_borrador',
            'intencion_ref', 'intencion:reglas:concurrencia:alta',
            'intencion_version', 1,
            'intencion_huella_sha256', repeat('1', 64),
            'esperado_revision', NULL,
            'esperado_huella_estado_sha256', NULL,
            'contenido_ref', 'reglas:concurrencia',
            'contenido_version', 1,
            'huella_contenido_sha256', encode(sha256(convert_to(
                'contenido-concurrencia', 'UTF8'
            )), 'hex'),
            'resultado_revision', 1, 'resultado_estado', 'borrador',
            'resultado_huella_estado_sha256', huella_resultado,
            'prueba_ref', NULL, 'prueba_version', NULL,
            'prueba_huella_sha256', NULL,
            'accion', 'bolsa.reglas_baremo.borrador.crear',
            'recurso_ref', 'reglas-baremo:' || huella_resultado,
            'correlacion_ref',
                'correlacion_6123456789abcdef0123456789abcdef',
            'huella_contexto_recurso_sha256', encode(sha256(contexto), 'hex'),
            'efectuar_en', to_char(ahora AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        ),
        jsonb_build_object(
            'esquema_huella',
                'vec.autorizacion.decision.reforzada.v2.solicitud-ligada',
            'decision_ref', 'decision:reglas:concurrencia:alta',
            'huella_decision_sha256', encode(sha256(decision), 'hex'),
            'verificada_en', to_char(ahora AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        ), decision, contexto, bytes_resultado
    );
END
$alta$;
COMMIT;
