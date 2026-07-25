BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000010_validadores_confirmacion_analisis', 0
    )
);

CREATE FUNCTION vec_contratacion_temporal.claves_json_exactas_v1(
    p_objeto jsonb,
    p_claves text[]
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.jsonb_typeof(p_objeto) = 'object'
       AND (
           SELECT pg_catalog.array_agg(clave ORDER BY clave)
             FROM pg_catalog.jsonb_object_keys(p_objeto) AS c(clave)
       ) IS NOT DISTINCT FROM p_claves
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.encuadrar_binario_analisis_v1(p_valor text)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.int4send(
               pg_catalog.octet_length(
                   pg_catalog.convert_to(p_valor, 'UTF8')
               )
           )
        || pg_catalog.convert_to(p_valor, 'UTF8')
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.microsegundos_unix_analisis_v1(p_instante text)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_instante timestamptz;
    v_microsegundos bigint;
BEGIN
    v_instante := p_instante::timestamptz;
    v_microsegundos := pg_catalog.floor(
        extract(epoch FROM v_instante) * 1000000
    )::bigint;
    RETURN pg_catalog.int8send(v_microsegundos);
EXCEPTION
    WHEN data_exception OR datetime_field_overflow
      OR invalid_text_representation THEN
        RETURN NULL;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.reconstruir_fuente_analisis_v1(
    p_fuente jsonb,
    p_organizacion_ref text,
    p_expediente_ref text,
    p_version numeric
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_publicacion jsonb := p_fuente -> 'publicacion';
    v_prueba bytea;
BEGIN
    v_prueba :=
        vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            p_fuente ->> 'tipo'
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            p_fuente ->> 'peticion_ref'
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            p_organizacion_ref
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            p_expediente_ref
        )
        || pg_catalog.int8send(p_version::bigint)
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            p_fuente ->> 'respuesta_huella_sha256'
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            p_fuente ->> 'autoridad_ref'
        )
        || pg_catalog.int8send((p_fuente ->> 'generacion')::bigint)
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            p_fuente ->> 'recibo_respuesta_ref'
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            p_fuente ->> 'sello_respuesta_hmac'
        )
        || vec_contratacion_temporal.microsegundos_unix_analisis_v1(
            p_fuente ->> 'emitida_en'
        )
        || vec_contratacion_temporal.microsegundos_unix_analisis_v1(
            p_fuente ->> 'valida_hasta'
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            p_fuente ->> 'verificador_ref'
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            p_fuente ->> 'material_huella_sha256'
        )
        || vec_contratacion_temporal.microsegundos_unix_analisis_v1(
            p_fuente ->> 'verificada_en'
        )
        || CASE WHEN v_publicacion = 'null'::jsonb
                THEN '\x00'::bytea ELSE '\x01'::bytea END;
    IF v_publicacion <> 'null'::jsonb THEN
        v_prueba := v_prueba
            || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
                v_publicacion ->> 'publicador_ref'
            )
            || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
                v_publicacion ->> 'publicacion_ref'
            )
            || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
                v_publicacion ->> 'recibo_verificacion_ref'
            )
            || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
                v_publicacion ->> 'huella_solicitud_sha256'
            )
            || vec_contratacion_temporal.microsegundos_unix_analisis_v1(
                v_publicacion ->> 'verificada_en'
            );
    END IF;
    RETURN v_prueba;
EXCEPTION
    WHEN data_exception OR datetime_field_overflow
      OR invalid_text_representation OR numeric_value_out_of_range THEN
        RETURN NULL;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.reconstruir_prueba_fuentes_analisis_v1(
    p_operacion jsonb
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    f jsonb := p_operacion -> 'fuentes';
    v_prueba bytea;
BEGIN
    v_prueba :=
        vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            'VEC-CT-CONSUMO-CONJUNTO-FUENTES-O3-V1'
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            p_operacion ->> 'artefacto_ref'
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            p_operacion ->> 'organizacion_ref'
        )
        || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
            p_operacion ->> 'expediente_ref'
        )
        || pg_catalog.int8send(
            (p_operacion ->> 'version_anterior')::bigint
        )
        || vec_contratacion_temporal.reconstruir_fuente_analisis_v1(
            f -> 'rc',
            p_operacion ->> 'organizacion_ref',
            p_operacion ->> 'expediente_ref',
            (p_operacion ->> 'version_anterior')::numeric
        )
        || CASE WHEN f -> 'coste' = 'null'::jsonb
                THEN '\x00'::bytea ELSE '\x01'::bytea END;
    IF f -> 'coste' <> 'null'::jsonb THEN
        v_prueba := v_prueba
            || vec_contratacion_temporal.reconstruir_fuente_analisis_v1(
                f -> 'coste',
                p_operacion ->> 'organizacion_ref',
                p_operacion ->> 'expediente_ref',
                (p_operacion ->> 'version_anterior')::numeric
            );
    END IF;
    RETURN v_prueba;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR numeric_value_out_of_range THEN
        RETURN NULL;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.entrada_confirmacion_analisis_valida_v1(
    o jsonb
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    f jsonb := o -> 'fuentes';
    a jsonb := o -> 'autorizacion';
    p jsonb := o -> 'politica';
    v_fuente jsonb;
    v_alias jsonb;
    v_generaciones numeric[];
    v_ambitos text[];
BEGIN
    IF pg_catalog.pg_column_size(o) > 2097152
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           o, ARRAY[
             'actor_ref', 'actuacion', 'aliases_consulta',
             'ambito_consulta_hmac', 'ambito_raiz_hmac',
             'analisis_derivado_huella_sha256',
             'artefacto_huella_sha256', 'artefacto_ref', 'autorizacion',
             'esquema', 'expediente_anterior', 'expediente_ref',
             'expediente_siguiente', 'fuentes', 'huella_consulta_hmac',
             'huella_semantica_hmac', 'operacion', 'organizacion_ref',
             'perfil_ref', 'politica', 'recibo_ref', 'reserva_ref',
             'version_anterior'
           ]::text[]
       )
       OR o ->> 'esquema' <>
          'vec.contratacion-temporal.confirmar-operacion-analisis.v1'
       OR o ->> 'operacion' NOT IN ('registrar', 'rectificar')
       OR pg_catalog.jsonb_typeof(o -> 'version_anterior') <> 'number'
       OR (o ->> 'version_anterior')::numeric
            NOT BETWEEN 1 AND 9007199254740990::numeric
       OR pg_catalog.jsonb_typeof(o -> 'expediente_anterior') <> 'object'
       OR pg_catalog.jsonb_typeof(o -> 'expediente_siguiente') <> 'object'
       OR pg_catalog.jsonb_typeof(o -> 'actuacion') <> 'object'
       OR pg_catalog.jsonb_typeof(o -> 'aliases_consulta') <> 'array'
       OR pg_catalog.jsonb_array_length(o -> 'aliases_consulta')
            NOT BETWEEN 1 AND 4 THEN
        RETURN false;
    END IF;
    IF coalesce(o ->> 'organizacion_ref', '')
            !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(o ->> 'expediente_ref', '')
            !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(o ->> 'actor_ref', '')
            !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(o ->> 'perfil_ref', '')
            !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(o ->> 'artefacto_ref', '')
            !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(o ->> 'artefacto_huella_sha256', '')
            !~ '^[0-9a-f]{64}$'
       OR coalesce(o ->> 'analisis_derivado_huella_sha256', '')
            !~ '^[0-9a-f]{64}$'
       OR coalesce(o ->> 'ambito_raiz_hmac', '') !~
          ('^hmac-sha256:vec[.]contratacion-temporal[.]analisis[.]'
           || 'ambito-idempotencia/v[1-9][0-9]{0,8}:[0-9a-f]{64}$')
       OR coalesce(o ->> 'huella_semantica_hmac', '') !~
          ('^hmac-sha256:vec[.]contratacion-temporal[.]analisis[.]'
           || 'huella-semantica/v[1-9][0-9]{0,8}:[0-9a-f]{64}$')
       OR coalesce(o ->> 'ambito_consulta_hmac', '') !~
          ('^hmac-sha256:vec[.]contratacion-temporal[.]analisis[.]'
           || 'ambito-idempotencia/v[1-9][0-9]{0,8}:[0-9a-f]{64}$')
       OR coalesce(o ->> 'huella_consulta_hmac', '') !~
          ('^hmac-sha256:vec[.]contratacion-temporal[.]analisis[.]'
           || 'huella-semantica/v[1-9][0-9]{0,8}:[0-9a-f]{64}$') THEN
        RETURN false;
    END IF;
    FOR v_alias IN
        SELECT e.v
          FROM pg_catalog.jsonb_array_elements(
              o -> 'aliases_consulta'
          ) AS e(v)
    LOOP
        IF NOT vec_contratacion_temporal.claves_json_exactas_v1(
               v_alias, ARRAY['ambito_hmac', 'generacion']::text[]
           )
           OR pg_catalog.jsonb_typeof(v_alias -> 'generacion') <> 'number'
           OR (v_alias ->> 'generacion')::numeric
                NOT BETWEEN 1 AND 999999999::numeric
           OR coalesce(v_alias ->> 'ambito_hmac', '') !~
              ('^hmac-sha256:vec[.]contratacion-temporal[.]analisis[.]'
               || 'ambito-idempotencia/v[1-9][0-9]{0,8}:'
               || '[0-9a-f]{64}$')
           OR substring(
               v_alias ->> 'ambito_hmac'
               FROM '/v([1-9][0-9]{0,8}):'
           )::numeric <> (v_alias ->> 'generacion')::numeric THEN
            RETURN false;
        END IF;
        v_generaciones := pg_catalog.array_append(
            v_generaciones, (v_alias ->> 'generacion')::numeric
        );
        v_ambitos := pg_catalog.array_append(
            v_ambitos, v_alias ->> 'ambito_hmac'
        );
    END LOOP;
    IF v_ambitos[1] <> o ->> 'ambito_consulta_hmac'
       OR pg_catalog.cardinality(v_generaciones) <>
          (
              SELECT pg_catalog.count(DISTINCT g)
                FROM pg_catalog.unnest(v_generaciones) AS x(g)
          )
       OR pg_catalog.cardinality(v_ambitos) <>
          (
              SELECT pg_catalog.count(DISTINCT h)
                FROM pg_catalog.unnest(v_ambitos) AS x(h)
          ) THEN
        RETURN false;
    END IF;
    IF NOT vec_contratacion_temporal.claves_json_exactas_v1(
           f, ARRAY[
             'conjunto_huella_sha256', 'coste',
             'prueba_canonica_hex', 'rc'
           ]::text[]
       )
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           a, ARRAY[
             'accion', 'contexto_recurso_huella_sha256',
             'decision_canonica_hex', 'decision_huella_sha256',
             'decision_ref', 'finalidad', 'motivo_canonico_hex',
             'perfil_activo_ref', 'perfil_version', 'persona_version',
             'principal_id', 'recurso_ref'
           ]::text[]
       )
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           p, ARRAY[
             'accion', 'definicion_ref', 'estado_previo',
             'exige_actor_distinto', 'fase_previa', 'finalidad',
             'huella_sha256', 'unidad_ref', 'version'
           ]::text[]
       ) THEN
        RETURN false;
    END IF;
    IF pg_catalog.jsonb_typeof(p -> 'exige_actor_distinto') <> 'boolean'
       OR pg_catalog.jsonb_typeof(p -> 'version') <> 'number'
       OR (p ->> 'version')::numeric
            NOT BETWEEN 1 AND 9007199254740991::numeric
       OR pg_catalog.jsonb_typeof(a -> 'persona_version') <> 'number'
       OR pg_catalog.jsonb_typeof(a -> 'perfil_version') <> 'number'
       OR (a ->> 'persona_version')::numeric
            NOT BETWEEN 1 AND 9007199254740991::numeric
       OR (a ->> 'perfil_version')::numeric
            NOT BETWEEN 1 AND 9007199254740991::numeric
       OR coalesce(f ->> 'conjunto_huella_sha256', '')
            !~ '^[0-9a-f]{64}$'
       OR coalesce(f ->> 'prueba_canonica_hex', '')
            !~ '^[0-9a-f]+$'
       OR pg_catalog.length(f ->> 'prueba_canonica_hex')
            NOT BETWEEN 128 AND 131072
       OR pg_catalog.length(f ->> 'prueba_canonica_hex') % 2 <> 0
       OR coalesce(a ->> 'decision_canonica_hex', '')
            !~ '^[0-9a-f]+$'
       OR pg_catalog.length(a ->> 'decision_canonica_hex')
            NOT BETWEEN 256 AND 1048576
       OR pg_catalog.length(a ->> 'decision_canonica_hex') % 2 <> 0
       OR coalesce(a ->> 'motivo_canonico_hex', '')
            !~ '^[0-9a-f]+$'
       OR pg_catalog.length(a ->> 'motivo_canonico_hex')
            NOT BETWEEN 64 AND 131072
       OR pg_catalog.length(a ->> 'motivo_canonico_hex') % 2 <> 0 THEN
        RETURN false;
    END IF;
    FOR v_fuente IN
        SELECT e.v
          FROM pg_catalog.jsonb_array_elements(
              CASE WHEN f -> 'coste' = 'null'::jsonb
                   THEN pg_catalog.jsonb_build_array(f -> 'rc')
                   ELSE pg_catalog.jsonb_build_array(
                       f -> 'rc', f -> 'coste'
                   ) END
          ) AS e(v)
    LOOP
        IF NOT vec_contratacion_temporal.claves_json_exactas_v1(
               v_fuente, ARRAY[
                 'autoridad_ref', 'emitida_en', 'generacion',
                 'material_huella_sha256', 'peticion_ref', 'publicacion',
                 'recibo_respuesta_ref', 'respuesta_huella_sha256',
                 'sello_respuesta_hmac', 'tipo', 'valida_hasta',
                 'verificada_en', 'verificador_ref'
               ]::text[]
           )
           OR pg_catalog.jsonb_typeof(v_fuente -> 'generacion') <> 'number'
           OR (v_fuente ->> 'generacion')::numeric
                NOT BETWEEN 1 AND 4294967295::numeric
           OR coalesce(v_fuente ->> 'respuesta_huella_sha256', '')
                !~ '^[0-9a-f]{64}$'
           OR coalesce(v_fuente ->> 'material_huella_sha256', '')
                !~ '^[0-9a-f]{64}$'
           OR v_fuente ->> 'respuesta_huella_sha256' <>
                v_fuente ->> 'material_huella_sha256'
           OR coalesce(v_fuente ->> 'sello_respuesta_hmac', '') !~
                ('^hmac-sha256:fuente-analisis-respuesta/v'
                 || (v_fuente ->> 'generacion') || ':[0-9a-f]{64}$')
           OR (v_fuente ->> 'verificador_ref') =
                (v_fuente ->> 'autoridad_ref')
           OR (v_fuente ->> 'valida_hasta')::timestamptz <=
                (v_fuente ->> 'emitida_en')::timestamptz
           OR (v_fuente ->> 'valida_hasta')::timestamptz >
                (v_fuente ->> 'emitida_en')::timestamptz +
                interval '5 seconds'
           OR (v_fuente ->> 'verificada_en')::timestamptz <
                (v_fuente ->> 'emitida_en')::timestamptz
           OR (v_fuente ->> 'verificada_en')::timestamptz >=
                (v_fuente ->> 'valida_hasta')::timestamptz THEN
            RETURN false;
        END IF;
        IF v_fuente -> 'publicacion' <> 'null'::jsonb
           AND NOT vec_contratacion_temporal.claves_json_exactas_v1(
               v_fuente -> 'publicacion', ARRAY[
                 'huella_solicitud_sha256', 'publicacion_ref',
                 'publicador_ref', 'recibo_verificacion_ref',
                 'verificada_en'
               ]::text[]
           ) THEN
            RETURN false;
        END IF;
    END LOOP;
    IF f #>> '{rc,tipo}' <> 'validacion_rc'
       OR (
           f -> 'coste' <> 'null'::jsonb
           AND f #>> '{coste,tipo}' <> 'calculo_coste'
       )
       OR vec_contratacion_temporal.reconstruir_prueba_fuentes_analisis_v1(o)
            IS DISTINCT FROM
          pg_catalog.decode(f ->> 'prueba_canonica_hex', 'hex')
       OR pg_catalog.encode(
           pg_catalog.sha256(
             vec_contratacion_temporal.reconstruir_prueba_fuentes_analisis_v1(o)
           ), 'hex'
       ) <> f ->> 'conjunto_huella_sha256' THEN
        RETURN false;
    END IF;
    RETURN true;
EXCEPTION
    WHEN data_exception OR datetime_field_overflow
      OR invalid_text_representation OR numeric_value_out_of_range THEN
        RETURN false;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.claves_json_exactas_v1(jsonb, text[]),
    vec_contratacion_temporal.encuadrar_binario_analisis_v1(text),
    vec_contratacion_temporal.microsegundos_unix_analisis_v1(text),
    vec_contratacion_temporal.reconstruir_fuente_analisis_v1(
        jsonb, text, text, numeric
    ),
    vec_contratacion_temporal.reconstruir_prueba_fuentes_analisis_v1(jsonb),
    vec_contratacion_temporal.entrada_confirmacion_analisis_valida_v1(jsonb)
FROM PUBLIC, vec_contratacion_temporal_ejecutor;

COMMIT;
