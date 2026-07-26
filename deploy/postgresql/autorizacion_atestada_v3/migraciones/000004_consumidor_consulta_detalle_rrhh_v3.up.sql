BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v3:migracion:000004', 0
    )
);

DO $prevalidacion$
DECLARE
    v_definicion text;
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_cuadro_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_detalle_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para consulta detalle RRHH VEC-AD-3';
    END IF;
    SELECT pg_catalog.regexp_replace(
               pg_catalog.pg_get_constraintdef(c.oid, true),
               '\s+', ' ', 'g'
           )
      INTO v_definicion
      FROM pg_catalog.pg_constraint c
     WHERE c.conrelid =
           'vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND c.conname =
           'clave_capacidad_version_audiencia_consumo_check'
       AND c.contype = 'c'
       AND c.convalidated
       AND c.conkey = ARRAY[8]::smallint[];
    IF v_definicion IS NULL
       OR v_definicion <>
          'CHECK (audiencia_consumo = ANY (ARRAY[''vec_contratacion_temporal.confirmar_alta_atestada.v1''::text, ''vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1''::text]))' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'gobierno de audiencias de consulta incompleto';
    END IF;
END
$prevalidacion$;

ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
    DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
    ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check
    CHECK (audiencia_consumo IN (
        'vec_contratacion_temporal.confirmar_alta_atestada.v1',
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
        'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
    )) NOT VALID;
ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
    VALIDATE CONSTRAINT
        clave_capacidad_version_audiencia_consumo_check;

CREATE FUNCTION
    vec_autorizacion_atestada_v3
    .registrar_y_consumir_consulta_detalle_rrhh_v3_atestada(
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
)
RETURNS TABLE (
    decision_ref text,
    efecto_ref text,
    huella_efecto_sha256 text,
    consumo_huella_sha256 text,
    auditoria_ref text,
    auditoria_huella_sha256 text,
    consumida_en timestamptz,
    consumo_nuevo boolean
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    c jsonb;
    d jsonb;
BEGIN
    IF vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(
           p_capacidad_canonica
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'consulta detalle RRHH VEC-AD-3 inválida';
    END IF;
    BEGIN
        c := pg_catalog.convert_from(
            p_capacidad_canonica, 'UTF8'
        )::jsonb;
        d := pg_catalog.convert_from(
            p_decision_canonica, 'UTF8'
        )::jsonb;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
          OR character_not_in_repertoire
          OR untranslatable_character THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'consulta detalle RRHH VEC-AD-3 inválida';
    END;
    IF c ->> 'audiencia_consumo' IS DISTINCT FROM
           'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
       OR c ->> 'operacion' IS DISTINCT FROM
           'contratacion_temporal.expediente.consultar'
       OR d ->> 'accion' IS DISTINCT FROM
           'contratacion_temporal.expediente.consultar'
       OR d ->> 'modulo_id' IS DISTINCT FROM
           'contratacion_temporal'
       OR d ->> 'tipo_recurso' IS DISTINCT FROM
           'expediente_contratacion_temporal'
       OR d ->> 'finalidad' IS DISTINCT FROM
           'tramitacion_expediente_contratacion_temporal'
       OR d ->> 'recurso_ref' IS DISTINCT FROM c ->> 'efecto_ref'
       OR d ->> 'contexto_recurso_huella_sha256' IS DISTINCT FROM
           c ->> 'huella_efecto_sha256' THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'consulta detalle RRHH VEC-AD-3 rechazada';
    END IF;
    RETURN QUERY
    SELECT *
      FROM vec_autorizacion_atestada_v3
           .consumir_consulta_rrhh_v3_interna(
          'detalle', p_capacidad_canonica, p_decision_canonica,
          p_motivo_canonico, p_contexto_actor_canonico,
          p_persona_version, p_perfil_version,
          p_payload_vec_ad_3, p_sobre_cose_sign1,
          p_evidencia_verificacion, p_raiz_publica_spki
      );
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3
    .registrar_y_consumir_consulta_detalle_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    ) FROM PUBLIC, vec_autorizacion_atestada_v3_consumidor,
           vec_autorizacion_atestada_v3_emisor;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v3
    .registrar_y_consumir_consulta_detalle_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    ) TO vec_contratacion_temporal_propietario;
COMMIT;
