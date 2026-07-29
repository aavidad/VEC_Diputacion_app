-- CT-000044: guardas comunes y único punto privado de consumo VEC-AD-3.

CREATE FUNCTION
vec_contratacion_temporal.acreditar_contexto_motor_consultas_rrhh_v1(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_material
        vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
)
RETURNS void
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
    v_sesion oid;
    v_propietario oid;
BEGIN
    SELECT rol.oid
      INTO v_sesion
      FROM pg_catalog.pg_roles rol
     WHERE rol.rolname = SESSION_USER
       AND rol.rolcanlogin
       AND NOT rol.rolsuper
       AND NOT rol.rolcreatedb
       AND NOT rol.rolcreaterole
       AND NOT rol.rolreplication
       AND NOT rol.rolbypassrls;
    SELECT rol.oid
      INTO v_propietario
      FROM pg_catalog.pg_roles rol
     WHERE rol.rolname = 'vec_contratacion_temporal_propietario'
       AND NOT rol.rolcanlogin
       AND NOT rol.rolsuper
       AND NOT rol.rolcreatedb
       AND NOT rol.rolcreaterole
       AND NOT rol.rolinherit
       AND NOT rol.rolreplication
       AND NOT rol.rolbypassrls;

    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR v_sesion IS NULL
       OR v_propietario IS NULL
       OR v_sesion = v_propietario
       OR NOT pg_catalog.pg_has_role(
           SESSION_USER,
           'vec_contratacion_temporal_consultor_rrhh',
           'MEMBER'
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
       )::interval > interval '6 seconds'
       OR p_alcance IS NULL
       OR p_material IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'motor de consultas RRHH rechazado';
    END IF;

    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(p_alcance);

    IF p_material.capacidad_canonica IS NULL
       OR pg_catalog.octet_length(
           p_material.capacidad_canonica
       ) NOT BETWEEN 512 AND 32768
       OR p_material.decision_canonica IS NULL
       OR pg_catalog.octet_length(
           p_material.decision_canonica
       ) NOT BETWEEN 1 AND 524288
       OR p_material.motivo_canonico IS NULL
       OR pg_catalog.octet_length(
           p_material.motivo_canonico
       ) NOT BETWEEN 1 AND 65536
       OR p_material.contexto_actor_canonico IS NULL
       OR pg_catalog.octet_length(
           p_material.contexto_actor_canonico
       ) NOT BETWEEN 1 AND 262144
       OR p_material.persona_version IS NULL
       OR p_material.persona_version NOT BETWEEN
          1 AND 9007199254740991::numeric
       OR p_material.persona_version <>
          pg_catalog.trunc(p_material.persona_version)
       OR p_material.perfil_version IS NULL
       OR p_material.perfil_version NOT BETWEEN
          1 AND 9007199254740991::numeric
       OR p_material.perfil_version <>
          pg_catalog.trunc(p_material.perfil_version)
       OR p_material.payload_vec_ad_3 IS NULL
       OR pg_catalog.octet_length(
           p_material.payload_vec_ad_3
       ) NOT BETWEEN 1 AND 1048576
       OR p_material.sobre_cose_sign_1 IS NULL
       OR pg_catalog.octet_length(
           p_material.sobre_cose_sign_1
       ) NOT BETWEEN 1 AND 1048576
       OR p_material.evidencia_verificacion IS NULL
       OR pg_catalog.octet_length(
           p_material.evidencia_verificacion
       ) NOT BETWEEN 1 AND 262144
       OR p_material.raiz_publica_spki IS NULL
       OR pg_catalog.octet_length(p_material.raiz_publica_spki) <> 44 THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'motor de consultas RRHH rechazado';
    END IF;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'motor de consultas RRHH rechazado';
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.consumir_autorizacion_motor_consultas_rrhh_v1(
    p_tipo_consulta text,
    p_material
        vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
)
RETURNS
    vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3
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
    v_consumo record;
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR p_material IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'consumo de consulta RRHH rechazado';
    END IF;

    IF p_tipo_consulta = 'cuadro' THEN
        SELECT *
          INTO STRICT v_consumo
          FROM vec_autorizacion_atestada_v3
               .registrar_y_consumir_consulta_cuadro_rrhh_v3_atestada(
              p_material.capacidad_canonica,
              p_material.decision_canonica,
              p_material.motivo_canonico,
              p_material.contexto_actor_canonico,
              p_material.persona_version,
              p_material.perfil_version,
              p_material.payload_vec_ad_3,
              p_material.sobre_cose_sign_1,
              p_material.evidencia_verificacion,
              p_material.raiz_publica_spki
          );
    ELSIF p_tipo_consulta = 'detalle' THEN
        SELECT *
          INTO STRICT v_consumo
          FROM vec_autorizacion_atestada_v3
               .registrar_y_consumir_consulta_detalle_rrhh_v3_atestada(
              p_material.capacidad_canonica,
              p_material.decision_canonica,
              p_material.motivo_canonico,
              p_material.contexto_actor_canonico,
              p_material.persona_version,
              p_material.perfil_version,
              p_material.payload_vec_ad_3,
              p_material.sobre_cose_sign_1,
              p_material.evidencia_verificacion,
              p_material.raiz_publica_spki
          );
    ELSE
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'tipo de consulta RRHH inválido';
    END IF;

    IF v_consumo.consumo_nuevo IS DISTINCT FROM true THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'consumo de consulta RRHH rechazado';
    END IF;

    RETURN ROW(
        v_consumo.decision_ref,
        v_consumo.efecto_ref,
        v_consumo.huella_efecto_sha256,
        v_consumo.consumo_huella_sha256,
        v_consumo.auditoria_ref,
        v_consumo.auditoria_huella_sha256,
        pg_catalog.date_trunc(
            'microseconds', v_consumo.consumida_en
        ),
        true
    )::vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'consumo de consulta RRHH rechazado';
END
$funcion$;

ALTER FUNCTION
vec_contratacion_temporal.acreditar_contexto_motor_consultas_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
) OWNER TO vec_contratacion_temporal_propietario;

ALTER FUNCTION
vec_contratacion_temporal.consumir_autorizacion_motor_consultas_rrhh_v1(
    text,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
) OWNER TO vec_contratacion_temporal_propietario;

-- Estas dos primitivas son piezas internas. En especial, los SET locales de
-- la primera limitan su propia ejecución: cada coordinador que la invoque
-- debe repetir y sellar los mismos límites en su catálogo.
REVOKE ALL ON FUNCTION
vec_contratacion_temporal.acreditar_contexto_motor_consultas_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
),
vec_contratacion_temporal.consumir_autorizacion_motor_consultas_rrhh_v1(
    text,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
)
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_consultor_rrhh,
    vec_contratacion_temporal_lector_resultado_cobertura;
