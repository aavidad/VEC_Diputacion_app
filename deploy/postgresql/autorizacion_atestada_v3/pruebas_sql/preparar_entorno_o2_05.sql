\set ON_ERROR_STOP 1

-- RBAC sintético específico de la vertical. No contiene datos personales.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $fixture_rbac$
DECLARE
    v_ahora timestamptz(6) := clock_timestamp();
    v_desde timestamptz(6) := v_ahora - interval '10 minutes';
    v_hasta timestamptz(6) := v_ahora + interval '2 hours';
    v_desde_z text := to_char(
        v_desde AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    v_hasta_z text := to_char(
        v_hasta AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    v_rol jsonb;
    v_control jsonb;
    v_asignacion jsonb;
BEGIN
    v_rol := jsonb_build_object(
        'rol_id', 'registro_ct_o205', 'version', 1,
        'nombre', 'Registro CT O2-05', 'estado', 'publicada',
        'concesiones', jsonb_build_array(jsonb_build_object(
            'accion', 'contratacion_temporal.solicitud.crear',
            'modulo_id', 'contratacion_temporal',
            'tipo_recurso', 'expediente_contratacion_temporal',
            'finalidades',
                jsonb_build_array(
                    'tramitar_necesidad_personal_temporal'
                ),
            'garantia_minima', 'alto',
            'campos_permitidos', jsonb_build_array('estado'),
            'obligaciones', jsonb_build_array('auditar')
        )),
        'publicada_por', 'usr_seguridad_sintetico_o205',
        'publicada_en', v_desde_z
    );
    INSERT INTO vec_autorizacion.version_rol(
        version_rol_ref, rol_id, version, huella_sha256,
        publicada_en, documento
    ) VALUES (
        'rol:registro_ct_o205:v1', 'registro_ct_o205', 1,
        repeat('c', 64), v_desde, v_rol
    );
    v_control := jsonb_build_object(
        'version_rol_ref', 'rol:registro_ct_o205:v1',
        'revision', 1, 'estado', 'habilitada',
        'actualizado_por', 'usr_seguridad_sintetico_o205',
        'actualizado_en', v_desde_z
    );
    INSERT INTO vec_autorizacion.control_vigencia_version_rol(
        version_rol_ref, revision, estado, huella_sha256,
        actualizado_en, documento
    ) VALUES (
        'rol:registro_ct_o205:v1', 1, 'habilitada',
        repeat('d', 64), v_desde, v_control
    );
    INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
        VALUES (
            'rol:registro_ct_o205:v1', 1, v_ahora,
            'usr_seguridad_sintetico_o205',
            'acto:control:registro-ct-o205'
        );
    v_asignacion := jsonb_build_object(
        'asignacion_id', 'registro_v3', 'version', 2,
        'perfil_activo_ref',
            'prf_sintetico_cccccccccccccccccccccccc',
        'principal_id',
            'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
        'version_rol_ref', 'rol:registro_ct_o205:v1',
        'estado', 'activa', 'emitida_en', v_desde_z,
        'vigente_desde', v_desde_z, 'vigente_hasta', v_hasta_z,
        'ambitos', jsonb_build_array(jsonb_build_object(
            'clave', 'unidad',
            'valores', jsonb_build_array('seleccion')
        ))
    );
    INSERT INTO vec_autorizacion.asignacion_perfil(
        asignacion_ref, asignacion_id, version, perfil_activo_ref,
        principal_id, version_rol_ref, huella_sha256,
        emitida_en, documento
    ) VALUES (
        'asignacion:registro_v3:v2',
        'registro_v3', 2,
        'prf_sintetico_cccccccccccccccccccccccc',
        'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
        'rol:registro_ct_o205:v1', repeat('e', 64),
        v_desde, v_asignacion
    );
    INSERT INTO vec_autorizacion.asignacion_perfil_actual(
        perfil_activo_ref, asignacion_ref, actualizada_en,
        actualizada_por, acto_ref
    ) VALUES (
        'prf_sintetico_cccccccccccccccccccccccc',
        'asignacion:registro_v3:v2',
        v_ahora, 'usr_rrhh_sintetico_o205',
        'acto:asignacion:registro-ct-o205'
    ) ON CONFLICT (perfil_activo_ref) DO UPDATE SET
        asignacion_ref = EXCLUDED.asignacion_ref,
        actualizada_en = EXCLUDED.actualizada_en,
        actualizada_por = EXCLUDED.actualizada_por,
        acto_ref = EXCLUDED.acto_ref;
END
$fixture_rbac$;
COMMIT;

-- Gobierno sintético. Los secretos solo existen en el contenedor efímero.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $fixture_gobierno$
DECLARE
    v_ahora timestamptz(6) := clock_timestamp();
    v_secreto_hmac bytea := public.gen_random_bytes(32);
    v_secreto_no_activado bytea := public.gen_random_bytes(32);
    v_spki bytea := decode(
        '302a300506032b6570032100' || repeat('11', 32),
        'hex'
    );
BEGIN
    INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version (
        clave_id, version, revision_gobierno,
        huella_gobierno_sha256, secreto_hmac,
        huella_secreto_sha256, emisor_id, audiencia_consumo,
        valida_desde, valida_hasta, acto_ref
    ) VALUES (
        'clave-capacidad-o205-1', 1, 1, repeat('1', 64),
        v_secreto_hmac,
        encode(sha256(v_secreto_hmac), 'hex'),
        'broker-o205-sintetico',
        'vec_contratacion_temporal.confirmar_alta_atestada.v1',
        v_ahora - interval '1 hour',
        v_ahora + interval '2 hours',
        'acto:clave-capacidad:o205:1'
    );
    INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision (
        orden, clave_id, version, establecida_en, acto_ref
    ) VALUES (
        1, 'clave-capacidad-o205-1', 1,
        v_ahora - interval '1 minute',
        'acto:puntero-clave:o205:1'
    );
    INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version (
        clave_id, version, revision_gobierno,
        huella_gobierno_sha256, secreto_hmac,
        huella_secreto_sha256, emisor_id, audiencia_consumo,
        valida_desde, valida_hasta, acto_ref
    ) VALUES (
        'clave-capacidad-o205-no-activada', 99, 99, repeat('9', 64),
        v_secreto_no_activado,
        encode(sha256(v_secreto_no_activado), 'hex'),
        'broker-o205-sintetico',
        'vec_contratacion_temporal.confirmar_alta_atestada.v1',
        v_ahora - interval '1 hour',
        v_ahora + interval '2 hours',
        'acto:clave-capacidad:o205:no-activada'
    );
    INSERT INTO
        vec_autorizacion_atestada_v3.configuracion_confianza_version (
            revision, secuencia, huella_configuracion_sha256,
            publicada_en, expira_en, acto_ref
        ) VALUES (
            'configuracion-o205-1', 1, repeat('2', 64),
            v_ahora - interval '10 minutes',
            v_ahora + interval '2 hours',
            'acto:configuracion:o205:1'
        );
    INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version (
        clave_id, version, clave_publica_spki, huella_spki_sha256,
        valida_desde, valida_hasta, suite, audiencia_despliegue,
        acto_ref
    ) VALUES (
        'raiz-o205-1', 1, v_spki, encode(sha256(v_spki), 'hex'),
        v_ahora - interval '1 hour',
        v_ahora + interval '2 hours',
        'VEC-AD-3-COSE-EDDSA-1',
        'vec-diputacion/pruebas/o205/consumidor',
        'acto:raiz:o205:1'
    );
    INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz
        VALUES ('configuracion-o205-1', 'raiz-o205-1', 1);
    INSERT INTO
        vec_autorizacion_atestada_v3.puntero_configuracion_actual (
            orden, configuracion_revision, establecida_en, acto_ref
        ) VALUES (
            1, 'configuracion-o205-1',
            v_ahora - interval '1 minute',
            'acto:puntero-configuracion:o205:1'
        );
END
$fixture_gobierno$;
COMMIT;
