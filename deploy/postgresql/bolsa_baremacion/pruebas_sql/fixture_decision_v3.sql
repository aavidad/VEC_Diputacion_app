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
    autenticacion_verificada timestamptz(6);
    sesion_emitida timestamptz(6);
    sesion_valida timestamptz(6);
    sesion_revalidada timestamptz(6);
    contexto_actor_ref text;
    contexto_actor_version numeric(20, 0);
    contexto_actor_huella text;
BEGIN
    SELECT sesion.autenticacion_verificada_en, sesion.sesion_emitida_en,
           control.sesion_valida_hasta, control.sesion_revalidada_en,
           contexto.contexto_actor_ref, contexto.version,
           contexto.huella_sha256
      INTO autenticacion_verificada, sesion_emitida, sesion_valida,
           sesion_revalidada, contexto_actor_ref,
           contexto_actor_version, contexto_actor_huella
      FROM vec_autorizacion.sesion_autenticacion_v1 AS sesion
      JOIN vec_autorizacion.control_sesion_actual_v1 AS actual
        ON actual.sesion_ref = sesion.sesion_ref
      JOIN vec_autorizacion.control_sesion_v1 AS control
        ON control.control_sesion_ref = actual.control_sesion_ref
       AND control.revision = actual.revision
      JOIN vec_autorizacion.contexto_actor_actual_v1 AS actor_actual
        ON actor_actual.cuenta_ref = sesion.cuenta_ref
       AND actor_actual.perfil_activo_ref =
           'prf_bolsa_postgresql_prueba_000001'
      JOIN vec_autorizacion.contexto_actor_v1 AS contexto
        ON contexto.cuenta_ref = actor_actual.cuenta_ref
       AND contexto.perfil_activo_ref = actor_actual.perfil_activo_ref
       AND contexto.contexto_actor_ref = actor_actual.contexto_actor_ref
       AND contexto.version = actor_actual.version
     WHERE sesion.sesion_ref = 'ses_bolsa_postgresql_prueba_000001';
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
            autenticacion_verificada,
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'sesion_emitida_en', to_char(
            sesion_emitida,
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'sesion_valida_hasta', to_char(
            sesion_valida,
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'sesion_revalidada_en', to_char(
            sesion_revalidada,
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'contexto_actor_ref', contexto_actor_ref,
        'contexto_actor_version', contexto_actor_version,
        'contexto_actor_huella_sha256', contexto_actor_huella
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
        'asignacion_ref', 'asignacion:bolsa_001:v2',
        'asignacion_huella_sha256', repeat('6', 64),
        'version_rol_ref', 'rol:bolsa_prueba:v2',
        'version_rol_huella_sha256', repeat('4', 64),
        'control_vigencia_version_rol_ref', 'rol:bolsa_prueba:v2',
        'control_vigencia_version_rol_revision', 1,
        'control_vigencia_version_rol_huella_sha256', repeat('5', 64),
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
        'asignacion:bolsa_001:v2', repeat('6', 64),
        'rol:bolsa_prueba:v2', repeat('4', 64),
        'rol:bolsa_prueba:v2', 1, repeat('5', 64), 1,
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
        'asignacion_ref', 'asignacion:bolsa_001:v2',
        'asignacion_huella_sha256', repeat('6', 64),
        'version_rol_ref', 'rol:bolsa_prueba:v2',
        'version_rol_huella_sha256', repeat('4', 64),
        'control_vigencia_version_rol_ref', 'rol:bolsa_prueba:v2',
        'control_vigencia_version_rol_revision', 1,
        'control_vigencia_version_rol_huella_sha256', repeat('5', 64),
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
