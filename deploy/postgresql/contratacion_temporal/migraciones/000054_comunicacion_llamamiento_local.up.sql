\set ON_ERROR_STOP on
-- Registro local de comunicación: no afirma entrega ni abre plazo legal.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:000054_comunicacion_llamamiento', 0));

CREATE TABLE vec_contratacion_temporal.comunicacion_llamamiento_local (
    comunicacion_ref text PRIMARY KEY,
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    llamamiento_ref text NOT NULL,
    clave_idempotencia uuid NOT NULL,
    seleccion_clave uuid NOT NULL REFERENCES vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    -- Versión local inicial del llamamiento; no es la versión del expediente.
    version_anterior numeric(20,0) NOT NULL CHECK (version_anterior = 1),
    version_resultante numeric(20,0) NOT NULL CHECK (version_resultante = version_anterior + 1),
    material_json jsonb NOT NULL CHECK (jsonb_typeof(material_json) = 'object'),
    recibo_json jsonb NOT NULL CHECK (jsonb_typeof(recibo_json) = 'object'),
    estado text NOT NULL CHECK (estado = 'registrada_localmente'),
    registrada_en timestamptz(6) NOT NULL,
    UNIQUE (organizacion_ref, clave_idempotencia),
    UNIQUE (organizacion_ref, expediente_ref, llamamiento_ref)
);
CREATE TABLE vec_contratacion_temporal.historia_comunicacion_llamamiento_local (
    auditoria_ref text PRIMARY KEY,
    comunicacion_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.comunicacion_llamamiento_local,
    decision_ref text NOT NULL UNIQUE,
    consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
    material_huella_sha256 text NOT NULL CHECK (material_huella_sha256 ~ '^[0-9a-f]{64}$'),
    registrada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_contratacion_temporal.outbox_comunicacion_llamamiento_local (
    intencion_ref text PRIMARY KEY,
    comunicacion_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.comunicacion_llamamiento_local,
    estado text NOT NULL CHECK (estado = 'pendiente'),
    tipo text NOT NULL CHECK (tipo = 'comunicacion.registrada_localmente'),
    carga_json jsonb NOT NULL CHECK (jsonb_typeof(carga_json) = 'object'),
    creada_en timestamptz(6) NOT NULL
);
COMMENT ON TABLE vec_contratacion_temporal.outbox_comunicacion_llamamiento_local IS
    'Intención local persistida; pendiente no acredita fichero generado, envío, entrega ni recepción.';

DO $seguridad$
DECLARE v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'comunicacion_llamamiento_local',
        'historia_comunicacion_llamamiento_local',
        'outbox_comunicacion_llamamiento_local'
    ] LOOP
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY', v_tabla);
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY', v_tabla);
        EXECUTE format('CREATE POLICY propietario ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true)', v_tabla);
        EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.%I FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()', v_tabla);
        EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM PUBLIC, vec_contratacion_temporal_ejecutor', v_tabla);
    END LOOP;
END
$seguridad$;

CREATE FUNCTION vec_contratacion_temporal.registrar_comunicacion_llamamiento_local_v1(
    p_material text,
    p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
    p_persona_version numeric, p_perfil_version numeric,
    p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    m jsonb; s jsonb; d jsonb; v_referencia text; v_ref jsonb;
    v_material_huella text; v_contexto_huella text;
    v_consumo record;
    v_seleccion vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
    v_previa vec_contratacion_temporal.comunicacion_llamamiento_local%ROWTYPE;
    v_comunicacion text; v_recibo text; v_intencion text;
    v_ahora timestamptz(6); v_resultado jsonb;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR session_user = current_user
       OR NOT pg_has_role(session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_has_role(session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
       OR pg_has_role(session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
       OR current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION 'comunicación denegada' USING ERRCODE = '42501';
    END IF;
    IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 16384
       OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
        RAISE EXCEPTION 'material de comunicación inválido' USING ERRCODE = '22023';
    END IF;
    m := p_material::jsonb;
    s := m -> 'solicitud';
    IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,
           ARRAY['solicitud','canal','politica']) IS NOT TRUE
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(s,
           ARRAY['ClaveIdempotencia','OrganizacionRef','ExpedienteRef','LlamamientoRef','VersionEsperada','PruebaEntregaRef']) IS NOT TRUE
       OR jsonb_typeof(s -> 'VersionEsperada') IS DISTINCT FROM 'number'
       OR (s ->> 'VersionEsperada') !~ '^[1-9][0-9]{0,15}$'
       OR (s ->> 'VersionEsperada')::numeric > 9007199254740990
       OR jsonb_typeof(s -> 'ClaveIdempotencia') IS DISTINCT FROM 'string'
       OR (s ->> 'ClaveIdempotencia') !~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
       OR s ->> 'ClaveIdempotencia' = '00000000-0000-4000-8000-000000000000' THEN
        RAISE EXCEPTION 'solicitud de comunicación inválida' USING ERRCODE = '22023';
    END IF;
    FOREACH v_referencia IN ARRAY ARRAY['OrganizacionRef','ExpedienteRef','LlamamientoRef','PruebaEntregaRef'] LOOP
        IF jsonb_typeof(s -> v_referencia) IS DISTINCT FROM 'string'
           OR (s ->> v_referencia) !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
            RAISE EXCEPTION 'referencia de comunicación inválida' USING ERRCODE = '22023';
        END IF;
    END LOOP;
    FOREACH v_ref IN ARRAY ARRAY[m -> 'canal', m -> 'politica'] LOOP
        IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(v_ref,
               ARRAY['Referencia','Version','HuellaSHA256']) IS NOT TRUE
           OR jsonb_typeof(v_ref -> 'Referencia') IS DISTINCT FROM 'string'
           OR (v_ref ->> 'Referencia') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR jsonb_typeof(v_ref -> 'Version') IS DISTINCT FROM 'number'
           OR (v_ref ->> 'Version') !~ '^[1-9][0-9]{0,15}$'
           OR (v_ref ->> 'Version')::numeric > 9007199254740991
           OR jsonb_typeof(v_ref -> 'HuellaSHA256') IS DISTINCT FROM 'string'
           OR (v_ref ->> 'HuellaSHA256') !~ '^[0-9a-f]{64}$'
           OR v_ref ->> 'HuellaSHA256' = repeat('0', 64) THEN
            RAISE EXCEPTION 'política o canal inválidos' USING ERRCODE = '22023';
        END IF;
    END LOOP;
    v_material_huella := encode(sha256(convert_to(p_material, 'UTF8')), 'hex');
    -- Las referencias validadas no contienen caracteres que requieran escape.
    -- Mismo contexto JSON canónico que RecursoRegistroComunicacionLlamamiento.
    v_contexto_huella := encode(sha256(convert_to(
        '{"ambitos":{"organizacion_ref":"' || (s ->> 'OrganizacionRef') ||
        '"},"atributos":{"material_sha256":"' || v_material_huella || '"}}', 'UTF8')), 'hex');
    d := convert_from(p_decision, 'UTF8')::jsonb;
    IF d ->> 'accion' IS DISTINCT FROM 'contratacion_temporal.llamamiento.comunicacion.registrar'
       OR d ->> 'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d ->> 'tipo_recurso' IS DISTINCT FROM 'comunicacion_llamamiento_contratacion_temporal'
       OR d ->> 'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
       OR d ->> 'recurso_ref' IS DISTINCT FROM s ->> 'ExpedienteRef'
       OR d ->> 'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'autorización de comunicación divergente' USING ERRCODE = '42501';
    END IF;

    -- Verificar y consumir V3 ANTES de cualquier resultado, también replay.
    -- Toda excepción posterior revierte este consumo junto al efecto local.
    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3.registrar_y_consumir_comunicacion_llamamiento_v3_atestada(
        p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version,
        p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
    IF v_consumo.efecto_ref IS DISTINCT FROM s ->> 'ExpedienteRef'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'consumo de comunicación divergente' USING ERRCODE = '42501';
    END IF;

    -- El núcleo puede reconocer un consumo histórico sin revalidar su
    -- vigencia actual. Ni siquiera un replay local admite esa capacidad:
    -- la misma intención necesita una decisión/capacidad V3 nueva.
    IF v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'comunicación requiere autorización nueva' USING ERRCODE = '42501';
    END IF;
    SELECT * INTO v_previa FROM vec_contratacion_temporal.comunicacion_llamamiento_local
      WHERE organizacion_ref = s ->> 'OrganizacionRef'
        AND clave_idempotencia = (s ->> 'ClaveIdempotencia')::uuid;
    IF FOUND THEN
        IF v_previa.actor_ref IS DISTINCT FROM d ->> 'principal_id'
           OR v_previa.perfil_ref IS DISTINCT FROM d ->> 'perfil_activo_ref' THEN
            RAISE EXCEPTION 'replay de comunicación denegado' USING ERRCODE = '42501';
        END IF;
        IF v_previa.material_json IS DISTINCT FROM m THEN
            RAISE EXCEPTION 'clave de comunicación divergente' USING ERRCODE = 'P0541';
        END IF;
        RETURN v_previa.recibo_json || jsonb_build_object('Estado', 'replay_registrada_localmente');
    END IF;
    -- Propiedad CT: se usa exclusivamente su recibo de selección confirmado.
    -- No se lee un agregado ni una tabla de Bolsa. El bloqueo serializa dos
    -- intentos sobre el mismo llamamiento sin inventar una selección.
    BEGIN
        SELECT e.* INTO STRICT v_seleccion
          FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
         WHERE e.situacion = 'confirmada'
           AND e.solicitud_json ->> 'organizacion_ref' = s ->> 'OrganizacionRef'
           AND e.solicitud_json ->> 'expediente_ref' = s ->> 'ExpedienteRef'
           AND e.recibo_json ->> 'llamamiento_ref' = s ->> 'LlamamientoRef'
           AND e.recibo_json -> 'propuesta_generada' = 'true'::jsonb
         FOR UPDATE;
    EXCEPTION WHEN no_data_found OR too_many_rows THEN
        RAISE EXCEPTION 'llamamiento confirmado no disponible' USING ERRCODE = '42501';
    END;
    -- El nombre PruebaEntregaRef es heredado del puerto probatorio. En modo
    -- local solo admite el recibo real de selección como antecedente.
    IF v_seleccion.recibo_json ->> 'recibo_ref' IS DISTINCT FROM s ->> 'PruebaEntregaRef' THEN
        RAISE EXCEPTION 'antecedente de comunicación no acreditado' USING ERRCODE = '42501';
    END IF;
    IF v_seleccion.recibo_json ->> 'organizacion_ref' IS DISTINCT FROM s ->> 'OrganizacionRef'
       OR v_seleccion.recibo_json ->> 'expediente_ref' IS DISTINCT FROM s ->> 'ExpedienteRef'
       OR s -> 'VersionEsperada' IS DISTINCT FROM '1'::jsonb
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.comunicacion_llamamiento_local
          WHERE organizacion_ref = s ->> 'OrganizacionRef'
            AND expediente_ref = s ->> 'ExpedienteRef'
            AND llamamiento_ref = s ->> 'LlamamientoRef') THEN
        RAISE EXCEPTION 'versión de comunicación en conflicto' USING ERRCODE = 'P0542';
    END IF;

    v_ahora := date_trunc('microseconds', clock_timestamp());
    v_comunicacion := 'comunicacion:' || gen_random_uuid()::text;
    v_recibo := 'recibo:' || gen_random_uuid()::text;
    v_intencion := 'outbox:' || gen_random_uuid()::text;
    v_resultado := jsonb_build_object(
        'Solicitud', s, 'ComunicacionRef', v_comunicacion,
        'Canal', m -> 'canal', 'Politica', m -> 'politica',
        'ReciboRef', v_recibo, 'AuditoriaRef', v_consumo.auditoria_ref,
        'VersionResultante', (s ->> 'VersionEsperada')::numeric + 1,
        'Estado', 'registrada_localmente', 'RegistradaEn', v_ahora,
        'IntencionEnvioRef', v_intencion);
    INSERT INTO vec_contratacion_temporal.comunicacion_llamamiento_local VALUES (
        v_comunicacion, s ->> 'OrganizacionRef', s ->> 'ExpedienteRef',
        s ->> 'LlamamientoRef', (s ->> 'ClaveIdempotencia')::uuid,
        v_seleccion.clave_idempotencia, d ->> 'principal_id', d ->> 'perfil_activo_ref',
        (s ->> 'VersionEsperada')::numeric, (s ->> 'VersionEsperada')::numeric + 1,
        m, v_resultado, 'registrada_localmente', v_ahora);
    INSERT INTO vec_contratacion_temporal.historia_comunicacion_llamamiento_local VALUES (
        v_consumo.auditoria_ref, v_comunicacion, v_consumo.decision_ref,
        v_consumo.consumo_huella_sha256, v_material_huella, v_ahora);
    INSERT INTO vec_contratacion_temporal.outbox_comunicacion_llamamiento_local VALUES (
        v_intencion, v_comunicacion, 'pendiente', 'comunicacion.registrada_localmente',
        jsonb_build_object('comunicacion_ref', v_comunicacion,
            'expediente_ref', s ->> 'ExpedienteRef',
            'llamamiento_ref', s ->> 'LlamamientoRef',
            'recibo_seleccion_ref', s ->> 'PruebaEntregaRef',
            'canal', m -> 'canal', 'politica', m -> 'politica'), v_ahora);
    RETURN v_resultado;
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_comunicacion_llamamiento_local_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_comunicacion_llamamiento_local_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_ejecutor;
COMMIT;
