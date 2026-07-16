-- Amplia el fixture V1 mediante nuevas versiones inmutables. No se modifica
-- ni relaja el rol anterior: el puntero de asignacion avanza de V1 a V2.
DO $fixture_rbac_v3$
DECLARE
    ahora timestamptz(6) := clock_timestamp();
    publicada timestamptz(6) := ahora - interval '1 minute';
    z_publicada text := to_char(
        publicada, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    z_vigente text := to_char(
        ahora + interval '1 day', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    rol jsonb;
    control jsonb;
    asignacion jsonb;
BEGIN
    rol := jsonb_build_object(
        'rol_id', 'bolsa_prueba', 'version', 2,
        'publicada_en', z_publicada,
        'concesiones', jsonb_build_array(
            jsonb_build_object(
                'accion', 'bolsa.baremacion.decision.reservar',
                'modulo_id', 'bolsa', 'tipo_recurso', 'baremacion',
                'finalidades', jsonb_build_array('gestion_bolsa'),
                'garantia_minima', 'alto',
                'campos_permitidos',
                    jsonb_build_array('reserva.decision'),
                'obligaciones', '[]'::jsonb
            ),
            jsonb_build_object(
                'accion', 'bolsa.baremacion.archivo.prevalidar',
                'modulo_id', 'bolsa', 'tipo_recurso', 'baremacion',
                'finalidades', jsonb_build_array('gestion_bolsa'),
                'garantia_minima', 'alto',
                'campos_permitidos',
                    jsonb_build_array('archivo_probatorio'),
                'obligaciones', '[]'::jsonb
            ),
            jsonb_build_object(
                'accion', 'bolsa.baremacion.decision.confirmar',
                'modulo_id', 'bolsa', 'tipo_recurso', 'baremacion',
                'finalidades', jsonb_build_array('gestion_bolsa'),
                'garantia_minima', 'alto',
                'campos_permitidos', jsonb_build_array(
                    'baremacion', 'decision', 'evidencia_transaccion'
                ),
                'obligaciones', '[]'::jsonb
            ),
            jsonb_build_object(
                'accion', 'bolsa.baremacion.version.consultar',
                'modulo_id', 'bolsa', 'tipo_recurso', 'baremacion',
                'finalidades', jsonb_build_array('gestion_bolsa'),
                'garantia_minima', 'alto',
                'campos_permitidos', jsonb_build_array('baremacion'),
                'obligaciones', '[]'::jsonb
            ),
            jsonb_build_object(
                'accion', 'bolsa.baremacion.transaccion.consultar',
                'modulo_id', 'bolsa', 'tipo_recurso', 'transaccion',
                'finalidades', jsonb_build_array('gestion_bolsa'),
                'garantia_minima', 'alto',
                'campos_permitidos', jsonb_build_array(
                    'auditoria', 'evento_outbox', 'evidencia_transaccion'
                ),
                'obligaciones', '[]'::jsonb
            )
        )
    );
    INSERT INTO vec_autorizacion.version_rol (
        version_rol_ref, rol_id, version, huella_sha256,
        publicada_en, documento
    ) VALUES (
        'rol:bolsa_prueba:v2', 'bolsa_prueba', 2, repeat('4', 64),
        publicada, rol
    );

    control := jsonb_build_object(
        'version_rol_ref', 'rol:bolsa_prueba:v2', 'revision', 1,
        'estado', 'habilitada', 'actualizado_en', z_publicada
    );
    INSERT INTO vec_autorizacion.control_vigencia_version_rol (
        version_rol_ref, revision, estado, huella_sha256,
        actualizado_en, documento
    ) VALUES (
        'rol:bolsa_prueba:v2', 1, 'habilitada', repeat('5', 64),
        publicada, control
    );
    INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual (
        version_rol_ref, revision, actualizada_en, actualizada_por, acto_ref
    ) VALUES (
        'rol:bolsa_prueba:v2', 1, ahora,
        'prueba:bolsa:v3', 'acto:rol:bolsa:prueba:v3'
    );

    asignacion := jsonb_build_object(
        'asignacion_id', 'bolsa_001', 'version', 2,
        'perfil_activo_ref', 'prf_bolsa_postgresql_prueba_000001',
        'principal_id', 'per_bolsa_postgresql_prueba_000001',
        'version_rol_ref', 'rol:bolsa_prueba:v2',
        'emitida_en', z_publicada,
        'vigente_desde', z_publicada,
        'vigente_hasta', z_vigente,
        'ambitos', jsonb_build_array(jsonb_build_object(
            'clave', 'unidad', 'valores', jsonb_build_array('seleccion')
        ))
    );
    INSERT INTO vec_autorizacion.asignacion_perfil (
        asignacion_ref, asignacion_id, version, perfil_activo_ref,
        principal_id, version_rol_ref, huella_sha256, emitida_en, documento
    ) VALUES (
        'asignacion:bolsa_001:v2', 'bolsa_001', 2,
        'prf_bolsa_postgresql_prueba_000001',
        'per_bolsa_postgresql_prueba_000001',
        'rol:bolsa_prueba:v2', repeat('6', 64), publicada, asignacion
    );
    UPDATE vec_autorizacion.asignacion_perfil_actual
       SET asignacion_ref = 'asignacion:bolsa_001:v2',
           actualizada_en = ahora,
           actualizada_por = 'prueba:bolsa:v3',
           acto_ref = 'acto:asignacion:bolsa:prueba:v3'
     WHERE perfil_activo_ref = 'prf_bolsa_postgresql_prueba_000001';
    IF NOT FOUND THEN
        RAISE EXCEPTION 'falta asignacion V1 autoritativa del fixture';
    END IF;
END
$fixture_rbac_v3$;
