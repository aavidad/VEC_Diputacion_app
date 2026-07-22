-- Datos RBAC, sesion y motivo exclusivamente sinteticos para V3.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $fixture$
DECLARE
    ahora timestamptz(6) := clock_timestamp();
    publicada timestamptz(6) := ahora - interval '10 minutes';
    valida timestamptz(6) := ahora + interval '30 minutes';
    z_publicada text := to_char(
        publicada AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    z_valida text := to_char(
        valida AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    rol jsonb;
    asignacion jsonb;
    control jsonb;
BEGIN
    rol := jsonb_build_object(
        'rol_id','registro_v3','version',1,'nombre','Registro V3',
        'estado','publicada',
        'concesiones',jsonb_build_array(jsonb_build_object(
            'accion','consultar','modulo_id','bolsa','tipo_recurso','expediente',
            'finalidades',jsonb_build_array('gestion'),
            'garantia_minima','alto',
            'campos_permitidos',jsonb_build_array('estado'),
            'obligaciones',jsonb_build_array('auditar')
        )),
        'publicada_por','usr_seguridad_sintetico_v3',
        'publicada_en',z_publicada
    );
    INSERT INTO vec_autorizacion.version_rol(
        version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento
    ) VALUES (
        'rol:registro_v3:v1','registro_v3',1,repeat('2',64),publicada,rol
    );
    control := jsonb_build_object(
        'version_rol_ref','rol:registro_v3:v1','revision',1,
        'estado','habilitada','actualizado_por','usr_seguridad_sintetico_v3',
        'actualizado_en',z_publicada
    );
    INSERT INTO vec_autorizacion.control_vigencia_version_rol(
        version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento
    ) VALUES (
        'rol:registro_v3:v1',1,'habilitada',repeat('3',64),publicada,control
    );
    INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual VALUES (
        'rol:registro_v3:v1',1,ahora,'usr_seguridad_sintetico_v3',
        'acto:control:registro-v3'
    );
    asignacion := jsonb_build_object(
        'asignacion_id','registro_v3','version',1,
        'perfil_activo_ref','prf_sintetico_cccccccccccccccccccccccc',
        'principal_id','per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
        'version_rol_ref','rol:registro_v3:v1','estado','activa',
        'emitida_en',z_publicada,'vigente_desde',z_publicada,
        'vigente_hasta',z_valida,
        'ambitos',jsonb_build_array(jsonb_build_object(
            'clave','unidad','valores',jsonb_build_array('seleccion')
        ))
    );
    INSERT INTO vec_autorizacion.asignacion_perfil(
        asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,
        version_rol_ref,huella_sha256,emitida_en,documento
    ) VALUES (
        'asignacion:registro_v3:v1','registro_v3',1,
        'prf_sintetico_cccccccccccccccccccccccc',
        'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
        'rol:registro_v3:v1',repeat('4',64),publicada,asignacion
    );
    INSERT INTO vec_autorizacion.asignacion_perfil_actual VALUES (
        'prf_sintetico_cccccccccccccccccccccccc',
        'asignacion:registro_v3:v1',ahora,'usr_rrhh_sintetico_v3',
        'acto:asignacion:registro-v3'
    );

    INSERT INTO vec_autorizacion.sesion_autenticacion_v1(
        sesion_ref,autenticacion_ref,autenticacion_huella_sha256,asercion_ref,
        cuenta_ref,cuenta_ordinaria_ref,cuenta_privilegiada,superficie,
        metodo_observado,garantia_observada,politica_garantia_ref,
        politica_garantia_huella_sha256,autenticacion_verificada_en,
        sesion_emitida_en
    ) VALUES (
        'ses_registro_v3_0000000000000000000000',
        'aut_registro_v3_0000000000000000000000',repeat('5',64),
        'ase_registro_v3_0000000000000000000000',
        'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',
        'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',false,'interna_corporativa',
        'certificado','alto','pga_registro_v3_0000000000000000000000',
        repeat('6',64),publicada,publicada
    );
    INSERT INTO vec_autorizacion.control_sesion_v1(
        control_sesion_ref,revision,sesion_ref,estado,huella_sha256,
        sesion_revalidada_en,sesion_valida_hasta
    ) VALUES (
        'cse_registro_v3_0000000000000000000000',1,
        'ses_registro_v3_0000000000000000000000','activa',repeat('7',64),
        publicada,valida
    );
    INSERT INTO vec_autorizacion.control_sesion_actual_v1 VALUES (
        'ses_registro_v3_0000000000000000000000',
        'cse_registro_v3_0000000000000000000000',1,ahora,
        'acto:sesion:registro-v3'
    );
END
$fixture$;
COMMIT;

BEGIN;
SET LOCAL ROLE vec_autorizacion_motivos_proyector;
SELECT vec_autorizacion.publicar_motivos_autorizacion_v2(
    'evento_33333333333333333333333333333333', 1, repeat('8',64),
    'motivos_v3', 1, repeat('9',64), clock_timestamp()-interval '1 minute',
    jsonb_build_array(jsonb_build_object(
        'clave','motivo_33333333333333333333333333333333',
        'vigente_desde',to_char(
            (clock_timestamp()-interval '1 minute') AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'vigente_hasta',NULL
    ))
);
COMMIT;
