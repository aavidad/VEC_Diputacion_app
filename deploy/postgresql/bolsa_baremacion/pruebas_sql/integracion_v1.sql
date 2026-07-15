-- Prueba integral y adversaria. El arnes la ejecuta como DBA dentro de una
-- transaccion que siempre revierte: no deja identidades ni datos funcionales.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE TEMP TABLE pg_temp.decision_bolsa_prueba (
    decision_ref text PRIMARY KEY,
    prueba jsonb NOT NULL,
    decision_canonica bytea NOT NULL
) ON COMMIT DROP;

CREATE FUNCTION pg_temp.crear_decision_bolsa_prueba(
    p_decision_ref text,
    p_accion text,
    p_tipo_recurso text,
    p_recurso_ref text,
    p_campos jsonb,
    p_instante timestamptz,
    p_registrar_atestacion boolean DEFAULT true
)
RETURNS void
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    emitida timestamptz(6) := p_instante - interval '2 seconds';
    valida_hasta timestamptz(6) := p_instante + interval '2 minutes';
    z_emitida text;
    z_valida text;
    z_verificada text;
    vinculo jsonb;
    documento jsonb;
    canonica jsonb;
    canonica_bytes bytea;
    huella_decision text;
    recurso_bytes bytea := convert_to(
        '{"ambitos":{"sujeto_ref":"sujeto:bolsa:001"},"atributos":{}}',
        'UTF8'
    );
    payload bytea;
    sobre bytea;
    evidencia bytea;
    atestacion_ref text := 'atestacion:' || p_decision_ref;
BEGIN
    z_emitida := to_char(emitida, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
    z_valida := to_char(
        valida_hasta, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    z_verificada := to_char(
        p_instante, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    vinculo := jsonb_build_object(
        'bloque_version', 1,
        'autenticacion_ref', 'aut_bolsa_postgresql_prueba_000001',
        'autenticacion_huella_sha256', repeat('a', 64),
        'asercion_ref', 'ase_bolsa_postgresql_prueba_000001',
        'sesion_ref', 'ses_bolsa_postgresql_prueba_000001',
        'control_sesion_ref', 'cse_bolsa_postgresql_prueba_000001',
        'control_sesion_revision', 1,
        'control_sesion_huella_sha256', repeat('b', 64),
        'cuenta_ref', 'cta_bolsa_postgresql_prueba_000001',
        'cuenta_ordinaria_ref', 'cta_bolsa_postgresql_prueba_000001',
        'principal_id', 'per_bolsa_postgresql_prueba_000001',
        'perfil_activo_ref', 'prf_bolsa_postgresql_prueba_000001',
        'cuenta_privilegiada', false,
        'superficie', 'externa_personal',
        'metodo_observado', 'certificado',
        'garantia_observada', 'alto',
        'politica_garantia_ref', 'pga_bolsa_postgresql_prueba_000001',
        'politica_garantia_huella_sha256', repeat('c', 64),
        'autenticacion_verificada_en', to_char(
            p_instante - interval '2 hours',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'sesion_emitida_en', to_char(
            p_instante - interval '90 minutes',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'sesion_valida_hasta', to_char(
            p_instante + interval '1 hour',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'sesion_revalidada_en', to_char(
            p_instante - interval '1 minute',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'contexto_actor_ref', 'vca_bolsa_postgresql_prueba_000001',
        'contexto_actor_version', 1,
        'contexto_actor_huella_sha256', repeat('d', 64)
    );
    documento := jsonb_build_object(
        'decision_ref', p_decision_ref,
        'concedida', true,
        'codigo', 'concedida',
        'principal_id', 'per_bolsa_postgresql_prueba_000001',
        'perfil_activo_ref', 'prf_bolsa_postgresql_prueba_000001',
        'accion', p_accion,
        'recurso_ref', p_recurso_ref,
        'modulo_id', 'bolsa',
        'tipo_recurso', p_tipo_recurso,
        'contexto_recurso_huella_sha256',
            encode(sha256(recurso_bytes), 'hex'),
        'finalidad', 'gestion_bolsa',
        'correlacion_ref', 'correlacion:bolsa:001',
        'vinculo_autenticacion_actor', vinculo,
        'asignacion_ref', 'asignacion:bolsa_001:v1',
        'asignacion_huella_sha256', repeat('3', 64),
        'version_rol_ref', 'rol:bolsa_prueba:v1',
        'version_rol_huella_sha256', repeat('1', 64),
        'control_vigencia_version_rol_ref', 'rol:bolsa_prueba:v1',
        'control_vigencia_version_rol_revision', 1,
        'control_vigencia_version_rol_huella_sha256', repeat('2', 64),
        'revision_catalogo_politicas', 1,
        'catalogo_politicas_huella_sha256',
            '4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945',
        'politicas_evaluadas_refs', '[]'::jsonb,
        'politicas_evaluadas_huellas_sha256', '{}'::jsonb,
        'politicas_refs', '[]'::jsonb,
        'politicas_huellas_sha256', '{}'::jsonb,
        'garantia_minima', 'alto',
        'campos_permitidos', p_campos,
        'obligaciones', '[]'::jsonb,
        'emitida_en', z_emitida,
        'valida_hasta', z_valida
    );
    INSERT INTO vec_autorizacion.decision_autorizacion (
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
        'per_bolsa_postgresql_prueba_000001',
        'prf_bolsa_postgresql_prueba_000001', p_accion, p_recurso_ref, 'bolsa',
        p_tipo_recurso, encode(sha256(recurso_bytes), 'hex'),
        'gestion_bolsa', 'correlacion:bolsa:001',
        'asignacion:bolsa_001:v1', repeat('3', 64),
        'rol:bolsa_prueba:v1', repeat('1', 64),
        'rol:bolsa_prueba:v1', 1, repeat('2', 64), 1,
        '4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945',
        '{}'::jsonb, '{}'::jsonb, emitida, valida_hasta, documento,
        clock_timestamp()
    );
    canonica := jsonb_build_object(
        'esquema',
            'vec.autorizacion.decision.reforzada.v1.autenticacion-actor',
        'decision_ref', p_decision_ref,
        'concedida', true,
        'codigo', 'concedida',
        'principal_id', 'per_bolsa_postgresql_prueba_000001',
        'perfil_activo_ref', 'prf_bolsa_postgresql_prueba_000001',
        'accion', p_accion,
        'recurso_ref', p_recurso_ref,
        'modulo_id', 'bolsa',
        'tipo_recurso', p_tipo_recurso,
        'contexto_recurso_huella_sha256',
            encode(sha256(recurso_bytes), 'hex'),
        'finalidad', 'gestion_bolsa',
        'correlacion_ref', 'correlacion:bolsa:001',
        'vinculo_autenticacion_actor', vinculo,
        'asignacion_ref', 'asignacion:bolsa_001:v1',
        'asignacion_huella_sha256', repeat('3', 64),
        'version_rol_ref', 'rol:bolsa_prueba:v1',
        'version_rol_huella_sha256', repeat('1', 64),
        'control_vigencia_version_rol_ref', 'rol:bolsa_prueba:v1',
        'control_vigencia_version_rol_revision', 1,
        'control_vigencia_version_rol_huella_sha256', repeat('2', 64),
        'revision_catalogo_politicas', 1,
        'catalogo_politicas_huella_sha256',
            '4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945',
        'politicas_evaluadas', '[]'::jsonb,
        'politicas_aplicables', '[]'::jsonb,
        'garantia_minima', 'alto',
        'campos_permitidos', p_campos,
        'obligaciones', '[]'::jsonb,
        'emitida_en', z_emitida,
        'valida_hasta', z_valida
    );
    IF (SELECT count(*) FROM jsonb_object_keys(canonica)) <> 30 THEN
        RAISE EXCEPTION 'fixture canonica de decision no tiene 30 claves';
    END IF;
    canonica_bytes := convert_to(canonica::text, 'UTF8');
    huella_decision := encode(sha256(canonica_bytes), 'hex');
    INSERT INTO pg_temp.decision_bolsa_prueba VALUES (
        p_decision_ref,
        jsonb_build_object(
            'esquema_huella',
                'vec.autorizacion.decision.reforzada.v1.autenticacion-actor',
            'decision_ref', p_decision_ref,
            'huella_decision_sha256', huella_decision,
            'verificada_en', z_verificada,
            'sujeto_ref', 'sujeto:bolsa:001'
        ),
        canonica_bytes
    );
    IF NOT p_registrar_atestacion THEN
        RETURN;
    END IF;
    payload := convert_to('payload:' || p_decision_ref, 'UTF8');
    sobre := convert_to('cose-sign1-prueba:' || p_decision_ref, 'UTF8');
    evidencia := convert_to('evidencia:' || p_decision_ref, 'UTF8');
    INSERT INTO vec_bolsa_baremacion.atestacion_pdp_version (
        atestacion_ref, version, estado, decision_ref,
        esquema_huella_decision, huella_decision_sha256,
        decision_canonica, suite, algoritmo_cose, audiencia_cose,
        clave_id, audiencia_despliegue, estado_confianza,
        huella_clave_sha256, payload_vec_ad_1, sobre_cose_sign1,
        evidencia_canonica, huella_payload_sha256,
        huella_sobre_sha256, huella_evidencia_sha256, verificada_en,
        raiz_valida_desde, raiz_valida_hasta, revision_confianza,
        huella_configuracion_sha256, configuracion_publicada_en,
        configuracion_expira_en, acto_ref, registrada_en
    ) VALUES (
        atestacion_ref, 1, 'activa', p_decision_ref,
        'vec.autorizacion.decision.reforzada.v1.autenticacion-actor',
        huella_decision, canonica_bytes, 'VEC-AD-COSE-EDDSA-1',
        'EdDSA', 'atestacion_autorizacion_pdp',
        'clave:bolsa:prueba:001', 'despliegue:bolsa:prueba', 'activa',
        repeat('e', 64), payload, sobre, evidencia,
        encode(sha256(payload), 'hex'), encode(sha256(sobre), 'hex'),
        encode(sha256(evidencia), 'hex'), p_instante,
        p_instante - interval '1 hour', p_instante + interval '1 hour',
        'confianza:bolsa:prueba:001', repeat('f', 64),
        p_instante - interval '1 hour', p_instante + interval '1 hour',
        'acto:atestacion:bolsa:prueba:001', clock_timestamp()
    );
    INSERT INTO vec_bolsa_baremacion.atestacion_pdp_actual (
        decision_ref, atestacion_ref, version, estado,
        actualizada_en, acto_ref
    ) VALUES (
        p_decision_ref, atestacion_ref, 1, 'activa', clock_timestamp(),
        'acto:atestacion-actual:bolsa:prueba:001'
    );
END
$funcion$;

DO $prueba$
DECLARE
    ahora timestamptz(6) := clock_timestamp();
    publicada timestamptz(6) := ahora - interval '2 hours';
    z_publicada text;
    z_vigente text;
    rol jsonb;
    control jsonb;
    asignacion jsonb;
    recurso_bytes bytea := convert_to(
        '{"ambitos":{"sujeto_ref":"sujeto:bolsa:001"},"atributos":{}}',
        'UTF8'
    );
    prueba_reserva jsonb;
    canonica_reserva bytea;
    prueba_confirmacion jsonb;
    canonica_confirmacion bytea;
    prueba_lectura jsonb;
    canonica_lectura bytea;
    prueba_sin_atestacion jsonb;
    canonica_sin_atestacion bytea;
    operacion_reserva jsonb;
    operacion_confirmacion jsonb;
    agregado jsonb;
    agregado_bytes bytea;
    resultado text;
    reserva_ref text;
    token_texto text := 'token-reserva-bolsa-prueba-001';
    huella_token text;
    expira timestamptz;
    numero text;
    huella text;
    agregado_devuelto bytea;
    confirmada timestamptz;
    auditoria_ref text;
    huella_auditoria text;
    evento_ref text;
    huella_evento text;
    numero_lectura text;
    huella_lectura text;
    agregado_lectura bytea;
    confirmada_lectura timestamptz;
    auditoria_lectura text;
    canonica_29 bytea;
    canonica_31 bytea;
    prueba_mutada jsonb;
BEGIN
    z_publicada := to_char(
        publicada, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    z_vigente := to_char(
        ahora + interval '1 day', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    rol := jsonb_build_object(
        'rol_id', 'bolsa_prueba', 'version', 1,
        'publicada_en', z_publicada,
        'concesiones', jsonb_build_array(
            jsonb_build_object(
                'accion', 'bolsa.baremacion.alta.reservar',
                'modulo_id', 'bolsa', 'tipo_recurso', 'baremacion',
                'finalidades', jsonb_build_array('gestion_bolsa'),
                'garantia_minima', 'alto',
                'campos_permitidos', jsonb_build_array('reserva.alta'),
                'obligaciones', '[]'::jsonb
            ),
            jsonb_build_object(
                'accion', 'bolsa.baremacion.alta.confirmar',
                'modulo_id', 'bolsa', 'tipo_recurso', 'baremacion',
                'finalidades', jsonb_build_array('gestion_bolsa'),
                'garantia_minima', 'alto',
                'campos_permitidos', jsonb_build_array(
                    'baremacion', 'evidencia_transaccion'
                ),
                'obligaciones', '[]'::jsonb
            ),
            jsonb_build_object(
                'accion', 'bolsa.baremacion.vigente.consultar',
                'modulo_id', 'bolsa', 'tipo_recurso', 'baremacion',
                'finalidades', jsonb_build_array('gestion_bolsa'),
                'garantia_minima', 'alto',
                'campos_permitidos', jsonb_build_array('baremacion'),
                'obligaciones', '[]'::jsonb
            )
        )
    );
    INSERT INTO vec_autorizacion.version_rol (
        version_rol_ref, rol_id, version, huella_sha256,
        publicada_en, documento
    ) VALUES (
        'rol:bolsa_prueba:v1', 'bolsa_prueba', 1, repeat('1', 64),
        publicada, rol
    );
    control := jsonb_build_object(
        'version_rol_ref', 'rol:bolsa_prueba:v1', 'revision', 1,
        'estado', 'habilitada', 'actualizado_en', z_publicada
    );
    INSERT INTO vec_autorizacion.control_vigencia_version_rol (
        version_rol_ref, revision, estado, huella_sha256,
        actualizado_en, documento
    ) VALUES (
        'rol:bolsa_prueba:v1', 1, 'habilitada', repeat('2', 64),
        publicada, control
    );
    INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual (
        version_rol_ref, revision, actualizada_en, actualizada_por, acto_ref
    ) VALUES (
        'rol:bolsa_prueba:v1', 1, publicada,
        'prueba:bolsa', 'acto:rol:bolsa:prueba'
    );
    asignacion := jsonb_build_object(
        'asignacion_id', 'bolsa_001', 'version', 1,
        'perfil_activo_ref', 'prf_bolsa_postgresql_prueba_000001',
        'principal_id', 'per_bolsa_postgresql_prueba_000001',
        'version_rol_ref', 'rol:bolsa_prueba:v1',
        'emitida_en', z_publicada, 'vigente_desde', z_publicada,
        'vigente_hasta', z_vigente,
        'ambitos', jsonb_build_array(jsonb_build_object(
            'clave', 'unidad', 'valores', jsonb_build_array('seleccion')
        ))
    );
    INSERT INTO vec_autorizacion.asignacion_perfil (
        asignacion_ref, asignacion_id, version, perfil_activo_ref,
        principal_id, version_rol_ref, huella_sha256, emitida_en, documento
    ) VALUES (
        'asignacion:bolsa_001:v1', 'bolsa_001', 1,
        'prf_bolsa_postgresql_prueba_000001',
        'per_bolsa_postgresql_prueba_000001',
        'rol:bolsa_prueba:v1', repeat('3', 64),
        publicada, asignacion
    );
    INSERT INTO vec_autorizacion.asignacion_perfil_actual (
        perfil_activo_ref, asignacion_ref, actualizada_en,
        actualizada_por, acto_ref
    ) VALUES (
        'prf_bolsa_postgresql_prueba_000001',
        'asignacion:bolsa_001:v1', publicada,
        'prueba:bolsa', 'acto:asignacion:bolsa:prueba'
    );

    INSERT INTO vec_autorizacion.sesion_autenticacion_v1 (
        sesion_ref, autenticacion_ref, autenticacion_huella_sha256,
        asercion_ref, cuenta_ref, cuenta_ordinaria_ref, cuenta_privilegiada,
        superficie, metodo_observado, garantia_observada,
        politica_garantia_ref, politica_garantia_huella_sha256,
        autenticacion_verificada_en, sesion_emitida_en
    ) VALUES (
        'ses_bolsa_postgresql_prueba_000001',
        'aut_bolsa_postgresql_prueba_000001',
        repeat('a', 64), 'ase_bolsa_postgresql_prueba_000001',
        'cta_bolsa_postgresql_prueba_000001',
        'cta_bolsa_postgresql_prueba_000001', false,
        'externa_personal', 'certificado', 'alto',
        'pga_bolsa_postgresql_prueba_000001', repeat('c', 64),
        ahora - interval '2 hours', ahora - interval '90 minutes'
    );
    INSERT INTO vec_autorizacion.control_sesion_v1 (
        control_sesion_ref, revision, sesion_ref, estado, huella_sha256,
        sesion_revalidada_en, sesion_valida_hasta
    ) VALUES (
        'cse_bolsa_postgresql_prueba_000001', 1,
        'ses_bolsa_postgresql_prueba_000001',
        'activa', repeat('b', 64), ahora - interval '1 minute',
        ahora + interval '1 hour'
    );
    INSERT INTO vec_autorizacion.control_sesion_actual_v1 (
        sesion_ref, control_sesion_ref, revision, actualizada_en, acto_ref
    ) VALUES (
        'ses_bolsa_postgresql_prueba_000001',
        'cse_bolsa_postgresql_prueba_000001', 1,
        ahora - interval '1 second', 'acto:sesion:bolsa:prueba'
    );
    INSERT INTO vec_autorizacion.contexto_actor_v1 (
        contexto_actor_ref, version, cuenta_ref, principal_id,
        perfil_activo_ref, estado, huella_sha256, vigente_desde, vigente_hasta
    ) VALUES (
        'vca_bolsa_postgresql_prueba_000001', 1,
        'cta_bolsa_postgresql_prueba_000001',
        'per_bolsa_postgresql_prueba_000001',
        'prf_bolsa_postgresql_prueba_000001', 'activo', repeat('d', 64),
        ahora - interval '1 hour', ahora + interval '1 hour'
    );
    INSERT INTO vec_autorizacion.contexto_actor_actual_v1 (
        cuenta_ref, perfil_activo_ref, contexto_actor_ref, version,
        actualizada_en, acto_ref
    ) VALUES (
        'cta_bolsa_postgresql_prueba_000001',
        'prf_bolsa_postgresql_prueba_000001',
        'vca_bolsa_postgresql_prueba_000001', 1,
        ahora - interval '1 second',
        'acto:contexto:bolsa:prueba'
    );

    PERFORM pg_temp.crear_decision_bolsa_prueba(
        'decision:bolsa:reserva:001',
        'bolsa.baremacion.alta.reservar', 'baremacion',
        'baremacion:001', jsonb_build_array('reserva.alta'), ahora
    );
    PERFORM pg_temp.crear_decision_bolsa_prueba(
        'decision:bolsa:confirmacion:001',
        'bolsa.baremacion.alta.confirmar', 'baremacion',
        'baremacion:001', jsonb_build_array(
            'baremacion', 'evidencia_transaccion'
        ), ahora
    );
    PERFORM pg_temp.crear_decision_bolsa_prueba(
        'decision:bolsa:lectura:001',
        'bolsa.baremacion.vigente.consultar', 'baremacion',
        'baremacion:001', jsonb_build_array('baremacion'), ahora
    );
    PERFORM pg_temp.crear_decision_bolsa_prueba(
        'decision:bolsa:sin-atestacion:001',
        'bolsa.baremacion.vigente.consultar', 'baremacion',
        'baremacion:001', jsonb_build_array('baremacion'), ahora, false
    );
    SELECT prueba, decision_canonica
      INTO prueba_reserva, canonica_reserva
      FROM pg_temp.decision_bolsa_prueba
     WHERE decision_ref = 'decision:bolsa:reserva:001';
    SELECT prueba, decision_canonica
      INTO prueba_confirmacion, canonica_confirmacion
      FROM pg_temp.decision_bolsa_prueba
     WHERE decision_ref = 'decision:bolsa:confirmacion:001';
    SELECT prueba, decision_canonica
      INTO prueba_lectura, canonica_lectura
      FROM pg_temp.decision_bolsa_prueba
     WHERE decision_ref = 'decision:bolsa:lectura:001';
    SELECT prueba, decision_canonica
      INTO prueba_sin_atestacion, canonica_sin_atestacion
      FROM pg_temp.decision_bolsa_prueba
     WHERE decision_ref = 'decision:bolsa:sin-atestacion:001';

    huella_token := encode(sha256(convert_to(token_texto, 'UTF8')), 'hex');
    operacion_reserva := jsonb_build_object(
        'esquema', 'vec.bolsa.baremacion.reserva-postgresql.v1',
        'reserva_ref', 'reserva:bolsa:001',
        'huella_token_sha256', huella_token,
        'ambito_idempotencia_sha256', repeat('4', 64),
        'clase', 'alta', 'baremacion_merito_ref', 'baremacion:001',
        'version_esperada', '0',
        'huella_version_esperada_sha256', '',
        'huella_solicitud_hmac', 'hmac-sha256:prueba:' || repeat('5', 64),
        'huella_efecto_sha256', repeat('6', 64),
        'solicitada_en', to_char(
            ahora - interval '1 second',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'expira_en', to_char(
            ahora + interval '30 seconds',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )
    );
    SELECT r.resultado, r.reserva_ref, r.expira_en, r.numero_version,
           r.huella_estado_sha256, r.agregado_canonico, r.confirmada_en,
           r.auditoria_ref, r.huella_auditoria_sha256,
           r.evento_outbox_ref, r.huella_evento_outbox_sha256
      INTO resultado, reserva_ref, expira, numero, huella,
           agregado_devuelto, confirmada, auditoria_ref, huella_auditoria,
           evento_ref, huella_evento
      FROM vec_bolsa_baremacion.reservar_cambio(
          operacion_reserva, prueba_reserva, canonica_reserva, recurso_bytes
      ) AS r;
    IF resultado <> 'reservada' OR reserva_ref <> 'reserva:bolsa:001' THEN
        RAISE EXCEPTION 'reserva valida rechazada: %', resultado;
    END IF;
    IF EXISTS (
        SELECT 1 FROM vec_bolsa_baremacion.token_reserva
         WHERE huella_token_sha256 = token_texto
    ) OR EXISTS (
        SELECT 1 FROM vec_bolsa_baremacion.reserva_version
         WHERE to_jsonb(reserva_version)::text LIKE '%' || token_texto || '%'
    ) THEN
        RAISE EXCEPTION 'el token en claro ha sido persistido';
    END IF;

    agregado := jsonb_build_object(
        'id', 'baremacion:001', 'proceso_ref', 'proceso:001',
        'solicitud_ref', 'solicitud:001', 'sujeto_ref', 'sujeto:bolsa:001',
        'criterio', jsonb_build_object('clave', 'criterio:001'),
        'evidencias_iniciales', jsonb_build_array('evidencia:001'),
        'puntos_declarados', 1,
        'calculo_inicial', jsonb_build_object('ref', 'calculo:001'),
        'creada_en', to_char(
            ahora - interval '1 minute',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'decisiones', '[]'::jsonb
    );
    agregado_bytes := convert_to(agregado::text, 'UTF8');
    operacion_confirmacion := jsonb_build_object(
        'esquema', 'vec.bolsa.baremacion.confirmacion-postgresql.v1',
        'huella_token_sha256', huella_token, 'clase', 'alta',
        'version_esperada', '0',
        'huella_version_esperada_sha256', '',
        'huella_solicitud_hmac', 'hmac-sha256:prueba:' || repeat('5', 64),
        'huella_efecto_sha256', repeat('7', 64),
        'huella_agregado_sha256', encode(sha256(agregado_bytes), 'hex'),
        'motivo_clave', 'alta_inicial',
        'motivo', 'Alta inicial de la baremacion',
        'confirmada_en', to_char(
            ahora, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'auditoria_ref', 'auditoria:bolsa:001',
        'evento_outbox_ref', 'evento:bolsa:001'
    );
    SELECT c.resultado, c.numero_version, c.huella_estado_sha256,
           c.agregado_canonico, c.confirmada_en, c.auditoria_ref,
           c.huella_auditoria_sha256, c.evento_outbox_ref,
           c.huella_evento_outbox_sha256
      INTO resultado, numero, huella, agregado_devuelto, confirmada,
           auditoria_ref, huella_auditoria, evento_ref, huella_evento
      FROM vec_bolsa_baremacion.confirmar_cambio(
          operacion_confirmacion, prueba_confirmacion,
          canonica_confirmacion, recurso_bytes, agregado_bytes
      ) AS c;
    IF resultado <> 'confirmada' OR numero <> '1'
       OR auditoria_ref <> 'auditoria:bolsa:001'
       OR evento_ref <> 'evento:bolsa:001'
       OR huella <> encode(sha256(agregado_bytes), 'hex') THEN
        RAISE EXCEPTION 'confirmacion atomica invalida: %/%', resultado, numero;
    END IF;
    IF (SELECT count(*) FROM vec_bolsa_baremacion.version_baremacion) <> 1
       OR (SELECT count(*) FROM vec_bolsa_baremacion.auditoria) <> 1
       OR (SELECT count(*) FROM vec_bolsa_baremacion.evento_outbox) <> 1 THEN
        RAISE EXCEPTION 'version, auditoria y outbox no son uno-a-uno';
    END IF;

    SELECT c.resultado
      INTO resultado
      FROM vec_bolsa_baremacion.confirmar_cambio(
          operacion_confirmacion, prueba_confirmacion,
          canonica_confirmacion, recurso_bytes, agregado_bytes
      ) AS c;
    IF resultado <> 'confirmada'
       OR (SELECT count(*) FROM vec_bolsa_baremacion.auditoria) <> 1 THEN
        RAISE EXCEPTION 'reintento exacto no fue idempotente';
    END IF;
    SELECT c.resultado
      INTO resultado
      FROM vec_bolsa_baremacion.confirmar_cambio(
          jsonb_set(
              operacion_confirmacion, '{huella_efecto_sha256}',
              to_jsonb(repeat('8', 64))
          ), prueba_confirmacion, canonica_confirmacion,
          recurso_bytes, agregado_bytes
      ) AS c;
    IF resultado <> 'idempotencia_reutilizada' THEN
        RAISE EXCEPTION 'reutilizacion de confirmacion admitida: %', resultado;
    END IF;

    SELECT l.resultado, l.numero_version, l.huella_estado_sha256,
           l.agregado_canonico, l.confirmada_en, l.auditoria_ref
      INTO resultado, numero_lectura, huella_lectura, agregado_lectura,
           confirmada_lectura, auditoria_lectura
      FROM vec_bolsa_baremacion.obtener_version_vigente(
          jsonb_build_object(
              'esquema',
                  'vec.bolsa.baremacion.lectura-vigente-postgresql.v1',
              'baremacion_merito_ref', 'baremacion:001',
              'huella_efecto_sha256', repeat('9', 64)
          ), prueba_lectura, canonica_lectura, recurso_bytes
      ) AS l;
    IF resultado <> 'obtenida' OR numero_lectura <> '1'
       OR huella_lectura <> huella OR agregado_lectura <> agregado_bytes
       OR auditoria_lectura <> auditoria_ref THEN
        RAISE EXCEPTION 'lectura sensible inconsistente: %', resultado;
    END IF;
    SELECT l.resultado
      INTO resultado
      FROM vec_bolsa_baremacion.obtener_version_vigente(
          jsonb_build_object(
              'esquema',
                  'vec.bolsa.baremacion.lectura-vigente-postgresql.v1',
              'baremacion_merito_ref', 'baremacion:001',
              'huella_efecto_sha256', repeat('a', 64)
          ), prueba_sin_atestacion, canonica_sin_atestacion, recurso_bytes
      ) AS l;
    IF resultado <> 'autorizacion_obsoleta' THEN
        RAISE EXCEPTION 'ausencia de atestacion no fallo cerrada: %', resultado;
    END IF;

    canonica_29 := convert_to(
        ((convert_from(canonica_lectura, 'UTF8')::jsonb) - 'obligaciones')::text,
        'UTF8'
    );
    prueba_mutada := jsonb_set(
        prueba_lectura, '{huella_decision_sha256}',
        to_jsonb(encode(sha256(canonica_29), 'hex'))
    );
    IF vec_autorizacion.revalidar_decision_bolsa_baremacion_v1(
        prueba_mutada, canonica_29, recurso_bytes,
        'bolsa.baremacion.vigente.consultar', 'baremacion',
        'baremacion:001', jsonb_build_array('baremacion'), clock_timestamp()
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'decision de 29 claves admitida';
    END IF;
    canonica_31 := convert_to(
        ((convert_from(canonica_lectura, 'UTF8')::jsonb) ||
         jsonb_build_object('extension_no_gobernada', true))::text,
        'UTF8'
    );
    prueba_mutada := jsonb_set(
        prueba_lectura, '{huella_decision_sha256}',
        to_jsonb(encode(sha256(canonica_31), 'hex'))
    );
    IF vec_autorizacion.revalidar_decision_bolsa_baremacion_v1(
        prueba_mutada, canonica_31, recurso_bytes,
        'bolsa.baremacion.vigente.consultar', 'baremacion',
        'baremacion:001', jsonb_build_array('baremacion'), clock_timestamp()
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'decision de 31 claves admitida';
    END IF;
END
$prueba$;

-- El consumidor queda ligado al LOGIN exacto de esta sesion. En produccion
-- esta configuracion se aplica mediante una migracion operativa separada.
INSERT INTO vec_bolsa_baremacion.consumidor_outbox_version (
    consumidor_ref, version, estado, rol_sesion, secuencia_inicial,
    registrada_en, acto_ref
) VALUES (
    'consumidor:bolsa:prueba', 1, 'activo', session_user::text, 0,
    clock_timestamp(), 'acto:consumidor:bolsa:prueba'
);
INSERT INTO vec_bolsa_baremacion.consumidor_outbox_actual (
    consumidor_ref, version, estado, actualizada_en
) VALUES (
    'consumidor:bolsa:prueba', 1, 'activo', clock_timestamp()
);

SET LOCAL ROLE vec_bolsa_baremacion_lector_outbox;
DO $entrega_outbox$
DECLARE
    token_uno bytea := decode(repeat('11', 32), 'hex');
    token_dos bytea := decode(repeat('22', 32), 'hex');
    token_ajeno bytea := decode(repeat('33', 32), 'hex');
    resultado text;
    evento_ref text;
    secuencia text;
    tipo text;
    carga jsonb;
    expira_en timestamptz;
BEGIN
    SELECT r.resultado, r.evento_ref, r.secuencia, r.tipo, r.carga,
           r.arrendamiento_expira_en
      INTO resultado, evento_ref, secuencia, tipo, carga, expira_en
      FROM vec_bolsa_baremacion.reclamar_evento_outbox(
          'consumidor:bolsa:prueba', token_uno, 60
      ) AS r;
    IF resultado <> 'reclamada' OR evento_ref <> 'evento:bolsa:001'
       OR secuencia <> '1' OR tipo <> 'bolsa.baremacion_creada.v1'
       OR carga ->> 'huella_registro_sha256' IS NULL
       OR expira_en <= clock_timestamp() THEN
        RAISE EXCEPTION 'reclamacion outbox invalida: %/%',
            resultado, evento_ref;
    END IF;

    -- Un reintento con el mismo token recupera el mismo arrendamiento; otro
    -- token no puede robarlo.
    SELECT r.resultado, r.evento_ref
      INTO resultado, evento_ref
      FROM vec_bolsa_baremacion.reclamar_evento_outbox(
          'consumidor:bolsa:prueba', token_uno, 60
      ) AS r;
    IF resultado <> 'reclamada' OR evento_ref <> 'evento:bolsa:001' THEN
        RAISE EXCEPTION 'reintento de reclamacion no idempotente';
    END IF;
    SELECT r.resultado INTO resultado
      FROM vec_bolsa_baremacion.reclamar_evento_outbox(
          'consumidor:bolsa:prueba', token_ajeno, 60
      ) AS r;
    IF resultado <> 'ocupada' THEN
        RAISE EXCEPTION 'arrendamiento pudo ser robado: %', resultado;
    END IF;

    SELECT vec_bolsa_baremacion.finalizar_entrega_outbox(
        'consumidor:bolsa:prueba', 'evento:bolsa:001', token_uno,
        'fallida', 'destino_temporalmente_no_disponible'
    ) INTO resultado;
    IF resultado <> 'fallida' THEN
        RAISE EXCEPTION 'fallo outbox no registrado: %', resultado;
    END IF;
    SELECT vec_bolsa_baremacion.finalizar_entrega_outbox(
        'consumidor:bolsa:prueba', 'evento:bolsa:001', token_uno,
        'fallida', 'destino_temporalmente_no_disponible'
    ) INTO resultado;
    IF resultado <> 'fallida' THEN
        RAISE EXCEPTION 'reintento de fallo no idempotente: %', resultado;
    END IF;

    SELECT r.resultado, r.evento_ref
      INTO resultado, evento_ref
      FROM vec_bolsa_baremacion.reclamar_evento_outbox(
          'consumidor:bolsa:prueba', token_dos, 60
      ) AS r;
    IF resultado <> 'reclamada' OR evento_ref <> 'evento:bolsa:001' THEN
        RAISE EXCEPTION 'segundo intento no reclamado: %', resultado;
    END IF;
    SELECT vec_bolsa_baremacion.finalizar_entrega_outbox(
        'consumidor:bolsa:prueba', 'evento:bolsa:001', token_dos,
        'entregada', NULL
    ) INTO resultado;
    IF resultado <> 'entregada' THEN
        RAISE EXCEPTION 'entrega outbox no confirmada: %', resultado;
    END IF;
    SELECT vec_bolsa_baremacion.finalizar_entrega_outbox(
        'consumidor:bolsa:prueba', 'evento:bolsa:001', token_dos,
        'entregada', NULL
    ) INTO resultado;
    IF resultado <> 'entregada' THEN
        RAISE EXCEPTION 'reintento de entrega no idempotente: %', resultado;
    END IF;
    SELECT r.resultado INTO resultado
      FROM vec_bolsa_baremacion.reclamar_evento_outbox(
          'consumidor:bolsa:prueba', token_ajeno, 60
      ) AS r;
    IF resultado <> 'sin_evento' THEN
        RAISE EXCEPTION 'cursor no avanzo: %', resultado;
    END IF;

    SELECT r.resultado INTO resultado
      FROM vec_bolsa_baremacion.reclamar_evento_outbox(
          'consumidor:no-registrado', token_ajeno, 60
      ) AS r;
    IF resultado <> 'consumidor_no_autorizado' THEN
        RAISE EXCEPTION 'consumidor no registrado admitido: %', resultado;
    END IF;
END
$entrega_outbox$;
RESET ROLE;

DO $inmutabilidad_entrega$
BEGIN
    IF (SELECT estado FROM vec_bolsa_baremacion.evento_outbox
         WHERE referencia = 'evento:bolsa:001') <> 'pendiente' THEN
        RAISE EXCEPTION 'la entrega muto el evento de dominio';
    END IF;
    IF (SELECT count(*) FROM vec_bolsa_baremacion.evento_outbox) <> 1
       OR (SELECT count(*) FROM
           vec_bolsa_baremacion.entrega_outbox_version) <> 4
       OR (SELECT count(*) FROM
           vec_bolsa_baremacion.cursor_outbox_version) <> 1 THEN
        RAISE EXCEPTION 'historia de entrega/cursor inesperada';
    END IF;
    IF (SELECT intento FROM vec_bolsa_baremacion.entrega_outbox_actual
         WHERE consumidor_ref = 'consumidor:bolsa:prueba'
           AND evento_ref = 'evento:bolsa:001') <> 2 THEN
        RAISE EXCEPTION 'contador de intentos outbox inconsistente';
    END IF;
END
$inmutabilidad_entrega$;

DO $privilegios$
BEGIN
    IF has_table_privilege(
        'vec_bolsa_baremacion_ejecutor',
        'vec_bolsa_baremacion.version_baremacion', 'SELECT'
    ) OR has_table_privilege(
        'vec_bolsa_baremacion_ejecutor',
        'vec_bolsa_baremacion.version_baremacion', 'INSERT'
    ) OR has_table_privilege(
        'vec_bolsa_baremacion_ejecutor',
        'vec_bolsa_baremacion.auditoria', 'UPDATE'
    ) THEN
        RAISE EXCEPTION 'el runtime posee acceso directo a tablas';
    END IF;
    IF has_function_privilege(
        'vec_bolsa_baremacion_ejecutor',
        'vec_autorizacion.revalidar_decision_bolsa_baremacion_v1(jsonb,bytea,bytea,text,text,text,jsonb,timestamp with time zone)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'el runtime puede invocar la frontera interna auth';
    END IF;
    IF NOT has_function_privilege(
        'vec_bolsa_baremacion_ejecutor',
        'vec_bolsa_baremacion.confirmar_cambio(jsonb,jsonb,bytea,bytea,bytea)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'el runtime no puede invocar su funcion cerrada';
    END IF;
    IF has_table_privilege(
        'vec_bolsa_baremacion_lector_outbox',
        'vec_bolsa_baremacion.evento_outbox', 'SELECT'
    ) OR has_table_privilege(
        'vec_bolsa_baremacion_lector_outbox',
        'vec_bolsa_baremacion.entrega_outbox_actual', 'UPDATE'
    ) THEN
        RAISE EXCEPTION 'el lector outbox posee acceso directo a tablas';
    END IF;
    IF has_function_privilege(
        'vec_bolsa_baremacion_ejecutor',
        'vec_bolsa_baremacion.reclamar_evento_outbox(text,bytea,integer)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'el runtime funcional puede consumir el outbox';
    END IF;
END
$privilegios$;

\if :{?CONFIRMAR_FIXTURE}
    \if :CONFIRMAR_FIXTURE
        COMMIT;
    \else
        ROLLBACK;
    \endif
\else
    ROLLBACK;
\endif
