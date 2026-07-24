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
    huella_atestacion_sha256 text NOT NULL,
    solicitud_canonica bytea,
    huella_solicitud_sha256 text
);
GRANT USAGE ON SCHEMA public
    TO vec_autorizacion_propietario,
       vec_convocatorias_ejecutor_prueba,
       vec_convocatorias_proyector_prueba;
GRANT SELECT ON public.fixture_decision_borrador_runtime
    TO vec_autorizacion_propietario,
       vec_bolsa_convocatorias_propietario,
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
    p_contexto bytea, p_duracion interval DEFAULT interval '2 minutes'
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
    IF p_duracion < interval '1 second'
       OR p_duracion > interval '2 minutes' THEN
        RAISE EXCEPTION 'vigencia de fixture V2 fuera de presupuesto';
    END IF;
    valida := ahora + p_duracion;
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
        'atestacion:' || p_decision_ref, evidencia_huella, NULL, NULL
    );
END
$funcion$;

GRANT INSERT ON public.fixture_decision_borrador_runtime
    TO vec_autorizacion_propietario;
GRANT UPDATE ON public.fixture_decision_borrador_runtime
    TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION public.crear_decision_borrador_runtime_prueba(
    text,text,text,text,text,jsonb,bytea,interval
) TO vec_autorizacion_propietario;

-- Replica el DTO cerrado de Go exclusivamente para la prueba cruzada. La
-- decision V2 compromete el SHA-256 de esta solicitud efectiva, no una
-- constante elegida por el fixture.
CREATE FUNCTION public.solicitud_borrador_runtime_v2_canonica(
    p_documento jsonb, p_contexto bytea, p_motivo jsonb
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    vinculo jsonb := p_documento -> 'vinculo_autenticacion_actor';
    recurso jsonb;
    referencia_motivo jsonb := p_motivo -> 'referencia';
    ambitos text;
    atributos text;
    resultado text;
BEGIN
    recurso := convert_from(p_contexto, 'UTF8')::jsonb;
    SELECT COALESCE(string_agg(
               '{"clave":' || to_jsonb(clave)::text ||
               ',"valor":' || to_jsonb(valor)::text || '}',
               ',' ORDER BY clave COLLATE "C"
           ), '')
      INTO ambitos
      FROM jsonb_each_text(recurso -> 'ambitos') AS e(clave, valor);
    SELECT COALESCE(string_agg(
               '{"clave":' || to_jsonb(clave)::text ||
               ',"valor":' || to_jsonb(valor)::text || '}',
               ',' ORDER BY clave COLLATE "C"
           ), '')
      INTO atributos
      FROM jsonb_each_text(recurso -> 'atributos') AS e(clave, valor);
    resultado :=
        '{"esquema":"vec.autorizacion.solicitud.v2.efectiva-minimizada"' ||
        ',"principal":{"id":' ||
            to_jsonb(p_documento ->> 'principal_id')::text || '}' ||
        ',"perfil_activo_ref":' ||
            to_jsonb(p_documento ->> 'perfil_activo_ref')::text ||
        ',"contexto_actor":{"referencia":' ||
            to_jsonb(vinculo ->> 'contexto_actor_ref')::text ||
            ',"version":' || (vinculo ->> 'contexto_actor_version') ||
            ',"huella_sha256":' ||
            to_jsonb(vinculo ->> 'contexto_actor_huella_sha256')::text || '}' ||
        ',"vinculo_autenticacion_actor":{"bloque_version":' ||
            (vinculo ->> 'bloque_version') ||
            ',"autenticacion_ref":' || to_jsonb(vinculo ->> 'autenticacion_ref')::text ||
            ',"autenticacion_huella_sha256":' || to_jsonb(vinculo ->> 'autenticacion_huella_sha256')::text ||
            ',"asercion_ref":' || to_jsonb(vinculo ->> 'asercion_ref')::text ||
            ',"sesion_ref":' || to_jsonb(vinculo ->> 'sesion_ref')::text ||
            ',"control_sesion_ref":' || to_jsonb(vinculo ->> 'control_sesion_ref')::text ||
            ',"control_sesion_revision":' || (vinculo ->> 'control_sesion_revision') ||
            ',"control_sesion_huella_sha256":' || to_jsonb(vinculo ->> 'control_sesion_huella_sha256')::text ||
            ',"cuenta_ref":' || to_jsonb(vinculo ->> 'cuenta_ref')::text ||
            ',"cuenta_ordinaria_ref":' || to_jsonb(vinculo ->> 'cuenta_ordinaria_ref')::text ||
            ',"principal_id":' || to_jsonb(vinculo ->> 'principal_id')::text ||
            ',"perfil_activo_ref":' || to_jsonb(vinculo ->> 'perfil_activo_ref')::text ||
            ',"cuenta_privilegiada":' || (vinculo -> 'cuenta_privilegiada')::text ||
            ',"superficie":' || to_jsonb(vinculo ->> 'superficie')::text ||
            ',"metodo_observado":' || to_jsonb(vinculo ->> 'metodo_observado')::text ||
            ',"garantia_observada":' || to_jsonb(vinculo ->> 'garantia_observada')::text ||
            ',"politica_garantia_ref":' || to_jsonb(vinculo ->> 'politica_garantia_ref')::text ||
            ',"politica_garantia_huella_sha256":' || to_jsonb(vinculo ->> 'politica_garantia_huella_sha256')::text ||
            ',"autenticacion_verificada_en":' || to_jsonb(vinculo ->> 'autenticacion_verificada_en')::text ||
            ',"sesion_emitida_en":' || to_jsonb(vinculo ->> 'sesion_emitida_en')::text ||
            ',"sesion_valida_hasta":' || to_jsonb(vinculo ->> 'sesion_valida_hasta')::text ||
            ',"sesion_revalidada_en":' || to_jsonb(vinculo ->> 'sesion_revalidada_en')::text ||
            ',"contexto_actor_ref":' || to_jsonb(vinculo ->> 'contexto_actor_ref')::text ||
            ',"contexto_actor_version":' || (vinculo ->> 'contexto_actor_version') ||
            ',"contexto_actor_huella_sha256":' || to_jsonb(vinculo ->> 'contexto_actor_huella_sha256')::text || '}' ||
        ',"accion":' || to_jsonb(p_documento ->> 'accion')::text ||
        ',"recurso":{"referencia":' || to_jsonb(p_documento ->> 'recurso_ref')::text ||
            ',"modulo_id":' || to_jsonb(p_documento ->> 'modulo_id')::text ||
            ',"tipo":' || to_jsonb(p_documento ->> 'tipo_recurso')::text ||
            ',"ambitos":[' || ambitos || '],"atributos":[' || atributos || ']}' ||
        ',"finalidad":' || to_jsonb(p_documento ->> 'finalidad')::text ||
        ',"correlacion_ref":' || to_jsonb(p_documento ->> 'correlacion_ref')::text ||
        ',"referencia_motivo":{"catalogo_id":' || to_jsonb(referencia_motivo ->> 'catalogo_id')::text ||
            ',"catalogo_version":' || (referencia_motivo ->> 'catalogo_version') ||
            ',"catalogo_huella_sha256":' || to_jsonb(referencia_motivo ->> 'catalogo_huella_sha256')::text ||
            ',"entrada_clave":' || to_jsonb(referencia_motivo ->> 'entrada_clave')::text || '}}';
    RETURN convert_to(resultado, 'UTF8');
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END
$funcion$;

CREATE FUNCTION public.convertir_decision_borrador_runtime_v2(
    p_decision_ref text
)
RETURNS void
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    fila record;
    motivo jsonb := jsonb_build_object(
        'esquema',
            'vec.autorizacion.motivo.v2.referencia-opaca-catalogada',
        'referencia', jsonb_build_object(
            'catalogo_id', 'motivos_convocatorias_runtime',
            'catalogo_version', 1,
            'catalogo_huella_sha256', repeat('9', 64),
            'entrada_clave',
                'motivo_0123456789abcdef0123456789abcdef'
        )
    );
    motivo_bytes bytea;
    documento jsonb;
    canonica bytea;
    huella text;
    v_solicitud_canonica bytea;
    v_huella_solicitud text;
BEGIN
    SELECT * INTO STRICT fila
      FROM public.fixture_decision_borrador_runtime
     WHERE decision_ref = p_decision_ref;
    motivo_bytes := convert_to(motivo::text, 'UTF8');
    documento := (
        convert_from(fila.decision_canonica, 'UTF8')::jsonb - 'esquema'
    ) || jsonb_build_object(
        'esquema',
            'vec.autorizacion.decision.reforzada.v2.solicitud-ligada',
        'esquema_huella_solicitud',
            'vec.autorizacion.solicitud.v2.efectiva-minimizada',
        'esquema_huella_motivo',
            'vec.autorizacion.motivo.v2.referencia-opaca-catalogada',
        'motivo_huella_sha256', encode(sha256(motivo_bytes), 'hex'),
        'correlacion_ref',
            'correlacion_0123456789abcdef0123456789abcdef'
    );
    v_solicitud_canonica := public.solicitud_borrador_runtime_v2_canonica(
        documento, fila.contexto_canonico, motivo
    );
    v_huella_solicitud := encode(sha256(v_solicitud_canonica), 'hex');
    documento := documento || jsonb_build_object(
        'solicitud_huella_sha256', v_huella_solicitud
    );
    canonica := convert_to(documento::text, 'UTF8');
    IF vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(
        canonica, motivo_bytes
    ) IS NOT TRUE THEN
        RAISE EXCEPTION 'no se registro decision V2 %', p_decision_ref;
    END IF;
    huella := encode(sha256(canonica), 'hex');
    UPDATE public.fixture_decision_borrador_runtime
       SET prueba = jsonb_build_object(
               'esquema_huella',
                   'vec.autorizacion.decision.reforzada.v2.solicitud-ligada',
               'decision_ref', p_decision_ref,
               'huella_decision_sha256', huella,
               'verificada_en', to_char(
                   date_trunc('microseconds', clock_timestamp())
                       AT TIME ZONE 'UTC',
                   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
               ),
               'principal_ref',
                   'per_convocatorias_runtime_prueba_000001'
           ),
           decision_canonica = canonica,
           huella_decision_sha256 = huella,
           solicitud_canonica = v_solicitud_canonica,
           huella_solicitud_sha256 = v_huella_solicitud
     WHERE decision_ref = p_decision_ref;
END
$funcion$;

GRANT EXECUTE ON FUNCTION public.convertir_decision_borrador_runtime_v2(text)
    TO vec_autorizacion_propietario;

SET LOCAL ROLE vec_autorizacion_propietario;
DO $decisiones$
DECLARE
    mutacion bytea;
    lectura bytea := convert_to(
        '{"ambitos":{"organizacion_ref":"org_0123456789abcdef"},"atributos":{}}',
        'UTF8'
    );
BEGIN
    IF vec_autorizacion.publicar_motivos_autorizacion_v2(
        'evento_0123456789abcdef0123456789abcdef', 1, repeat('7', 64),
        'motivos_convocatorias_runtime', 1, repeat('9', 64),
        date_trunc('microseconds', clock_timestamp()) - interval '1 minute',
        jsonb_build_array(jsonb_build_object(
            'clave', 'motivo_0123456789abcdef0123456789abcdef',
            'vigente_desde', to_char(
                (date_trunc('microseconds', clock_timestamp())
                    - interval '1 minute') AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
            'vigente_hasta', NULL
        ))
    ) IS NOT TRUE THEN
        RAISE EXCEPTION 'no se publico el motivo V2 de la prueba';
    END IF;
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
        'version_convocatoria_gobernada', 'convocatoria-lectura-kms#1',
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
    PERFORM public.crear_decision_borrador_runtime_prueba(
        'decision-runtime-listar-v1-rechazada',
        'bolsa.convocatoria.borrador.listar',
        'coleccion_versiones_convocatoria_gobernada',
        'borradores:org_0123456789abcdef',
        'consulta_interna_convocatorias',
        '["version_convocatoria"]'::jsonb, lectura
    );
    PERFORM public.convertir_decision_borrador_runtime_v2(
        'decision-runtime-listar-valida'
    );
    PERFORM public.convertir_decision_borrador_runtime_v2(
        'decision-runtime-consultar-valida'
    );
    PERFORM public.convertir_decision_borrador_runtime_v2(
        'decision-runtime-listar-campos-invalidos'
    );
    IF EXISTS (
        SELECT 1
          FROM public.fixture_decision_borrador_runtime AS fixture
          JOIN vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
               AS registro USING (decision_ref)
         WHERE fixture.decision_ref IN (
                   'decision-runtime-listar-valida',
                   'decision-runtime-consultar-valida',
                   'decision-runtime-listar-campos-invalidos'
               )
           AND (
               fixture.solicitud_canonica IS NULL
               OR fixture.huella_solicitud_sha256 IS NULL
               OR fixture.huella_solicitud_sha256 = repeat('6', 64)
               OR encode(sha256(fixture.solicitud_canonica), 'hex') IS
                  DISTINCT FROM fixture.huella_solicitud_sha256
               OR registro.solicitud_huella_sha256 IS DISTINCT FROM
                  fixture.huella_solicitud_sha256
               OR registro.documento_v2 ->> 'solicitud_huella_sha256' IS
                  DISTINCT FROM fixture.huella_solicitud_sha256
           )
    ) THEN
        RAISE EXCEPTION
            'fixture V2 no comprometio su solicitud efectiva canonica';
    END IF;
END
$decisiones$;

RESET ROLE;
COMMIT;

BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
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
             'decision-runtime-consultar-valida',
             'decision-runtime-listar-v1-rechazada'
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

-- Paquete positivo acreditado sobre las tablas instaladas por 000004. La
-- huella estructurada del sobre es distinta de la huella del ciphertext; el
-- lector 000005 debe cotejar ambas con su semantica propia.
DO $paquete_000004$
DECLARE
    ahora timestamptz(6) := date_trunc('microseconds', clock_timestamp());
    aad bytea := convert_to('{"esquema":"aad-prueba-000004"}', 'UTF8');
    envuelto bytea := decode(repeat('ef', 32), 'hex');
    nonce_prueba bytea := decode(repeat('ab', 12), 'hex');
    cifrado bytea := decode(repeat('cd', 32), 'hex');
    perfil jsonb := jsonb_build_object(
        'referencia', 'perfil:cifrado:borradores:runtime',
        'version', 1, 'huella_contenido_sha256', repeat('a', 64),
        'algoritmo_aead', 'A256GCM',
        'algoritmo_envoltura_clave', 'A256KW'
    );
    procedencia jsonb := jsonb_build_object(
        'esquema', 'vec.acto.procedencia.v1',
        'perfil_ejecucion', 'pruebas', 'autoridad', 'autoritativo',
        'proveedor_ref', 'proveedor-pruebas-000004',
        'migrable_produccion', true
    );
    envoltura jsonb;
    sobre jsonb;
    huella_envoltura text;
    huella_sobre text;
    firma jsonb;
    atestacion jsonb;
    auditoria bytea := convert_to('{"evento":"auditoria-kms"}', 'UTF8');
    evento bytea := convert_to('{"evento":"outbox-kms"}', 'UTF8');
    cuerpo bytea := convert_to('{"recibo":"cuerpo-kms"}', 'UTF8');
    acreditacion bytea := convert_to('{"acreditacion":"kms"}', 'UTF8');
    recibo bytea := convert_to('{"recibo":"kms-final"}', 'UTF8');
    transaccion text := 'transaccion:lectura-kms:000004';
BEGIN
    envoltura := jsonb_build_object(
        'esquema', 'bolsa.convocatoria.borrador.clave-envuelta.v1',
        'clave_maestra_ref', 'clave:kms:borradores:runtime',
        'version_clave', 1, 'huella_aad', encode(sha256(aad), 'hex')
    );
    huella_envoltura :=
        vec_bolsa_convocatorias.huella_envoltura_clave_borrador_v1(
            perfil, envoltura, envuelto
        );
    envoltura := envoltura || jsonb_build_object(
        'huella_envoltura_sha256', huella_envoltura
    );
    sobre := jsonb_build_object(
        'esquema', 'bolsa.convocatoria.borrador.sobre-aead.v1',
        'huella_aad', encode(sha256(aad), 'hex')
    );
    huella_sobre := vec_bolsa_convocatorias.huella_sobre_aead_borrador_v1(
        perfil, sobre, nonce_prueba, cifrado
    );
    firma := jsonb_build_object(
        'algoritmo_firma', 'Ed25519',
        'verificador_ref', 'verificador:kms:runtime',
        'huella_clave_publica_sha256', repeat('b', 64),
        'huella_preimagen_sha256', repeat('c', 64),
        'firma_base64url_sin_relleno', rtrim(translate(replace(
            encode(decode(repeat('01', 64), 'hex'), 'base64'), E'\n', ''
        ), '+/', '-_'), '=')
    );
    atestacion := jsonb_build_object(
        'esquema', 'bolsa.convocatoria.borrador.atestacion-kms.v1',
        'atestacion_ref', 'atestacion:kms:runtime:000004',
        'version_atestacion', 1, 'estado', 'vigente',
        'perfil', perfil,
        'clave_maestra_ref', 'clave:kms:borradores:runtime',
        'version_clave', 1, 'huella_aad', encode(sha256(aad), 'hex'),
        'huella_envoltura_sha256', huella_envoltura,
        'huella_sobre_sha256', huella_sobre,
        'verificador_ref', 'verificador:kms:runtime',
        'procedencia', procedencia, 'firma', firma,
        'emitida_en', to_char((ahora - interval '1 minute') AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'valida_hasta', to_char((ahora + interval '5 minutes') AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    );

    INSERT INTO vec_bolsa_convocatorias.borrador_convocatoria_version VALUES (
        'convocatoria-lectura-kms', 1, 'convocatoria-lectura-kms#1', 1,
        repeat('8', 64), cifrado, encode(sha256(cifrado), 'hex'),
        'A256GCM', 'clave:kms:borradores:runtime', 1, nonce_prueba,
        substring(cifrado FROM 17 FOR 16), 'atestacion:kms:runtime:000004',
        repeat('c', 64), 'v1', 'kms-lectura-001',
        'Borrador KMS de integracion', 'bolsa', ARRAY['auxiliar'],
        'expediente:kms:runtime', 'org_0123456789abcdef', NULL,
        1, 0, 1, 0, ahora, ahora, ahora
    );
    INSERT INTO vec_bolsa_convocatorias.cifrado_kms_borrador VALUES (
        'convocatoria-lectura-kms', 1, 1, transaccion,
        perfil ->> 'referencia', 1, perfil ->> 'huella_contenido_sha256',
        'A256GCM', 'A256KW', aad, encode(sha256(aad), 'hex'), envuelto,
        huella_envoltura, nonce_prueba, cifrado,
        encode(sha256(cifrado), 'hex'), huella_sobre,
        '{}'::jsonb, '{}'::jsonb, atestacion, procedencia,
        'desarrollo:t20', 1000, 500, ahora
    );
    INSERT INTO vec_bolsa_convocatorias.auditoria_borrador VALUES (
        'auditoria:lectura-kms:000004', 1,
        'decision-runtime-consultar-valida', transaccion, auditoria,
        repeat('0', 64), encode(sha256(auditoria), 'hex'), ahora
    );
    INSERT INTO vec_bolsa_convocatorias.outbox_borrador VALUES (
        'evento:lectura-kms:000004', 1, 'borrador_creado', transaccion,
        'convocatoria-lectura-kms', 1, 1,
        'auditoria:lectura-kms:000004', evento,
        encode(sha256(evento), 'hex'), ahora
    );
    INSERT INTO vec_bolsa_convocatorias.acreditacion_kms_borrador VALUES (
        'acreditacion:lectura-kms:000004', 'recibo:lectura-kms:000004',
        transaccion, 'convocatoria-lectura-kms', 1, 1,
        'auditoria:lectura-kms:000004', 'evento:lectura-kms:000004',
        cuerpo, encode(sha256(cuerpo), 'hex'), '{}'::jsonb,
        acreditacion, encode(sha256(acreditacion), 'hex'),
        recibo, encode(sha256(recibo), 'hex'), procedencia, ahora
    );
    INSERT INTO vec_bolsa_convocatorias.borrador_convocatoria_actual VALUES (
        'convocatoria-lectura-kms', 1, 1, repeat('8', 64), ahora
    );
END
$paquete_000004$;
RESET ROLE;
COMMIT;

BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

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
          FROM vec_bolsa_convocatorias.preparar_confirmacion_borrador_v2(
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

-- Puente exclusivo de prueba: permite que el LOGIN minimo ejecute el corpus
-- positivo contra el mismo predicado privado sin consumir otra decision de
-- lectura. Se elimina antes del down y nunca forma parte de una migracion.
CREATE FUNCTION public.texto_selector_borrador_runtime_valido(p_texto text)
RETURNS boolean
LANGUAGE sql
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT vec_bolsa_convocatorias.texto_selector_borradores_valido_v1(
        p_texto
    )
$funcion$;
ALTER FUNCTION public.texto_selector_borrador_runtime_valido(text)
    OWNER TO vec_bolsa_convocatorias_propietario;
REVOKE ALL ON FUNCTION public.texto_selector_borrador_runtime_valido(text)
    FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.texto_selector_borrador_runtime_valido(text)
    TO vec_convocatorias_ejecutor_prueba;

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
    selector_invalido jsonb;
    paquete record;
    codigo integer;
    texto_valido text := 'Título ágil';
    texto_adversario text;
    total_incompatibles integer := 0;
BEGIN
    IF unicode_version() <> '16.0' THEN
        RAISE EXCEPTION 'runner requiere PostgreSQL Unicode 16, obtuvo %',
            unicode_version();
    END IF;
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
    FOREACH selector_invalido IN ARRAY ARRAY[
        jsonb_build_object(
            'limite',10,'cursor','','texto',U&'E\0301','categoria',''
        ),
        jsonb_build_object(
            'limite',10,'cursor','','texto',repeat('a',181),'categoria',''
        ),
        jsonb_build_object(
            'limite',10,'cursor','','texto','a'||chr(1),'categoria',''
        ),
        jsonb_build_object(
            'limite',10,'cursor','','texto','a'||chr(8203),'categoria',''
        ),
        jsonb_build_object(
            'limite',10,'cursor','','texto',chr(65533),'categoria',''
        ),
        jsonb_build_object(
            'limite',10,'cursor','','texto',chr(160)||'a','categoria',''
        ),
        jsonb_build_object(
            'limite',10,'cursor','','texto','','categoria','Auxiliar'
        ),
        jsonb_build_object(
            'limite','10','cursor','','texto','','categoria',''
        ),
        jsonb_build_object(
            'limite',10,'cursor','','texto',
            'x'||chr(6863)||chr(6877),'categoria',''
        ),
        jsonb_build_object(
            'limite',10,'cursor','','texto',
            'x'||chr(6877)||chr(6863),'categoria',''
        ),
        jsonb_build_object(
            'limite',10,'cursor','','texto',
            'x'||chr(6891)||chr(769),'categoria',''
        ),
        jsonb_build_object(
            'limite',10,'cursor','','texto',
            'x'||chr(69370)||chr(124643),'categoria',''
        )
    ] LOOP
        BEGIN
            PERFORM vec_bolsa_convocatorias.listar_borradores_v1(
                selector_invalido, lectura, d.prueba,
                d.decision_canonica, contexto
            );
            RAISE EXCEPTION 'rol runtime acepto selector no canonico: %',
                selector_invalido;
        EXCEPTION WHEN invalid_parameter_value THEN NULL;
        END;
    END LOOP;

    -- Las 16 secuencias que empezaron a componer en Unicode 16. En Unicode
    -- 15 parecian NFC aunque sus escalares aislados fueran inertes.
    FOREACH texto_adversario IN ARRAY ARRAY[
        chr(67026)||chr(775),
        chr(67034)||chr(775),
        chr(70530)||chr(70601),
        chr(70532)||chr(70587),
        chr(70539)||chr(70594),
        chr(70544)||chr(70601),
        chr(70594)||chr(70594),
        chr(70594)||chr(70584),
        chr(70594)||chr(70601),
        chr(90398)||chr(90398),
        chr(90398)||chr(90409),
        chr(90398)||chr(90399),
        chr(90409)||chr(90399),
        chr(90398)||chr(90400),
        chr(93543)||chr(93543),
        chr(93539)||chr(93543)
    ] LOOP
        BEGIN
            PERFORM vec_bolsa_convocatorias.listar_borradores_v1(
                jsonb_build_object(
                    'limite',10,'cursor','','texto',texto_adversario,
                    'categoria',''
                ), lectura, d.prueba, d.decision_canonica, contexto
            );
            RAISE EXCEPTION
                'rol runtime acepto secuencia NFC dependiente de version: %',
                texto_adversario;
        EXCEPTION WHEN invalid_parameter_value THEN NULL;
        END;
    END LOOP;

    -- Perfil comun exacto para x/text Unicode 15/17 y PostgreSQL Unicode 16.
    -- Cada escalar de cada rango atraviesa el wrapper como el LOGIN runtime;
    -- el rechazo sucede antes de consultar/consumir la decision PDP.
    FOR codigo IN
        SELECT serie.valor
          FROM (VALUES
              (2199,2199),
              (6863,6877),(6880,6891),
              (67017,67017),(67026,67026),(67034,67034),(67044,67044),
              (68969,68973),(69370,69371),
              (70530,70533),(70539,70539),(70542,70542),
              (70544,70545),(70584,70584),(70587,70587),
              (70594,70594),(70597,70597),(70599,70601),
              (70606,70608),(90398,90409),(90415,90415),
              (93539,93539),(93543,93546),
              (124398,124399),(124643,124643),(124646,124646),
              (124654,124655),(124661,124661)
          ) AS limite(desde, hasta)
         CROSS JOIN LATERAL generate_series(
             limite.desde, limite.hasta
         ) AS serie(valor)
    LOOP
        total_incompatibles := total_incompatibles + 1;
        BEGIN
            PERFORM vec_bolsa_convocatorias.listar_borradores_v1(
                jsonb_build_object(
                    'limite',10,'cursor','','texto',chr(codigo),
                    'categoria',''
                ), lectura, d.prueba, d.decision_canonica, contexto
            );
            RAISE EXCEPTION
                'rol runtime acepto U+% fuera del perfil Unicode comun',
                upper(to_hex(codigo));
        EXCEPTION WHEN invalid_parameter_value THEN NULL;
        END;
    END LOOP;
    IF total_incompatibles <> 82 THEN
        RAISE EXCEPTION 'corpus NFC comun incompleto: %',
            total_incompatibles;
    END IF;

    -- Vecinos exactos de todos los rangos posteriores a Unicode 16. Deben
    -- seguir permitidos; no se bloquean bloques completos por aproximacion.
    FOREACH codigo IN ARRAY ARRAY[
        6862,6878,6879,6892,69369,69372,
        124642,124644,124645,124647,124653,124656,124660,124662
    ] LOOP
        -- La x separa segmentos de combinacion. Todos los vecinos atraviesan
        -- juntos la unica lectura consumible de esta decision de prueba.
        texto_valido := texto_valido || 'x' || chr(codigo);
    END LOOP;

    -- Vecinos de los 21 intervalos del salto NFC 15->16 y de todos los
    -- intervalos 16->17. Cada uno se prueba por separado para localizar una
    -- ampliacion accidental del filtro.
    FOREACH codigo IN ARRAY ARRAY[
        2198,2200,67016,67018,67025,67027,67033,67035,67043,67045,
        68968,68974,70529,70534,70538,70540,70541,70543,70546,
        70583,70585,70586,70588,70593,70595,70596,70598,70602,
        70605,70609,90397,90410,90414,90416,93538,93540,93542,
        93547,124397,124400,
        6862,6878,6879,6892,69369,69372,124642,124644,124645,
        124647,124653,124656,124660,124662
    ] LOOP
        IF public.texto_selector_borrador_runtime_valido(chr(codigo))
           IS NOT TRUE THEN
            RAISE EXCEPTION 'rechazo vecino NFC estable: U+%',
                upper(to_hex(codigo));
        END IF;
    END LOOP;

    -- U+A7F1 y U+1CCD6..U+1CCF9 cambian solo NFKC. El selector promete NFC
    -- y debe conservarlos; rechazarlos seria una restriccion funcional falsa.
    IF public.texto_selector_borrador_runtime_valido(chr(42993))
       IS NOT TRUE THEN
        RAISE EXCEPTION 'rechazo U+A7F1 con cambio solo NFKC';
    END IF;
    FOR codigo IN 117974..118009 LOOP
        IF public.texto_selector_borrador_runtime_valido(chr(codigo))
           IS NOT TRUE THEN
            RAISE EXCEPTION 'rechazo cambio solo NFKC: U+%',
                upper(to_hex(codigo));
        END IF;
    END LOOP;

    lista := vec_bolsa_convocatorias.listar_borradores_v1(
        jsonb_build_object(
            'limite',10,'cursor','','texto',texto_valido,'categoria',''
        ), lectura, d.prueba, d.decision_canonica, contexto
    );
    IF lista ->> 'esquema' <> 'vec.bolsa.borradores.lista.v1' THEN
        RAISE EXCEPTION 'wrapper listar no alcanzo cuerpo: %', lista;
    END IF;

    BEGIN
        PERFORM vec_bolsa_convocatorias.listar_borradores_v1(
            jsonb_build_object(
                'limite',10,'cursor','cursor-libre-no-opaco',
                'texto','','categoria',''
            ), lectura, d.prueba, d.decision_canonica, contexto
        );
        RAISE EXCEPTION 'listado acepto cursor no emitido por PostgreSQL';
    EXCEPTION WHEN invalid_parameter_value THEN NULL;
    END;

    BEGIN
        PERFORM vec_bolsa_convocatorias.listar_borradores_v1(
            jsonb_build_object(
                'limite',10,'cursor','','texto','','categoria',''
            ), lectura, d.prueba || jsonb_build_object(
                'huella_decision_sha256', repeat('0', 64)
            ), d.decision_canonica, contexto
        );
        RAISE EXCEPTION 'listado acepto prueba V2 con huella alterada';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
    BEGIN
        PERFORM vec_bolsa_convocatorias.listar_borradores_v1(
            jsonb_build_object(
                'limite',10,'cursor','','texto','','categoria',''
            ), lectura, d.prueba, d.decision_canonica,
            convert_to(
                '{"ambitos":{"organizacion_ref":"org_fedcba9876543210"},"atributos":{}}',
                'UTF8'
            )
        );
        RAISE EXCEPTION 'listado acepto contexto canonico alterado';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
    SELECT * INTO STRICT invalida
      FROM public.fixture_decision_borrador_runtime
     WHERE decision_ref = 'decision-runtime-listar-v1-rechazada';
    BEGIN
        PERFORM vec_bolsa_convocatorias.listar_borradores_v1(
            jsonb_build_object(
                'limite',10,'cursor','','texto','','categoria',''
            ), jsonb_build_object(
                'decision_ref', invalida.decision_ref,
                'huella_decision_sha256',
                    invalida.huella_decision_sha256,
                'atestacion_ref', invalida.atestacion_ref,
                'atestacion_version', 1,
                'estado_atestacion', 'activa',
                'huella_atestacion_sha256',
                    invalida.huella_atestacion_sha256,
                'accion', 'bolsa.convocatoria.borrador.listar',
                'recurso_ref', 'borradores:org_0123456789abcdef',
                'organizacion_ref', 'org_0123456789abcdef',
                'unidad_gestion_ref', ''
            ), invalida.prueba, invalida.decision_canonica, contexto
        );
        RAISE EXCEPTION 'listado acepto decision historica V1';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;

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
        'recurso_ref', 'convocatoria-lectura-kms#1',
        'organizacion_ref', 'org_0123456789abcdef',
        'unidad_gestion_ref', ''
    );
    SELECT * INTO STRICT paquete
      FROM vec_bolsa_convocatorias.obtener_borrador_v1(
          'convocatoria-lectura-kms#1', lectura, d.prueba,
          d.decision_canonica, contexto
      );
    IF paquete.metadatos #>> '{referencia_estado,referencia}' <>
          'convocatoria-lectura-kms#1'
       OR paquete.metadatos #>> '{referencia_estado,revision}' <> '1'
       OR paquete.metadatos #>> '{referencia_estado,huella_estado_sha256}' <>
          repeat('8', 64)
       OR paquete.aad_canonica IS DISTINCT FROM
          convert_to('{"esquema":"aad-prueba-000004"}', 'UTF8')
       OR paquete.huella_aad_sha256 IS DISTINCT FROM encode(sha256(
              convert_to('{"esquema":"aad-prueba-000004"}', 'UTF8')
          ), 'hex')
       OR paquete.perfil ->> 'referencia' <>
          'perfil:cifrado:borradores:runtime'
       OR paquete.esquema_envoltura <>
          'bolsa.convocatoria.borrador.clave-envuelta.v1'
       OR paquete.clave_maestra_ref <> 'clave:kms:borradores:runtime'
       OR paquete.version_clave <> 1
       OR paquete.material_clave_envuelto IS DISTINCT FROM
          decode(repeat('ef', 32), 'hex')
       OR paquete.esquema_sobre <>
          'bolsa.convocatoria.borrador.sobre-aead.v1'
       OR paquete.nonce IS DISTINCT FROM decode(repeat('ab', 12), 'hex')
       OR paquete.contenido_cifrado IS DISTINCT FROM
          decode(repeat('cd', 32), 'hex')
       OR paquete.huella_sobre_sha256 IS DISTINCT FROM
          paquete.atestacion_kms ->> 'huella_sobre_sha256'
       OR paquete.procedencia ->> 'autoridad' <> 'autoritativo' THEN
        RAISE EXCEPTION '000005 no recupero paquete exacto de 000004: %',
            paquete;
    END IF;

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
            'convocatoria-lectura-kms#1', lectura, invalida.prueba,
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
