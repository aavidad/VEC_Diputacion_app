-- CT-000045: única frontera exterior para consultar un detalle RRHH.
CREATE FUNCTION
vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    p_capacidad_canonica bytea,
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_contexto_actor_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric,
    p_payload_vec_ad_3 bytea,
    p_sobre_cose_sign_1 bytea,
    p_evidencia_verificacion bytea,
    p_raiz_publica_spki bytea
)
RETURNS TABLE(
    contenido_canonico bytea,
    esquema text,
    acceso_ref text,
    secuencia numeric,
    anterior_sha256 text,
    huella_sha256 text,
    vinculo_identidad_huella_sha256 text,
    alcance_huella_sha256 text,
    registrada_en timestamptz,
    auditoria_vec_ref text,
    auditoria_vec_huella_sha256 text,
    consumo_vec_huella_sha256 text,
    contenido_huella_sha256 text,
    resultado_huella_sha256 text,
    cursor_huella_sha256 text,
    generada_en timestamptz,
    expediente_ref text,
    version_expediente numeric,
    total smallint,
    recibo_sello_sha256 text
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
SET idle_in_transaction_session_timeout = '6s'
AS $funcion$
DECLARE
    v_login pg_catalog.pg_roles%ROWTYPE;
    v_capacidad jsonb;
    v_decision jsonb;
    v_consulta_canonica bytea;
    v_contexto_recurso bytea;
    v_contexto_huella text;
    v_decision_huella text;
    v_material
        vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3;
    v_resultado
        vec_contratacion_temporal.resultado_motor_detalle_rrhh_v1;
    v_cierre
        vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2;
    v_detalle
        vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
    v_resumen
        vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
    v_contenido bytea;
BEGIN
    SELECT *
      INTO v_login
      FROM pg_catalog.pg_roles rol
     WHERE rol.rolname = SESSION_USER;
    IF CURRENT_USER <>
           'vec_contratacion_temporal_propietario'
       OR SESSION_USER = CURRENT_USER
       OR v_login.oid IS NULL
       OR NOT v_login.rolcanlogin
       OR NOT v_login.rolinherit
       OR v_login.rolsuper
       OR v_login.rolcreatedb
       OR v_login.rolcreaterole
       OR v_login.rolreplication
       OR v_login.rolbypassrls
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_auth_members membresia
            WHERE membresia.member = v_login.oid
       ) <> 1
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members membresia
             JOIN pg_catalog.pg_roles grupo
               ON grupo.oid = membresia.roleid
            WHERE membresia.member = v_login.oid
              AND grupo.rolname =
                  'vec_contratacion_temporal_consultor_rrhh'
              AND NOT membresia.admin_option
              AND membresia.inherit_option
              AND NOT membresia.set_option
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members membresia
            WHERE membresia.roleid = v_login.oid
       )
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_roles grupo
            WHERE grupo.rolname =
                  'vec_contratacion_temporal_consultor_rrhh'
              AND NOT grupo.rolcanlogin
              AND grupo.rolinherit
              AND NOT grupo.rolsuper
              AND NOT grupo.rolcreatedb
              AND NOT grupo.rolcreaterole
              AND NOT grupo.rolreplication
              AND NOT grupo.rolbypassrls
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members membresia
            WHERE membresia.member =
                  'vec_contratacion_temporal_consultor_rrhh'
                      ::pg_catalog.regrole
       )
       OR pg_catalog.pg_is_in_recovery()
       OR pg_catalog.current_setting('transaction_isolation')
          <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR pg_catalog.current_setting('lock_timeout') = '0'
       OR pg_catalog.current_setting('lock_timeout')::interval >
          interval '1 second'
       OR pg_catalog.current_setting('statement_timeout') = '0'
       OR pg_catalog.current_setting('statement_timeout')::interval >
          interval '4 seconds'
       OR pg_catalog.current_setting(
           'idle_in_transaction_session_timeout'
       ) = '0'
       OR pg_catalog.current_setting(
           'idle_in_transaction_session_timeout'
       )::interval > interval '6 seconds' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    -- Los límites O(1) preceden a los cánones CT40 y, por tanto, a cualquier
    -- expresión regular sobre varlena exterior.
    IF p_alcance IS NOT DISTINCT FROM
           NULL::vec_contratacion_temporal.alcance_consulta_rrhh_v1
       OR pg_catalog.octet_length(
           COALESCE(p_alcance.organizacion_ref, '')
       ) > 160
       OR pg_catalog.octet_length(
           COALESCE(p_alcance.clase_ambito, '')
       ) > 16
       OR pg_catalog.octet_length(
           COALESCE(p_alcance.ambito_ref, '')
       ) > 160
       OR p_consulta IS NOT DISTINCT FROM
          NULL::vec_contratacion_temporal.consulta_detalle_rrhh_v1
       OR pg_catalog.octet_length(
           COALESCE(p_consulta.expediente_ref, '')
       ) > 160 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    -- Limita antes de decodificar o analizar cualquier pieza controlada por
    -- el llamador. Repite exactamente la frontera privada CT44.
    IF p_capacidad_canonica IS NULL
       OR pg_catalog.octet_length(
           p_capacidad_canonica
       ) NOT BETWEEN 512 AND 32768
       OR p_decision_canonica IS NULL
       OR pg_catalog.octet_length(
           p_decision_canonica
       ) NOT BETWEEN 1 AND 524288
       OR p_motivo_canonico IS NULL
       OR pg_catalog.octet_length(
           p_motivo_canonico
       ) NOT BETWEEN 1 AND 65536
       OR p_contexto_actor_canonico IS NULL
       OR pg_catalog.octet_length(
           p_contexto_actor_canonico
       ) NOT BETWEEN 1 AND 262144
       OR p_persona_version IS NULL
       OR p_persona_version NOT BETWEEN
          1 AND 9007199254740991::numeric
       OR p_persona_version <> pg_catalog.trunc(p_persona_version)
       OR p_perfil_version IS NULL
       OR p_perfil_version NOT BETWEEN
          1 AND 9007199254740991::numeric
       OR p_perfil_version <> pg_catalog.trunc(p_perfil_version)
       OR p_payload_vec_ad_3 IS NULL
       OR pg_catalog.octet_length(
           p_payload_vec_ad_3
       ) NOT BETWEEN 1 AND 1048576
       OR p_sobre_cose_sign_1 IS NULL
       OR pg_catalog.octet_length(
           p_sobre_cose_sign_1
       ) NOT BETWEEN 1 AND 1048576
       OR p_evidencia_verificacion IS NULL
       OR pg_catalog.octet_length(
           p_evidencia_verificacion
       ) NOT BETWEEN 1 AND 262144
       OR p_raiz_publica_spki IS NULL
       OR pg_catalog.octet_length(p_raiz_publica_spki) <> 44 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(
        p_alcance
    );
    v_consulta_canonica :=
        vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
            p_consulta
        );
    v_capacidad := pg_catalog.convert_from(
        p_capacidad_canonica, 'UTF8'
    )::jsonb;
    v_decision := pg_catalog.convert_from(
        p_decision_canonica, 'UTF8'
    )::jsonb;
    v_contexto_recurso := pg_catalog.convert_to(
        '{"ambitos":{"ambito_ref":"' || p_alcance.ambito_ref
        || '","clase_ambito":"' || p_alcance.clase_ambito
        || '","organizacion_ref":"' || p_alcance.organizacion_ref
        || '"},"atributos":{"consulta_dominio":"'
        || 'vec.contratacion_temporal.consulta_rrhh.detalle.v1'
        || '","consulta_huella_sha256":"'
        || pg_catalog.encode(
            pg_catalog.sha256(v_consulta_canonica), 'hex'
        ) || '"}}', 'UTF8'
    );
    v_contexto_huella := pg_catalog.encode(
        pg_catalog.sha256(v_contexto_recurso), 'hex'
    );
    v_decision_huella := pg_catalog.encode(
        pg_catalog.sha256(p_decision_canonica), 'hex'
    );
    IF v_capacidad ->> 'operacion' IS DISTINCT FROM
           'contratacion_temporal.expediente.consultar'
       OR v_capacidad ->> 'audiencia_consumo' IS DISTINCT FROM
          'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
       OR v_capacidad ->> 'efecto_ref' IS DISTINCT FROM
          p_consulta.expediente_ref
       OR v_capacidad ->> 'huella_efecto_sha256' IS DISTINCT FROM
          v_contexto_huella
       OR v_capacidad ->> 'huella_decision_sha256' IS DISTINCT FROM
          v_decision_huella
       OR v_decision ->> 'accion' IS DISTINCT FROM
          'contratacion_temporal.expediente.consultar'
       OR v_decision ->> 'modulo_id' IS DISTINCT FROM
          'contratacion_temporal'
       OR v_decision ->> 'tipo_recurso' IS DISTINCT FROM
          'expediente_contratacion_temporal'
       OR v_decision ->> 'finalidad' IS DISTINCT FROM
          'tramitacion_expediente_contratacion_temporal'
       OR v_decision ->> 'recurso_ref' IS DISTINCT FROM
          p_consulta.expediente_ref
       OR v_decision ->> 'contexto_recurso_huella_sha256'
          IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    v_material := ROW(
        p_capacidad_canonica, p_decision_canonica,
        p_motivo_canonico, p_contexto_actor_canonico,
        p_persona_version, p_perfil_version,
        p_payload_vec_ad_3, p_sobre_cose_sign_1,
        p_evidencia_verificacion, p_raiz_publica_spki
    )::vec_contratacion_temporal
       .material_autorizacion_consulta_rrhh_v3;
    v_resultado :=
        vec_contratacion_temporal.motor_consultar_detalle_rrhh_v1(
            p_alcance, p_consulta, v_material
        );
    v_cierre := v_resultado.cierre;
    v_detalle := v_resultado.detalle;
    v_resumen := v_detalle.resumen;
    IF v_resultado IS NOT DISTINCT FROM
           NULL::vec_contratacion_temporal.resultado_motor_detalle_rrhh_v1
       OR v_cierre IS NOT DISTINCT FROM
          NULL::vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2
       OR v_detalle IS NOT DISTINCT FROM
          NULL::vec_contratacion_temporal
              .entrada_detalle_expediente_rrhh_v1
       OR v_resumen IS NOT DISTINCT FROM
          NULL::vec_contratacion_temporal.resumen_publicacion_rrhh_v1
       OR v_resultado.generada_en IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;
    v_contenido :=
        vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
            v_resultado.generada_en, v_detalle
        );
    IF v_cierre.generada_en IS DISTINCT FROM
           v_resultado.generada_en
       OR v_cierre.total IS DISTINCT FROM 1::smallint
       OR v_cierre.cursor_huella_sha256 IS DISTINCT FROM ''
       OR v_cierre.alcance_huella_sha256 IS DISTINCT FROM ''
       OR v_cierre.expediente_ref IS DISTINCT FROM
          v_resumen.expediente_ref
       OR v_cierre.version_expediente IS DISTINCT FROM
          v_resumen.version
       OR v_cierre.contenido_huella_sha256 IS DISTINCT FROM
          pg_catalog.encode(pg_catalog.sha256(v_contenido), 'hex') THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    RETURN QUERY SELECT
        v_contenido, v_cierre.esquema, v_cierre.acceso_ref,
        v_cierre.secuencia, v_cierre.anterior_sha256,
        v_cierre.huella_sha256,
        v_cierre.vinculo_identidad_huella_sha256,
        v_cierre.alcance_huella_sha256, v_cierre.registrada_en,
        v_cierre.auditoria_vec_ref,
        v_cierre.auditoria_vec_huella_sha256,
        v_cierre.consumo_vec_huella_sha256,
        v_cierre.contenido_huella_sha256,
        v_cierre.resultado_huella_sha256,
        v_cierre.cursor_huella_sha256, v_cierre.generada_en,
        v_cierre.expediente_ref, v_cierre.version_expediente,
        v_cierre.total, v_cierre.recibo_sello_sha256;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
END
$funcion$;

ALTER FUNCTION
vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) OWNER TO vec_contratacion_temporal_propietario;

COMMENT ON FUNCTION
vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) IS
'Fachada nominal de detalle RRHH: consume una capacidad y devuelve contenido canónico con Recibo V2.';
