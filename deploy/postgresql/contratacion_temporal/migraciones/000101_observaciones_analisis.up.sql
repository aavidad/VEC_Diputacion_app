\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000101',0));

-- Admite 'observaciones' opcional en el objeto analisis (string 1..4000 caracteres).
-- No altera el calculo de la huella ni las huellas existentes.

CREATE OR REPLACE FUNCTION
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
    v_tiene_observaciones boolean;
    v_rc_validada boolean;
BEGIN
    v_tiene_coste := pg_catalog.jsonb_exists(a, 'coste_previsto');
    v_tiene_observaciones := pg_catalog.jsonb_exists(a, 'observaciones');
    v_rc_validada := v ->> 'resultado' = 'validada';
    IF v_tiene_coste THEN
        v_claves := v_claves ||
            ARRAY['coste_previsto', 'fuente_coste_ref']::text[];
    END IF;
    IF v_tiene_observaciones THEN
        v_claves := v_claves || ARRAY['observaciones']::text[];
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
    IF v_tiene_observaciones THEN
        IF pg_catalog.jsonb_typeof(a -> 'observaciones') <> 'string'
           OR pg_catalog.length(a ->> 'observaciones') < 1
           OR pg_catalog.length(a ->> 'observaciones') > 4000 THEN
            RETURN NULL;
        END IF;
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

COMMIT;
