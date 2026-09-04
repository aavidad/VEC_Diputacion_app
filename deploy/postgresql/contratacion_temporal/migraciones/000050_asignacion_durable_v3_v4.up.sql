BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000050_asignacion_durable_v3_v4', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
           'vec_contratacion_temporal.expediente_version_integral') IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.expediente_integral_actual') IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.actuacion_expediente_integral') IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.outbox_expediente_integral') IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.registrar_y_consumir_asignacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR NOT pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_propietario',
           'vec_autorizacion_atestada_v3.registrar_y_consumir_asignacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
           'EXECUTE'
       )
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.reserva_asignacion') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para asignacion durable v3-v4';
    END IF;
END
$prevalidacion$;

CREATE TABLE vec_contratacion_temporal.reserva_asignacion (
    ambito_hmac text PRIMARY KEY,
    huella_peticion_hmac text NOT NULL,
    operacion text NOT NULL,
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    version_expediente numeric(20, 0) NOT NULL,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    unidad_ref text NOT NULL,
    responsable_ref text NOT NULL,
    reserva_ref text NOT NULL UNIQUE,
    recibo_ref text NOT NULL UNIQUE,
    notificacion_ref text NOT NULL UNIQUE,
    bandeja_ref text NOT NULL UNIQUE,
    auditoria_ref text NOT NULL UNIQUE,
    evento_ref text NOT NULL UNIQUE,
    expediente_anterior_json jsonb NOT NULL,
    estado text NOT NULL DEFAULT 'reservada',
    reservada_en timestamptz(6) NOT NULL,
    confirmada_en timestamptz(6),
    FOREIGN KEY (expediente_ref, version_expediente)
        REFERENCES vec_contratacion_temporal.expediente_version_integral,
    CHECK (operacion = 'asignar'),
    CHECK (version_expediente = 3),
    CHECK (estado IN ('reservada', 'confirmada')),
    CHECK (pg_catalog.jsonb_typeof(expediente_anterior_json) = 'object'),
    CHECK (reservada_en = pg_catalog.date_trunc('microseconds', reservada_en)),
    CHECK (confirmada_en IS NULL OR
           confirmada_en = pg_catalog.date_trunc('microseconds', confirmada_en)),
    CHECK ((estado = 'reservada' AND confirmada_en IS NULL) OR
           (estado = 'confirmada' AND confirmada_en IS NOT NULL))
);

CREATE TABLE vec_contratacion_temporal.terminal_asignacion (
    ambito_hmac text PRIMARY KEY,
    huella_peticion_hmac text NOT NULL,
    decision_ref text NOT NULL UNIQUE,
    decision_huella_sha256 text NOT NULL,
    consumo_huella_sha256 text NOT NULL UNIQUE,
    recibo_json jsonb NOT NULL,
    terminal_json jsonb NOT NULL,
    auditoria_json jsonb NOT NULL,
    notificacion_json jsonb NOT NULL,
    bandeja_json jsonb NOT NULL,
    carga_huella_sha256 text NOT NULL,
    confirmada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (ambito_hmac)
        REFERENCES vec_contratacion_temporal.reserva_asignacion,
    CHECK (decision_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (carga_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (pg_catalog.jsonb_typeof(recibo_json) = 'object'),
    CHECK (pg_catalog.jsonb_typeof(terminal_json) = 'object'),
    CHECK (pg_catalog.jsonb_typeof(auditoria_json) = 'object'),
    CHECK (pg_catalog.jsonb_typeof(notificacion_json) = 'object'),
    CHECK (pg_catalog.jsonb_typeof(bandeja_json) = 'object'),
    CHECK (confirmada_en = pg_catalog.date_trunc('microseconds', confirmada_en))
);

DO $seguridad$
DECLARE
    v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'reserva_asignacion', 'terminal_asignacion'
    ] LOOP
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE POLICY %I ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true)',
            v_tabla || '_propietario', v_tabla
        );
        IF v_tabla = 'terminal_asignacion' THEN
            EXECUTE pg_catalog.format(
                'CREATE TRIGGER %I BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.%I FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()',
                v_tabla || '_inmutable', v_tabla
            );
        END IF;
    END LOOP;
END
$seguridad$;

CREATE FUNCTION vec_contratacion_temporal.asignacion_claves_exactas_v1(
    p_documento jsonb,
    p_claves text[]
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.jsonb_typeof(p_documento) = 'object'
       AND ARRAY(
           SELECT clave
             FROM pg_catalog.jsonb_object_keys(p_documento) AS k(clave)
            ORDER BY clave
       ) = ARRAY(SELECT clave FROM pg_catalog.unnest(p_claves) clave ORDER BY clave)
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.preparar_asignacion_v1(p_operacion jsonb)
RETURNS TABLE (
    resultado text, expediente_json text, reserva_ref text, recibo_ref text,
    notificacion_ref text, bandeja_ref text, auditoria_ref text, evento_ref text,
    ambito_hmac text, huella_peticion_hmac text, operacion text,
    organizacion_ref text, actor_ref text, perfil_ref text, unidad_ref text,
    responsable_ref text, estado text, version_resultante bigint,
    concesion_v3_decision_ref text, confirmada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_activo jsonb := p_operacion #> '{sellos_hmac,activo}';
    v_retenido jsonb;
    v_reserva vec_contratacion_temporal.reserva_asignacion%ROWTYPE;
    v_actual record;
    v_terminal vec_contratacion_temporal.terminal_asignacion%ROWTYPE;
    v_ahora timestamptz(6);
    v_reserva_encontrada boolean := false;
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
       OR pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.pg_column_size(p_operacion) > 65536
       OR NOT vec_contratacion_temporal.asignacion_claves_exactas_v1(
           p_operacion, ARRAY['actor_ref','esquema','expediente_ref',
           'operacion','organizacion_ref','perfil_ref','referencias_candidatas',
           'responsable_ref','sellos_hmac','unidad_ref','version_expediente'])
       OR p_operacion ->> 'esquema' <>
          'vec.contratacion-temporal.preparar-asignacion.v1'
       OR p_operacion ->> 'operacion' <> 'asignar'
       OR (p_operacion ->> 'version_expediente')::numeric <> 3
       OR NOT vec_contratacion_temporal.asignacion_claves_exactas_v1(
           p_operacion -> 'referencias_candidatas',
           ARRAY['auditoria_ref','bandeja_ref','evento_ref','notificacion_ref',
                 'recibo_ref','reserva_ref'])
       OR NOT vec_contratacion_temporal.asignacion_claves_exactas_v1(
           v_activo, ARRAY['ambito_hmac','generacion','huella_peticion_hmac'])
       OR coalesce(v_activo ->> 'ambito_hmac', '') !~
          '^hmac-sha256:vec[.]contratacion-temporal[.]asignacion[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
       OR coalesce(v_activo ->> 'huella_peticion_hmac', '') !~
          '^hmac-sha256:vec[.]contratacion-temporal[.]asignacion[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'preparacion de asignacion no autorizada';
    END IF;

    SELECT r.* INTO v_reserva
      FROM vec_contratacion_temporal.reserva_asignacion r
     WHERE r.ambito_hmac = v_activo ->> 'ambito_hmac'
     FOR UPDATE;
    v_reserva_encontrada := FOUND;
    IF NOT v_reserva_encontrada THEN
        FOR v_retenido IN
            SELECT valor
              FROM pg_catalog.jsonb_array_elements(
                  coalesce(p_operacion #> '{sellos_hmac,retenidos}', '[]'::jsonb)
              ) AS e(valor)
        LOOP
            SELECT r.* INTO v_reserva
              FROM vec_contratacion_temporal.reserva_asignacion r
             WHERE r.ambito_hmac = v_retenido ->> 'ambito_hmac'
             FOR UPDATE;
            IF FOUND THEN
                v_reserva_encontrada := true;
                EXIT;
            END IF;
        END LOOP;
    END IF;

    IF v_reserva_encontrada THEN
        IF v_reserva.huella_peticion_hmac IS DISTINCT FROM
              coalesce(v_retenido ->> 'huella_peticion_hmac',
                       v_activo ->> 'huella_peticion_hmac')
           OR v_reserva.operacion IS DISTINCT FROM p_operacion ->> 'operacion'
           OR v_reserva.organizacion_ref IS DISTINCT FROM p_operacion ->> 'organizacion_ref'
           OR v_reserva.expediente_ref IS DISTINCT FROM p_operacion ->> 'expediente_ref'
           OR v_reserva.version_expediente IS DISTINCT FROM
              (p_operacion ->> 'version_expediente')::numeric
           OR v_reserva.actor_ref IS DISTINCT FROM p_operacion ->> 'actor_ref'
           OR v_reserva.perfil_ref IS DISTINCT FROM p_operacion ->> 'perfil_ref'
           OR v_reserva.unidad_ref IS DISTINCT FROM p_operacion ->> 'unidad_ref'
           OR v_reserva.responsable_ref IS DISTINCT FROM p_operacion ->> 'responsable_ref' THEN
            resultado := 'idempotencia_reutilizada';
        ELSIF v_reserva.estado = 'confirmada' THEN
            resultado := 'confirmada';
        ELSE
            resultado := 'reutilizada';
        END IF;
    ELSE
        SELECT a.version, v.agregado_json
          INTO STRICT v_actual
          FROM vec_contratacion_temporal.expediente_integral_actual a
          JOIN vec_contratacion_temporal.expediente_version_integral v
            USING (expediente_ref, version)
         WHERE a.expediente_ref = p_operacion ->> 'expediente_ref'
         FOR UPDATE OF a, v;
        IF v_actual.version <> 3
           OR v_actual.agregado_json ->> 'organizacion_ref' <>
              p_operacion ->> 'organizacion_ref'
           OR v_actual.agregado_json ->> 'fase_actual' <> 'asignacion_unidad'
           OR v_actual.agregado_json ? 'asignacion' THEN
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'expediente no asignable en version 3';
        END IF;
        v_ahora := pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp());
        INSERT INTO vec_contratacion_temporal.reserva_asignacion (
            ambito_hmac, huella_peticion_hmac, operacion, organizacion_ref,
            expediente_ref, version_expediente, actor_ref, perfil_ref,
            unidad_ref, responsable_ref, reserva_ref, recibo_ref,
            notificacion_ref, bandeja_ref, auditoria_ref, evento_ref,
            expediente_anterior_json, reservada_en
        ) VALUES (
            v_activo ->> 'ambito_hmac', v_activo ->> 'huella_peticion_hmac',
            p_operacion ->> 'operacion', p_operacion ->> 'organizacion_ref',
            p_operacion ->> 'expediente_ref', 3,
            p_operacion ->> 'actor_ref', p_operacion ->> 'perfil_ref',
            p_operacion ->> 'unidad_ref', p_operacion ->> 'responsable_ref',
            p_operacion #>> '{referencias_candidatas,reserva_ref}',
            p_operacion #>> '{referencias_candidatas,recibo_ref}',
            p_operacion #>> '{referencias_candidatas,notificacion_ref}',
            p_operacion #>> '{referencias_candidatas,bandeja_ref}',
            p_operacion #>> '{referencias_candidatas,auditoria_ref}',
            p_operacion #>> '{referencias_candidatas,evento_ref}',
            v_actual.agregado_json, v_ahora
        ) RETURNING * INTO v_reserva;
        resultado := 'reservada';
    END IF;

    IF v_reserva.estado = 'confirmada' THEN
        SELECT t.* INTO STRICT v_terminal
          FROM vec_contratacion_temporal.terminal_asignacion t
         WHERE t.ambito_hmac = v_reserva.ambito_hmac;
    END IF;
    RETURN QUERY SELECT resultado, v_reserva.expediente_anterior_json::text,
        v_reserva.reserva_ref, v_reserva.recibo_ref,
        v_reserva.notificacion_ref, v_reserva.bandeja_ref,
        v_reserva.auditoria_ref, v_reserva.evento_ref,
        v_reserva.ambito_hmac, v_reserva.huella_peticion_hmac,
        v_reserva.operacion, v_reserva.organizacion_ref,
        v_reserva.actor_ref, v_reserva.perfil_ref, v_reserva.unidad_ref,
        v_reserva.responsable_ref, v_reserva.estado,
        CASE WHEN v_reserva.estado = 'confirmada' THEN 4::bigint END,
        CASE WHEN v_reserva.estado = 'confirmada'
             THEN v_terminal.decision_ref END,
        v_reserva.confirmada_en;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.consultar_asignacion_v1(p_consulta jsonb)
RETURNS TABLE (terminal_json jsonb)
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
AS $funcion$
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR p_consulta ->> 'esquema' <>
          'vec.contratacion-temporal.consultar-asignacion.v1' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de asignacion no autorizada';
    END IF;
    RETURN QUERY
    SELECT t.terminal_json
      FROM vec_contratacion_temporal.reserva_asignacion r
      JOIN vec_contratacion_temporal.terminal_asignacion t USING (ambito_hmac)
     WHERE r.estado = 'confirmada'
       AND r.ambito_hmac = p_consulta ->> 'ambito_idempotencia_hmac_activo'
       AND r.huella_peticion_hmac = p_consulta ->> 'huella_peticion_hmac_activa'
       AND r.operacion = p_consulta ->> 'operacion'
       AND r.organizacion_ref = p_consulta ->> 'organizacion_ref'
       AND r.expediente_ref = p_consulta ->> 'expediente_ref'
       AND r.version_expediente = (p_consulta ->> 'version_expediente')::numeric
       AND r.actor_ref = p_consulta ->> 'actor_ref'
       AND r.perfil_ref = p_consulta ->> 'perfil_ref'
       AND r.unidad_ref = p_consulta ->> 'unidad_ref'
       AND r.responsable_ref = p_consulta ->> 'responsable_ref';
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.confirmar_asignacion_v1(
    p_operacion jsonb,
    p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
    p_persona_version numeric, p_perfil_version numeric,
    p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea
)
RETURNS TABLE (recibo_json jsonb)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    r vec_contratacion_temporal.reserva_asignacion%ROWTYPE;
    t vec_contratacion_temporal.terminal_asignacion%ROWTYPE;
    v_actual record;
    v_consumo record;
    v_ahora timestamptz(6);
    v_carga_huella text;
    v_agregado_huella text;
    v_prueba bytea;
    v_payload bytea;
    v_anterior text;
    v_secuencia numeric;
    v_recibo jsonb;
    v_terminal jsonb;
    v_actuacion jsonb;
    v_expediente_esperado jsonb;
    v_decision jsonb;
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR p_operacion ->> 'esquema' <>
          'vec.contratacion-temporal.confirmar-asignacion.v1' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'confirmacion de asignacion no autorizada';
    END IF;
    BEGIN
        v_decision := pg_catalog.convert_from(p_decision, 'UTF8')::jsonb;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
          OR character_not_in_repertoire OR untranslatable_character THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'decision de asignacion invalida';
    END;
    v_carga_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(p_operacion::text, 'UTF8')), 'hex');
    SELECT x.* INTO STRICT r
      FROM vec_contratacion_temporal.reserva_asignacion x
     WHERE x.ambito_hmac = p_operacion ->> 'ambito_idempotencia_hmac'
       AND x.huella_peticion_hmac = p_operacion ->> 'huella_peticion_hmac'
     FOR UPDATE;
    IF r.estado = 'confirmada' THEN
        SELECT x.* INTO STRICT t
          FROM vec_contratacion_temporal.terminal_asignacion x
         WHERE x.ambito_hmac = r.ambito_hmac;
        IF t.carga_huella_sha256 <> v_carga_huella THEN
            RAISE EXCEPTION USING ERRCODE = '23505',
                MESSAGE = 'replay de asignacion divergente';
        END IF;
        RETURN QUERY SELECT t.recibo_json;
        RETURN;
    END IF;
    IF r.operacion <> 'asignar'
       OR r.organizacion_ref <> p_operacion ->> 'organizacion_ref'
       OR r.expediente_ref <> p_operacion ->> 'expediente_ref'
       OR r.version_expediente <> (p_operacion ->> 'version_anterior')::numeric
       OR r.actor_ref <> p_operacion ->> 'actor_ref'
       OR r.perfil_ref <> p_operacion ->> 'perfil_ref'
       OR r.unidad_ref <> p_operacion ->> 'unidad_ref'
       OR r.responsable_ref <> p_operacion ->> 'responsable_ref'
       OR r.expediente_anterior_json <> p_operacion -> 'expediente_anterior'
       OR p_operacion #>> '{politica,accion}' <>
          'contratacion_temporal.unidad.asignar'
       OR p_operacion #>> '{autorizacion,accion}' <>
          'contratacion_temporal.unidad.asignar'
       OR p_operacion #>> '{politica,finalidad}' <>
          'gestionar_contratacion_temporal'
       OR p_operacion #>> '{autorizacion,finalidad}' <>
          'gestionar_contratacion_temporal'
       OR v_decision ->> 'principal_id' <> r.actor_ref
       OR v_decision ->> 'perfil_activo_ref' <> r.perfil_ref
       OR v_decision ->> 'recurso_ref' <> r.expediente_ref
       OR v_decision ->> 'accion' <>
          'contratacion_temporal.unidad.asignar'
       OR v_decision ->> 'modulo_id' <> 'contratacion_temporal'
       OR v_decision ->> 'tipo_recurso' <>
          'asignacion_contratacion_temporal'
       OR v_decision ->> 'finalidad' <> 'gestionar_contratacion_temporal'
       OR v_decision ->> 'decision_ref' <>
          p_operacion #>> '{autorizacion,decision_ref}'
       OR v_decision ->> 'contexto_recurso_huella_sha256' <>
          p_operacion #>> '{autorizacion,contexto_recurso_huella_sha256}'
       OR p_operacion #>> '{autorizacion,principal_id}' <> r.actor_ref
       OR p_operacion #>> '{autorizacion,perfil_activo_ref}' <> r.perfil_ref
       OR p_operacion #>> '{autorizacion,recurso_ref}' <> r.expediente_ref
       OR p_operacion #>> '{autorizacion,decision_canonica_hex}' <>
          pg_catalog.encode(p_decision, 'hex')
       OR p_operacion #>> '{autorizacion,motivo_canonico_hex}' <>
          pg_catalog.encode(p_motivo, 'hex')
       OR (p_operacion #>> '{autorizacion,persona_version}')::numeric <>
          p_persona_version
       OR (p_operacion #>> '{autorizacion,perfil_version}')::numeric <>
          p_perfil_version
       OR p_operacion #>> '{autorizacion,decision_huella_sha256}' <>
          pg_catalog.encode(pg_catalog.sha256(p_decision), 'hex') THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'coordenadas de asignacion divergentes';
    END IF;
    SELECT v.* INTO STRICT v_actual
      FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v
        USING (expediente_ref, version)
     WHERE a.expediente_ref = r.expediente_ref
     FOR UPDATE OF a, v;
    IF v_actual.version <> 3
       OR v_actual.agregado_json <> r.expediente_anterior_json THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'CAS de asignacion perdido';
    END IF;

    v_actuacion := pg_catalog.jsonb_build_object(
        'secuencia', pg_catalog.jsonb_array_length(
            v_actual.agregado_json -> 'actuaciones') + 1,
        'version_expediente', 4,
        'accion_clave', 'contratacion_temporal.unidad.asignar',
        'actor_ref', r.actor_ref,
        'unidad_ref', p_operacion #>> '{politica,unidad_ejecutora_ref}',
        'recibo_ref', r.recibo_ref,
        'realizada_en', p_operacion -> 'instante_efecto',
        'fase_origen', v_actual.agregado_json -> 'fase_actual',
        'fase_destino', v_actual.agregado_json -> 'fase_actual',
        'estado_origen', v_actual.agregado_json -> 'estado_actual',
        'estado_destino', v_actual.agregado_json -> 'estado_actual'
    );
    v_expediente_esperado := pg_catalog.jsonb_set(
        pg_catalog.jsonb_set(
            pg_catalog.jsonb_set(
                pg_catalog.jsonb_set(
                    v_actual.agregado_json,
                    '{version}', pg_catalog.to_jsonb(4::bigint), false
                ),
                '{actualizado_en}', p_operacion -> 'instante_efecto', false
            ),
            '{actuaciones}',
            (v_actual.agregado_json -> 'actuaciones') ||
                pg_catalog.jsonb_build_array(v_actuacion),
            false
        ),
        '{asignacion}',
        pg_catalog.jsonb_build_object(
            'unidad_ref', r.unidad_ref,
            'responsable_ref', r.responsable_ref,
            'notificacion_ref', r.notificacion_ref,
            'asignada_en', p_operacion -> 'instante_efecto',
            'actuacion_registro', pg_catalog.jsonb_build_object(
                'secuencia', pg_catalog.jsonb_array_length(
                    v_actual.agregado_json -> 'actuaciones') + 1,
                'version_expediente', 4,
                'accion_clave', 'contratacion_temporal.unidad.asignar',
                'fase_destino', v_actual.agregado_json -> 'fase_actual',
                'recibo_ref', r.recibo_ref,
                'unidad_asignada_ref', r.unidad_ref,
                'responsable_asignado_ref', r.responsable_ref,
                'notificacion_ref', r.notificacion_ref
            )
        ),
        true
    );
    IF p_operacion -> 'actuacion' IS DISTINCT FROM v_actuacion
       OR p_operacion -> 'expediente_siguiente' IS DISTINCT FROM
          v_expediente_esperado THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'proyeccion de asignacion divergente';
    END IF;

    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3.registrar_y_consumir_asignacion_v3_atestada(
          p_capacidad, p_decision, p_motivo, p_contexto,
          p_persona_version, p_perfil_version, p_payload, p_sobre,
          p_evidencia, p_raiz
      );
    IF v_consumo.decision_ref <>
          p_operacion #>> '{autorizacion,decision_ref}'
       OR v_consumo.efecto_ref <> r.expediente_ref
       OR v_consumo.huella_efecto_sha256 <>
          p_operacion #>> '{autorizacion,contexto_recurso_huella_sha256}' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consumo de autorizacion de asignacion divergente';
    END IF;
    v_ahora := pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp());
    v_agregado_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to((p_operacion -> 'expediente_siguiente')::text, 'UTF8')), 'hex');
    v_prueba := pg_catalog.convert_to(
        'VEC-CT-EXPEDIENTE-ASIGNACION-V1' || chr(10) ||
        r.expediente_ref || chr(10) || '4' || chr(10) || v_agregado_huella ||
        chr(10) || r.reserva_ref || chr(10) || r.recibo_ref || chr(10) ||
        v_consumo.decision_ref || chr(10) || v_ahora::text, 'UTF8');
    INSERT INTO vec_contratacion_temporal.expediente_version_integral (
        expediente_ref, version, agregado_json, agregado_json_huella_sha256,
        prueba_canonica, prueba_huella_sha256, flujo_ref, flujo_version,
        flujo_huella_sha256, fase_clave, estado, origen_version,
        operacion_ref, registrada_en
    ) VALUES (
        r.expediente_ref, 4, p_operacion -> 'expediente_siguiente',
        v_agregado_huella, v_prueba,
        pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex'),
        v_actual.flujo_ref, v_actual.flujo_version,
        v_actual.flujo_huella_sha256,
        p_operacion #>> '{expediente_siguiente,fase_actual}',
        p_operacion #>> '{expediente_siguiente,estado_actual}',
        'asignacion_o5', r.reserva_ref, v_ahora
    );
    UPDATE vec_contratacion_temporal.expediente_integral_actual
       SET version = 4, actualizada_en = v_ahora, operacion_ref = r.reserva_ref
     WHERE expediente_ref = r.expediente_ref AND version = 3;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001', MESSAGE = 'CAS final perdido';
    END IF;
    v_prueba := pg_catalog.convert_to(
        'VEC-CT-ACTUACION-ASIGNACION-V1' || chr(10) ||
        (p_operacion -> 'actuacion')::text || chr(10) || r.recibo_ref ||
        chr(10) || v_ahora::text, 'UTF8');
    INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral (
        expediente_ref, secuencia, version_expediente, operacion_ref,
        recibo_ref, actuacion_json, actuacion_json_huella_sha256,
        prueba_canonica, prueba_huella_sha256, registrada_en
    ) VALUES (
        r.expediente_ref, 4, 4, r.reserva_ref, r.recibo_ref,
        p_operacion -> 'actuacion', pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to((p_operacion -> 'actuacion')::text, 'UTF8')), 'hex'),
        v_prueba, pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex'), v_ahora
    );
    SELECT secuencia_outbox, cabeza_outbox_sha256
      INTO STRICT v_secuencia, v_anterior
      FROM vec_contratacion_temporal.control_cadenas_expediente_integral
     WHERE control_id FOR UPDATE;
    v_secuencia := v_secuencia + 1;
    v_payload := pg_catalog.convert_to(pg_catalog.jsonb_build_object(
        'esquema', 'vec.contratacion-temporal.asignacion-confirmada.v1',
        'expediente_ref', r.expediente_ref, 'version_resultante', 4,
        'recibo_ref', r.recibo_ref, 'unidad_ref', r.unidad_ref,
        'responsable_ref', r.responsable_ref)::text, 'UTF8');
    INSERT INTO vec_contratacion_temporal.outbox_expediente_integral (
        evento_ref, secuencia, operacion_ref, expediente_ref,
        version_expediente, tipo_evento, payload_canonico,
        payload_huella_sha256, anterior_sha256, huella_sha256, registrada_en
    ) VALUES (
        r.evento_ref, v_secuencia, r.reserva_ref, r.expediente_ref, 4,
        'contratacion_temporal.asignacion_confirmada', v_payload,
        pg_catalog.encode(pg_catalog.sha256(v_payload), 'hex'), v_anterior,
        pg_catalog.encode(pg_catalog.sha256(v_anterior::bytea || v_payload), 'hex'),
        v_ahora
    );
    UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral
       SET secuencia_outbox = v_secuencia,
           cabeza_outbox_sha256 = pg_catalog.encode(
               pg_catalog.sha256(v_anterior::bytea || v_payload), 'hex'),
           actualizada_en = v_ahora
     WHERE control_id;
    v_recibo := pg_catalog.jsonb_build_object(
        'operacion', r.operacion, 'organizacion_ref', r.organizacion_ref,
        'expediente_ref', r.expediente_ref, 'version_anterior', 3,
        'version_resultante', 4, 'unidad_ref', r.unidad_ref,
        'responsable_ref', r.responsable_ref, 'recibo_ref', r.recibo_ref,
        'notificacion_ref', r.notificacion_ref, 'bandeja_ref', r.bandeja_ref,
        'auditoria_ref', r.auditoria_ref, 'evento_ref', r.evento_ref,
        'concesion_v3_decision_ref', v_consumo.decision_ref,
        'ambito_idempotencia_hmac', r.ambito_hmac,
        'huella_peticion_hmac', r.huella_peticion_hmac,
        'confirmada_en', v_ahora);
    v_terminal := pg_catalog.jsonb_build_object(
        'expediente_anterior', r.expediente_anterior_json,
        'recibo', v_recibo,
        'referencias', pg_catalog.jsonb_build_object(
            'ReservaRef', r.reserva_ref, 'ReciboRef', r.recibo_ref,
            'NotificacionRef', r.notificacion_ref, 'BandejaRef', r.bandeja_ref,
            'AuditoriaRef', r.auditoria_ref, 'EventoRef', r.evento_ref),
        'ambito_idempotencia_hmac', r.ambito_hmac,
        'huella_peticion_hmac', r.huella_peticion_hmac,
        'operacion', r.operacion, 'organizacion_ref', r.organizacion_ref,
        'actor_ref', r.actor_ref, 'perfil_ref', r.perfil_ref,
        'unidad_ref', r.unidad_ref, 'responsable_ref', r.responsable_ref,
        'destino_evidencia_ref', p_operacion #>> '{destino,evidencia_ref}',
        'destino_evidencia_huella_sha256',
            p_operacion #>> '{destino,evidencia_huella_sha256}',
        'politica_ref', p_operacion #>> '{politica,definicion_ref}',
        'politica_version', (p_operacion #>> '{politica,definicion_version}')::numeric,
        'politica_huella_sha256', p_operacion #>> '{politica,definicion_huella_sha256}',
        'finalidad', p_operacion #>> '{politica,finalidad}');
    INSERT INTO vec_contratacion_temporal.terminal_asignacion (
        ambito_hmac, huella_peticion_hmac, decision_ref,
        decision_huella_sha256, consumo_huella_sha256, recibo_json,
        terminal_json, auditoria_json, notificacion_json, bandeja_json,
        carga_huella_sha256, confirmada_en
    ) VALUES (
        r.ambito_hmac, r.huella_peticion_hmac, v_consumo.decision_ref,
        p_operacion #>> '{autorizacion,decision_huella_sha256}',
        v_consumo.consumo_huella_sha256, v_recibo, v_terminal,
        pg_catalog.jsonb_build_object('auditoria_ref', r.auditoria_ref,
            'decision_ref', v_consumo.decision_ref, 'registrada_en', v_ahora),
        pg_catalog.jsonb_build_object('notificacion_ref', r.notificacion_ref,
            'estado', 'pendiente', 'creada_en', v_ahora),
        pg_catalog.jsonb_build_object('bandeja_ref', r.bandeja_ref,
            'unidad_ref', r.unidad_ref, 'estado', 'pendiente', 'creada_en', v_ahora),
        v_carga_huella, v_ahora
    );
    UPDATE vec_contratacion_temporal.reserva_asignacion
       SET estado = 'confirmada', confirmada_en = v_ahora
     WHERE ambito_hmac = r.ambito_hmac AND estado = 'reservada';
    RETURN QUERY SELECT v_recibo;
END
$funcion$;

REVOKE ALL ON TABLE
    vec_contratacion_temporal.reserva_asignacion,
    vec_contratacion_temporal.terminal_asignacion
FROM PUBLIC, vec_contratacion_temporal_ejecutor;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.asignacion_claves_exactas_v1(jsonb,text[]),
    vec_contratacion_temporal.preparar_asignacion_v1(jsonb),
    vec_contratacion_temporal.consultar_asignacion_v1(jsonb),
    vec_contratacion_temporal.confirmar_asignacion_v1(
        jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.preparar_asignacion_v1(jsonb),
    vec_contratacion_temporal.consultar_asignacion_v1(jsonb),
    vec_contratacion_temporal.confirmar_asignacion_v1(
        jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
TO vec_contratacion_temporal_ejecutor;

COMMIT;
