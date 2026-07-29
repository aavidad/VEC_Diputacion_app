-- CT-000044B: coordinadores privados y atómicos de consulta RRHH.
--
-- La frontera recibe alcance, consulta y las diez piezas VEC-AD-3. No admite
-- actor, perfil ni sesión libres. Cada llamada debe vivir en una transacción
-- SERIALIZABLE de lectura-escritura iniciada por la fachada autorizada.

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
    v_decision jsonb;
    v_contexto_actor jsonb;
    v_actor_ref text;
    v_perfil_ref text;
    v_sesion_ref text;
    v_sesion_huella text;
    v_consulta_canonica bytea;
    v_contexto_recurso bytea;
    v_contexto_huella text;
    v_estado
        vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
    v_materializacion
        vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
    v_salida
        vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1;
    v_consumo
        vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3;
    v_contexto
        vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2;
    v_contenido
        vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2;
    v_cierre
        vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2;
    v_generada_en timestamptz(6);
    v_familia_ref text;
BEGIN
    PERFORM
        vec_contratacion_temporal.acreditar_contexto_motor_consultas_rrhh_v1(
            p_alcance, p_material
        );
    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
        p_consulta
    );

    -- El consumo sucede exactamente una vez y antes de cualquier lectura.
    v_consumo :=
        vec_contratacion_temporal
        .consumir_autorizacion_motor_consultas_rrhh_v1(
            'cuadro', p_material
        );
    v_decision := pg_catalog.convert_from(
        p_material.decision_canonica, 'UTF8'
    )::jsonb;
    v_contexto_actor := pg_catalog.convert_from(
        p_material.contexto_actor_canonico, 'UTF8'
    )::jsonb;
    IF v_decision ->> 'decision_ref' IS DISTINCT FROM
           v_consumo.decision_ref
       OR v_decision ->> 'principal_id' IS DISTINCT FROM
          v_contexto_actor ->> 'principal_ref'
       OR v_decision ->> 'perfil_activo_ref' IS DISTINCT FROM
          v_contexto_actor ->> 'perfil_activo_ref'
       OR v_contexto_actor ->> 'persona_version' IS DISTINCT FROM
          p_material.persona_version::text
       OR v_contexto_actor ->> 'perfil_version' IS DISTINCT FROM
          p_material.perfil_version::text THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de cuadro RRHH rechazada';
    END IF;
    -- No existe un validador puro reutilizable para esta ligadura. Se
    -- reconstruye el mismo contexto canónico de CT43 antes de leer datos.
    v_consulta_canonica :=
        vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
            p_consulta
        );
    v_contexto_recurso := pg_catalog.convert_to(
        '{"ambitos":{"ambito_ref":"' || p_alcance.ambito_ref
        || '","clase_ambito":"' || p_alcance.clase_ambito
        || '","organizacion_ref":"' || p_alcance.organizacion_ref
        || '"},"atributos":{"consulta_dominio":"'
        || 'vec.contratacion_temporal.consulta_rrhh.cuadro.v1'
        || '","consulta_huella_sha256":"'
        || pg_catalog.encode(
            pg_catalog.sha256(v_consulta_canonica), 'hex'
        ) || '"}}', 'UTF8'
    );
    v_contexto_huella := pg_catalog.encode(
        pg_catalog.sha256(v_contexto_recurso), 'hex'
    );
    IF v_decision ->> 'accion' IS DISTINCT FROM
           'contratacion_temporal.cuadro.consultar'
       OR v_decision ->> 'modulo_id' IS DISTINCT FROM
          'contratacion_temporal'
       OR v_decision ->> 'tipo_recurso' IS DISTINCT FROM
          'cuadro_rrhh_contratacion_temporal'
       OR v_decision ->> 'finalidad' IS DISTINCT FROM
          'gestion_operativa_contratacion_temporal'
       OR v_decision ->> 'recurso_ref' IS DISTINCT FROM
          p_alcance.ambito_ref
       OR v_decision ->> 'contexto_recurso_huella_sha256'
          IS DISTINCT FROM v_contexto_huella
       OR v_consumo.efecto_ref IS DISTINCT FROM p_alcance.ambito_ref
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM
          v_contexto_huella THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de cuadro RRHH rechazada';
    END IF;
    v_actor_ref := v_decision ->> 'principal_id';
    v_perfil_ref := v_decision ->> 'perfil_activo_ref';
    v_sesion_ref := v_decision #>>
        '{vinculo_autenticacion_actor,sesion_ref}';
    v_sesion_huella := v_decision #>>
        '{vinculo_autenticacion_actor,control_sesion_huella_sha256}';

    v_estado :=
        vec_contratacion_temporal
        .resolver_estado_cursor_cuadro_rrhh_v1(
            p_alcance, p_consulta, v_actor_ref, v_perfil_ref,
            p_material.perfil_version, v_sesion_ref, v_sesion_huella
        );
    v_materializacion :=
        vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
            p_alcance, p_consulta, v_estado
        );
    v_salida :=
        vec_contratacion_temporal
        .preparar_salida_cursor_cuadro_rrhh_v1(
            v_estado, v_materializacion
        );
    v_generada_en := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    v_familia_ref := CASE
        WHEN v_estado.es_continuacion THEN v_estado.familia_ref
        WHEN v_salida.hay_mas THEN v_salida.familia_ref
        ELSE NULL
    END;
    v_contexto := ROW(
        p_alcance.organizacion_ref, p_alcance.clase_ambito,
        p_alcance.ambito_ref, p_consulta,
        NULL::vec_contratacion_temporal.consulta_detalle_rrhh_v1,
        v_familia_ref
    );
    v_contenido := ROW(
        'cuadro', v_generada_en, v_materializacion.resumenes,
        v_materializacion.hay_mas, v_salida.cursor_huella,
        NULL::vec_contratacion_temporal
            .entrada_detalle_expediente_rrhh_v1
    );
    v_cierre :=
        vec_contratacion_temporal.cerrar_prueba_resultado_recibo_rrhh_v2(
            v_contexto, v_contenido, v_consumo,
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
    PERFORM
        vec_contratacion_temporal.aplicar_efectos_cursor_cuadro_rrhh_v1(
            p_alcance, p_consulta, v_estado, v_salida, v_consumo,
            p_material.decision_canonica, v_cierre
        );

    RETURN ROW(
        v_generada_en, v_materializacion.resumenes,
        v_materializacion.hay_mas, v_salida.cursor_siguiente, v_cierre
    )::vec_contratacion_temporal.resultado_motor_cuadro_rrhh_v1;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de cuadro RRHH rechazada';
END;
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
    v_decision jsonb;
    v_contexto_actor jsonb;
    v_consulta_canonica bytea;
    v_contexto_recurso bytea;
    v_contexto_huella text;
    v_corte_global numeric(20, 0);
    v_materializacion
        vec_contratacion_temporal.materializacion_detalle_rrhh_v1;
    v_consumo
        vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3;
    v_contexto
        vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2;
    v_contenido
        vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2;
    v_cierre
        vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2;
    v_generada_en timestamptz(6);
BEGIN
    PERFORM
        vec_contratacion_temporal.acreditar_contexto_motor_consultas_rrhh_v1(
            p_alcance, p_material
        );
    PERFORM vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
        p_consulta
    );

    v_consumo :=
        vec_contratacion_temporal
        .consumir_autorizacion_motor_consultas_rrhh_v1(
            'detalle', p_material
        );
    v_decision := pg_catalog.convert_from(
        p_material.decision_canonica, 'UTF8'
    )::jsonb;
    v_contexto_actor := pg_catalog.convert_from(
        p_material.contexto_actor_canonico, 'UTF8'
    )::jsonb;
    IF v_decision ->> 'decision_ref' IS DISTINCT FROM
           v_consumo.decision_ref
       OR v_decision ->> 'principal_id' IS DISTINCT FROM
          v_contexto_actor ->> 'principal_ref'
       OR v_decision ->> 'perfil_activo_ref' IS DISTINCT FROM
          v_contexto_actor ->> 'perfil_activo_ref'
       OR v_contexto_actor ->> 'persona_version' IS DISTINCT FROM
          p_material.persona_version::text
       OR v_contexto_actor ->> 'perfil_version' IS DISTINCT FROM
          p_material.perfil_version::text THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de detalle RRHH rechazada';
    END IF;
    v_consulta_canonica :=
        vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
            p_consulta
        );
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
    IF v_decision ->> 'accion' IS DISTINCT FROM
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
          IS DISTINCT FROM v_contexto_huella
       OR v_consumo.efecto_ref IS DISTINCT FROM
          p_consulta.expediente_ref
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM
          v_contexto_huella THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de detalle RRHH rechazada';
    END IF;
    SELECT control.ultimo_corte
      INTO STRICT v_corte_global
      FROM vec_contratacion_temporal.control_publicacion_rrhh control
     WHERE control.control;
    v_materializacion :=
        vec_contratacion_temporal.materializar_detalle_rrhh_v1(
            p_alcance, p_consulta, v_corte_global
        );
    v_generada_en := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    v_contexto := ROW(
        p_alcance.organizacion_ref, p_alcance.clase_ambito,
        p_alcance.ambito_ref,
        NULL::vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
        p_consulta, NULL
    );
    v_contenido := ROW(
        'detalle', v_generada_en,
        ARRAY[]::vec_contratacion_temporal.resumen_publicacion_rrhh_v1[],
        false, ''::bytea, v_materializacion.detalle
    );
    v_cierre :=
        vec_contratacion_temporal.cerrar_prueba_resultado_recibo_rrhh_v2(
            v_contexto, v_contenido, v_consumo,
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
    RETURN ROW(
        v_generada_en, v_materializacion.detalle, v_cierre
    )::vec_contratacion_temporal.resultado_motor_detalle_rrhh_v1;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de detalle RRHH rechazada';
END;
$funcion$;

ALTER FUNCTION
vec_contratacion_temporal.motor_consultar_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
) OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION
vec_contratacion_temporal.motor_consultar_detalle_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
) OWNER TO vec_contratacion_temporal_propietario;

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

COMMENT ON FUNCTION
vec_contratacion_temporal.motor_consultar_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
) IS
'Coordina consumo VEC, lectura única, prueba CT43 y cursores en una transacción.';
COMMENT ON FUNCTION
vec_contratacion_temporal.motor_consultar_detalle_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
) IS
'Coordina consumo VEC, detalle único y prueba CT43 sin crear cursores.';
