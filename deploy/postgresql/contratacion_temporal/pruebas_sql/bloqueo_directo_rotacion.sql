\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL statement_timeout = '15s';
SET LOCAL idle_in_transaction_session_timeout = '20s';

SELECT *
FROM vec_contratacion_temporal.preparar_alta_v2(
    pg_catalog.jsonb_build_object(
        'esquema', 'vec.contratacion-temporal.preparar-alta.v2',
        'sellos_hmac',
        pg_catalog.jsonb_build_object(
            'activo',
            pg_catalog.jsonb_build_object(
                'generacion', 2,
                'ambito_hmac',
                'hmac-sha256:vec.contratacion-temporal.'
                    || 'ambito-idempotencia/v2:' || repeat('a', 64),
                'huella_peticion_hmac',
                'hmac-sha256:vec.contratacion-temporal.'
                    || 'huella-peticion/v2:' || repeat('c', 64)
            ),
            'retenidos',
            pg_catalog.jsonb_build_array(
                pg_catalog.jsonb_build_object(
                    'generacion', 1,
                    'ambito_hmac',
                    'hmac-sha256:vec.contratacion-temporal.'
                        || 'ambito-idempotencia/v1:' || repeat('d', 64),
                    'huella_peticion_hmac',
                    'hmac-sha256:vec.contratacion-temporal.'
                        || 'huella-peticion/v1:' || repeat('b', 64)
                )
            )
        ),
        'organizacion_ref', 'organizacion:diputacion-granada',
        'actor_ref', 'actor:tecnica-rrhh-001',
        'perfil_ref', 'perfil:tecnica-rrhh',
        'reserva_ref_candidata', 'reserva:bloqueo-no-debe-ganar',
        'referencias_candidatas',
        pg_catalog.jsonb_build_object(
            'expediente_ref', 'expediente:bloqueo-no-debe-ganar',
            'numero_visible', '2026/BLOQ-NO-GANA',
            'recibo_ref', 'recibo:bloqueo-no-debe-ganar'
        )
    )
);

COMMIT;
