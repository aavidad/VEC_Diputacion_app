-- Funcion atomica cerrada. No se concede EXECUTE hasta que un registrador
-- COSE productivo pueda alimentar atestaciones sin autoridad compartida.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
BEGIN
    IF to_regclass('vec_bolsa_llamamientos.propuesta') IS NULL OR
       to_regprocedure(
          'vec_autorizacion.revalidar_decision_bolsa_llamamientos_v1(jsonb,bytea,bytea,text,text,text,jsonb,timestamp with time zone)'
       ) IS NULL OR
       to_regprocedure(
          'vec_bolsa_llamamientos.guardar_propuesta_v1(jsonb,jsonb,bytea,bytea)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar guardado';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_bolsa_llamamientos.guardar_propuesta_v1(
    p_operacion jsonb,
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_propuesta_canonica bytea
)
RETURNS TABLE (
    resultado text,
    propuesta_ref text,
    huella_propuesta_sha256 text,
    propuesta_canonica bytea,
    huella_documento_sha256 text,
    decision_ref text,
    huella_decision_sha256 text,
    atestacion_ref text,
    atestacion_canonica bytea,
    huella_atestacion_sha256 text,
    consumo_ref text,
    consumo_canonico bytea,
    huella_consumo_sha256 text,
    auditoria_ref text,
    registro_auditoria bytea,
    huella_auditoria_sha256 text,
    evento_ref text,
    evento_canonico bytea,
    huella_evento_sha256 text,
    confirmada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_ahora timestamptz;
    v_propuesta jsonb;
    v_propuesta_json json;
    v_necesidad record;
    v_bolsa record;
    v_politica record;
    v_instantanea record;
    v_evaluacion record;
    v_atestacion record;
    v_existente record;
    v_recurso_canonico bytea;
    v_total_evaluaciones integer;
    v_indice integer;
    v_ultima jsonb;
    v_consumo_ref text;
    v_consumo_canonico bytea;
    v_huella_consumo text;
    v_auditoria_ref text;
    v_registro_auditoria bytea;
    v_huella_auditoria text;
    v_evento_ref text;
    v_evento_canonico bytea;
    v_huella_evento text;
    v_ultima_secuencia bigint;
    v_huella_anterior text;
BEGIN
    IF current_setting('transaction_isolation') <> 'serializable' OR
       current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION USING ERRCODE = '25001',
            MESSAGE = 'guardado requiere SERIALIZABLE READ WRITE';
    END IF;
    v_ahora := clock_timestamp();

    IF p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object' OR
       (SELECT count(*) FROM jsonb_object_keys(p_operacion)) <> 11 OR
       NOT (p_operacion ?& ARRAY[
           'esquema', 'propuesta_ref', 'necesidad_ref',
           'version_necesidad', 'huella_necesidad_sha256',
           'huella_propuesta_sha256', 'huella_documento_sha256',
           'accion', 'finalidad', 'tipo_recurso', 'solicitada_en'
       ]) OR
       p_operacion ->> 'esquema' IS DISTINCT FROM
          'vec.bolsa.llamamiento.guardar-postgresql.v1' OR
       p_operacion ->> 'accion' IS DISTINCT FROM
          'bolsa.llamamiento.proponer' OR
       p_operacion ->> 'finalidad' IS DISTINCT FROM
          'gestion_propuestas_llamamiento' OR
       p_operacion ->> 'tipo_recurso' IS DISTINCT FROM
          'necesidad_cobertura' OR
       (p_operacion ->> 'version_necesidad') !~ '^[1-9][0-9]{0,18}$' OR
       vec_bolsa_llamamientos.texto_opaco_valido(
          p_operacion ->> 'propuesta_ref', 512
       ) IS NOT TRUE OR
       vec_bolsa_llamamientos.texto_opaco_valido(
          p_operacion ->> 'necesidad_ref', 512
       ) IS NOT TRUE OR
       vec_bolsa_llamamientos.huella_valida(
          p_operacion ->> 'huella_necesidad_sha256'
       ) IS NOT TRUE OR
       vec_bolsa_llamamientos.huella_valida(
          p_operacion ->> 'huella_propuesta_sha256'
       ) IS NOT TRUE OR
       vec_bolsa_llamamientos.huella_valida(
          p_operacion ->> 'huella_documento_sha256'
       ) IS NOT TRUE OR
       vec_bolsa_llamamientos.instante_utc_microsegundo_valido(
          p_operacion ->> 'solicitada_en'
       ) IS NOT TRUE OR
       (p_operacion ->> 'solicitada_en')::timestamptz > v_ahora OR
       v_ahora - (p_operacion ->> 'solicitada_en')::timestamptz >
          interval '30 seconds' OR
       p_prueba IS NULL OR jsonb_typeof(p_prueba) <> 'object' OR
       (SELECT count(*) FROM jsonb_object_keys(p_prueba)) <> 5 OR
       NOT (p_prueba ?& ARRAY[
          'esquema_huella', 'decision_ref', 'huella_decision_sha256',
          'verificada_en', 'principal_ref'
       ]) OR
       octet_length(p_decision_canonica) NOT BETWEEN 1 AND 1048576 OR
       octet_length(p_propuesta_canonica) NOT BETWEEN 2 AND 33554432 OR
       encode(sha256(p_decision_canonica), 'hex') IS DISTINCT FROM
          p_prueba ->> 'huella_decision_sha256' OR
       encode(sha256(p_propuesta_canonica), 'hex') IS DISTINCT FROM
          p_operacion ->> 'huella_documento_sha256' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'sobre de llamamiento invalido';
    END IF;

    BEGIN
        v_propuesta_json := convert_from(p_propuesta_canonica, 'UTF8')::json;
        v_propuesta := v_propuesta_json::jsonb;
    EXCEPTION WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'propuesta no es JSON valido';
    END;
    IF jsonb_typeof(v_propuesta) <> 'object' OR
       (SELECT count(*) FROM json_object_keys(v_propuesta_json)) <> 22 OR
       (SELECT count(*) FROM jsonb_object_keys(v_propuesta)) <> 22 OR
       NOT (v_propuesta ?& ARRAY[
          'propuesta_ref', 'bolsa_ref', 'version_bolsa',
          'huella_bolsa_sha256', 'necesidad_ref', 'version_necesidad',
          'huella_necesidad_sha256', 'instantanea_ref',
          'version_instantanea', 'huella_instantanea_sha256',
          'politica_ref', 'version_politica', 'huella_politica_sha256',
          'instante_referencia', 'instantanea_generada_en',
          'total_participaciones_instantanea', 'evaluaciones',
          'participacion_seleccionada_ref', 'sujeto_seleccionado_ref',
          'orden_seleccionado', 'generada_en', 'huella_contenido_sha256'
       ]) OR
       v_propuesta ->> 'propuesta_ref' IS DISTINCT FROM
          p_operacion ->> 'propuesta_ref' OR
       v_propuesta ->> 'necesidad_ref' IS DISTINCT FROM
          p_operacion ->> 'necesidad_ref' OR
       v_propuesta ->> 'version_necesidad' IS DISTINCT FROM
          p_operacion ->> 'version_necesidad' OR
       v_propuesta ->> 'huella_necesidad_sha256' IS DISTINCT FROM
          p_operacion ->> 'huella_necesidad_sha256' OR
       v_propuesta ->> 'huella_contenido_sha256' IS DISTINCT FROM
          p_operacion ->> 'huella_propuesta_sha256' OR
       jsonb_typeof(v_propuesta -> 'evaluaciones') <> 'array' OR
       vec_bolsa_llamamientos.instante_utc_microsegundo_valido(
          v_propuesta ->> 'generada_en'
       ) IS NOT TRUE OR
       (v_propuesta ->> 'generada_en')::timestamptz > v_ahora THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'documento de propuesta incoherente';
    END IF;

    SELECT necesidad.*, actual.estado AS estado_actual
      INTO STRICT v_necesidad
      FROM vec_bolsa_llamamientos.necesidad_actual AS actual
      JOIN vec_bolsa_llamamientos.necesidad_autoritativa AS necesidad
        ON necesidad.necesidad_ref = actual.necesidad_ref
       AND necesidad.version = actual.version
       AND necesidad.huella_necesidad_sha256 =
           actual.huella_necesidad_sha256
     WHERE actual.necesidad_ref = p_operacion ->> 'necesidad_ref'
     FOR UPDATE OF actual, necesidad;
    IF v_necesidad.version::text IS DISTINCT FROM
          p_operacion ->> 'version_necesidad' OR
       v_necesidad.huella_necesidad_sha256 IS DISTINCT FROM
          p_operacion ->> 'huella_necesidad_sha256' OR
       v_necesidad.estado_actual <> 'abierta' OR
       v_ahora >= v_necesidad.fin_previsto THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'necesidad no vigente';
    END IF;

    SELECT * INTO STRICT v_bolsa
      FROM vec_bolsa_llamamientos.bolsa_autoritativa
     WHERE bolsa_ref = v_necesidad.bolsa_ref
       AND version = v_necesidad.version_bolsa
       AND huella_bolsa_sha256 = v_necesidad.huella_bolsa_sha256
     FOR SHARE;
    SELECT * INTO STRICT v_politica
      FROM vec_bolsa_llamamientos.politica_autoritativa
     WHERE politica_ref = v_propuesta ->> 'politica_ref'
       AND version = (v_propuesta ->> 'version_politica')::bigint
       AND huella_politica_sha256 = v_propuesta ->> 'huella_politica_sha256'
     FOR SHARE;
    SELECT * INTO STRICT v_instantanea
      FROM vec_bolsa_llamamientos.instantanea_autoritativa
     WHERE instantanea_ref = v_propuesta ->> 'instantanea_ref'
       AND version = (v_propuesta ->> 'version_instantanea')::bigint
       AND huella_instantanea_sha256 =
           v_propuesta ->> 'huella_instantanea_sha256'
     FOR SHARE;
    IF v_bolsa.estado <> 'vigente' OR v_ahora < v_bolsa.vigente_desde OR
       (v_bolsa.vigente_hasta IS NOT NULL AND v_ahora >= v_bolsa.vigente_hasta) OR
       v_politica.estado <> 'publicada' OR v_ahora < v_politica.vigente_desde OR
       (v_politica.vigente_hasta IS NOT NULL AND v_ahora >= v_politica.vigente_hasta) OR
       v_propuesta ->> 'bolsa_ref' IS DISTINCT FROM v_bolsa.bolsa_ref OR
       (v_propuesta ->> 'version_bolsa')::bigint <> v_bolsa.version OR
       v_propuesta ->> 'huella_bolsa_sha256' IS DISTINCT FROM
          v_bolsa.huella_bolsa_sha256 OR
       v_instantanea.bolsa_ref IS DISTINCT FROM v_bolsa.bolsa_ref OR
       v_instantanea.version_bolsa <> v_bolsa.version OR
       v_instantanea.huella_bolsa_sha256 IS DISTINCT FROM
          v_bolsa.huella_bolsa_sha256 OR
       (v_propuesta ->> 'total_participaciones_instantanea')::bigint <>
          v_instantanea.total_participaciones OR
       (v_propuesta ->> 'instante_referencia')::timestamptz <>
          v_instantanea.referida_en OR
       (v_propuesta ->> 'instantanea_generada_en')::timestamptz <>
          v_instantanea.generada_en THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'fuentes autoritativas no coinciden';
    END IF;

    v_total_evaluaciones := jsonb_array_length(v_propuesta -> 'evaluaciones');
    IF v_total_evaluaciones < 1 OR
       v_total_evaluaciones > v_instantanea.total_participaciones OR
       (v_propuesta ->> 'orden_seleccionado')::bigint <>
          v_total_evaluaciones THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'prefijo de evaluaciones invalido';
    END IF;
    FOR v_indice IN 0..v_total_evaluaciones - 1 LOOP
        SELECT * INTO STRICT v_evaluacion
          FROM vec_bolsa_llamamientos.evaluacion_autoritativa
         WHERE instantanea_ref = v_instantanea.instantanea_ref
           AND version_instantanea = v_instantanea.version
           AND orden = v_indice + 1
         FOR SHARE;
        IF v_evaluacion.evaluacion_canonica IS DISTINCT FROM
              v_propuesta -> 'evaluaciones' -> v_indice OR
           (v_indice < v_total_evaluaciones - 1 AND
              v_evaluacion.resultado <> 'no_elegible') OR
           (v_indice = v_total_evaluaciones - 1 AND
              v_evaluacion.resultado <> 'elegible') THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'evaluacion no autoritativa';
        END IF;
    END LOOP;
    v_ultima := v_propuesta -> 'evaluaciones' -> (v_total_evaluaciones - 1);
    IF v_ultima ->> 'participacion_ref' IS DISTINCT FROM
          v_propuesta ->> 'participacion_seleccionada_ref' OR
       v_ultima ->> 'sujeto_ref' IS DISTINCT FROM
          v_propuesta ->> 'sujeto_seleccionado_ref' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'seleccion no corresponde al primer elegible';
    END IF;

    v_recurso_canonico := convert_to(
        '{"ambitos":{"categoria_ref":' ||
        to_json(v_necesidad.categoria_ref)::text || ',"unidad_ref":' ||
        to_json(v_necesidad.unidad_ref)::text || '},"atributos":{}}',
        'UTF8'
    );
    IF vec_autorizacion.revalidar_decision_bolsa_llamamientos_v1(
        p_prueba, p_decision_canonica, v_recurso_canonico,
        'bolsa.llamamiento.proponer', 'necesidad_cobertura',
        v_necesidad.necesidad_ref, '[]'::jsonb, v_ahora
    ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'autorizacion no revalidada';
    END IF;

    SELECT version_atestacion.* INTO STRICT v_atestacion
      FROM vec_bolsa_llamamientos.atestacion_autorizacion_actual AS actual
      JOIN vec_bolsa_llamamientos.atestacion_autorizacion_version AS version_atestacion
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
     FOR UPDATE OF actual, version_atestacion;

    SELECT propuesta_almacen.*, uso.consumo_ref, uso.consumo_canonico,
           uso.huella_consumo_sha256, uso.atestacion_ref,
           uso.atestacion_version, auditoria_almacen.auditoria_ref,
           auditoria_almacen.registro_canonico,
           auditoria_almacen.huella_auditoria_sha256,
           evento.evento_ref, evento.evento_canonico,
           evento.huella_evento_sha256
      INTO v_existente
      FROM vec_bolsa_llamamientos.propuesta AS propuesta_almacen
      JOIN vec_bolsa_llamamientos.uso_decision AS uso
        ON uso.propuesta_ref = propuesta_almacen.propuesta_ref
      JOIN vec_bolsa_llamamientos.auditoria AS auditoria_almacen
        ON auditoria_almacen.consumo_ref = uso.consumo_ref
      JOIN vec_bolsa_llamamientos.outbox AS evento
        ON evento.propuesta_ref = propuesta_almacen.propuesta_ref
     WHERE propuesta_almacen.propuesta_ref = p_operacion ->> 'propuesta_ref'
     FOR SHARE OF propuesta_almacen, uso, auditoria_almacen, evento;
    IF FOUND THEN
        IF v_existente.propuesta_canonica IS DISTINCT FROM p_propuesta_canonica OR
           v_existente.decision_ref IS DISTINCT FROM p_prueba ->> 'decision_ref' OR
           v_existente.atestacion_ref IS DISTINCT FROM v_atestacion.atestacion_ref OR
           v_existente.atestacion_version <> v_atestacion.version THEN
            RAISE EXCEPTION USING ERRCODE = 'PBL01',
                MESSAGE = 'referencia de propuesta ya usada';
        END IF;
        RETURN QUERY SELECT 'repetida'::text, v_existente.propuesta_ref,
            v_existente.huella_propuesta_sha256,
            v_existente.propuesta_canonica, v_existente.huella_documento_sha256,
            v_existente.decision_ref, p_prueba ->> 'huella_decision_sha256',
            v_atestacion.atestacion_ref, v_atestacion.evidencia_canonica,
            v_atestacion.huella_evidencia_sha256, v_existente.consumo_ref,
            v_existente.consumo_canonico, v_existente.huella_consumo_sha256,
            v_existente.auditoria_ref, v_existente.registro_canonico,
            v_existente.huella_auditoria_sha256, v_existente.evento_ref,
            v_existente.evento_canonico, v_existente.huella_evento_sha256,
            v_existente.confirmada_en;
        RETURN;
    END IF;
    IF EXISTS (
        SELECT 1 FROM vec_bolsa_llamamientos.propuesta
         WHERE necesidad_ref = v_necesidad.necesidad_ref
           AND version_necesidad = v_necesidad.version
           AND huella_necesidad_sha256 = v_necesidad.huella_necesidad_sha256
    ) THEN
        RAISE EXCEPTION USING ERRCODE = 'PBL02',
            MESSAGE = 'version de necesidad ya propuesta';
    END IF;
    IF EXISTS (
        SELECT 1 FROM vec_bolsa_llamamientos.uso_decision AS uso_existente
         WHERE uso_existente.decision_ref = p_prueba ->> 'decision_ref'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = 'PBL03',
            MESSAGE = 'decision ya consumida';
    END IF;
    IF EXISTS (
        SELECT 1 FROM vec_bolsa_llamamientos.referencia_consumida
         WHERE referencia = v_instantanea.instantanea_ref OR referencia IN (
             SELECT referencias.valor
               FROM jsonb_array_elements(v_propuesta -> 'evaluaciones')
                    AS evaluacion(valor_evaluacion),
             LATERAL (VALUES (
                 evaluacion.valor_evaluacion ->> 'entrada_evaluacion_ref'
             ), (
                 evaluacion.valor_evaluacion ->> 'resultado_evaluacion_ref'
             )) AS referencias(valor)
         )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = 'PBL04',
            MESSAGE = 'recibo o instantanea ya consumidos';
    END IF;

    v_consumo_ref := 'consumo-' || encode(sha256(convert_to(
        (p_prueba ->> 'decision_ref') || ':' ||
        (p_operacion ->> 'huella_documento_sha256'), 'UTF8'
    )), 'hex');
    v_consumo_canonico := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.llamamiento.consumo.v1',
        'consumo_ref', v_consumo_ref,
        'decision_ref', p_prueba ->> 'decision_ref',
        'principal_ref', p_prueba ->> 'principal_ref',
        'propuesta_ref', p_operacion ->> 'propuesta_ref',
        'necesidad_ref', v_necesidad.necesidad_ref,
        'version_necesidad', v_necesidad.version,
        'huella_necesidad_sha256', v_necesidad.huella_necesidad_sha256,
        'huella_propuesta_sha256', p_operacion ->> 'huella_propuesta_sha256',
        'huella_documento_sha256', p_operacion ->> 'huella_documento_sha256',
        'atestacion_ref', v_atestacion.atestacion_ref,
        'huella_atestacion_sha256', v_atestacion.huella_evidencia_sha256,
        'consumida_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_consumo := encode(sha256(v_consumo_canonico), 'hex');

    INSERT INTO vec_bolsa_llamamientos.propuesta VALUES (
        p_operacion ->> 'propuesta_ref', v_necesidad.necesidad_ref,
        v_necesidad.version, v_necesidad.huella_necesidad_sha256,
        v_instantanea.instantanea_ref, v_instantanea.version,
        v_instantanea.huella_instantanea_sha256, p_propuesta_canonica,
        p_operacion ->> 'huella_propuesta_sha256',
        p_operacion ->> 'huella_documento_sha256',
        p_prueba ->> 'decision_ref', v_ahora
    );
    INSERT INTO vec_bolsa_llamamientos.referencia_consumida VALUES (
        v_instantanea.instantanea_ref, 'instantanea',
        p_operacion ->> 'propuesta_ref', v_ahora
    );
    FOR v_indice IN 0..v_total_evaluaciones - 1 LOOP
        INSERT INTO vec_bolsa_llamamientos.referencia_consumida VALUES
          (v_propuesta -> 'evaluaciones' -> v_indice ->> 'entrada_evaluacion_ref',
           'entrada_evaluacion', p_operacion ->> 'propuesta_ref', v_ahora),
          (v_propuesta -> 'evaluaciones' -> v_indice ->> 'resultado_evaluacion_ref',
           'resultado_evaluacion', p_operacion ->> 'propuesta_ref', v_ahora);
    END LOOP;
    INSERT INTO vec_bolsa_llamamientos.uso_decision VALUES (
        p_prueba ->> 'decision_ref', v_consumo_ref,
        p_operacion ->> 'propuesta_ref', v_atestacion.atestacion_ref,
        v_atestacion.version, v_consumo_canonico, v_huella_consumo, v_ahora
    );

    SELECT ultima_secuencia, ultima_huella_sha256
      INTO STRICT v_ultima_secuencia, v_huella_anterior
      FROM vec_bolsa_llamamientos.auditoria_actual
     WHERE control_id FOR UPDATE;
    v_auditoria_ref := 'auditoria-' || encode(sha256(convert_to(
        v_consumo_ref || ':' || v_huella_anterior, 'UTF8'
    )), 'hex');
    v_registro_auditoria := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.llamamiento.auditoria.v1',
        'auditoria_ref', v_auditoria_ref,
        'secuencia', v_ultima_secuencia + 1,
        'huella_anterior_sha256', v_huella_anterior,
        'consumo_ref', v_consumo_ref,
        'decision_ref', p_prueba ->> 'decision_ref',
        'propuesta_ref', p_operacion ->> 'propuesta_ref',
        'huella_propuesta_sha256', p_operacion ->> 'huella_propuesta_sha256',
        'huella_consumo_sha256', v_huella_consumo,
        'registrada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_auditoria := encode(sha256(v_registro_auditoria), 'hex');
    INSERT INTO vec_bolsa_llamamientos.auditoria VALUES (
        v_auditoria_ref, v_ultima_secuencia + 1, v_consumo_ref,
        v_registro_auditoria, v_huella_anterior, v_huella_auditoria, v_ahora
    );
    UPDATE vec_bolsa_llamamientos.auditoria_actual
       SET ultima_secuencia = v_ultima_secuencia + 1,
           ultima_huella_sha256 = v_huella_auditoria,
           actualizada_en = v_ahora
     WHERE control_id;

    v_evento_ref := 'evento-' || encode(sha256(convert_to(
        (p_operacion ->> 'propuesta_ref') || ':' || v_huella_auditoria,
        'UTF8'
    )), 'hex');
    v_evento_canonico := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.llamamiento.outbox.v1',
        'evento_ref', v_evento_ref,
        'tipo', 'bolsa.llamamiento.propuesta_confirmada.v1',
        'agregado_ref', p_operacion ->> 'propuesta_ref',
        'huella_propuesta_sha256', p_operacion ->> 'huella_propuesta_sha256',
        'auditoria_ref', v_auditoria_ref,
        'huella_auditoria_sha256', v_huella_auditoria,
        'emitido_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_evento := encode(sha256(v_evento_canonico), 'hex');
    INSERT INTO vec_bolsa_llamamientos.outbox VALUES (
        v_evento_ref, p_operacion ->> 'propuesta_ref',
        v_evento_canonico, v_huella_evento, v_ahora, NULL
    );

    RETURN QUERY SELECT 'confirmada'::text,
        (p_operacion ->> 'propuesta_ref')::text,
        (p_operacion ->> 'huella_propuesta_sha256')::text,
        p_propuesta_canonica,
        (p_operacion ->> 'huella_documento_sha256')::text,
        (p_prueba ->> 'decision_ref')::text,
        (p_prueba ->> 'huella_decision_sha256')::text,
        v_atestacion.atestacion_ref::text,
        v_atestacion.evidencia_canonica::bytea,
        v_atestacion.huella_evidencia_sha256::text,
        v_consumo_ref, v_consumo_canonico, v_huella_consumo,
        v_auditoria_ref, v_registro_auditoria, v_huella_auditoria,
        v_evento_ref, v_evento_canonico, v_huella_evento, v_ahora;
EXCEPTION
    WHEN no_data_found OR too_many_rows THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'no existe una unica fuente autoritativa';
END
$funcion$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.guardar_propuesta_v1(
    jsonb, jsonb, bytea, bytea
) FROM PUBLIC, vec_bolsa_llamamientos_ejecutor,
    vec_bolsa_llamamientos_proyector_autoritativo,
    vec_bolsa_llamamientos_registrador_atestacion,
    vec_bolsa_llamamientos_despachador_outbox;

COMMENT ON FUNCTION vec_bolsa_llamamientos.guardar_propuesta_v1(
    jsonb, jsonb, bytea, bytea
) IS 'Guardado atomico cerrado: relee fuentes, revalida autoridad, consume decision, encadena auditoria y crea outbox. Sin EXECUTE hasta disponer de registrador COSE real.';
COMMIT;
