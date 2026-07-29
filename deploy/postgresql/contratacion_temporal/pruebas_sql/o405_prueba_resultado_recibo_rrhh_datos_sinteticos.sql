\set ON_ERROR_STOP on

-- Decisión positiva mínima mediante la API de autoridad. La integración
-- completa de autorización presupone otra topología de ACL y no se carga aquí.
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $decision_base$
DECLARE
    v_recibo record;
    v_sesion record;
    v_vinculo jsonb;
    v_decision jsonb;
    v_motivo bytea;
    v_emitida timestamptz(6) := pg_catalog.clock_timestamp();
    v_hasta timestamptz(6) := v_emitida + interval '2 minutes';
    v_resultado record;
BEGIN
    SELECT * INTO STRICT v_recibo
      FROM vec_contexto_actor_v1.registros_contexto
     WHERE registro_contexto_ref =
           'rca_registro_v3_000000000000000000000000';
    SELECT base.autenticacion_verificada_en, base.sesion_emitida_en,
           control.sesion_revalidada_en, control.sesion_valida_hasta
      INTO STRICT v_sesion
      FROM vec_autorizacion.sesion_autenticacion_v1 base
      JOIN vec_autorizacion.control_sesion_v1 control
        USING (sesion_ref)
     WHERE base.sesion_ref =
           'ses_registro_v3_0000000000000000000000';
    v_motivo := pg_catalog.convert_to(
        '{"esquema":"vec.autorizacion.motivo.v2.'
        || 'referencia-opaca-catalogada","referencia":{"catalogo_id":'
        || '"motivos_v3","catalogo_version":1,'
        || '"catalogo_huella_sha256":"'
        || pg_catalog.repeat('9', 64)
        || '","entrada_clave":"motivo_33333333333333333333333333333333"}}',
        'UTF8'
    );
    v_vinculo := pg_catalog.jsonb_build_object(
        'esquema',
          'vec.autenticacion-actor.vinculo.v2.contexto-registrado',
        'bloque_version', 2,
        'autenticacion_ref',
          'aut_registro_v3_0000000000000000000000',
        'autenticacion_huella_sha256', pg_catalog.repeat('5', 64),
        'asercion_ref', 'ase_registro_v3_0000000000000000000000',
        'sesion_ref', 'ses_registro_v3_0000000000000000000000',
        'control_sesion_ref',
          'cse_registro_v3_0000000000000000000000',
        'control_sesion_revision', 1,
        'control_sesion_huella_sha256', pg_catalog.repeat('7', 64),
        'cuenta_ref', 'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',
        'cuenta_ordinaria_ref',
          'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',
        'principal_id', 'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
        'perfil_activo_ref',
          'prf_sintetico_cccccccccccccccccccccccc',
        'cuenta_privilegiada', false,
        'superficie', 'interna_corporativa',
        'metodo_observado', 'certificado',
        'garantia_observada', 'alto',
        'politica_garantia_ref',
          'pga_registro_v3_0000000000000000000000',
        'politica_garantia_huella_sha256',
          pg_catalog.repeat('6', 64),
        'autenticacion_verificada_en', pg_catalog.to_char(
            v_sesion.autenticacion_verificada_en AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'sesion_emitida_en', pg_catalog.to_char(
            v_sesion.sesion_emitida_en AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'sesion_valida_hasta', pg_catalog.to_char(
            v_sesion.sesion_valida_hasta AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'sesion_revalidada_en', pg_catalog.to_char(
            v_sesion.sesion_revalidada_en AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'registro_contexto_ref', v_recibo.registro_contexto_ref,
        'contexto_actor_esquema',
          'vec.contexto-actor.vinculado.v2',
        'contexto_actor_ref',
          'vca_sintetico_dddddddddddddddddddddddd',
        'contexto_actor_version', 2,
        'contexto_actor_cuenta_version', 2,
        'contexto_actor_huella_sha256', v_recibo.huella_sha256,
        'manifiesto_procedencia_huella_sha256',
          v_recibo.manifiesto_procedencia_huella_sha256,
        'autoridad_efectiva', 'autoridad_maestra_acreditada'
    );
    v_decision := pg_catalog.jsonb_build_object(
        'esquema',
          'vec.autorizacion.decision.v3.solicitud-ligada.actor-v2',
        'bloque_version', 3,
        'decision_ref', 'decision:registro-v3:positiva',
        'concedida', true, 'codigo', 'concedida',
        'principal_id', 'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
        'perfil_activo_ref',
          'prf_sintetico_cccccccccccccccccccccccc',
        'accion', 'consultar',
        'recurso_ref', 'expediente_000000000000000000000000',
        'modulo_id', 'bolsa', 'tipo_recurso', 'expediente',
        'contexto_recurso_huella_sha256',
          pg_catalog.repeat('a', 64),
        'finalidad', 'gestion',
        'correlacion_ref',
          'correlacion_11111111111111111111111111111111',
        'esquema_huella_solicitud',
          'vec.autorizacion.solicitud.v3.efectiva-minimizada.actor-v2',
        'solicitud_huella_sha256', pg_catalog.repeat('b', 64),
        'esquema_huella_motivo',
          'vec.autorizacion.motivo.v2.referencia-opaca-catalogada',
        'motivo_huella_sha256',
          pg_catalog.encode(pg_catalog.sha256(v_motivo), 'hex'),
        'vinculo_autenticacion_actor', v_vinculo,
        'asignacion_ref', 'asignacion:registro_v3:v3',
        'asignacion_huella_sha256', pg_catalog.repeat('6', 64),
        'version_rol_ref', 'rol:consulta_rrhh_v3:v1',
        'version_rol_huella_sha256', pg_catalog.repeat('4', 64),
        'control_vigencia_version_rol_ref',
          'rol:consulta_rrhh_v3:v1',
        'control_vigencia_version_rol_revision', 1,
        'control_vigencia_version_rol_huella_sha256',
          pg_catalog.repeat('5', 64),
        'revision_catalogo_politicas', 1,
        'catalogo_politicas_huella_sha256',
          '4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945',
        'politicas_evaluadas', '[]'::jsonb,
        'politicas_aplicables', '[]'::jsonb,
        'garantia_minima', 'alto',
        'campos_permitidos', pg_catalog.jsonb_build_array('estado'),
        'obligaciones', pg_catalog.jsonb_build_array('auditar'),
        'emitida_en', pg_catalog.to_char(
            v_emitida AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'valida_hasta', pg_catalog.to_char(
            v_hasta AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )
    );
    SELECT * INTO STRICT v_resultado
      FROM vec_autorizacion.registrar_decision_contexto_actor_v3(
          vec_autorizacion.decision_contexto_actor_v3_canonica(
              v_decision
          ),
          v_motivo, 2, 2
      );
    IF NOT v_resultado.concedida OR NOT EXISTS (
        SELECT 1
          FROM vec_autorizacion
               .decision_concedida_contexto_actor_v3 decision
         WHERE decision.decision_ref =
               'decision:registro-v3:positiva'
    ) THEN
        RAISE EXCEPTION 'no se creó la decisión base CT43';
    END IF;
END
$decision_base$;
COMMIT;

-- Identidad sintética equivalente a la sesión VEC usada por los vectores
-- nominales. No contiene datos personales ni secretos reutilizables.
BEGIN;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

INSERT INTO vec_identidad_sesiones_v1.cuenta(
    cuenta_ref, cuenta_privilegiada, cuenta_ordinaria_ref,
    provisionada_en, acto_ref
) VALUES (
    'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',
    false, NULL, pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    ),
    'opr_ct43_cuenta_00000000000000000001'
);
INSERT INTO vec_identidad_sesiones_v1.estado_cuenta(
    cuenta_ref, revision, estado, registrada_en, acto_ref
) VALUES (
    'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',
    1, 'activa', pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    ),
    'opr_ct43_estado_00000000000000000001'
);
INSERT INTO vec_identidad_sesiones_v1.estado_cuenta_actual(
    cuenta_ref, revision, actualizada_en, acto_ref
) VALUES (
    'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',
    1, pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    ),
    'opr_ct43_actual_00000000000000000001'
);
INSERT INTO vec_identidad_sesiones_v1.consumo_asercion(
    operacion_ref, esquema_hmac, dominio_hmac_ref,
    clave_hmac_id, clave_hmac_version,
    asercion_id_hmac, sesion_id_hmac, sujeto_id_hmac,
    cuenta_id_hmac, cuenta_ordinaria_id_hmac,
    autenticacion_ref, autenticacion_huella_sha256,
    asercion_ref, sesion_ref, control_sesion_ref,
    control_sesion_revision, cuenta_ref, cuenta_revision,
    cuenta_ordinaria_ref, cuenta_ordinaria_revision, consumida_en
) VALUES (
    'opr_ct43_asercion_0000000000000000001',
    'vec.identidad.hmac-sha256.v1',
    'idh_ct43_sintetico_000000000000000001',
    'clave-hsm-sintetica-ct43', 1,
    pg_catalog.decode(pg_catalog.repeat('11', 32), 'hex'),
    pg_catalog.decode(pg_catalog.repeat('22', 32), 'hex'),
    pg_catalog.decode(pg_catalog.repeat('33', 32), 'hex'),
    pg_catalog.decode(pg_catalog.repeat('44', 32), 'hex'),
    NULL,
    'aut_registro_v3_0000000000000000000000',
    pg_catalog.repeat('5', 64),
    'ase_registro_v3_0000000000000000000000',
    'ses_registro_v3_0000000000000000000000',
    'cse_registro_v3_0000000000000000000000',
    1, 'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa', 1,
    'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa', 1,
    pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    )
);
COMMIT;

CREATE TABLE public.vectores_cierre_ct43 (
    caso text PRIMARY KEY,
    perfil text NOT NULL CHECK (perfil IN ('cuadro', 'detalle')),
    contexto
        vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2
        NOT NULL,
    contenido
        vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2
        NOT NULL
);
REVOKE ALL ON TABLE public.vectores_cierre_ct43 FROM PUBLIC;
GRANT SELECT ON TABLE public.vectores_cierre_ct43
    TO vec_contratacion_temporal_propietario;

CREATE FUNCTION public.preparar_vector_cierre_ct43(
    p_caso text,
    p_perfil text
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_contexto
        vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2;
    v_contenido
        vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2;
    v_consulta_cuadro
        vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
    v_consulta_detalle
        vec_contratacion_temporal.consulta_detalle_rrhh_v1;
    v_resumen vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
    v_solicitud vec_contratacion_temporal.solicitud_operativa_rrhh_v1;
    v_detalle
        vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
    v_hitos vec_contratacion_temporal.hito_expediente_rrhh_v1[];
    v_vector public.vectores_consulta_rrhh_v3%ROWTYPE;
    v_clave record;
    v_decision jsonb;
    v_capacidad jsonb;
    v_decision_canonica bytea;
    v_consulta_canonica bytea;
    v_contexto_recurso bytea;
    v_contexto_huella text;
    v_recurso text;
    v_dominio text;
    v_ahora timestamptz(6) := pg_catalog.date_trunc(
        'milliseconds', pg_catalog.clock_timestamp()
    ) + interval '1 microsecond';
    v_emitida_z text;
    v_expira_z text;
    v_hasta_z text;
    v_organizacion text := 'organizacion:diputacion-granada';
    v_expediente text := 'expediente:rrhh:minimizado';
BEGIN
    v_emitida_z := pg_catalog.to_char(
        v_ahora AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    v_expira_z := pg_catalog.to_char(
        (v_ahora + interval '4.999998 seconds') AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    v_hasta_z := pg_catalog.to_char(
        (v_ahora + interval '2 minutes') AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    PERFORM public.preparar_vector_consulta_rrhh_v3(
        p_caso, p_perfil
    );
    v_consulta_cuadro := ROW('', '', '', 10, '');
    v_consulta_detalle := ROW(v_expediente, 1);
    v_resumen := ROW(
        v_expediente, v_organizacion, '2026/CT-MIN', 1,
        'flujo:rrhh:minimizado', 1, pg_catalog.repeat('a', 64),
        'solicitud', 'en_curso', 'centro:rrhh:minimizado',
        'categoria:rrhh:minimizada', '', '',
        v_ahora - interval '2 minutes',
        v_ahora - interval '1 minute'
    );
    IF p_perfil = 'cuadro' THEN
        v_contexto := ROW(
            v_organizacion, 'organizacion', v_organizacion,
            v_consulta_cuadro,
            NULL::vec_contratacion_temporal.consulta_detalle_rrhh_v1,
            NULL
        );
        v_contenido := ROW(
            'cuadro', v_ahora, ARRAY[v_resumen], false, ''::bytea,
            NULL::vec_contratacion_temporal
                .entrada_detalle_expediente_rrhh_v1
        );
        v_consulta_canonica :=
            vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(
                v_consulta_cuadro
            );
        v_recurso := v_organizacion;
        v_dominio :=
            'vec.contratacion_temporal.consulta_rrhh.cuadro.v1';
    ELSIF p_perfil = 'detalle' THEN
        v_solicitud := ROW(
            'C2', 'sustitucion',
            '2026-08-26T00:00:00Z',
            '2026-09-26T00:00:00Z'
        );
        v_hitos := ARRAY[
            ROW(
                1, 1, 'actuacion.minimizada.1',
                v_ahora - interval '1 minute',
                '', 'solicitud', 'pendiente', 'en_curso'
            )::vec_contratacion_temporal.hito_expediente_rrhh_v1
        ];
        v_detalle := ROW(
            v_resumen, v_solicitud,
            false,
            NULL::vec_contratacion_temporal.analisis_operativo_rrhh_v1,
            0, false,
            NULL::vec_contratacion_temporal.cobertura_operativa_rrhh_v1,
            0, false,
            NULL::vec_contratacion_temporal.asignacion_operativa_rrhh_v1,
            0, v_hitos
        );
        v_contexto := ROW(
            v_organizacion, 'organizacion', v_organizacion,
            NULL::vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
            v_consulta_detalle, NULL
        );
        v_contenido := ROW(
            'detalle', v_ahora,
            ARRAY[]::vec_contratacion_temporal
                .resumen_publicacion_rrhh_v1[],
            false, ''::bytea, v_detalle
        );
        v_consulta_canonica :=
            vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
                v_consulta_detalle
            );
        v_recurso := v_expediente;
        v_dominio :=
            'vec.contratacion_temporal.consulta_rrhh.detalle.v1';
    ELSE
        RAISE EXCEPTION 'perfil CT43 de prueba desconocido';
    END IF;
    v_contexto_recurso := pg_catalog.convert_to(
        '{"ambitos":{"ambito_ref":"' || v_contexto.ambito_ref
        || '","clase_ambito":"' || v_contexto.clase_ambito
        || '","organizacion_ref":"' || v_contexto.organizacion_ref
        || '"},"atributos":{"consulta_dominio":"' || v_dominio
        || '","consulta_huella_sha256":"'
        || pg_catalog.encode(
            pg_catalog.sha256(v_consulta_canonica), 'hex'
        ) || '"}}', 'UTF8'
    );
    v_contexto_huella := pg_catalog.encode(
        pg_catalog.sha256(v_contexto_recurso), 'hex'
    );
    SELECT * INTO STRICT v_vector
      FROM public.vectores_consulta_rrhh_v3
     WHERE caso = p_caso;
    v_decision := pg_catalog.convert_from(
        v_vector.decision, 'UTF8'
    )::jsonb;
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{recurso_ref}',
        pg_catalog.to_jsonb(v_recurso), false
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{contexto_recurso_huella_sha256}',
        pg_catalog.to_jsonb(v_contexto_huella), false
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{emitida_en}',
        pg_catalog.to_jsonb(v_emitida_z), false
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{valida_hasta}',
        pg_catalog.to_jsonb(v_hasta_z), false
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
        v_capacidad, '{efecto_ref}',
        pg_catalog.to_jsonb(v_recurso), false
    );
    v_capacidad := pg_catalog.jsonb_set(
        v_capacidad, '{huella_efecto_sha256}',
        pg_catalog.to_jsonb(v_contexto_huella), false
    );
    v_capacidad := pg_catalog.jsonb_set(
        v_capacidad, '{emitida_en}',
        pg_catalog.to_jsonb(v_emitida_z), false
    );
    v_capacidad := pg_catalog.jsonb_set(
        v_capacidad, '{expira_en}',
        pg_catalog.to_jsonb(v_expira_z), false
    );
    v_capacidad := pg_catalog.jsonb_set(
        v_capacidad, '{verificada_en}',
        pg_catalog.to_jsonb(v_emitida_z), false
    );
    v_capacidad := pg_catalog.jsonb_set(
        v_capacidad, '{decision_valida_hasta}',
        pg_catalog.to_jsonb(v_hasta_z), false
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
    INSERT INTO public.vectores_cierre_ct43(
        caso, perfil, contexto, contenido
    ) VALUES (
        p_caso, p_perfil, v_contexto, v_contenido
    )
    ON CONFLICT (caso) DO UPDATE
       SET perfil = EXCLUDED.perfil,
           contexto = EXCLUDED.contexto,
           contenido = EXCLUDED.contenido;
END
$funcion$;
REVOKE ALL ON FUNCTION
    public.preparar_vector_cierre_ct43(text, text)
FROM PUBLIC;

CREATE FUNCTION
vec_contratacion_temporal.prueba_cerrar_resultado_recibo_ct43(
    p_caso text
)
RETURNS vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_vector public.vectores_consulta_rrhh_v3%ROWTYPE;
    v_cierre public.vectores_cierre_ct43%ROWTYPE;
    v_consumo
        vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3;
BEGIN
    SELECT * INTO STRICT v_vector
      FROM public.vectores_consulta_rrhh_v3
     WHERE caso = p_caso;
    SELECT * INTO STRICT v_cierre
      FROM public.vectores_cierre_ct43
     WHERE caso = p_caso;
    v_consumo := (
        SELECT ROW(
            consumo.decision_ref, consumo.efecto_ref,
            consumo.huella_efecto_sha256,
            consumo.consumo_huella_sha256,
            consumo.auditoria_ref,
            consumo.auditoria_huella_sha256,
            consumo.consumida_en, consumo.consumo_nuevo
        )::vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3
          FROM vec_contratacion_temporal
               .prueba_consumir_consulta_rrhh_v3(
                   p_caso, v_cierre.perfil, ''
               ) consumo
    );
    v_cierre.contenido := ROW(
        (v_cierre.contenido).tipo_consulta,
        pg_catalog.date_trunc(
            'microseconds', pg_catalog.clock_timestamp()
        ),
        (v_cierre.contenido).resumenes,
        (v_cierre.contenido).hay_mas,
        (v_cierre.contenido).cursor_huella,
        (v_cierre.contenido).detalle
    )::vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2;
    RETURN
        vec_contratacion_temporal.cerrar_prueba_resultado_recibo_rrhh_v2(
            v_cierre.contexto, v_cierre.contenido, v_consumo,
            v_vector.capacidad, v_vector.decision, v_vector.motivo,
            v_vector.contexto, v_vector.persona_version,
            v_vector.perfil_version, v_vector.payload,
            v_vector.cose, v_vector.evidencia, v_vector.spki
        );
END
$funcion$;
ALTER FUNCTION
    vec_contratacion_temporal.prueba_cerrar_resultado_recibo_ct43(text)
OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.prueba_cerrar_resultado_recibo_ct43(text)
FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.prueba_cerrar_resultado_recibo_ct43(text)
TO vec_contratacion_temporal_consultor_rrhh;

CREATE FUNCTION public.forzar_sqlstate_prueba_ct43()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_estado text := pg_catalog.current_setting(
        'vec.prueba_ct43_sqlstate', true
    );
BEGIN
    IF v_estado IN ('40001', '40P01', '55P03', '57014') THEN
        RAISE EXCEPTION USING ERRCODE = v_estado,
            MESSAGE = 'estado transitorio sintético CT43';
    END IF;
    RETURN NEW;
END
$funcion$;
REVOKE ALL ON FUNCTION public.forzar_sqlstate_prueba_ct43()
FROM PUBLIC;
