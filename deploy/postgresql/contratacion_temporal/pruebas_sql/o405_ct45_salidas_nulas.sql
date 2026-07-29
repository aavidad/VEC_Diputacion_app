\set ON_ERROR_STOP on

-- Instrumentación exclusiva del PostgreSQL efímero. Conserva los motores
-- reales bajo nombres temporales y coloca delante un wrapper con idéntica
-- frontera, propietario, seguridad y configuración. El motor real ejecuta
-- primero; el rechazo posterior de la fachada debe revertir todos sus efectos.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;

ALTER FUNCTION
vec_contratacion_temporal.motor_consultar_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
) RENAME TO motor_consultar_cuadro_rrhh_real_ct45;
ALTER FUNCTION
vec_contratacion_temporal.motor_consultar_detalle_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
) RENAME TO motor_consultar_detalle_rrhh_real_ct45;

CREATE FUNCTION
vec_contratacion_temporal.motor_consultar_cuadro_rrhh_v1(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    p_material
        vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
)
RETURNS vec_contratacion_temporal.resultado_motor_cuadro_rrhh_v1
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
    v_variante text := pg_catalog.current_setting(
        'vec.prueba_ct45_salida_nula', true
    );
    v_resultado
        vec_contratacion_temporal.resultado_motor_cuadro_rrhh_v1;
    v_cierre
        vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2;
BEGIN
    v_resultado :=
        vec_contratacion_temporal
        .motor_consultar_cuadro_rrhh_real_ct45(
            p_alcance, p_consulta, p_material
        );
    v_cierre := v_resultado.cierre;
    CASE v_variante
        WHEN 'cuadro_cursor_huella' THEN
            v_cierre.cursor_huella_sha256 := NULL;
        WHEN 'cuadro_expediente' THEN
            v_cierre.expediente_ref := NULL;
        WHEN 'cuadro_version' THEN
            v_cierre.version_expediente := NULL;
        ELSE
            RAISE EXCEPTION 'variante NULL de cuadro CT45 desconocida';
    END CASE;
    v_resultado.cierre := v_cierre;
    RETURN v_resultado;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.motor_consultar_detalle_rrhh_v1(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    p_material
        vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
)
RETURNS vec_contratacion_temporal.resultado_motor_detalle_rrhh_v1
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
    v_variante text := pg_catalog.current_setting(
        'vec.prueba_ct45_salida_nula', true
    );
    v_resultado
        vec_contratacion_temporal.resultado_motor_detalle_rrhh_v1;
    v_cierre
        vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2;
BEGIN
    v_resultado :=
        vec_contratacion_temporal
        .motor_consultar_detalle_rrhh_real_ct45(
            p_alcance, p_consulta, p_material
        );
    v_cierre := v_resultado.cierre;
    CASE v_variante
        WHEN 'detalle_cursor_huella' THEN
            v_cierre.cursor_huella_sha256 := NULL;
        WHEN 'detalle_alcance_huella' THEN
            v_cierre.alcance_huella_sha256 := NULL;
        ELSE
            RAISE EXCEPTION 'variante NULL de detalle CT45 desconocida';
    END CASE;
    v_resultado.cierre := v_cierre;
    RETURN v_resultado;
END
$funcion$;

REVOKE ALL ON FUNCTION
vec_contratacion_temporal.motor_consultar_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
),
vec_contratacion_temporal.motor_consultar_detalle_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
) FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_consultor_rrhh,
    vec_contratacion_temporal_lector_resultado_cobertura;
COMMIT;

-- La ayuda usa SESSION_USER runtime real; SECURITY DEFINER solo le permite
-- leer el vector sintético sin conceder al login acceso a tablas o tipos.
CREATE FUNCTION vec_contratacion_temporal.invocar_salida_nula_ct45(
    p_caso text,
    p_perfil text,
    p_variante text
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_vector public.vectores_consulta_rrhh_v3%ROWTYPE;
BEGIN
    PERFORM pg_catalog.set_config(
        'vec.prueba_ct45_salida_nula', p_variante, true
    );
    SELECT * INTO STRICT v_vector
      FROM public.vectores_consulta_rrhh_v3
     WHERE caso = p_caso AND perfil = p_perfil;
    IF p_perfil = 'cuadro' THEN
        PERFORM *
          FROM vec_contratacion_temporal
               .consultar_cuadro_rrhh_atestado_v1(
                   ROW(
                       'organizacion:diputacion-granada',
                       'organizacion',
                       'organizacion:diputacion-granada'
                   ),
                   ROW('', '', '', 10::smallint, ''),
                   v_vector.capacidad, v_vector.decision,
                   v_vector.motivo, v_vector.contexto,
                   v_vector.persona_version, v_vector.perfil_version,
                   v_vector.payload, v_vector.cose,
                   v_vector.evidencia, v_vector.spki
               );
    ELSIF p_perfil = 'detalle' THEN
        PERFORM *
          FROM vec_contratacion_temporal
               .consultar_detalle_rrhh_atestado_v1(
                   ROW(
                       'organizacion:diputacion-granada',
                       'organizacion',
                       'organizacion:diputacion-granada'
                   ),
                   ROW('expediente:rrhh:minimizado', 1::numeric),
                   v_vector.capacidad, v_vector.decision,
                   v_vector.motivo, v_vector.contexto,
                   v_vector.persona_version, v_vector.perfil_version,
                   v_vector.payload, v_vector.cose,
                   v_vector.evidencia, v_vector.spki
               );
    ELSE
        RAISE EXCEPTION 'perfil NULL CT45 desconocido';
    END IF;
END
$funcion$;

REVOKE ALL ON FUNCTION
vec_contratacion_temporal.invocar_salida_nula_ct45(text, text, text)
FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.invocar_salida_nula_ct45(text, text, text)
TO vec_c2d2_registro_runtime;
