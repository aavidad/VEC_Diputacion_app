\set ON_ERROR_STOP on

-- Fachadas exclusivas del fixture adversarial CT-000044B. Ninguna devuelve
-- el token cuando la prueba solo necesita acreditar el cierre o una carrera.

CREATE FUNCTION
vec_contratacion_temporal.prueba_invocar_motor_cuadro_controlado_ct44b(
    p_caso text,
    p_cursor text,
    p_organizacion_ref text,
    p_devolver_cursor boolean
)
RETURNS text
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
    v_vector public.vectores_consulta_rrhh_v3%ROWTYPE;
    v_material
        vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3;
    v_resultado
        vec_contratacion_temporal.resultado_motor_cuadro_rrhh_v1;
BEGIN
    SELECT * INTO STRICT v_vector
      FROM public.vectores_consulta_rrhh_v3
     WHERE caso = p_caso AND perfil = 'cuadro';
    v_material := ROW(
        v_vector.capacidad, v_vector.decision, v_vector.motivo,
        v_vector.contexto, v_vector.persona_version,
        v_vector.perfil_version, v_vector.payload, v_vector.cose,
        v_vector.evidencia, v_vector.spki
    );
    v_resultado :=
        vec_contratacion_temporal.motor_consultar_cuadro_rrhh_v1(
            ROW(
                p_organizacion_ref, 'organizacion',
                p_organizacion_ref
            ),
            ROW('', '', '', 10::smallint, p_cursor),
            v_material
        );
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .prueba_resultado_recibo_rrhh_v2 prueba
         WHERE prueba.acceso_ref = (v_resultado.cierre).acceso_ref
           AND prueba.generada_en = v_resultado.generada_en
           AND prueba.resumenes IS NOT DISTINCT FROM
               v_resultado.resumenes
           AND prueba.hay_mas IS NOT DISTINCT FROM
               v_resultado.hay_mas
    ) THEN
        RAISE EXCEPTION 'resultado controlado CT44B divergente';
    END IF;
    IF p_devolver_cursor IS TRUE THEN
        RETURN v_resultado.cursor_siguiente;
    END IF;
    RETURN (v_resultado.cierre).acceso_ref;
END
$funcion$;

-- Reutiliza un cierre terminal durable para atacar exclusivamente la forma
-- transitoria entregada a 055. La función no crea una segunda autoridad.
CREATE FUNCTION
vec_contratacion_temporal.prueba_forma_terminal_efectos_ct44b(
    p_variante text,
    p_cursor text
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
    v_prueba
        vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2%ROWTYPE;
    v_vector public.vectores_consulta_rrhh_v3%ROWTYPE;
    v_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
    v_estado
        vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
    v_salida
        vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1;
    v_consumo
        vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3;
    v_cierre
        vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2;
    v_corte numeric(20, 0);
    v_token text := pg_catalog.repeat('A', 43);
    v_token_huella text;
    v_familia_ref text;
    v_caso text;
BEGIN
    v_caso := CASE
        WHEN p_variante IN (
            'familia_ajena', 'keyset_ajeno', 'estado_ajeno'
        ) THEN 'ct44b_pagina_1'
        ELSE 'ct44b_inicial_final'
    END;
    SELECT prueba.* INTO STRICT v_prueba
      FROM vec_contratacion_temporal
           .prueba_resultado_recibo_rrhh_v2 prueba
     WHERE prueba.decision_ref =
           'decision:consulta-rrhh:' || v_caso;
    SELECT * INTO STRICT v_vector
      FROM public.vectores_consulta_rrhh_v3
     WHERE caso = v_caso AND perfil = 'cuadro';
    SELECT ultimo_corte INTO STRICT v_corte
      FROM vec_contratacion_temporal.control_publicacion_rrhh
     WHERE control;

    v_consulta := ROW('', '', '', 10::smallint, '');
    v_estado := ROW(
        false, NULL, v_corte, 0, NULL, NULL, NULL, NULL, NULL,
        NULL, NULL
    );
    IF p_variante = 'cursor_nulo' THEN
        v_salida := ROW(
            false, '', NULL, NULL, 0, NULL, NULL, NULL, NULL
        );
    ELSIF p_variante = 'cursor_siguiente_no_vacio' THEN
        v_salida := ROW(
            false, v_token, ''::bytea, NULL, 0, NULL, NULL, NULL, NULL
        );
    ELSIF p_variante = 'cursor_no_vacio' THEN
        v_salida := ROW(
            false, '', pg_catalog.decode(pg_catalog.repeat('f', 64), 'hex'),
            NULL, 0, NULL, NULL, NULL, NULL
        );
    ELSIF p_variante = 'cierre_terminal_intercambiable' THEN
        v_token_huella := pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to(v_token, 'UTF8')
        ), 'hex');
        v_salida := ROW(
            true, v_token, pg_catalog.decode(v_token_huella, 'hex'),
            'familia:cursor:rrhh:'
                || pg_catalog.repeat('e', 32),
            2, v_token_huella, NULL,
            (v_prueba.resumenes[1]).actualizado_en,
            (v_prueba.resumenes[1]).expediente_ref
        );
    ELSIF p_variante IN (
        'familia_ajena', 'keyset_ajeno', 'estado_ajeno'
    ) THEN
        IF p_cursor !~ '^[A-Za-z0-9_-]{43}$' THEN
            RAISE EXCEPTION 'cursor hostil CT44B inválido';
        END IF;
        SELECT alcance.familia_ref INTO STRICT v_familia_ref
          FROM vec_contratacion_temporal.alcance_acceso_rrhh alcance
         WHERE alcance.acceso_ref = v_prueba.acceso_ref;
        v_token_huella := pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to(p_cursor, 'UTF8')
        ), 'hex');
        v_salida := ROW(
            true, p_cursor, pg_catalog.decode(v_token_huella, 'hex'),
            v_familia_ref, 2, v_token_huella, NULL,
            (v_prueba.resumenes[v_prueba.total]).actualizado_en,
            (v_prueba.resumenes[v_prueba.total]).expediente_ref
        );
        IF p_variante = 'familia_ajena' THEN
            v_salida.familia_ref :=
                'familia:cursor:rrhh:' || pg_catalog.repeat('e', 32);
        ELSIF p_variante = 'keyset_ajeno' THEN
            v_salida.ultimo_expediente_ref :=
                'expediente:rrhh:keyset-ajeno';
        ELSE
            SELECT true, cursor.familia_ref, familia.corte_global,
                   cursor.pagina, cursor.token_huella_sha256,
                   cursor.acceso_emision_ref, cursor.emitida_en,
                   cursor.familia_creada_en, cursor.familia_valida_hasta,
                   cursor.ultimo_actualizado_en,
                   cursor.ultimo_expediente_ref
              INTO STRICT v_estado
              FROM vec_contratacion_temporal.cursor_cuadro_rrhh cursor
              JOIN vec_contratacion_temporal
                   .familia_cursor_cuadro_rrhh familia
                USING (familia_ref)
             WHERE cursor.token_huella_sha256 = v_token_huella;
            v_estado.ultimo_expediente_ref :=
                'expediente:rrhh:estado-ajeno';
            v_salida.pagina_nueva := v_estado.pagina_presentada + 1;
            v_salida.padre_token_huella_sha256 := v_token_huella;
        END IF;
    ELSIF p_variante = 'consulta_ajena' THEN
        v_consulta.texto := 'ajena';
        v_salida := ROW(
            false, '', ''::bytea, NULL, 0, NULL, NULL, NULL, NULL
        );
    ELSIF p_variante = 'limite_ajeno' THEN
        v_consulta.limite := 9;
        v_salida := ROW(
            false, '', ''::bytea, NULL, 0, NULL, NULL, NULL, NULL
        );
    ELSIF p_variante = 'corte_ajeno' THEN
        v_estado.corte_global := v_corte - 1;
        v_salida := ROW(
            false, '', ''::bytea, NULL, 0, NULL, NULL, NULL, NULL
        );
    ELSE
        RAISE EXCEPTION 'variante terminal CT44B desconocida';
    END IF;

    v_consumo := ROW(
        v_prueba.decision_ref,
        'organizacion:diputacion-granada',
        v_prueba.alcance_huella_sha256,
        v_prueba.consumo_vec_huella_sha256,
        v_prueba.auditoria_vec_ref,
        v_prueba.auditoria_vec_huella_sha256,
        v_prueba.registrada_en, true
    );
    v_cierre := ROW(
        'vec.contratacion-temporal.recibo-acceso-rrhh.o4-05.v2',
        v_prueba.acceso_ref, v_prueba.secuencia,
        v_prueba.anterior_sha256, v_prueba.huella_sha256,
        v_prueba.vinculo_identidad_huella_sha256,
        v_prueba.alcance_huella_sha256, v_prueba.registrada_en,
        v_prueba.auditoria_vec_ref,
        v_prueba.auditoria_vec_huella_sha256,
        v_prueba.consumo_vec_huella_sha256,
        v_prueba.contenido_huella_sha256,
        v_prueba.resultado_huella_sha256,
        COALESCE(v_prueba.cursor_huella_sha256, ''),
        v_prueba.generada_en, '', 0, v_prueba.total,
        v_prueba.recibo_sello_sha256
    );
    PERFORM
        vec_contratacion_temporal.aplicar_efectos_cursor_cuadro_rrhh_v1(
            ROW(
                'organizacion:diputacion-granada', 'organizacion',
                'organizacion:diputacion-granada'
            ),
            v_consulta,
            v_estado, v_salida, v_consumo, v_vector.decision, v_cierre
        );
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.prueba_invocar_motor_detalle_controlado_ct44b(
    p_caso text,
    p_expediente_ref text,
    p_version numeric
)
RETURNS text
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
    v_vector public.vectores_consulta_rrhh_v3%ROWTYPE;
    v_material
        vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3;
    v_resultado
        vec_contratacion_temporal.resultado_motor_detalle_rrhh_v1;
BEGIN
    SELECT * INTO STRICT v_vector
      FROM public.vectores_consulta_rrhh_v3
     WHERE caso = p_caso AND perfil = 'detalle';
    v_material := ROW(
        v_vector.capacidad, v_vector.decision, v_vector.motivo,
        v_vector.contexto, v_vector.persona_version,
        v_vector.perfil_version, v_vector.payload, v_vector.cose,
        v_vector.evidencia, v_vector.spki
    );
    v_resultado :=
        vec_contratacion_temporal.motor_consultar_detalle_rrhh_v1(
            ROW(
                'organizacion:diputacion-granada', 'organizacion',
                'organizacion:diputacion-granada'
            ),
            ROW(p_expediente_ref, p_version),
            v_material
        );
    RETURN (v_resultado.cierre).acceso_ref;
END
$funcion$;

ALTER FUNCTION
vec_contratacion_temporal.prueba_invocar_motor_cuadro_controlado_ct44b(
    text, text, text, boolean
) OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION
vec_contratacion_temporal.prueba_forma_terminal_efectos_ct44b(text, text)
OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION
vec_contratacion_temporal.prueba_invocar_motor_detalle_controlado_ct44b(
    text, text, numeric
)
OWNER TO vec_contratacion_temporal_propietario;

REVOKE ALL ON FUNCTION
vec_contratacion_temporal.prueba_invocar_motor_cuadro_controlado_ct44b(
    text, text, text, boolean
),
vec_contratacion_temporal.prueba_forma_terminal_efectos_ct44b(text, text),
vec_contratacion_temporal.prueba_invocar_motor_detalle_controlado_ct44b(
    text, text, numeric
)
FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
vec_contratacion_temporal.prueba_invocar_motor_cuadro_controlado_ct44b(
    text, text, text, boolean
),
vec_contratacion_temporal.prueba_forma_terminal_efectos_ct44b(text, text),
vec_contratacion_temporal.prueba_invocar_motor_detalle_controlado_ct44b(
    text, text, numeric
)
TO vec_c2d2_registro_runtime;
