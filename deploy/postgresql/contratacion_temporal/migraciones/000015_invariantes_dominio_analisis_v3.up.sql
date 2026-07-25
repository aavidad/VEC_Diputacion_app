BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000015_invariantes_dominio_analisis_v3',
        0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.confirmar_operacion_analisis_v2(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.huella_analisis_derivado_v2(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.confirmar_operacion_analisis_v3(jsonb)'
       ) IS NOT NULL
       OR unicode_version() <> '16.0' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para invariantes O3 v3';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION
vec_contratacion_temporal.referencia_dominio_analisis_valida_v3(
    p_valor text
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT p_valor IS NOT NULL
       AND p_valor ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.clave_dominio_analisis_valida_v3(
    p_valor text
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT p_valor IS NOT NULL
       AND p_valor ~ '^[a-z][a-z0-9._-]{1,79}$'
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.grupo_dominio_analisis_valido_v3(
    p_valor text
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT p_valor IS NOT NULL
       AND p_valor ~ '^[A-Z][A-Z0-9/+.-]{0,19}$'
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.huella_dominio_analisis_valida_v3(
    p_valor text
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT p_valor IS NOT NULL
       AND p_valor ~ '^[a-f0-9]{64}$'
       AND p_valor <> pg_catalog.repeat('0', 64)
$funcion$;

-- Reproduce textoValido de Go para el motivo de una validacion RC negativa:
-- TrimSpace Unicode, NFC, limite por runas y controles salvo LF/TAB.
-- PostgreSQL 18 usa Unicode 16 y x/text v0.40 usa Unicode 15 con Go 1.26.
-- Se excluyen las runas cuyo perfil NFC cambia entre Unicode 15, 16 y 17
-- para que ninguna version acepte una representacion rechazada por otra.
CREATE FUNCTION
vec_contratacion_temporal.texto_dominio_analisis_valido_v3(
    p_valor text,
    p_maximo integer,
    p_permite_vacio boolean
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_codigo integer;
    v_indice integer;
    v_primero integer;
    v_ultimo integer;
BEGIN
    IF unicode_version() <> '16.0'
       OR p_valor IS NULL OR p_maximo IS NULL OR p_maximo < 0
       OR p_permite_vacio IS NULL
       OR (NOT p_permite_vacio AND p_valor = '')
       OR pg_catalog.char_length(p_valor) > p_maximo
       OR normalize(p_valor, NFC) IS DISTINCT FROM p_valor THEN
        RETURN false;
    END IF;
    IF p_valor = '' THEN
        RETURN true;
    END IF;
    v_primero := pg_catalog.ascii(pg_catalog.substr(p_valor, 1, 1));
    v_ultimo := pg_catalog.ascii(
        pg_catalog.substr(
            p_valor, pg_catalog.char_length(p_valor), 1
        )
    );
    IF v_primero BETWEEN 9 AND 13
       OR v_primero IN (32, 133, 160, 5760, 8232, 8233, 8239, 8287, 12288)
       OR v_primero BETWEEN 8192 AND 8202
       OR v_ultimo BETWEEN 9 AND 13
       OR v_ultimo IN (32, 133, 160, 5760, 8232, 8233, 8239, 8287, 12288)
       OR v_ultimo BETWEEN 8192 AND 8202 THEN
        RETURN false;
    END IF;
    FOR v_indice IN 1..pg_catalog.char_length(p_valor) LOOP
        v_codigo := pg_catalog.ascii(
            pg_catalog.substr(p_valor, v_indice, 1)
        );
        IF (v_codigo BETWEEN 0 AND 31 AND v_codigo NOT IN (9, 10))
           OR v_codigo BETWEEN 127 AND 159
           OR v_codigo = 2199
           OR v_codigo BETWEEN 6863 AND 6877
           OR v_codigo BETWEEN 6880 AND 6891
           OR v_codigo IN (67017, 67026, 67034, 67044)
           OR v_codigo BETWEEN 68969 AND 68973
           OR v_codigo BETWEEN 69370 AND 69371
           OR v_codigo BETWEEN 70530 AND 70533
           OR v_codigo IN (70539, 70542)
           OR v_codigo BETWEEN 70544 AND 70545
           OR v_codigo IN (70584, 70587, 70594, 70597)
           OR v_codigo BETWEEN 70599 AND 70601
           OR v_codigo BETWEEN 70606 AND 70608
           OR v_codigo BETWEEN 90398 AND 90409
           OR v_codigo = 90415
           OR v_codigo = 93539
           OR v_codigo BETWEEN 93543 AND 93546
           OR v_codigo BETWEEN 124398 AND 124399
           OR v_codigo IN (124643, 124646, 124661)
           OR v_codigo BETWEEN 124654 AND 124655 THEN
            RETURN false;
        END IF;
    END LOOP;
    RETURN true;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR numeric_value_out_of_range THEN
        RETURN false;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.analisis_rrhh_valido_v3(a jsonb)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v jsonb := a -> 'validacion_rc';
    v_vinculo jsonb := a -> 'actuacion_registro';
    v_inicio date;
    v_fin date;
    v_limite date;
    v_resultado text;
    v_tiene_coste boolean;
BEGIN
    -- V2 conserva el contrato JSON exacto, los tipos, los enteros canonicos,
    -- los limites monetarios y la representacion temporal. V3 añade las
    -- invariantes semanticas que aplica AnalisisRRHH.Validar en Go.
    IF vec_contratacion_temporal.huella_analisis_derivado_v2(a) IS NULL
       OR NOT vec_contratacion_temporal
           .clave_dominio_analisis_valida_v3(a ->> 'modalidad_clave')
       OR NOT vec_contratacion_temporal
           .referencia_dominio_analisis_valida_v3(a ->> 'categoria_ref')
       OR NOT vec_contratacion_temporal
           .grupo_dominio_analisis_valido_v3(a ->> 'grupo_subgrupo')
       OR NOT vec_contratacion_temporal
           .clave_dominio_analisis_valida_v3(a ->> 'causa_clave')
       OR NOT vec_contratacion_temporal
           .referencia_dominio_analisis_valida_v3(
               a #>> '{entrada_rc_esperada,referencia}'
           )
       OR NOT vec_contratacion_temporal
           .huella_dominio_analisis_valida_v3(
               a #>> '{entrada_rc_esperada,huella_sha256}'
           )
       OR NOT vec_contratacion_temporal
           .clave_dominio_analisis_valida_v3(
               v_vinculo ->> 'accion_clave'
           )
       OR NOT vec_contratacion_temporal
           .clave_dominio_analisis_valida_v3(
               v_vinculo ->> 'fase_destino'
           )
       OR NOT vec_contratacion_temporal
           .referencia_dominio_analisis_valida_v3(
               v_vinculo ->> 'recibo_ref'
           )
       OR (v_vinculo ->> 'secuencia')::numeric <>
          (v_vinculo ->> 'version_expediente')::numeric
       OR NOT vec_contratacion_temporal
           .referencia_dominio_analisis_valida_v3(v ->> 'entrada_ref')
       OR NOT vec_contratacion_temporal
           .huella_dominio_analisis_valida_v3(
               v ->> 'huella_entrada_sha256'
           )
       OR NOT vec_contratacion_temporal
           .referencia_dominio_analisis_valida_v3(v ->> 'fuente_ref')
       OR NOT vec_contratacion_temporal
           .referencia_dominio_analisis_valida_v3(v ->> 'recibo_ref')
       OR a #>> '{entrada_rc_esperada,referencia}'
          IS DISTINCT FROM v ->> 'entrada_ref'
       OR a #>> '{entrada_rc_esperada,huella_sha256}'
          IS DISTINCT FROM v ->> 'huella_entrada_sha256'
       OR v ->> 'validada_en' = '0001-01-01T00:00:00Z'
       OR a #>> '{periodo,inicio}' = '0001-01-01T00:00:00Z'
       OR a #>> '{periodo,fin}' = '0001-01-01T00:00:00Z' THEN
        RETURN false;
    END IF;

    v_inicio := pg_catalog.substr(
        a #>> '{periodo,inicio}', 1, 10
    )::date;
    v_fin := pg_catalog.substr(a #>> '{periodo,fin}', 1, 10)::date;
    -- time.Time.AddDate normaliza un 29 de febrero sobre un año no bisiesto
    -- al 1 de marzo; construir desde el primer día reproduce esa semantica.
    v_limite := pg_catalog.make_date(
        extract(year FROM v_inicio)::integer + 100,
        extract(month FROM v_inicio)::integer,
        1
    ) + (extract(day FROM v_inicio)::integer - 1);
    IF v_fin > v_limite THEN
        RETURN false;
    END IF;

    v_resultado := v ->> 'resultado';
    IF v_resultado = 'validada' THEN
        IF v ->> 'fecha_rc' = '0001-01-01T00:00:00Z'
           OR NOT vec_contratacion_temporal
               .referencia_dominio_analisis_valida_v3(v ->> 'numero')
           OR NOT vec_contratacion_temporal
               .referencia_dominio_analisis_valida_v3(
                   v ->> 'documento_ref'
               )
           OR v #>> '{importe,moneda}' <> 'EUR' THEN
            RETURN false;
        END IF;
    ELSIF NOT vec_contratacion_temporal.texto_dominio_analisis_valido_v3(
              v ->> 'motivo', 1000, false
          ) THEN
        RETURN false;
    END IF;

    v_tiene_coste := pg_catalog.jsonb_exists(a, 'coste_previsto');
    IF v_tiene_coste
       AND (
           a #>> '{coste_previsto,moneda}' <> 'EUR'
           OR NOT vec_contratacion_temporal
               .referencia_dominio_analisis_valida_v3(
                   a ->> 'fuente_coste_ref'
               )
           OR (
               v_resultado = 'validada'
               AND (a #>> '{coste_previsto,centimos}')::numeric >
                   (v #>> '{importe,centimos}')::numeric
           )
       ) THEN
        RETURN false;
    END IF;
    RETURN true;
EXCEPTION
    WHEN data_exception OR datetime_field_overflow
      OR invalid_text_representation OR numeric_value_out_of_range THEN
        RETURN false;
END
$funcion$;

-- El cuerpo SQL registra una dependencia de catalogo sobre V2. PostgreSQL no
-- permite retirar 000014 mientras 000015 permanezca instalada.
CREATE FUNCTION
vec_contratacion_temporal.ejecutar_confirmacion_analisis_base_v3(o jsonb)
RETURNS TABLE (recibo_json jsonb)
LANGUAGE sql
VOLATILE
SECURITY INVOKER
SET search_path = pg_catalog
BEGIN ATOMIC
    SELECT c.recibo_json
      FROM vec_contratacion_temporal.confirmar_operacion_analisis_v2(o) c;
END;

CREATE FUNCTION
vec_contratacion_temporal.confirmar_operacion_analisis_v3(o jsonb)
RETURNS TABLE (recibo_json jsonb)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
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
       OR o ->> 'operacion' NOT IN ('registrar', 'rectificar')
       OR vec_contratacion_temporal.analisis_rrhh_valido_v3(
              o #> '{expediente_siguiente,analisis}'
          ) IS NOT TRUE
       OR (
           o ->> 'operacion' = 'rectificar'
           AND vec_contratacion_temporal.analisis_rrhh_valido_v3(
                   o #> '{expediente_anterior,analisis}'
               ) IS NOT TRUE
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'frontera de confirmacion O3 v3 no autorizada';
    END IF;
    RETURN QUERY
    SELECT c.recibo_json
      FROM vec_contratacion_temporal
           .ejecutar_confirmacion_analisis_base_v3(o) c;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.referencia_dominio_analisis_valida_v3(text),
    vec_contratacion_temporal.clave_dominio_analisis_valida_v3(text),
    vec_contratacion_temporal.grupo_dominio_analisis_valido_v3(text),
    vec_contratacion_temporal.huella_dominio_analisis_valida_v3(text),
    vec_contratacion_temporal.texto_dominio_analisis_valido_v3(
        text, integer, boolean
    ),
    vec_contratacion_temporal.analisis_rrhh_valido_v3(jsonb),
    vec_contratacion_temporal.ejecutar_confirmacion_analisis_base_v3(jsonb),
    vec_contratacion_temporal.confirmar_operacion_analisis_v3(jsonb)
FROM PUBLIC, vec_contratacion_temporal_ejecutor;

REVOKE EXECUTE ON FUNCTION
vec_contratacion_temporal.confirmar_operacion_analisis_v2(jsonb)
FROM vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION
vec_contratacion_temporal.confirmar_operacion_analisis_v3(jsonb)
TO vec_contratacion_temporal_ejecutor;

COMMIT;
