-- Siembra una decision historica de 30 claves ANTES de aplicar 000002. Sirve
-- para demostrar que la evolucion conserva evidencia sin admitir nuevas V1.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $siembra$
DECLARE
    ahora timestamptz(6) := clock_timestamp();
    publicada timestamptz(6) := ahora - interval '2 hours';
    valida_hasta timestamptz(6) := ahora + interval '2 minutes';
    vigente_hasta timestamptz(6) := ahora + interval '1 day';
    z_publicada text;
    z_ahora text;
    z_valida text;
    z_vigente text;
    documento_rol jsonb;
    documento_control jsonb;
    documento_asignacion jsonb;
    documento_decision jsonb;
BEGIN
    z_publicada := to_char(publicada, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
    z_ahora := to_char(ahora, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
    z_valida := to_char(valida_hasta, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
    z_vigente := to_char(vigente_hasta, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');

    documento_rol := jsonb_build_object(
        'rol_id', 'legacy_v1', 'version', 1,
        'publicada_en', z_publicada,
        'concesiones', jsonb_build_array(jsonb_build_object(
            'accion', 'vec.documentos.ejecucion.ejecutar_plan_v4',
            'modulo_id', 'bolsa', 'tipo_recurso', 'documento',
            'finalidades', jsonb_build_array('gestion_bolsa'),
            'garantia_minima', 'alto',
            'campos_permitidos', '[]'::jsonb,
            'obligaciones', '[]'::jsonb
        ))
    );
    INSERT INTO vec_autorizacion.version_rol (
        version_rol_ref, rol_id, version, huella_sha256,
        publicada_en, documento
    ) VALUES (
        'rol:legacy_v1:v1', 'legacy_v1', 1, repeat('1', 64),
        publicada, documento_rol
    );

    documento_control := jsonb_build_object(
        'version_rol_ref', 'rol:legacy_v1:v1', 'revision', 1,
        'estado', 'habilitada', 'actualizado_en', z_publicada
    );
    INSERT INTO vec_autorizacion.control_vigencia_version_rol (
        version_rol_ref, revision, estado, huella_sha256,
        actualizado_en, documento
    ) VALUES (
        'rol:legacy_v1:v1', 1, 'habilitada', repeat('2', 64),
        publicada, documento_control
    );
    INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual (
        version_rol_ref, revision, actualizada_en, actualizada_por, acto_ref
    ) VALUES (
        'rol:legacy_v1:v1', 1, publicada,
        'migracion:prueba:legacy', 'acto:prueba:legacy:control'
    );

    documento_asignacion := jsonb_build_object(
        'asignacion_id', 'legacy_v1', 'version', 1,
        'perfil_activo_ref', 'perfil:legacy-v1',
        'principal_id', 'persona:legacy-v1',
        'version_rol_ref', 'rol:legacy_v1:v1',
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
        'asignacion:legacy_v1:v1', 'legacy_v1', 1,
        'perfil:legacy-v1', 'persona:legacy-v1', 'rol:legacy_v1:v1',
        repeat('3', 64), publicada, documento_asignacion
    );
    INSERT INTO vec_autorizacion.asignacion_perfil_actual (
        perfil_activo_ref, asignacion_ref, actualizada_en,
        actualizada_por, acto_ref
    ) VALUES (
        'perfil:legacy-v1', 'asignacion:legacy_v1:v1', publicada,
        'migracion:prueba:legacy', 'acto:prueba:legacy:asignacion'
    );

    documento_decision := jsonb_build_object(
        'decision_ref', 'decision:prueba:legacy:v1',
        'concedida', true, 'codigo', 'concedida',
        'principal_id', 'persona:legacy-v1',
        'perfil_activo_ref', 'perfil:legacy-v1',
        'accion', 'vec.documentos.ejecucion.ejecutar_plan_v4',
        'recurso_ref', 'recurso:prueba:legacy:v1',
        'modulo_id', 'bolsa', 'tipo_recurso', 'documento',
        'contexto_recurso_huella_sha256', repeat('4', 64),
        'finalidad', 'gestion_bolsa',
        'correlacion_ref', 'correlacion:prueba:legacy:v1',
        'asignacion_ref', 'asignacion:legacy_v1:v1',
        'asignacion_huella_sha256', repeat('3', 64),
        'version_rol_ref', 'rol:legacy_v1:v1',
        'version_rol_huella_sha256', repeat('1', 64),
        'control_vigencia_version_rol_ref', 'rol:legacy_v1:v1',
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
        'campos_permitidos', '[]'::jsonb,
        'obligaciones', '[]'::jsonb,
        'emitida_en', z_ahora, 'valida_hasta', z_valida
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
        'decision:prueba:legacy:v1', true, 'concedida',
        'persona:legacy-v1', 'perfil:legacy-v1',
        'vec.documentos.ejecucion.ejecutar_plan_v4',
        'recurso:prueba:legacy:v1', 'bolsa', 'documento', repeat('4', 64),
        'gestion_bolsa', 'correlacion:prueba:legacy:v1',
        'asignacion:legacy_v1:v1', repeat('3', 64),
        'rol:legacy_v1:v1', repeat('1', 64), 'rol:legacy_v1:v1', 1,
        repeat('2', 64), 1,
        '4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945',
        '{}'::jsonb, '{}'::jsonb, ahora, valida_hasta,
        documento_decision, ahora
    );
END
$siembra$;

COMMIT;
