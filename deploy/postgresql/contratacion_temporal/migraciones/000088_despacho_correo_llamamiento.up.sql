\set ON_ERROR_STOP on
-- CT88 candidata: reserva durable del intento SMTP. No contiene correo,
-- asunto, cuerpo, destinatario, entrega, plazo ni efecto en Bolsa.
-- Instalación candidata: AD3-32 → T13/5 sobre T13/1 → CT88. Resultado,
-- consumo V3, auditoría T13 y outbox comparten TX. Exportación global pendiente.
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
       OR to_regclass('vec_contratacion_temporal.despacho_correo_llamamiento_outbox_v1') IS NOT NULL
       OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
                  WHERE n.nspname='vec_contratacion_temporal'
                    AND p.proname IN ('reservar_intento_despacho_correo_llamamiento_v1','registrar_resultado_intento_despacho_correo_llamamiento_v1','correo88_validar_resultado_v1')) THEN
        RAISE EXCEPTION 'CT88: antecedentes u objetos incompatibles' USING ERRCODE='55000';
    END IF;
    f := to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_despacho_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
    IF f IS NULL OR NOT has_function_privilege(current_user, f, 'EXECUTE') THEN
        RAISE EXCEPTION 'CT88: AD3-32 requerido' USING ERRCODE='55000';
    END IF;
    f := to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_resultado_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
    IF f IS NULL OR NOT has_function_privilege(current_user, f, 'EXECUTE') THEN
        RAISE EXCEPTION 'CT88: AD3-32 resultado requerido' USING ERRCODE='55000';
    END IF;
    f := to_regprocedure('vec_contratacion_temporal.incorporacion75_ventana(jsonb,jsonb,timestamp with time zone)');
    IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE')
       OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f
            AND proowner=current_user::regrole AND provolatile='i' AND NOT prosecdef) THEN
        RAISE EXCEPTION 'CT88: validador temporal existente requerido' USING ERRCODE='55000';
    END IF;
    f := to_regprocedure('vec_bolsa_registro_accesos.registrar_resultado_correo_llamamiento_ct_v1(bytea,text,bytea,bytea,text,text,text,boolean,text)');
    IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE')
       OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f
           AND proowner='vec_bolsa_accesos_propietario'::regrole AND prosecdef AND provolatile='v'
           AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s','row_security=on'])
       OR NOT COALESCE((SELECT count(*)=2 AND count(DISTINCT a.grantee)=2
           AND bool_and(a.grantee IN ('vec_bolsa_accesos_propietario'::regrole,current_user::regrole)
                        AND a.grantor='vec_bolsa_accesos_propietario'::regrole
                        AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
           FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
           WHERE p.oid=f),false) THEN
        RAISE EXCEPTION 'CT88: autoridad nominal T13/5 requerida' USING ERRCODE='55000';
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
    resultado_huella_sha256 text NOT NULL CHECK (resultado_huella_sha256 ~ '^[0-9a-f]{64}$' AND resultado_huella_sha256 <> repeat('0',64)),
    decision_ref text NOT NULL UNIQUE CHECK (decision_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$' AND consumo_huella_sha256 <> repeat('0',64)),
    auditoria_consumo_ref text NOT NULL UNIQUE CHECK (auditoria_consumo_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    auditoria_ref text NOT NULL UNIQUE CHECK (auditoria_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    estado text NOT NULL CHECK (estado IN ('no_aceptado_transitorio','no_aceptado_permanente','indeterminado','aceptado_por_relay')),
    plantilla_ref text NOT NULL CHECK (plantilla_ref = 'llamamiento_rrhh_v1'),
    version_resultante integer NOT NULL CHECK (version_resultante = 2),
    registrado_en timestamptz(6) NOT NULL CHECK (isfinite(registrado_en)),
    UNIQUE (intento_ref,auditoria_ref,resultado_huella_sha256,estado,plantilla_ref,version_resultante,registrado_en)
);
COMMENT ON TABLE vec_contratacion_temporal.despacho_correo_llamamiento_resultado_v1 IS
    'Hecho de resultado observado: enlaza auditoría de consumo y autoridad común; no sustituye auditoría ni acredita entrega.';

CREATE TABLE vec_contratacion_temporal.despacho_correo_llamamiento_outbox_v1 (
    evento_ref text PRIMARY KEY CHECK (evento_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    intento_ref text NOT NULL UNIQUE,
    auditoria_ref text NOT NULL UNIQUE,
    resultado_huella_sha256 text NOT NULL,
    estado_resultado text NOT NULL,
    plantilla_ref text NOT NULL,
    version_resultante integer NOT NULL,
    creada_en timestamptz(6) NOT NULL,
    tipo text NOT NULL CHECK (tipo = 'contratacion_temporal.llamamiento.correo.resultado_registrado'),
    estado text NOT NULL CHECK (estado = 'pendiente'),
    FOREIGN KEY (intento_ref,auditoria_ref,resultado_huella_sha256,estado_resultado,plantilla_ref,version_resultante,creada_en)
        REFERENCES vec_contratacion_temporal.despacho_correo_llamamiento_resultado_v1
            (intento_ref,auditoria_ref,resultado_huella_sha256,estado,plantilla_ref,version_resultante,registrado_en)
);
COMMENT ON TABLE vec_contratacion_temporal.despacho_correo_llamamiento_outbox_v1 IS
    'Evento minimizado 1:1 del resultado, pendiente de publicación; no acredita exportación, entrega ni plazo.';

DO $seguridad$
DECLARE t text; a record;
BEGIN
    FOREACH t IN ARRAY ARRAY['despacho_correo_llamamiento_intento_v1','despacho_correo_llamamiento_resultado_v1','despacho_correo_llamamiento_outbox_v1'] LOOP
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY',t);
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY',t);
        EXECUTE format('CREATE POLICY propietario ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true)',t);
        EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM PUBLIC',t);
        FOR a IN SELECT DISTINCT x.grantee FROM pg_class c,
            LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
            WHERE c.oid=to_regclass('vec_contratacion_temporal.'||t)
              AND x.grantee<>0 AND x.grantee<>c.relowner LOOP
            EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM %I',t,pg_get_userbyid(a.grantee));
        END LOOP;
        EXECUTE format('REVOKE ALL ON TYPE vec_contratacion_temporal.%I FROM PUBLIC',t);
        FOR a IN SELECT DISTINCT x.grantee FROM pg_type ty,
            LATERAL aclexplode(coalesce(ty.typacl,acldefault('T',ty.typowner))) x
            WHERE ty.oid=(SELECT c.reltype FROM pg_class c WHERE c.oid=to_regclass('vec_contratacion_temporal.'||t))
              AND x.grantee<>0 AND x.grantee<>ty.typowner LOOP
            EXECUTE format('REVOKE ALL ON TYPE vec_contratacion_temporal.%I FROM %I',t,pg_get_userbyid(a.grantee));
        END LOOP;
        IF t <> 'despacho_correo_llamamiento_intento_v1' THEN
            EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_contratacion_temporal.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()',t);
        END IF;
    END LOOP;
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

-- Canonicalización compartida con el DTO de dominio: orden fijo, sin espacios,
-- escapes alternativos, claves duplicadas ni campos ajenos. No lee estado.
CREATE FUNCTION vec_contratacion_temporal.correo88_validar_resultado_v1(p_solicitud text)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
DECLARE s jsonb; canonico text;
BEGIN
    IF p_solicitud IS NULL OR octet_length(p_solicitud) NOT BETWEEN 2 AND 16384 THEN
        RAISE EXCEPTION 'solicitud de resultado inválida' USING ERRCODE='22023';
    END IF;
    BEGIN
        s := p_solicitud::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'solicitud de resultado inválida' USING ERRCODE='22023';
    END;
    IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(s,ARRAY[
        'OrganizacionRef','ExpedienteRef','LlamamientoRef','ComunicacionRef',
        'IntencionEnvioRef','IntentoRef','SolicitudHuella','Estado','PlantillaRef',
        'VersionEsperada']) IS NOT TRUE
       OR EXISTS (SELECT 1 FROM jsonb_each(s) x WHERE x.key <> 'VersionEsperada'
                  AND jsonb_typeof(x.value) IS DISTINCT FROM 'string')
       OR EXISTS (SELECT 1 FROM jsonb_each(s) x WHERE x.key IN (
                      'OrganizacionRef','ExpedienteRef','LlamamientoRef',
                      'ComunicacionRef','IntencionEnvioRef','IntentoRef')
                  AND x.value #>> '{}' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR s->>'SolicitudHuella' !~ '^[0-9a-f]{64}$'
       OR s->>'SolicitudHuella' = repeat('0',64)
       OR s->>'Estado' NOT IN ('no_aceptado_transitorio','no_aceptado_permanente',
                               'indeterminado','aceptado_por_relay')
       OR s->>'PlantillaRef' IS DISTINCT FROM 'llamamiento_rrhh_v1'
       OR s->'VersionEsperada' IS DISTINCT FROM '1'::jsonb THEN
        RAISE EXCEPTION 'material de resultado inválido' USING ERRCODE='22023';
    END IF;
    canonico := '{"OrganizacionRef":"'||(s->>'OrganizacionRef')||
        '","ExpedienteRef":"'||(s->>'ExpedienteRef')||
        '","LlamamientoRef":"'||(s->>'LlamamientoRef')||
        '","ComunicacionRef":"'||(s->>'ComunicacionRef')||
        '","IntencionEnvioRef":"'||(s->>'IntencionEnvioRef')||
        '","IntentoRef":"'||(s->>'IntentoRef')||
        '","SolicitudHuella":"'||(s->>'SolicitudHuella')||
        '","Estado":"'||(s->>'Estado')||
        '","PlantillaRef":"'||(s->>'PlantillaRef')||'","VersionEsperada":1}';
    IF p_solicitud IS DISTINCT FROM canonico THEN
        RAISE EXCEPTION 'material de resultado no canónico' USING ERRCODE='22023';
    END IF;
    RETURN s;
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.correo88_validar_resultado_v1(text)
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;

-- El replay de este finalizador sólo pertenece al poseedor original del secreto.
-- Reservar en replay no devuelve secreto y no vuelve a llamar SMTP ni finalización.
CREATE FUNCTION vec_contratacion_temporal.registrar_resultado_intento_despacho_correo_llamamiento_v1(
    p_solicitud text, p_capacidad_finalizacion bytea, p_auditoria bytea,
    p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
    p_persona_version numeric, p_perfil_version numeric,
    p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    s jsonb; d jsonb; c jsonb; recibo_auditoria jsonb; h text; ha text; contexto_h text; consumo record;
    i vec_contratacion_temporal.despacho_correo_llamamiento_intento_v1%ROWTYPE;
    r vec_contratacion_temporal.despacho_correo_llamamiento_resultado_v1%ROWTYPE;
    o vec_contratacion_temporal.despacho_correo_llamamiento_outbox_v1%ROWTYPE;
    ahora timestamptz(6); auditoria text; evento text; filas bigint;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario' OR session_user = current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
       OR current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION 'resultado de despacho denegado' USING ERRCODE='42501';
    END IF;
    IF p_capacidad_finalizacion IS NULL OR octet_length(p_capacidad_finalizacion) <> 32
       OR p_capacidad_finalizacion = decode(repeat('00',32),'hex') THEN
        RAISE EXCEPTION 'resultado de despacho denegado' USING ERRCODE='42501';
    END IF;
    IF p_auditoria IS NULL OR octet_length(p_auditoria) NOT BETWEEN 2 AND 16384 THEN
        RAISE EXCEPTION 'auditoría de resultado requerida' USING ERRCODE='22023';
    END IF;
    ha := encode(sha256(p_auditoria),'hex');
    s := vec_contratacion_temporal.correo88_validar_resultado_v1(p_solicitud);
    h := encode(sha256(convert_to(p_solicitud,'UTF8')),'hex');
    contexto_h := encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||'"},"atributos":{"auditoria_sha256":"'||ha||'","material_sha256":"'||h||'"}}','UTF8')),'hex');
    IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
        RAISE EXCEPTION 'autorización de resultado inválida' USING ERRCODE='42501';
    END IF;
    d := convert_from(p_decision,'UTF8')::jsonb;
    IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.llamamiento.correo.registrar_resultado'
       OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'resultado_correo_llamamiento_contratacion_temporal'
       OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
       OR d->>'recurso_ref' IS DISTINCT FROM s->>'IntentoRef'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h THEN
        RAISE EXCEPTION 'autorización de resultado divergente' USING ERRCODE='42501';
    END IF;
    -- Mismo orden de locks que reserva: autoridad central antes del intento.
    -- Cualquier rechazo posterior revierte también consumo y auditoría de acceso.
    SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_resultado_correo_llamamiento_ct_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF consumo.efecto_ref IS DISTINCT FROM s->>'IntentoRef'
       OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_h
       OR consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'consumo de resultado divergente' USING ERRCODE='42501';
    END IF;
    c := convert_from(p_capacidad,'UTF8')::jsonb;
    SELECT * INTO i FROM vec_contratacion_temporal.despacho_correo_llamamiento_intento_v1
     WHERE intento_ref=s->>'IntentoRef' FOR UPDATE;
    IF NOT FOUND
       OR i.organizacion_ref IS DISTINCT FROM s->>'OrganizacionRef'
       OR i.expediente_ref IS DISTINCT FROM s->>'ExpedienteRef'
       OR i.llamamiento_ref IS DISTINCT FROM s->>'LlamamientoRef'
       OR i.comunicacion_ref IS DISTINCT FROM s->>'ComunicacionRef'
       OR i.intencion_envio_ref IS DISTINCT FROM s->>'IntencionEnvioRef'
       OR i.solicitud_huella_sha256 IS DISTINCT FROM s->>'SolicitudHuella'
       OR i.finalizacion_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(
           'vec.ct88.finalizacion.v1'||chr(10)||(s->>'IntentoRef')||chr(10)||
           (s->>'SolicitudHuella')||chr(10),'UTF8')||p_capacidad_finalizacion),'hex') THEN
        RAISE EXCEPTION 'resultado de despacho denegado' USING ERRCODE='42501';
    END IF;
    IF i.estado <> 'iniciado' THEN
        SELECT * INTO r FROM vec_contratacion_temporal.despacho_correo_llamamiento_resultado_v1
         WHERE intento_ref=i.intento_ref;
        IF NOT FOUND OR r.resultado_huella_sha256 IS DISTINCT FROM h
           OR r.estado IS DISTINCT FROM i.estado OR r.estado IS DISTINCT FROM s->>'Estado'
           OR r.plantilla_ref IS DISTINCT FROM i.plantilla_ref
           OR r.plantilla_ref IS DISTINCT FROM s->>'PlantillaRef'
           OR r.registrado_en IS DISTINCT FROM i.resultado_registrado_en
           OR r.version_resultante IS DISTINCT FROM 2 THEN
            RAISE EXCEPTION 'resultado de despacho inmutable o incompleto' USING ERRCODE='P0883';
        END IF;
        SELECT * INTO o FROM vec_contratacion_temporal.despacho_correo_llamamiento_outbox_v1
         WHERE intento_ref=i.intento_ref;
        IF NOT FOUND OR o.auditoria_ref IS DISTINCT FROM r.auditoria_ref
           OR o.resultado_huella_sha256 IS DISTINCT FROM h
           OR o.estado_resultado IS DISTINCT FROM r.estado
           OR o.plantilla_ref IS DISTINCT FROM r.plantilla_ref
           OR o.version_resultante IS DISTINCT FROM r.version_resultante
           OR o.creada_en IS DISTINCT FROM r.registrado_en THEN
            RAISE EXCEPTION 'outbox de resultado ausente o divergente' USING ERRCODE='P0883';
        END IF;
        recibo_auditoria:=vec_bolsa_registro_accesos.registrar_resultado_correo_llamamiento_ct_v1(
            p_auditoria,p_solicitud,p_decision,p_contexto,consumo.decision_ref,
            consumo.auditoria_ref,consumo.consumo_huella_sha256,true,r.auditoria_ref);
        IF recibo_auditoria->>'id' IS NULL OR recibo_auditoria->>'id' !~ '^acc_[0-9a-f]{40}$'
           OR recibo_auditoria->>'id'=r.auditoria_ref THEN
            RAISE EXCEPTION 'auditoría de recuperación divergente' USING ERRCODE='55000';
        END IF;
        PERFORM vec_contratacion_temporal.incorporacion75_ventana(c,d,clock_timestamp());
        RETURN jsonb_build_object('IntentoRef',i.intento_ref,'SolicitudHuella',i.solicitud_huella_sha256,
            'Estado',r.estado,'PlantillaRef',r.plantilla_ref,'VersionResultante',r.version_resultante,
            'AuditoriaRef',r.auditoria_ref,'EventoRef',o.evento_ref,'RegistradoEn',r.registrado_en,'YaRegistrado',true);
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.despacho_correo_llamamiento_resultado_v1 WHERE intento_ref=i.intento_ref)
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.despacho_correo_llamamiento_outbox_v1 WHERE intento_ref=i.intento_ref) THEN
        RAISE EXCEPTION 'resultado de despacho previo incompatible' USING ERRCODE='P0883';
    END IF;
    ahora := date_trunc('microseconds',clock_timestamp());
    recibo_auditoria:=vec_bolsa_registro_accesos.registrar_resultado_correo_llamamiento_ct_v1(
        p_auditoria,p_solicitud,p_decision,p_contexto,consumo.decision_ref,
        consumo.auditoria_ref,consumo.consumo_huella_sha256,false,NULL);
    auditoria:=recibo_auditoria->>'id';
    IF auditoria IS NULL OR auditoria !~ '^acc_[0-9a-f]{40}$' THEN
        RAISE EXCEPTION 'auditoría de resultado divergente' USING ERRCODE='55000';
    END IF;
    PERFORM vec_contratacion_temporal.incorporacion75_ventana(c,d,clock_timestamp());
    evento := 'evento-correo:'||gen_random_uuid()::text;
    UPDATE vec_contratacion_temporal.despacho_correo_llamamiento_intento_v1
       SET estado=s->>'Estado',plantilla_ref=s->>'PlantillaRef',resultado_registrado_en=ahora
     WHERE intento_ref=i.intento_ref AND estado='iniciado'
       AND resultado_registrado_en IS NULL AND plantilla_ref IS NULL;
    GET DIAGNOSTICS filas = ROW_COUNT;
    IF filas <> 1 THEN
        RAISE EXCEPTION 'versión de intento divergente' USING ERRCODE='P0883';
    END IF;
    INSERT INTO vec_contratacion_temporal.despacho_correo_llamamiento_resultado_v1 VALUES (
        i.intento_ref,h,consumo.decision_ref,consumo.consumo_huella_sha256,
        consumo.auditoria_ref,auditoria,s->>'Estado',s->>'PlantillaRef',2,ahora);
    INSERT INTO vec_contratacion_temporal.despacho_correo_llamamiento_outbox_v1 VALUES (
        evento,i.intento_ref,auditoria,h,s->>'Estado',s->>'PlantillaRef',2,ahora,
        'contratacion_temporal.llamamiento.correo.resultado_registrado','pendiente');
    PERFORM vec_contratacion_temporal.incorporacion75_ventana(c,d,clock_timestamp());
    RETURN jsonb_build_object('IntentoRef',i.intento_ref,'SolicitudHuella',i.solicitud_huella_sha256,
        'Estado',s->>'Estado','PlantillaRef',s->>'PlantillaRef','VersionResultante',2,
        'AuditoriaRef',auditoria,'EventoRef',evento,'RegistradoEn',ahora,'YaRegistrado',false);
EXCEPTION WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN
    RAISE EXCEPTION 'resultado de despacho no disponible' USING ERRCODE='P0882';
WHEN data_exception THEN
    RAISE EXCEPTION 'material de resultado inválido' USING ERRCODE='22023';
END
$funcion$;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.reservar_intento_despacho_correo_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC, vec_contratacion_temporal_ejecutor;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_resultado_intento_despacho_correo_llamamiento_v1(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.reservar_intento_despacho_correo_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_resultado_intento_despacho_correo_llamamiento_v1(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;
-- Eliminar ACL por defecto ajenas sólo en las tres funciones nuevas.
DO $acl_funciones$
DECLARE f regprocedure; a record;
BEGIN
    FOREACH f IN ARRAY ARRAY[
        'vec_contratacion_temporal.correo88_validar_resultado_v1(text)'::regprocedure,
        'vec_contratacion_temporal.reservar_intento_despacho_correo_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
        'vec_contratacion_temporal.registrar_resultado_intento_despacho_correo_llamamiento_v1(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
    ] LOOP
        FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,
            LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
            WHERE p.oid=f AND x.grantee<>0 AND x.grantee<>p.proowner
              AND (f='vec_contratacion_temporal.correo88_validar_resultado_v1(text)'::regprocedure
                   OR x.grantee<>'vec_contratacion_temporal_ejecutor'::regrole) LOOP
            EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f,pg_get_userbyid(a.grantee));
        END LOOP;
    END LOOP;
END
$acl_funciones$;
COMMIT;
