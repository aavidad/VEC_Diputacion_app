BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000013_barrera_reforzada_analisis', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.entrada_confirmacion_analisis_valida_v1(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.transicion_confirmacion_analisis_valida_v1(jsonb,jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.huella_contexto_recurso_analisis_v1(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.huella_analisis_derivado_v2(jsonb)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para barrera reforzada O3';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION
vec_contratacion_temporal.numero_entero_json_canonico_v2(
    p_valor jsonb,
    p_minimo numeric,
    p_maximo numeric
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_texto text;
    v_numero numeric;
BEGIN
    IF pg_catalog.jsonb_typeof(p_valor) <> 'number'
       OR p_minimo IS NULL OR p_maximo IS NULL
       OR p_minimo > p_maximo THEN
        RETURN false;
    END IF;
    v_texto := p_valor::text;
    IF v_texto !~ '^(0|-[1-9][0-9]*|[1-9][0-9]*)$' THEN
        RETURN false;
    END IF;
    v_numero := v_texto::numeric;
    RETURN v_numero BETWEEN p_minimo AND p_maximo;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR numeric_value_out_of_range THEN
        RETURN false;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.campos_texto_json_v2(
    p_objeto jsonb,
    p_campos text[]
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.count(*) = pg_catalog.cardinality(p_campos)
       AND pg_catalog.bool_and(
           pg_catalog.jsonb_typeof(p_objeto -> campo) = 'string'
       )
      FROM pg_catalog.unnest(p_campos) AS c(campo)
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.instante_utc_json_canonico_v2(
    p_valor jsonb,
    p_fecha_civil boolean
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_texto text;
    v_instante timestamptz;
BEGIN
    IF pg_catalog.jsonb_typeof(p_valor) <> 'string'
       OR p_fecha_civil IS NULL THEN
        RETURN false;
    END IF;
    v_texto := p_valor #>> '{}';
    IF p_fecha_civil THEN
        IF v_texto !~
           '^[0-9]{4}-[0-9]{2}-[0-9]{2}T00:00:00Z$' THEN
            RETURN false;
        END IF;
    ELSIF v_texto !~
       ('^[0-9]{4}-[0-9]{2}-[0-9]{2}T'
        || '[0-2][0-9]:[0-5][0-9]:[0-5][0-9]'
        || '([.][0-9]{0,5}[1-9])?Z$') THEN
        RETURN false;
    END IF;
    v_instante := v_texto::timestamptz;
    RETURN pg_catalog.isfinite(v_instante)
       AND extract(microseconds FROM v_instante) =
           pg_catalog.trunc(extract(microseconds FROM v_instante));
EXCEPTION
    WHEN data_exception OR datetime_field_overflow
      OR invalid_text_representation THEN
        RETURN false;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.texto_instante_utc_go_v2(p_instante text)
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_base text;
BEGIN
    v_base := pg_catalog.to_char(
        p_instante::timestamptz AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US'
    );
    v_base := pg_catalog.rtrim(v_base, '0');
    IF pg_catalog.right(v_base, 1) = '.' THEN
        v_base := pg_catalog.left(v_base, -1);
    END IF;
    RETURN v_base || 'Z';
EXCEPTION
    WHEN data_exception OR datetime_field_overflow
      OR invalid_text_representation THEN
        RETURN NULL;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(
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
    analisis jsonb;
    indice integer;
    cantidad integer;
BEGIN
    resultado := pg_catalog.jsonb_set(
        resultado, '{creado_en}',
        pg_catalog.to_jsonb(
            vec_contratacion_temporal.texto_instante_utc_go_v2(
                resultado ->> 'creado_en'
            )
        ), false
    );
    resultado := pg_catalog.jsonb_set(
        resultado, '{actualizado_en}',
        pg_catalog.to_jsonb(
            vec_contratacion_temporal.texto_instante_utc_go_v2(
                resultado ->> 'actualizado_en'
            )
        ), false
    );
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
            resultado, '{solicitud,documentos_adjuntos}',
            'null'::jsonb, false
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
                actuacion := pg_catalog.jsonb_set(
                    actuacion, '{realizada_en}',
                    pg_catalog.to_jsonb(
                        vec_contratacion_temporal.texto_instante_utc_go_v2(
                            actuacion ->> 'realizada_en'
                        )
                    ), false
                );
                resultado := pg_catalog.jsonb_set(
                    resultado,
                    ARRAY['actuaciones', indice::text],
                    actuacion,
                    false
                );
            END LOOP;
        END IF;
    END IF;
    IF pg_catalog.jsonb_typeof(resultado -> 'analisis') = 'object' THEN
        analisis := resultado -> 'analisis';
        analisis := pg_catalog.jsonb_set(
            analisis, '{validacion_rc,validada_en}',
            pg_catalog.to_jsonb(
                vec_contratacion_temporal.texto_instante_utc_go_v2(
                    analisis #>> '{validacion_rc,validada_en}'
                )
            ), false
        );
        resultado := pg_catalog.jsonb_set(
            resultado, '{analisis}', analisis, false
        );
    END IF;
    RETURN resultado;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.actuacion_analisis_valida_v2(a jsonb)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_claves text[] := ARRAY[
      'accion_clave', 'actor_ref', 'estado_destino', 'estado_origen',
      'fase_destino', 'fase_origen', 'realizada_en', 'recibo_ref',
      'secuencia', 'unidad_ref', 'version_expediente'
    ]::text[];
    v_documento jsonb;
BEGIN
    IF pg_catalog.jsonb_exists(a, 'observaciones') THEN
        v_claves := pg_catalog.array_append(v_claves, 'observaciones');
    END IF;
    IF pg_catalog.jsonb_exists(a, 'documentos_ref') THEN
        v_claves := pg_catalog.array_append(v_claves, 'documentos_ref');
    END IF;
    SELECT pg_catalog.array_agg(x ORDER BY x)
      INTO v_claves FROM pg_catalog.unnest(v_claves) AS c(x);
    IF NOT vec_contratacion_temporal.claves_json_exactas_v1(a, v_claves)
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           a -> 'secuencia', 1, 9007199254740991::numeric
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           a -> 'version_expediente', 1, 9007199254740991::numeric
       )
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
           a -> 'realizada_en', false
       )
       OR pg_catalog.jsonb_typeof(a -> 'accion_clave') <> 'string'
       OR pg_catalog.jsonb_typeof(a -> 'actor_ref') <> 'string'
       OR pg_catalog.jsonb_typeof(a -> 'unidad_ref') <> 'string'
       OR pg_catalog.jsonb_typeof(a -> 'recibo_ref') <> 'string'
       OR pg_catalog.jsonb_typeof(a -> 'fase_origen') <> 'string'
       OR pg_catalog.jsonb_typeof(a -> 'fase_destino') <> 'string'
       OR pg_catalog.jsonb_typeof(a -> 'estado_origen') <> 'string'
       OR pg_catalog.jsonb_typeof(a -> 'estado_destino') <> 'string'
       OR (
           pg_catalog.jsonb_exists(a, 'observaciones')
           AND pg_catalog.jsonb_typeof(a -> 'observaciones') <> 'string'
       )
       OR (
           pg_catalog.jsonb_exists(a, 'documentos_ref')
           AND pg_catalog.jsonb_typeof(a -> 'documentos_ref') <> 'array'
       ) THEN
        RETURN false;
    END IF;
    FOR v_documento IN
        SELECT e.v
          FROM pg_catalog.jsonb_array_elements(
              coalesce(a -> 'documentos_ref', '[]'::jsonb)
          ) AS e(v)
    LOOP
        IF pg_catalog.jsonb_typeof(v_documento) <> 'string' THEN
            RETURN false;
        END IF;
    END LOOP;
    RETURN true;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.huella_analisis_derivado_v2(a jsonb)
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v jsonb := a -> 'validacion_rc';
    v_vinculo jsonb := a -> 'actuacion_registro';
    v_prueba bytea;
    v_claves text[] := ARRAY[
      'actuacion_registro', 'categoria_ref', 'causa_clave',
      'entrada_rc_esperada', 'grupo_subgrupo', 'modalidad_clave',
      'periodo', 'porcentaje_jornada', 'validacion_rc'
    ]::text[];
    v_claves_validacion text[];
    v_tiene_coste boolean;
    v_rc_validada boolean;
BEGIN
    v_tiene_coste := pg_catalog.jsonb_exists(a, 'coste_previsto');
    v_rc_validada := v ->> 'resultado' = 'validada';
    IF v_tiene_coste THEN
        v_claves := v_claves ||
            ARRAY['coste_previsto', 'fuente_coste_ref']::text[];
    END IF;
    SELECT pg_catalog.array_agg(x ORDER BY x)
      INTO v_claves FROM pg_catalog.unnest(v_claves) AS c(x);
    v_claves_validacion := CASE WHEN v_rc_validada THEN ARRAY[
      'documento_ref', 'entrada_ref', 'fecha_rc', 'fuente_ref',
      'huella_entrada_sha256', 'importe', 'numero', 'recibo_ref',
      'resultado', 'validada_en'
    ]::text[] ELSE ARRAY[
      'entrada_ref', 'fuente_ref', 'huella_entrada_sha256',
      'motivo', 'recibo_ref', 'resultado', 'validada_en'
    ]::text[] END;
    IF NOT vec_contratacion_temporal.claves_json_exactas_v1(a, v_claves)
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           a -> 'periodo', ARRAY['fin', 'inicio']::text[]
       )
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           a -> 'entrada_rc_esperada',
           ARRAY['huella_sha256', 'referencia']::text[]
       )
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           v_vinculo, ARRAY[
             'accion_clave', 'fase_destino', 'recibo_ref', 'secuencia',
             'version_expediente'
           ]::text[]
       )
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           v, v_claves_validacion
       )
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
           a #> '{periodo,inicio}', true
       )
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
           a #> '{periodo,fin}', true
       )
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
           v -> 'validada_en', false
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           a -> 'porcentaje_jornada', 1, 10000
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           v_vinculo -> 'secuencia', 2, 9007199254740991::numeric
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           v_vinculo -> 'version_expediente',
           2, 9007199254740991::numeric
       )
       OR v ->> 'resultado'
            NOT IN ('validada', 'no_requerida', 'rechazada')
       OR pg_catalog.jsonb_typeof(a -> 'modalidad_clave') <> 'string'
       OR pg_catalog.jsonb_typeof(a -> 'categoria_ref') <> 'string'
       OR pg_catalog.jsonb_typeof(a -> 'grupo_subgrupo') <> 'string'
       OR pg_catalog.jsonb_typeof(a -> 'causa_clave') <> 'string'
       OR pg_catalog.jsonb_typeof(
              a #> '{entrada_rc_esperada,referencia}'
          ) <> 'string'
       OR pg_catalog.jsonb_typeof(
              a #> '{entrada_rc_esperada,huella_sha256}'
          ) <> 'string'
       OR pg_catalog.jsonb_typeof(v -> 'resultado') <> 'string'
       OR pg_catalog.jsonb_typeof(v -> 'entrada_ref') <> 'string'
       OR pg_catalog.jsonb_typeof(v -> 'huella_entrada_sha256') <> 'string'
       OR pg_catalog.jsonb_typeof(v -> 'fuente_ref') <> 'string'
       OR pg_catalog.jsonb_typeof(v -> 'recibo_ref') <> 'string'
       OR pg_catalog.jsonb_typeof(v_vinculo -> 'accion_clave') <> 'string'
       OR pg_catalog.jsonb_typeof(v_vinculo -> 'fase_destino') <> 'string'
       OR pg_catalog.jsonb_typeof(v_vinculo -> 'recibo_ref') <> 'string'
       OR (a #>> '{periodo,fin}')::timestamptz <
          (a #>> '{periodo,inicio}')::timestamptz THEN
        RETURN NULL;
    END IF;
    IF v_rc_validada THEN
        IF NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
               v -> 'fecha_rc', true
           )
           OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
               v -> 'importe', ARRAY['centimos', 'moneda']::text[]
           )
           OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
               v #> '{importe,centimos}',
               1, 922337203685477::numeric
           )
           OR pg_catalog.jsonb_typeof(v #> '{importe,moneda}') <> 'string'
           OR pg_catalog.jsonb_typeof(v -> 'numero') <> 'string'
           OR pg_catalog.jsonb_typeof(v -> 'documento_ref') <> 'string'
           OR (v ->> 'fecha_rc')::timestamptz >
              (v ->> 'validada_en')::timestamptz THEN
            RETURN NULL;
        END IF;
    ELSIF pg_catalog.jsonb_typeof(v -> 'motivo') <> 'string' THEN
        RETURN NULL;
    END IF;
    IF v_tiene_coste THEN
        IF NOT vec_contratacion_temporal.claves_json_exactas_v1(
               a -> 'coste_previsto',
               ARRAY['centimos', 'moneda']::text[]
           )
           OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
               a #> '{coste_previsto,centimos}',
               1, 922337203685477::numeric
           )
           OR pg_catalog.jsonb_typeof(
                  a #> '{coste_previsto,moneda}'
              ) <> 'string'
           OR pg_catalog.jsonb_typeof(a -> 'fuente_coste_ref') <> 'string' THEN
            RETURN NULL;
        END IF;
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
        || pg_catalog.int8send((a ->> 'porcentaje_jornada')::bigint)
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
        || CASE WHEN v_rc_validada THEN '\x01'::bytea ELSE '\x00'::bytea END;
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
      || CASE WHEN v_rc_validada THEN '\x01'::bytea ELSE '\x00'::bytea END;
    IF v_rc_validada THEN
        v_prueba := v_prueba
          || pg_catalog.int8send((v #>> '{importe,centimos}')::bigint)
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
      || CASE WHEN v_tiene_coste THEN '\x01'::bytea ELSE '\x00'::bytea END;
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
vec_contratacion_temporal.expediente_analisis_valido_v2(
    e jsonb,
    p_exige_analisis boolean
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    s jsonb := e -> 'solicitud';
    rc jsonb := e #> '{solicitud,rc}';
    v_claves text[] := ARRAY[
      'actuaciones', 'actualizado_en', 'creado_en', 'estado_actual',
      'fase_actual', 'flujo', 'numero_visible', 'organizacion_ref',
      'referencia', 'solicitud', 'version'
    ]::text[];
    v_claves_solicitud text[] := ARRAY[
      'categoria_ref', 'centro_ref', 'contacto_ref', 'detalle',
      'documentos_adjuntos', 'grupo_subgrupo', 'motivo_clave',
      'periodo', 'rc'
    ]::text[];
    v_claves_rc text[];
    v_actuacion jsonb;
    v_documento jsonb;
BEGIN
    IF p_exige_analisis IS NULL
       OR pg_catalog.jsonb_typeof(e) <> 'object'
       OR pg_catalog.jsonb_exists(e, 'via_cobertura')
       OR pg_catalog.jsonb_exists(e, 'asignacion') THEN
        RETURN false;
    END IF;
    IF p_exige_analisis THEN
        v_claves := pg_catalog.array_append(v_claves, 'analisis');
        SELECT pg_catalog.array_agg(x ORDER BY x)
          INTO v_claves
          FROM pg_catalog.unnest(v_claves) AS c(x);
    ELSIF pg_catalog.jsonb_exists(e, 'analisis') THEN
        RETURN false;
    END IF;
    IF pg_catalog.jsonb_exists(s, 'observaciones') THEN
        v_claves_solicitud :=
            pg_catalog.array_append(v_claves_solicitud, 'observaciones');
        SELECT pg_catalog.array_agg(x ORDER BY x)
          INTO v_claves_solicitud
          FROM pg_catalog.unnest(v_claves_solicitud) AS c(x);
    END IF;
    v_claves_rc := CASE WHEN rc ->> 'existe' = 'true' THEN ARRAY[
      'documento_ref', 'existe', 'fecha', 'importe', 'numero'
    ]::text[] ELSE ARRAY['existe', 'fecha', 'importe']::text[] END;
    IF NOT vec_contratacion_temporal.claves_json_exactas_v1(e, v_claves)
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           e -> 'flujo',
           ARRAY['definicion_ref', 'huella_sha256', 'version']::text[]
       )
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           s, v_claves_solicitud
       )
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           s -> 'periodo', ARRAY['fin', 'inicio']::text[]
       )
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           rc, v_claves_rc
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           e -> 'version', 1, 9007199254740991::numeric
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           e #> '{flujo,version}', 1, 9007199254740991::numeric
       )
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
           e -> 'creado_en', false
       )
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
           e -> 'actualizado_en', false
       )
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
           s #> '{periodo,inicio}', true
       )
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
           s #> '{periodo,fin}', true
       )
       OR pg_catalog.jsonb_typeof(e -> 'referencia') <> 'string'
       OR pg_catalog.jsonb_typeof(e -> 'organizacion_ref') <> 'string'
       OR pg_catalog.jsonb_typeof(e -> 'numero_visible') <> 'string'
       OR pg_catalog.jsonb_typeof(e -> 'fase_actual') <> 'string'
       OR pg_catalog.jsonb_typeof(e -> 'estado_actual') <> 'string'
       OR pg_catalog.jsonb_typeof(e #> '{flujo,definicion_ref}') <> 'string'
       OR pg_catalog.jsonb_typeof(e #> '{flujo,huella_sha256}') <> 'string'
       OR pg_catalog.jsonb_typeof(s -> 'centro_ref') <> 'string'
       OR pg_catalog.jsonb_typeof(s -> 'contacto_ref') <> 'string'
       OR pg_catalog.jsonb_typeof(s -> 'categoria_ref') <> 'string'
       OR pg_catalog.jsonb_typeof(s -> 'grupo_subgrupo') <> 'string'
       OR pg_catalog.jsonb_typeof(s -> 'motivo_clave') <> 'string'
       OR pg_catalog.jsonb_typeof(s -> 'detalle') <> 'string'
       OR pg_catalog.jsonb_typeof(rc -> 'existe') <> 'boolean'
       OR pg_catalog.jsonb_typeof(e -> 'actuaciones') <> 'array'
       OR pg_catalog.jsonb_typeof(s -> 'documentos_adjuntos')
            NOT IN ('array', 'null')
       OR (e ->> 'actualizado_en')::timestamptz <
          (e ->> 'creado_en')::timestamptz
       OR (s #>> '{periodo,fin}')::timestamptz <
          (s #>> '{periodo,inicio}')::timestamptz THEN
        RETURN false;
    END IF;
    IF p_exige_analisis
       AND vec_contratacion_temporal.huella_analisis_derivado_v2(
               e -> 'analisis'
           ) IS NULL THEN
        RETURN false;
    END IF;
    IF rc ->> 'existe' = 'true' THEN
        IF NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
               rc -> 'fecha', true
           )
           OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
               rc -> 'importe', ARRAY['centimos', 'moneda']::text[]
           )
           OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
               rc #> '{importe,centimos}', 1,
               9223372036854775807::numeric
           )
           OR pg_catalog.jsonb_typeof(rc -> 'numero') <> 'string'
           OR pg_catalog.jsonb_typeof(rc -> 'documento_ref') <> 'string'
           OR pg_catalog.jsonb_typeof(rc #> '{importe,moneda}') <> 'string' THEN
            RETURN false;
        END IF;
    ELSE
        IF rc ->> 'fecha' <> '0001-01-01T00:00:00Z'
           OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
               rc -> 'importe', ARRAY['centimos', 'moneda']::text[]
           )
           OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
               rc #> '{importe,centimos}', 0, 0
           )
           OR rc #>> '{importe,moneda}' <> '' THEN
            RETURN false;
        END IF;
    END IF;
    FOR v_documento IN
        SELECT d.v
          FROM pg_catalog.jsonb_array_elements(
              CASE
                WHEN s -> 'documentos_adjuntos' = 'null'::jsonb
                THEN '[]'::jsonb
                ELSE s -> 'documentos_adjuntos'
              END
          ) AS d(v)
    LOOP
        IF pg_catalog.jsonb_typeof(v_documento) <> 'string' THEN
            RETURN false;
        END IF;
    END LOOP;
    FOR v_actuacion IN
        SELECT a.v
          FROM pg_catalog.jsonb_array_elements(e -> 'actuaciones') AS a(v)
    LOOP
        IF vec_contratacion_temporal.actuacion_analisis_valida_v2(
               v_actuacion
           ) IS NOT TRUE THEN
            RETURN false;
        END IF;
    END LOOP;
    RETURN true;
EXCEPTION
    WHEN data_exception OR datetime_field_overflow
      OR invalid_text_representation OR numeric_value_out_of_range THEN
        RETURN false;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.numero_entero_json_canonico_v2(
        jsonb, numeric, numeric
    ),
    vec_contratacion_temporal.campos_texto_json_v2(jsonb, text[]),
    vec_contratacion_temporal.instante_utc_json_canonico_v2(jsonb, boolean),
    vec_contratacion_temporal.texto_instante_utc_go_v2(text),
    vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(jsonb),
    vec_contratacion_temporal.actuacion_analisis_valida_v2(jsonb),
    vec_contratacion_temporal.huella_analisis_derivado_v2(jsonb),
    vec_contratacion_temporal.expediente_analisis_valido_v2(jsonb, boolean)
FROM PUBLIC, vec_contratacion_temporal_ejecutor;

COMMIT;
