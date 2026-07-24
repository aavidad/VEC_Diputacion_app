BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000011_transicion_confirmacion_analisis', 0
    )
);

CREATE FUNCTION
vec_contratacion_temporal.transicion_confirmacion_analisis_valida_v1(
    o jsonb,
    p_agregado_actual jsonb
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    anterior jsonb := o -> 'expediente_anterior';
    siguiente jsonb := o -> 'expediente_siguiente';
    actuacion jsonb := o -> 'actuacion';
    politica jsonb := o -> 'politica';
    v_numero_actuaciones integer;
    v_actor_anterior text;
    v_claves_actuacion text[];
BEGIN
    IF anterior IS DISTINCT FROM p_agregado_actual
       OR pg_catalog.jsonb_typeof(anterior -> 'actuaciones') <> 'array'
       OR pg_catalog.jsonb_typeof(siguiente -> 'actuaciones') <> 'array'
       OR pg_catalog.jsonb_typeof(siguiente -> 'analisis') <> 'object'
       OR siguiente -> 'via_cobertura' IS NOT NULL
       OR siguiente -> 'asignacion' IS NOT NULL THEN
        RETURN false;
    END IF;
    v_numero_actuaciones :=
        pg_catalog.jsonb_array_length(anterior -> 'actuaciones');
    IF v_numero_actuaciones < 1
       OR pg_catalog.jsonb_array_length(siguiente -> 'actuaciones') <>
          v_numero_actuaciones + 1
       OR siguiente -> 'actuaciones' -> v_numero_actuaciones
            IS DISTINCT FROM actuacion
       OR (
           SELECT pg_catalog.jsonb_agg(e.v ORDER BY e.i)
             FROM pg_catalog.jsonb_array_elements(
                 siguiente -> 'actuaciones'
             ) WITH ORDINALITY AS e(v, i)
            WHERE e.i <= v_numero_actuaciones
       ) IS DISTINCT FROM anterior -> 'actuaciones' THEN
        RETURN false;
    END IF;
    IF anterior ->> 'referencia' <> o ->> 'expediente_ref'
       OR siguiente ->> 'referencia' <> o ->> 'expediente_ref'
       OR anterior ->> 'organizacion_ref' <> o ->> 'organizacion_ref'
       OR siguiente ->> 'organizacion_ref' <> o ->> 'organizacion_ref'
       OR (anterior ->> 'version')::numeric <>
            (o ->> 'version_anterior')::numeric
       OR (siguiente ->> 'version')::numeric <>
            (o ->> 'version_anterior')::numeric + 1
       OR siguiente -> 'numero_visible'
            IS DISTINCT FROM anterior -> 'numero_visible'
       OR siguiente -> 'flujo' IS DISTINCT FROM anterior -> 'flujo'
       OR siguiente -> 'solicitud' IS DISTINCT FROM anterior -> 'solicitud'
       OR siguiente -> 'creado_en' IS DISTINCT FROM anterior -> 'creado_en'
       OR siguiente -> 'fase_actual'
            IS DISTINCT FROM actuacion -> 'fase_destino'
       OR siguiente -> 'estado_actual'
            IS DISTINCT FROM actuacion -> 'estado_destino'
       OR siguiente -> 'actualizado_en'
            IS DISTINCT FROM actuacion -> 'realizada_en' THEN
        RETURN false;
    END IF;
    IF (actuacion ->> 'secuencia')::numeric <>
           (siguiente ->> 'version')::numeric
       OR (actuacion ->> 'version_expediente')::numeric <>
           (siguiente ->> 'version')::numeric
       OR actuacion ->> 'accion_clave' <> politica ->> 'accion'
       OR actuacion ->> 'actor_ref' <> o ->> 'actor_ref'
       OR actuacion ->> 'unidad_ref' <> politica ->> 'unidad_ref'
       OR actuacion ->> 'recibo_ref' <> o ->> 'recibo_ref'
       OR actuacion -> 'fase_origen'
            IS DISTINCT FROM anterior -> 'fase_actual'
       OR actuacion -> 'fase_destino'
            IS DISTINCT FROM anterior -> 'fase_actual'
       OR actuacion -> 'estado_origen'
            IS DISTINCT FROM anterior -> 'estado_actual'
       OR actuacion -> 'estado_destino'
            IS DISTINCT FROM anterior -> 'estado_actual'
       OR politica -> 'fase_previa'
            IS DISTINCT FROM anterior -> 'fase_actual'
       OR politica -> 'estado_previo'
            IS DISTINCT FROM anterior -> 'estado_actual' THEN
        RETURN false;
    END IF;
    v_claves_actuacion := CASE
        WHEN o ->> 'operacion' = 'registrar' THEN ARRAY[
          'accion_clave', 'actor_ref', 'estado_destino', 'estado_origen',
          'fase_destino', 'fase_origen', 'realizada_en', 'recibo_ref',
          'secuencia', 'unidad_ref', 'version_expediente'
        ]::text[]
        ELSE ARRAY[
          'accion_clave', 'actor_ref', 'estado_destino', 'estado_origen',
          'fase_destino', 'fase_origen', 'observaciones', 'realizada_en',
          'recibo_ref', 'secuencia', 'unidad_ref', 'version_expediente'
        ]::text[]
    END;
    IF NOT vec_contratacion_temporal.claves_json_exactas_v1(
           actuacion, v_claves_actuacion
       )
       OR siguiente #>> '{analisis,actuacion_registro,secuencia}' <>
            actuacion ->> 'secuencia'
       OR siguiente #>>
          '{analisis,actuacion_registro,version_expediente}' <>
            actuacion ->> 'version_expediente'
       OR siguiente #>> '{analisis,actuacion_registro,accion_clave}' <>
            actuacion ->> 'accion_clave'
       OR siguiente #>> '{analisis,actuacion_registro,fase_destino}' <>
            actuacion ->> 'fase_destino'
       OR siguiente #>> '{analisis,actuacion_registro,recibo_ref}' <>
            actuacion ->> 'recibo_ref' THEN
        RETURN false;
    END IF;
    IF o ->> 'operacion' = 'registrar' THEN
        IF anterior -> 'analisis' IS NOT NULL
           OR politica ->> 'accion' <>
              'contratacion_temporal.analisis.registrar'
           OR (politica ->> 'exige_actor_distinto')::boolean THEN
            RETURN false;
        END IF;
    ELSE
        IF pg_catalog.jsonb_typeof(anterior -> 'analisis') <> 'object'
           OR politica ->> 'accion' <>
              'contratacion_temporal.analisis.rectificar'
           OR NOT (politica ->> 'exige_actor_distinto')::boolean
           OR coalesce(actuacion ->> 'observaciones', '') !~
              '^contratacion_temporal[.]analisis[.]rectificacion[.]'
           OR coalesce(
               anterior #>> '{analisis,actuacion_registro,secuencia}', ''
           ) !~ '^[1-9][0-9]{0,15}$' THEN
            RETURN false;
        END IF;
        SELECT e.v ->> 'actor_ref'
          INTO v_actor_anterior
          FROM pg_catalog.jsonb_array_elements(
              anterior -> 'actuaciones'
          ) WITH ORDINALITY AS e(v, i)
         WHERE e.i = (
             anterior #>> '{analisis,actuacion_registro,secuencia}'
         )::integer;
        IF v_actor_anterior IS NULL
           OR v_actor_anterior = o ->> 'actor_ref' THEN
            RETURN false;
        END IF;
    END IF;
    RETURN true;
EXCEPTION
    WHEN data_exception OR datetime_field_overflow
      OR invalid_text_representation OR numeric_value_out_of_range THEN
        RETURN false;
END
$funcion$;

REVOKE ALL ON FUNCTION
vec_contratacion_temporal.transicion_confirmacion_analisis_valida_v1(
    jsonb, jsonb
) FROM PUBLIC, vec_contratacion_temporal_ejecutor;

COMMIT;
