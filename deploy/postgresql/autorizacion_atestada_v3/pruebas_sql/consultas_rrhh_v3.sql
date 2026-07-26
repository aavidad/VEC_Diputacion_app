\set ON_ERROR_STOP 1

BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $rbac$
DECLARE
    v_ahora timestamptz(6) := pg_catalog.clock_timestamp();
    v_desde timestamptz(6) := v_ahora - interval '10 minutes';
    v_hasta timestamptz(6) := v_ahora + interval '2 hours';
    v_rol jsonb;
    v_control jsonb;
    v_asignacion jsonb;
BEGIN
    v_rol := pg_catalog.jsonb_build_object(
        'rol_id', 'consulta_rrhh_v3', 'version', 1,
        'nombre', 'Consulta RRHH V3 sintética', 'estado', 'publicada',
        'concesiones', pg_catalog.jsonb_build_array(
            pg_catalog.jsonb_build_object(
                'accion', 'contratacion_temporal.cuadro.consultar',
                'modulo_id', 'contratacion_temporal',
                'tipo_recurso', 'cuadro_rrhh_contratacion_temporal',
                'finalidades', pg_catalog.jsonb_build_array(
                    'gestion_operativa_contratacion_temporal'
                ),
                'garantia_minima', 'alto',
                'campos_permitidos', pg_catalog.jsonb_build_array('estado'),
                'obligaciones', pg_catalog.jsonb_build_array('auditar')
            ),
            pg_catalog.jsonb_build_object(
                'accion', 'contratacion_temporal.expediente.consultar',
                'modulo_id', 'contratacion_temporal',
                'tipo_recurso', 'expediente_contratacion_temporal',
                'finalidades', pg_catalog.jsonb_build_array(
                    'tramitacion_expediente_contratacion_temporal'
                ),
                'garantia_minima', 'alto',
                'campos_permitidos', pg_catalog.jsonb_build_array('estado'),
                'obligaciones', pg_catalog.jsonb_build_array('auditar')
            )
        ),
        'publicada_por', 'usr_seguridad_sintetico_consulta_rrhh',
        'publicada_en', pg_catalog.to_char(
            v_desde AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )
    );
    INSERT INTO vec_autorizacion.version_rol(
        version_rol_ref, rol_id, version, huella_sha256,
        publicada_en, documento
    ) VALUES (
        'rol:consulta_rrhh_v3:v1', 'consulta_rrhh_v3', 1,
        pg_catalog.repeat('4', 64), v_desde, v_rol
    );
    v_control := pg_catalog.jsonb_build_object(
        'version_rol_ref', 'rol:consulta_rrhh_v3:v1',
        'revision', 1, 'estado', 'habilitada',
        'actualizado_por', 'usr_seguridad_sintetico_consulta_rrhh',
        'actualizado_en', pg_catalog.to_char(
            v_desde AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )
    );
    INSERT INTO vec_autorizacion.control_vigencia_version_rol(
        version_rol_ref, revision, estado, huella_sha256,
        actualizado_en, documento
    ) VALUES (
        'rol:consulta_rrhh_v3:v1', 1, 'habilitada',
        pg_catalog.repeat('5', 64), v_desde, v_control
    );
    INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
        VALUES (
            'rol:consulta_rrhh_v3:v1', 1, v_ahora,
            'usr_seguridad_sintetico_consulta_rrhh',
            'acto:control:consulta-rrhh-v3'
        );
    v_asignacion := pg_catalog.jsonb_build_object(
        'asignacion_id', 'registro_v3', 'version', 3,
        'perfil_activo_ref',
            'prf_sintetico_cccccccccccccccccccccccc',
        'principal_id',
            'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
        'version_rol_ref', 'rol:consulta_rrhh_v3:v1',
        'estado', 'activa',
        'emitida_en', pg_catalog.to_char(
            v_desde AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'vigente_desde', pg_catalog.to_char(
            v_desde AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'vigente_hasta', pg_catalog.to_char(
            v_hasta AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'ambitos', pg_catalog.jsonb_build_array(
            pg_catalog.jsonb_build_object(
                'clave', 'unidad',
                'valores', pg_catalog.jsonb_build_array('seleccion')
            )
        )
    );
    INSERT INTO vec_autorizacion.asignacion_perfil(
        asignacion_ref, asignacion_id, version, perfil_activo_ref,
        principal_id, version_rol_ref, huella_sha256,
        emitida_en, documento
    ) VALUES (
        'asignacion:registro_v3:v3',
        'registro_v3', 3,
        'prf_sintetico_cccccccccccccccccccccccc',
        'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
        'rol:consulta_rrhh_v3:v1', pg_catalog.repeat('6', 64),
        v_desde, v_asignacion
    );
    UPDATE vec_autorizacion.asignacion_perfil_actual
       SET asignacion_ref = 'asignacion:registro_v3:v3',
           actualizada_en = v_ahora,
           actualizada_por = 'usr_rrhh_sintetico_consulta_rrhh',
           acto_ref = 'acto:asignacion:consulta-rrhh-v3'
     WHERE perfil_activo_ref =
           'prf_sintetico_cccccccccccccccccccccccc';
    IF NOT FOUND THEN
        RAISE EXCEPTION 'fixture RBAC de consulta RRHH incompleto';
    END IF;
END
$rbac$;
COMMIT;

BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $claves$
DECLARE
    v_ahora timestamptz(6) := pg_catalog.clock_timestamp();
    v_secreto_cuadro bytea := public.gen_random_bytes(32);
    v_secreto_detalle bytea := public.gen_random_bytes(32);
BEGIN
    INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version(
        clave_id, version, revision_gobierno,
        huella_gobierno_sha256, secreto_hmac,
        huella_secreto_sha256, emisor_id, audiencia_consumo,
        valida_desde, valida_hasta, acto_ref
    ) VALUES
    (
        'clave-consulta-cuadro-rrhh-v3', 2, 2,
        pg_catalog.repeat('7', 64), v_secreto_cuadro,
        pg_catalog.encode(pg_catalog.sha256(v_secreto_cuadro), 'hex'),
        'broker-consulta-rrhh-sintetico',
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
        v_ahora - interval '1 hour', v_ahora + interval '2 hours',
        'acto:clave:consulta-cuadro-rrhh-v3'
    ),
    (
        'clave-consulta-detalle-rrhh-v3', 3, 3,
        pg_catalog.repeat('8', 64), v_secreto_detalle,
        pg_catalog.encode(pg_catalog.sha256(v_secreto_detalle), 'hex'),
        'broker-consulta-rrhh-sintetico',
        'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1',
        v_ahora - interval '1 hour', v_ahora + interval '2 hours',
        'acto:clave:consulta-detalle-rrhh-v3'
    );
    INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision(
        orden, clave_id, version, establecida_en, acto_ref
    ) VALUES
    (
        2, 'clave-consulta-cuadro-rrhh-v3', 2,
        v_ahora - interval '1 minute',
        'acto:puntero:consulta-cuadro-rrhh-v3'
    ),
    (
        3, 'clave-consulta-detalle-rrhh-v3', 3,
        v_ahora - interval '1 minute',
        'acto:puntero:consulta-detalle-rrhh-v3'
    );
END
$claves$;
COMMIT;

CREATE TABLE public.vectores_consulta_rrhh_v3 (
    caso text PRIMARY KEY,
    perfil text NOT NULL,
    capacidad bytea NOT NULL,
    decision bytea NOT NULL,
    motivo bytea NOT NULL,
    contexto bytea NOT NULL,
    persona_version numeric NOT NULL,
    perfil_version numeric NOT NULL,
    payload bytea NOT NULL,
    cose bytea NOT NULL,
    evidencia bytea NOT NULL,
    spki bytea NOT NULL
);
REVOKE ALL ON TABLE public.vectores_consulta_rrhh_v3 FROM PUBLIC;

CREATE FUNCTION public.preparar_vector_consulta_rrhh_v3(
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
    v_base record;
    v_contexto record;
    v_clave record;
    v_config record;
    v_raiz record;
    v_rbac record;
    v_decision jsonb;
    v_capacidad jsonb;
    v_decision_bytes bytea;
    v_motivo bytea;
    v_payload bytea;
    v_cose bytea;
    v_evidencia bytea;
    v_ahora timestamptz(6) := pg_catalog.clock_timestamp();
    v_emitida_z text;
    v_expira_z text;
    v_hasta_z text;
    v_accion text;
    v_audiencia text;
    v_tipo text;
    v_finalidad text;
    v_recurso text;
    v_huella_recurso text;
BEGIN
    IF p_caso !~ '^[a-z0-9_-]{3,48}$'
       OR p_perfil NOT IN ('cuadro', 'detalle') THEN
        RAISE EXCEPTION 'vector de consulta RRHH inválido';
    END IF;
    IF p_perfil = 'cuadro' THEN
        v_accion := 'contratacion_temporal.cuadro.consultar';
        v_audiencia :=
            'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1';
        v_tipo := 'cuadro_rrhh_contratacion_temporal';
        v_finalidad := 'gestion_operativa_contratacion_temporal';
        v_recurso := 'unidad:seleccion';
    ELSE
        v_accion := 'contratacion_temporal.expediente.consultar';
        v_audiencia :=
            'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1';
        v_tipo := 'expediente_contratacion_temporal';
        v_finalidad := 'tramitacion_expediente_contratacion_temporal';
        v_recurso := 'expediente:ct:consulta:' || p_caso;
    END IF;
    v_huella_recurso := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(
            'contexto-recurso:' || p_perfil || ':' || p_caso, 'UTF8'
        )
    ), 'hex');
    SELECT documento, motivo_canonico
      INTO STRICT v_base
      FROM vec_autorizacion.decision_concedida_contexto_actor_v3
     WHERE decision_ref = 'decision:registro-v3:positiva';
    SELECT registro_contexto_ref, representacion_canonica, huella_sha256
      INTO STRICT v_contexto
      FROM vec_contexto_actor_v1.registros_contexto
     WHERE registro_contexto_ref =
           'rca_registro_v3_000000000000000000000000';
    SELECT * INTO STRICT v_clave
      FROM vec_autorizacion_atestada_v3.clave_capacidad_version
     WHERE audiencia_consumo = v_audiencia;
    SELECT c.* INTO STRICT v_config
      FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p
      JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version c
        ON c.revision = p.configuracion_revision
     ORDER BY p.orden DESC LIMIT 1;
    SELECT r.* INTO STRICT v_raiz
      FROM vec_autorizacion_atestada_v3.configuracion_raiz cr
      JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r
        ON r.clave_id = cr.raiz_clave_id
       AND r.version = cr.raiz_version
     WHERE cr.configuracion_revision = v_config.revision;
    SELECT a.asignacion_ref, a.huella_sha256 AS asignacion_huella,
           a.version_rol_ref, rol.huella_sha256 AS rol_huella,
           control.revision AS control_revision,
           control.huella_sha256 AS control_huella
      INTO STRICT v_rbac
      FROM vec_autorizacion.asignacion_perfil_actual actual
      JOIN vec_autorizacion.asignacion_perfil a
        ON a.perfil_activo_ref = actual.perfil_activo_ref
       AND a.asignacion_ref = actual.asignacion_ref
      JOIN vec_autorizacion.version_rol rol
        ON rol.version_rol_ref = a.version_rol_ref
      JOIN vec_autorizacion.control_vigencia_version_rol_actual ca
        ON ca.version_rol_ref = rol.version_rol_ref
      JOIN vec_autorizacion.control_vigencia_version_rol control
        ON control.version_rol_ref = ca.version_rol_ref
       AND control.revision = ca.revision
     WHERE actual.perfil_activo_ref =
           'prf_sintetico_cccccccccccccccccccccccc';
    v_emitida_z := pg_catalog.to_char(
        v_ahora AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    v_expira_z := pg_catalog.to_char(
        (v_ahora + interval '5 seconds') AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    v_hasta_z := pg_catalog.to_char(
        (v_ahora + interval '2 minutes') AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    v_decision := v_base.documento;
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{decision_ref}',
        pg_catalog.to_jsonb('decision:consulta-rrhh:' || p_caso)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{accion}', pg_catalog.to_jsonb(v_accion)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{recurso_ref}', pg_catalog.to_jsonb(v_recurso)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{modulo_id}', '"contratacion_temporal"'
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{tipo_recurso}', pg_catalog.to_jsonb(v_tipo)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{contexto_recurso_huella_sha256}',
        pg_catalog.to_jsonb(v_huella_recurso)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{finalidad}', pg_catalog.to_jsonb(v_finalidad)
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
        pg_catalog.to_jsonb(v_rbac.asignacion_ref)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{asignacion_huella_sha256}',
        pg_catalog.to_jsonb(v_rbac.asignacion_huella)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{version_rol_ref}',
        pg_catalog.to_jsonb(v_rbac.version_rol_ref)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{version_rol_huella_sha256}',
        pg_catalog.to_jsonb(v_rbac.rol_huella)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{control_vigencia_version_rol_ref}',
        pg_catalog.to_jsonb(v_rbac.version_rol_ref)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{control_vigencia_version_rol_revision}',
        pg_catalog.to_jsonb(v_rbac.control_revision)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{control_vigencia_version_rol_huella_sha256}',
        pg_catalog.to_jsonb(v_rbac.control_huella)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{emitida_en}', pg_catalog.to_jsonb(v_emitida_z)
    );
    v_decision := pg_catalog.jsonb_set(
        v_decision, '{valida_hasta}', pg_catalog.to_jsonb(v_hasta_z)
    );
    v_decision_bytes :=
        vec_autorizacion.decision_contexto_actor_v3_canonica(v_decision);
    v_motivo := v_base.motivo_canonico;
    v_payload := pg_catalog.convert_to(
        'VEC-AD-3-consulta:' || p_caso, 'UTF8'
    );
    v_cose := pg_catalog.convert_to(
        'COSE-Sign1-sintetico:' || p_caso, 'UTF8'
    );
    v_evidencia := pg_catalog.convert_to(
        'evidencia-confianza-sintetica:' || p_caso, 'UTF8'
    );
    v_capacidad := pg_catalog.jsonb_build_object(
        'esquema',
          'vec.autorizacion.capacidad-registro-consumo-atestado.v3',
        'version', 3, 'clave_id', v_clave.clave_id,
        'clave_version', v_clave.version,
        'revision_gobierno', v_clave.revision_gobierno,
        'huella_gobierno_sha256', v_clave.huella_gobierno_sha256,
        'emisor_id', v_clave.emisor_id,
        'audiencia_consumo', v_audiencia,
        'nonce', pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to(
                'nonce:' || p_caso || ':' || v_emitida_z, 'UTF8'
            )
        ), 'hex'),
        'emitida_en', v_emitida_z, 'expira_en', v_expira_z,
        'decision_ref', v_decision ->> 'decision_ref',
        'huella_decision_sha256',
          pg_catalog.encode(pg_catalog.sha256(v_decision_bytes), 'hex'),
        'huella_motivo_sha256',
          pg_catalog.encode(pg_catalog.sha256(v_motivo), 'hex'),
        'huella_payload_vec_ad_3_sha256',
          pg_catalog.encode(pg_catalog.sha256(v_payload), 'hex'),
        'huella_sobre_cose_sign1_sha256',
          pg_catalog.encode(pg_catalog.sha256(v_cose), 'hex'),
        'huella_prueba_confianza_sha256',
          pg_catalog.encode(pg_catalog.sha256(v_evidencia), 'hex'),
        'contexto_ref', v_contexto.registro_contexto_ref,
        'huella_contexto_sha256', v_contexto.huella_sha256,
        'audiencia_despliegue', v_raiz.audiencia_despliegue,
        'operacion', v_accion, 'efecto_ref', v_recurso,
        'huella_efecto_sha256', v_huella_recurso,
        'decision_valida_hasta', v_hasta_z,
        'verificada_en', v_emitida_z,
        'revision_confianza', v_config.revision,
        'configuracion_secuencia', v_config.secuencia,
        'huella_configuracion_sha256',
          v_config.huella_configuracion_sha256,
        'configuracion_publicada_en', pg_catalog.to_char(
            v_config.publicada_en AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'configuracion_expira_en', pg_catalog.to_char(
            v_config.expira_en AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'raiz_clave_id', v_raiz.clave_id,
        'raiz_version', v_raiz.version,
        'huella_raiz_spki_sha256', v_raiz.huella_spki_sha256,
        'raiz_valida_desde', pg_catalog.to_char(
            v_raiz.valida_desde AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'raiz_valida_hasta', pg_catalog.to_char(
            v_raiz.valida_hasta AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'suite', v_raiz.suite, 'mac_sha256', pg_catalog.repeat('f', 64)
    );
    v_capacidad := pg_catalog.jsonb_set(
        v_capacidad, '{mac_sha256}',
        pg_catalog.to_jsonb(pg_catalog.encode(public.hmac(
            vec_autorizacion_atestada_v3.preimagen_mac(v_capacidad),
            v_clave.secreto_hmac, 'sha256'
        ), 'hex'))
    );
    DELETE FROM public.vectores_consulta_rrhh_v3 WHERE caso = p_caso;
    INSERT INTO public.vectores_consulta_rrhh_v3 VALUES (
        p_caso, p_perfil,
        vec_autorizacion_atestada_v3.capacidad_canonica(v_capacidad),
        v_decision_bytes, v_motivo, v_contexto.representacion_canonica,
        2, 2, v_payload, v_cose, v_evidencia,
        v_raiz.clave_publica_spki
    );
END
$funcion$;

CREATE FUNCTION public.adulterar_decision_consulta_rrhh_v3(
    p_caso text,
    p_campo text,
    p_valor text
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    d jsonb;
BEGIN
    IF p_campo NOT IN (
        'accion', 'modulo_id', 'tipo_recurso', 'finalidad'
    ) THEN
        RAISE EXCEPTION 'campo de decisión no permitido';
    END IF;
    SELECT pg_catalog.convert_from(decision, 'UTF8')::jsonb
      INTO STRICT d
      FROM public.vectores_consulta_rrhh_v3
     WHERE caso = p_caso;
    d := pg_catalog.jsonb_set(
        d, ARRAY[p_campo], pg_catalog.to_jsonb(p_valor), false
    );
    UPDATE public.vectores_consulta_rrhh_v3
       SET decision =
           vec_autorizacion.decision_contexto_actor_v3_canonica(d)
     WHERE caso = p_caso;
END
$funcion$;

CREATE FUNCTION public.adulterar_capacidad_consulta_rrhh_v3(
    p_caso text,
    p_campo text,
    p_valor text
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    c jsonb;
BEGIN
    IF p_campo NOT IN ('audiencia_consumo', 'operacion') THEN
        RAISE EXCEPTION 'campo de capacidad no permitido';
    END IF;
    SELECT pg_catalog.convert_from(capacidad, 'UTF8')::jsonb
      INTO STRICT c
      FROM public.vectores_consulta_rrhh_v3
     WHERE caso = p_caso;
    c := pg_catalog.jsonb_set(
        c, ARRAY[p_campo], pg_catalog.to_jsonb(p_valor), false
    );
    UPDATE public.vectores_consulta_rrhh_v3
       SET capacidad =
           vec_autorizacion_atestada_v3.capacidad_canonica(c)
     WHERE caso = p_caso;
END
$funcion$;

CREATE FUNCTION public.crear_colision_replay_consulta_rrhh_v3(
    p_destino text,
    p_decision text,
    p_nonce text
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v public.vectores_consulta_rrhh_v3%ROWTYPE;
    v_nonce_cruzado text;
    c jsonb;
    k record;
BEGIN
    SELECT * INTO STRICT v
      FROM public.vectores_consulta_rrhh_v3
     WHERE caso = p_decision;
    SELECT pg_catalog.convert_from(capacidad, 'UTF8')::jsonb ->> 'nonce'
      INTO STRICT v_nonce_cruzado
      FROM public.vectores_consulta_rrhh_v3
     WHERE caso = p_nonce;
    c := pg_catalog.convert_from(v.capacidad, 'UTF8')::jsonb;
    c := pg_catalog.jsonb_set(
        c, '{nonce}', pg_catalog.to_jsonb(v_nonce_cruzado), false
    );
    SELECT * INTO STRICT k
      FROM vec_autorizacion_atestada_v3.clave_capacidad_version
     WHERE clave_id = c ->> 'clave_id'
       AND version = (c ->> 'clave_version')::numeric;
    c := pg_catalog.jsonb_set(
        c, '{mac_sha256}', pg_catalog.to_jsonb(pg_catalog.encode(public.hmac(
            vec_autorizacion_atestada_v3.preimagen_mac(c),
            k.secreto_hmac, 'sha256'
        ), 'hex')), false
    );
    v.caso := p_destino;
    v.capacidad :=
        vec_autorizacion_atestada_v3.capacidad_canonica(c);
    INSERT INTO public.vectores_consulta_rrhh_v3 VALUES (
        v.caso, v.perfil, v.capacidad, v.decision, v.motivo,
        v.contexto, v.persona_version, v.perfil_version,
        v.payload, v.cose, v.evidencia, v.spki
    );
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.prueba_consumir_consulta_rrhh_v3(
    p_caso text,
    p_fachada text,
    p_pieza text DEFAULT ''
)
RETURNS TABLE (
    decision_ref text,
    efecto_ref text,
    huella_efecto_sha256 text,
    consumo_huella_sha256 text,
    auditoria_ref text,
    auditoria_huella_sha256 text,
    consumida_en timestamptz,
    consumo_nuevo boolean
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v public.vectores_consulta_rrhh_v3%ROWTYPE;
BEGIN
    SELECT * INTO STRICT v
      FROM public.vectores_consulta_rrhh_v3
     WHERE caso = p_caso;
    CASE p_pieza
        WHEN '' THEN NULL;
        WHEN 'capacidad' THEN
            v.capacidad := pg_catalog.set_byte(
                v.capacidad, pg_catalog.octet_length(v.capacidad) - 1, 93
            );
        WHEN 'decision' THEN
            v.decision := pg_catalog.set_byte(
                v.decision, pg_catalog.octet_length(v.decision) - 1, 93
            );
        WHEN 'decision_null' THEN
            v.decision := NULL;
        WHEN 'motivo' THEN
            v.motivo := pg_catalog.set_byte(v.motivo, 0, 91);
        WHEN 'contexto' THEN
            v.contexto := pg_catalog.set_byte(v.contexto, 0, 91);
        WHEN 'persona_version' THEN
            v.persona_version := v.persona_version + 1;
        WHEN 'perfil_version' THEN
            v.perfil_version := v.perfil_version + 1;
        WHEN 'payload' THEN
            v.payload := pg_catalog.set_byte(v.payload, 0, 91);
        WHEN 'cose' THEN
            v.cose := pg_catalog.set_byte(v.cose, 0, 91);
        WHEN 'evidencia' THEN
            v.evidencia := pg_catalog.set_byte(v.evidencia, 0, 91);
        WHEN 'spki' THEN
            v.spki := pg_catalog.set_byte(v.spki, 43, 255);
        WHEN 'spki_null' THEN
            v.spki := NULL;
        WHEN 'spki_x25519' THEN
            v.spki := pg_catalog.decode(
                '302a300506032b656e032100' || pg_catalog.repeat('11', 32),
                'hex'
            );
        WHEN 'spki_der_no_canonico' THEN
            v.spki := pg_catalog.decode(
                '302a300506032b6570032101' || pg_catalog.repeat('11', 32),
                'hex'
            );
        WHEN 'spki_rsa' THEN
            v.spki := pg_catalog.decode(
                '303a300d06092a864886f70d0101010500032900' ||
                pg_catalog.repeat('11', 40),
                'hex'
            );
        ELSE
            RAISE EXCEPTION 'pieza de prueba desconocida';
    END CASE;
    IF p_fachada = 'cuadro' THEN
        RETURN QUERY
        SELECT *
          FROM vec_autorizacion_atestada_v3
               .registrar_y_consumir_consulta_cuadro_rrhh_v3_atestada(
              v.capacidad, v.decision, v.motivo, v.contexto,
              v.persona_version, v.perfil_version,
              v.payload, v.cose, v.evidencia, v.spki
          );
    ELSIF p_fachada = 'detalle' THEN
        RETURN QUERY
        SELECT *
          FROM vec_autorizacion_atestada_v3
               .registrar_y_consumir_consulta_detalle_rrhh_v3_atestada(
              v.capacidad, v.decision, v.motivo, v.contexto,
              v.persona_version, v.perfil_version,
              v.payload, v.cose, v.evidencia, v.spki
          );
    ELSE
        RAISE EXCEPTION 'fachada de prueba desconocida';
    END IF;
END
$funcion$;

ALTER FUNCTION vec_contratacion_temporal.prueba_consumir_consulta_rrhh_v3(
    text, text, text
) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.prueba_consumir_consulta_rrhh_v3(
        text, text, text
    ) FROM PUBLIC;
REVOKE ALL ON FUNCTION
    public.preparar_vector_consulta_rrhh_v3(text, text),
    public.adulterar_decision_consulta_rrhh_v3(text, text, text),
    public.adulterar_capacidad_consulta_rrhh_v3(text, text, text),
    public.crear_colision_replay_consulta_rrhh_v3(text, text, text)
    FROM PUBLIC;
GRANT USAGE ON SCHEMA public
    TO vec_contratacion_temporal_propietario;
GRANT USAGE ON SCHEMA vec_contratacion_temporal
    TO vec_contratacion_temporal_consultor_rrhh;
GRANT SELECT ON public.vectores_consulta_rrhh_v3
    TO vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.prueba_consumir_consulta_rrhh_v3(
        text, text, text
    ) TO vec_contratacion_temporal_consultor_rrhh;
