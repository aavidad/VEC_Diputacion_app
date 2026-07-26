-- Wrapper VEC interno de O4-04D. La función exterior pertenece a O4-04E.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:o4_04:migraciones', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_autorizacion.decision_contexto_actor_v3_valida(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.decision_contexto_actor_v3_canonica(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.registrar_decision_contexto_actor_v3(bytea,bytea,numeric,numeric)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_v3(bytea,bytea,numeric,numeric)'
       ) IS NULL
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_contratacion_temporal_propietario'
              AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls
       )
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(bytea,bytea,numeric,numeric,jsonb)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para wrapper VEC O4-04D';
    END IF;
END
$prevalidacion$;

-- Los cuerpos SQL estándar registran dependencias fuertes. Los down de VEC
-- base fallan con DROP RESTRICT mientras este wrapper siga instalado.
CREATE FUNCTION vec_autorizacion.o404d_decision_cobertura_v3_exacta_v1(
    p_documento jsonb,
    p_canon bytea
)
RETURNS boolean
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog
BEGIN ATOMIC
    SELECT vec_autorizacion.decision_contexto_actor_v3_valida(p_documento)
       AND vec_autorizacion.decision_contexto_actor_v3_canonica(p_documento)
           = p_canon;
END;

CREATE FUNCTION vec_autorizacion.o404d_registrar_decision_v3_base_v1(
    p_decision bytea,
    p_motivo bytea,
    p_persona numeric,
    p_perfil numeric
)
RETURNS TABLE (
    concedida boolean,
    codigo text,
    decision_huella_sha256 text,
    registrada_en timestamptz
)
LANGUAGE sql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
BEGIN ATOMIC
    SELECT * FROM vec_autorizacion.registrar_decision_contexto_actor_v3(
        p_decision, p_motivo, p_persona, p_perfil
    );
END;

CREATE FUNCTION vec_autorizacion.o404d_registrar_decision_v3_viva_v1(
    p_decision bytea,
    p_motivo bytea,
    p_persona numeric,
    p_perfil numeric
)
RETURNS TABLE (
    concedida boolean,
    codigo text,
    decision_huella_sha256 text,
    registrada_en timestamptz,
    revalidada_en timestamptz
)
LANGUAGE sql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
BEGIN ATOMIC
    SELECT *
      FROM vec_autorizacion
           .registrar_y_revalidar_decision_contexto_actor_v3(
               p_decision, p_motivo, p_persona, p_perfil
           );
END;

CREATE FUNCTION
vec_autorizacion.o404d_material_recurso_cobertura_v1(
    p_rama text,
    p_organizacion_ref text,
    p_expediente_ref text,
    p_version_expediente numeric,
    p_reserva_ref text,
    p_huella_orden_sha256 text,
    p_lote_huella_sha256 text
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_material bytea := ''::bytea;
    v_texto text;
BEGIN
    IF p_rama IS DISTINCT FROM 'concedida'
       AND p_rama IS DISTINCT FROM 'denegada'
       OR p_version_expediente IS NULL
       OR p_version_expediente IS DISTINCT FROM
          pg_catalog.trunc(p_version_expediente)
       OR p_version_expediente NOT BETWEEN
          2 AND 9007199254740990::numeric
       OR p_huella_orden_sha256 !~ '^[a-f0-9]{64}$'
       OR (
           p_rama IS NOT DISTINCT FROM 'concedida'
           AND p_lote_huella_sha256 !~ '^[a-f0-9]{64}$'
       )
       OR (
           p_rama IS NOT DISTINCT FROM 'denegada'
           AND p_lote_huella_sha256 IS NOT NULL
       ) THEN
        RETURN NULL;
    END IF;
    FOREACH v_texto IN ARRAY ARRAY[
        'VEC-CT-RECURSO-COBERTURA-O4-04D-V1',
        p_rama, p_organizacion_ref, p_expediente_ref, p_reserva_ref
    ]::text[]
    LOOP
        IF v_texto IS NULL THEN
            RETURN NULL;
        END IF;
        v_material := v_material
            || pg_catalog.int4send(
                   pg_catalog.octet_length(
                       pg_catalog.convert_to(v_texto, 'UTF8')
                   )
               )
            || pg_catalog.convert_to(v_texto, 'UTF8');
    END LOOP;
    v_material := v_material
        || pg_catalog.int8send(p_version_expediente::bigint)
        || pg_catalog.decode(p_huella_orden_sha256, 'hex');
    IF p_lote_huella_sha256 IS NULL THEN
        RETURN v_material || E'\\x00'::bytea;
    END IF;
    RETURN v_material || E'\\x01'::bytea
        || pg_catalog.decode(p_lote_huella_sha256, 'hex');
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR numeric_value_out_of_range THEN
        RETURN NULL;
END
$funcion$;

CREATE FUNCTION
vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric,
    p_vinculo_efecto jsonb
)
RETURNS TABLE (
    rama text,
    concedida boolean,
    codigo text,
    decision_ref text,
    correlacion_ref text,
    organizacion_ref text,
    expediente_ref text,
    version_expediente numeric,
    reserva_ref text,
    contexto_recurso_huella_sha256 text,
    decision_huella_sha256 text,
    huella_orden_sha256 text,
    lote_huella_sha256 text,
    prueba_vinculo_sha256 text,
    registrada_en timestamptz,
    revalidada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_decision jsonb;
    v_claves text[];
    v_registro record;
    v_revalidado record;
    v_statement_ms numeric;
    v_idle_ms numeric;
    v_prueba_vinculo text;
    v_contexto_recurso_huella text;
BEGIN
    SELECT CASE WHEN unit = 'ms' AND setting ~ '^[0-9]{1,18}$'
                THEN setting::numeric END
      INTO v_statement_ms FROM pg_catalog.pg_settings
     WHERE name = 'statement_timeout';
    SELECT CASE WHEN unit = 'ms' AND setting ~ '^[0-9]{1,18}$'
                THEN setting::numeric END
      INTO v_idle_ms FROM pg_catalog.pg_settings
     WHERE name = 'idle_in_transaction_session_timeout';
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR v_statement_ms IS NULL OR v_statement_ms NOT BETWEEN 1 AND 15000
       OR v_idle_ms IS NULL OR v_idle_ms NOT BETWEEN 1 AND 20000
       OR p_decision_canonica IS NULL
       OR pg_catalog.octet_length(p_decision_canonica)
          NOT BETWEEN 128 AND 524288
       OR p_motivo_canonico IS NULL
       OR pg_catalog.octet_length(p_motivo_canonico)
          NOT BETWEEN 32 AND 65536
       OR p_vinculo_efecto IS NULL
       OR pg_catalog.jsonb_typeof(p_vinculo_efecto) <> 'object'
       OR pg_catalog.pg_column_size(p_vinculo_efecto) > 16384 THEN
        RETURN;
    END IF;
    SELECT pg_catalog.array_agg(clave ORDER BY clave COLLATE "C")
      INTO v_claves
      FROM pg_catalog.jsonb_object_keys(p_vinculo_efecto) c(clave);
    IF (
           p_vinculo_efecto ->> 'rama' = 'concedida'
           AND v_claves IS DISTINCT FROM ARRAY[
               'accion', 'contexto_recurso_huella_sha256',
               'correlacion_ref', 'decision_ref', 'expediente_ref',
               'finalidad', 'huella_orden_sha256',
               'lote_huella_sha256', 'organizacion_ref', 'rama',
               'reserva_ref', 'version_expediente'
           ]::text[]
       )
       OR (
           p_vinculo_efecto ->> 'rama' = 'denegada'
           AND v_claves IS DISTINCT FROM ARRAY[
               'accion', 'contexto_recurso_huella_sha256',
               'correlacion_ref', 'decision_ref', 'expediente_ref',
               'finalidad', 'huella_orden_sha256',
               'organizacion_ref', 'rama', 'reserva_ref',
               'version_expediente'
           ]::text[]
       )
       OR p_vinculo_efecto ->> 'rama' NOT IN ('concedida', 'denegada')
       OR p_vinculo_efecto ->> 'accion' NOT IN (
           'contratacion_temporal.cobertura.decidir',
           'contratacion_temporal.cobertura.rectificar'
       )
       OR (p_vinculo_efecto ->> 'organizacion_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_vinculo_efecto ->> 'expediente_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_vinculo_efecto ->> 'reserva_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_vinculo_efecto ->> 'decision_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_vinculo_efecto ->> 'correlacion_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_vinculo_efecto ->> 'finalidad') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR pg_catalog.jsonb_typeof(
              p_vinculo_efecto -> 'version_expediente'
          ) <> 'number'
       OR (p_vinculo_efecto ->> 'version_expediente') !~
          '^([2-9]|[1-9][0-9]+)$'
       OR (p_vinculo_efecto ->> 'version_expediente')::numeric >
          9007199254740990::numeric
       OR (p_vinculo_efecto ->> 'contexto_recurso_huella_sha256') !~
          '^[a-f0-9]{64}$'
       OR (p_vinculo_efecto ->> 'huella_orden_sha256') !~
          '^[a-f0-9]{64}$'
       OR (
           p_vinculo_efecto ->> 'rama' = 'concedida'
           AND (p_vinculo_efecto ->> 'lote_huella_sha256') !~
              '^[a-f0-9]{64}$'
       ) THEN
        RETURN;
    END IF;
    BEGIN
        v_decision := pg_catalog.convert_from(
            p_decision_canonica, 'UTF8'
        )::jsonb;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
          OR character_not_in_repertoire OR untranslatable_character THEN
            RETURN;
    END;
    v_contexto_recurso_huella := pg_catalog.encode(
        pg_catalog.sha256(
            vec_autorizacion.o404d_material_recurso_cobertura_v1(
                p_vinculo_efecto ->> 'rama',
                p_vinculo_efecto ->> 'organizacion_ref',
                p_vinculo_efecto ->> 'expediente_ref',
                (p_vinculo_efecto ->> 'version_expediente')::numeric,
                p_vinculo_efecto ->> 'reserva_ref',
                p_vinculo_efecto ->> 'huella_orden_sha256',
                p_vinculo_efecto ->> 'lote_huella_sha256'
            )
        ),
        'hex'
    );
    IF vec_autorizacion.o404d_decision_cobertura_v3_exacta_v1(
           v_decision, p_decision_canonica
       ) IS NOT TRUE
       OR v_decision ->> 'modulo_id' IS DISTINCT FROM
          'contratacion_temporal'
       OR v_decision ->> 'tipo_recurso' IS DISTINCT FROM
          'decision_cobertura_gobernada'
       OR v_decision ->> 'accion' IS DISTINCT FROM
          p_vinculo_efecto ->> 'accion'
       OR v_decision ->> 'recurso_ref' IS DISTINCT FROM
          p_vinculo_efecto ->> 'reserva_ref'
       OR v_decision ->> 'decision_ref' IS DISTINCT FROM
          p_vinculo_efecto ->> 'decision_ref'
       OR v_decision ->> 'correlacion_ref' IS DISTINCT FROM
          p_vinculo_efecto ->> 'correlacion_ref'
       OR v_decision ->> 'finalidad' IS DISTINCT FROM
          p_vinculo_efecto ->> 'finalidad'
       OR v_decision ->> 'contexto_recurso_huella_sha256'
          IS DISTINCT FROM
          p_vinculo_efecto ->> 'contexto_recurso_huella_sha256'
       OR v_contexto_recurso_huella IS DISTINCT FROM
          p_vinculo_efecto ->> 'contexto_recurso_huella_sha256'
       OR (
           p_vinculo_efecto ->> 'rama' = 'concedida'
           AND (
               v_decision ->> 'concedida' IS DISTINCT FROM 'true'
               OR v_decision ->> 'codigo' IS DISTINCT FROM 'concedida'
           )
       )
       OR (
           p_vinculo_efecto ->> 'rama' = 'denegada'
           AND v_decision ->> 'concedida' IS DISTINCT FROM 'false'
       ) THEN
        RETURN;
    END IF;
    IF p_vinculo_efecto ->> 'rama' = 'concedida' THEN
        SELECT * INTO v_revalidado
          FROM vec_autorizacion
               .o404d_registrar_decision_v3_viva_v1(
                   p_decision_canonica, p_motivo_canonico,
                   p_persona_version, p_perfil_version
               );
        IF NOT FOUND OR v_revalidado.concedida IS NOT TRUE
           OR v_revalidado.codigo IS DISTINCT FROM 'concedida' THEN
            RETURN;
        END IF;
        v_prueba_vinculo := pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.decode(
                v_revalidado.decision_huella_sha256, 'hex'
            )
            || pg_catalog.decode(v_contexto_recurso_huella, 'hex')
            || pg_catalog.decode(
                p_vinculo_efecto ->> 'huella_orden_sha256', 'hex'
            )
            || pg_catalog.decode(
                p_vinculo_efecto ->> 'lote_huella_sha256', 'hex'
            )
            || pg_catalog.int4send(pg_catalog.octet_length(
                v_decision ->> 'decision_ref'
            ))
            || pg_catalog.convert_to(
                v_decision ->> 'decision_ref', 'UTF8'
            )
            || pg_catalog.int4send(pg_catalog.octet_length(
                v_decision ->> 'correlacion_ref'
            ))
            || pg_catalog.convert_to(
                v_decision ->> 'correlacion_ref', 'UTF8'
            )
            || pg_catalog.int8send((
                extract(epoch FROM v_revalidado.registrada_en)::numeric
                * 1000000
            )::bigint)
            || pg_catalog.int8send((
                extract(epoch FROM v_revalidado.revalidada_en)::numeric
                * 1000000
            )::bigint)
        ), 'hex');
        RETURN QUERY SELECT
            'concedida'::text, true, v_revalidado.codigo,
            v_decision ->> 'decision_ref',
            v_decision ->> 'correlacion_ref',
            p_vinculo_efecto ->> 'organizacion_ref',
            p_vinculo_efecto ->> 'expediente_ref',
            (p_vinculo_efecto ->> 'version_expediente')::numeric,
            p_vinculo_efecto ->> 'reserva_ref',
            v_contexto_recurso_huella,
            v_revalidado.decision_huella_sha256,
            p_vinculo_efecto ->> 'huella_orden_sha256',
            p_vinculo_efecto ->> 'lote_huella_sha256',
            v_prueba_vinculo,
            v_revalidado.registrada_en, v_revalidado.revalidada_en;
        RETURN;
    END IF;
    SELECT * INTO v_registro
      FROM vec_autorizacion.o404d_registrar_decision_v3_base_v1(
          p_decision_canonica, p_motivo_canonico,
          p_persona_version, p_perfil_version
      );
    IF NOT FOUND OR v_registro.concedida IS NOT FALSE
       OR v_registro.codigo IS DISTINCT FROM
          v_decision ->> 'codigo' THEN
        RETURN;
    END IF;
    v_prueba_vinculo := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.decode(v_registro.decision_huella_sha256, 'hex')
        || pg_catalog.decode(v_contexto_recurso_huella, 'hex')
        || pg_catalog.decode(
            p_vinculo_efecto ->> 'huella_orden_sha256', 'hex'
        )
        || pg_catalog.int4send(pg_catalog.octet_length(
            v_decision ->> 'decision_ref'
        ))
        || pg_catalog.convert_to(v_decision ->> 'decision_ref', 'UTF8')
        || pg_catalog.int4send(pg_catalog.octet_length(
            v_decision ->> 'correlacion_ref'
        ))
        || pg_catalog.convert_to(
            v_decision ->> 'correlacion_ref', 'UTF8'
        )
        || pg_catalog.int8send((
            extract(epoch FROM v_registro.registrada_en)::numeric
            * 1000000
        )::bigint)
        || pg_catalog.int8send(0)
    ), 'hex');
    RETURN QUERY SELECT
        'denegada'::text, false, v_registro.codigo,
        v_decision ->> 'decision_ref',
        v_decision ->> 'correlacion_ref',
        p_vinculo_efecto ->> 'organizacion_ref',
        p_vinculo_efecto ->> 'expediente_ref',
        (p_vinculo_efecto ->> 'version_expediente')::numeric,
        p_vinculo_efecto ->> 'reserva_ref',
        v_contexto_recurso_huella,
        v_registro.decision_huella_sha256,
        p_vinculo_efecto ->> 'huella_orden_sha256',
        NULL::text, v_prueba_vinculo,
        v_registro.registrada_en, NULL::timestamptz;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR datetime_field_overflow OR no_data_found OR too_many_rows THEN
        RETURN;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion.o404d_decision_cobertura_v3_exacta_v1(jsonb, bytea),
    vec_autorizacion.o404d_registrar_decision_v3_base_v1(
        bytea, bytea, numeric, numeric
    ),
    vec_autorizacion.o404d_registrar_decision_v3_viva_v1(
        bytea, bytea, numeric, numeric
    ),
    vec_autorizacion.o404d_material_recurso_cobertura_v1(
        text, text, text, numeric, text, text, text
    ),
    vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(
        bytea, bytea, numeric, numeric, jsonb
    )
FROM PUBLIC, vec_autorizacion_registro,
       vec_autorizacion_fuente, vec_autorizacion_motivos_proyector,
       vec_autorizacion_motivos_evaluador,
       vec_contratacion_temporal_ejecutor,
       vec_contratacion_temporal_migrador,
       vec_contratacion_temporal_gobernador;
GRANT USAGE ON SCHEMA vec_autorizacion
    TO vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION
vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(
    bytea, bytea, numeric, numeric, jsonb
) TO vec_contratacion_temporal_propietario;

COMMENT ON FUNCTION
vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(
    bytea, bytea, numeric, numeric, jsonb
) IS
    'Wrapper interno O4-04D: resuelve y coteja el canon del recurso autorizado; solo O4-04E podrá componerlo con la persistencia en la misma transacción.';

COMMIT;
