\set ON_ERROR_STOP 1

-- Fixture exclusivamente sintético para probar las dos variantes positivas que
-- faltaban en O3-04. La base es efímera y los objetos de public se retiran al
-- acabar; ninguna identidad representa a una persona real.
CREATE TABLE public.vectores_confirmacion_analisis_o3_variantes (
    caso text PRIMARY KEY,
    operacion jsonb NOT NULL
);
REVOKE ALL ON TABLE
    public.vectores_confirmacion_analisis_o3_variantes
FROM PUBLIC;

BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $autorizacion_b$
DECLARE
    v_ahora timestamptz(6) := pg_catalog.clock_timestamp();
    v_desde timestamptz(6) := v_ahora - interval '10 minutes';
    v_hasta timestamptz(6) := v_ahora + interval '45 minutes';
    v_desde_z text := pg_catalog.to_char(
        v_desde AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    v_hasta_z text := pg_catalog.to_char(
        v_hasta AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    v_asignacion jsonb;
BEGIN
    v_asignacion := pg_catalog.jsonb_build_object(
        'asignacion_id', 'rectificacion_o3b',
        'version', 1,
        'perfil_activo_ref',
            'prf_o3b_cccccccccccccccccccccccccccc',
        'principal_id',
            'per_o3b_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
        'version_rol_ref', 'rol:registro_ct_o205:v1',
        'estado', 'activa',
        'emitida_en', v_desde_z,
        'vigente_desde', v_desde_z,
        'vigente_hasta', v_hasta_z,
        'ambitos', pg_catalog.jsonb_build_array(
            pg_catalog.jsonb_build_object(
                'clave', 'unidad',
                'valores', pg_catalog.jsonb_build_array('seleccion')
            )
        )
    );
    INSERT INTO vec_autorizacion.asignacion_perfil(
        asignacion_ref, asignacion_id, version, perfil_activo_ref,
        principal_id, version_rol_ref, huella_sha256, emitida_en, documento
    ) VALUES (
        'asignacion:rectificacion_o3b:v1', 'rectificacion_o3b', 1,
        'prf_o3b_cccccccccccccccccccccccccccc',
        'per_o3b_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
        'rol:registro_ct_o205:v1', pg_catalog.repeat('b', 64),
        v_desde, v_asignacion
    );
    INSERT INTO vec_autorizacion.asignacion_perfil_actual VALUES (
        'prf_o3b_cccccccccccccccccccccccccccc',
        'asignacion:rectificacion_o3b:v1', v_ahora,
        'usr_rrhh_sintetico_o3b', 'acto:asignacion:rectificacion-o3b'
    );
    INSERT INTO vec_autorizacion.sesion_autenticacion_v1(
        sesion_ref, autenticacion_ref, autenticacion_huella_sha256,
        asercion_ref, cuenta_ref, cuenta_ordinaria_ref,
        cuenta_privilegiada, superficie, metodo_observado,
        garantia_observada, politica_garantia_ref,
        politica_garantia_huella_sha256, autenticacion_verificada_en,
        sesion_emitida_en
    ) VALUES (
        'ses_o3b_111111111111111111111111111111',
        'aut_o3b_222222222222222222222222222222',
        pg_catalog.repeat('c', 64),
        'ase_o3b_333333333333333333333333333333',
        'cta_o3b_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
        'cta_o3b_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
        false, 'interna_corporativa', 'certificado', 'alto',
        'pga_o3b_444444444444444444444444444444',
        pg_catalog.repeat('d', 64), v_desde, v_desde
    );
    INSERT INTO vec_autorizacion.control_sesion_v1 VALUES (
        'cse_o3b_555555555555555555555555555555', 1,
        'ses_o3b_111111111111111111111111111111',
        'activa', pg_catalog.repeat('e', 64), v_desde, v_hasta, v_ahora
    );
    INSERT INTO vec_autorizacion.control_sesion_actual_v1 VALUES (
        'ses_o3b_111111111111111111111111111111',
        'cse_o3b_555555555555555555555555555555',
        1, v_ahora, 'acto:sesion:o3b'
    );
END
$autorizacion_b$;
COMMIT;

CREATE FUNCTION public.construir_vector_confirmacion_o3_variante(
    p_caso text,
    p_operacion text
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
    v_fuente_coste jsonb;
    v_fuentes jsonb;
    v_politica jsonb;
    v_operacion jsonb;
    v_decision jsonb;
    v_vinculo jsonb;
    v_autorizacion jsonb;
    v_decision_bytes bytea;
    v_motivo bytea;
    v_prueba bytea;
    v_registro record;
    v_contexto record;
    v_sesion record;
    v_asignacion record;
    v_rol record;
    v_control record;
    v_ahora timestamptz(6) := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    v_emitida timestamptz(6) := v_ahora - interval '1 second';
    v_verificada timestamptz(6) := v_ahora - interval '500 milliseconds';
    v_valida_hasta timestamptz(6) := v_ahora + interval '4 seconds';
    v_emitida_z text;
    v_verificada_z text;
    v_valida_hasta_z text;
    v_efecto_z text;
    v_fecha_rc_z text := '2026-07-01T00:00:00Z';
    v_contexto_huella text;
    v_version bigint;
    v_indice integer;
    v_motivo_rectificacion text;
    v_sufijo text;
BEGIN
    IF p_caso NOT IN ('registrar_rc_coste', 'rectificar_rc_coste')
       OR p_operacion NOT IN ('registrar', 'rectificar') THEN
        RAISE EXCEPTION 'variante O3 inválida';
    END IF;
    SELECT * INTO STRICT r
      FROM vec_contratacion_temporal.reserva_operacion_analisis
     WHERE expediente_ref = 'expediente:ct:o205:rc_coste'
       AND operacion = p_operacion;
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
            CASE v_indice WHEN 0 THEN ARRAY['creado_en']
                          ELSE ARRAY['actualizado_en'] END,
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
    v_emitida_z := public.instante_go_analisis_o3(v_emitida);
    v_verificada_z := public.instante_go_analisis_o3(v_verificada);
    v_valida_hasta_z := public.instante_go_analisis_o3(v_valida_hasta);
    v_efecto_z := public.instante_go_analisis_o3(v_ahora);
    v_version := r.version_expediente::bigint + 1;
    v_motivo_rectificacion := CASE p_operacion
        WHEN 'registrar' THEN 'no_aplica'
        ELSE 'contratacion_temporal.analisis.rectificacion.ajuste_coste'
    END;
    v_actuacion := pg_catalog.jsonb_build_object(
        'secuencia', v_version,
        'version_expediente', v_version,
        'accion_clave',
            'contratacion_temporal.analisis.' || p_operacion,
        'actor_ref', r.actor_ref,
        'unidad_ref', 'unidad:seleccion',
        'recibo_ref', r.recibo_ref,
        'realizada_en', v_efecto_z,
        'fase_origen', v_anterior ->> 'fase_actual',
        'fase_destino', v_anterior ->> 'fase_actual',
        'estado_origen', v_anterior ->> 'estado_actual',
        'estado_destino', v_anterior ->> 'estado_actual'
    );
    IF p_operacion = 'rectificar' THEN
        v_actuacion := v_actuacion || pg_catalog.jsonb_build_object(
            'observaciones', v_motivo_rectificacion
        );
    END IF;
    v_sufijo := CASE p_operacion
        WHEN 'registrar' THEN 'registro'
        ELSE 'rectificacion'
    END;
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
            'referencia', 'entrada:rc:o3:rc-coste',
            'huella_sha256', pg_catalog.repeat('4', 64)
        ),
        'actuacion_registro', pg_catalog.jsonb_build_object(
            'secuencia', v_version,
            'version_expediente', v_version,
            'accion_clave',
                'contratacion_temporal.analisis.' || p_operacion,
            'fase_destino', v_anterior ->> 'fase_actual',
            'recibo_ref', r.recibo_ref
        ),
        'validacion_rc', pg_catalog.jsonb_build_object(
            'resultado', 'validada',
            'entrada_ref', 'entrada:rc:o3:rc-coste',
            'huella_entrada_sha256', pg_catalog.repeat('4', 64),
            'fuente_ref', 'fuente:presupuestaria:o3:' || v_sufijo,
            'recibo_ref', 'recibo:fuente:rc:o3:' || v_sufijo,
            'validada_en', v_verificada_z,
            'fecha_rc', v_fecha_rc_z,
            'numero', 'rc:2026:000042',
            'importe', pg_catalog.jsonb_build_object(
                'centimos', 5000000, 'moneda', 'EUR'
            ),
            'documento_ref', 'documento:rc:2026:000042'
        ),
        'coste_previsto', pg_catalog.jsonb_build_object(
            'centimos', CASE p_operacion
                WHEN 'registrar' THEN 4000000 ELSE 3900000 END,
            'moneda', 'EUR'
        ),
        'fuente_coste_ref', 'fuente:coste:o3:' || v_sufijo
    );
    v_siguiente := v_anterior || pg_catalog.jsonb_build_object(
        'version', v_version,
        'fase_actual', v_anterior -> 'fase_actual',
        'estado_actual', v_anterior -> 'estado_actual',
        'actualizado_en', v_efecto_z,
        'analisis', v_analisis,
        'actuaciones', (v_anterior -> 'actuaciones') ||
            pg_catalog.jsonb_build_array(v_actuacion)
    );
    v_fuente_rc := pg_catalog.jsonb_build_object(
        'tipo', 'validacion_rc',
        'peticion_ref', 'peticion:fuente:rc:o3:' || v_sufijo,
        'respuesta_huella_sha256', pg_catalog.repeat(
            CASE p_operacion WHEN 'registrar' THEN '6' ELSE '8' END, 64
        ),
        'autoridad_ref', 'fuente:presupuestaria:o3:' || v_sufijo,
        'generacion', 1,
        'recibo_respuesta_ref', 'recibo:fuente:rc:o3:' || v_sufijo,
        'sello_respuesta_hmac',
            'hmac-sha256:fuente-analisis-respuesta/v1:' ||
            pg_catalog.repeat(
                CASE p_operacion WHEN 'registrar' THEN '7' ELSE '9' END, 64
            ),
        'verificador_ref', 'verificador:fuente:rc:o3',
        'material_huella_sha256', pg_catalog.repeat(
            CASE p_operacion WHEN 'registrar' THEN '6' ELSE '8' END, 64
        ),
        'emitida_en', v_emitida_z,
        'valida_hasta', v_valida_hasta_z,
        'verificada_en', v_verificada_z,
        'publicacion', NULL
    );
    v_fuente_coste := pg_catalog.jsonb_build_object(
        'tipo', 'calculo_coste',
        'peticion_ref', 'peticion:fuente:coste:o3:' || v_sufijo,
        'respuesta_huella_sha256', pg_catalog.repeat(
            CASE p_operacion WHEN 'registrar' THEN 'a' ELSE 'c' END, 64
        ),
        'autoridad_ref', 'fuente:coste:o3:' || v_sufijo,
        'generacion', 1,
        'recibo_respuesta_ref', 'recibo:fuente:coste:o3:' || v_sufijo,
        'sello_respuesta_hmac',
            'hmac-sha256:fuente-analisis-respuesta/v1:' ||
            pg_catalog.repeat(
                CASE p_operacion WHEN 'registrar' THEN 'b' ELSE 'd' END, 64
            ),
        'verificador_ref', 'verificador:fuente:coste:o3',
        'material_huella_sha256', pg_catalog.repeat(
            CASE p_operacion WHEN 'registrar' THEN 'a' ELSE 'c' END, 64
        ),
        'emitida_en', v_emitida_z,
        'valida_hasta', v_valida_hasta_z,
        'verificada_en', v_verificada_z,
        'publicacion', NULL
    );
    v_fuentes := pg_catalog.jsonb_build_object(
        'conjunto_huella_sha256', pg_catalog.repeat('0', 64),
        'prueba_canonica_hex', pg_catalog.repeat('0', 128),
        'rc', v_fuente_rc,
        'coste', v_fuente_coste
    );
    v_politica := pg_catalog.jsonb_build_object(
        'definicion_ref', 'politica:analisis:o3',
        'version', 1,
        'huella_sha256', pg_catalog.repeat('8', 64),
        'accion', 'contratacion_temporal.analisis.' || p_operacion,
        'finalidad', 'tramitar_analisis_contratacion_temporal',
        'fase_previa', v_anterior ->> 'fase_actual',
        'estado_previo', v_anterior ->> 'estado_actual',
        'unidad_ref', 'unidad:seleccion',
        'motivo_rectificacion_clave', v_motivo_rectificacion,
        'exige_actor_distinto', p_operacion = 'rectificar'
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
                'generacion', 2, 'ambito_hmac', r.ambito_raiz_hmac
            )
        ),
        'expediente_anterior', v_anterior,
        'expediente_siguiente', v_siguiente,
        'actuacion', v_actuacion,
        'fuentes', v_fuentes,
        'autorizacion', '{}'::jsonb,
        'politica', v_politica
    );
    v_prueba :=
        vec_contratacion_temporal.reconstruir_prueba_fuentes_analisis_v1(
            v_operacion
        );
    v_fuentes := pg_catalog.jsonb_set(
        v_fuentes, '{prueba_canonica_hex}',
        pg_catalog.to_jsonb(pg_catalog.encode(v_prueba, 'hex'))
    );
    v_fuentes := pg_catalog.jsonb_set(
        v_fuentes, '{conjunto_huella_sha256}',
        pg_catalog.to_jsonb(pg_catalog.encode(
            pg_catalog.sha256(v_prueba), 'hex'
        ))
    );
    v_operacion := pg_catalog.jsonb_set(
        v_operacion, '{fuentes}', v_fuentes
    );
    v_contexto_huella :=
        vec_contratacion_temporal.huella_contexto_recurso_analisis_v2(
            v_operacion
        );
    SELECT documento, motivo_canonico
      INTO STRICT v_decision, v_motivo
      FROM vec_autorizacion.decision_concedida_contexto_actor_v3
     WHERE decision_ref = 'decision:registro-v3:positiva';
    IF p_operacion = 'rectificar' THEN
        SELECT * INTO STRICT v_contexto
          FROM vec_contexto_actor_v1.registros_contexto
         WHERE registro_contexto_ref =
               'rca_o3b_666666666666666666666666666666';
        SELECT base.*, control.control_sesion_ref,
               control.revision AS control_revision,
               control.huella_sha256 AS control_huella,
               control.sesion_revalidada_en, control.sesion_valida_hasta
          INTO STRICT v_sesion
          FROM vec_autorizacion.sesion_autenticacion_v1 base
          JOIN vec_autorizacion.control_sesion_actual_v1 actual
            USING (sesion_ref)
          JOIN vec_autorizacion.control_sesion_v1 control
            USING (sesion_ref, control_sesion_ref, revision)
         WHERE base.sesion_ref =
               'ses_o3b_111111111111111111111111111111';
        SELECT * INTO STRICT v_asignacion
          FROM vec_autorizacion.asignacion_perfil
         WHERE asignacion_ref = 'asignacion:rectificacion_o3b:v1';
        v_vinculo := pg_catalog.jsonb_build_object(
            'esquema',
                'vec.autenticacion-actor.vinculo.v2.contexto-registrado',
            'bloque_version', 2,
            'autenticacion_ref', v_sesion.autenticacion_ref,
            'autenticacion_huella_sha256',
                v_sesion.autenticacion_huella_sha256,
            'asercion_ref', v_sesion.asercion_ref,
            'sesion_ref', v_sesion.sesion_ref,
            'control_sesion_ref', v_sesion.control_sesion_ref,
            'control_sesion_revision', v_sesion.control_revision,
            'control_sesion_huella_sha256', v_sesion.control_huella,
            'cuenta_ref', v_sesion.cuenta_ref,
            'cuenta_ordinaria_ref', v_sesion.cuenta_ordinaria_ref,
            'principal_id', r.actor_ref,
            'perfil_activo_ref', r.perfil_ref,
            'cuenta_privilegiada', false,
            'superficie', v_sesion.superficie,
            'metodo_observado', v_sesion.metodo_observado,
            'garantia_observada', v_sesion.garantia_observada,
            'politica_garantia_ref', v_sesion.politica_garantia_ref,
            'politica_garantia_huella_sha256',
                v_sesion.politica_garantia_huella_sha256,
            'autenticacion_verificada_en',
                public.instante_go_analisis_o3(
                    v_sesion.autenticacion_verificada_en
                ),
            'sesion_emitida_en',
                public.instante_go_analisis_o3(v_sesion.sesion_emitida_en),
            'sesion_valida_hasta',
                public.instante_go_analisis_o3(
                    v_sesion.sesion_valida_hasta
                ),
            'sesion_revalidada_en',
                public.instante_go_analisis_o3(
                    v_sesion.sesion_revalidada_en
                ),
            'registro_contexto_ref', v_contexto.registro_contexto_ref,
            'contexto_actor_esquema', 'vec.contexto-actor.vinculado.v2',
            'contexto_actor_ref',
                'vca_o3b_dddddddddddddddddddddddddddd',
            'contexto_actor_version', 2,
            'contexto_actor_cuenta_version', 2,
            'contexto_actor_huella_sha256', v_contexto.huella_sha256,
            'manifiesto_procedencia_huella_sha256',
                v_contexto.manifiesto_procedencia_huella_sha256,
            'autoridad_efectiva', 'autoridad_maestra_acreditada'
        );
        v_decision := pg_catalog.jsonb_set(
            v_decision, '{vinculo_autenticacion_actor}', v_vinculo
        );
        v_decision := pg_catalog.jsonb_set(
            v_decision, '{asignacion_ref}',
            pg_catalog.to_jsonb(v_asignacion.asignacion_ref)
        );
        v_decision := pg_catalog.jsonb_set(
            v_decision, '{asignacion_huella_sha256}',
            pg_catalog.to_jsonb(v_asignacion.huella_sha256)
        );
    ELSE
        SELECT asignacion.*, rol.huella_sha256 AS rol_huella,
               control.revision AS control_revision,
               control.huella_sha256 AS control_huella
          INTO STRICT v_asignacion
          FROM vec_autorizacion.asignacion_perfil_actual actual
          JOIN vec_autorizacion.asignacion_perfil asignacion
            USING (perfil_activo_ref, asignacion_ref)
          JOIN vec_autorizacion.version_rol rol
            USING (version_rol_ref)
          JOIN vec_autorizacion.control_vigencia_version_rol_actual ca
            USING (version_rol_ref)
          JOIN vec_autorizacion.control_vigencia_version_rol control
            USING (version_rol_ref, revision)
         WHERE actual.perfil_activo_ref = r.perfil_ref;
    END IF;
    SELECT rol.*, control.revision AS control_revision,
           control.huella_sha256 AS control_huella
      INTO STRICT v_rol
      FROM vec_autorizacion.version_rol rol
      JOIN vec_autorizacion.control_vigencia_version_rol_actual actual
        USING (version_rol_ref)
      JOIN vec_autorizacion.control_vigencia_version_rol control
        USING (version_rol_ref, revision)
     WHERE rol.version_rol_ref = v_asignacion.version_rol_ref;
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{decision_ref}',
        pg_catalog.to_jsonb('decision:ct:o3:' || p_caso)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{principal_id}', pg_catalog.to_jsonb(r.actor_ref)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{perfil_activo_ref}', pg_catalog.to_jsonb(r.perfil_ref)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{accion}', pg_catalog.to_jsonb(v_politica ->> 'accion')
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{recurso_ref}', pg_catalog.to_jsonb(r.expediente_ref)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{modulo_id}', '"contratacion_temporal"'
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{tipo_recurso}', '"analisis_contratacion_temporal"'
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
        pg_catalog.to_jsonb('correlacion_' || pg_catalog.substr(
            pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
                'correlacion:' || p_caso, 'UTF8'
            )), 'hex'), 1, 32
        ))
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{solicitud_huella_sha256}',
        pg_catalog.to_jsonb(pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to('solicitud:' || p_caso, 'UTF8')
        ), 'hex'))
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{asignacion_ref}',
        pg_catalog.to_jsonb(v_asignacion.asignacion_ref)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{asignacion_huella_sha256}',
        pg_catalog.to_jsonb(v_asignacion.huella_sha256)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{version_rol_ref}',
        pg_catalog.to_jsonb(v_rol.version_rol_ref)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{version_rol_huella_sha256}',
        pg_catalog.to_jsonb(v_rol.huella_sha256)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{control_vigencia_version_rol_ref}',
        pg_catalog.to_jsonb(v_rol.version_rol_ref)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{control_vigencia_version_rol_revision}',
        pg_catalog.to_jsonb(v_rol.control_revision)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{control_vigencia_version_rol_huella_sha256}',
        pg_catalog.to_jsonb(v_rol.control_huella)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{emitida_en}', pg_catalog.to_jsonb(v_efecto_z)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{valida_hasta}',
        pg_catalog.to_jsonb(public.instante_go_analisis_o3(
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
        RAISE EXCEPTION 'no se pudo registrar decisión VEC para %', p_caso;
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
    INSERT INTO public.vectores_confirmacion_analisis_o3_variantes
    VALUES (p_caso, v_operacion)
    ON CONFLICT (caso) DO UPDATE
       SET operacion = EXCLUDED.operacion;
END
$funcion$;

CREATE FUNCTION public.invocar_confirmacion_o3_variante(p_caso text)
RETURNS TABLE (recibo_json jsonb)
LANGUAGE sql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
    SELECT resultado.recibo_json
      FROM public.vectores_confirmacion_analisis_o3_variantes vector,
           LATERAL
           vec_contratacion_temporal.confirmar_operacion_analisis_v3(
               vector.operacion
           ) AS resultado
     WHERE vector.caso = p_caso
$funcion$;

CREATE FUNCTION public.mutar_vector_o3_variante(
    p_origen text,
    p_destino text,
    p_tipo text
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_operacion jsonb;
    v_valor text;
BEGIN
    SELECT operacion INTO STRICT v_operacion
      FROM public.vectores_confirmacion_analisis_o3_variantes
     WHERE caso = p_origen;
    IF p_tipo = 'unidad' THEN
        v_operacion := pg_catalog.jsonb_set(
            v_operacion, '{politica,unidad_ref}', '"unidad:intervencion"'
        );
        v_operacion := pg_catalog.jsonb_set(
            v_operacion, '{actuacion,unidad_ref}', '"unidad:intervencion"'
        );
        v_operacion := pg_catalog.jsonb_set(
            v_operacion,
            ARRAY[
                'expediente_siguiente', 'actuaciones',
                ((v_operacion ->> 'version_anterior')::integer)::text,
                'unidad_ref'
            ],
            '"unidad:intervencion"'
        );
    ELSIF p_tipo = 'motivo' THEN
        v_valor :=
            'contratacion_temporal.analisis.rectificacion.ajuste_periodo';
        v_operacion := pg_catalog.jsonb_set(
            v_operacion, '{politica,motivo_rectificacion_clave}',
            pg_catalog.to_jsonb(v_valor)
        );
        v_operacion := pg_catalog.jsonb_set(
            v_operacion, '{actuacion,observaciones}',
            pg_catalog.to_jsonb(v_valor)
        );
        v_operacion := pg_catalog.jsonb_set(
            v_operacion,
            ARRAY[
                'expediente_siguiente', 'actuaciones',
                ((v_operacion ->> 'version_anterior')::integer)::text,
                'observaciones'
            ],
            pg_catalog.to_jsonb(v_valor)
        );
    ELSE
        RAISE EXCEPTION 'mutación O3 desconocida';
    END IF;
    INSERT INTO public.vectores_confirmacion_analisis_o3_variantes
    VALUES (p_destino, v_operacion)
    ON CONFLICT (caso) DO UPDATE SET operacion = EXCLUDED.operacion;
END
$funcion$;

REVOKE ALL ON FUNCTION
    public.construir_vector_confirmacion_o3_variante(text, text),
    public.invocar_confirmacion_o3_variante(text),
    public.mutar_vector_o3_variante(text, text, text)
FROM PUBLIC;
