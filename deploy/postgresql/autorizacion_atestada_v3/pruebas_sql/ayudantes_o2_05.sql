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
    v_rbac record;
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
    v_huella_alta text;
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
     WHERE p.establecida_en <= clock_timestamp()
     ORDER BY p.orden DESC LIMIT 1;
    SELECT r.* INTO STRICT v_raiz
      FROM vec_autorizacion_atestada_v3.configuracion_raiz cr
      JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r
        ON r.clave_id = cr.raiz_clave_id
       AND r.version = cr.raiz_version
     WHERE cr.configuracion_revision = v_config.revision
     ORDER BY r.version DESC LIMIT 1;
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
    v_alta := jsonb_build_object(
        'esquema', 'vec.contratacion-temporal.efecto-alta.v2',
        'reserva_ref', 'reserva:ct:o205:' || p_caso,
        'expediente_ref', 'expediente:ct:o205:' || p_caso,
        'numero_visible', '2026/' || p_caso,
        'recibo_ref', 'recibo:ct:o205:' || p_caso,
        'organizacion_ref', 'organizacion:dipgra',
        'actor_ref', 'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
        'perfil_ref', 'prf_sintetico_cccccccccccccccccccccccc',
        'version', 1,
        'flujo', jsonb_build_object(
            'definicion_ref', 'flujo:contratacion-temporal',
            'version', 1,
            'huella_sha256', repeat('7', 64)
        ),
        'fase_actual', 'solicitud_registrada',
        'estado_actual', 'en_curso',
        'solicitud', jsonb_build_object(
            'centro_ref', 'centro:seleccion',
            'contacto_ref', 'contacto:seleccion',
            'categoria_ref', 'categoria:auxiliar',
            'grupo_subgrupo', 'C2',
            'motivo_clave', 'acumulacion_tareas',
            'detalle', 'Necesidad temporal de prueba O2-05',
            'periodo', jsonb_build_object(
                'inicio', '2026-08-01T00:00:00Z',
                'fin', '2026-08-31T00:00:00Z'
            ),
            'rc', jsonb_build_object(
                'existe', false,
                'numero', '',
                'fecha', '0001-01-01T00:00:00Z',
                'importe', jsonb_build_object(
                    'centimos', 0,
                    'moneda', ''
                ),
                'documento_ref', ''
            ),
            'documentos_adjuntos', jsonb_build_array(),
            'observaciones', ''
        ),
        'creado_en', v_emitida_z,
        'actualizado_en', v_emitida_z,
        'actuacion', jsonb_build_object(
            'secuencia', 1,
            'version_expediente', 1,
            'accion_clave', 'registrar_solicitud',
            'actor_ref', 'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
            'unidad_ref', 'unidad:seleccion',
            'recibo_ref', 'recibo:ct:o205:' || p_caso,
            'realizada_en', v_emitida_z,
            'fase_origen', '',
            'fase_destino', 'solicitud_registrada',
            'estado_origen', 'pendiente',
            'estado_destino', 'en_curso',
            'observaciones', '',
            'documentos_ref', jsonb_build_array()
        )
    );
    v_huella_alta := encode(sha256(
        vec_contratacion_temporal.reconstruir_efecto_alta_v2(v_alta)
    ), 'hex');
    v_contexto_recurso := convert_to(
        '{"ambitos":{"categoria_ref":"categoria:auxiliar","centro_ref":"centro:seleccion","organizacion_ref":"organizacion:dipgra"},"atributos":{"efecto_huella_sha256":"' ||
        v_huella_alta || '","flujo_huella_sha256":"' ||
        repeat('7', 64) ||
        '","flujo_ref":"flujo:contratacion-temporal","flujo_version":"1","huella_peticion_hmac_activa":"' ||
        v_huella_v2 || '"}}',
        'UTF8'
    );
    v_huella_contexto := encode(sha256(v_contexto_recurso), 'hex');
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
        to_jsonb(v_rbac.asignacion_ref)
    );
    v_decision := jsonb_set(
        v_decision, '{asignacion_huella_sha256}',
        to_jsonb(v_rbac.asignacion_huella)
    );
    v_decision := jsonb_set(
        v_decision, '{version_rol_ref}',
        to_jsonb(v_rbac.version_rol_ref)
    );
    v_decision := jsonb_set(
        v_decision, '{version_rol_huella_sha256}',
        to_jsonb(v_rbac.rol_huella)
    );
    v_decision := jsonb_set(
        v_decision, '{control_vigencia_version_rol_ref}',
        to_jsonb(v_rbac.version_rol_ref)
    );
    v_decision := jsonb_set(
        v_decision, '{control_vigencia_version_rol_revision}',
        to_jsonb(v_rbac.control_revision)
    );
    v_decision := jsonb_set(
        v_decision, '{control_vigencia_version_rol_huella_sha256}',
        to_jsonb(v_rbac.control_huella)
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
        vec_contratacion_temporal.reconstruir_efecto_alta_v2(v_alta),
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

CREATE FUNCTION public.mutar_efecto_o2_05(
    p_caso text,
    p_ruta text,
    p_valor jsonb
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_alta jsonb;
BEGIN
    SELECT pg_catalog.convert_from(alta, 'UTF8')::jsonb
      INTO STRICT v_alta
      FROM public.vectores_o2_05
     WHERE caso = p_caso;
    v_alta := pg_catalog.jsonb_set(
        v_alta, pg_catalog.string_to_array(p_ruta, '.'), p_valor, false
    );
    UPDATE public.vectores_o2_05
       SET alta =
           vec_contratacion_temporal.reconstruir_efecto_alta_v2(v_alta)
     WHERE caso = p_caso;
END
$funcion$;

CREATE FUNCTION public.mutar_tipo_capacidad_o2_05(
    p_caso text,
    p_campo text
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_capacidad jsonb;
BEGIN
    SELECT pg_catalog.convert_from(capacidad, 'UTF8')::jsonb
      INTO STRICT v_capacidad
      FROM public.vectores_o2_05
     WHERE caso = p_caso;
    v_capacidad := pg_catalog.jsonb_set(
        v_capacidad, ARRAY[p_campo],
        pg_catalog.to_jsonb(v_capacidad ->> p_campo), false
    );
    UPDATE public.vectores_o2_05
       SET capacidad =
           vec_autorizacion_atestada_v3.capacidad_canonica(v_capacidad)
     WHERE caso = p_caso;
END
$funcion$;

CREATE FUNCTION public.durabilizar_decision_o2_05(p_caso text)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v public.vectores_o2_05%ROWTYPE;
    r record;
BEGIN
    SELECT * INTO STRICT v
      FROM public.vectores_o2_05
     WHERE caso = p_caso;
    SELECT * INTO r
      FROM vec_autorizacion.registrar_decision_contexto_actor_v3(
          v.decision, v.motivo, v.persona_version, v.perfil_version
      );
    IF NOT FOUND OR r.concedida IS NOT TRUE THEN
        RAISE EXCEPTION 'no se pudo durabilizar decisión O2-05';
    END IF;
END
$funcion$;

CREATE FUNCTION public.exportar_entrada_go_o2_05(p_caso text)
RETURNS text
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v public.vectores_o2_05%ROWTYPE;
    c jsonb;
    d jsonb;
    k record;
    r record;
    contexto record;
    catalogo record;
    asignacion_actual record;
    politicas jsonb;
    ahora timestamptz(6) := clock_timestamp();
    secuencia numeric;
    version_raiz numeric;
BEGIN
    SELECT * INTO STRICT v
      FROM public.vectores_o2_05
     WHERE caso = p_caso;
    c := convert_from(v.capacidad, 'UTF8')::jsonb;
    d := convert_from(v.decision, 'UTF8')::jsonb;
    SELECT * INTO STRICT k
      FROM vec_autorizacion_atestada_v3.clave_capacidad_version
     WHERE clave_id = c ->> 'clave_id'
       AND version = (c ->> 'clave_version')::numeric;
    SELECT * INTO STRICT contexto
      FROM vec_contexto_actor_v1.registros_contexto
     WHERE registro_contexto_ref = c ->> 'contexto_ref';
    SELECT revision, huella_sha256 INTO STRICT catalogo
      FROM vec_autorizacion.control_catalogo_politicas
     WHERE control_id = true;
    SELECT a.asignacion_id, a.version
      INTO STRICT asignacion_actual
      FROM vec_autorizacion.asignacion_perfil_actual actual
      JOIN vec_autorizacion.asignacion_perfil a
        ON a.perfil_activo_ref = actual.perfil_activo_ref
       AND a.asignacion_ref = actual.asignacion_ref
     WHERE actual.perfil_activo_ref =
           'prf_sintetico_cccccccccccccccccccccccc';
    SELECT coalesce(jsonb_agg(
               p.documento ORDER BY p.politica_ref COLLATE "C"
           ), '[]'::jsonb)
      INTO politicas
      FROM vec_autorizacion.politica_restrictiva_actual actual
      JOIN vec_autorizacion.politica_restrictiva p
        ON p.politica_id = actual.politica_id
       AND p.politica_ref = actual.politica_ref;
    SELECT configuracion_secuencia_minima + 1,
           raiz_version_minima + 1
      INTO STRICT secuencia, version_raiz
      FROM vec_autorizacion_atestada_v3.checkpoint_gobierno
     WHERE control_id;
    SELECT * INTO STRICT r
      FROM vec_autorizacion_atestada_v3.raiz_confianza_version
     WHERE clave_id = c ->> 'raiz_clave_id'
       AND version = (c ->> 'raiz_version')::numeric;
    RETURN jsonb_build_object(
        'caso', p_caso,
        'ahora', to_char(
            ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'decision_plantilla_b64', encode(v.decision, 'base64'),
        'motivo_b64', encode(v.motivo, 'base64'),
        'contexto_b64', encode(v.contexto, 'base64'),
        'manifiesto_b64',
            encode(contexto.manifiesto_procedencia_canonico, 'base64'),
        'manifiesto_huella_sha256',
            contexto.manifiesto_procedencia_huella_sha256,
        'autoridad_efectiva', contexto.autoridad_efectiva,
        'resuelto_en', to_char(
            contexto.resuelto_en AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'alta_b64', encode(v.alta, 'base64'),
        'sellos_b64', encode(v.sellos, 'base64'),
        'efecto_huella_sha256', encode(sha256(v.alta), 'hex'),
        'clave_id', k.clave_id,
        'clave_version', k.version,
        'revision_gobierno', k.revision_gobierno,
        'huella_gobierno_sha256', k.huella_gobierno_sha256,
        'emisor_id', k.emisor_id,
        'audiencia_consumo', k.audiencia_consumo,
        'clave_hmac_b64', encode(k.secreto_hmac, 'base64'),
        'clave_valida_desde', to_char(
            k.valida_desde AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'clave_valida_hasta', to_char(
            k.valida_hasta AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'revision_confianza', 'configuracion-go-o205-' || p_caso,
        'secuencia_confianza', secuencia,
        'raiz_clave_id', 'raiz-go-o205-' || p_caso,
        'raiz_version', version_raiz,
        'audiencia_despliegue', r.audiencia_despliegue,
        'politicas', politicas,
        'revision_catalogo', catalogo.revision,
        'huella_catalogo_sha256', catalogo.huella_sha256,
        'asignacion_id', asignacion_actual.asignacion_id,
        'asignacion_version', asignacion_actual.version + 1,
        'persona_version', v.persona_version,
        'perfil_version', v.perfil_version
    )::text;
END
$funcion$;

CREATE FUNCTION public.aplicar_bundle_go_o2_05(
    p_caso text,
    p_bundle jsonb
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_capacidad bytea := decode(p_bundle ->> 'capacidad_b64', 'base64');
    v_decision bytea := decode(p_bundle ->> 'decision_b64', 'base64');
    v_spki bytea := decode(p_bundle ->> 'spki_b64', 'base64');
    c jsonb := convert_from(v_capacidad, 'UTF8')::jsonb;
    d jsonb := convert_from(v_decision, 'UTF8')::jsonb;
    rol jsonb := p_bundle -> 'version_rol_documento';
    control jsonb := p_bundle -> 'control_rol_documento';
    asignacion jsonb := p_bundle -> 'asignacion_documento';
    orden numeric;
BEGIN
    IF p_caso !~ '^[a-z0-9_-]{3,32}$'
       OR vec_autorizacion_atestada_v3.capacidad_canonica(c)
          IS DISTINCT FROM v_capacidad
       OR vec_autorizacion.decision_contexto_actor_v3_canonica(d)
          IS DISTINCT FROM v_decision THEN
        RAISE EXCEPTION 'bundle Go O2-05 no canónico';
    END IF;
    INSERT INTO vec_autorizacion.version_rol(
        version_rol_ref, rol_id, version, huella_sha256,
        publicada_en, documento
    ) VALUES (
        d ->> 'version_rol_ref', rol ->> 'rol_id',
        (rol ->> 'version')::numeric,
        d ->> 'version_rol_huella_sha256',
        (rol ->> 'publicada_en')::timestamptz, rol
    );
    INSERT INTO vec_autorizacion.control_vigencia_version_rol(
        version_rol_ref, revision, estado, huella_sha256,
        actualizado_en, documento
    ) VALUES (
        control ->> 'version_rol_ref',
        (control ->> 'revision')::numeric,
        control ->> 'estado',
        d ->> 'control_vigencia_version_rol_huella_sha256',
        (control ->> 'actualizado_en')::timestamptz, control
    );
    INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
    VALUES (
        control ->> 'version_rol_ref',
        (control ->> 'revision')::numeric,
        clock_timestamp(), 'autoridad-o205-go',
        'acto:control:o205:go:' || p_caso
    );
    INSERT INTO vec_autorizacion.asignacion_perfil(
        asignacion_ref, asignacion_id, version, perfil_activo_ref,
        principal_id, version_rol_ref, huella_sha256,
        emitida_en, documento
    ) VALUES (
        d ->> 'asignacion_ref', asignacion ->> 'asignacion_id',
        (asignacion ->> 'version')::numeric,
        asignacion ->> 'perfil_activo_ref',
        asignacion ->> 'principal_id',
        asignacion ->> 'version_rol_ref',
        d ->> 'asignacion_huella_sha256',
        (asignacion ->> 'emitida_en')::timestamptz, asignacion
    );
    INSERT INTO vec_autorizacion.asignacion_perfil_actual
    VALUES (
        asignacion ->> 'perfil_activo_ref', d ->> 'asignacion_ref',
        clock_timestamp(), 'autoridad-o205-go',
        'acto:asignacion:o205:go:' || p_caso
    ) ON CONFLICT (perfil_activo_ref) DO UPDATE SET
        asignacion_ref = EXCLUDED.asignacion_ref,
        actualizada_en = EXCLUDED.actualizada_en,
        actualizada_por = EXCLUDED.actualizada_por,
        acto_ref = EXCLUDED.acto_ref;
    INSERT INTO
        vec_autorizacion_atestada_v3.configuracion_confianza_version(
            revision, secuencia, huella_configuracion_sha256,
            publicada_en, expira_en, acto_ref
        ) VALUES (
            c ->> 'revision_confianza',
            (c ->> 'configuracion_secuencia')::numeric,
            c ->> 'huella_configuracion_sha256',
            (c ->> 'configuracion_publicada_en')::timestamptz,
            (c ->> 'configuracion_expira_en')::timestamptz,
            'acto:configuracion:o205:go:' || p_caso
        );
    INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version(
        clave_id, version, clave_publica_spki, huella_spki_sha256,
        valida_desde, valida_hasta, suite, audiencia_despliegue,
        acto_ref
    ) VALUES (
        c ->> 'raiz_clave_id', (c ->> 'raiz_version')::numeric,
        v_spki, encode(sha256(v_spki), 'hex'),
        (c ->> 'raiz_valida_desde')::timestamptz,
        (c ->> 'raiz_valida_hasta')::timestamptz,
        c ->> 'suite', c ->> 'audiencia_despliegue',
        'acto:raiz:o205:go:' || p_caso
    );
    INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz
    VALUES (
        c ->> 'revision_confianza',
        c ->> 'raiz_clave_id', (c ->> 'raiz_version')::numeric
    );
    SELECT coalesce(max(puntero.orden), 0) + 1 INTO orden
      FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual puntero;
    INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual
    VALUES (
        orden, c ->> 'revision_confianza', clock_timestamp(),
        'acto:puntero-configuracion:o205:go:' || p_caso
    );
    UPDATE public.vectores_o2_05 SET
        capacidad = v_capacidad,
        decision = v_decision,
        motivo = decode(p_bundle ->> 'motivo_b64', 'base64'),
        contexto = decode(p_bundle ->> 'contexto_b64', 'base64'),
        persona_version = (p_bundle ->> 'persona_version')::numeric,
        perfil_version = (p_bundle ->> 'perfil_version')::numeric,
        payload = decode(p_bundle ->> 'payload_b64', 'base64'),
        cose = decode(p_bundle ->> 'cose_b64', 'base64'),
        evidencia = decode(p_bundle ->> 'evidencia_b64', 'base64'),
        spki = v_spki,
        alta = decode(p_bundle ->> 'alta_b64', 'base64'),
        sellos = decode(p_bundle ->> 'sellos_b64', 'base64')
     WHERE caso = p_caso;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'vector O2-05 ausente';
    END IF;
END
$funcion$;

REVOKE ALL ON FUNCTION public.preparar_vector_o2_05(
    text, text, numeric
) FROM PUBLIC;
REVOKE ALL ON FUNCTION public.invocar_vector_o2_05(text) FROM PUBLIC;
REVOKE ALL ON FUNCTION public.mutar_efecto_o2_05(
    text, text, jsonb
) FROM PUBLIC;
REVOKE ALL ON FUNCTION public.mutar_tipo_capacidad_o2_05(
    text, text
) FROM PUBLIC;
REVOKE ALL ON FUNCTION public.durabilizar_decision_o2_05(text)
    FROM PUBLIC;
REVOKE ALL ON FUNCTION public.exportar_entrada_go_o2_05(text)
    FROM PUBLIC;
REVOKE ALL ON FUNCTION public.aplicar_bundle_go_o2_05(text, jsonb)
    FROM PUBLIC;
