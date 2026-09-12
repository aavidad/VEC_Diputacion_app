\set ON_ERROR_STOP on
-- CT88 candidata: reserva durable del intento SMTP. No contiene correo,
-- asunto, cuerpo, destinatario, entrega, plazo ni efecto en Bolsa.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended(
    'vec_contratacion_temporal:000088_despacho_correo_llamamiento', 0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;

DO $pre$
DECLARE f oid;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR getdatabaseencoding() <> 'UTF8'
       OR to_regclass('vec_contratacion_temporal.comunicacion_llamamiento_local') IS NULL
       OR to_regclass('vec_contratacion_temporal.outbox_comunicacion_llamamiento_local') IS NULL
       OR to_regclass('vec_contratacion_temporal.despacho_correo_llamamiento_intento_v1') IS NOT NULL
       OR to_regclass('vec_contratacion_temporal.despacho_correo_llamamiento_resultado_v1') IS NOT NULL
       OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
                  WHERE n.nspname='vec_contratacion_temporal'
                    AND p.proname IN ('reservar_intento_despacho_correo_llamamiento_v1','registrar_resultado_intento_despacho_correo_llamamiento_v1')) THEN
        RAISE EXCEPTION 'CT88: antecedentes u objetos incompatibles' USING ERRCODE='55000';
    END IF;
    f := to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_despacho_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
    IF f IS NULL OR NOT has_function_privilege(current_user, f, 'EXECUTE') THEN
        RAISE EXCEPTION 'CT88: AD3-32 requerido' USING ERRCODE='55000';
    END IF;
    f := to_regprocedure('public.gen_random_bytes(integer)');
    IF f IS NULL OR NOT has_function_privilege(current_user, f, 'EXECUTE') THEN
        RAISE EXCEPTION 'CT88: primitiva aleatoria requerida' USING ERRCODE='55000';
    END IF;
END
$pre$;

CREATE TABLE vec_contratacion_temporal.despacho_correo_llamamiento_intento_v1 (
    intento_ref text PRIMARY KEY CHECK (intento_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    organizacion_ref text NOT NULL CHECK (organizacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    expediente_ref text NOT NULL CHECK (expediente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    llamamiento_ref text NOT NULL CHECK (llamamiento_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    comunicacion_ref text NOT NULL REFERENCES vec_contratacion_temporal.comunicacion_llamamiento_local,
    intencion_envio_ref text NOT NULL REFERENCES vec_contratacion_temporal.outbox_comunicacion_llamamiento_local(intencion_ref),
    solicitud_huella_sha256 text NOT NULL CHECK (solicitud_huella_sha256 ~ '^[0-9a-f]{64}$' AND solicitud_huella_sha256 <> repeat('0',64)),
    finalizacion_huella_sha256 text NOT NULL CHECK (finalizacion_huella_sha256 ~ '^[0-9a-f]{64}$' AND finalizacion_huella_sha256 <> repeat('0',64)),
    decision_ref text NOT NULL CHECK (decision_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$' AND consumo_huella_sha256 <> repeat('0',64)),
    auditoria_ref text NOT NULL UNIQUE CHECK (auditoria_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    message_id text NOT NULL UNIQUE CHECK (message_id ~ '^<[A-Za-z0-9._:+-]{3,160}@[A-Za-z0-9.-]{3,160}>$'),
    fecha_origen timestamptz(6) NOT NULL CHECK (isfinite(fecha_origen)),
    estado text NOT NULL CHECK (estado IN ('iniciado','no_aceptado_transitorio','no_aceptado_permanente','indeterminado','aceptado_por_relay')),
    plantilla_ref text,
    resultado_registrado_en timestamptz(6),
    CHECK ((estado = 'iniciado') = (resultado_registrado_en IS NULL)),
    CHECK ((estado = 'iniciado') = (plantilla_ref IS NULL)),
    CHECK (plantilla_ref IS NULL OR plantilla_ref = 'llamamiento_rrhh_v1'),
    UNIQUE (organizacion_ref, expediente_ref, llamamiento_ref, comunicacion_ref, intencion_envio_ref),
    UNIQUE (intento_ref, solicitud_huella_sha256)
);
COMMENT ON TABLE vec_contratacion_temporal.despacho_correo_llamamiento_intento_v1 IS
    'Reserva de intento de correo: no conserva dirección, asunto, cuerpo ni acredita entrega, plazo o efecto administrativo.';

CREATE TABLE vec_contratacion_temporal.despacho_correo_llamamiento_resultado_v1 (
    intento_ref text PRIMARY KEY REFERENCES vec_contratacion_temporal.despacho_correo_llamamiento_intento_v1,
    estado text NOT NULL CHECK (estado IN ('no_aceptado_transitorio','no_aceptado_permanente','indeterminado','aceptado_por_relay')),
    plantilla_ref text NOT NULL CHECK (plantilla_ref = 'llamamiento_rrhh_v1'),
    registrado_en timestamptz(6) NOT NULL CHECK (isfinite(registrado_en))
);

DO $seguridad$
DECLARE t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['despacho_correo_llamamiento_intento_v1','despacho_correo_llamamiento_resultado_v1'] LOOP
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY',t);
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY',t);
        EXECUTE format('CREATE POLICY propietario ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true)',t);
        EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM PUBLIC, vec_contratacion_temporal_ejecutor',t);
    END LOOP;
    CREATE TRIGGER resultado_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.despacho_correo_llamamiento_resultado_v1
        FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
END
$seguridad$;

CREATE FUNCTION vec_contratacion_temporal.reservar_intento_despacho_correo_llamamiento_v1(
    p_solicitud text,
    p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
    p_persona_version numeric, p_perfil_version numeric,
    p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE s jsonb; d jsonb; consumo record; previa record;
    h text; contexto_h text; ahora timestamptz(6); intento text; mensaje text; secreto bytea;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR session_user = current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
       OR current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION 'reserva de despacho denegada' USING ERRCODE='42501';
    END IF;
    IF p_solicitud IS NULL OR octet_length(p_solicitud) NOT BETWEEN 2 AND 16384 THEN
        RAISE EXCEPTION 'solicitud de despacho inválida' USING ERRCODE='22023';
    END IF;
    s := p_solicitud::jsonb;
    IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(s,ARRAY['OrganizacionRef','ExpedienteRef','LlamamientoRef','ComunicacionRef','IntencionEnvioRef']) IS NOT TRUE THEN
        RAISE EXCEPTION 'coordenadas de despacho inválidas' USING ERRCODE='22023';
    END IF;
    IF EXISTS (SELECT 1 FROM jsonb_each(s) x WHERE jsonb_typeof(x.value) <> 'string' OR x.value #>> '{}' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$') THEN
        RAISE EXCEPTION 'referencia de despacho inválida' USING ERRCODE='22023';
    END IF;
    h := encode(sha256(convert_to(p_solicitud,'UTF8')),'hex');
    contexto_h := encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||'"},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
    d := convert_from(p_decision,'UTF8')::jsonb;
    IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.llamamiento.correo.despachar'
       OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'despacho_correo_llamamiento_contratacion_temporal'
       OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
       OR d->>'recurso_ref' IS DISTINCT FROM s->>'IntencionEnvioRef'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h THEN
        RAISE EXCEPTION 'autorización de despacho divergente' USING ERRCODE='42501';
    END IF;
    SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_despacho_correo_llamamiento_ct_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF consumo.efecto_ref IS DISTINCT FROM s->>'IntencionEnvioRef'
       OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_h
       OR consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'consumo de despacho divergente' USING ERRCODE='42501';
    END IF;
    SELECT i.* INTO previa FROM vec_contratacion_temporal.despacho_correo_llamamiento_intento_v1 i
      JOIN vec_contratacion_temporal.comunicacion_llamamiento_local c ON c.comunicacion_ref=i.comunicacion_ref
      JOIN vec_contratacion_temporal.outbox_comunicacion_llamamiento_local o ON o.intencion_ref=i.intencion_envio_ref
     WHERE i.organizacion_ref=s->>'OrganizacionRef' AND i.expediente_ref=s->>'ExpedienteRef'
       AND i.llamamiento_ref=s->>'LlamamientoRef' AND i.comunicacion_ref=s->>'ComunicacionRef'
       AND i.intencion_envio_ref=s->>'IntencionEnvioRef'
       AND c.organizacion_ref=i.organizacion_ref AND c.expediente_ref=i.expediente_ref AND c.llamamiento_ref=i.llamamiento_ref
       AND o.comunicacion_ref=i.comunicacion_ref AND o.estado='pendiente'
     FOR UPDATE OF i;
    IF FOUND THEN
        IF previa.solicitud_huella_sha256 IS DISTINCT FROM h THEN RAISE EXCEPTION 'reserva de despacho divergente' USING ERRCODE='P0881'; END IF;
        RETURN jsonb_build_object('IntentoRef',previa.intento_ref,'MessageID',previa.message_id,'FechaOrigen',previa.fecha_origen,'SolicitudHuella',previa.solicitud_huella_sha256,'Estado',previa.estado,'YaReservado',true);
    END IF;
    PERFORM 1 FROM vec_contratacion_temporal.comunicacion_llamamiento_local c
      JOIN vec_contratacion_temporal.outbox_comunicacion_llamamiento_local o ON o.comunicacion_ref=c.comunicacion_ref
     WHERE c.comunicacion_ref=s->>'ComunicacionRef' AND c.organizacion_ref=s->>'OrganizacionRef'
       AND c.expediente_ref=s->>'ExpedienteRef' AND c.llamamiento_ref=s->>'LlamamientoRef'
       AND o.intencion_ref=s->>'IntencionEnvioRef' AND o.estado='pendiente'
     FOR KEY SHARE OF c,o;
    IF NOT FOUND THEN RAISE EXCEPTION 'intención de comunicación no disponible' USING ERRCODE='42501'; END IF;
    ahora := date_trunc('microseconds',clock_timestamp());
    intento := 'intento-correo:'||gen_random_uuid()::text;
    mensaje := '<'||replace(intento,':','.')||'@vec.local>';
    secreto := public.gen_random_bytes(32);
    INSERT INTO vec_contratacion_temporal.despacho_correo_llamamiento_intento_v1(
        intento_ref,organizacion_ref,expediente_ref,llamamiento_ref,comunicacion_ref,intencion_envio_ref,solicitud_huella_sha256,finalizacion_huella_sha256,decision_ref,consumo_huella_sha256,auditoria_ref,message_id,fecha_origen,estado)
    VALUES (intento,s->>'OrganizacionRef',s->>'ExpedienteRef',s->>'LlamamientoRef',s->>'ComunicacionRef',s->>'IntencionEnvioRef',h,
        encode(sha256(convert_to('vec.ct88.finalizacion.v1'||chr(10)||intento||chr(10)||h||chr(10),'UTF8')||secreto),'hex'),
        consumo.decision_ref,consumo.consumo_huella_sha256,consumo.auditoria_ref,mensaje,ahora,'iniciado');
    RETURN jsonb_build_object('IntentoRef',intento,'MessageID',mensaje,'FechaOrigen',ahora,'SolicitudHuella',h,'Estado','iniciado','YaReservado',false,'CapacidadFinalizacion',encode(secreto,'hex'));
EXCEPTION WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN
    RAISE EXCEPTION 'reserva de despacho no disponible' USING ERRCODE='P0882';
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.registrar_resultado_intento_despacho_correo_llamamiento_v1(
    p_intento_ref text, p_solicitud_huella_sha256 text, p_capacidad_finalizacion bytea,
    p_estado text, p_plantilla_ref text
) RETURNS void
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE i vec_contratacion_temporal.despacho_correo_llamamiento_intento_v1%ROWTYPE; ahora timestamptz(6);
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario' OR session_user = current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
       OR current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off' THEN RAISE EXCEPTION 'resultado de despacho denegado' USING ERRCODE='42501'; END IF;
    IF p_capacidad_finalizacion IS NULL OR octet_length(p_capacidad_finalizacion) <> 32
       OR p_capacidad_finalizacion = decode(repeat('00',32),'hex') THEN
        RAISE EXCEPTION 'resultado de despacho denegado' USING ERRCODE='42501';
    END IF;
    IF p_intento_ref IS NULL OR p_intento_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_solicitud_huella_sha256 IS NULL OR p_solicitud_huella_sha256 !~ '^[0-9a-f]{64}$' OR p_solicitud_huella_sha256=repeat('0',64)
       OR p_estado IS NULL OR p_estado NOT IN ('no_aceptado_transitorio','no_aceptado_permanente','indeterminado','aceptado_por_relay')
       OR p_plantilla_ref IS DISTINCT FROM 'llamamiento_rrhh_v1' THEN RAISE EXCEPTION 'resultado de despacho inválido' USING ERRCODE='22023'; END IF;
    SELECT * INTO i FROM vec_contratacion_temporal.despacho_correo_llamamiento_intento_v1 WHERE intento_ref=p_intento_ref FOR UPDATE;
    IF NOT FOUND
       OR i.solicitud_huella_sha256 IS DISTINCT FROM p_solicitud_huella_sha256
       OR i.finalizacion_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to('vec.ct88.finalizacion.v1'||chr(10)||p_intento_ref||chr(10)||p_solicitud_huella_sha256||chr(10),'UTF8')||p_capacidad_finalizacion),'hex') THEN
        RAISE EXCEPTION 'resultado de despacho denegado' USING ERRCODE='42501';
    END IF;
    IF i.estado <> 'iniciado' THEN
        IF i.estado IS DISTINCT FROM p_estado OR i.plantilla_ref IS DISTINCT FROM p_plantilla_ref THEN RAISE EXCEPTION 'resultado de despacho inmutable' USING ERRCODE='P0883'; END IF;
        RETURN;
    END IF;
    ahora:=date_trunc('microseconds',clock_timestamp());
    UPDATE vec_contratacion_temporal.despacho_correo_llamamiento_intento_v1 SET estado=p_estado,plantilla_ref=p_plantilla_ref,resultado_registrado_en=ahora WHERE intento_ref=p_intento_ref;
    INSERT INTO vec_contratacion_temporal.despacho_correo_llamamiento_resultado_v1 VALUES(p_intento_ref,p_estado,p_plantilla_ref,ahora);
END
$funcion$;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.reservar_intento_despacho_correo_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC, vec_contratacion_temporal_ejecutor;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_resultado_intento_despacho_correo_llamamiento_v1(text,text,bytea,text,text) FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.reservar_intento_despacho_correo_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_resultado_intento_despacho_correo_llamamiento_v1(text,text,bytea,text,text) TO vec_contratacion_temporal_ejecutor;
COMMIT;
