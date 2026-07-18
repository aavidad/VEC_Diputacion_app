-- Acredita la frontera publicada con session_user de LOGIN real. Todo el
-- sembrado RBAC/ABAC y los efectos se revierten; ninguna cuenta recibe al
-- propietario ni acceso a tablas del esquema de convocatorias.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE TABLE public.fixture_decision_borrador_runtime (
    decision_ref text PRIMARY KEY,
    prueba jsonb NOT NULL,
    decision_canonica bytea NOT NULL,
    contexto_canonico bytea NOT NULL,
    huella_decision_sha256 text NOT NULL,
    atestacion_ref text NOT NULL,
    huella_atestacion_sha256 text NOT NULL
);
GRANT USAGE ON SCHEMA public
    TO vec_autorizacion_propietario,
       vec_convocatorias_ejecutor_prueba,
       vec_convocatorias_proyector_prueba;
GRANT SELECT ON public.fixture_decision_borrador_runtime
    TO vec_bolsa_convocatorias_propietario,
       vec_convocatorias_ejecutor_prueba,
       vec_convocatorias_proyector_prueba;
GRANT SELECT ON public.fixture_reserva_borrador_concurrente
    TO vec_autorizacion_propietario,
       vec_convocatorias_ejecutor_prueba,
       vec_convocatorias_proyector_prueba;

SET LOCAL ROLE vec_autorizacion_propietario;

DO $rbac$
DECLARE
    ahora timestamptz(6) := date_trunc('microseconds', clock_timestamp());
    publicada timestamptz(6) := ahora - interval '1 hour';
    valida_hasta timestamptz(6) := ahora + interval '10 minutes';
    documento_rol jsonb;
    documento_control jsonb;
    documento_asignacion jsonb;
BEGIN
    documento_rol := jsonb_build_object(
        'rol_id', 'convocatorias_runtime_prueba', 'version', 1,
        'publicada_en', to_char(publicada AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'concesiones', jsonb_build_array(
            jsonb_build_object(
                'accion', 'bolsa.convocatoria.borrador.crear',
                'modulo_id', 'bolsa',
                'tipo_recurso', 'version_convocatoria_gobernada',
                'finalidades', jsonb_build_array('gobierno_convocatorias'),
                'garantia_minima', 'alto',
                'campos_permitidos', jsonb_build_array(
                    'auditoria','evento_outbox','version_convocatoria'
                ), 'obligaciones', '[]'::jsonb
            ),
            jsonb_build_object(
                'accion', 'bolsa.convocatoria.borrador.actualizar',
                'modulo_id', 'bolsa',
                'tipo_recurso', 'version_convocatoria_gobernada',
                'finalidades', jsonb_build_array('gobierno_convocatorias'),
                'garantia_minima', 'alto',
                'campos_permitidos', jsonb_build_array(
                    'auditoria','evento_outbox','version_convocatoria'
                ), 'obligaciones', '[]'::jsonb
            ),
            jsonb_build_object(
                'accion', 'bolsa.convocatoria.borrador.listar',
                'modulo_id', 'bolsa',
                'tipo_recurso',
                    'coleccion_versiones_convocatoria_gobernada',
                'finalidades', jsonb_build_array(
                    'consulta_interna_convocatorias'
                ), 'garantia_minima', 'alto',
                'campos_permitidos', jsonb_build_array(
                    'version_convocatoria'
                ), 'obligaciones', '[]'::jsonb
            ),
            jsonb_build_object(
                'accion', 'bolsa.convocatoria.borrador.consultar',
                'modulo_id', 'bolsa',
                'tipo_recurso', 'version_convocatoria_gobernada',
                'finalidades', jsonb_build_array(
                    'consulta_interna_convocatorias'
                ), 'garantia_minima', 'alto',
                'campos_permitidos', jsonb_build_array(
                    'version_convocatoria'
                ), 'obligaciones', '[]'::jsonb
            )
        )
    );
    INSERT INTO vec_autorizacion.version_rol(
        version_rol_ref, rol_id, version, huella_sha256,
        publicada_en, documento
    ) VALUES (
        'rol:convocatorias_runtime_prueba:v1',
        'convocatorias_runtime_prueba', 1, repeat('1',64),
        publicada, documento_rol
    );
    documento_control := jsonb_build_object(
        'version_rol_ref', 'rol:convocatorias_runtime_prueba:v1',
        'revision', 1, 'estado', 'habilitada',
        'actualizado_en', to_char(publicada AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    );
    INSERT INTO vec_autorizacion.control_vigencia_version_rol(
        version_rol_ref, revision, estado, huella_sha256,
        actualizado_en, documento
    ) VALUES (
        'rol:convocatorias_runtime_prueba:v1', 1, 'habilitada',
        repeat('2',64), publicada, documento_control
    );
    INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
    VALUES (
        'rol:convocatorias_runtime_prueba:v1', 1, publicada,
        'prueba:runtime', 'acto:runtime:rol'
    );
    documento_asignacion := jsonb_build_object(
        'asignacion_id', 'convocatorias_runtime_prueba', 'version', 1,
        'perfil_activo_ref', 'prf_convocatorias_runtime_prueba_000001',
        'principal_id', 'per_convocatorias_runtime_prueba_000001',
        'version_rol_ref', 'rol:convocatorias_runtime_prueba:v1',
        'emitida_en', to_char(publicada AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'vigente_desde', to_char(publicada AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'vigente_hasta', to_char(valida_hasta AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'ambitos', jsonb_build_array(jsonb_build_object(
            'clave','organizacion','valores',
            jsonb_build_array('org_0123456789abcdef')
        ))
    );
    INSERT INTO vec_autorizacion.asignacion_perfil(
        asignacion_ref, asignacion_id, version, perfil_activo_ref,
        principal_id, version_rol_ref, huella_sha256, emitida_en, documento
    ) VALUES (
        'asignacion:convocatorias_runtime_prueba:v1',
        'convocatorias_runtime_prueba', 1,
        'prf_convocatorias_runtime_prueba_000001',
        'per_convocatorias_runtime_prueba_000001',
        'rol:convocatorias_runtime_prueba:v1', repeat('3',64),
        publicada, documento_asignacion
    );
    INSERT INTO vec_autorizacion.asignacion_perfil_actual VALUES (
        'prf_convocatorias_runtime_prueba_000001',
        'asignacion:convocatorias_runtime_prueba:v1', publicada,
        'prueba:runtime', 'acto:runtime:asignacion'
    );

    INSERT INTO vec_autorizacion.sesion_autenticacion_v1(
        sesion_ref, autenticacion_ref, autenticacion_huella_sha256,
        asercion_ref, cuenta_ref, cuenta_ordinaria_ref, cuenta_privilegiada,
        superficie, metodo_observado, garantia_observada,
        politica_garantia_ref, politica_garantia_huella_sha256,
        autenticacion_verificada_en, sesion_emitida_en
    ) VALUES (
        'ses_convocatorias_runtime_prueba_000001',
        'aut_convocatorias_runtime_prueba_000001', repeat('a',64),
        'ase_convocatorias_runtime_prueba_000001',
        'cta_convocatorias_runtime_prueba_000001',
        'cta_convocatorias_runtime_prueba_000001', false,
        'interna_corporativa', 'kerberos_ad', 'alto',
        'pga_convocatorias_runtime_prueba_000001', repeat('c',64),
        ahora - interval '10 minutes', ahora - interval '9 minutes'
    );
    INSERT INTO vec_autorizacion.control_sesion_v1(
        control_sesion_ref, revision, sesion_ref, estado, huella_sha256,
        sesion_revalidada_en, sesion_valida_hasta
    ) VALUES (
        'cse_convocatorias_runtime_prueba_000001', 1,
        'ses_convocatorias_runtime_prueba_000001', 'activa', repeat('b',64),
        ahora - interval '1 minute', valida_hasta
    );
    INSERT INTO vec_autorizacion.control_sesion_actual_v1 VALUES (
        'ses_convocatorias_runtime_prueba_000001',
        'cse_convocatorias_runtime_prueba_000001', 1,
        ahora - interval '1 second', 'acto:runtime:sesion'
    );
    INSERT INTO vec_autorizacion.contexto_actor_v1(
        contexto_actor_ref, version, cuenta_ref, principal_id,
        perfil_activo_ref, estado, huella_sha256, vigente_desde, vigente_hasta
    ) VALUES (
        'vca_convocatorias_runtime_prueba_000001', 1,
        'cta_convocatorias_runtime_prueba_000001',
        'per_convocatorias_runtime_prueba_000001',
        'prf_convocatorias_runtime_prueba_000001', 'activo', repeat('d',64),
        ahora - interval '1 hour', valida_hasta
    );
    INSERT INTO vec_autorizacion.contexto_actor_actual_v1 VALUES (
        'cta_convocatorias_runtime_prueba_000001',
        'prf_convocatorias_runtime_prueba_000001',
        'vca_convocatorias_runtime_prueba_000001', 1,
        ahora - interval '1 second', 'acto:runtime:contexto'
    );
END
$rbac$;

RESET ROLE;
CREATE FUNCTION public.crear_decision_borrador_runtime_prueba(
    p_decision_ref text, p_accion text, p_tipo_recurso text,
    p_recurso_ref text, p_finalidad text, p_campos jsonb,
    p_contexto bytea
)
RETURNS void
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    ahora timestamptz(6) := date_trunc('microseconds', clock_timestamp());
    emitida timestamptz(6) := ahora - interval '1 second';
    valida timestamptz(6) := ahora + interval '2 minutes';
    autenticacion_verificada timestamptz(6);
    sesion_emitida timestamptz(6);
    sesion_revalidada timestamptz(6);
    sesion_valida_hasta timestamptz(6);
    control record;
    vinculo jsonb;
    documento jsonb;
    canonica jsonb;
    canonica_bytes bytea;
    huella text;
    evidencia_huella text := encode(
        sha256(convert_to('{}', 'UTF8')), 'hex'
    );
BEGIN
    SELECT revision, huella_sha256 INTO STRICT control
      FROM vec_autorizacion.control_catalogo_politicas
     WHERE control_id;
    SELECT s.autenticacion_verificada_en, s.sesion_emitida_en,
           c.sesion_revalidada_en, c.sesion_valida_hasta
      INTO STRICT autenticacion_verificada, sesion_emitida,
                  sesion_revalidada, sesion_valida_hasta
      FROM vec_autorizacion.sesion_autenticacion_v1 AS s
      JOIN vec_autorizacion.control_sesion_actual_v1 AS a
        ON a.sesion_ref = s.sesion_ref
      JOIN vec_autorizacion.control_sesion_v1 AS c
        ON c.sesion_ref = a.sesion_ref
       AND c.control_sesion_ref = a.control_sesion_ref
       AND c.revision = a.revision
     WHERE s.sesion_ref = 'ses_convocatorias_runtime_prueba_000001';
    vinculo := jsonb_build_object(
        'bloque_version', 1,
        'autenticacion_ref', 'aut_convocatorias_runtime_prueba_000001',
        'autenticacion_huella_sha256', repeat('a',64),
        'asercion_ref', 'ase_convocatorias_runtime_prueba_000001',
        'sesion_ref', 'ses_convocatorias_runtime_prueba_000001',
        'control_sesion_ref', 'cse_convocatorias_runtime_prueba_000001',
        'control_sesion_revision', 1,
        'control_sesion_huella_sha256', repeat('b',64),
        'cuenta_ref', 'cta_convocatorias_runtime_prueba_000001',
        'cuenta_ordinaria_ref', 'cta_convocatorias_runtime_prueba_000001',
        'principal_id', 'per_convocatorias_runtime_prueba_000001',
        'perfil_activo_ref', 'prf_convocatorias_runtime_prueba_000001',
        'cuenta_privilegiada', false,
        'superficie', 'interna_corporativa',
        'metodo_observado', 'kerberos_ad',
        'garantia_observada', 'alto',
        'politica_garantia_ref',
            'pga_convocatorias_runtime_prueba_000001',
        'politica_garantia_huella_sha256', repeat('c',64),
        'autenticacion_verificada_en', to_char(
            autenticacion_verificada AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'sesion_emitida_en', to_char(
            sesion_emitida AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'sesion_valida_hasta', to_char(
            sesion_valida_hasta AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'sesion_revalidada_en', to_char(
            sesion_revalidada AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'contexto_actor_ref', 'vca_convocatorias_runtime_prueba_000001',
        'contexto_actor_version', 1,
        'contexto_actor_huella_sha256', repeat('d',64)
    );

    documento := jsonb_build_object(
        'decision_ref', p_decision_ref, 'concedida', true,
        'codigo', 'concedida',
        'principal_id', 'per_convocatorias_runtime_prueba_000001',
        'perfil_activo_ref', 'prf_convocatorias_runtime_prueba_000001',
        'accion', p_accion, 'recurso_ref', p_recurso_ref,
        'modulo_id', 'bolsa', 'tipo_recurso', p_tipo_recurso,
        'contexto_recurso_huella_sha256',
            encode(sha256(p_contexto), 'hex'),
        'finalidad', p_finalidad,
        'correlacion_ref', 'correlacion:convocatorias:runtime',
        'vinculo_autenticacion_actor', vinculo,
        'asignacion_ref', 'asignacion:convocatorias_runtime_prueba:v1',
        'asignacion_huella_sha256', repeat('3',64),
        'version_rol_ref', 'rol:convocatorias_runtime_prueba:v1',
        'version_rol_huella_sha256', repeat('1',64),
        'control_vigencia_version_rol_ref',
            'rol:convocatorias_runtime_prueba:v1',
        'control_vigencia_version_rol_revision', 1,
        'control_vigencia_version_rol_huella_sha256', repeat('2',64),
        'revision_catalogo_politicas', control.revision,
        'catalogo_politicas_huella_sha256', control.huella_sha256,
        'politicas_evaluadas_refs', '[]'::jsonb,
        'politicas_evaluadas_huellas_sha256', '{}'::jsonb,
        'politicas_refs', '[]'::jsonb,
        'politicas_huellas_sha256', '{}'::jsonb,
        'garantia_minima', 'alto', 'campos_permitidos', p_campos,
        'obligaciones', '[]'::jsonb,
        'emitida_en', to_char(emitida AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'valida_hasta', to_char(valida AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    );
    INSERT INTO vec_autorizacion.decision_autorizacion(
        decision_ref, concedida, codigo, principal_id, perfil_activo_ref,
        accion, recurso_ref, modulo_id, tipo_recurso,
        contexto_recurso_huella_sha256, finalidad, correlacion_ref,
        asignacion_ref, asignacion_huella_sha256, version_rol_ref,
        version_rol_huella_sha256, control_vigencia_version_rol_ref,
        control_vigencia_version_rol_revision,
        control_vigencia_version_rol_huella_sha256,
        revision_catalogo_politicas, catalogo_politicas_huella_sha256,
        politicas_evaluadas_manifesto, politicas_aplicadas_manifesto,
        emitida_en, valida_hasta, documento, registrada_en
    ) VALUES (
        p_decision_ref, true, 'concedida',
        'per_convocatorias_runtime_prueba_000001',
        'prf_convocatorias_runtime_prueba_000001', p_accion,
        p_recurso_ref, 'bolsa', p_tipo_recurso,
        encode(sha256(p_contexto), 'hex'), p_finalidad,
        'correlacion:convocatorias:runtime',
        'asignacion:convocatorias_runtime_prueba:v1', repeat('3',64),
        'rol:convocatorias_runtime_prueba:v1', repeat('1',64),
        'rol:convocatorias_runtime_prueba:v1', 1, repeat('2',64),
        control.revision, control.huella_sha256, '{}'::jsonb, '{}'::jsonb,
        emitida, valida, documento, clock_timestamp()
    );
    canonica := jsonb_build_object(
        'esquema',
            'vec.autorizacion.decision.reforzada.v1.autenticacion-actor',
        'decision_ref', documento -> 'decision_ref',
        'concedida', documento -> 'concedida',
        'codigo', documento -> 'codigo',
        'principal_id', documento -> 'principal_id',
        'perfil_activo_ref', documento -> 'perfil_activo_ref',
        'accion', documento -> 'accion',
        'recurso_ref', documento -> 'recurso_ref',
        'modulo_id', documento -> 'modulo_id',
        'tipo_recurso', documento -> 'tipo_recurso',
        'contexto_recurso_huella_sha256',
            documento -> 'contexto_recurso_huella_sha256',
        'finalidad', documento -> 'finalidad',
        'correlacion_ref', documento -> 'correlacion_ref',
        'vinculo_autenticacion_actor', vinculo,
        'asignacion_ref', documento -> 'asignacion_ref',
        'asignacion_huella_sha256',
            documento -> 'asignacion_huella_sha256',
        'version_rol_ref', documento -> 'version_rol_ref',
        'version_rol_huella_sha256', documento -> 'version_rol_huella_sha256',
        'control_vigencia_version_rol_ref',
            documento -> 'control_vigencia_version_rol_ref',
        'control_vigencia_version_rol_revision',
            documento -> 'control_vigencia_version_rol_revision',
        'control_vigencia_version_rol_huella_sha256',
            documento -> 'control_vigencia_version_rol_huella_sha256',
        'revision_catalogo_politicas',
            documento -> 'revision_catalogo_politicas',
        'catalogo_politicas_huella_sha256',
            documento -> 'catalogo_politicas_huella_sha256',
        'politicas_evaluadas', '[]'::jsonb,
        'politicas_aplicables', '[]'::jsonb,
        'garantia_minima', documento -> 'garantia_minima',
        'campos_permitidos', p_campos,
        'obligaciones', '[]'::jsonb,
        'emitida_en', documento -> 'emitida_en',
        'valida_hasta', documento -> 'valida_hasta'
    );
    canonica_bytes := convert_to(canonica::text, 'UTF8');
    huella := encode(sha256(canonica_bytes), 'hex');
    INSERT INTO public.fixture_decision_borrador_runtime VALUES (
        p_decision_ref,
        jsonb_build_object(
            'esquema_huella',
                'vec.autorizacion.decision.reforzada.v1.autenticacion-actor',
            'decision_ref', p_decision_ref,
            'huella_decision_sha256', huella,
            'verificada_en', to_char(
                date_trunc('microseconds', clock_timestamp())
                    AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
            'principal_ref', 'per_convocatorias_runtime_prueba_000001'
        ), canonica_bytes, p_contexto, huella,
        'atestacion:' || p_decision_ref, evidencia_huella
    );
END
$funcion$;

GRANT INSERT ON public.fixture_decision_borrador_runtime
    TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION public.crear_decision_borrador_runtime_prueba(
    text,text,text,text,text,jsonb,bytea
) TO vec_autorizacion_propietario;

SET LOCAL ROLE vec_autorizacion_propietario;
DO $decisiones$
DECLARE
    mutacion bytea;
    lectura bytea := convert_to(
        '{"ambitos":{"organizacion_ref":"org_0123456789abcdef"},"atributos":{}}',
        'UTF8'
    );
BEGIN
    SELECT contexto INTO STRICT mutacion
      FROM public.fixture_reserva_borrador_concurrente;
    PERFORM public.crear_decision_borrador_runtime_prueba(
        'decision-runtime-crear-valida',
        'bolsa.convocatoria.borrador.crear',
        'version_convocatoria_gobernada', 'convocatoria-concurrente#1',
        'gobierno_convocatorias',
        '["auditoria","evento_outbox","version_convocatoria"]'::jsonb,
        mutacion
    );
    PERFORM public.crear_decision_borrador_runtime_prueba(
        'decision-runtime-accion-invalida',
        'bolsa.convocatoria.borrador.actualizar',
        'version_convocatoria_gobernada', 'convocatoria-concurrente#1',
        'gobierno_convocatorias',
        '["auditoria","evento_outbox","version_convocatoria"]'::jsonb,
        mutacion
    );
    PERFORM public.crear_decision_borrador_runtime_prueba(
        'decision-runtime-recurso-invalido',
        'bolsa.convocatoria.borrador.crear',
        'version_convocatoria_gobernada', 'otra-convocatoria#1',
        'gobierno_convocatorias',
        '["auditoria","evento_outbox","version_convocatoria"]'::jsonb,
        mutacion
    );
    PERFORM public.crear_decision_borrador_runtime_prueba(
        'decision-runtime-campos-invalidos',
        'bolsa.convocatoria.borrador.crear',
        'version_convocatoria_gobernada', 'convocatoria-concurrente#1',
        'gobierno_convocatorias', '["version_convocatoria"]'::jsonb,
        mutacion
    );
    PERFORM public.crear_decision_borrador_runtime_prueba(
        'decision-runtime-listar-valida',
        'bolsa.convocatoria.borrador.listar',
        'coleccion_versiones_convocatoria_gobernada',
        'borradores:org_0123456789abcdef',
        'consulta_interna_convocatorias',
        '["version_convocatoria"]'::jsonb, lectura
    );
    PERFORM public.crear_decision_borrador_runtime_prueba(
        'decision-runtime-consultar-valida',
        'bolsa.convocatoria.borrador.consultar',
        'version_convocatoria_gobernada', 'convocatoria-inexistente#1',
        'consulta_interna_convocatorias',
        '["version_convocatoria"]'::jsonb, lectura
    );
    PERFORM public.crear_decision_borrador_runtime_prueba(
        'decision-runtime-listar-campos-invalidos',
        'bolsa.convocatoria.borrador.listar',
        'coleccion_versiones_convocatoria_gobernada',
        'borradores:org_0123456789abcdef',
        'consulta_interna_convocatorias', '["auditoria"]'::jsonb, lectura
    );
END
$decisiones$;

RESET ROLE;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
DO $atestaciones_lectura$
DECLARE
    fila record;
    ahora timestamptz(6) := date_trunc('microseconds', clock_timestamp());
    evidencia bytea := convert_to('{}', 'UTF8');
    sobre bytea;
BEGIN
    FOR fila IN
        SELECT * FROM public.fixture_decision_borrador_runtime
         WHERE decision_ref IN (
             'decision-runtime-listar-valida',
             'decision-runtime-consultar-valida'
         )
    LOOP
        sobre := convert_to('cose-prueba:' || fila.decision_ref, 'UTF8');
        INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_version(
            decision_ref, atestacion_ref, version, estado,
            huella_decision_sha256, evidencia_canonica,
            huella_evidencia_sha256, sobre_cose_sign1,
            huella_sobre_sha256, clave_id, revision_confianza,
            verificada_en, valida_desde, valida_hasta, registrada_en
        ) VALUES (
            fila.decision_ref, fila.atestacion_ref, 1, 'activa',
            fila.huella_decision_sha256, evidencia,
            fila.huella_atestacion_sha256, sobre,
            encode(sha256(sobre),'hex'), 'clave-pdp-runtime',
            'confianza-runtime', ahora, ahora - interval '1 minute',
            ahora + interval '2 minutes', ahora
        );
        INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_actual
        VALUES (fila.decision_ref, fila.atestacion_ref, 1, 'activa', ahora);
        INSERT INTO vec_bolsa_convocatorias.atestacion_pdp_borrador VALUES (
            fila.decision_ref, fila.atestacion_ref, 1, 'activa',
            fila.huella_decision_sha256, fila.huella_atestacion_sha256,
            'verificador:' || fila.decision_ref, ahora, ahora
        );
    END LOOP;
END
$atestaciones_lectura$;
RESET ROLE;

CREATE FUNCTION public.probar_proyector_borrador_runtime()
RETURNS void
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    f record;
    d record;
    invalida record;
    fila record;
    primaria jsonb;
BEGIN
    SELECT * INTO STRICT f
      FROM public.fixture_reserva_borrador_concurrente;
    SELECT * INTO STRICT d
      FROM public.fixture_decision_borrador_runtime
     WHERE decision_ref = 'decision-runtime-crear-valida';
    SELECT * INTO STRICT fila
      FROM vec_bolsa_convocatorias.consultar_identidades_borrador_v1(
          f.reserva -> 'identidades_consulta'
      );
    IF fila.estado <> 'reservado' OR fila.revision <> 1
       OR jsonb_typeof(fila.identidades_consultadas) <> 'array'
       OR jsonb_array_length(fila.identidades_consultadas) = 0
       OR EXISTS (
           SELECT 1
             FROM jsonb_array_elements(
                      fila.identidades_consultadas
                  ) AS r(identidad)
            WHERE NOT EXISTS (
                SELECT 1
                  FROM jsonb_array_elements(
                           f.reserva -> 'identidades_consulta'
                       ) AS c(identidad)
                 WHERE c.identidad IS NOT DISTINCT FROM r.identidad
            )
       )
       OR fila.identidad_primaria IS NULL
       OR (fila.identidades_consultadas::text ||
           fila.identidad_primaria::text) ~ '(principal|motivo)' THEN
        RAISE EXCEPTION 'wrapper consultar no alcanzo cuerpo: %', fila;
    END IF;
    primaria := fila.identidad_primaria;
    SELECT * INTO STRICT fila
      FROM vec_bolsa_convocatorias.reservar_decision_borrador_v1(
          f.reserva, d.prueba, f.material, f.version_canonica,
          d.decision_canonica, f.contexto
      );
    IF fila.estado <> 'en_curso' OR fila.revision <> 1
       OR jsonb_typeof(fila.identidades_consultadas) <> 'array'
       OR jsonb_array_length(fila.identidades_consultadas) = 0
       OR EXISTS (
           SELECT 1
             FROM jsonb_array_elements(
                      fila.identidades_consultadas
                  ) AS r(identidad)
            WHERE NOT EXISTS (
                SELECT 1
                  FROM jsonb_array_elements(
                           f.reserva -> 'identidades_consulta'
                       ) AS c(identidad)
                 WHERE c.identidad IS NOT DISTINCT FROM r.identidad
            )
       )
       OR fila.identidad_primaria IS DISTINCT FROM primaria THEN
        RAISE EXCEPTION 'wrapper reservar no alcanzo cuerpo: %', fila;
    END IF;
    BEGIN
        PERFORM *
          FROM vec_bolsa_convocatorias.reclamar_reserva_borrador_v1(
              1, 1, f.reserva, d.prueba, f.material, f.version_canonica,
              d.decision_canonica, f.contexto
          );
        RAISE EXCEPTION 'reclaim valido no alcanzo rechazo de estado del cuerpo';
    EXCEPTION WHEN serialization_failure THEN NULL;
    END;
    BEGIN
        PERFORM *
          FROM vec_bolsa_convocatorias.preparar_confirmacion_borrador_v1(
              jsonb_build_object('identidad', f.reserva -> 'identidad'),
              d.prueba, '{}'::jsonb, d.decision_canonica, f.contexto,
              f.material, f.version_canonica, convert_to('{}', 'UTF8'),
              decode(repeat('aa', 32), 'hex'),
              decode(repeat('bb', 12), 'hex'),
              decode(repeat('cc', 16), 'hex')
          );
        RAISE EXCEPTION 'fase A acepto evidencia KMS incompleta';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;

    SELECT * INTO STRICT invalida
      FROM public.fixture_decision_borrador_runtime
     WHERE decision_ref = 'decision-runtime-accion-invalida';
    BEGIN
        PERFORM * FROM vec_bolsa_convocatorias.reservar_decision_borrador_v1(
            f.reserva, invalida.prueba, f.material, f.version_canonica,
            invalida.decision_canonica, f.contexto
        );
        RAISE EXCEPTION 'reserva acepto decision con accion incorrecta';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
    SELECT * INTO STRICT invalida
      FROM public.fixture_decision_borrador_runtime
     WHERE decision_ref = 'decision-runtime-recurso-invalido';
    BEGIN
        PERFORM * FROM vec_bolsa_convocatorias.reclamar_reserva_borrador_v1(
            1, 1, f.reserva, invalida.prueba, f.material,
            f.version_canonica, invalida.decision_canonica, f.contexto
        );
        RAISE EXCEPTION 'reclaim acepto decision con recurso incorrecto';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
    SELECT * INTO STRICT invalida
      FROM public.fixture_decision_borrador_runtime
     WHERE decision_ref = 'decision-runtime-campos-invalidos';
    BEGIN
        PERFORM * FROM vec_bolsa_convocatorias.reservar_decision_borrador_v1(
            f.reserva, invalida.prueba, f.material, f.version_canonica,
            invalida.decision_canonica, f.contexto
        );
        RAISE EXCEPTION 'reserva acepto campos incorrectos';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
    SELECT * INTO STRICT fila
      FROM vec_bolsa_convocatorias.reconciliar_operacion_borrador_v1(
          f.reserva -> 'identidad', 'reservado', 1, 1,
          clock_timestamp()
      );
    IF fila.estado NOT IN ('reservado','no_aplicado') THEN
        RAISE EXCEPTION 'wrapper reconciliar no alcanzo cuerpo: %', fila;
    END IF;
END
$funcion$;

CREATE FUNCTION public.probar_ejecutor_borrador_runtime()
RETURNS void
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    d record;
    invalida record;
    lectura jsonb;
    contexto bytea := convert_to(
        '{"ambitos":{"organizacion_ref":"org_0123456789abcdef"},"atributos":{}}',
        'UTF8'
    );
    lista jsonb;
BEGIN
    SELECT * INTO STRICT d
      FROM public.fixture_decision_borrador_runtime
     WHERE decision_ref = 'decision-runtime-listar-valida';
    lectura := jsonb_build_object(
        'decision_ref', d.decision_ref,
        'huella_decision_sha256', d.huella_decision_sha256,
        'atestacion_ref', d.atestacion_ref, 'atestacion_version', 1,
        'estado_atestacion', 'activa',
        'huella_atestacion_sha256', d.huella_atestacion_sha256,
        'accion', 'bolsa.convocatoria.borrador.listar',
        'recurso_ref', 'borradores:org_0123456789abcdef',
        'organizacion_ref', 'org_0123456789abcdef',
        'unidad_gestion_ref', ''
    );
    lista := vec_bolsa_convocatorias.listar_borradores_v1(
        jsonb_build_object(
            'limite',10,'cursor','','texto','','categoria',''
        ), lectura, d.prueba, d.decision_canonica, contexto
    );
    IF lista ->> 'esquema' <> 'vec.bolsa.borradores.lista.v1' THEN
        RAISE EXCEPTION 'wrapper listar no alcanzo cuerpo: %', lista;
    END IF;

    SELECT * INTO STRICT d
      FROM public.fixture_decision_borrador_runtime
     WHERE decision_ref = 'decision-runtime-consultar-valida';
    lectura := jsonb_build_object(
        'decision_ref', d.decision_ref,
        'huella_decision_sha256', d.huella_decision_sha256,
        'atestacion_ref', d.atestacion_ref, 'atestacion_version', 1,
        'estado_atestacion', 'activa',
        'huella_atestacion_sha256', d.huella_atestacion_sha256,
        'accion', 'bolsa.convocatoria.borrador.consultar',
        'recurso_ref', 'convocatoria-inexistente#1',
        'organizacion_ref', 'org_0123456789abcdef',
        'unidad_gestion_ref', ''
    );
    BEGIN
        PERFORM * FROM vec_bolsa_convocatorias.obtener_borrador_v1(
            'convocatoria-inexistente#1', lectura, d.prueba,
            d.decision_canonica, contexto
        );
        RAISE EXCEPTION 'obtener valido no alcanzo ausencia del cuerpo';
    EXCEPTION WHEN no_data_found THEN NULL;
    END;

    SELECT * INTO STRICT invalida
      FROM public.fixture_decision_borrador_runtime
     WHERE decision_ref = 'decision-runtime-listar-campos-invalidos';
    BEGIN
        PERFORM vec_bolsa_convocatorias.listar_borradores_v1(
            jsonb_build_object(
                'limite',10,'cursor','','texto','','categoria',''
            ), lectura || jsonb_build_object(
                'accion','bolsa.convocatoria.borrador.listar',
                'recurso_ref','borradores:org_0123456789abcdef'
            ), invalida.prueba, invalida.decision_canonica, contexto
        );
        RAISE EXCEPTION 'listado acepto campos incorrectos';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
    SELECT * INTO STRICT invalida
      FROM public.fixture_decision_borrador_runtime
     WHERE decision_ref = 'decision-runtime-listar-valida';
    BEGIN
        PERFORM * FROM vec_bolsa_convocatorias.obtener_borrador_v1(
            'convocatoria-inexistente#1', lectura, invalida.prueba,
            invalida.decision_canonica, contexto
        );
        RAISE EXCEPTION 'detalle acepto accion/recurso de listado';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
END
$funcion$;

GRANT EXECUTE ON FUNCTION public.probar_proyector_borrador_runtime()
    TO vec_convocatorias_proyector_prueba;
GRANT EXECUTE ON FUNCTION public.probar_ejecutor_borrador_runtime()
    TO vec_convocatorias_ejecutor_prueba;

RESET ROLE;
SET SESSION AUTHORIZATION vec_convocatorias_proyector_prueba;
SELECT public.probar_proyector_borrador_runtime();
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_convocatorias_ejecutor_prueba;
SELECT public.probar_ejecutor_borrador_runtime();
RESET SESSION AUTHORIZATION;

ROLLBACK;
