\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:000048:replay-alta:o2-07', 0
));
DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.confirmar_alta_atestada_v2(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.reconciliar_agregado_alta_v1(bytea,text,text,text,text,text,text)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.confirmar_alta_atestada_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para replay de alta';
    END IF;
END
$prevalidacion$;

DO $tipar_estado_previo$
DECLARE
    v_definicion text;
    v_sin_tipo text := E'ERRCODE = ''55000'',\n            MESSAGE = ''estado previo de alta incoherente'';';
    v_tipado text := E'ERRCODE = ''V2070'',\n            MESSAGE = ''estado previo de alta incoherente'';';
BEGIN
    SELECT pg_catalog.pg_get_functiondef(
        'vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)'::regprocedure
    ) INTO STRICT v_definicion;
    IF pg_catalog.length(v_definicion) - pg_catalog.length(
           pg_catalog.replace(v_definicion, v_sin_tipo, '')
       ) <> pg_catalog.length(v_sin_tipo)
       OR pg_catalog.strpos(v_definicion, v_tipado) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'contrato de estado previo de alta incompatible';
    END IF;
    EXECUTE pg_catalog.replace(v_definicion, v_sin_tipo, v_tipado);
END
$tipar_estado_previo$;

CREATE FUNCTION vec_contratacion_temporal.confirmar_alta_atestada_v3(
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
    v_confirmacion
        vec_contratacion_temporal.confirmacion_agregado_alta%ROWTYPE;
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
        RETURN QUERY SELECT *
          FROM vec_contratacion_temporal.confirmar_alta_atestada_v2(
              p_capacidad_canonica, p_decision_canonica,
              p_motivo_canonico, p_contexto_actor_canonico,
              p_persona_version, p_perfil_version,
              p_payload_vec_ad_3, p_sobre_cose_sign1,
              p_evidencia_verificacion, p_raiz_publica_spki,
              p_alta_canonica, p_sellos_hmac_canonicos
          );
        RETURN;
    EXCEPTION WHEN SQLSTATE 'V2070' THEN
        NULL;
    END;

    BEGIN
        a := pg_catalog.convert_from(p_alta_canonica, 'UTF8')::jsonb;
        s := pg_catalog.convert_from(p_sellos_hmac_canonicos, 'UTF8')::jsonb;
    EXCEPTION WHEN data_exception OR invalid_text_representation
      OR character_not_in_repertoire OR untranslatable_character THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'replay de alta invalido';
    END;
    IF pg_catalog.jsonb_typeof(a) <> 'object'
       OR pg_catalog.jsonb_typeof(s) <> 'object'
       OR pg_catalog.jsonb_typeof(s -> 'activo') <> 'object' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'replay de alta invalido';
    END IF;

    SELECT confirmacion.* INTO STRICT v_confirmacion
      FROM vec_contratacion_temporal.confirmacion_agregado_alta confirmacion
     WHERE confirmacion.expediente_ref = a ->> 'expediente_ref'
       AND confirmacion.reserva_ref = a ->> 'reserva_ref'
       AND confirmacion.numero_visible = a ->> 'numero_visible'
       AND confirmacion.recibo_ref = a ->> 'recibo_ref'
       AND confirmacion.ambito_hmac = s #>> '{activo,ambito_hmac}';

    RETURN QUERY SELECT *
      FROM vec_contratacion_temporal.reconciliar_agregado_alta_v1(
          p_alta_canonica,
          s #>> '{activo,ambito_hmac}',
          s #>> '{activo,huella_hmac}',
          v_confirmacion.decision_ref,
          v_confirmacion.efecto_ref,
          v_confirmacion.huella_efecto_sha256,
          v_confirmacion.consumo_huella_sha256
      );
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'integridad del replay de alta no acreditada';
    END IF;
END
$funcion$;

ALTER FUNCTION vec_contratacion_temporal.confirmar_alta_atestada_v3(
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea, bytea, bytea
) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.confirmar_alta_atestada_v2(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea, bytea, bytea
    ),
    vec_contratacion_temporal.confirmar_alta_atestada_v3(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea, bytea, bytea
    ) FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.confirmar_alta_atestada_v3(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea, bytea, bytea
    ) TO vec_contratacion_temporal_ejecutor;
COMMIT;
