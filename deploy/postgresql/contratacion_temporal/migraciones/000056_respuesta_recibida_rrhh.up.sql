\set ON_ERROR_STOP on
-- Declaración de respuesta recibida por correo registrada por RRHH.
-- No verifica origen, firma ni custodia; no resuelve aceptación/renuncia.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000056_respuesta_recibida_rrhh', 0));

CREATE TABLE vec_contratacion_temporal.respuesta_recibida_rrhh (
    justificante_ref text PRIMARY KEY,
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    llamamiento_ref text NOT NULL,
    comunicacion_ref text NOT NULL REFERENCES vec_contratacion_temporal.comunicacion_llamamiento_local,
    seleccion_clave uuid NOT NULL REFERENCES vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6,
    clave_idempotencia uuid NOT NULL,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    version_comunicacion numeric(20,0) NOT NULL CHECK (version_comunicacion = 2),
    respuesta text NOT NULL CHECK (respuesta IN ('aceptacion', 'renuncia')),
    correo_ref text NOT NULL CHECK (correo_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    correo_sha256 text NOT NULL CHECK (correo_sha256 ~ '^[0-9a-f]{64}$' AND correo_sha256 <> repeat('0',64)),
    recibida_en timestamptz(6) NOT NULL CHECK (isfinite(recibida_en) AND recibida_en <> '0001-01-01T00:00:00Z'::timestamptz),
    -- Se conservan los bytes JSON de Go, no una reserialización de jsonb.
    material text NOT NULL CHECK (octet_length(material) BETWEEN 1 AND 16384),
    material_json jsonb NOT NULL CHECK (jsonb_typeof(material_json) = 'object' AND material_json = material::jsonb),
    material_huella_sha256 text NOT NULL CHECK (material_huella_sha256 = encode(sha256(convert_to(material,'UTF8')),'hex')),
    recibo_ref text NOT NULL UNIQUE,
    recibo_json jsonb NOT NULL CHECK (jsonb_typeof(recibo_json) = 'object'),
    estado text NOT NULL CHECK (estado = 'registrada_por_rrhh'),
    registrada_en timestamptz(6) NOT NULL CHECK (isfinite(registrada_en) AND recibida_en <= registrada_en),
    UNIQUE (organizacion_ref, clave_idempotencia),
    UNIQUE (organizacion_ref, comunicacion_ref)
);
CREATE TABLE vec_contratacion_temporal.historia_respuesta_recibida_rrhh (
    auditoria_ref text PRIMARY KEY,
    justificante_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.respuesta_recibida_rrhh,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    decision_ref text NOT NULL UNIQUE,
    consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
    evidencia_huella_sha256 text NOT NULL CHECK (evidencia_huella_sha256 ~ '^[0-9a-f]{64}$'),
    material_huella_sha256 text NOT NULL CHECK (material_huella_sha256 ~ '^[0-9a-f]{64}$'),
    registrada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_contratacion_temporal.outbox_respuesta_recibida_rrhh (
    evento_ref text PRIMARY KEY,
    justificante_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.respuesta_recibida_rrhh,
    estado text NOT NULL CHECK (estado = 'pendiente'),
    tipo text NOT NULL CHECK (tipo = 'llamamiento.respuesta.registrada_por_rrhh'),
    carga_json jsonb NOT NULL CHECK (jsonb_typeof(carga_json) = 'object'),
    creada_en timestamptz(6) NOT NULL
);
COMMENT ON TABLE vec_contratacion_temporal.respuesta_recibida_rrhh IS
    'Declaración append-only de RRHH. Referencia y SHA256 declarados, sin verificar origen, firma ni custodia del correo; no aceptación terminal.';
COMMENT ON TABLE vec_contratacion_temporal.outbox_respuesta_recibida_rrhh IS
    'Evento de registro de declaración; no acredita envío, entrega, apertura de plazo ni cambio de estado en Bolsa o expediente.';

DO $seguridad$
DECLARE v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'respuesta_recibida_rrhh', 'historia_respuesta_recibida_rrhh', 'outbox_respuesta_recibida_rrhh'
    ] LOOP
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY', v_tabla);
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY', v_tabla);
        EXECUTE format('CREATE POLICY propietario ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true)', v_tabla);
        EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.%I FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()', v_tabla);
        EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM PUBLIC, vec_contratacion_temporal_ejecutor', v_tabla);
    END LOOP;
END
$seguridad$;

CREATE FUNCTION vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v1(
    p_material text,
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    s jsonb; d jsonb; v_referencia text;
    v_material_huella text; v_contexto_huella text;
    v_consumo record;
    v_comunicacion vec_contratacion_temporal.comunicacion_llamamiento_local%ROWTYPE;
    v_seleccion vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
    v_previa vec_contratacion_temporal.respuesta_recibida_rrhh%ROWTYPE;
    v_recibida timestamptz(6); v_ahora timestamptz(6);
    v_justificante text; v_recibo text; v_evento text; v_resultado jsonb;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR session_user = current_user
       OR NOT pg_has_role(session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_has_role(session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
       OR pg_has_role(session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
       OR current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION 'registro de respuesta denegado' USING ERRCODE = 'P0563';
    END IF;
    IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 16384 THEN
        RAISE EXCEPTION 'material de respuesta inválido' USING ERRCODE = 'P0560';
    END IF;
    BEGIN
        s := p_material::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'JSON de respuesta inválido' USING ERRCODE = 'P0560';
    END;
    IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(s, ARRAY[
        'ClaveIdempotencia','OrganizacionRef','ExpedienteRef','LlamamientoRef',
        'ComunicacionRef','VersionComunicacionEsperada','Respuesta','CorreoRef','CorreoSHA256','RecibidaEn'
    ]) IS NOT TRUE THEN
        RAISE EXCEPTION 'campos de respuesta inválidos' USING ERRCODE = 'P0560';
    END IF;
    -- jsonb elimina claves repetidas: comprobar también el documento original.
    IF (SELECT count(*) FROM json_each(p_material::json)) <> 10
       OR jsonb_typeof(s -> 'ClaveIdempotencia') IS DISTINCT FROM 'string'
       OR (s ->> 'ClaveIdempotencia') !~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
       OR s ->> 'ClaveIdempotencia' = '00000000-0000-4000-8000-000000000000'
       OR jsonb_typeof(s -> 'VersionComunicacionEsperada') IS DISTINCT FROM 'number'
       OR (s ->> 'VersionComunicacionEsperada') !~ '^[1-9][0-9]{0,19}$'
       OR jsonb_typeof(s -> 'Respuesta') IS DISTINCT FROM 'string'
       OR (s ->> 'Respuesta') NOT IN ('aceptacion','renuncia')
       OR jsonb_typeof(s -> 'CorreoSHA256') IS DISTINCT FROM 'string'
       OR (s ->> 'CorreoSHA256') !~ '^[0-9a-f]{64}$'
       OR s ->> 'CorreoSHA256' = repeat('0',64) THEN
        RAISE EXCEPTION 'solicitud de respuesta inválida' USING ERRCODE = 'P0560';
    END IF;
    FOREACH v_referencia IN ARRAY ARRAY[
        'OrganizacionRef','ExpedienteRef','LlamamientoRef','ComunicacionRef','CorreoRef'
    ] LOOP
        IF jsonb_typeof(s -> v_referencia) IS DISTINCT FROM 'string'
           OR (s ->> v_referencia) !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
            RAISE EXCEPTION 'referencia de respuesta inválida' USING ERRCODE = 'P0560';
        END IF;
    END LOOP;
    IF s -> 'VersionComunicacionEsperada' IS DISTINCT FROM '2'::jsonb THEN
        RAISE EXCEPTION 'versión de comunicación incompatible' USING ERRCODE = 'P0562';
    END IF;
    IF jsonb_typeof(s -> 'RecibidaEn') IS DISTINCT FROM 'string'
       OR (s ->> 'RecibidaEn') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]([.][0-9]{1,6})?Z$' THEN
        RAISE EXCEPTION 'fecha de recepción debe ser UTC con precisión máxima de microsegundos' USING ERRCODE = 'P0560';
    END IF;
    BEGIN
        v_recibida := (s ->> 'RecibidaEn')::timestamptz;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'fecha de recepción inválida' USING ERRCODE = 'P0560';
    END;
    IF NOT isfinite(v_recibida) OR v_recibida = '0001-01-01T00:00:00Z'::timestamptz THEN
        RAISE EXCEPTION 'fecha de recepción vacía' USING ERRCODE = 'P0560';
    END IF;
    v_material_huella := encode(sha256(convert_to(p_material,'UTF8')),'hex');
    -- Mismo contexto canónico del recurso Go; referencias sin escapes JSON.
    v_contexto_huella := encode(sha256(convert_to(
        '{"ambitos":{"organizacion_ref":"' || (s ->> 'OrganizacionRef') ||
        '"},"atributos":{"material_sha256":"' || v_material_huella || '"}}','UTF8')),'hex');
    IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
        RAISE EXCEPTION 'decisión de respuesta inválida' USING ERRCODE = 'P0563';
    END IF;
    BEGIN
        d := convert_from(p_decision,'UTF8')::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'decisión de respuesta inválida' USING ERRCODE = 'P0563';
    END;
    IF d ->> 'accion' IS DISTINCT FROM 'contratacion_temporal.llamamiento.respuesta.registrar'
       OR d ->> 'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d ->> 'tipo_recurso' IS DISTINCT FROM 'respuesta_recibida_llamamiento_contratacion_temporal'
       OR d ->> 'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
       OR d ->> 'recurso_ref' IS DISTINCT FROM s ->> 'ExpedienteRef'
       OR d ->> 'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'autorización de respuesta divergente' USING ERRCODE = 'P0563';
    END IF;
    -- El mismo commit acredita autorización fresca y todos los efectos CT.
    -- Un error, también un conflicto, revierte el consumo nuevo.
    BEGIN
        SELECT * INTO STRICT v_consumo
          FROM vec_autorizacion_atestada_v3.registrar_y_consumir_respuesta_recibida_rrhh_v3_atestada(
            p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,
            p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    EXCEPTION
        WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN
            RAISE EXCEPTION 'registro de respuesta no disponible' USING ERRCODE = 'P0564';
        WHEN OTHERS THEN
            RAISE EXCEPTION 'consumo de respuesta denegado' USING ERRCODE = 'P0563';
    END;
    IF v_consumo.consumo_nuevo IS NOT TRUE
       OR v_consumo.efecto_ref IS DISTINCT FROM s ->> 'ExpedienteRef'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'respuesta requiere consumo nuevo ligado al material' USING ERRCODE = 'P0563';
    END IF;
    SELECT * INTO v_previa FROM vec_contratacion_temporal.respuesta_recibida_rrhh
     WHERE organizacion_ref = s ->> 'OrganizacionRef'
       AND clave_idempotencia = (s ->> 'ClaveIdempotencia')::uuid;
    IF FOUND THEN
        IF v_previa.actor_ref IS DISTINCT FROM d ->> 'principal_id'
           OR v_previa.perfil_ref IS DISTINCT FROM d ->> 'perfil_activo_ref' THEN
            RAISE EXCEPTION 'replay de respuesta denegado' USING ERRCODE = 'P0563';
        END IF;
        IF v_previa.material IS DISTINCT FROM p_material OR v_previa.material_json IS DISTINCT FROM s THEN
            RAISE EXCEPTION 'clave de respuesta divergente' USING ERRCODE = 'P0561';
        END IF;
        RETURN v_previa.recibo_json || jsonb_build_object('Estado','replay_registrada_por_rrhh');
    END IF;
    -- Solo antecedentes CT persistidos. La comunicación local es inmutable;
    -- su bloqueo ordena intentos sobre la misma comunicación, no la modifica.
    SELECT * INTO v_comunicacion FROM vec_contratacion_temporal.comunicacion_llamamiento_local
     WHERE comunicacion_ref = s ->> 'ComunicacionRef' FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'comunicación original no disponible' USING ERRCODE = 'P0562';
    END IF;
    IF v_comunicacion.organizacion_ref IS DISTINCT FROM s ->> 'OrganizacionRef'
       OR v_comunicacion.expediente_ref IS DISTINCT FROM s ->> 'ExpedienteRef'
       OR v_comunicacion.llamamiento_ref IS DISTINCT FROM s ->> 'LlamamientoRef'
       OR v_comunicacion.estado IS DISTINCT FROM 'registrada_localmente'
       OR v_comunicacion.version_resultante IS DISTINCT FROM 2::numeric THEN
        RAISE EXCEPTION 'comunicación original incompatible' USING ERRCODE = 'P0562';
    END IF;
    SELECT * INTO v_seleccion FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
     WHERE clave_idempotencia = v_comunicacion.seleccion_clave FOR SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'selección original no disponible' USING ERRCODE = 'P0562';
    END IF;
    IF v_seleccion.situacion IS DISTINCT FROM 'confirmada'
       OR v_seleccion.solicitud_json ->> 'organizacion_ref' IS DISTINCT FROM s ->> 'OrganizacionRef'
       OR v_seleccion.solicitud_json ->> 'expediente_ref' IS DISTINCT FROM s ->> 'ExpedienteRef'
       OR v_seleccion.recibo_json ->> 'organizacion_ref' IS DISTINCT FROM s ->> 'OrganizacionRef'
       OR v_seleccion.recibo_json ->> 'expediente_ref' IS DISTINCT FROM s ->> 'ExpedienteRef'
       OR v_seleccion.recibo_json ->> 'llamamiento_ref' IS DISTINCT FROM s ->> 'LlamamientoRef'
       OR v_seleccion.recibo_json -> 'propuesta_generada' IS DISTINCT FROM 'true'::jsonb
       OR v_seleccion.recibo_json ->> 'recibo_ref' IS DISTINCT FROM
           v_comunicacion.material_json -> 'solicitud' ->> 'PruebaEntregaRef'
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.respuesta_recibida_rrhh
           WHERE organizacion_ref = s ->> 'OrganizacionRef' AND comunicacion_ref = s ->> 'ComunicacionRef') THEN
        RAISE EXCEPTION 'antecedente o respuesta original en conflicto' USING ERRCODE = 'P0562';
    END IF;
    v_ahora := date_trunc('microseconds',clock_timestamp());
    IF v_recibida > v_ahora THEN
        RAISE EXCEPTION 'recepción posterior al registro' USING ERRCODE = 'P0560';
    END IF;
    v_justificante := 'justificante:' || gen_random_uuid()::text;
    v_recibo := 'recibo:' || gen_random_uuid()::text;
    v_evento := 'outbox:' || gen_random_uuid()::text;
    v_resultado := jsonb_build_object(
        'Solicitud',s,'JustificanteRef',v_justificante,'ReciboRef',v_recibo,
        'AuditoriaRef',v_consumo.auditoria_ref,
        'RegistradaEn',to_char(v_ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'Estado','registrada_por_rrhh');
    INSERT INTO vec_contratacion_temporal.respuesta_recibida_rrhh (
        justificante_ref,organizacion_ref,expediente_ref,llamamiento_ref,comunicacion_ref,
        seleccion_clave,clave_idempotencia,actor_ref,perfil_ref,version_comunicacion,
        respuesta,correo_ref,correo_sha256,recibida_en,material,material_json,
        material_huella_sha256,recibo_ref,recibo_json,estado,registrada_en
    ) VALUES (
        v_justificante,s ->> 'OrganizacionRef',s ->> 'ExpedienteRef',s ->> 'LlamamientoRef',
        s ->> 'ComunicacionRef',v_seleccion.clave_idempotencia,(s ->> 'ClaveIdempotencia')::uuid,
        d ->> 'principal_id',d ->> 'perfil_activo_ref',2,s ->> 'Respuesta',s ->> 'CorreoRef',
        s ->> 'CorreoSHA256',v_recibida,p_material,s,v_material_huella,v_recibo,v_resultado,
        'registrada_por_rrhh',v_ahora);
    INSERT INTO vec_contratacion_temporal.historia_respuesta_recibida_rrhh (
        auditoria_ref,justificante_ref,actor_ref,perfil_ref,decision_ref,
        consumo_huella_sha256,evidencia_huella_sha256,material_huella_sha256,registrada_en
    ) VALUES (
        v_consumo.auditoria_ref,v_justificante,d ->> 'principal_id',d ->> 'perfil_activo_ref',
        v_consumo.decision_ref,v_consumo.consumo_huella_sha256,
        encode(sha256(p_evidencia),'hex'),v_material_huella,v_ahora);
    INSERT INTO vec_contratacion_temporal.outbox_respuesta_recibida_rrhh (
        evento_ref,justificante_ref,estado,tipo,carga_json,creada_en
    ) VALUES (
        v_evento,v_justificante,'pendiente','llamamiento.respuesta.registrada_por_rrhh',
        jsonb_build_object('justificante_ref',v_justificante,'recibo_ref',v_recibo,
            'organizacion_ref',s ->> 'OrganizacionRef','expediente_ref',s ->> 'ExpedienteRef',
            'llamamiento_ref',s ->> 'LlamamientoRef','comunicacion_ref',s ->> 'ComunicacionRef',
            'respuesta_declarada',s ->> 'Respuesta','estado','registrada_por_rrhh'),v_ahora);
    RETURN v_resultado;
EXCEPTION
    WHEN unique_violation THEN
        RAISE EXCEPTION 'conflicto de registro de respuesta' USING ERRCODE = 'P0561';
    WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN
        RAISE EXCEPTION 'registro de respuesta no disponible' USING ERRCODE = 'P0564';
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_ejecutor;
COMMIT;
