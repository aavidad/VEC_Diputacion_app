\set ON_ERROR_STOP on
-- CT118: registro de firmas de prueba de los borradores del expediente.
-- Historia de solo adición: cada fila es una firma verificada o una
-- devolución con motivo de un paso del circuito de firma (catálogo con su
-- huella). La autorización V3 (AD3-85) se consume en la misma transacción que
-- escribe la firma, su auditoría y su outbox.
-- Una firma registrada aquí está verificada por el validador, pero NO tiene
-- eficacia administrativa hasta pasar por el portafirmas corporativo.
-- Requiere AD3-85. Números de error propios: P1181..P1185.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended(
    'vec_contratacion_temporal:000118_registro_firmas_documento', 0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;

DO $pre$
DECLARE f oid;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR getdatabaseencoding() <> 'UTF8'
       OR to_regclass('vec_contratacion_temporal.expediente_integral_actual') IS NULL
       OR to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NULL
       OR to_regclass('vec_contratacion_temporal.firma_documento_v1') IS NOT NULL
       OR to_regclass('vec_contratacion_temporal.firma_documento_auditoria_v1') IS NOT NULL
       OR to_regclass('vec_contratacion_temporal.firma_documento_outbox_v1') IS NOT NULL
       OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
                  WHERE n.nspname='vec_contratacion_temporal'
                    AND p.proname IN ('registrar_firma_documento_v1','consultar_firmas_documento_v1')) THEN
        RAISE EXCEPTION 'CT118: antecedentes u objetos incompatibles' USING ERRCODE='55000';
    END IF;
    f := to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_documento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
    IF f IS NULL OR NOT has_function_privilege(current_user, f, 'EXECUTE') THEN
        RAISE EXCEPTION 'CT118: AD3-85 requerido' USING ERRCODE='55000';
    END IF;
    f := to_regprocedure('vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(jsonb,text[])');
    IF f IS NULL OR NOT has_function_privilege(current_user, f, 'EXECUTE') THEN
        RAISE EXCEPTION 'CT118: validador de claves requerido' USING ERRCODE='55000';
    END IF;
    f := to_regprocedure('vec_contratacion_temporal.rechazar_mutacion_historia_v1()');
    IF f IS NULL THEN
        RAISE EXCEPTION 'CT118: guarda de historia requerida' USING ERRCODE='55000';
    END IF;
END
$pre$;

CREATE TABLE vec_contratacion_temporal.firma_documento_v1 (
    firma_ref text PRIMARY KEY CHECK (firma_ref ~ '^firma-ct:[0-9a-f-]{36}$'),
    organizacion_ref text NOT NULL CHECK (organizacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    expediente_ref text NOT NULL CHECK (expediente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    expediente_version numeric(20,0) NOT NULL CHECK (expediente_version BETWEEN 1 AND 9007199254740991::numeric),
    documento text NOT NULL CHECK (documento ~ '^[a-z][a-z0-9_]{1,63}$'),
    secuencia integer NOT NULL CHECK (secuencia BETWEEN 1 AND 100000),
    clave_idempotencia text NOT NULL CHECK (clave_idempotencia ~ '^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$'),
    solicitud_huella_sha256 text NOT NULL CHECK (solicitud_huella_sha256 ~ '^[0-9a-f]{64}$'),
    catalogo_ref text NOT NULL CHECK (catalogo_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    catalogo_huella_sha256 text NOT NULL CHECK (catalogo_huella_sha256 ~ '^[0-9a-f]{64}$' AND catalogo_huella_sha256 <> repeat('0',64)),
    paso_ref text NOT NULL CHECK (paso_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,255}$'),
    paso_orden integer NOT NULL CHECK (paso_orden BETWEEN 1 AND 16),
    resultado text NOT NULL CHECK (resultado IN ('firmado','devuelto')),
    motivo_devolucion text CHECK (motivo_devolucion IS NULL OR (char_length(motivo_devolucion) BETWEEN 3 AND 500
        AND motivo_devolucion !~ '[[:cntrl:]]' AND btrim(motivo_devolucion) = motivo_devolucion)),
    original_huella_sha256 text CHECK (original_huella_sha256 ~ '^[0-9a-f]{64}$'),
    firmado_huella_sha256 text CHECK (firmado_huella_sha256 ~ '^[0-9a-f]{64}$'),
    certificado_huella_sha256 text CHECK (certificado_huella_sha256 ~ '^[0-9a-f]{64}$'),
    firmante_ref text CHECK (firmante_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    politica_verificacion text CHECK (politica_verificacion = 'politica:vec:firma:verificacion-autonoma:v1'),
    verificacion_estado text CHECK (verificacion_estado = 'valida'),
    verificacion_motivo text CHECK (verificacion_motivo = 'verificada'),
    revocacion_estado text CHECK (revocacion_estado = 'vigente'),
    sello_tiempo_estado text CHECK (sello_tiempo_estado IN ('no_presente','valido','no_comprobado')),
    actor_ref text NOT NULL CHECK (char_length(actor_ref) BETWEEN 3 AND 512 AND actor_ref !~ '[[:cntrl:][:space:]]'),
    perfil_ref text NOT NULL CHECK (char_length(perfil_ref) BETWEEN 3 AND 512 AND perfil_ref !~ '[[:cntrl:][:space:]]'),
    decision_ref text NOT NULL CHECK (decision_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$' AND consumo_huella_sha256 <> repeat('0',64)),
    auditoria_consumo_ref text NOT NULL UNIQUE CHECK (auditoria_consumo_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo-firma-ct:[0-9a-f-]{36}$'),
    registrada_en timestamptz(6) NOT NULL CHECK (isfinite(registrada_en)),
    -- Una firma lleva todo su dictamen; una devolución no lleva ninguno.
    CHECK ((resultado = 'firmado') = (motivo_devolucion IS NULL)),
    CHECK (resultado <> 'firmado' OR (original_huella_sha256 IS NOT NULL AND firmado_huella_sha256 IS NOT NULL
        AND certificado_huella_sha256 IS NOT NULL AND firmante_ref IS NOT NULL AND politica_verificacion IS NOT NULL
        AND verificacion_estado IS NOT NULL AND verificacion_motivo IS NOT NULL AND revocacion_estado IS NOT NULL
        AND sello_tiempo_estado IS NOT NULL AND original_huella_sha256 <> firmado_huella_sha256)),
    CHECK (resultado <> 'devuelto' OR (original_huella_sha256 IS NULL AND firmado_huella_sha256 IS NULL
        AND certificado_huella_sha256 IS NULL AND firmante_ref IS NULL AND politica_verificacion IS NULL
        AND verificacion_estado IS NULL AND verificacion_motivo IS NULL AND revocacion_estado IS NULL
        AND sello_tiempo_estado IS NULL)),
    UNIQUE (organizacion_ref, expediente_ref, documento, secuencia),
    UNIQUE (organizacion_ref, expediente_ref, clave_idempotencia),
    UNIQUE (firma_ref, recibo_ref, registrada_en)
);
COMMENT ON TABLE vec_contratacion_temporal.firma_documento_v1 IS
    'Firmas de prueba verificadas y devoluciones del circuito de firma de los borradores CT. Solo adición. Sin eficacia administrativa hasta el portafirmas corporativo; no conserva el documento ni la firma, solo sus huellas.';

CREATE TABLE vec_contratacion_temporal.firma_documento_auditoria_v1 (
    auditoria_ref text PRIMARY KEY CHECK (auditoria_ref ~ '^auditoria-firma-ct:[0-9a-f-]{36}$'),
    firma_ref text NOT NULL UNIQUE,
    recibo_ref text NOT NULL UNIQUE,
    operacion text NOT NULL CHECK (operacion = 'contratacion_temporal.documento.firmar'),
    resultado text NOT NULL CHECK (resultado IN ('firmado','devuelto')),
    decision_ref text NOT NULL,
    consumo_huella_sha256 text NOT NULL UNIQUE,
    auditoria_consumo_ref text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (firma_ref, recibo_ref, registrada_en)
        REFERENCES vec_contratacion_temporal.firma_documento_v1 (firma_ref, recibo_ref, registrada_en)
);
COMMENT ON TABLE vec_contratacion_temporal.firma_documento_auditoria_v1 IS
    'Auditoría 1:1 de cada firma o devolución registrada, ligada a la decisión y al consumo V3.';

CREATE TABLE vec_contratacion_temporal.firma_documento_outbox_v1 (
    evento_ref text PRIMARY KEY CHECK (evento_ref ~ '^evento-firma-ct:[0-9a-f-]{36}$'),
    firma_ref text NOT NULL UNIQUE,
    recibo_ref text NOT NULL UNIQUE,
    tipo text NOT NULL CHECK (tipo IN ('contratacion_temporal.documento.firmado','contratacion_temporal.documento.devuelto')),
    estado text NOT NULL CHECK (estado = 'pendiente'),
    creada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (firma_ref, recibo_ref, creada_en)
        REFERENCES vec_contratacion_temporal.firma_documento_v1 (firma_ref, recibo_ref, registrada_en)
);
COMMENT ON TABLE vec_contratacion_temporal.firma_documento_outbox_v1 IS
    'Evento minimizado 1:1 de la firma o devolución, pendiente de publicación; solo referencias opacas.';

DO $seguridad$
DECLARE t text; a record;
BEGIN
    FOREACH t IN ARRAY ARRAY['firma_documento_v1','firma_documento_auditoria_v1','firma_documento_outbox_v1'] LOOP
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
        EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_contratacion_temporal.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()',t);
    END LOOP;
END
$seguridad$;

-- La solicitud la canonicaliza el adaptador Go: orden fijo de claves, sin
-- espacios. Su huella es la del material que la decisión V3 ata al recurso.
CREATE FUNCTION vec_contratacion_temporal.registrar_firma_documento_v1(
    p_solicitud text,
    p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
    p_persona_version numeric, p_perfil_version numeric,
    p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE s jsonb; d jsonb; consumo record; previa record; version_actual numeric; organizacion text;
    h text; contexto_h text; ahora timestamptz(6); firma text; recibo text; auditoria text; evento text;
    siguiente integer; firmado boolean; recurso text;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR session_user = current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
       OR current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION 'registro de firma denegado' USING ERRCODE='42501';
    END IF;
    IF p_solicitud IS NULL OR octet_length(p_solicitud) NOT BETWEEN 2 AND 8192 THEN
        RAISE EXCEPTION 'solicitud de firma inválida' USING ERRCODE='22023';
    END IF;
    s := p_solicitud::jsonb;
    IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(s,ARRAY[
        'OrganizacionRef','ExpedienteRef','VersionExpediente','Documento','CatalogoRef','CatalogoHuella',
        'PasoRef','PasoOrden','Secuencia','Resultado','MotivoDevolucion','OriginalHuella','FirmadoHuella',
        'CertificadoHuella','FirmanteRef','PoliticaVerificacion','RevocacionEstado','SelloTiempoEstado',
        'ClaveIdempotencia']) IS NOT TRUE
       OR jsonb_typeof(s->'VersionExpediente') <> 'number' OR jsonb_typeof(s->'PasoOrden') <> 'number'
       OR jsonb_typeof(s->'Secuencia') <> 'number'
       OR (s->>'VersionExpediente') !~ '^[1-9][0-9]{0,15}$' OR (s->>'PasoOrden') !~ '^[1-9][0-9]?$'
       OR (s->>'Secuencia') !~ '^[1-9][0-9]{0,5}$'
       OR EXISTS (SELECT 1 FROM jsonb_each(s) x WHERE x.key NOT IN ('VersionExpediente','PasoOrden','Secuencia')
                  AND jsonb_typeof(x.value) NOT IN ('string','null'))
       OR s->>'Resultado' IS NULL OR s->>'Resultado' NOT IN ('firmado','devuelto')
       OR (s->>'ClaveIdempotencia') !~ '^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$' THEN
        RAISE EXCEPTION 'solicitud de firma inválida' USING ERRCODE='22023';
    END IF;
    firmado := s->>'Resultado' = 'firmado';
    IF firmado IS DISTINCT FROM (jsonb_typeof(s->'MotivoDevolucion') = 'null')
       OR EXISTS (SELECT 1 FROM unnest(ARRAY['OriginalHuella','FirmadoHuella','CertificadoHuella','FirmanteRef',
                  'PoliticaVerificacion','RevocacionEstado','SelloTiempoEstado']) k
                  WHERE (jsonb_typeof(s->k) = 'null') = firmado) THEN
        RAISE EXCEPTION 'solicitud de firma incoherente' USING ERRCODE='22023';
    END IF;
    h := encode(sha256(convert_to(p_solicitud,'UTF8')),'hex');
    recurso := 'operacion-firma-ct:'||(s->>'ClaveIdempotencia');
    contexto_h := encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
        '"},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
    BEGIN
        d := convert_from(p_decision,'UTF8')::jsonb;
    EXCEPTION WHEN others THEN
        RAISE EXCEPTION 'decisión de firma inválida' USING ERRCODE='42501';
    END;
    IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.documento.firmar'
       OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'firma_documento_contratacion_temporal'
       OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
       OR d->>'recurso_ref' IS DISTINCT FROM recurso
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
       OR coalesce(d->>'principal_id','') = '' OR coalesce(d->>'perfil_activo_ref','') = '' THEN
        RAISE EXCEPTION 'autorización de firma divergente' USING ERRCODE='42501';
    END IF;
    SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_firma_documento_ct_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF consumo.efecto_ref IS DISTINCT FROM recurso
       OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_h
       OR consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'consumo de firma divergente' USING ERRCODE='42501';
    END IF;
    -- Una sola escritura a la vez por documento del expediente.
    PERFORM pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:firma_documento:'||
        (s->>'ExpedienteRef')||':'||(s->>'Documento'),0));
    SELECT * INTO previa FROM vec_contratacion_temporal.firma_documento_v1
     WHERE organizacion_ref=s->>'OrganizacionRef' AND expediente_ref=s->>'ExpedienteRef'
       AND clave_idempotencia=s->>'ClaveIdempotencia';
    IF FOUND THEN
        IF previa.solicitud_huella_sha256 IS DISTINCT FROM h THEN
            RAISE EXCEPTION 'clave de firma reutilizada con otro material' USING ERRCODE='P1181';
        END IF;
        RETURN jsonb_build_object('FirmaRef',previa.firma_ref,'ReciboRef',previa.recibo_ref,
            'Secuencia',previa.secuencia,'Resultado',previa.resultado,'ExpedienteVersion',previa.expediente_version,
            'ActorRef',previa.actor_ref,'PerfilRef',previa.perfil_ref,'RegistradaEn',previa.registrada_en,
            'SolicitudHuella',previa.solicitud_huella_sha256,'YaRegistrada',true);
    END IF;
    SELECT a.version, v.agregado_json->>'organizacion_ref' INTO version_actual, organizacion
      FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v
        ON v.expediente_ref=a.expediente_ref AND v.version=a.version
     WHERE a.expediente_ref=s->>'ExpedienteRef'
     FOR KEY SHARE OF a;
    IF NOT FOUND OR organizacion IS DISTINCT FROM s->>'OrganizacionRef' THEN
        RAISE EXCEPTION 'expediente de firma no disponible' USING ERRCODE='42501';
    END IF;
    IF version_actual <> (s->>'VersionExpediente')::numeric THEN
        RAISE EXCEPTION 'versión del expediente en conflicto' USING ERRCODE='P1182';
    END IF;
    SELECT coalesce(max(secuencia),0)+1 INTO siguiente FROM vec_contratacion_temporal.firma_documento_v1
     WHERE organizacion_ref=s->>'OrganizacionRef' AND expediente_ref=s->>'ExpedienteRef'
       AND documento=s->>'Documento';
    IF siguiente <> (s->>'Secuencia')::integer THEN
        RAISE EXCEPTION 'secuencia de firma en conflicto' USING ERRCODE='P1183';
    END IF;
    -- Cadena: a partir del paso 2 se firma exactamente el mismo borrador que
    -- firmó el paso anterior del mismo circuito (misma huella de catálogo).
    -- Cada paso firma el borrador por separado; el orden lo da el circuito.
    IF firmado AND (s->>'PasoOrden')::integer > 1 AND NOT EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1 f
         WHERE f.organizacion_ref=s->>'OrganizacionRef' AND f.expediente_ref=s->>'ExpedienteRef'
           AND f.documento=s->>'Documento' AND f.resultado='firmado'
           AND f.catalogo_huella_sha256=s->>'CatalogoHuella'
           AND f.paso_orden=(s->>'PasoOrden')::integer-1
           AND f.original_huella_sha256=s->>'OriginalHuella') THEN
        RAISE EXCEPTION 'la firma no es sobre el borrador del paso anterior' USING ERRCODE='P1184';
    END IF;
    ahora := date_trunc('microseconds',clock_timestamp());
    firma := 'firma-ct:'||gen_random_uuid()::text;
    recibo := 'recibo-firma-ct:'||gen_random_uuid()::text;
    auditoria := 'auditoria-firma-ct:'||gen_random_uuid()::text;
    evento := 'evento-firma-ct:'||gen_random_uuid()::text;
    INSERT INTO vec_contratacion_temporal.firma_documento_v1 (
        firma_ref,organizacion_ref,expediente_ref,expediente_version,documento,secuencia,clave_idempotencia,
        solicitud_huella_sha256,catalogo_ref,catalogo_huella_sha256,paso_ref,paso_orden,resultado,motivo_devolucion,
        original_huella_sha256,firmado_huella_sha256,certificado_huella_sha256,firmante_ref,politica_verificacion,
        verificacion_estado,verificacion_motivo,revocacion_estado,sello_tiempo_estado,actor_ref,perfil_ref,
        decision_ref,consumo_huella_sha256,auditoria_consumo_ref,recibo_ref,registrada_en)
    VALUES (firma,s->>'OrganizacionRef',s->>'ExpedienteRef',version_actual,s->>'Documento',siguiente,
        s->>'ClaveIdempotencia',h,s->>'CatalogoRef',s->>'CatalogoHuella',s->>'PasoRef',(s->>'PasoOrden')::integer,
        s->>'Resultado',s->>'MotivoDevolucion',s->>'OriginalHuella',s->>'FirmadoHuella',s->>'CertificadoHuella',
        s->>'FirmanteRef',s->>'PoliticaVerificacion',CASE WHEN firmado THEN 'valida' END,
        CASE WHEN firmado THEN 'verificada' END,s->>'RevocacionEstado',s->>'SelloTiempoEstado',
        d->>'principal_id',d->>'perfil_activo_ref',consumo.decision_ref,consumo.consumo_huella_sha256,
        consumo.auditoria_ref,recibo,ahora);
    INSERT INTO vec_contratacion_temporal.firma_documento_auditoria_v1 VALUES (
        auditoria,firma,recibo,'contratacion_temporal.documento.firmar',s->>'Resultado',consumo.decision_ref,
        consumo.consumo_huella_sha256,consumo.auditoria_ref,ahora);
    INSERT INTO vec_contratacion_temporal.firma_documento_outbox_v1 VALUES (
        evento,firma,recibo,CASE WHEN firmado THEN 'contratacion_temporal.documento.firmado'
        ELSE 'contratacion_temporal.documento.devuelto' END,'pendiente',ahora);
    RETURN jsonb_build_object('FirmaRef',firma,'ReciboRef',recibo,'Secuencia',siguiente,'Resultado',s->>'Resultado',
        'ExpedienteVersion',version_actual,'ActorRef',d->>'principal_id','PerfilRef',d->>'perfil_activo_ref',
        'RegistradaEn',ahora,'SolicitudHuella',h,'YaRegistrada',false);
EXCEPTION WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN
    RAISE EXCEPTION 'registro de firma no disponible' USING ERRCODE='P1185';
WHEN data_exception THEN
    RAISE EXCEPTION 'solicitud de firma inválida' USING ERRCODE='22023';
END
$funcion$;

-- Lectura de la historia de firmas de un expediente, en orden de registro.
-- Devuelve solo huellas y referencias; nunca el documento ni la firma.
CREATE FUNCTION vec_contratacion_temporal.consultar_firmas_documento_v1(
    p_organizacion_ref text, p_expediente_ref text
) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR session_user = current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER') THEN
        RAISE EXCEPTION 'consulta de firmas denegada' USING ERRCODE='42501';
    END IF;
    IF p_organizacion_ref IS NULL OR p_organizacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_expediente_ref IS NULL OR p_expediente_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
        RAISE EXCEPTION 'consulta de firmas inválida' USING ERRCODE='22023';
    END IF;
    RETURN coalesce((SELECT jsonb_agg(jsonb_build_object(
        'FirmaRef',f.firma_ref,'ReciboRef',f.recibo_ref,'Documento',f.documento,'Secuencia',f.secuencia,
        'ExpedienteVersion',f.expediente_version,'CatalogoRef',f.catalogo_ref,'CatalogoHuella',f.catalogo_huella_sha256,
        'PasoRef',f.paso_ref,'PasoOrden',f.paso_orden,'Resultado',f.resultado,'MotivoDevolucion',f.motivo_devolucion,
        'OriginalHuella',f.original_huella_sha256,'FirmadoHuella',f.firmado_huella_sha256,
        'CertificadoHuella',f.certificado_huella_sha256,'FirmanteRef',f.firmante_ref,
        'SelloTiempoEstado',f.sello_tiempo_estado,'ActorRef',f.actor_ref,'PerfilRef',f.perfil_ref,
        'RegistradaEn',f.registrada_en) ORDER BY f.documento,f.secuencia)
      FROM (SELECT * FROM vec_contratacion_temporal.firma_documento_v1
             WHERE organizacion_ref=p_organizacion_ref AND expediente_ref=p_expediente_ref
             ORDER BY documento,secuencia LIMIT 1000) f),'[]'::jsonb);
END
$funcion$;

DO $acl_funciones$
DECLARE f regprocedure; a record;
BEGIN
    FOREACH f IN ARRAY ARRAY[
        'vec_contratacion_temporal.registrar_firma_documento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
        'vec_contratacion_temporal.consultar_firmas_documento_v1(text,text)'::regprocedure
    ] LOOP
        EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f);
        FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,
            LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
            WHERE p.oid=f AND x.grantee<>0 AND x.grantee<>p.proowner LOOP
            EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f,pg_get_userbyid(a.grantee));
        END LOOP;
        EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_contratacion_temporal_ejecutor',f);
    END LOOP;
END
$acl_funciones$;
COMMIT;
