\set ON_ERROR_STOP on

-- Fixture efímero: construye únicamente escalares integrados y ataca la
-- prevalidación exterior. Los octetos 0xff no UTF-8 prueban la normalización
-- uniforme; el orden pre-canon se acredita aparte contra el cuerpo catalogado.
CREATE FUNCTION vec_contratacion_temporal.invocar_limite_fachada_ct45(
    p_perfil text,
    p_variante text
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY INVOKER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1 :=
        ROW(
            'organizacion:diputacion-granada',
            'organizacion',
            'organizacion:diputacion-granada'
        );
    v_cuadro vec_contratacion_temporal.consulta_cuadro_rrhh_v1 :=
        ROW('', '', '', 10::smallint, '');
    v_detalle vec_contratacion_temporal.consulta_detalle_rrhh_v1 :=
        ROW('expediente:rrhh:minimizado', 1::numeric);
    v_capacidad bytea := pg_catalog.decode(
        pg_catalog.repeat('ff', 512), 'hex'
    );
    v_decision bytea := pg_catalog.decode('ff', 'hex');
    v_motivo bytea := pg_catalog.decode('ff', 'hex');
    v_contexto bytea := pg_catalog.decode('ff', 'hex');
    v_persona numeric := 1;
    v_perfil numeric := 1;
    v_payload bytea := pg_catalog.decode('ff', 'hex');
    v_sobre bytea := pg_catalog.decode('ff', 'hex');
    v_evidencia bytea := pg_catalog.decode('ff', 'hex');
    v_raiz bytea := pg_catalog.decode(
        pg_catalog.repeat('ff', 44), 'hex'
    );
BEGIN
    CASE p_variante
        WHEN 'forma_valida_no_utf8' THEN NULL;
        WHEN 'alcance_nulo' THEN v_alcance := NULL;
        WHEN 'organizacion_superior' THEN
            v_alcance.organizacion_ref := pg_catalog.repeat('x', 161);
        WHEN 'clase_ambito_superior' THEN
            v_alcance.clase_ambito := pg_catalog.repeat('x', 17);
        WHEN 'ambito_superior' THEN
            v_alcance.ambito_ref := pg_catalog.repeat('x', 161);
        WHEN 'consulta_nula' THEN
            v_cuadro := NULL;
            v_detalle := NULL;
        WHEN 'cuadro_texto_superior' THEN
            IF p_perfil <> 'cuadro' THEN
                RAISE EXCEPTION 'variante cuadro aplicada a detalle';
            END IF;
            v_cuadro.texto := pg_catalog.repeat('x', 161);
        WHEN 'cuadro_estado_superior' THEN
            IF p_perfil <> 'cuadro' THEN
                RAISE EXCEPTION 'variante cuadro aplicada a detalle';
            END IF;
            v_cuadro.estado_clave := pg_catalog.repeat('x', 17);
        WHEN 'cuadro_fase_superior' THEN
            IF p_perfil <> 'cuadro' THEN
                RAISE EXCEPTION 'variante cuadro aplicada a detalle';
            END IF;
            v_cuadro.fase_clave := pg_catalog.repeat('x', 81);
        WHEN 'cuadro_cursor_superior' THEN
            IF p_perfil <> 'cuadro' THEN
                RAISE EXCEPTION 'variante cuadro aplicada a detalle';
            END IF;
            v_cuadro.cursor := pg_catalog.repeat('x', 44);
        WHEN 'detalle_expediente_superior' THEN
            IF p_perfil <> 'detalle' THEN
                RAISE EXCEPTION 'variante detalle aplicada a cuadro';
            END IF;
            v_detalle.expediente_ref := pg_catalog.repeat('x', 161);
        WHEN 'capacidad_nula' THEN v_capacidad := NULL;
        WHEN 'capacidad_inferior' THEN
            v_capacidad := pg_catalog.decode(
                pg_catalog.repeat('ff', 511), 'hex'
            );
        WHEN 'capacidad_superior' THEN
            v_capacidad := pg_catalog.decode(
                pg_catalog.repeat('ff', 32769), 'hex'
            );
        WHEN 'decision_nula' THEN v_decision := NULL;
        WHEN 'decision_inferior' THEN v_decision := ''::bytea;
        WHEN 'decision_superior' THEN
            v_decision := pg_catalog.decode(
                pg_catalog.repeat('ff', 524289), 'hex'
            );
        WHEN 'motivo_nulo' THEN v_motivo := NULL;
        WHEN 'motivo_inferior' THEN v_motivo := ''::bytea;
        WHEN 'motivo_superior' THEN
            v_motivo := pg_catalog.decode(
                pg_catalog.repeat('ff', 65537), 'hex'
            );
        WHEN 'contexto_nulo' THEN v_contexto := NULL;
        WHEN 'contexto_inferior' THEN v_contexto := ''::bytea;
        WHEN 'contexto_superior' THEN
            v_contexto := pg_catalog.decode(
                pg_catalog.repeat('ff', 262145), 'hex'
            );
        WHEN 'persona_nula' THEN v_persona := NULL;
        WHEN 'persona_inferior' THEN v_persona := 0;
        WHEN 'persona_superior' THEN
            v_persona := 9007199254740992::numeric;
        WHEN 'persona_fraccionaria' THEN v_persona := 1.5;
        WHEN 'perfil_nulo' THEN v_perfil := NULL;
        WHEN 'perfil_inferior' THEN v_perfil := 0;
        WHEN 'perfil_superior' THEN
            v_perfil := 9007199254740992::numeric;
        WHEN 'perfil_fraccionario' THEN v_perfil := 1.5;
        WHEN 'payload_nulo' THEN v_payload := NULL;
        WHEN 'payload_inferior' THEN v_payload := ''::bytea;
        WHEN 'payload_superior' THEN
            v_payload := pg_catalog.decode(
                pg_catalog.repeat('ff', 1048577), 'hex'
            );
        WHEN 'sobre_nulo' THEN v_sobre := NULL;
        WHEN 'sobre_inferior' THEN v_sobre := ''::bytea;
        WHEN 'sobre_superior' THEN
            v_sobre := pg_catalog.decode(
                pg_catalog.repeat('ff', 1048577), 'hex'
            );
        WHEN 'evidencia_nula' THEN v_evidencia := NULL;
        WHEN 'evidencia_inferior' THEN v_evidencia := ''::bytea;
        WHEN 'evidencia_superior' THEN
            v_evidencia := pg_catalog.decode(
                pg_catalog.repeat('ff', 262145), 'hex'
            );
        WHEN 'raiz_nula' THEN v_raiz := NULL;
        WHEN 'raiz_inferior' THEN
            v_raiz := pg_catalog.decode(
                pg_catalog.repeat('ff', 43), 'hex'
            );
        WHEN 'raiz_superior' THEN
            v_raiz := pg_catalog.decode(
                pg_catalog.repeat('ff', 45), 'hex'
            );
        ELSE
            RAISE EXCEPTION 'variante límite CT45 desconocida';
    END CASE;

    IF p_perfil = 'cuadro' THEN
        PERFORM *
          FROM vec_contratacion_temporal
               .consultar_cuadro_rrhh_atestado_v1(
                   v_alcance, v_cuadro, v_capacidad, v_decision,
                   v_motivo, v_contexto, v_persona, v_perfil,
                   v_payload, v_sobre, v_evidencia, v_raiz
               );
    ELSIF p_perfil = 'detalle' THEN
        PERFORM *
          FROM vec_contratacion_temporal
               .consultar_detalle_rrhh_atestado_v1(
                   v_alcance, v_detalle, v_capacidad, v_decision,
                   v_motivo, v_contexto, v_persona, v_perfil,
                   v_payload, v_sobre, v_evidencia, v_raiz
               );
    ELSE
        RAISE EXCEPTION 'perfil límite CT45 desconocido';
    END IF;
END
$funcion$;

REVOKE ALL ON FUNCTION
vec_contratacion_temporal.invocar_limite_fachada_ct45(text, text)
FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
vec_contratacion_temporal.invocar_limite_fachada_ct45(text, text)
TO vec_c2d2_registro_runtime;
