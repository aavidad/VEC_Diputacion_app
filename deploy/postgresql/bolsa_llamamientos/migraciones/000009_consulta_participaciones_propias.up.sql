-- B11: fachada exterior mínima de participaciones constituidas propias.
-- Preimagen: 000007/000008 conservan constitución inmutable y vínculo opaco.
-- Dependencia deliberada: AD3-000041 debe instalar el consumidor/revalidador V3
-- nominal B11 antes de esta migración. Esta migración no amplía V3 por sí misma.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000009', 0)
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass('vec_bolsa_llamamientos.vinculo_candidato') IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_bolsa_llamamientos.listar_participaciones_candidato_v1(text)'
       ) IS NULL
       OR pg_catalog.to_regrole(
           'vec_bolsa_llamamientos_consultor_participaciones_propias'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.registrar_consumo_participaciones_propias_b11_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.revalidar_consumo_participaciones_propias_b11_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_bolsa_llamamientos.consultar_participaciones_propias_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'dependencias incompatibles para consulta propia B11';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_bolsa_llamamientos.consultar_participaciones_propias_v1(
    p_consulta_canonica bytea,
    p_capacidad_canonica bytea,
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_contexto_actor_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric,
    p_payload_vec_ad_3 bytea,
    p_sobre_cose_sign1 bytea,
    p_evidencia_verificacion bytea,
    p_raiz_publica_spki bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
SET statement_timeout = '15s'
AS $funcion$
DECLARE
    v_consulta jsonb;
    v_decision jsonb;
    v_contexto jsonb;
    v_principal_ref text;
    v_candidato_ref text;
    v_numero_candidatos integer;
    v_preimagen_recurso text;
    v_huella_recurso text;
    v_consumo record;
    v_revalidacion record;
    v_resultado jsonb;
BEGIN
    -- La sesión exterior solo puede ostentar este único grupo nominal B11.
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR session_user = current_user
       OR pg_catalog.pg_is_in_recovery()
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members m
             JOIN pg_catalog.pg_roles r ON r.oid = m.roleid
             JOIN pg_catalog.pg_roles u ON u.oid = m.member
            WHERE u.rolname = session_user
              AND r.rolname = 'vec_bolsa_llamamientos_consultor_participaciones_propias'
              AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option
       )
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members m
             JOIN pg_catalog.pg_roles u ON u.oid = m.member
            WHERE u.rolname = session_user) <> 1
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles u
            WHERE u.rolname = session_user
              AND (NOT u.rolcanlogin OR NOT u.rolinherit OR u.rolsuper
                   OR u.rolcreatedb OR u.rolcreaterole OR u.rolreplication OR u.rolbypassrls)
       )
       OR pg_catalog.pg_has_role(session_user, 'vec_bolsa_llamamientos_propietario', 'MEMBER')
       OR pg_catalog.pg_has_role(session_user, 'vec_bolsa_llamamientos_migrador', 'MEMBER')
       OR pg_catalog.pg_has_role(session_user, 'vec_bolsa_llamamientos_ejecutor', 'MEMBER') THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'consulta propia B11 rechazada';
    END IF;

    IF p_consulta_canonica IS NULL
       OR pg_catalog.octet_length(p_consulta_canonica) NOT BETWEEN 1 AND 1024 THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'consulta propia B11 inválida';
    END IF;
    BEGIN
        v_consulta := pg_catalog.convert_from(p_consulta_canonica, 'UTF8')::jsonb;
        v_decision := pg_catalog.convert_from(p_decision_canonica, 'UTF8')::jsonb;
        v_contexto := pg_catalog.convert_from(p_contexto_actor_canonico, 'UTF8')::jsonb;
    EXCEPTION WHEN data_exception OR invalid_text_representation
              OR character_not_in_repertoire OR untranslatable_character THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'consulta propia B11 inválida';
    END;
    IF v_consulta <> '{"esquema":"vec.bolsa.consulta-participaciones-propias.b11.v1","version":1}'::jsonb
       OR v_contexto ->> 'esquema' <> 'vec.contexto-actor.vinculado.v2'
       OR v_contexto ->> 'persona_version' <> p_persona_version::text
       OR v_contexto ->> 'perfil_version' <> p_perfil_version::text
       OR pg_catalog.jsonb_typeof(v_contexto -> 'vinculos') <> 'array' THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'consulta propia B11 inválida';
    END IF;
    IF v_decision -> 'campos_permitidos' IS DISTINCT FROM
          '["bolsa_ref","categoria_ref","estado_bolsa","orden","total_participaciones","version_bolsa","vigente_desde","vigente_hasta"]'::jsonb
       OR v_decision -> 'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'consulta propia B11 rechazada';
    END IF;

    -- El consumidor V3 registra la auditoría antes de cualquier lectura propia.
    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3
           .registrar_consumo_participaciones_propias_b11_v3(
               p_capacidad_canonica, p_decision_canonica, p_motivo_canonico,
               p_contexto_actor_canonico, p_persona_version, p_perfil_version,
               p_payload_vec_ad_3, p_sobre_cose_sign1, p_evidencia_verificacion,
               p_raiz_publica_spki);
    -- Una lectura no se recupera con un consumo y auditoría históricos: el
    -- siguiente intento requiere una concesión V3 nueva, antes de cualquier
    -- revalidación o acceso a la proyección propia.
    IF v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'P0663',
            MESSAGE = 'consulta propia B11 requiere consumo nuevo';
    END IF;
    SELECT * INTO STRICT v_revalidacion
      FROM vec_autorizacion_atestada_v3
           .revalidar_consumo_participaciones_propias_b11_v3(
               p_capacidad_canonica, p_decision_canonica, p_motivo_canonico,
               p_contexto_actor_canonico, p_persona_version, p_perfil_version,
               p_payload_vec_ad_3, p_sobre_cose_sign1, p_evidencia_verificacion,
               p_raiz_publica_spki);
    IF v_consumo.decision_ref IS NULL OR v_consumo.auditoria_ref IS NULL
       OR v_revalidacion.decision_ref IS DISTINCT FROM v_consumo.decision_ref THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'consulta propia B11 rechazada';
    END IF;

    -- El principal V2 es siempre persona. El candidato se deriva después de
    -- revalidar, de un único vínculo activo y nunca de una entrada exterior.
    v_principal_ref := v_contexto ->> 'principal_ref';
    IF v_principal_ref IS DISTINCT FROM v_contexto ->> 'persona_ref'
       OR v_principal_ref !~ '^per_[A-Za-z0-9_-]{22,128}$' THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'consulta propia B11 rechazada';
    END IF;
    SELECT pg_catalog.count(*), pg_catalog.min(e ->> 'referencia')
      INTO v_numero_candidatos, v_candidato_ref
      FROM pg_catalog.jsonb_array_elements(v_contexto -> 'vinculos') e
     WHERE e ->> 'tipo' = 'candidato'
       AND e ->> 'estado' = 'activo'
       AND e ->> 'referencia' ~ '^can_[A-Za-z0-9_-]{22,128}$';
    IF v_numero_candidatos <> 1
       OR pg_catalog.to_json(v_candidato_ref)::text IS DISTINCT FROM
          '"' || v_candidato_ref || '"' THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'consulta propia B11 rechazada';
    END IF;
    v_preimagen_recurso := '{"ambitos":{"candidato_ref":' || pg_catalog.to_json(v_candidato_ref)::text
        || '},"atributos":{"finalidad":"consulta_posicion_propia_bolsa"}}';
    v_huella_recurso := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_preimagen_recurso,'UTF8')),'hex');
    IF v_decision ->> 'recurso_ref' IS DISTINCT FROM
          'candidato:' || v_candidato_ref
       OR v_consumo.efecto_ref IS DISTINCT FROM
          'candidato:' || v_candidato_ref
       OR v_decision ->> 'contexto_recurso_huella_sha256' IS DISTINCT FROM v_huella_recurso
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM
          v_huella_recurso THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'consulta propia B11 rechazada';
    END IF;
    SELECT pg_catalog.jsonb_build_object(
        'esquema', 'vec.bolsa.mi-bolsa.v1',
        'consultada_en', pg_catalog.to_char(
            pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp()) AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'participaciones', COALESCE(pg_catalog.jsonb_agg(
            pg_catalog.jsonb_build_object(
                'bolsa_ref', p.bolsa_ref,
                'categoria_ref', p.categoria_ref,
                'version_bolsa', p.version_bolsa,
                'orden', p.orden,
                'total_participaciones', p.total_participaciones,
                'estado_bolsa', p.estado,
                'vigente_desde', p.vigente_desde,
                'vigente_hasta', p.vigente_hasta
            ) ORDER BY p.bolsa_ref, p.version_bolsa, p.orden
        ), '[]'::jsonb)
    ) INTO v_resultado
      FROM vec_bolsa_llamamientos.listar_participaciones_candidato_v1(v_candidato_ref) p;
    RETURN v_resultado;
EXCEPTION WHEN no_data_found OR too_many_rows OR invalid_text_representation
              OR numeric_value_out_of_range OR datetime_field_overflow THEN
    RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'consulta propia B11 inválida';
END
$funcion$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_participaciones_propias_v1(
    bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea
) FROM PUBLIC, vec_bolsa_llamamientos_ejecutor, vec_bolsa_llamamientos_migrador;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos
    TO vec_bolsa_llamamientos_consultor_participaciones_propias;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_participaciones_propias_v1(
    bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea
) TO vec_bolsa_llamamientos_consultor_participaciones_propias;
COMMIT;
