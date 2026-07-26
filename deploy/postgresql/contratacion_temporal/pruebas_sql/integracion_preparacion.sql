\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL TRANSACTION ISOLATION LEVEL SERIALIZABLE;

DO $prueba$
DECLARE
    v_fila record;
    v_operacion jsonb := pg_catalog.jsonb_build_object(
        'esquema',
        'vec.contratacion-temporal.preparar-alta.v1',
        'ambito_hmac',
        'hmac-sha256:vec.contratacion-temporal.'
            || 'ambito-idempotencia/v1:'
            || repeat('d', 64),
        'huella_peticion_hmac',
        'hmac-sha256:vec.contratacion-temporal.'
            || 'huella-peticion/v1:'
            || repeat('b', 64),
        'organizacion_ref',
        'organizacion:diputacion-granada',
        'actor_ref',
        'actor:tecnica-rrhh-001',
        'perfil_ref',
        'perfil:tecnica-rrhh',
        'reserva_ref_candidata',
        'reserva:alta-001',
        'referencias_candidatas',
        pg_catalog.jsonb_build_object(
            'expediente_ref',
            'expediente:ct-2026-0001',
            'numero_visible',
            '2026/CT-0001',
            'recibo_ref',
            'recibo:alta-001'
        )
    );
BEGIN
    SELECT *
      INTO STRICT v_fila
      FROM vec_contratacion_temporal.preparar_alta_v1(v_operacion);
    IF v_fila.resultado <> 'reservada'
       OR v_fila.estado <> 'reservada'
       OR v_fila.ambito_hmac <>
           'hmac-sha256:vec.contratacion-temporal.'
           || 'ambito-idempotencia/v1:'
           || repeat('d', 64)
       OR v_fila.reserva_ref <> 'reserva:alta-001'
       OR v_fila.version_expediente IS NOT NULL
       OR v_fila.auditoria_ref IS NOT NULL
       OR v_fila.evento_ref IS NOT NULL
       OR v_fila.confirmada_en IS NOT NULL THEN
        RAISE EXCEPTION 'primera reserva incoherente: %', row_to_json(v_fila);
    END IF;

    v_operacion := jsonb_set(
        v_operacion,
        '{reserva_ref_candidata}',
        '"reserva:alta-candidata-002"'::jsonb
    );
    v_operacion := jsonb_set(
        v_operacion,
        '{referencias_candidatas,expediente_ref}',
        '"expediente:ct-2026-0002"'::jsonb
    );
    SELECT *
      INTO STRICT v_fila
      FROM vec_contratacion_temporal.preparar_alta_v1(v_operacion);
    IF v_fila.resultado <> 'reutilizada'
       OR v_fila.reserva_ref <> 'reserva:alta-001'
       OR v_fila.expediente_ref <> 'expediente:ct-2026-0001' THEN
        RAISE EXCEPTION 'reintento no estable: %', row_to_json(v_fila);
    END IF;

    v_operacion := jsonb_set(
        v_operacion,
        '{huella_peticion_hmac}',
        to_jsonb(
            'hmac-sha256:vec.contratacion-temporal.'
            || 'huella-peticion/v1:'
            || repeat('c', 64)
        )
    );
    SELECT *
      INTO STRICT v_fila
      FROM vec_contratacion_temporal.preparar_alta_v1(v_operacion);
    IF v_fila.resultado <> 'idempotencia_reutilizada'
       OR v_fila.reserva_ref <> 'reserva:alta-candidata-002'
       OR v_fila.expediente_ref <> 'expediente:ct-2026-0002'
       OR v_fila.huella_peticion_hmac
           <> 'hmac-sha256:vec.contratacion-temporal.'
              || 'huella-peticion/v1:'
              || repeat('c', 64)
       OR v_fila.ambito_hmac <>
           'hmac-sha256:vec.contratacion-temporal.'
           || 'ambito-idempotencia/v1:'
           || repeat('d', 64)
       OR v_fila.estado <> 'reservada' THEN
        RAISE EXCEPTION 'conflicto semántico no detectado: %', row_to_json(v_fila);
    END IF;

    v_operacion := jsonb_set(
        v_operacion,
        '{huella_peticion_hmac}',
        to_jsonb(
            'hmac-sha256:vec.contratacion-temporal.'
            || 'huella-peticion/v1:'
            || repeat('b', 64)
        )
    );
    v_operacion := jsonb_set(
        v_operacion,
        '{organizacion_ref}',
        '"organizacion:otra-entidad"'::jsonb
    );
    SELECT *
      INTO STRICT v_fila
      FROM vec_contratacion_temporal.preparar_alta_v1(v_operacion);
    IF v_fila.resultado <> 'idempotencia_reutilizada'
       OR v_fila.ambito_hmac <>
           'hmac-sha256:vec.contratacion-temporal.'
           || 'ambito-idempotencia/v1:'
           || repeat('d', 64) THEN
        RAISE EXCEPTION
            'el ámbito no quedó ligado a organización: %',
            row_to_json(v_fila);
    END IF;

    BEGIN
        PERFORM *
          FROM vec_contratacion_temporal.preparar_alta_v1(
              jsonb_set(
                  v_operacion,
                  '{ambito_hmac}',
                  to_jsonb(
                      'hmac-sha256:vec.contratacion-temporal.'
                      || 'ambito-idempotencia/v1:'
                      || repeat('0', 64)
                  )
              )
          );
        RAISE EXCEPTION 'una HMAC nula fue admitida';
    EXCEPTION
        WHEN invalid_parameter_value THEN
            NULL;
    END;

    BEGIN
        PERFORM *
          FROM vec_contratacion_temporal.preparar_alta_v1(
              jsonb_set(
                  v_operacion,
                  '{huella_peticion_hmac}',
                  to_jsonb(
                      'hmac-sha256:vec.contratacion-temporal.'
                      || 'huella-peticion/v1:'
                      || repeat('0', 64)
                  )
              )
          );
        RAISE EXCEPTION 'una huella HMAC nula fue admitida';
    EXCEPTION
        WHEN invalid_parameter_value THEN
            NULL;
    END;

    BEGIN
        PERFORM *
          FROM vec_contratacion_temporal.preparar_alta_v1(
              jsonb_set(
                  v_operacion,
                  '{ambito_hmac}',
                  to_jsonb(
                      'hmac-sha256:dominio-no-autorizado/v1:'
                      || repeat('a', 64)
                  )
              )
          );
        RAISE EXCEPTION 'un dominio HMAC no autorizado fue admitido';
    EXCEPTION
        WHEN invalid_parameter_value THEN
            NULL;
    END;

    BEGIN
        PERFORM *
          FROM vec_contratacion_temporal.preparar_alta_v1(
              jsonb_set(
                  v_operacion,
                  '{huella_peticion_hmac}',
                  to_jsonb(
                      'hmac-sha256:dominio-no-autorizado/v1:'
                      || repeat('a', 64)
                  )
              )
          );
        RAISE EXCEPTION
            'un dominio de huella HMAC no autorizado fue admitido';
    EXCEPTION
        WHEN invalid_parameter_value THEN
            NULL;
    END;
END
$prueba$;

COMMIT;
