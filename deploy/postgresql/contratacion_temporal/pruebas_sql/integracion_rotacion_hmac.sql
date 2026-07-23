\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL TRANSACTION ISOLATION LEVEL SERIALIZABLE;

DO $prueba$
DECLARE
    v_fila record;
    v_rotada jsonb := pg_catalog.jsonb_build_object(
        'esquema',
        'vec.contratacion-temporal.preparar-alta.v2',
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
        'reserva_ref_candidata', 'reserva:rotacion-no-debe-ganar',
        'referencias_candidatas',
        pg_catalog.jsonb_build_object(
            'expediente_ref', 'expediente:rotacion-no-debe-ganar',
            'numero_visible', '2026/ROT-NO-GANA',
            'recibo_ref', 'recibo:rotacion-no-debe-ganar'
        )
    );
    v_nueva_a jsonb;
    v_nueva_b jsonb;
BEGIN
    SELECT *
      INTO STRICT v_fila
      FROM vec_contratacion_temporal.preparar_alta_v2(v_rotada);
    IF v_fila.resultado <> 'reutilizada'
       OR v_fila.reserva_ref <> 'reserva:alta-001'
       OR v_fila.expediente_ref <> 'expediente:ct-2026-0001'
       OR v_fila.numero_visible <> '2026/CT-0001'
       OR v_fila.recibo_ref <> 'recibo:alta-001'
       OR v_fila.ambito_hmac <>
           'hmac-sha256:vec.contratacion-temporal.'
           || 'ambito-idempotencia/v1:' || repeat('d', 64) THEN
        RAISE EXCEPTION
            'la rotación creó o devolvió otra reserva: %',
            row_to_json(v_fila);
    END IF;

    v_nueva_a := jsonb_set(
        jsonb_set(
            jsonb_set(
                jsonb_set(
                    v_rotada,
                    '{sellos_hmac,activo,ambito_hmac}',
                    to_jsonb(
                        'hmac-sha256:vec.contratacion-temporal.'
                        || 'ambito-idempotencia/v2:' || repeat('e', 64)
                    )
                ),
                '{sellos_hmac,activo,huella_peticion_hmac}',
                to_jsonb(
                    'hmac-sha256:vec.contratacion-temporal.'
                    || 'huella-peticion/v2:' || repeat('1', 64)
                )
            ),
            '{sellos_hmac,retenidos,0,ambito_hmac}',
            to_jsonb(
                'hmac-sha256:vec.contratacion-temporal.'
                || 'ambito-idempotencia/v1:' || repeat('f', 64)
            )
        ),
        '{sellos_hmac,retenidos,0,huella_peticion_hmac}',
        to_jsonb(
            'hmac-sha256:vec.contratacion-temporal.'
            || 'huella-peticion/v1:' || repeat('2', 64)
        )
    );
    v_nueva_a := jsonb_set(
        jsonb_set(
            jsonb_set(
                jsonb_set(
                    v_nueva_a,
                    '{reserva_ref_candidata}',
                    '"reserva:rotacion-a"'::jsonb
                ),
                '{referencias_candidatas,expediente_ref}',
                '"expediente:rotacion-a"'::jsonb
            ),
            '{referencias_candidatas,numero_visible}',
            '"2026/ROT-A"'::jsonb
        ),
        '{referencias_candidatas,recibo_ref}',
        '"recibo:rotacion-a"'::jsonb
    );
    SELECT *
      INTO STRICT v_fila
      FROM vec_contratacion_temporal.preparar_alta_v2(v_nueva_a);
    IF v_fila.resultado <> 'reservada'
       OR v_fila.reserva_ref <> 'reserva:rotacion-a'
       OR v_fila.ambito_hmac <>
           'hmac-sha256:vec.contratacion-temporal.'
           || 'ambito-idempotencia/v2:' || repeat('e', 64) THEN
        RAISE EXCEPTION 'alta nativa v2 incoherente: %', row_to_json(v_fila);
    END IF;

    BEGIN
        PERFORM *
        FROM vec_contratacion_temporal.preparar_alta_v2(
            jsonb_set(
                v_nueva_a,
                '{sellos_hmac,retenidos}',
                '[]'::jsonb
            )
        );
        RAISE EXCEPTION 'la ausencia de v1 durante la ventana fue aceptada';
    EXCEPTION
        WHEN invalid_parameter_value THEN
            NULL;
    END;

    SELECT *
      INTO STRICT v_fila
      FROM vec_contratacion_temporal.preparar_alta_v2(
          jsonb_set(
              jsonb_set(
                  v_nueva_a,
                  '{sellos_hmac,activo,huella_peticion_hmac}',
                  to_jsonb(
                      'hmac-sha256:vec.contratacion-temporal.'
                      || 'huella-peticion/v2:' || repeat('7', 64)
                  )
              ),
              '{sellos_hmac,retenidos,0,huella_peticion_hmac}',
              to_jsonb(
                  'hmac-sha256:vec.contratacion-temporal.'
                  || 'huella-peticion/v1:' || repeat('8', 64)
              )
          )
      );
    IF v_fila.resultado <> 'idempotencia_reutilizada'
       OR v_fila.reserva_ref <> 'reserva:rotacion-a' THEN
        RAISE EXCEPTION 'conflicto semántico no detectado: %', row_to_json(v_fila);
    END IF;

    v_nueva_b := jsonb_set(
        jsonb_set(
            jsonb_set(
                jsonb_set(
                    v_nueva_a,
                    '{sellos_hmac,activo,ambito_hmac}',
                    to_jsonb(
                        'hmac-sha256:vec.contratacion-temporal.'
                        || 'ambito-idempotencia/v2:' || repeat('3', 64)
                    )
                ),
                '{sellos_hmac,retenidos,0,ambito_hmac}',
                to_jsonb(
                    'hmac-sha256:vec.contratacion-temporal.'
                    || 'ambito-idempotencia/v1:' || repeat('4', 64)
                )
            ),
            '{sellos_hmac,activo,huella_peticion_hmac}',
            to_jsonb(
                'hmac-sha256:vec.contratacion-temporal.'
                || 'huella-peticion/v2:' || repeat('5', 64)
            )
        ),
        '{sellos_hmac,retenidos,0,huella_peticion_hmac}',
        to_jsonb(
            'hmac-sha256:vec.contratacion-temporal.'
            || 'huella-peticion/v1:' || repeat('6', 64)
        )
    );
    v_nueva_b := jsonb_set(
        jsonb_set(
            jsonb_set(
                jsonb_set(
                    v_nueva_b,
                    '{reserva_ref_candidata}',
                    '"reserva:rotacion-b"'::jsonb
                ),
                '{referencias_candidatas,expediente_ref}',
                '"expediente:rotacion-b"'::jsonb
            ),
            '{referencias_candidatas,numero_visible}',
            '"2026/ROT-B"'::jsonb
        ),
        '{referencias_candidatas,recibo_ref}',
        '"recibo:rotacion-b"'::jsonb
    );
    PERFORM * FROM vec_contratacion_temporal.preparar_alta_v2(v_nueva_b);

    BEGIN
        PERFORM *
        FROM vec_contratacion_temporal.preparar_alta_v2(
            jsonb_set(
                v_nueva_a,
                '{sellos_hmac,retenidos,0,ambito_hmac}',
                v_nueva_b #> '{sellos_hmac,retenidos,0,ambito_hmac}'
            )
        );
        RAISE EXCEPTION 'la convergencia de dos reservas fue aceptada';
    EXCEPTION
        WHEN unique_violation THEN
            NULL;
    END;
END
$prueba$;

COMMIT;
