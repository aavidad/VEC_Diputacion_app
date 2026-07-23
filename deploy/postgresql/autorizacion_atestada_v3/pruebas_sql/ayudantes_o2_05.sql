\set ON_ERROR_STOP 1

-- Superficie exclusiva de la base efímera. Se elimina antes de auditar ACL.
CREATE TABLE public.vectores_o2_05 (
    caso text PRIMARY KEY,
    capacidad bytea NOT NULL,
    decision bytea NOT NULL,
    motivo bytea NOT NULL,
    contexto bytea NOT NULL,
    persona_version numeric NOT NULL,
    perfil_version numeric NOT NULL,
    payload bytea NOT NULL,
    cose bytea NOT NULL,
    evidencia bytea NOT NULL,
    spki bytea NOT NULL,
    alta bytea NOT NULL,
    sellos bytea NOT NULL
);
REVOKE ALL ON TABLE public.vectores_o2_05 FROM PUBLIC;

CREATE FUNCTION public.preparar_vector_o2_05(
    p_caso text,
    p_variante text DEFAULT 'valido',
    p_clave_version numeric DEFAULT 1
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
    v_decision jsonb;
    v_capacidad jsonb;
    v_alta jsonb;
    v_sellos jsonb;
    v_decision_bytes bytea;
    v_capacidad_bytes bytea;
    v_motivo bytea;
    v_payload bytea;
    v_cose bytea;
    v_evidencia bytea;
    v_ahora timestamptz(6) := clock_timestamp();
    v_hasta timestamptz(6);
    v_emitida_z text;
    v_expira_z text;
    v_hasta_z text;
    v_ambito_v2 text;
    v_huella_v2 text;
    v_ambito_v1 text;
    v_huella_v1 text;
    v_contexto_recurso bytea;
    v_huella_contexto text;
    v_nonce text;
BEGIN
    IF p_caso !~ '^[a-z0-9_-]{3,32}$'
       OR p_variante NOT IN (
           'valido', 'efecto_cruzado', 'decision_cruzada',
           'expirada', 'alias_cruzado', 'rollback_configuracion',
           'rollback_raiz'
       ) THEN
        RAISE EXCEPTION 'caso de prueba inválido';
    END IF;
    SELECT documento, motivo_canonico
      INTO STRICT v_base
      FROM vec_autorizacion.decision_concedida_contexto_actor_v3
     WHERE decision_ref = 'decision:registro-v3:positiva';
    SELECT registro_contexto_ref, representacion_canonica,
           huella_sha256
      INTO STRICT v_contexto
      FROM vec_contexto_actor_v1.registros_contexto
     WHERE registro_contexto_ref =
           'rca_registro_v3_000000000000000000000000';
    SELECT * INTO STRICT v_clave
      FROM vec_autorizacion_atestada_v3.clave_capacidad_version
     WHERE version = p_clave_version;
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
     WHERE cr.configuracion_revision = v_config.revision
     ORDER BY r.version DESC LIMIT 1;

    v_ambito_v2 :=
      'hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v2:' ||
      encode(sha256(convert_to('ambito-v2:' || p_caso, 'UTF8')), 'hex');
    v_huella_v2 :=
      'hmac-sha256:vec.contratacion-temporal.huella-peticion/v2:' ||
      encode(sha256(convert_to('huella-v2:' || p_caso, 'UTF8')), 'hex');
    v_ambito_v1 :=
      'hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:' ||
      encode(sha256(convert_to('ambito-v1:' || p_caso, 'UTF8')), 'hex');
    v_huella_v1 :=
      'hmac-sha256:vec.contratacion-temporal.huella-peticion/v1:' ||
      encode(sha256(convert_to('huella-v1:' || p_caso, 'UTF8')), 'hex');
    v_contexto_recurso := convert_to(
        '{"ambitos":{"categoria_ref":"categoria:auxiliar","centro_ref":"centro:seleccion","organizacion_ref":"organizacion:dipgra"},"atributos":{"flujo_huella_sha256":"' ||
        repeat('7', 64) ||
        '","flujo_ref":"flujo:contratacion-temporal","flujo_version":"1","huella_peticion_hmac_activa":"' ||
        v_huella_v2 || '"}}',
        'UTF8'
    );
    v_huella_contexto := encode(sha256(v_contexto_recurso), 'hex');
    v_hasta := v_ahora + interval '2 minutes';
    IF p_variante = 'expirada' THEN
        v_ahora := v_ahora - interval '10 seconds';
    END IF;
    v_emitida_z := to_char(
        v_ahora AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    v_expira_z := to_char(
        (v_ahora + interval '5 seconds') AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    v_hasta_z := to_char(
        v_hasta AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    v_decision := v_base.documento;
    v_decision := jsonb_set(
        v_decision, '{decision_ref}',
        to_jsonb('decision:ct:o205:' || p_caso)
    );
    v_decision := jsonb_set(
        v_decision, '{accion}',
        '"contratacion_temporal.solicitud.crear"'
    );
    v_decision := jsonb_set(
        v_decision, '{recurso_ref}', to_jsonb(v_ambito_v2)
    );
    v_decision := jsonb_set(
        v_decision, '{modulo_id}', '"contratacion_temporal"'
    );
    v_decision := jsonb_set(
        v_decision, '{tipo_recurso}',
        '"expediente_contratacion_temporal"'
    );
    v_decision := jsonb_set(
        v_decision, '{contexto_recurso_huella_sha256}',
        to_jsonb(v_huella_contexto)
    );
    v_decision := jsonb_set(
        v_decision, '{finalidad}',
        '"tramitar_necesidad_personal_temporal"'
    );
    v_decision := jsonb_set(
        v_decision, '{correlacion_ref}',
        to_jsonb('correlacion_' || substring(encode(sha256(
            convert_to('correlacion:ct:o205:' || p_caso, 'UTF8')
        ), 'hex') FROM 1 FOR 32))
    );
    v_decision := jsonb_set(
        v_decision, '{solicitud_huella_sha256}',
        to_jsonb(encode(sha256(convert_to(
            'solicitud-autorizacion:' || p_caso, 'UTF8'
        )), 'hex'))
    );
    v_decision := jsonb_set(
        v_decision, '{asignacion_ref}',
        '"asignacion:registro_v3:v2"'
    );
    v_decision := jsonb_set(
        v_decision, '{asignacion_huella_sha256}',
        to_jsonb(repeat('e', 64))
    );
    v_decision := jsonb_set(
        v_decision, '{version_rol_ref}',
        '"rol:registro_ct_o205:v1"'
    );
    v_decision := jsonb_set(
        v_decision, '{version_rol_huella_sha256}',
        to_jsonb(repeat('c', 64))
    );
    v_decision := jsonb_set(
        v_decision, '{control_vigencia_version_rol_ref}',
        '"rol:registro_ct_o205:v1"'
    );
    v_decision := jsonb_set(
        v_decision, '{control_vigencia_version_rol_huella_sha256}',
        to_jsonb(repeat('d', 64))
    );
    v_decision := jsonb_set(
        v_decision, '{emitida_en}', to_jsonb(v_emitida_z)
    );
    v_decision := jsonb_set(
        v_decision, '{valida_hasta}', to_jsonb(v_hasta_z)
    );
    v_decision_bytes :=
        vec_autorizacion.decision_contexto_actor_v3_canonica(v_decision);
    v_motivo := v_base.motivo_canonico;
    v_payload := convert_to('VEC-AD-3:' || p_caso, 'UTF8');
    v_cose := convert_to('COSE-Sign1-sintetico:' || p_caso, 'UTF8');
    v_evidencia := convert_to(
        'prueba-confianza-sintetica:' || p_caso, 'UTF8'
    );
    v_nonce := encode(sha256(convert_to(
        'nonce:' || p_caso || ':' || v_emitida_z, 'UTF8'
    )), 'hex');

    v_capacidad := jsonb_build_object(
        'esquema',
          'vec.autorizacion.capacidad-registro-consumo-atestado.v3',
        'version', 3,
        'clave_id', v_clave.clave_id,
        'clave_version', v_clave.version,
        'revision_gobierno', v_clave.revision_gobierno,
        'huella_gobierno_sha256', v_clave.huella_gobierno_sha256,
        'emisor_id', v_clave.emisor_id,
        'audiencia_consumo', v_clave.audiencia_consumo,
        'nonce', v_nonce,
        'emitida_en', v_emitida_z,
        'expira_en', v_expira_z,
        'decision_ref', v_decision ->> 'decision_ref',
        'huella_decision_sha256',
          encode(sha256(v_decision_bytes), 'hex'),
        'huella_motivo_sha256', encode(sha256(v_motivo), 'hex'),
        'huella_payload_vec_ad_3_sha256',
          encode(sha256(v_payload), 'hex'),
        'huella_sobre_cose_sign1_sha256',
          encode(sha256(v_cose), 'hex'),
        'huella_prueba_confianza_sha256',
          encode(sha256(v_evidencia), 'hex'),
        'contexto_ref', v_contexto.registro_contexto_ref,
        'huella_contexto_sha256', v_contexto.huella_sha256,
        'audiencia_despliegue', v_raiz.audiencia_despliegue,
        'operacion', v_decision ->> 'accion',
        'efecto_ref', v_ambito_v2,
        'huella_efecto_sha256', v_huella_contexto,
        'decision_valida_hasta', v_hasta_z,
        'verificada_en', v_emitida_z,
        'revision_confianza', v_config.revision,
        'configuracion_secuencia', v_config.secuencia,
        'huella_configuracion_sha256',
          v_config.huella_configuracion_sha256,
        'configuracion_publicada_en', to_char(
            v_config.publicada_en AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'configuracion_expira_en', to_char(
            v_config.expira_en AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'raiz_clave_id', v_raiz.clave_id,
        'raiz_version', v_raiz.version,
        'huella_raiz_spki_sha256', v_raiz.huella_spki_sha256,
        'raiz_valida_desde', to_char(
            v_raiz.valida_desde AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'raiz_valida_hasta', to_char(
            v_raiz.valida_hasta AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'suite', v_raiz.suite,
        'mac_sha256', repeat('f', 64)
    );
    IF p_variante = 'efecto_cruzado' THEN
        v_capacidad := jsonb_set(
            v_capacidad, '{efecto_ref}',
            '"hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v2:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"'
        );
    ELSIF p_variante = 'decision_cruzada' THEN
        v_capacidad := jsonb_set(
            v_capacidad, '{decision_ref}',
            to_jsonb('decision:ct:o205:otra-' || p_caso)
        );
    ELSIF p_variante = 'rollback_configuracion' THEN
        v_capacidad := jsonb_set(
            v_capacidad, '{configuracion_secuencia}',
            to_jsonb(greatest(v_config.secuencia - 1, 1))
        );
    ELSIF p_variante = 'rollback_raiz' THEN
        v_capacidad := jsonb_set(
            v_capacidad, '{raiz_version}',
            to_jsonb(greatest(v_raiz.version - 1, 1))
        );
    END IF;
    v_capacidad := jsonb_set(
        v_capacidad, '{mac_sha256}', to_jsonb(encode(public.hmac(
            vec_autorizacion_atestada_v3.preimagen_mac(v_capacidad),
            v_clave.secreto_hmac, 'sha256'
        ), 'hex'))
    );
    v_capacidad_bytes :=
        vec_autorizacion_atestada_v3.capacidad_canonica(v_capacidad);

    v_alta := jsonb_build_object(
        'esquema', 'vec.contratacion-temporal.alta-persistencia.v1',
        'reserva_ref', 'reserva:ct:o205:' || p_caso,
        'expediente_ref', 'expediente:ct:o205:' || p_caso,
        'numero_visible', '2026/' || p_caso,
        'recibo_ref', 'recibo:ct:o205:' || p_caso,
        'organizacion_ref', 'organizacion:dipgra',
        'centro_ref', 'centro:seleccion',
        'categoria_ref', 'categoria:auxiliar',
        'actor_ref', 'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
        'perfil_ref', 'prf_sintetico_cccccccccccccccccccccccc',
        'version', 1,
        'flujo_ref', 'flujo:contratacion-temporal',
        'flujo_version', 1,
        'flujo_huella_sha256', repeat('7', 64),
        'fase_clave', 'solicitud_registrada',
        'estado', 'en_curso',
        'solicitud_huella_sha256', encode(sha256(convert_to(
            'solicitud-centro:' || p_caso, 'UTF8'
        )), 'hex'),
        'accion_clave', 'registrar_solicitud',
        'unidad_ref', 'unidad:seleccion',
        'realizada_en', v_emitida_z
    );
    v_sellos := jsonb_build_object(
        'esquema', 'vec.contratacion-temporal.sellos-hmac.v1',
        'activo', jsonb_build_object(
            'generacion', 2, 'ambito_hmac', v_ambito_v2,
            'huella_hmac', CASE
                WHEN p_variante = 'alias_cruzado'
                THEN v_huella_v1
                ELSE v_huella_v2
            END
        ),
        'retenidos', jsonb_build_array(jsonb_build_object(
            'generacion', 1, 'ambito_hmac', v_ambito_v1,
            'huella_hmac', CASE
                WHEN p_variante = 'alias_cruzado'
                THEN v_huella_v2
                ELSE v_huella_v1
            END
        ))
    );
    DELETE FROM public.vectores_o2_05 WHERE caso = p_caso;
    INSERT INTO public.vectores_o2_05 VALUES (
        p_caso, v_capacidad_bytes, v_decision_bytes, v_motivo,
        v_contexto.representacion_canonica, 2, 2, v_payload,
        v_cose, v_evidencia, v_raiz.clave_publica_spki,
        vec_contratacion_temporal.reconstruir_alta_v1(v_alta),
        vec_contratacion_temporal.reconstruir_sellos_hmac_v1(v_sellos)
    );
END
$funcion$;

CREATE FUNCTION public.invocar_vector_o2_05(p_caso text)
RETURNS TABLE (
    expediente_ref text,
    numero_visible text,
    version numeric,
    recibo_ref text,
    auditoria_ref text,
    evento_ref text,
    confirmada_en timestamptz,
    recibo_huella_sha256 text
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v public.vectores_o2_05%ROWTYPE;
BEGIN
    SELECT * INTO STRICT v
      FROM public.vectores_o2_05
     WHERE caso = p_caso;
    RETURN QUERY
    SELECT *
      FROM vec_contratacion_temporal.confirmar_alta_atestada_v1(
          v.capacidad, v.decision, v.motivo, v.contexto,
          v.persona_version, v.perfil_version, v.payload, v.cose,
          v.evidencia, v.spki, v.alta, v.sellos
      );
END
$funcion$;

REVOKE ALL ON FUNCTION public.preparar_vector_o2_05(
    text, text, numeric
) FROM PUBLIC;
REVOKE ALL ON FUNCTION public.invocar_vector_o2_05(text) FROM PUBLIC;
