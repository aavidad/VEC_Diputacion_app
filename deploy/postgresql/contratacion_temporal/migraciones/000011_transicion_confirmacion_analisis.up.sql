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

-- La alta O2 conserva en su evidencia de entrada claves opcionales vacías
-- que encoding/json omite al rehidratar el agregado de dominio en Go. La
-- comparación CAS normaliza exclusivamente esas representaciones equivalentes;
-- nunca elimina valores informativos ni acepta una mutación funcional.
CREATE FUNCTION
vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v1(
    p_agregado jsonb
)
RETURNS jsonb
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    resultado jsonb := p_agregado;
    actuacion jsonb;
    indice integer;
    cantidad integer;
BEGIN
    IF resultado #>> '{solicitud,observaciones}' = '' THEN
        resultado := resultado #- '{solicitud,observaciones}';
    END IF;
    IF resultado #>> '{solicitud,rc,numero}' = '' THEN
        resultado := resultado #- '{solicitud,rc,numero}';
    END IF;
    IF resultado #>> '{solicitud,rc,documento_ref}' = '' THEN
        resultado := resultado #- '{solicitud,rc,documento_ref}';
    END IF;
    IF resultado #> '{solicitud,documentos_adjuntos}' = '[]'::jsonb THEN
        resultado := pg_catalog.jsonb_set(
            resultado,
            '{solicitud,documentos_adjuntos}',
            'null'::jsonb,
            false
        );
    END IF;
    IF pg_catalog.jsonb_typeof(resultado -> 'actuaciones') = 'array' THEN
        cantidad := pg_catalog.jsonb_array_length(
            resultado -> 'actuaciones'
        );
        IF cantidad > 0 THEN
            FOR indice IN 0..cantidad - 1 LOOP
                actuacion := resultado #> ARRAY[
                    'actuaciones', indice::text
                ];
                IF actuacion ->> 'observaciones' = '' THEN
                    actuacion := actuacion - 'observaciones';
                END IF;
                IF actuacion -> 'documentos_ref' = '[]'::jsonb THEN
                    actuacion := actuacion - 'documentos_ref';
                END IF;
                resultado := pg_catalog.jsonb_set(
                    resultado,
                    ARRAY['actuaciones', indice::text],
                    actuacion,
                    false
                );
            END LOOP;
        END IF;
    END IF;
    RETURN resultado;
END
$funcion$;

-- Calcula desde el JSON que va a persistirse la misma dirección de contenido
-- funcional que el núcleo Go incorporó a la decisión V3.
CREATE FUNCTION
vec_contratacion_temporal.huella_analisis_derivado_v1(
    p_analisis jsonb
)
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    a jsonb := p_analisis;
    v jsonb := p_analisis -> 'validacion_rc';
    v_prueba bytea;
    v_tiene_coste boolean;
    v_rc_validada boolean;
BEGIN
    IF pg_catalog.jsonb_typeof(a) <> 'object'
       OR pg_catalog.jsonb_typeof(a -> 'periodo') <> 'object'
       OR pg_catalog.jsonb_typeof(a -> 'entrada_rc_esperada') <> 'object'
       OR pg_catalog.jsonb_typeof(a -> 'actuacion_registro') <> 'object'
       OR pg_catalog.jsonb_typeof(v) <> 'object' THEN
        RETURN NULL;
    END IF;
    v_tiene_coste := pg_catalog.jsonb_exists(a, 'coste_previsto');
    v_rc_validada := v ->> 'resultado' = 'validada';
    IF NOT vec_contratacion_temporal.claves_json_exactas_v1(
           a,
           CASE WHEN v_tiene_coste THEN ARRAY[
             'actuacion_registro', 'categoria_ref', 'causa_clave',
             'coste_previsto', 'entrada_rc_esperada',
             'fuente_coste_ref', 'grupo_subgrupo', 'modalidad_clave',
             'periodo', 'porcentaje_jornada', 'validacion_rc'
           ]::text[] ELSE ARRAY[
             'actuacion_registro', 'categoria_ref', 'causa_clave',
             'entrada_rc_esperada', 'grupo_subgrupo', 'modalidad_clave',
             'periodo', 'porcentaje_jornada', 'validacion_rc'
           ]::text[] END
       )
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           a -> 'periodo', ARRAY['fin', 'inicio']::text[]
       )
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           a -> 'entrada_rc_esperada',
           ARRAY['huella_sha256', 'referencia']::text[]
       )
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           a -> 'actuacion_registro', ARRAY[
             'accion_clave', 'fase_destino', 'recibo_ref', 'secuencia',
             'version_expediente'
           ]::text[]
       )
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           v,
           CASE WHEN v_rc_validada THEN ARRAY[
             'documento_ref', 'entrada_ref', 'fecha_rc', 'fuente_ref',
             'huella_entrada_sha256', 'importe', 'numero', 'recibo_ref',
             'resultado', 'validada_en'
           ]::text[] ELSE ARRAY[
             'entrada_ref', 'fuente_ref', 'huella_entrada_sha256',
             'motivo', 'recibo_ref', 'resultado', 'validada_en'
           ]::text[] END
       )
       OR (
           v_rc_validada AND (
               pg_catalog.jsonb_typeof(v -> 'importe') <> 'object'
               OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
                   v -> 'importe', ARRAY['centimos', 'moneda']::text[]
               )
           )
       )
       OR (
           v_tiene_coste AND (
               pg_catalog.jsonb_typeof(a -> 'coste_previsto') <> 'object'
               OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
                   a -> 'coste_previsto',
                   ARRAY['centimos', 'moneda']::text[]
               )
           )
       )
       OR pg_catalog.jsonb_typeof(a -> 'porcentaje_jornada') <> 'number'
       OR (a ->> 'porcentaje_jornada')::numeric
            NOT BETWEEN 1 AND 10000
       OR v ->> 'resultado'
            NOT IN ('validada', 'no_requerida', 'rechazada') THEN
        RETURN NULL;
    END IF;

    v_prueba :=
        vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            'VEC-CT-ANALISIS-DERIVADO-O3-V1'
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            a ->> 'modalidad_clave'
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            a ->> 'categoria_ref'
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            a ->> 'grupo_subgrupo'
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            a ->> 'causa_clave'
        )
        || vec_contratacion_temporal.microsegundos_unix_analisis_v1(
            a #>> '{periodo,inicio}'
        )
        || vec_contratacion_temporal.microsegundos_unix_analisis_v1(
            a #>> '{periodo,fin}'
        )
        || pg_catalog.int8send(
            (a ->> 'porcentaje_jornada')::bigint
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            a #>> '{entrada_rc_esperada,referencia}'
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            a #>> '{entrada_rc_esperada,huella_sha256}'
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            v ->> 'resultado'
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            v ->> 'entrada_ref'
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            v ->> 'huella_entrada_sha256'
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            v ->> 'fuente_ref'
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            v ->> 'recibo_ref'
        )
        || vec_contratacion_temporal.microsegundos_unix_analisis_v1(
            v ->> 'validada_en'
        )
        || CASE WHEN v_rc_validada
                THEN pg_catalog.decode('01', 'hex')
                ELSE pg_catalog.decode('00', 'hex') END;
    IF v_rc_validada THEN
        v_prueba := v_prueba
          || vec_contratacion_temporal.microsegundos_unix_analisis_v1(
              v ->> 'fecha_rc'
          );
    END IF;
    v_prueba := v_prueba
      || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
          coalesce(v ->> 'numero', '')
      )
      || CASE WHEN v_rc_validada
              THEN pg_catalog.decode('01', 'hex')
              ELSE pg_catalog.decode('00', 'hex') END;
    IF v_rc_validada THEN
        v_prueba := v_prueba
          || pg_catalog.int8send(
              (v #>> '{importe,centimos}')::bigint
          )
          || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
              v #>> '{importe,moneda}'
          );
    END IF;
    v_prueba := v_prueba
      || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
          coalesce(v ->> 'documento_ref', '')
      )
      || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
          coalesce(v ->> 'motivo', '')
      )
      || CASE WHEN v_tiene_coste
              THEN pg_catalog.decode('01', 'hex')
              ELSE pg_catalog.decode('00', 'hex') END;
    IF v_tiene_coste THEN
        v_prueba := v_prueba
          || pg_catalog.int8send(
              (a #>> '{coste_previsto,centimos}')::bigint
          )
          || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
              a #>> '{coste_previsto,moneda}'
          )
          || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
              a ->> 'fuente_coste_ref'
          );
    END IF;
    RETURN pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex');
EXCEPTION
    WHEN data_exception OR datetime_field_overflow
      OR invalid_text_representation OR numeric_value_out_of_range THEN
        RETURN NULL;
END
$funcion$;

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
    IF vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v1(
           anterior
       ) IS DISTINCT FROM
       vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v1(
           p_agregado_actual
       )
       OR pg_catalog.jsonb_typeof(anterior -> 'actuaciones') <> 'array'
       OR pg_catalog.jsonb_typeof(siguiente -> 'actuaciones') <> 'array'
       OR pg_catalog.jsonb_typeof(siguiente -> 'analisis') <> 'object'
       OR vec_contratacion_temporal.huella_analisis_derivado_v1(
              siguiente -> 'analisis'
          ) IS DISTINCT FROM o ->> 'analisis_derivado_huella_sha256'
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
vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v1(
    jsonb
) FROM PUBLIC, vec_contratacion_temporal_ejecutor;

REVOKE ALL ON FUNCTION
vec_contratacion_temporal.huella_analisis_derivado_v1(jsonb)
FROM PUBLIC, vec_contratacion_temporal_ejecutor;

REVOKE ALL ON FUNCTION
vec_contratacion_temporal.transicion_confirmacion_analisis_valida_v1(
    jsonb, jsonb
) FROM PUBLIC, vec_contratacion_temporal_ejecutor;

COMMIT;
