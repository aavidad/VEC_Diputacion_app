\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

-- Comprobar autoridad antes del estado; solo ausencia de terminal reanudable.
DO $terminal$
DECLARE
    v_def text; v_acl aclitem[];
    v_anterior text := $anterior$    ELSIF v_ejecucion.situacion <> 'confirmada' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'terminal O6 no confirmado';
    ELSIF v_consulta->>'organizacion_ref' IS DISTINCT FROM v_ejecucion.solicitud_json->>'organizacion_ref'$anterior$;
    v_autoridad text := $autoridad$    ELSIF v_consulta->>'organizacion_ref' IS DISTINCT FROM v_ejecucion.solicitud_json->>'organizacion_ref'$autoridad$;
    v_marca text := $marca$        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'replay O6 denegado';
    ELSE
$marca$;
    v_extension text := $extension$        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'replay O6 denegado';
    ELSIF v_ejecucion.situacion = 'indeterminada'
       AND v_ejecucion.efecto = 'preparar_orden'
       AND v_ejecucion.ventana_orden_abierta IS TRUE
       AND v_ejecucion.ventana_llamamiento_abierta IS FALSE
       AND v_ejecucion.recibo_json IS NULL AND v_ejecucion.artefacto_canonico IS NULL THEN
        RETURN QUERY SELECT '', '', '', '', '', '';
    ELSIF v_ejecucion.situacion <> 'confirmada' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'terminal O6 no confirmado';
    ELSE
$extension$;
BEGIN
    SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl
      FROM pg_proc p WHERE p.oid = 'vec_contratacion_temporal.resolver_terminal_autorizado_seleccion_llamamiento_o6_v2(uuid,text)'::regprocedure
       AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
    IF length(v_def)-length(replace(v_def,v_anterior,'')) <> length(v_anterior)
       OR length(v_def)-length(replace(v_def,v_marca,'')) <> length(v_marca) THEN
        RAISE EXCEPTION 'terminal O6 incompatible con reanudación' USING ERRCODE = '55000';
    END IF;
    v_def := replace(v_def,v_anterior,v_autoridad);
    v_def := replace(v_def,v_marca,v_extension);
    EXECUTE v_def;
    IF (SELECT proacl FROM pg_proc WHERE oid = 'vec_contratacion_temporal.resolver_terminal_autorizado_seleccion_llamamiento_o6_v2(uuid,text)'::regprocedure) IS DISTINCT FROM v_acl THEN
        RAISE EXCEPTION 'cambio de terminal alteró permisos' USING ERRCODE = '55000';
    END IF;
END
$terminal$;

-- Solo reanuda la preparación de orden indeterminada. No lee Bolsa, no borra
-- intención ni ventanas y no declara confirmado ningún llamamiento.
CREATE TABLE vec_contratacion_temporal.historia_reanudacion_seleccion_llamamiento (
    auditoria_ref text PRIMARY KEY,
    clave_idempotencia uuid NOT NULL REFERENCES vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6,
    huella_semantica text NOT NULL CHECK (huella_semantica ~ '^[0-9a-f]{64}$'),
    fencing_anterior bigint NOT NULL CHECK (fencing_anterior BETWEEN 1 AND 9007199254740990),
    fencing_nuevo bigint NOT NULL CHECK (fencing_nuevo = fencing_anterior + 1),
    reserva_anterior_sha256 text NOT NULL CHECK (reserva_anterior_sha256 ~ '^[0-9a-f]{64}$'),
    reserva_nueva_sha256 text NOT NULL CHECK (reserva_nueva_sha256 ~ '^[0-9a-f]{64}$' AND reserva_nueva_sha256 <> reserva_anterior_sha256),
    decision_ref text NOT NULL UNIQUE,
    consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
    reanudada_en timestamptz(6) NOT NULL,
    lease_hasta timestamptz(6) NOT NULL CHECK (lease_hasta = reanudada_en + interval '30 seconds'),
    UNIQUE (clave_idempotencia, fencing_nuevo)
);
CREATE TABLE vec_contratacion_temporal.outbox_reanudacion_seleccion_llamamiento (
    evento_ref text PRIMARY KEY,
    auditoria_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.historia_reanudacion_seleccion_llamamiento,
    tipo text NOT NULL CHECK (tipo = 'seleccion.preparacion_orden.reanudada'),
    carga_json jsonb NOT NULL CHECK (jsonb_typeof(carga_json) = 'object'),
    creada_en timestamptz(6) NOT NULL
);
DO $seguridad$
DECLARE v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'historia_reanudacion_seleccion_llamamiento', 'outbox_reanudacion_seleccion_llamamiento'
    ] LOOP
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY', v_tabla);
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY', v_tabla);
        EXECUTE format('CREATE POLICY propietario ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true)', v_tabla);
        EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.%I FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()', v_tabla);
        EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM PUBLIC, vec_contratacion_temporal_ejecutor', v_tabla);
    END LOOP;
END
$seguridad$;

CREATE FUNCTION vec_contratacion_temporal.reanudar_preparacion_orden_seleccion_v1(
    p_solicitud_texto text,
    p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
    p_persona_version numeric, p_perfil_version numeric,
    p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea
) RETURNS TABLE (
    situacion text, solicitud_json text, reserva_ref text,
    efecto text, recibo_json text, artefacto_json text
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '15s'
SET idle_in_transaction_session_timeout = '20s'
AS $funcion$
DECLARE
    s jsonb; d jsonb; v_material text; v_material_huella text; v_contexto_huella text;
    v_ejecucion vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
    v_consumo record;
    v_ahora timestamptz(6); v_reserva text; v_evento text;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario' OR session_user = current_user
       OR NOT pg_has_role(session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_has_role(session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
       OR pg_has_role(session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
       OR current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION 'reanudación de selección denegada' USING ERRCODE = '42501';
    END IF;
    IF p_solicitud_texto IS NULL OR octet_length(p_solicitud_texto) NOT BETWEEN 1 AND 1048576
       OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
        RAISE EXCEPTION 'solicitud de reanudación inválida' USING ERRCODE = '22023';
    END IF;
    s := vec_contratacion_temporal.solicitud_desde_texto_seleccion_llamamiento_o6_v1(p_solicitud_texto);
    IF s IS NULL OR s->'version_expediente' IS DISTINCT FROM '6'::jsonb
       OR vec_contratacion_temporal.huella_solicitud_seleccion_llamamiento_o6_v1(s)
          IS DISTINCT FROM s->>'huella_semantica' THEN
        RAISE EXCEPTION 'solicitud de reanudación inválida' USING ERRCODE = '22023';
    END IF;
    -- Mismo canon de cinco campos que NuevoRecursoReanudacionSeleccionLlamamiento.
    v_material := '{"organizacion_ref":' || (s->'organizacion_ref')::text ||
        ',"expediente_ref":' || (s->'expediente_ref')::text ||
        ',"version_expediente":6,"clave_idempotencia":' || (s->'clave_idempotencia')::text ||
        ',"huella_semantica":' || (s->'huella_semantica')::text || '}';
    v_material_huella := encode(sha256(convert_to(v_material, 'UTF8')), 'hex');
    v_contexto_huella := encode(sha256(convert_to(
        '{"ambitos":{"organizacion_ref":' || (s->'organizacion_ref')::text ||
        '},"atributos":{"material_sha256":"' || v_material_huella || '"}}', 'UTF8')), 'hex');
    d := convert_from(p_decision, 'UTF8')::jsonb;
    IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.llamamiento.reanudar_orden'
       OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'reanudacion_seleccion_contratacion_temporal'
       OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
       OR d->>'recurso_ref' IS DISTINCT FROM s->>'expediente_ref'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'autorización de reanudación divergente' USING ERRCODE = '42501';
    END IF;
    -- El bloqueo y la comprobación del estado preceden al consumo fresco.
    SELECT e.* INTO v_ejecucion
      FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
     WHERE e.clave_idempotencia = (s->>'clave_idempotencia')::uuid FOR UPDATE;
    IF NOT FOUND OR v_ejecucion.solicitud_json IS DISTINCT FROM s
       OR v_ejecucion.huella_semantica IS DISTINCT FROM s->>'huella_semantica' THEN
        RAISE EXCEPTION 'intención de reanudación divergente' USING ERRCODE = '42501';
    END IF;
    v_ahora := date_trunc('microseconds', clock_timestamp());
    IF v_ejecucion.situacion IS DISTINCT FROM 'indeterminada'
       OR v_ejecucion.efecto IS DISTINCT FROM 'preparar_orden'
       OR v_ejecucion.ventana_orden_abierta IS NOT TRUE
       OR v_ejecucion.ventana_llamamiento_abierta IS NOT FALSE
       OR v_ejecucion.recibo_json IS NOT NULL OR v_ejecucion.artefacto_canonico IS NOT NULL
       OR v_ejecucion.lease_hasta > v_ahora
       OR v_ejecucion.fencing_version >= 9007199254740991 THEN
        RAISE EXCEPTION 'reanudación no disponible para este estado' USING ERRCODE = '55000';
    END IF;
    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3.registrar_y_consumir_reanudacion_seleccion_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.consumo_nuevo IS NOT TRUE
       OR v_consumo.efecto_ref IS DISTINCT FROM s->>'expediente_ref'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'reanudación requiere autorización nueva ligada' USING ERRCODE = '42501';
    END IF;
    v_ahora := date_trunc('microseconds', clock_timestamp());
    v_reserva := vec_contratacion_temporal.nuevo_token_fencing_seleccion_llamamiento_o6_v2();
    IF v_reserva IS NOT DISTINCT FROM v_ejecucion.reserva_ref THEN
        RAISE EXCEPTION 'reserva de reanudación no renovada' USING ERRCODE = '55000';
    END IF;
    -- No se alteran solicitud, UUID, huella, efecto, ventanas ni recibos.
    UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
       SET situacion = 'propietaria', reserva_ref = v_reserva,
           fencing_version = v_ejecucion.fencing_version + 1,
           lease_hasta = v_ahora + interval '30 seconds', actualizada_en = v_ahora
     WHERE e.clave_idempotencia = v_ejecucion.clave_idempotencia;
    INSERT INTO vec_contratacion_temporal.historia_reanudacion_seleccion_llamamiento VALUES (
        v_consumo.auditoria_ref, v_ejecucion.clave_idempotencia, v_ejecucion.huella_semantica,
        v_ejecucion.fencing_version, v_ejecucion.fencing_version + 1,
        encode(sha256(convert_to(v_ejecucion.reserva_ref,'UTF8')),'hex'),
        encode(sha256(convert_to(v_reserva,'UTF8')),'hex'),
        v_consumo.decision_ref, v_consumo.consumo_huella_sha256, v_ahora, v_ahora + interval '30 seconds');
    v_evento := 'evento:ct:reanudacion:' || gen_random_uuid()::text;
    INSERT INTO vec_contratacion_temporal.outbox_reanudacion_seleccion_llamamiento VALUES (
        v_evento, v_consumo.auditoria_ref, 'seleccion.preparacion_orden.reanudada',
        jsonb_build_object('organizacion_ref',s->>'organizacion_ref',
            'expediente_ref',s->>'expediente_ref','clave_idempotencia',v_ejecucion.clave_idempotencia,
            'huella_semantica',v_ejecucion.huella_semantica,
            'fencing_version',v_ejecucion.fencing_version + 1), v_ahora);
    RETURN QUERY SELECT 'propietaria', v_ejecucion.solicitud_json::text,
        v_reserva, 'preparar_orden', '', '';
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.reanudar_preparacion_orden_seleccion_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.reanudar_preparacion_orden_seleccion_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_ejecutor;
COMMIT;
