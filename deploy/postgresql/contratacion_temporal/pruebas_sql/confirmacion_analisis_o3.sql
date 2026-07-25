\set ON_ERROR_STOP 1

CREATE FUNCTION public.instante_go_analisis_o3(p_instante timestamptz)
RETURNS text
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.rtrim(
               pg_catalog.rtrim(
                   pg_catalog.to_char(
                       p_instante AT TIME ZONE 'UTC',
                       'YYYY-MM-DD"T"HH24:MI:SS.US'
                   ),
                   '0'
               ),
               '.'
           ) || 'Z'
$funcion$;

REVOKE ALL ON FUNCTION
    public.instante_go_analisis_o3(timestamptz)
FROM PUBLIC;

CREATE TABLE public.vector_confirmacion_analisis_o3 (
    caso text PRIMARY KEY,
    operacion jsonb NOT NULL
);
REVOKE ALL ON TABLE public.vector_confirmacion_analisis_o3 FROM PUBLIC;

CREATE FUNCTION public.preparar_vector_confirmacion_analisis_o3(
    p_decision_ref text DEFAULT 'decision:ct:o3:analisis-001'
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    r vec_contratacion_temporal.reserva_operacion_analisis%ROWTYPE;
    v_anterior jsonb;
    v_siguiente jsonb;
    v_actuacion jsonb;
    v_analisis jsonb;
    v_fuente_rc jsonb;
    v_fuentes jsonb;
    v_politica jsonb;
    v_autorizacion jsonb;
    v_operacion jsonb;
    v_decision jsonb;
    v_decision_bytes bytea;
    v_motivo bytea;
    v_registro record;
    v_ahora timestamptz(6) := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    v_emitida timestamptz(6);
    v_verificada timestamptz(6);
    v_valida_hasta timestamptz(6);
    v_emitida_z text;
    v_verificada_z text;
    v_valida_hasta_z text;
    v_efecto_z text;
    v_contexto_huella text;
    v_prueba_fuentes bytea;
    v_indice integer;
BEGIN
    IF p_decision_ref !~ '^decision:ct:o3:analisis-[0-9]{3}$' THEN
        RAISE EXCEPTION 'referencia de decisión sintética inválida';
    END IF;
    SELECT * INTO STRICT r
      FROM vec_contratacion_temporal.reserva_operacion_analisis
     WHERE expediente_ref = 'expediente:ct:o205:alta_valida';
    SELECT version.agregado_json INTO STRICT v_anterior
      FROM vec_contratacion_temporal.expediente_integral_actual actual
      JOIN vec_contratacion_temporal.expediente_version_integral version
        USING (expediente_ref, version)
     WHERE actual.expediente_ref = r.expediente_ref;
    v_anterior :=
        vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(
            v_anterior
        );
    FOREACH v_indice IN ARRAY ARRAY[0, 1] LOOP
        v_anterior := pg_catalog.jsonb_set(
            v_anterior,
            CASE v_indice
                WHEN 0 THEN ARRAY['creado_en']
                ELSE ARRAY['actualizado_en']
            END,
            pg_catalog.to_jsonb(public.instante_go_analisis_o3(
                CASE v_indice
                    WHEN 0 THEN (v_anterior ->> 'creado_en')::timestamptz
                    ELSE (v_anterior ->> 'actualizado_en')::timestamptz
                END
            ))
        );
    END LOOP;
    FOR v_indice IN 0..
        pg_catalog.jsonb_array_length(v_anterior -> 'actuaciones') - 1
    LOOP
        v_anterior := pg_catalog.jsonb_set(
            v_anterior,
            ARRAY['actuaciones', v_indice::text, 'realizada_en'],
            pg_catalog.to_jsonb(public.instante_go_analisis_o3(
                (v_anterior #>> ARRAY[
                    'actuaciones', v_indice::text, 'realizada_en'
                ])::timestamptz
            ))
        );
    END LOOP;
    v_emitida := v_ahora - interval '1 second';
    v_verificada := v_ahora - interval '500 milliseconds';
    v_valida_hasta := v_ahora + interval '4 seconds';
    v_emitida_z := public.instante_go_analisis_o3(v_emitida);
    v_verificada_z := public.instante_go_analisis_o3(v_verificada);
    v_valida_hasta_z := public.instante_go_analisis_o3(v_valida_hasta);
    v_efecto_z := public.instante_go_analisis_o3(v_ahora);
    v_actuacion := pg_catalog.jsonb_build_object(
        'secuencia', 2,
        'version_expediente', 2,
        'accion_clave', 'contratacion_temporal.analisis.registrar',
        'actor_ref', r.actor_ref,
        'unidad_ref', 'unidad:seleccion',
        'recibo_ref', r.recibo_ref,
        'realizada_en', v_efecto_z,
        'fase_origen', v_anterior ->> 'fase_actual',
        'fase_destino', v_anterior ->> 'fase_actual',
        'estado_origen', v_anterior ->> 'estado_actual',
        'estado_destino', v_anterior ->> 'estado_actual'
    );
    v_analisis := pg_catalog.jsonb_build_object(
        'modalidad_clave', 'contratacion_temporal.interinidad',
        'categoria_ref', 'categoria:auxiliar',
        'grupo_subgrupo', 'C2',
        'causa_clave', 'acumulacion_tareas',
        'periodo', pg_catalog.jsonb_build_object(
            'inicio', '2026-08-01T00:00:00Z',
            'fin', '2026-08-31T00:00:00Z'
        ),
        'porcentaje_jornada', 10000,
        'entrada_rc_esperada', pg_catalog.jsonb_build_object(
            'referencia', 'entrada:rc:o3:001',
            'huella_sha256', pg_catalog.repeat('4', 64)
        ),
        'actuacion_registro', pg_catalog.jsonb_build_object(
            'secuencia', 2,
            'version_expediente', 2,
            'accion_clave', 'contratacion_temporal.analisis.registrar',
            'fase_destino', v_anterior ->> 'fase_actual',
            'recibo_ref', r.recibo_ref
        ),
        'validacion_rc', pg_catalog.jsonb_build_object(
            'resultado', 'no_requerida',
            'entrada_ref', 'entrada:rc:o3:001',
            'huella_entrada_sha256', pg_catalog.repeat('4', 64),
            'fuente_ref', 'fuente:presupuestaria:o3',
            'recibo_ref', 'recibo:fuente:rc:o3',
            'validada_en', v_verificada_z,
            'motivo', 'contratacion_temporal.rc.no_requerida'
        )
    );
    v_siguiente := v_anterior || pg_catalog.jsonb_build_object(
        'version', 2,
        'fase_actual', v_anterior -> 'fase_actual',
        'estado_actual', v_anterior -> 'estado_actual',
        'actualizado_en', v_efecto_z,
        'analisis', v_analisis,
        'actuaciones',
            (v_anterior -> 'actuaciones') ||
            pg_catalog.jsonb_build_array(v_actuacion)
    );
    v_fuente_rc := pg_catalog.jsonb_build_object(
        'tipo', 'validacion_rc',
        'peticion_ref', 'peticion:fuente:rc:o3',
        'respuesta_huella_sha256', pg_catalog.repeat('6', 64),
        'autoridad_ref', 'fuente:presupuestaria:o3',
        'generacion', 1,
        'recibo_respuesta_ref', 'recibo:fuente:rc:o3',
        'sello_respuesta_hmac',
            'hmac-sha256:fuente-analisis-respuesta/v1:' ||
            pg_catalog.repeat('7', 64),
        'verificador_ref', 'verificador:fuente:o3',
        'material_huella_sha256', pg_catalog.repeat('6', 64),
        'emitida_en', v_emitida_z,
        'valida_hasta', v_valida_hasta_z,
        'verificada_en', v_verificada_z,
        'publicacion', NULL
    );
    v_fuentes := pg_catalog.jsonb_build_object(
        'conjunto_huella_sha256', pg_catalog.repeat('0', 64),
        'prueba_canonica_hex', pg_catalog.repeat('0', 128),
        'rc', v_fuente_rc,
        'coste', NULL
    );
    v_politica := pg_catalog.jsonb_build_object(
        'definicion_ref', 'politica:analisis:o3',
        'version', 1,
        'huella_sha256', pg_catalog.repeat('8', 64),
        'accion', 'contratacion_temporal.analisis.registrar',
        'finalidad', 'tramitar_analisis_contratacion_temporal',
        'fase_previa', v_anterior ->> 'fase_actual',
        'estado_previo', v_anterior ->> 'estado_actual',
        'unidad_ref', 'unidad:seleccion',
        'motivo_rectificacion_clave', 'no_aplica',
        'exige_actor_distinto', false
    );
    v_operacion := pg_catalog.jsonb_build_object(
        'esquema',
            'vec.contratacion-temporal.confirmar-operacion-analisis.v1',
        'reserva_ref', r.reserva_ref,
        'recibo_ref', r.recibo_ref,
        'operacion', r.operacion,
        'organizacion_ref', r.organizacion_ref,
        'expediente_ref', r.expediente_ref,
        'version_anterior', r.version_expediente,
        'actor_ref', r.actor_ref,
        'perfil_ref', r.perfil_ref,
        'artefacto_ref', r.artefacto_ref,
        'artefacto_huella_sha256', r.artefacto_huella_sha256,
        'ambito_raiz_hmac', r.ambito_raiz_hmac,
        'huella_semantica_hmac', r.huella_semantica_raiz_hmac,
        'ambito_consulta_hmac', r.ambito_raiz_hmac,
        'huella_consulta_hmac', r.huella_semantica_raiz_hmac,
        'aliases_consulta', pg_catalog.jsonb_build_array(
            pg_catalog.jsonb_build_object(
                'generacion', 2,
                'ambito_hmac', r.ambito_raiz_hmac
            )
        ),
        'expediente_anterior', v_anterior,
        'expediente_siguiente', v_siguiente,
        'actuacion', v_actuacion,
        'fuentes', v_fuentes,
        'autorizacion', '{}'::jsonb,
        'politica', v_politica
    );
    v_prueba_fuentes :=
        vec_contratacion_temporal.reconstruir_prueba_fuentes_analisis_v1(
            v_operacion
        );
    v_fuentes := pg_catalog.jsonb_set(
        v_fuentes, '{prueba_canonica_hex}',
        pg_catalog.to_jsonb(pg_catalog.encode(v_prueba_fuentes, 'hex'))
    );
    v_fuentes := pg_catalog.jsonb_set(
        v_fuentes, '{conjunto_huella_sha256}',
        pg_catalog.to_jsonb(pg_catalog.encode(
            pg_catalog.sha256(v_prueba_fuentes), 'hex'
        ))
    );
    v_operacion := pg_catalog.jsonb_set(
        v_operacion, '{fuentes}', v_fuentes
    );
    v_contexto_huella :=
        vec_contratacion_temporal.huella_contexto_recurso_analisis_v1(
            v_operacion
        );
    SELECT pg_catalog.convert_from(decision, 'UTF8')::jsonb, motivo
      INTO STRICT v_decision, v_motivo
      FROM public.vectores_o2_05
     WHERE caso = 'alta_valida';
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{decision_ref}', pg_catalog.to_jsonb(p_decision_ref)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{accion}',
        '"contratacion_temporal.analisis.registrar"'
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{recurso_ref}', pg_catalog.to_jsonb(r.expediente_ref)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{tipo_recurso}',
        '"analisis_contratacion_temporal"'
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{contexto_recurso_huella_sha256}',
        pg_catalog.to_jsonb(v_contexto_huella)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{finalidad}',
        '"tramitar_analisis_contratacion_temporal"'
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{correlacion_ref}',
        '"correlacion_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"'
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{solicitud_huella_sha256}',
        pg_catalog.to_jsonb(pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to('solicitud:o3:analisis-001', 'UTF8')
        ), 'hex'))
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{emitida_en}',
        pg_catalog.to_jsonb(
            vec_contratacion_temporal.instante_utc_v1(v_ahora)
        )
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{valida_hasta}',
        pg_catalog.to_jsonb(vec_contratacion_temporal.instante_utc_v1(
            v_ahora + interval '2 minutes'
        ))
    );
    v_decision_bytes :=
        vec_autorizacion.decision_contexto_actor_v3_canonica(v_decision);
    SELECT * INTO v_registro
      FROM vec_autorizacion.registrar_decision_contexto_actor_v3(
          v_decision_bytes, v_motivo, 2, 2
      );
    IF NOT FOUND OR v_registro.concedida IS NOT TRUE THEN
        RAISE EXCEPTION
            'no se pudo registrar decisión O3: %, valida=%, canon=%, motivo=%',
            pg_catalog.row_to_json(v_registro),
            vec_autorizacion.decision_contexto_actor_v3_valida(v_decision),
            vec_autorizacion.decision_contexto_actor_v3_canonica(v_decision)
                = v_decision_bytes,
            pg_catalog.encode(pg_catalog.sha256(v_motivo), 'hex')
                = v_decision ->> 'motivo_huella_sha256';
    END IF;
    v_autorizacion := pg_catalog.jsonb_build_object(
        'decision_canonica_hex',
            pg_catalog.encode(v_decision_bytes, 'hex'),
        'motivo_canonico_hex', pg_catalog.encode(v_motivo, 'hex'),
        'persona_version', 2,
        'perfil_version', 2,
        'decision_ref', v_decision ->> 'decision_ref',
        'decision_huella_sha256', pg_catalog.encode(
            pg_catalog.sha256(v_decision_bytes), 'hex'
        ),
        'principal_id', r.actor_ref,
        'perfil_activo_ref', r.perfil_ref,
        'accion', v_politica ->> 'accion',
        'recurso_ref', r.expediente_ref,
        'contexto_recurso_huella_sha256', v_contexto_huella,
        'finalidad', v_politica ->> 'finalidad'
    );
    v_operacion := pg_catalog.jsonb_set(
        v_operacion, '{autorizacion}', v_autorizacion
    );
    INSERT INTO public.vector_confirmacion_analisis_o3
    VALUES ('registrar', v_operacion)
    ON CONFLICT (caso) DO UPDATE
       SET operacion = EXCLUDED.operacion;
END
$funcion$;

CREATE FUNCTION public.invocar_vector_confirmacion_analisis_o3()
RETURNS TABLE (recibo_json jsonb)
LANGUAGE sql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
    SELECT resultado.recibo_json
      FROM public.vector_confirmacion_analisis_o3 vector,
           LATERAL
           vec_contratacion_temporal.confirmar_operacion_analisis_v3(
               vector.operacion
           ) AS resultado
     WHERE vector.caso = 'registrar'
$funcion$;

REVOKE ALL ON FUNCTION
    public.preparar_vector_confirmacion_analisis_o3(text),
    public.invocar_vector_confirmacion_analisis_o3()
FROM PUBLIC;
