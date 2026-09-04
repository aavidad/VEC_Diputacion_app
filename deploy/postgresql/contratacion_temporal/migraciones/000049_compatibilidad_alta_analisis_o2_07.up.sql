\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:000049:compatibilidad-alta-analisis:o2-07', 0
));

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.preparar_operacion_analisis_v1(jsonb)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'frontera de análisis incompatible con O2-07';
    END IF;
END
$prevalidacion$;

DO $ampliar_normalizador$
DECLARE
    v_definicion text;
    v_ancla text := $ancla$    IF resultado #>> '{solicitud,observaciones}' = '' THEN
$ancla$;
    v_bloque text := $bloque$    -- O2-07 usa una forma canónica de transporte para RC ausente.
    -- Solo su valor neutro exacto se convierte a la forma del dominio Go.
    IF resultado #>> '{solicitud,rc,existe}' = 'false'
       AND resultado #>> '{solicitud,rc,fecha}' = ''
       AND resultado #>> '{solicitud,rc,importe,centimos}' = '0'
       AND resultado #>> '{solicitud,rc,importe,moneda}' = 'EUR'
       AND coalesce(resultado #>> '{solicitud,rc,numero}', '') = ''
       AND coalesce(
           resultado #>> '{solicitud,rc,documento_ref}', ''
       ) = '' THEN
        resultado := pg_catalog.jsonb_set(
            resultado, '{solicitud,rc,fecha}',
            pg_catalog.to_jsonb('0001-01-01T00:00:00Z'::text), false
        );
        resultado := pg_catalog.jsonb_set(
            resultado, '{solicitud,rc,importe,moneda}',
            pg_catalog.to_jsonb(''::text), false
        );
    END IF;
    IF resultado #>> '{solicitud,periodo,inicio}' ~
           '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
        resultado := pg_catalog.jsonb_set(
            resultado, '{solicitud,periodo,inicio}',
            pg_catalog.to_jsonb(
                (resultado #>> '{solicitud,periodo,inicio}') ||
                'T00:00:00Z'
            ), false
        );
    END IF;
    IF resultado #>> '{solicitud,periodo,fin}' ~
           '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
        resultado := pg_catalog.jsonb_set(
            resultado, '{solicitud,periodo,fin}',
            pg_catalog.to_jsonb(
                (resultado #>> '{solicitud,periodo,fin}') ||
                'T00:00:00Z'
            ), false
        );
    END IF;
$bloque$;
BEGIN
    SELECT pg_catalog.pg_get_functiondef(
        'vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(jsonb)'::regprocedure
    ) INTO STRICT v_definicion;
    IF pg_catalog.length(v_definicion) - pg_catalog.length(
           pg_catalog.replace(v_definicion, v_ancla, '')
       ) <> pg_catalog.length(v_ancla)
       OR pg_catalog.strpos(v_definicion, v_bloque) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'normalizador de análisis incompatible con O2-07';
    END IF;
    EXECUTE pg_catalog.replace(
        v_definicion, v_ancla, v_bloque || v_ancla
    );
END
$ampliar_normalizador$;

DO $normalizar_salida_reserva$
DECLARE
    v_definicion text;
    v_anterior text := $anterior$        CASE WHEN v_estado = 'reservada' THEN v_expediente::text ELSE '' END,
$anterior$;
    v_nueva text := $nueva$        CASE WHEN v_estado = 'reservada' THEN
            vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(
                v_expediente
            )::text ELSE '' END,
$nueva$;
BEGIN
    SELECT pg_catalog.pg_get_functiondef(
        'vec_contratacion_temporal.preparar_operacion_analisis_v1(jsonb)'::regprocedure
    ) INTO STRICT v_definicion;
    IF pg_catalog.length(v_definicion) - pg_catalog.length(
           pg_catalog.replace(v_definicion, v_anterior, '')
       ) <> pg_catalog.length(v_anterior)
       OR pg_catalog.strpos(v_definicion, v_nueva) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'salida de reserva de análisis incompatible con O2-07';
    END IF;
    EXECUTE pg_catalog.replace(v_definicion, v_anterior, v_nueva);
END
$normalizar_salida_reserva$;

DO $postvalidacion$
DECLARE
    v_neutra jsonb := '{
      "creado_en":"2026-09-04T00:00:00Z",
      "actualizado_en":"2026-09-04T00:00:00Z",
      "solicitud":{
        "observaciones":"",
        "periodo":{"inicio":"2027-01-01","fin":"2027-03-31"},
        "rc":{"existe":false,"numero":"","fecha":"",
              "importe":{"centimos":0,"moneda":"EUR"},
              "documento_ref":""},
        "documentos_adjuntos":[]
      },
      "actuaciones":[]
    }'::jsonb;
    v_residual jsonb := pg_catalog.jsonb_set(
        v_neutra, '{solicitud,rc,numero}', '"rc:residual"'::jsonb, false
    );
    v_normalizada jsonb;
BEGIN
    v_normalizada := vec_contratacion_temporal
        .normalizar_agregado_dominio_analisis_v2(v_neutra);
    IF v_normalizada #>> '{solicitud,rc,fecha}' <>
           '0001-01-01T00:00:00Z'
       OR v_normalizada #>> '{solicitud,rc,importe,moneda}' <> ''
       OR pg_catalog.jsonb_exists(
           v_normalizada #> '{solicitud,rc}', 'numero'
       )
       OR pg_catalog.jsonb_exists(
           v_normalizada #> '{solicitud,rc}', 'documento_ref'
       )
       OR v_normalizada #>> '{solicitud,periodo,inicio}' <>
          '2027-01-01T00:00:00Z'
       OR v_normalizada #>> '{solicitud,periodo,fin}' <>
          '2027-03-31T00:00:00Z' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'compatibilidad neutra de análisis no acreditada';
    END IF;
    v_normalizada := vec_contratacion_temporal
        .normalizar_agregado_dominio_analisis_v2(v_residual);
    IF v_normalizada #>> '{solicitud,rc,fecha}' <> ''
       OR v_normalizada #>> '{solicitud,rc,importe,moneda}' <> 'EUR'
       OR v_normalizada #>> '{solicitud,rc,numero}' <> 'rc:residual' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'dato residual de análisis aceptado';
    END IF;
END
$postvalidacion$;

COMMIT;
