-- Consulta exacta atomica. La funcion queda deliberadamente SIN EXECUTE para
-- el rol runtime hasta disponer del registrador COSE productivo que alimente
-- el catalogo de atestaciones. El propietario NOLOGIN puede validarla en una
-- integracion controlada, pero no es una identidad de aplicacion.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
BEGIN
    IF to_regclass('vec_bolsa_convocatorias.version_convocatoria') IS NULL
       OR to_regclass(
           'vec_bolsa_convocatorias.atestacion_autorizacion_version'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion.revalidar_decision_bolsa_convocatorias_v1(jsonb,bytea,bytea,text,text,text,jsonb,timestamp with time zone)'
       ) IS NULL
       OR to_regprocedure(
           'vec_bolsa_convocatorias.obtener_version_exacta_v1(jsonb,jsonb,bytea,bytea)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar la consulta exacta';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_bolsa_convocatorias.obtener_version_exacta_v1(
    p_operacion jsonb,
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_recurso_canonico bytea
)
RETURNS TABLE (
    resultado text,
    version_canonica bytea,
    huella_version_sha256 text,
    instancia_flujo_canonica bytea,
    huella_instancia_flujo_sha256 text,
    autorizacion_ref text,
    huella_autorizacion_sha256 text,
    atestacion_autorizacion_ref text,
    huella_atestacion_autorizacion_sha256 text,
    consumo_autorizacion_ref text,
    auditoria_ref text,
    huella_auditoria_sha256 text,
    consultada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_convocatoria_id text;
    v_secuencia bigint;
    v_incluir_flujo boolean;
    v_accion text;
    v_recurso_ref text;
    v_solicitada_en timestamptz;
    v_ahora timestamptz;
    v_campos jsonb;
    v_principal_ref text;
    v_version record;
    v_flujo record;
    v_atestacion record;
    v_flujo_canonico bytea := ''::bytea;
    v_huella_flujo text := '';
    v_huella_recurso text;
    v_efecto_canonico bytea;
    v_huella_efecto text;
    v_consumo_ref text;
    v_auditoria_ref text;
    v_ultima_secuencia bigint;
    v_huella_anterior text;
    v_registro_auditoria bytea;
    v_huella_auditoria text;
BEGIN
    IF current_setting('transaction_isolation') <> 'serializable' THEN
        RAISE EXCEPTION USING ERRCODE = '25001',
            MESSAGE = 'consulta rechazada: requiere transaccion SERIALIZABLE';
    END IF;
    IF p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_operacion)) <> 7
       OR NOT (p_operacion ?& ARRAY[
           'esquema', 'convocatoria_id', 'secuencia',
           'incluir_instancia_flujo', 'accion', 'recurso_ref',
           'solicitada_en'
       ])
       OR jsonb_typeof(p_operacion -> 'esquema') <> 'string'
       OR p_operacion ->> 'esquema' IS DISTINCT FROM
          'vec.bolsa.convocatoria.consulta-postgresql.v1'
       OR jsonb_typeof(p_operacion -> 'convocatoria_id') <> 'string'
       OR jsonb_typeof(p_operacion -> 'secuencia') <> 'number'
       OR jsonb_typeof(p_operacion -> 'incluir_instancia_flujo') <>
          'boolean'
       OR jsonb_typeof(p_operacion -> 'accion') <> 'string'
       OR jsonb_typeof(p_operacion -> 'recurso_ref') <> 'string'
       OR jsonb_typeof(p_operacion -> 'solicitada_en') <> 'string'
       OR (p_operacion ->> 'secuencia') !~ '^[1-9][0-9]{0,18}$'
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           p_operacion ->> 'solicitada_en'
       ) IS NOT TRUE
       OR p_prueba IS NULL OR jsonb_typeof(p_prueba) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_prueba)) <> 5
       OR NOT (p_prueba ?& ARRAY[
           'esquema_huella', 'decision_ref', 'huella_decision_sha256',
           'verificada_en', 'principal_ref'
       ])
       OR p_prueba ->> 'esquema_huella' IS DISTINCT FROM
          'vec.autorizacion.decision.reforzada.v1.autenticacion-actor'
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           p_prueba ->> 'decision_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           p_prueba ->> 'principal_ref', 512
       ) IS NOT TRUE
       OR (p_prueba ->> 'huella_decision_sha256') !~ '^[0-9a-f]{64}$'
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           p_prueba ->> 'verificada_en'
       ) IS NOT TRUE
       OR p_decision_canonica IS NULL
       OR p_recurso_canonico IS NULL
       OR octet_length(p_decision_canonica) NOT BETWEEN 1 AND 1048576
       OR octet_length(p_recurso_canonico) NOT BETWEEN 1 AND 65536
       OR encode(sha256(p_decision_canonica), 'hex') IS DISTINCT FROM
          p_prueba ->> 'huella_decision_sha256' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'consulta exacta invalida';
    END IF;

    BEGIN
        v_convocatoria_id := p_operacion ->> 'convocatoria_id';
        v_secuencia := (p_operacion ->> 'secuencia')::bigint;
        v_incluir_flujo := (p_operacion ->> 'incluir_instancia_flujo')::boolean;
        v_accion := p_operacion ->> 'accion';
        v_recurso_ref := p_operacion ->> 'recurso_ref';
        v_solicitada_en := (p_operacion ->> 'solicitada_en')::timestamptz;
    EXCEPTION WHEN data_exception OR invalid_text_representation
        OR numeric_value_out_of_range OR datetime_field_overflow THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'consulta exacta invalida';
    END;
    v_principal_ref := p_prueba ->> 'principal_ref';
    IF vec_bolsa_convocatorias.texto_opaco_valido(
           v_convocatoria_id, 480
       ) IS NOT TRUE
       OR strpos(v_convocatoria_id, '#') <> 0
       OR v_recurso_ref <> v_convocatoria_id || '#' || v_secuencia::text
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           v_recurso_ref, 512
       ) IS NOT TRUE
       OR (v_incluir_flujo AND v_accion <>
           'bolsa.convocatoria.version_con_flujo.consultar')
       OR (NOT v_incluir_flujo AND v_accion <>
           'bolsa.convocatoria.version.consultar')
       OR convert_from(p_recurso_canonico, 'UTF8')::jsonb IS DISTINCT FROM
          '{"ambitos":{},"atributos":{}}'::jsonb THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'consulta exacta invalida';
    END IF;

    v_ahora := clock_timestamp();
    IF v_ahora < v_solicitada_en
       OR v_ahora - v_solicitada_en > interval '30 seconds' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'ventana de consulta invalida';
    END IF;
    v_campos := CASE WHEN v_incluir_flujo THEN
        '["instancia_flujo","version_convocatoria"]'::jsonb
    ELSE '["version_convocatoria"]'::jsonb END;

    IF vec_autorizacion.revalidar_decision_bolsa_convocatorias_v1(
           p_prueba, p_decision_canonica, p_recurso_canonico,
           v_accion, 'version_convocatoria_gobernada', v_recurso_ref,
           v_campos, v_ahora
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'autorizacion no revalidada';
    END IF;

    SELECT version_atestacion.atestacion_ref,
           version_atestacion.version,
           version_atestacion.huella_evidencia_sha256,
           version_atestacion.huella_decision_sha256
      INTO STRICT v_atestacion
      FROM vec_bolsa_convocatorias.atestacion_autorizacion_actual AS actual
      JOIN vec_bolsa_convocatorias.atestacion_autorizacion_version
        AS version_atestacion
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

    SELECT almacen.version_canonica, almacen.huella_version_sha256,
           almacen.estado, almacen.instancia_flujo_ref
      INTO STRICT v_version
      FROM vec_bolsa_convocatorias.version_convocatoria AS almacen
     WHERE almacen.convocatoria_id = v_convocatoria_id
       AND almacen.secuencia = v_secuencia
     FOR SHARE OF almacen;

    IF v_version.estado = 'borrador' AND v_incluir_flujo
       OR v_version.estado <> 'borrador' AND NOT v_incluir_flujo THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'modalidad de consulta incompatible con el estado';
    END IF;
    IF v_incluir_flujo THEN
        SELECT flujo.instancia_flujo_ref,
               flujo.instancia_flujo_canonica,
               flujo.huella_instancia_flujo_sha256
          INTO STRICT v_flujo
          FROM vec_bolsa_convocatorias.instancia_flujo_version AS flujo
         WHERE flujo.convocatoria_id = v_convocatoria_id
           AND flujo.secuencia = v_secuencia
           AND flujo.instancia_flujo_ref = v_version.instancia_flujo_ref
         FOR SHARE OF flujo;
        v_flujo_canonico := v_flujo.instancia_flujo_canonica;
        v_huella_flujo := v_flujo.huella_instancia_flujo_sha256;
    END IF;

    v_huella_recurso := encode(sha256(p_recurso_canonico), 'hex');
    v_efecto_canonico := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.convocatoria.efecto-consulta.v1',
        'decision_ref', p_prueba ->> 'decision_ref',
        'principal_ref', v_principal_ref,
        'accion', v_accion,
        'recurso_ref', v_recurso_ref,
        'huella_recurso_sha256', v_huella_recurso,
        'huella_version_sha256', v_version.huella_version_sha256,
        'huella_instancia_flujo_sha256', v_huella_flujo,
        'atestacion_ref', v_atestacion.atestacion_ref,
        'huella_atestacion_sha256',
            v_atestacion.huella_evidencia_sha256,
        'consultada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_efecto := encode(sha256(v_efecto_canonico), 'hex');
    v_consumo_ref := 'consumo-convocatoria-' || encode(sha256(convert_to(
        'consumo:' || (p_prueba ->> 'decision_ref') || ':' ||
        v_huella_efecto, 'UTF8'
    )), 'hex');

    INSERT INTO vec_bolsa_convocatorias.uso_decision_consulta(
        decision_ref, consumo_autorizacion_ref, principal_ref, accion,
        recurso_ref, convocatoria_id, secuencia,
        huella_decision_sha256, huella_recurso_sha256, atestacion_ref,
        atestacion_version, huella_atestacion_sha256,
        huella_efecto_sha256, consumida_en
    ) VALUES (
        p_prueba ->> 'decision_ref', v_consumo_ref, v_principal_ref,
        v_accion, v_recurso_ref, v_convocatoria_id, v_secuencia,
        p_prueba ->> 'huella_decision_sha256', v_huella_recurso,
        v_atestacion.atestacion_ref, v_atestacion.version,
        v_atestacion.huella_evidencia_sha256, v_huella_efecto, v_ahora
    );

    SELECT actual.ultima_secuencia, actual.ultima_huella_sha256
      INTO STRICT v_ultima_secuencia, v_huella_anterior
      FROM vec_bolsa_convocatorias.auditoria_actual AS actual
     WHERE actual.control_id = true
     FOR UPDATE OF actual;
    v_auditoria_ref := 'auditoria-convocatoria-' || encode(sha256(convert_to(
        'auditoria:' || v_consumo_ref || ':' || v_huella_anterior,
        'UTF8'
    )), 'hex');
    v_registro_auditoria := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.convocatoria.auditoria-consulta.v1',
        'auditoria_ref', v_auditoria_ref,
        'secuencia', v_ultima_secuencia + 1,
        'huella_anterior_sha256', v_huella_anterior,
        'consumo_autorizacion_ref', v_consumo_ref,
        'decision_ref', p_prueba ->> 'decision_ref',
        'principal_ref', v_principal_ref,
        'accion', v_accion,
        'recurso_ref', v_recurso_ref,
        'huella_efecto_sha256', v_huella_efecto,
        'registrada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_auditoria := encode(sha256(v_registro_auditoria), 'hex');

    INSERT INTO vec_bolsa_convocatorias.auditoria(
        auditoria_ref, secuencia, consumo_autorizacion_ref,
        registro_canonico, huella_anterior_sha256,
        huella_auditoria_sha256, registrada_en
    ) VALUES (
        v_auditoria_ref, v_ultima_secuencia + 1, v_consumo_ref,
        v_registro_auditoria, v_huella_anterior,
        v_huella_auditoria, v_ahora
    );
    UPDATE vec_bolsa_convocatorias.auditoria_actual
       SET ultima_secuencia = v_ultima_secuencia + 1,
           ultima_huella_sha256 = v_huella_auditoria,
           actualizada_en = v_ahora
     WHERE control_id = true;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'no se pudo avanzar la cadena de auditoria';
    END IF;

    RETURN QUERY SELECT
        'encontrada'::text,
        v_version.version_canonica::bytea,
        v_version.huella_version_sha256::text,
        v_flujo_canonico,
        v_huella_flujo,
        (p_prueba ->> 'decision_ref')::text,
        (p_prueba ->> 'huella_decision_sha256')::text,
        v_atestacion.atestacion_ref::text,
        v_atestacion.huella_evidencia_sha256::text,
        v_consumo_ref,
        v_auditoria_ref,
        v_huella_auditoria,
        v_ahora;
EXCEPTION
    WHEN no_data_found OR too_many_rows THEN
        RAISE EXCEPTION USING ERRCODE = 'P0002',
            MESSAGE = 'consulta exacta sin evidencia completa';
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_bolsa_convocatorias.obtener_version_exacta_v1(
        jsonb, jsonb, bytea, bytea
    ) FROM PUBLIC,
        vec_bolsa_convocatorias_ejecutor_consulta,
        vec_bolsa_convocatorias_proyector_gobierno,
        vec_bolsa_convocatorias_registrador_atestacion;

COMMENT ON FUNCTION
    vec_bolsa_convocatorias.obtener_version_exacta_v1(
        jsonb, jsonb, bytea, bytea
    ) IS
    'Consulta exacta SERIALIZABLE con revalidacion, atestacion, consumo unico y auditoria encadenada. Sin EXECUTE runtime hasta conectar el registrador COSE productivo.';
COMMIT;
