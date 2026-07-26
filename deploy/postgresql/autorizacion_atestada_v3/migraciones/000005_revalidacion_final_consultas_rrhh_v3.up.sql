-- Revalidación final, nominal y sin segundo consumo para consultas RRHH.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v3:migracion:000005', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_cuadro_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_detalle_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.revalidar_consumo_consulta_cuadro_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.revalidar_consumo_consulta_detalle_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NOT NULL
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_contratacion_temporal_propietario'
              AND NOT rolcanlogin
              AND NOT rolsuper
              AND NOT rolcreatedb
              AND NOT rolcreaterole
              AND NOT rolinherit
              AND NOT rolreplication
              AND NOT rolbypassrls
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para revalidación final RRHH';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION
vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(
    p_perfil_consulta text,
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
    consumo_huella_sha256 text,
    revalidada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '1s'
AS $funcion$
DECLARE
    c jsonb;
    d jsonb;
    x jsonb;
    v_clave record;
    v_puntero_clave record;
    v_config record;
    v_raiz record;
    v_consumo record;
    v_ahora timestamptz(6);
    v_viva_en timestamptz(6);
    v_huella_consumo text;
    v_statement numeric;
    v_idle numeric;
    v_audiencia text;
    v_operacion text;
    v_tipo_recurso text;
    v_finalidad text;
BEGIN
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR current_user <> 'vec_autorizacion_atestada_v3_propietario'
       OR session_user = current_user
       OR pg_catalog.pg_is_in_recovery()
       OR pg_catalog.to_regrole(
           'vec_contratacion_temporal_consultor_rrhh'
       ) IS NULL
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members m
             JOIN pg_catalog.pg_roles r ON r.oid = m.roleid
             JOIN pg_catalog.pg_roles u ON u.oid = m.member
            WHERE u.rolname = session_user
              AND r.rolname =
                  'vec_contratacion_temporal_consultor_rrhh'
              AND NOT m.admin_option
              AND m.inherit_option
              AND NOT m.set_option
       )
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_auth_members m
             JOIN pg_catalog.pg_roles u ON u.oid = m.member
            WHERE u.rolname = session_user
       ) <> 1
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_roles u
            WHERE u.rolname = session_user
              AND (
                  NOT u.rolcanlogin OR NOT u.rolinherit OR u.rolsuper
                  OR u.rolcreatedb OR u.rolcreaterole
                  OR u.rolreplication OR u.rolbypassrls
              )
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members m
             JOIN pg_catalog.pg_roles g ON g.oid = m.member
            WHERE g.rolname =
                  'vec_contratacion_temporal_consultor_rrhh'
       )
       OR pg_catalog.pg_has_role(
           session_user,
           'vec_autorizacion_atestada_v3_propietario', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user,
           'vec_autorizacion_atestada_v3_migrador', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user,
           'vec_autorizacion_atestada_v3_emisor', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user,
           'vec_autorizacion_atestada_v3_consumidor', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user,
           'vec_contratacion_temporal_confirmador_cobertura', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user,
           'vec_contratacion_temporal_gobernador', 'MEMBER')
       OR COALESCE((
           SELECT pg_catalog.pg_has_role(session_user, r.oid, 'MEMBER')
             FROM pg_catalog.pg_roles r
            WHERE r.rolname =
                  'vec_contratacion_temporal_lector_resultado_cobertura'
       ), false) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'revalidación final RRHH rechazada';
    END IF;

    SELECT setting::numeric INTO v_statement
      FROM pg_catalog.pg_settings
     WHERE name = 'statement_timeout' AND unit = 'ms';
    SELECT setting::numeric INTO v_idle
      FROM pg_catalog.pg_settings
     WHERE name = 'idle_in_transaction_session_timeout' AND unit = 'ms';
    IF v_statement IS NULL OR v_statement NOT BETWEEN 1 AND 15000
       OR v_idle IS NULL OR v_idle NOT BETWEEN 1 AND 20000
       OR p_perfil_consulta NOT IN ('cuadro', 'detalle')
       OR vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(
           p_capacidad_canonica
       ) IS NOT TRUE
       OR p_decision_canonica IS NULL
       OR p_motivo_canonico IS NULL
       OR p_contexto_actor_canonico IS NULL
       OR p_persona_version IS NULL
       OR p_perfil_version IS NULL
       OR p_payload_vec_ad_3 IS NULL
       OR p_sobre_cose_sign1 IS NULL
       OR p_evidencia_verificacion IS NULL
       OR p_raiz_publica_spki IS NULL
       OR pg_catalog.octet_length(p_decision_canonica)
          NOT BETWEEN 1 AND 524288
       OR pg_catalog.octet_length(p_motivo_canonico)
          NOT BETWEEN 1 AND 65536
       OR pg_catalog.octet_length(p_contexto_actor_canonico)
          NOT BETWEEN 1 AND 262144
       OR pg_catalog.octet_length(p_payload_vec_ad_3)
          NOT BETWEEN 1 AND 1048576
       OR pg_catalog.octet_length(p_sobre_cose_sign1)
          NOT BETWEEN 1 AND 1048576
       OR pg_catalog.octet_length(p_evidencia_verificacion)
          NOT BETWEEN 1 AND 262144
       OR pg_catalog.octet_length(p_raiz_publica_spki) <> 44
       OR pg_catalog.substr(p_raiz_publica_spki, 1, 12) <>
          pg_catalog.decode('302a300506032b6570032100', 'hex')
       OR p_persona_version NOT BETWEEN
          1 AND 9007199254740991::numeric
       OR p_perfil_version NOT BETWEEN
          1 AND 9007199254740991::numeric
       OR pg_catalog.scale(p_persona_version) <> 0
       OR pg_catalog.scale(p_perfil_version) <> 0 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'entrada de revalidación final RRHH inválida';
    END IF;

    BEGIN
        c := pg_catalog.convert_from(
            p_capacidad_canonica, 'UTF8'
        )::jsonb;
        d := pg_catalog.convert_from(
            p_decision_canonica, 'UTF8'
        )::jsonb;
        x := pg_catalog.convert_from(
            p_contexto_actor_canonico, 'UTF8'
        )::jsonb;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
          OR character_not_in_repertoire
          OR untranslatable_character THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'entrada de revalidación final RRHH inválida';
    END;

    IF p_perfil_consulta = 'cuadro' THEN
        v_audiencia :=
            'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1';
        v_operacion := 'contratacion_temporal.cuadro.consultar';
        v_tipo_recurso := 'cuadro_rrhh_contratacion_temporal';
        v_finalidad := 'gestion_operativa_contratacion_temporal';
    ELSE
        v_audiencia :=
            'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1';
        v_operacion := 'contratacion_temporal.expediente.consultar';
        v_tipo_recurso := 'expediente_contratacion_temporal';
        v_finalidad := 'tramitacion_expediente_contratacion_temporal';
    END IF;

    IF vec_autorizacion_atestada_v3.capacidad_tipos_validos(c) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.capacidad_canonica(c)
          IS DISTINCT FROM p_capacidad_canonica
       OR c ->> 'esquema' <>
          'vec.autorizacion.capacidad-registro-consumo-atestado.v3'
       OR c ->> 'version' <> '3'
       OR c ->> 'audiencia_consumo' <> v_audiencia
       OR c ->> 'operacion' <> v_operacion
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'
       OR d ->> 'accion' <> v_operacion
       OR d ->> 'modulo_id' <> 'contratacion_temporal'
       OR d ->> 'tipo_recurso' <> v_tipo_recurso
       OR d ->> 'finalidad' <> v_finalidad
       OR d ->> 'recurso_ref' <> c ->> 'efecto_ref'
       OR d ->> 'contexto_recurso_huella_sha256' <>
          c ->> 'huella_efecto_sha256'
       OR d ->> 'decision_ref' <> c ->> 'decision_ref'
       OR d ->> 'motivo_huella_sha256' <>
          c ->> 'huella_motivo_sha256'
       OR d ->> 'valida_hasta' <> c ->> 'decision_valida_hasta'
       OR d #>> '{vinculo_autenticacion_actor,registro_contexto_ref}' <>
          c ->> 'contexto_ref'
       OR d #>> (
          '{vinculo_autenticacion_actor,contexto_actor_huella_sha256}'
       ) <> c ->> 'huella_contexto_sha256'
       OR d ->> 'principal_id' <> x ->> 'principal_ref'
       OR d ->> 'perfil_activo_ref' <> x ->> 'perfil_activo_ref'
       OR x ->> 'esquema' <> 'vec.contexto-actor.vinculado.v2'
       OR x ->> 'persona_version' <> p_persona_version::text
       OR x ->> 'perfil_version' <> p_perfil_version::text
       OR pg_catalog.encode(
           pg_catalog.sha256(p_decision_canonica), 'hex'
       ) <> c ->> 'huella_decision_sha256'
       OR pg_catalog.encode(
           pg_catalog.sha256(p_motivo_canonico), 'hex'
       ) <> c ->> 'huella_motivo_sha256'
       OR pg_catalog.encode(
           pg_catalog.sha256(p_contexto_actor_canonico), 'hex'
       ) <> c ->> 'huella_contexto_sha256'
       OR pg_catalog.encode(
           pg_catalog.sha256(p_payload_vec_ad_3), 'hex'
       ) <> c ->> 'huella_payload_vec_ad_3_sha256'
       OR pg_catalog.encode(
           pg_catalog.sha256(p_sobre_cose_sign1), 'hex'
       ) <> c ->> 'huella_sobre_cose_sign1_sha256'
       OR pg_catalog.encode(
           pg_catalog.sha256(p_evidencia_verificacion), 'hex'
       ) <> c ->> 'huella_prueba_confianza_sha256'
       OR pg_catalog.encode(
           pg_catalog.sha256(p_raiz_publica_spki), 'hex'
       ) <> c ->> 'huella_raiz_spki_sha256' THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'ligadura de revalidación final RRHH rechazada';
    END IF;

    PERFORM 1
      FROM vec_autorizacion_atestada_v3.checkpoint_gobierno
     WHERE control_id
     FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'gobierno VEC-AD-3 no disponible';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(
            'vec_autorizacion_atestada_v3:decision:'
            || (c ->> 'decision_ref'), 0
        )
    );

    SELECT t.*, a.nonce, a.consumo_huella_sha256, a.consumida_en,
           u.auditoria_ref, u.huella_sha256 AS auditoria_huella_sha256
      INTO v_consumo
      FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 t
      JOIN vec_autorizacion_atestada_v3.consumo_decision_v3 a
        USING (decision_ref, efecto_ref, huella_efecto_sha256)
      JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 u
        USING (decision_ref, efecto_ref, huella_efecto_sha256)
     WHERE t.decision_ref = c ->> 'decision_ref';
    v_huella_consumo := pg_catalog.encode(
        pg_catalog.sha256(
            vec_autorizacion_atestada_v3.encuadrar_mac(
                pg_catalog.encode(p_capacidad_canonica, 'base64')
            )
            || vec_autorizacion_atestada_v3.encuadrar_mac(
                pg_catalog.encode(p_decision_canonica, 'base64')
            )
            || vec_autorizacion_atestada_v3.encuadrar_mac(
                pg_catalog.encode(p_contexto_actor_canonico, 'base64')
            )
            || vec_autorizacion_atestada_v3.encuadrar_mac(
                c ->> 'efecto_ref'
            )
            || vec_autorizacion_atestada_v3.encuadrar_mac(
                c ->> 'huella_efecto_sha256'
            )
        ),
        'hex'
    );
    IF NOT FOUND
       OR v_consumo.decision_canonica <> p_decision_canonica
       OR v_consumo.motivo_canonico <> p_motivo_canonico
       OR v_consumo.contexto_actor_canonico <>
          p_contexto_actor_canonico
       OR v_consumo.payload_vec_ad_3 <> p_payload_vec_ad_3
       OR v_consumo.sobre_cose_sign1 <> p_sobre_cose_sign1
       OR v_consumo.evidencia_verificacion <>
          p_evidencia_verificacion
       OR v_consumo.raiz_publica_spki <> p_raiz_publica_spki
       OR v_consumo.capacidad_canonica <> p_capacidad_canonica
       OR v_consumo.huella_capacidad_sha256 <>
          pg_catalog.encode(
              pg_catalog.sha256(p_capacidad_canonica), 'hex'
          )
       OR v_consumo.huella_decision_sha256 <>
          c ->> 'huella_decision_sha256'
       OR v_consumo.efecto_ref <> c ->> 'efecto_ref'
       OR v_consumo.huella_efecto_sha256 <>
          c ->> 'huella_efecto_sha256'
       OR v_consumo.nonce <> c ->> 'nonce'
       OR v_consumo.consumo_huella_sha256 <> v_huella_consumo
       OR v_consumo.auditoria_ref IS NULL
       OR v_consumo.auditoria_huella_sha256 IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'consumo durable RRHH no acreditado';
    END IF;

    v_ahora := pg_catalog.clock_timestamp();
    SELECT k.* INTO v_clave
      FROM vec_autorizacion_atestada_v3.clave_capacidad_version k
     WHERE k.clave_id = c ->> 'clave_id'
       AND k.version = (c ->> 'clave_version')::numeric
     FOR SHARE;
    SELECT p.* INTO v_puntero_clave
      FROM vec_autorizacion_atestada_v3.puntero_clave_emision p
     WHERE p.clave_id = c ->> 'clave_id'
       AND p.version = (c ->> 'clave_version')::numeric
       AND p.establecida_en <= v_ahora
     ORDER BY p.orden DESC
     LIMIT 1
     FOR SHARE;
    SELECT cfg.*, cp.configuracion_secuencia_minima,
           cp.raiz_version_minima
      INTO v_config
      FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p
      JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version cfg
        ON cfg.revision = p.configuracion_revision
      JOIN vec_autorizacion_atestada_v3.checkpoint_gobierno cp
        ON cp.control_id
     WHERE p.establecida_en <= v_ahora
     ORDER BY p.orden DESC
     LIMIT 1
     FOR SHARE OF p, cfg;
    SELECT r.* INTO v_raiz
      FROM vec_autorizacion_atestada_v3.configuracion_raiz cr
      JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r
        ON r.clave_id = cr.raiz_clave_id
       AND r.version = cr.raiz_version
     WHERE cr.configuracion_revision = v_config.revision
       AND r.clave_id = c ->> 'raiz_clave_id'
       AND r.version = (c ->> 'raiz_version')::numeric
     FOR SHARE OF r;

    IF v_clave.clave_id IS NULL
       OR v_puntero_clave.clave_id IS NULL
       OR v_config.revision IS NULL
       OR v_raiz.clave_id IS NULL
       OR v_clave.revision_gobierno <>
          (c ->> 'revision_gobierno')::numeric
       OR v_clave.huella_gobierno_sha256 <>
          c ->> 'huella_gobierno_sha256'
       OR v_clave.emisor_id <> c ->> 'emisor_id'
       OR v_clave.audiencia_consumo <> v_audiencia
       OR (c ->> 'emitida_en')::timestamptz < v_clave.valida_desde
       OR (c ->> 'expira_en')::timestamptz > v_clave.valida_hasta
       OR v_config.revision <> c ->> 'revision_confianza'
       OR v_config.secuencia <>
          (c ->> 'configuracion_secuencia')::numeric
       OR v_config.secuencia < v_config.configuracion_secuencia_minima
       OR v_config.huella_configuracion_sha256 <>
          c ->> 'huella_configuracion_sha256'
       OR v_config.publicada_en <>
          (c ->> 'configuracion_publicada_en')::timestamptz
       OR v_config.expira_en <>
          (c ->> 'configuracion_expira_en')::timestamptz
       OR v_raiz.version < v_config.raiz_version_minima
       OR v_raiz.huella_spki_sha256 <>
          c ->> 'huella_raiz_spki_sha256'
       OR v_raiz.clave_publica_spki <> p_raiz_publica_spki
       OR v_raiz.valida_desde <>
          (c ->> 'raiz_valida_desde')::timestamptz
       OR v_raiz.valida_hasta <>
          (c ->> 'raiz_valida_hasta')::timestamptz
       OR v_raiz.suite <> c ->> 'suite'
       OR v_raiz.audiencia_despliegue <>
          c ->> 'audiencia_despliegue'
       OR (c ->> 'verificada_en')::timestamptz <
          v_config.publicada_en
       OR (c ->> 'verificada_en')::timestamptz >= v_config.expira_en
       OR (c ->> 'verificada_en')::timestamptz < v_raiz.valida_desde
       OR (c ->> 'verificada_en')::timestamptz >= v_raiz.valida_hasta
       OR vec_autorizacion_atestada_v3.bytea_igual_constante(
           public.hmac(
               vec_autorizacion_atestada_v3.preimagen_mac(c),
               v_clave.secreto_hmac,
               'sha256'
           ),
           pg_catalog.decode(c ->> 'mac_sha256', 'hex')
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'gobierno de revalidación final RRHH rechazado';
    END IF;

    IF v_ahora < (c ->> 'emitida_en')::timestamptz
       OR v_ahora >= (c ->> 'expira_en')::timestamptz
       OR v_ahora >= (c ->> 'decision_valida_hasta')::timestamptz
       OR v_ahora >= v_config.expira_en
       OR v_ahora >= v_raiz.valida_hasta
       OR EXISTS (
           SELECT 1
             FROM vec_autorizacion_atestada_v3
                  .revocacion_clave_capacidad r
            WHERE r.clave_id = v_clave.clave_id
              AND r.version = v_clave.version
              AND r.revocada_en <= v_ahora
       )
       OR EXISTS (
           SELECT 1
             FROM vec_autorizacion_atestada_v3
                  .revocacion_configuracion r
            WHERE r.configuracion_revision = v_config.revision
              AND r.revocada_en <= v_ahora
       )
       OR EXISTS (
           SELECT 1
             FROM vec_autorizacion_atestada_v3.revocacion_raiz r
            WHERE r.raiz_clave_id = v_raiz.clave_id
              AND r.raiz_version = v_raiz.version
              AND r.revocada_en <= v_ahora
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'vigencia final RRHH agotada';
    END IF;

    v_viva_en :=
        vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
            p_decision_canonica,
            p_motivo_canonico,
            p_persona_version,
            p_perfil_version
        );
    v_ahora := pg_catalog.clock_timestamp();
    IF v_viva_en IS NULL
       OR v_viva_en > v_ahora
       OR v_viva_en < v_consumo.consumida_en
       OR v_ahora >= (c ->> 'expira_en')::timestamptz
       OR v_ahora >= (c ->> 'decision_valida_hasta')::timestamptz
       OR v_ahora >= v_config.expira_en
       OR v_ahora >= v_raiz.valida_hasta
       OR EXISTS (
           SELECT 1
             FROM vec_autorizacion_atestada_v3
                  .revocacion_clave_capacidad r
            WHERE r.clave_id = v_clave.clave_id
              AND r.version = v_clave.version
              AND r.revocada_en <= v_ahora
       )
       OR EXISTS (
           SELECT 1
             FROM vec_autorizacion_atestada_v3
                  .revocacion_configuracion r
            WHERE r.configuracion_revision = v_config.revision
              AND r.revocada_en <= v_ahora
       )
       OR EXISTS (
           SELECT 1
             FROM vec_autorizacion_atestada_v3.revocacion_raiz r
            WHERE r.raiz_clave_id = v_raiz.clave_id
              AND r.raiz_version = v_raiz.version
              AND r.revocada_en <= v_ahora
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'revalidación viva final RRHH rechazada';
    END IF;

    RETURN QUERY SELECT
        c ->> 'decision_ref',
        v_huella_consumo,
        pg_catalog.date_trunc('microseconds', v_ahora);
EXCEPTION
    WHEN invalid_text_representation OR datetime_field_overflow
      OR numeric_value_out_of_range OR no_data_found OR too_many_rows THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'entrada de revalidación final RRHH inválida';
END
$funcion$;

CREATE FUNCTION
vec_autorizacion_atestada_v3
    .revalidar_consumo_consulta_cuadro_rrhh_v3_atestada(
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
    consumo_huella_sha256 text,
    revalidada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '1s'
AS $funcion$
BEGIN
    RETURN QUERY
    SELECT *
      FROM vec_autorizacion_atestada_v3
           .revalidar_consumo_consulta_rrhh_v3_interna(
          'cuadro',
          p_capacidad_canonica,
          p_decision_canonica,
          p_motivo_canonico,
          p_contexto_actor_canonico,
          p_persona_version,
          p_perfil_version,
          p_payload_vec_ad_3,
          p_sobre_cose_sign1,
          p_evidencia_verificacion,
          p_raiz_publica_spki
      );
END
$funcion$;

CREATE FUNCTION
vec_autorizacion_atestada_v3
    .revalidar_consumo_consulta_detalle_rrhh_v3_atestada(
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
    consumo_huella_sha256 text,
    revalidada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '1s'
AS $funcion$
BEGIN
    RETURN QUERY
    SELECT *
      FROM vec_autorizacion_atestada_v3
           .revalidar_consumo_consulta_rrhh_v3_interna(
          'detalle',
          p_capacidad_canonica,
          p_decision_canonica,
          p_motivo_canonico,
          p_contexto_actor_canonico,
          p_persona_version,
          p_perfil_version,
          p_payload_vec_ad_3,
          p_sobre_cose_sign1,
          p_evidencia_verificacion,
          p_raiz_publica_spki
      );
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3
    .revalidar_consumo_consulta_rrhh_v3_interna(
        text, bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    )
FROM PUBLIC,
    vec_autorizacion_atestada_v3_migrador,
    vec_autorizacion_atestada_v3_emisor,
    vec_autorizacion_atestada_v3_consumidor,
    vec_contratacion_temporal_propietario;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3
    .revalidar_consumo_consulta_cuadro_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    ),
    vec_autorizacion_atestada_v3
    .revalidar_consumo_consulta_detalle_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    )
FROM PUBLIC,
    vec_autorizacion_atestada_v3_migrador,
    vec_autorizacion_atestada_v3_emisor,
    vec_autorizacion_atestada_v3_consumidor,
    vec_contratacion_temporal_propietario;

GRANT EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v3
    .revalidar_consumo_consulta_cuadro_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    ),
    vec_autorizacion_atestada_v3
    .revalidar_consumo_consulta_detalle_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    )
TO vec_contratacion_temporal_propietario;

COMMIT;
