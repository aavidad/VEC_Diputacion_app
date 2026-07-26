BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000014_frontera_durable_analisis_v2', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.huella_analisis_derivado_v2(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.confirmar_operacion_analisis_v1(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.confirmar_operacion_analisis_v2(jsonb)'
       ) IS NOT NULL
       OR EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.confirmacion_operacion_analisis
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para frontera durable O3 v2';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION
vec_contratacion_temporal.huella_contexto_recurso_analisis_v2(o jsonb)
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    p jsonb := o -> 'politica';
    v_analisis_huella text;
    v_json text;
BEGIN
    v_analisis_huella :=
        vec_contratacion_temporal.huella_analisis_derivado_v2(
            o #> '{expediente_siguiente,analisis}'
        );
    IF pg_catalog.jsonb_typeof(o) <> 'object'
       OR pg_catalog.jsonb_typeof(p) <> 'object'
       OR v_analisis_huella IS NULL THEN
        RETURN NULL;
    END IF;
    v_json :=
        '{"ambitos":{"estado_previo":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p ->> 'estado_previo'
        ) || ',"expediente_ref":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            o ->> 'expediente_ref'
        ) || ',"fase_previa":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p ->> 'fase_previa'
        ) || ',"organizacion_ref":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            o ->> 'organizacion_ref'
        ) || '},"atributos":{"analisis_derivado_huella_sha256":' ||
        vec_contratacion_temporal.texto_json_go_v1(v_analisis_huella) ||
        ',"artefacto_analisis_huella_sha256":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            o ->> 'artefacto_huella_sha256'
        ) || ',"artefacto_analisis_ref":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            o ->> 'artefacto_ref'
        ) || ',"conjunto_fuentes_huella_sha256":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            o #>> '{fuentes,conjunto_huella_sha256}'
        ) || ',"exige_actor_distinto":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            CASE WHEN (p ->> 'exige_actor_distinto')::boolean
                 THEN 'true' ELSE 'false' END
        ) || ',"huella_semantica_hmac":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            o ->> 'huella_semantica_hmac'
        ) || ',"motivo_rectificacion_clave":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p ->> 'motivo_rectificacion_clave'
        ) || ',"operacion":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            o ->> 'operacion'
        ) || ',"politica_huella_sha256":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p ->> 'huella_sha256'
        ) || ',"politica_ref":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p ->> 'definicion_ref'
        ) || ',"politica_version":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p ->> 'version'
        ) || ',"unidad_politica_ref":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            p ->> 'unidad_ref'
        ) || ',"version_expediente_esperada":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            o ->> 'version_anterior'
        ) || '}}';
    RETURN pg_catalog.encode(
        pg_catalog.sha256(pg_catalog.convert_to(v_json, 'UTF8')), 'hex'
    );
EXCEPTION
    WHEN data_exception OR invalid_text_representation THEN
        RETURN NULL;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.entrada_confirmacion_analisis_valida_v2(o jsonb)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    p jsonb := o -> 'politica';
    f jsonb := o -> 'fuentes';
    a jsonb := o -> 'autorizacion';
    v_base jsonb;
    v_fuente jsonb;
    v_alias jsonb;
BEGIN
    IF NOT vec_contratacion_temporal.claves_json_exactas_v1(
           o, ARRAY[
             'actor_ref', 'actuacion', 'aliases_consulta',
             'ambito_consulta_hmac', 'ambito_raiz_hmac',
             'artefacto_huella_sha256', 'artefacto_ref', 'autorizacion',
             'esquema', 'expediente_anterior', 'expediente_ref',
             'expediente_siguiente', 'fuentes', 'huella_consulta_hmac',
             'huella_semantica_hmac', 'operacion', 'organizacion_ref',
             'perfil_ref', 'politica', 'recibo_ref', 'reserva_ref',
             'version_anterior'
           ]::text[]
       )
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           p, ARRAY[
             'accion', 'definicion_ref', 'estado_previo',
             'exige_actor_distinto', 'fase_previa', 'finalidad',
             'huella_sha256', 'motivo_rectificacion_clave',
             'unidad_ref', 'version'
           ]::text[]
       )
       OR NOT vec_contratacion_temporal.campos_texto_json_v2(
           o, ARRAY[
             'actor_ref', 'ambito_consulta_hmac', 'ambito_raiz_hmac',
             'artefacto_huella_sha256', 'artefacto_ref', 'esquema',
             'expediente_ref', 'huella_consulta_hmac',
             'huella_semantica_hmac', 'operacion', 'organizacion_ref',
             'perfil_ref', 'recibo_ref', 'reserva_ref'
           ]::text[]
       )
       OR NOT vec_contratacion_temporal.campos_texto_json_v2(
           p, ARRAY[
             'accion', 'definicion_ref', 'estado_previo', 'fase_previa',
             'finalidad', 'huella_sha256', 'motivo_rectificacion_clave',
             'unidad_ref'
           ]::text[]
       )
       OR NOT vec_contratacion_temporal.campos_texto_json_v2(
           f, ARRAY[
             'conjunto_huella_sha256', 'prueba_canonica_hex'
           ]::text[]
       )
       OR NOT vec_contratacion_temporal.campos_texto_json_v2(
           a, ARRAY[
             'accion', 'contexto_recurso_huella_sha256',
             'decision_canonica_hex', 'decision_huella_sha256',
             'decision_ref', 'finalidad', 'motivo_canonico_hex',
             'perfil_activo_ref', 'principal_id', 'recurso_ref'
           ]::text[]
       )
       OR pg_catalog.jsonb_typeof(p -> 'motivo_rectificacion_clave')
            <> 'string'
       OR (
           o ->> 'operacion' = 'registrar'
           AND p ->> 'motivo_rectificacion_clave' <> 'no_aplica'
       )
       OR (
           o ->> 'operacion' = 'rectificar'
           AND coalesce(p ->> 'motivo_rectificacion_clave', '')
               !~ '^[a-z0-9][a-z0-9._-]{2,119}$'
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           o -> 'version_anterior', 1, 9007199254740990::numeric
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p -> 'version', 1, 9007199254740991::numeric
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           a -> 'persona_version', 1, 9007199254740991::numeric
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           a -> 'perfil_version', 1, 9007199254740991::numeric
       )
       OR vec_contratacion_temporal.actuacion_analisis_valida_v2(
              o -> 'actuacion'
          ) IS NOT TRUE
       OR vec_contratacion_temporal.expediente_analisis_valido_v2(
              o -> 'expediente_anterior',
              o ->> 'operacion' = 'rectificar'
          ) IS NOT TRUE
       OR vec_contratacion_temporal.expediente_analisis_valido_v2(
              o -> 'expediente_siguiente', true
          ) IS NOT TRUE THEN
        RETURN false;
    END IF;
    v_base := pg_catalog.jsonb_set(
        o, '{politica}', p - 'motivo_rectificacion_clave', false
    );
    IF vec_contratacion_temporal.entrada_confirmacion_analisis_valida_v1(
           v_base
       ) IS NOT TRUE THEN
        RETURN false;
    END IF;
    FOR v_alias IN
        SELECT e.v
          FROM pg_catalog.jsonb_array_elements(o -> 'aliases_consulta') e(v)
    LOOP
        IF NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
               v_alias -> 'generacion', 1, 999999999
           )
           OR NOT vec_contratacion_temporal.campos_texto_json_v2(
               v_alias, ARRAY['ambito_hmac']::text[]
           ) THEN
            RETURN false;
        END IF;
    END LOOP;
    FOR v_fuente IN
        SELECT e.v
          FROM pg_catalog.jsonb_array_elements(
              CASE WHEN f -> 'coste' = 'null'::jsonb
                   THEN pg_catalog.jsonb_build_array(f -> 'rc')
                   ELSE pg_catalog.jsonb_build_array(
                       f -> 'rc', f -> 'coste'
                   ) END
          ) e(v)
    LOOP
        IF NOT vec_contratacion_temporal.campos_texto_json_v2(
               v_fuente, ARRAY[
                 'autoridad_ref', 'emitida_en', 'material_huella_sha256',
                 'peticion_ref', 'recibo_respuesta_ref',
                 'respuesta_huella_sha256', 'sello_respuesta_hmac', 'tipo',
                 'valida_hasta', 'verificada_en', 'verificador_ref'
               ]::text[]
           )
           OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
               v_fuente -> 'generacion', 1, 4294967295::numeric
           )
           OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
               v_fuente -> 'emitida_en', false
           )
           OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
               v_fuente -> 'valida_hasta', false
           )
           OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
               v_fuente -> 'verificada_en', false
           )
           OR (
               v_fuente -> 'publicacion' <> 'null'::jsonb
               AND (
                   NOT vec_contratacion_temporal.campos_texto_json_v2(
                       v_fuente -> 'publicacion', ARRAY[
                         'huella_solicitud_sha256', 'publicacion_ref',
                         'publicador_ref', 'recibo_verificacion_ref',
                         'verificada_en'
                       ]::text[]
                   )
                   OR NOT vec_contratacion_temporal
                       .instante_utc_json_canonico_v2(
                           v_fuente #> '{publicacion,verificada_en}', false
                       )
               )
           ) THEN
            RETURN false;
        END IF;
    END LOOP;
    RETURN vec_contratacion_temporal.huella_contexto_recurso_analisis_v2(o)
           IS NOT NULL;
EXCEPTION
    WHEN data_exception OR datetime_field_overflow
      OR invalid_text_representation OR numeric_value_out_of_range THEN
        RETURN false;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.transicion_confirmacion_analisis_valida_v2(
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
    v_analisis_huella text;
BEGIN
    IF vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(
           o -> 'expediente_anterior'
       ) IS DISTINCT FROM
       vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(
           p_agregado_actual
       ) THEN
        RETURN false;
    END IF;
    IF vec_contratacion_temporal.transicion_confirmacion_analisis_valida_v1(
           o, p_agregado_actual
       ) IS NOT TRUE
       OR (o #>> '{actuacion,realizada_en}')::timestamptz <
          (p_agregado_actual ->> 'actualizado_en')::timestamptz
       OR (o #>> '{expediente_siguiente,analisis,validacion_rc,validada_en}')
             ::timestamptz >
          (o #>> '{actuacion,realizada_en}')::timestamptz THEN
        RETURN false;
    END IF;
    v_analisis_huella :=
        vec_contratacion_temporal.huella_analisis_derivado_v2(
            o #> '{expediente_siguiente,analisis}'
        );
    RETURN v_analisis_huella IS NOT NULL
       AND o #>> '{politica,unidad_ref}' =
           o #>> '{actuacion,unidad_ref}'
       AND (
           o ->> 'operacion' = 'registrar'
           OR o #>> '{politica,motivo_rectificacion_clave}' =
              o #>> '{actuacion,observaciones}'
       );
EXCEPTION
    WHEN data_exception OR datetime_field_overflow
      OR invalid_text_representation OR numeric_value_out_of_range THEN
        RETURN false;
END
$funcion$;

-- Conserva la semántica publicada de v1 y corrige exclusivamente la
-- equivalencia de representación que O2 y encoding/json dan a valores vacíos.
CREATE OR REPLACE FUNCTION
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
    IF vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(
           anterior
       ) IS DISTINCT FROM
       vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(
           p_agregado_actual
       )
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

-- La implementación transaccional v1 invoca este nombre. Se refuerza por una
-- migración posterior sin reescribir 000009 y admite la proyección interna
-- sin el motivo solo porque la envoltura v2 ya lo validó y selló.
CREATE OR REPLACE FUNCTION
vec_contratacion_temporal.huella_contexto_recurso_analisis_v1(
    p_operacion jsonb
)
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    o jsonb := p_operacion;
    v_operacion jsonb := p_operacion;
    v_motivo text;
BEGIN
    IF NOT pg_catalog.jsonb_exists(
           o -> 'politica', 'motivo_rectificacion_clave'
       ) THEN
        v_motivo := CASE
            WHEN o ->> 'operacion' = 'registrar' THEN 'no_aplica'
            ELSE o #>> '{actuacion,observaciones}'
        END;
        v_operacion := pg_catalog.jsonb_set(
            o, '{politica,motivo_rectificacion_clave}',
            pg_catalog.to_jsonb(v_motivo), true
        );
    END IF;
    RETURN vec_contratacion_temporal
        .huella_contexto_recurso_analisis_v2(v_operacion);
END
$funcion$;

CREATE TABLE
vec_contratacion_temporal.vinculo_replay_operacion_analisis_v2 (
    ambito_raiz_hmac text PRIMARY KEY
        REFERENCES vec_contratacion_temporal
            .confirmacion_operacion_analisis(ambito_raiz_hmac)
        ON DELETE RESTRICT,
    peticion_huella_sha256 text NOT NULL,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    creada_en timestamptz(6) NOT NULL,
    CONSTRAINT vinculo_replay_peticion_huella_formato CHECK (
        peticion_huella_sha256 ~ '^[0-9a-f]{64}$'
    ),
    CONSTRAINT vinculo_replay_actor_no_vacio CHECK (
        pg_catalog.length(actor_ref) BETWEEN 3 AND 160
    ),
    CONSTRAINT vinculo_replay_perfil_no_vacio CHECK (
        pg_catalog.length(perfil_ref) BETWEEN 3 AND 160
    )
);
ALTER TABLE vec_contratacion_temporal.vinculo_replay_operacion_analisis_v2
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.vinculo_replay_operacion_analisis_v2
    FORCE ROW LEVEL SECURITY;
CREATE POLICY vinculo_replay_operacion_analisis_v2_propietario
    ON vec_contratacion_temporal.vinculo_replay_operacion_analisis_v2
    TO vec_contratacion_temporal_propietario
    USING (true) WITH CHECK (true);
REVOKE ALL ON TABLE
    vec_contratacion_temporal.vinculo_replay_operacion_analisis_v2
FROM PUBLIC, vec_contratacion_temporal_ejecutor;

-- El cuerpo SQL estándar registra una dependencia de catálogo sobre v1. Así
-- 000012 no puede retirarse mientras esta frontera siga instalada.
CREATE FUNCTION
vec_contratacion_temporal.ejecutar_confirmacion_analisis_base_v2(o jsonb)
RETURNS TABLE (recibo_json jsonb)
LANGUAGE sql
VOLATILE
SECURITY INVOKER
SET search_path = pg_catalog
BEGIN ATOMIC
    SELECT c.recibo_json
      FROM vec_contratacion_temporal.confirmar_operacion_analisis_v1(o) c;
END;

CREATE FUNCTION
vec_contratacion_temporal.confirmar_operacion_analisis_v2(o jsonb)
RETURNS TABLE (recibo_json jsonb)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_huella text;
    v_vinculo record;
    v_actual jsonb;
    v_base jsonb;
    v_recibo jsonb;
    v_ahora timestamptz(6);
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_propietario', 'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_migrador', 'MEMBER'
       )
       OR pg_catalog.current_setting('transaction_isolation') <>
          'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR o IS NULL
       OR vec_contratacion_temporal.entrada_confirmacion_analisis_valida_v2(o)
          IS NOT TRUE THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'frontera de confirmación O3 v2 no autorizada';
    END IF;
    v_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(o::text, 'UTF8')
    ), 'hex');
    SELECT *
      INTO v_vinculo
      FROM vec_contratacion_temporal.vinculo_replay_operacion_analisis_v2 v
     WHERE v.ambito_raiz_hmac = o ->> 'ambito_raiz_hmac';
    v_base := pg_catalog.jsonb_set(
        o, '{politica}',
        (o -> 'politica') - 'motivo_rectificacion_clave', false
    );
    IF FOUND THEN
        IF v_vinculo.peticion_huella_sha256 IS DISTINCT FROM v_huella
           OR v_vinculo.actor_ref IS DISTINCT FROM o ->> 'actor_ref'
           OR v_vinculo.perfil_ref IS DISTINCT FROM o ->> 'perfil_ref' THEN
            RAISE EXCEPTION USING
                ERRCODE = '23505',
                MESSAGE = 'replay O3 v2 divergente';
        END IF;
        RETURN QUERY
        SELECT c.recibo_json
          FROM vec_contratacion_temporal
               .ejecutar_confirmacion_analisis_base_v2(
              v_base
          ) c;
        RETURN;
    END IF;
    v_ahora := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    IF (o #>> '{actuacion,realizada_en}')::timestamptz > v_ahora THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'actuación O3 futura';
    END IF;
    SELECT version.agregado_json
      INTO STRICT v_actual
      FROM vec_contratacion_temporal.expediente_integral_actual actual
      JOIN vec_contratacion_temporal.expediente_version_integral version
        USING (expediente_ref, version)
     WHERE actual.expediente_ref = o ->> 'expediente_ref';
    IF vec_contratacion_temporal.transicion_confirmacion_analisis_valida_v2(
           o, v_actual
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING
            ERRCODE = '40001',
            MESSAGE = 'transición O3 v2 en conflicto';
    END IF;
    SELECT c.recibo_json
      INTO STRICT v_recibo
      FROM vec_contratacion_temporal
           .ejecutar_confirmacion_analisis_base_v2(
          v_base
      ) c;
    INSERT INTO
    vec_contratacion_temporal.vinculo_replay_operacion_analisis_v2
    VALUES (
        o ->> 'ambito_raiz_hmac', v_huella,
        o ->> 'actor_ref', o ->> 'perfil_ref', v_ahora
    )
    ON CONFLICT (ambito_raiz_hmac) DO NOTHING;
    SELECT *
      INTO STRICT v_vinculo
      FROM vec_contratacion_temporal.vinculo_replay_operacion_analisis_v2 v
     WHERE v.ambito_raiz_hmac = o ->> 'ambito_raiz_hmac';
    IF v_vinculo.peticion_huella_sha256 IS DISTINCT FROM v_huella
       OR v_vinculo.actor_ref IS DISTINCT FROM o ->> 'actor_ref'
       OR v_vinculo.perfil_ref IS DISTINCT FROM o ->> 'perfil_ref' THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'carrera de replay O3 v2 divergente';
    END IF;
    RETURN QUERY SELECT v_recibo;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.huella_contexto_recurso_analisis_v2(jsonb),
    vec_contratacion_temporal.entrada_confirmacion_analisis_valida_v2(jsonb),
    vec_contratacion_temporal.transicion_confirmacion_analisis_valida_v2(
        jsonb, jsonb
    ),
    vec_contratacion_temporal.huella_contexto_recurso_analisis_v1(jsonb),
    vec_contratacion_temporal.confirmar_operacion_analisis_v1(jsonb),
    vec_contratacion_temporal.ejecutar_confirmacion_analisis_base_v2(jsonb),
    vec_contratacion_temporal.confirmar_operacion_analisis_v2(jsonb)
FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION
vec_contratacion_temporal.confirmar_operacion_analisis_v1(jsonb)
FROM vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION
vec_contratacion_temporal.confirmar_operacion_analisis_v2(jsonb)
TO vec_contratacion_temporal_ejecutor;

COMMIT;
