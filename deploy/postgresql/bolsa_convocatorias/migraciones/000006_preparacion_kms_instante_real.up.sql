-- Separa el instante real de preparacion del not-before de confirmacion. La
-- v1 queda como nucleo propietario para no reescribir una migracion publicada;
-- el runtime solo puede invocar esta v2 cerrada.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
BEGIN
    IF to_regprocedure(
           'vec_bolsa_convocatorias.preparar_confirmacion_borrador_v1(jsonb,jsonb,jsonb,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR to_regclass(
           'vec_bolsa_convocatorias.preparacion_confirmacion_kms_borrador'
       ) IS NULL
       OR to_regprocedure(
           'vec_bolsa_convocatorias.preparar_confirmacion_borrador_v2(jsonb,jsonb,jsonb,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar preparacion KMS v2';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_bolsa_convocatorias.preparar_confirmacion_borrador_v2(
    p_confirmacion jsonb,
    p_prueba jsonb,
    p_evidencia_cifrado jsonb,
    p_decision_canonica bytea,
    p_contexto_recurso_canonico bytea,
    p_material_canonico bytea,
    p_version_canonica bytea,
    p_aad_canonica bytea,
    p_material_clave_envuelto bytea,
    p_nonce bytea,
    p_texto_cifrado bytea
)
RETURNS TABLE (
    resultado text, estado_diario text, revision_diario bigint,
    cercado bigint, transaccion_ref text, accion text,
    estado_principal_ref text, estado_principal_revision bigint,
    estado_principal_huella_sha256 text,
    auditoria_ref text, huella_auditoria_sha256 text,
    evento_outbox_ref text, huella_evento_outbox_sha256 text,
    confirmada_en timestamptz, recibo jsonb,
    requiere_revalidacion_kms boolean, preparacion_ref text,
    recibo_cuerpo jsonb, cuerpo_recibo_canonico bytea,
    huella_cuerpo_recibo_sha256 text, preparada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_resultado record;
    v_preparada_en timestamptz;
BEGIN
    SELECT f.* INTO STRICT v_resultado
      FROM vec_bolsa_convocatorias.preparar_confirmacion_borrador_v1(
          p_confirmacion, p_prueba, p_evidencia_cifrado,
          p_decision_canonica, p_contexto_recurso_canonico,
          p_material_canonico, p_version_canonica, p_aad_canonica,
          p_material_clave_envuelto, p_nonce, p_texto_cifrado
      ) AS f;

    IF v_resultado.requiere_revalidacion_kms THEN
        IF v_resultado.resultado IS DISTINCT FROM 'preparada'
           OR v_resultado.estado_diario IS DISTINCT FROM 'en_curso'
           OR v_resultado.preparacion_ref IS NULL THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'fase A KMS no acredito preparacion cerrada';
        END IF;
        SELECT p.creada_en INTO STRICT v_preparada_en
          FROM vec_bolsa_convocatorias.preparacion_confirmacion_kms_borrador p
         WHERE p.preparacion_ref = v_resultado.preparacion_ref
           AND p.transaccion_ref = v_resultado.transaccion_ref
           AND p.confirmada_en = v_resultado.confirmada_en;
        IF v_preparada_en > date_trunc('microseconds', clock_timestamp())
           OR v_preparada_en >= v_resultado.confirmada_en
           OR v_resultado.confirmada_en - v_preparada_en <
              interval '100 milliseconds'
           OR v_resultado.confirmada_en - v_preparada_en >
              interval '5 seconds' THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'ventana temporal KMS incoherente';
        END IF;
    ELSIF v_resultado.preparacion_ref IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'resultado sin revalidacion conserva preparacion KMS';
    END IF;

    RETURN QUERY SELECT
        v_resultado.resultado::text,
        v_resultado.estado_diario::text,
        v_resultado.revision_diario::bigint,
        v_resultado.cercado::bigint,
        v_resultado.transaccion_ref::text,
        v_resultado.accion::text,
        v_resultado.estado_principal_ref::text,
        v_resultado.estado_principal_revision::bigint,
        v_resultado.estado_principal_huella_sha256::text,
        v_resultado.auditoria_ref::text,
        v_resultado.huella_auditoria_sha256::text,
        v_resultado.evento_outbox_ref::text,
        v_resultado.huella_evento_outbox_sha256::text,
        v_resultado.confirmada_en::timestamptz,
        v_resultado.recibo::jsonb,
        v_resultado.requiere_revalidacion_kms::boolean,
        v_resultado.preparacion_ref::text,
        v_resultado.recibo_cuerpo::jsonb,
        v_resultado.cuerpo_recibo_canonico::bytea,
        v_resultado.huella_cuerpo_recibo_sha256::text,
        v_preparada_en::timestamptz;
EXCEPTION WHEN NO_DATA_FOUND OR TOO_MANY_ROWS THEN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'fase A KMS no produjo un resultado unico';
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_bolsa_convocatorias.preparar_confirmacion_borrador_v2(
        jsonb,jsonb,jsonb,bytea,bytea,bytea,bytea,bytea,bytea,bytea,
        bytea
    ) FROM PUBLIC,
       vec_bolsa_convocatorias_ejecutor_consulta,
       vec_bolsa_convocatorias_registrador_atestacion,
       vec_bolsa_convocatorias_verificador_recibo;
REVOKE EXECUTE ON FUNCTION
    vec_bolsa_convocatorias.preparar_confirmacion_borrador_v1(
        jsonb,jsonb,jsonb,bytea,bytea,bytea,bytea,bytea,bytea,bytea,
        bytea
    ) FROM vec_bolsa_convocatorias_proyector_gobierno;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_convocatorias.preparar_confirmacion_borrador_v2(
        jsonb,jsonb,jsonb,bytea,bytea,bytea,bytea,bytea,bytea,bytea,
        bytea
    ) TO vec_bolsa_convocatorias_proyector_gobierno;

COMMENT ON FUNCTION
    vec_bolsa_convocatorias.preparar_confirmacion_borrador_v2(
        jsonb,jsonb,jsonb,bytea,bytea,bytea,bytea,bytea,bytea,bytea,
        bytea
    ) IS
    'Fase A runtime: devuelve separadamente preparada_en real y confirmada_en not-before. El KMS debe recibir preparada_en como instante de solicitud y terminar antes de confirmada_en.';

COMMIT;
