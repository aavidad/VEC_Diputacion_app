CREATE FUNCTION vec_contratacion_temporal.confirmar_alta_atestada_v2(
    p_capacidad_canonica bytea, p_decision_canonica bytea,
    p_motivo_canonico bytea, p_contexto_actor_canonico bytea,
    p_persona_version numeric, p_perfil_version numeric,
    p_payload_vec_ad_3 bytea, p_sobre_cose_sign1 bytea,
    p_evidencia_verificacion bytea, p_raiz_publica_spki bytea,
    p_alta_canonica bytea, p_sellos_hmac_canonicos bytea
)
RETURNS TABLE (
    expediente_ref text, numero_visible text, version numeric,
    recibo_ref text, auditoria_ref text, evento_ref text,
    confirmada_en timestamptz, recibo_huella_sha256 text
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET TimeZone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '15s'
SET idle_in_transaction_session_timeout = '20s'
AS $funcion$
DECLARE
    a jsonb;
    s jsonb;
    v_pares jsonb;
    v_raices text[];
    v_candidatura vec_contratacion_temporal.candidatura_alta_tecnica%ROWTYPE;
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(session_user,
           'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(session_user,
           'vec_contratacion_temporal_migrador', 'MEMBER')
       OR pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'confirmacion de alta rechazada';
    END IF;
    BEGIN
        a := pg_catalog.convert_from(p_alta_canonica, 'UTF8')::jsonb;
        s := pg_catalog.convert_from(p_sellos_hmac_canonicos, 'UTF8')::jsonb;
    EXCEPTION WHEN data_exception OR invalid_text_representation
      OR character_not_in_repertoire OR untranslatable_character THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'candidatura de alta invalida';
    END;
    IF pg_catalog.jsonb_typeof(a) <> 'object'
       OR pg_catalog.jsonb_typeof(s) <> 'object'
       OR pg_catalog.jsonb_typeof(s -> 'activo') <> 'object'
       OR pg_catalog.jsonb_typeof(s -> 'retenidos') <> 'array'
       OR pg_catalog.jsonb_array_length(s -> 'retenidos') > 3 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'candidatura de alta invalida';
    END IF;
    v_pares := pg_catalog.jsonb_build_array(s -> 'activo') || (s -> 'retenidos');
    SELECT pg_catalog.array_agg(DISTINCT alias.ambito_raiz_hmac
                                ORDER BY alias.ambito_raiz_hmac)
      INTO v_raices
      FROM pg_catalog.jsonb_array_elements(v_pares) AS p(valor)
      JOIN vec_contratacion_temporal.candidatura_alta_alias alias
        ON alias.generacion = (p.valor ->> 'generacion')::integer
       AND alias.ambito_hmac = p.valor ->> 'ambito_hmac'
       AND alias.huella_hmac = p.valor ->> 'huella_hmac';
    IF pg_catalog.cardinality(v_raices) <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'candidatura de alta no acreditada';
    END IF;
    SELECT * INTO STRICT v_candidatura
      FROM vec_contratacion_temporal.candidatura_alta_tecnica
     WHERE ambito_raiz_hmac = v_raices[1];
    IF (SELECT pg_catalog.count(*)
          FROM pg_catalog.jsonb_array_elements(v_pares) AS p(valor)
          JOIN vec_contratacion_temporal.candidatura_alta_alias alias
            ON alias.ambito_raiz_hmac = v_candidatura.ambito_raiz_hmac
           AND alias.generacion = (p.valor ->> 'generacion')::integer
           AND alias.ambito_hmac = p.valor ->> 'ambito_hmac'
           AND alias.huella_hmac = p.valor ->> 'huella_hmac')
       <> pg_catalog.jsonb_array_length(v_pares)
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.jsonb_array_elements(v_pares) AS p(valor)
            WHERE p.valor ->> 'ambito_hmac' = v_candidatura.ambito_raiz_hmac
              AND p.valor ->> 'huella_hmac' = v_candidatura.huella_raiz_hmac
       )
       OR a ->> 'reserva_ref' <> v_candidatura.reserva_ref
       OR a ->> 'expediente_ref' <> v_candidatura.expediente_ref
       OR a ->> 'numero_visible' <> v_candidatura.numero_visible
       OR a ->> 'recibo_ref' <> v_candidatura.recibo_ref
       OR a ->> 'organizacion_ref' <> v_candidatura.organizacion_ref
       OR a ->> 'actor_ref' <> v_candidatura.actor_ref
       OR a ->> 'perfil_ref' <> v_candidatura.perfil_ref
       OR a ->> 'creado_en' <>
          vec_contratacion_temporal.instante_utc_v1(v_candidatura.instante_efecto)
       OR a ->> 'actualizado_en' <>
          vec_contratacion_temporal.instante_utc_v1(v_candidatura.instante_efecto)
       OR a #>> '{actuacion,realizada_en}' <>
          vec_contratacion_temporal.instante_utc_v1(v_candidatura.instante_efecto) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'candidatura de alta no acreditada';
    END IF;
    RETURN QUERY SELECT *
      FROM vec_contratacion_temporal.confirmar_alta_atestada_v1(
          p_capacidad_canonica, p_decision_canonica, p_motivo_canonico,
          p_contexto_actor_canonico, p_persona_version, p_perfil_version,
          p_payload_vec_ad_3, p_sobre_cose_sign1, p_evidencia_verificacion,
          p_raiz_publica_spki, p_alta_canonica, p_sellos_hmac_canonicos
      );
END
$funcion$;
