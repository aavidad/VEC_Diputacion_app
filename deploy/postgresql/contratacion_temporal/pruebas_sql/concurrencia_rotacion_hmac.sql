BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL statement_timeout = '15s';
SET LOCAL idle_in_transaction_session_timeout = '20s';
SET LOCAL TRANSACTION ISOLATION LEVEL SERIALIZABLE;

SELECT pg_catalog.pg_sleep(0.10);

SELECT 1 / CASE
           WHEN resultado IN ('reservada', 'reutilizada')
            AND estado = 'reservada'
           THEN 1
           ELSE 0
       END AS contrato_valido
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
                    || 'ambito-idempotencia/v2:' || repeat('7', 64),
                'huella_peticion_hmac',
                'hmac-sha256:vec.contratacion-temporal.'
                    || 'huella-peticion/v2:' || repeat('9', 64)
            ),
            'retenidos',
            pg_catalog.jsonb_build_array(
                pg_catalog.jsonb_build_object(
                    'generacion', 1,
                    'ambito_hmac',
                    'hmac-sha256:vec.contratacion-temporal.'
                        || 'ambito-idempotencia/v1:' || repeat('8', 64),
                    'huella_peticion_hmac',
                    'hmac-sha256:vec.contratacion-temporal.'
                        || 'huella-peticion/v1:' || repeat('a', 64)
                )
            )
        ),
        'organizacion_ref', 'organizacion:diputacion-granada',
        'actor_ref', 'actor:concurrencia-rotacion',
        'perfil_ref', 'perfil:tecnica-rrhh',
        'reserva_ref_candidata',
        pg_catalog.format('reserva:rotacion-concurrencia-%s', :client_id),
        'referencias_candidatas',
        pg_catalog.jsonb_build_object(
            'expediente_ref',
            pg_catalog.format(
                'expediente:rotacion-concurrencia-%s',
                :client_id
            ),
            'numero_visible',
            pg_catalog.format('2026/ROTC-%s', :client_id),
            'recibo_ref',
            pg_catalog.format('recibo:rotacion-concurrencia-%s', :client_id)
        )
    )
);

COMMIT;
