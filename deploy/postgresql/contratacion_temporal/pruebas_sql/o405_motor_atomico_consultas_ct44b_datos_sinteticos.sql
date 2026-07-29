\set ON_ERROR_STOP on

-- Fixture focal CT-000044B. Solo contiene referencias sintéticas y no
-- persiste tokens claros. Las fachadas de prueba traducen a valores simples.

CREATE FUNCTION public.preparar_corpus_inicial_ct44b()
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_agregado jsonb;
    v_prueba bytea;
    v_corte numeric(20, 0);
    v_actualizado timestamptz(6) := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp() - interval '2 hours'
    );
BEGIN
    PERFORM pg_catalog.set_config(
        'session_replication_role', 'replica', true
    );
    SELECT ultimo_corte + 1 INTO STRICT v_corte
      FROM vec_contratacion_temporal.control_publicacion_rrhh
     WHERE control;
    v_agregado := pg_catalog.jsonb_build_object(
        'referencia', 'expediente:rrhh:minimizado',
        'organizacion_ref', 'organizacion:diputacion-granada',
        'numero_visible', '2026/CT-MIN',
        'version', 1,
        'flujo', pg_catalog.jsonb_build_object(
            'definicion_ref', 'flujo:rrhh:minimizado',
            'version', 1,
            'huella_sha256', pg_catalog.repeat('a', 64)
        ),
        'fase_actual', 'solicitud',
        'estado_actual', 'en_curso',
        'solicitud', pg_catalog.jsonb_build_object(
            'centro_ref', 'centro:rrhh:minimizado',
            'categoria_ref', 'categoria:rrhh:minimizada',
            'grupo_subgrupo', 'C2',
            'motivo_clave', 'sustitucion',
            'periodo', pg_catalog.jsonb_build_object(
                'inicio', '2026-08-01T00:00:00.000000Z',
                'fin', '2026-09-01T00:00:00.000000Z'
            )
        ),
        'creado_en', pg_catalog.to_char(
            (v_actualizado - interval '1 minute') AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'actualizado_en', pg_catalog.to_char(
            v_actualizado AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'actuaciones', pg_catalog.jsonb_build_array(
            pg_catalog.jsonb_build_object(
                'secuencia', 1,
                'version_expediente', 1,
                'accion_clave', 'actuacion.minimizada.1',
                'realizada_en', pg_catalog.to_char(
                    v_actualizado AT TIME ZONE 'UTC',
                    'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
                ),
                'fase_origen', '',
                'fase_destino', 'solicitud',
                'estado_origen', 'pendiente',
                'estado_destino', 'en_curso'
            )
        )
    );
    v_prueba := pg_catalog.convert_to(
        pg_catalog.repeat('prueba:ct44b:detalle:1:', 8), 'UTF8'
    );
    INSERT INTO vec_contratacion_temporal.expediente_version_integral (
        expediente_ref, version, agregado_json,
        agregado_json_huella_sha256, prueba_canonica,
        prueba_huella_sha256, flujo_ref, flujo_version,
        flujo_huella_sha256, fase_clave, estado, origen_version,
        operacion_ref, registrada_en
    ) VALUES (
        'expediente:rrhh:minimizado', 1, v_agregado,
        pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to(v_agregado::text, 'UTF8')
        ), 'hex'),
        v_prueba, pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex'),
        'flujo:rrhh:minimizado', 1, pg_catalog.repeat('a', 64),
        'solicitud', 'en_curso', 'alta_o2',
        'operacion:ct44b:detalle:1', v_actualizado
    );
    INSERT INTO vec_contratacion_temporal.publicacion_version_rrhh (
        expediente_ref, version, corte_global, organizacion_ref,
        numero_visible, flujo_ref, flujo_version, flujo_huella_sha256,
        fase_clave, estado_clave, centro_ref, categoria_ref,
        modalidad_clave, unidad_ref, creado_en, actualizado_en,
        agregado_huella_sha256, registrada_en
    ) VALUES (
        'expediente:rrhh:minimizado', 1, v_corte,
        'organizacion:diputacion-granada', '2026/CT-MIN',
        'flujo:rrhh:minimizado', 1, pg_catalog.repeat('a', 64),
        'solicitud', 'en_curso', 'centro:rrhh:minimizado',
        'categoria:rrhh:minimizada', NULL, NULL,
        v_actualizado - interval '1 minute', v_actualizado,
        pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to(v_agregado::text, 'UTF8')
        ), 'hex'), v_actualizado
    );
    UPDATE vec_contratacion_temporal.control_publicacion_rrhh
       SET ultimo_corte = v_corte,
           actualizada_en = pg_catalog.date_trunc(
               'microseconds', pg_catalog.clock_timestamp()
           )
     WHERE control;
END
$funcion$;

CREATE FUNCTION public.ampliar_corpus_cuadro_ct44b()
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_corte numeric(20, 0);
    v_indice integer;
    v_actualizado timestamptz(6) := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp() - interval '1 hour'
    );
BEGIN
    PERFORM pg_catalog.set_config(
        'session_replication_role', 'replica', true
    );
    SELECT ultimo_corte INTO STRICT v_corte
      FROM vec_contratacion_temporal.control_publicacion_rrhh
     WHERE control;
    FOR v_indice IN 1..20 LOOP
        INSERT INTO vec_contratacion_temporal.publicacion_version_rrhh (
            expediente_ref, version, corte_global, organizacion_ref,
            numero_visible, flujo_ref, flujo_version,
            flujo_huella_sha256, fase_clave, estado_clave,
            centro_ref, categoria_ref, modalidad_clave, unidad_ref,
            creado_en, actualizado_en, agregado_huella_sha256,
            registrada_en
        ) VALUES (
            'expediente:rrhh:ct44b:'
                || pg_catalog.lpad(v_indice::text, 3, '0'),
            1, v_corte + v_indice,
            'organizacion:diputacion-granada',
            '2026/CT44B-' || pg_catalog.lpad(v_indice::text, 3, '0'),
            'flujo:rrhh:minimizado', 1, pg_catalog.repeat('a', 64),
            'solicitud', 'en_curso', 'centro:rrhh:minimizado',
            'categoria:rrhh:minimizada', NULL, NULL,
            v_actualizado - interval '1 minute',
            v_actualizado + v_indice * interval '1 microsecond',
            pg_catalog.repeat('b', 64),
            v_actualizado + v_indice * interval '1 microsecond'
        );
    END LOOP;
    UPDATE vec_contratacion_temporal.control_publicacion_rrhh
       SET ultimo_corte = v_corte + 20,
           actualizada_en = pg_catalog.date_trunc(
               'microseconds', pg_catalog.clock_timestamp()
           )
     WHERE control;
END
$funcion$;

CREATE FUNCTION public.ajustar_cursor_vector_ct44b(
    p_caso text,
    p_cursor text
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_vector public.vectores_consulta_rrhh_v3%ROWTYPE;
    v_clave record;
    v_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
    v_canon bytea;
    v_contexto bytea;
    v_contexto_huella text;
    v_decision jsonb;
    v_decision_canonica bytea;
    v_capacidad jsonb;
BEGIN
    IF p_cursor !~ '^[A-Za-z0-9_-]{43}$' THEN
        RAISE EXCEPTION 'cursor CT44B de prueba inválido';
    END IF;
    SELECT * INTO STRICT v_vector
      FROM public.vectores_consulta_rrhh_v3
     WHERE caso = p_caso AND perfil = 'cuadro';
    v_consulta := ROW('', '', '', 10, p_cursor);
    v_canon :=
        vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
            v_consulta
        );
    v_contexto := pg_catalog.convert_to(
        '{"ambitos":{"ambito_ref":"organizacion:diputacion-granada",'
        || '"clase_ambito":"organizacion","organizacion_ref":'
        || '"organizacion:diputacion-granada"},"atributos":'
        || '{"consulta_dominio":"vec.contratacion_temporal.'
        || 'consulta_rrhh.cuadro.v1","consulta_huella_sha256":"'
        || pg_catalog.encode(pg_catalog.sha256(v_canon), 'hex')
        || '"}}', 'UTF8'
    );
    v_contexto_huella := pg_catalog.encode(
        pg_catalog.sha256(v_contexto), 'hex'
    );
    v_decision := pg_catalog.convert_from(
        v_vector.decision, 'UTF8'
    )::jsonb;
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{contexto_recurso_huella_sha256}',
        pg_catalog.to_jsonb(v_contexto_huella), false
    );
    v_decision_canonica :=
        vec_autorizacion.decision_contexto_actor_v3_canonica(
            v_decision
        );
    v_capacidad := pg_catalog.convert_from(
        v_vector.capacidad, 'UTF8'
    )::jsonb;
    v_capacidad := pg_catalog.jsonb_set(
        v_capacidad, '{huella_decision_sha256}',
        pg_catalog.to_jsonb(pg_catalog.encode(
            pg_catalog.sha256(v_decision_canonica), 'hex'
        )), false
    );
    v_capacidad := pg_catalog.jsonb_set(
        v_capacidad, '{huella_efecto_sha256}',
        pg_catalog.to_jsonb(v_contexto_huella), false
    );
    v_capacidad := pg_catalog.jsonb_set(
        v_capacidad, '{mac_sha256}',
        pg_catalog.to_jsonb(pg_catalog.repeat('f', 64)), false
    );
    SELECT * INTO STRICT v_clave
      FROM vec_autorizacion_atestada_v3.clave_capacidad_version
     WHERE clave_id = v_capacidad ->> 'clave_id'
       AND version = (v_capacidad ->> 'clave_version')::numeric;
    v_capacidad := pg_catalog.jsonb_set(
        v_capacidad, '{mac_sha256}',
        pg_catalog.to_jsonb(pg_catalog.encode(public.hmac(
            vec_autorizacion_atestada_v3.preimagen_mac(v_capacidad),
            v_clave.secreto_hmac, 'sha256'
        ), 'hex')), false
    );
    UPDATE public.vectores_consulta_rrhh_v3
       SET capacidad =
               vec_autorizacion_atestada_v3.capacidad_canonica(
                   v_capacidad
               ),
           decision = v_decision_canonica
     WHERE caso = p_caso;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.prueba_invocar_motor_cuadro_ct44b(
    p_caso text,
    p_cursor text
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
                'organizacion:diputacion-granada', 'organizacion',
                'organizacion:diputacion-granada'
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
           AND COALESCE(prueba.cursor_huella_sha256, '') =
               (v_resultado.cierre).cursor_huella_sha256
    ) THEN
        RAISE EXCEPTION 'resultado y prueba CT44B divergentes';
    END IF;
    RETURN v_resultado.cursor_siguiente;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.prueba_invocar_motor_detalle_ct44b(
    p_caso text,
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
            ROW('expediente:rrhh:minimizado', p_version),
            v_material
        );
    IF ((v_resultado.detalle).resumen).version <> 1
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .prueba_resultado_recibo_rrhh_v2 prueba
            WHERE prueba.acceso_ref = (v_resultado.cierre).acceso_ref
              AND prueba.generada_en = v_resultado.generada_en
              AND prueba.detalle IS NOT DISTINCT FROM v_resultado.detalle
              AND prueba.tipo_consulta = 'detalle'
       ) THEN
        RAISE EXCEPTION 'detalle y prueba CT44B divergentes';
    END IF;
    RETURN (v_resultado.cierre).acceso_ref;
END
$funcion$;

ALTER FUNCTION
vec_contratacion_temporal.prueba_invocar_motor_cuadro_ct44b(text, text)
OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION
vec_contratacion_temporal.prueba_invocar_motor_detalle_ct44b(text, numeric)
OWNER TO vec_contratacion_temporal_propietario;

REVOKE ALL ON FUNCTION
    public.preparar_corpus_inicial_ct44b(),
    public.ampliar_corpus_cuadro_ct44b(),
    public.ajustar_cursor_vector_ct44b(text, text),
    vec_contratacion_temporal.prueba_invocar_motor_cuadro_ct44b(text, text),
    vec_contratacion_temporal.prueba_invocar_motor_detalle_ct44b(text, numeric)
FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.prueba_invocar_motor_cuadro_ct44b(text, text),
    vec_contratacion_temporal.prueba_invocar_motor_detalle_ct44b(text, numeric)
TO vec_c2d2_registro_runtime;

SELECT public.preparar_corpus_inicial_ct44b();
