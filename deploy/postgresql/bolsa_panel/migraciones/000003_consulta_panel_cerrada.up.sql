-- Consulta atomica del panel. Deliberadamente no se concede EXECUTE al rol
-- runtime hasta que el verificador COSE productivo pueda registrar y revocar
-- atestaciones mediante una capacidad aislada.
BEGIN;
SET LOCAL ROLE vec_bolsa_panel_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
BEGIN
    IF to_regclass('vec_bolsa_panel.proyeccion_panel') IS NULL
       OR to_regclass('vec_bolsa_panel.consulta_confirmada') IS NULL
       OR to_regprocedure(
           'vec_autorizacion.revalidar_decision_panel_bolsa_v2(jsonb,bytea,bytea,text,text,text,timestamp with time zone)'
       ) IS NULL
       OR to_regprocedure(
           'vec_bolsa_panel.consultar_panel_interno_v1(jsonb,jsonb,bytea,bytea,text)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar la consulta del panel';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_bolsa_panel.consultar_panel_interno_v1(
    p_operacion jsonb,
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_correlacion_ref text
)
RETURNS TABLE (panel_canonico bytea)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_clase text;
    v_organizacion text;
    v_unidad text := '';
    v_accion text;
    v_recurso_ref text;
    v_consultada_en timestamptz;
    v_verificada_en timestamptz;
    v_ahora timestamptz;
    v_motivo jsonb;
    v_referencia_motivo jsonb;
    v_contexto_canonico bytea;
    v_huella_contexto text;
    v_huella_operacion text;
    v_huella_motivo text;
    v_existente record;
    v_atestacion record;
    v_proyeccion record;
    v_convocatorias jsonb;
    v_actuaciones jsonb;
    v_selector jsonb;
    v_lectura_ref text;
    v_auditoria_ref text;
    v_secuencia bigint;
    v_huella_anterior text;
    v_registro_auditoria bytea;
    v_huella_auditoria text;
    v_panel jsonb;
    v_panel_bytes bytea;
BEGIN
    IF current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION USING ERRCODE = '25001',
            MESSAGE = 'consulta rechazada: requiere SERIALIZABLE READ WRITE';
    END IF;
    IF p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object'
       OR jsonb_typeof(p_operacion -> 'esquema') <> 'string'
       OR p_operacion ->> 'esquema' IS DISTINCT FROM
          'vec.bolsa.panel.interno.consulta-postgresql.v1'
       OR jsonb_typeof(p_operacion -> 'clase_ambito') <> 'string'
       OR jsonb_typeof(p_operacion -> 'organizacion_ref') <> 'string'
       OR jsonb_typeof(p_operacion -> 'accion') <> 'string'
       OR jsonb_typeof(p_operacion -> 'recurso_ref') <> 'string'
       OR jsonb_typeof(p_operacion -> 'consultada_en') <> 'string'
       OR p_operacion ->> 'accion' IS DISTINCT FROM
          'bolsa.panel_interno.consultar'
       OR vec_bolsa_panel.referencia_opaca_valida(
              p_operacion ->> 'organizacion_ref', 'org_'
          ) IS NOT TRUE
       OR vec_bolsa_panel.instante_utc_microsegundo_valido(
              p_operacion ->> 'consultada_en'
          ) IS NOT TRUE
       OR p_prueba IS NULL OR jsonb_typeof(p_prueba) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_prueba)) <> 4
       OR NOT (p_prueba ?& ARRAY[
           'esquema_huella', 'decision_ref', 'huella_decision_sha256',
           'verificada_en'
       ])
       OR p_prueba ->> 'esquema_huella' IS DISTINCT FROM
          'vec.autorizacion.decision.reforzada.v2.solicitud-ligada'
       OR jsonb_typeof(p_prueba -> 'decision_ref') <> 'string'
       OR octet_length(p_prueba ->> 'decision_ref') NOT BETWEEN 1 AND 512
       OR (p_prueba ->> 'decision_ref') !~ '^[^*[:space:][:cntrl:]]+$'
       OR jsonb_typeof(p_prueba -> 'huella_decision_sha256') <> 'string'
       OR (p_prueba ->> 'huella_decision_sha256') !~ '^[0-9a-f]{64}$'
       OR jsonb_typeof(p_prueba -> 'verificada_en') <> 'string'
       OR vec_bolsa_panel.instante_utc_microsegundo_valido(
              p_prueba ->> 'verificada_en'
          ) IS NOT TRUE
       OR p_decision_canonica IS NULL
       OR octet_length(p_decision_canonica) NOT BETWEEN 1 AND 524288
       OR p_motivo_canonico IS NULL
       OR octet_length(p_motivo_canonico) NOT BETWEEN 1 AND 65536
       OR p_correlacion_ref !~ '^correlacion_[0-9a-f]{32}$'
       OR encode(sha256(p_decision_canonica), 'hex') IS DISTINCT FROM
          p_prueba ->> 'huella_decision_sha256' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'consulta del panel invalida';
    END IF;

    v_clase := p_operacion ->> 'clase_ambito';
    v_organizacion := p_operacion ->> 'organizacion_ref';
    v_accion := p_operacion ->> 'accion';
    v_recurso_ref := p_operacion ->> 'recurso_ref';
    IF v_clase = 'organizacion' THEN
        IF (SELECT count(*) FROM jsonb_object_keys(p_operacion)) <> 6
           OR p_operacion ? 'unidad_gestion_ref'
           OR v_recurso_ref <> 'panel:' || v_organizacion THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'selector de organizacion invalido';
        END IF;
    ELSIF v_clase = 'unidad_gestion' THEN
        IF (SELECT count(*) FROM jsonb_object_keys(p_operacion)) <> 7
           OR jsonb_typeof(p_operacion -> 'unidad_gestion_ref') <> 'string'
           OR vec_bolsa_panel.referencia_opaca_valida(
                  p_operacion ->> 'unidad_gestion_ref', 'uni_'
              ) IS NOT TRUE THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'selector de unidad invalido';
        END IF;
        v_unidad := p_operacion ->> 'unidad_gestion_ref';
        IF v_recurso_ref <> 'panel:' || v_organizacion || ':' || v_unidad THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'recurso de unidad invalido';
        END IF;
    ELSE
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'clase de alcance invalida';
    END IF;

    BEGIN
        v_consultada_en := (p_operacion ->> 'consultada_en')::timestamptz;
        v_verificada_en := (p_prueba ->> 'verificada_en')::timestamptz;
        v_motivo := convert_from(p_motivo_canonico, 'UTF8')::jsonb;
    EXCEPTION
        WHEN character_not_in_repertoire OR untranslatable_character
            OR invalid_text_representation OR data_exception
            OR datetime_field_overflow THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'documentos de consulta invalidos';
    END;
    IF jsonb_typeof(v_motivo) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(v_motivo)) <> 2
       OR NOT (v_motivo ?& ARRAY['esquema', 'referencia'])
       OR v_motivo ->> 'esquema' IS DISTINCT FROM
          'vec.autorizacion.motivo.v2.referencia-opaca-catalogada'
       OR jsonb_typeof(v_motivo -> 'referencia') <> 'object'
       OR (SELECT count(*)
             FROM jsonb_object_keys(v_motivo -> 'referencia')) <> 4
       OR NOT ((v_motivo -> 'referencia') ?& ARRAY[
           'catalogo_id', 'catalogo_version',
           'catalogo_huella_sha256', 'entrada_clave'
       ]) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'motivo V2 invalido';
    END IF;
    v_referencia_motivo := v_motivo -> 'referencia';
    IF jsonb_typeof(v_referencia_motivo -> 'catalogo_id') <> 'string'
       OR (v_referencia_motivo ->> 'catalogo_id') !~
          '^[a-z][a-z0-9._-]{0,127}$'
       OR jsonb_typeof(v_referencia_motivo -> 'catalogo_version') <> 'number'
       OR (v_referencia_motivo ->> 'catalogo_version') !~
          '^[1-9][0-9]{0,9}$'
       OR (v_referencia_motivo ->> 'catalogo_version')::numeric NOT BETWEEN
          1 AND 2147483647
       OR jsonb_typeof(
              v_referencia_motivo -> 'catalogo_huella_sha256'
          ) <> 'string'
       OR (v_referencia_motivo ->> 'catalogo_huella_sha256') !~
          '^[0-9a-f]{64}$'
       OR v_referencia_motivo ->> 'catalogo_huella_sha256' =
          repeat('0', 64)
       OR jsonb_typeof(v_referencia_motivo -> 'entrada_clave') <> 'string'
       OR (v_referencia_motivo ->> 'entrada_clave') !~
          '^motivo_[0-9a-f]{32}$' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'referencia de motivo V2 invalida';
    END IF;

    -- Preimagen exacta de RecursoAutorizable.HuellaContextoAutorizacionSHA256.
    -- Los valores estan restringidos a alfabetos sin escapes JSON.
    v_contexto_canonico := convert_to(
        '{"ambitos":{"clase":"' || v_clase ||
        '","organizacion_ref":"' || v_organizacion || '"' ||
        CASE WHEN v_clase = 'unidad_gestion' THEN
            ',"unidad_gestion_ref":"' || v_unidad || '"'
        ELSE '' END ||
        '},"atributos":{"motivo_catalogo_huella":"' ||
        (v_referencia_motivo ->> 'catalogo_huella_sha256') ||
        '","motivo_catalogo_id":"' ||
        (v_referencia_motivo ->> 'catalogo_id') ||
        '","motivo_catalogo_version":"' ||
        (v_referencia_motivo ->> 'catalogo_version') ||
        '","motivo_entrada_clave":"' ||
        (v_referencia_motivo ->> 'entrada_clave') || '"}}',
        'UTF8'
    );
    v_huella_contexto := encode(sha256(v_contexto_canonico), 'hex');
    v_huella_operacion := encode(
        sha256(convert_to(p_operacion::text, 'UTF8')), 'hex'
    );
    v_huella_motivo := encode(sha256(p_motivo_canonico), 'hex');
    v_ahora := clock_timestamp();
    IF v_ahora < v_consultada_en
       OR v_ahora - v_consultada_en > interval '30 seconds'
       OR v_consultada_en < v_verificada_en THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'ventana temporal de consulta invalida';
    END IF;

    IF vec_autorizacion.revalidar_decision_panel_bolsa_v2(
           p_prueba, p_decision_canonica, p_motivo_canonico,
           p_correlacion_ref, v_recurso_ref, v_huella_contexto, v_ahora
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'autorizacion V2 del panel no revalidada';
    END IF;

    SELECT version_atestacion.atestacion_ref,
           version_atestacion.version,
           version_atestacion.huella_evidencia_sha256
      INTO v_atestacion
      FROM vec_bolsa_panel.atestacion_autorizacion_actual AS actual
      JOIN vec_bolsa_panel.atestacion_autorizacion_version AS version_atestacion
        ON version_atestacion.decision_ref = actual.decision_ref
       AND version_atestacion.atestacion_ref = actual.atestacion_ref
       AND version_atestacion.version = actual.version
       AND version_atestacion.estado = actual.estado
     WHERE actual.decision_ref = p_prueba ->> 'decision_ref'
       AND actual.estado = 'activa'
       AND version_atestacion.huella_decision_sha256 =
           p_prueba ->> 'huella_decision_sha256'
       AND v_ahora >= version_atestacion.valida_desde
       AND v_ahora < version_atestacion.valida_hasta
     FOR SHARE OF actual, version_atestacion;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'falta una atestacion criptografica activa';
    END IF;

    SELECT consumo.operacion_huella_sha256,
           consumo.huella_decision_sha256,
           consumo.huella_motivo_sha256,
           consumo.correlacion_ref, consumo.clase_ambito,
           consumo.organizacion_ref, consumo.unidad_gestion_ref,
           consumo.panel_canonico
      INTO v_existente
      FROM vec_bolsa_panel.consulta_confirmada AS consumo
     WHERE consumo.decision_ref = p_prueba ->> 'decision_ref'
     FOR SHARE OF consumo;
    IF FOUND THEN
        IF v_existente.operacion_huella_sha256 <> v_huella_operacion
           OR v_existente.huella_decision_sha256 <>
              p_prueba ->> 'huella_decision_sha256'
           OR v_existente.huella_motivo_sha256 <> v_huella_motivo
           OR v_existente.correlacion_ref <> p_correlacion_ref
           OR v_existente.clase_ambito <> v_clase
           OR v_existente.organizacion_ref <> v_organizacion
           OR v_existente.unidad_gestion_ref <> v_unidad THEN
            RAISE EXCEPTION USING ERRCODE = '23505',
                MESSAGE = 'decision ya consumida para otro efecto';
        END IF;
        panel_canonico := v_existente.panel_canonico;
        RETURN NEXT;
        RETURN;
    END IF;

    SELECT proyeccion.revision, proyeccion.actualizada_en,
           proyeccion.indicadores
      INTO v_proyeccion
      FROM vec_bolsa_panel.proyeccion_actual AS actual
      JOIN vec_bolsa_panel.proyeccion_panel AS proyeccion
        ON proyeccion.clase_ambito = actual.clase_ambito
       AND proyeccion.organizacion_ref = actual.organizacion_ref
       AND proyeccion.unidad_gestion_ref = actual.unidad_gestion_ref
       AND proyeccion.revision = actual.revision
     WHERE actual.clase_ambito = v_clase
       AND actual.organizacion_ref = v_organizacion
       AND actual.unidad_gestion_ref = v_unidad
     FOR SHARE OF actual, proyeccion;
    IF NOT FOUND OR v_proyeccion.actualizada_en > v_consultada_en THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'no existe una proyeccion coherente para el alcance';
    END IF;

    SELECT COALESCE(jsonb_agg(resumen.documento ORDER BY resumen.ordinal),
                    '[]'::jsonb)
      INTO v_convocatorias
      FROM vec_bolsa_panel.convocatoria_resumen AS resumen
     WHERE resumen.clase_ambito = v_clase
       AND resumen.organizacion_ref = v_organizacion
       AND resumen.unidad_gestion_ref = v_unidad
       AND resumen.revision = v_proyeccion.revision;
    SELECT COALESCE(jsonb_agg(actuacion.documento ORDER BY actuacion.ordinal),
                    '[]'::jsonb)
      INTO v_actuaciones
      FROM vec_bolsa_panel.actuacion_pendiente AS actuacion
     WHERE actuacion.clase_ambito = v_clase
       AND actuacion.organizacion_ref = v_organizacion
       AND actuacion.unidad_gestion_ref = v_unidad
       AND actuacion.revision = v_proyeccion.revision;

    SELECT ultima_secuencia, ultima_huella_sha256
      INTO STRICT v_secuencia, v_huella_anterior
      FROM vec_bolsa_panel.auditoria_actual
     WHERE control_id = true
     FOR UPDATE;
    v_secuencia := v_secuencia + 1;
    v_lectura_ref := 'lec_' || encode(sha256(convert_to(
        'lectura:' || (p_prueba ->> 'decision_ref') || ':' ||
        v_huella_operacion, 'UTF8'
    )), 'hex');
    v_auditoria_ref := 'aud_' || encode(sha256(convert_to(
        'auditoria:' || v_secuencia::text || ':' || v_lectura_ref,
        'UTF8'
    )), 'hex');
    v_registro_auditoria := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.panel.auditoria-lectura.v1',
        'auditoria_ref', v_auditoria_ref,
        'secuencia', v_secuencia,
        'lectura_ref', v_lectura_ref,
        'decision_ref', p_prueba ->> 'decision_ref',
        'huella_decision_sha256',
            p_prueba ->> 'huella_decision_sha256',
        'correlacion_ref', p_correlacion_ref,
        'recurso_ref', v_recurso_ref,
        'revision', v_proyeccion.revision,
        'huella_operacion_sha256', v_huella_operacion,
        'huella_motivo_sha256', v_huella_motivo,
        'atestacion_ref', v_atestacion.atestacion_ref,
        'huella_atestacion_sha256',
            v_atestacion.huella_evidencia_sha256,
        'confirmada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_auditoria := encode(sha256(
        decode(v_huella_anterior, 'hex') || v_registro_auditoria
    ), 'hex');
    INSERT INTO vec_bolsa_panel.auditoria(
        auditoria_ref, secuencia, lectura_ref, registro_canonico,
        huella_anterior_sha256, huella_auditoria_sha256, registrada_en
    ) VALUES (
        v_auditoria_ref, v_secuencia, v_lectura_ref,
        v_registro_auditoria, v_huella_anterior, v_huella_auditoria, v_ahora
    );
    UPDATE vec_bolsa_panel.auditoria_actual
       SET ultima_secuencia = v_secuencia,
           ultima_huella_sha256 = v_huella_auditoria,
           actualizada_en = v_ahora
     WHERE control_id = true;

    v_selector := jsonb_build_object(
        'clase', v_clase, 'organizacion_ref', v_organizacion
    );
    IF v_clase = 'unidad_gestion' THEN
        v_selector := v_selector || jsonb_build_object(
            'unidad_gestion_ref', v_unidad
        );
    END IF;
    v_panel := jsonb_build_object(
        'esquema', 'vec.bolsa.panel.interno.v1',
        'selector', v_selector,
        'origen', jsonb_build_object(
            'revision', v_proyeccion.revision,
            'actualizada_en', to_char(
                v_proyeccion.actualizada_en AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
            'demostracion', false
        ),
        'prueba_lectura', jsonb_build_object(
            'lectura_ref', v_lectura_ref,
            'auditoria_ref', v_auditoria_ref,
            'auditoria_secuencia', v_secuencia,
            'decision_ref', p_prueba ->> 'decision_ref',
            'huella_decision_sha256',
                p_prueba ->> 'huella_decision_sha256',
            'correlacion_ref', p_correlacion_ref,
            'confirmada_en', to_char(v_ahora AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        ),
        'indicadores', v_proyeccion.indicadores,
        'convocatorias', v_convocatorias,
        'actuaciones_pendientes', v_actuaciones
    );
    v_panel_bytes := convert_to(v_panel::text, 'UTF8');
    IF octet_length(v_panel_bytes) NOT BETWEEN 2 AND 2097152 THEN
        RAISE EXCEPTION USING ERRCODE = '54000',
            MESSAGE = 'respuesta del panel fuera de limite';
    END IF;

    INSERT INTO vec_bolsa_panel.consulta_confirmada(
        decision_ref, correlacion_ref, clase_ambito, organizacion_ref,
        unidad_gestion_ref, revision, operacion_huella_sha256,
        huella_decision_sha256, huella_motivo_sha256,
        atestacion_ref, atestacion_version, huella_atestacion_sha256,
        lectura_ref, auditoria_ref, auditoria_secuencia,
        panel_canonico, panel_huella_sha256, confirmada_en
    ) VALUES (
        p_prueba ->> 'decision_ref', p_correlacion_ref, v_clase,
        v_organizacion, v_unidad, v_proyeccion.revision,
        v_huella_operacion, p_prueba ->> 'huella_decision_sha256',
        v_huella_motivo, v_atestacion.atestacion_ref,
        v_atestacion.version, v_atestacion.huella_evidencia_sha256,
        v_lectura_ref, v_auditoria_ref, v_secuencia, v_panel_bytes,
        encode(sha256(v_panel_bytes), 'hex'), v_ahora
    );
    panel_canonico := v_panel_bytes;
    RETURN NEXT;
END
$funcion$;

REVOKE ALL ON FUNCTION vec_bolsa_panel.consultar_panel_interno_v1(
    jsonb, jsonb, bytea, bytea, text
) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_panel.consultar_panel_interno_v1(
    jsonb, jsonb, bytea, bytea, text
) FROM vec_bolsa_panel_ejecutor_consulta,
       vec_bolsa_panel_proyector,
       vec_bolsa_panel_registrador_atestacion;

COMMENT ON FUNCTION vec_bolsa_panel.consultar_panel_interno_v1(
    jsonb, jsonb, bytea, bytea, text
) IS
    'Consulta agregada sin PII: revalida V2, motivo, alcance, atestacion activa, consume idempotentemente y audita. Cerrada a runtime hasta autoridad COSE productiva.';
COMMIT;
