BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL TRANSACTION ISOLATION LEVEL SERIALIZABLE;

-- Los clientes se liberan aproximadamente a la vez. Cada uno propone
-- referencias distintas para el mismo ámbito y la misma petición.
SELECT pg_catalog.pg_sleep(0.10);

SELECT 1 / CASE
           WHEN resultado IN ('reservada', 'reutilizada')
            AND ambito_hmac =
                'hmac-sha256:vec.contratacion-temporal.'
                || 'ambito-idempotencia/v1:'
                || repeat('e', 64)
            AND estado = 'reservada'
           THEN 1
           ELSE 0
       END AS contrato_valido
  FROM vec_contratacion_temporal.preparar_alta_v1(
      pg_catalog.jsonb_build_object(
          'esquema',
          'vec.contratacion-temporal.preparar-alta.v1',
          'ambito_hmac',
          'hmac-sha256:vec.contratacion-temporal.'
              || 'ambito-idempotencia/v1:'
              || repeat('e', 64),
          'huella_peticion_hmac',
          'hmac-sha256:vec.contratacion-temporal.'
              || 'huella-peticion/v1:'
              || repeat('f', 64),
          'organizacion_ref',
          'organizacion:diputacion-granada',
          'actor_ref',
          'actor:concurrencia-prueba',
          'perfil_ref',
          'perfil:tecnica-rrhh',
          'reserva_ref_candidata',
          pg_catalog.format('reserva:concurrencia-%s', :client_id),
          'referencias_candidatas',
          pg_catalog.jsonb_build_object(
              'expediente_ref',
              pg_catalog.format(
                  'expediente:concurrencia-%s',
                  :client_id
              ),
              'numero_visible',
              pg_catalog.format('2026/CONC-%s', :client_id),
              'recibo_ref',
              pg_catalog.format('recibo:concurrencia-%s', :client_id)
          )
      )
  );

COMMIT;
