-- Frontera estrecha entre contratación temporal y la autoridad VEC V3.
-- El ejecutor runtime no recibe acceso: solo el propietario NOLOGIN de la
-- persistencia puede invocarla desde su función SECURITY DEFINER.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:autorizacion:analisis-v3:000001', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)'
       ) IS NULL
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_contratacion_temporal_propietario'
              AND rolcanlogin IS FALSE
              AND rolsuper IS FALSE
              AND rolbypassrls IS FALSE
       )
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.revalidar_decision_analisis_contratacion_temporal_v1(bytea,bytea,numeric,numeric,jsonb)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar la frontera de autorización del análisis';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION
vec_autorizacion.revalidar_decision_analisis_contratacion_temporal_v1(
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric,
    p_vinculo_efecto jsonb
)
RETURNS TABLE (
    revalidada_en timestamptz,
    decision_ref text,
    decision_huella_sha256 text
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_decision jsonb;
    v_claves text[];
    v_revalidada_en timestamptz(6);
BEGIN
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR p_vinculo_efecto IS NULL
       OR pg_catalog.jsonb_typeof(p_vinculo_efecto) <> 'object'
       OR pg_catalog.pg_column_size(p_vinculo_efecto) > 16384
       OR p_decision_canonica IS NULL
       OR pg_catalog.octet_length(p_decision_canonica)
            NOT BETWEEN 128 AND 524288
       OR p_motivo_canonico IS NULL
       OR pg_catalog.octet_length(p_motivo_canonico)
            NOT BETWEEN 32 AND 65536 THEN
        RETURN;
    END IF;
    SELECT pg_catalog.array_agg(clave ORDER BY clave)
      INTO v_claves
      FROM pg_catalog.jsonb_object_keys(p_vinculo_efecto) AS c(clave);
    IF v_claves IS DISTINCT FROM ARRAY[
           'accion', 'contexto_recurso_huella_sha256', 'decision_ref',
           'finalidad', 'perfil_activo_ref', 'principal_id', 'recurso_ref'
       ]::text[]
       OR p_vinculo_efecto ->> 'accion' NOT IN (
           'contratacion_temporal.analisis.registrar',
           'contratacion_temporal.analisis.rectificar'
       )
       OR coalesce(
           p_vinculo_efecto ->> 'contexto_recurso_huella_sha256', ''
       ) !~ '^[0-9a-f]{64}$' THEN
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
    IF vec_autorizacion.decision_contexto_actor_v3_valida(
           v_decision
       ) IS NOT TRUE
       OR v_decision ->> 'concedida' <> 'true'
       OR v_decision ->> 'codigo' <> 'concedida'
       OR v_decision ->> 'modulo_id' <> 'contratacion_temporal'
       OR v_decision ->> 'tipo_recurso' <>
          'analisis_contratacion_temporal'
       OR v_decision ->> 'accion' <>
          p_vinculo_efecto ->> 'accion'
       OR v_decision ->> 'contexto_recurso_huella_sha256' <>
          p_vinculo_efecto ->> 'contexto_recurso_huella_sha256'
       OR v_decision ->> 'decision_ref' <>
          p_vinculo_efecto ->> 'decision_ref'
       OR v_decision ->> 'finalidad' <>
          p_vinculo_efecto ->> 'finalidad'
       OR v_decision ->> 'perfil_activo_ref' <>
          p_vinculo_efecto ->> 'perfil_activo_ref'
       OR v_decision ->> 'principal_id' <>
          p_vinculo_efecto ->> 'principal_id'
       OR v_decision ->> 'recurso_ref' <>
          p_vinculo_efecto ->> 'recurso_ref' THEN
        RETURN;
    END IF;
    v_revalidada_en :=
      vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
          p_decision_canonica,
          p_motivo_canonico,
          p_persona_version,
          p_perfil_version
      );
    IF v_revalidada_en IS NULL THEN
        RETURN;
    END IF;
    RETURN QUERY SELECT
        v_revalidada_en,
        v_decision ->> 'decision_ref',
        pg_catalog.encode(
            pg_catalog.sha256(p_decision_canonica), 'hex'
        );
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR datetime_field_overflow OR no_data_found OR too_many_rows THEN
        RETURN;
END
$funcion$;

REVOKE ALL ON FUNCTION
vec_autorizacion.revalidar_decision_analisis_contratacion_temporal_v1(
    bytea, bytea, numeric, numeric, jsonb
) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion
    TO vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION
vec_autorizacion.revalidar_decision_analisis_contratacion_temporal_v1(
    bytea, bytea, numeric, numeric, jsonb
) TO vec_contratacion_temporal_propietario;

COMMENT ON FUNCTION
vec_autorizacion.revalidar_decision_analisis_contratacion_temporal_v1(
    bytea, bytea, numeric, numeric, jsonb
) IS
    'Revalida una decisión VEC V3 viva limitada a registrar o rectificar un análisis de contratación temporal; no consume ni confirma el efecto.';

COMMIT;
