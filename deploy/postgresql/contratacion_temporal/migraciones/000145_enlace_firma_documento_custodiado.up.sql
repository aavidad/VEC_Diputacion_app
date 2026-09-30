\set ON_ERROR_STOP on
-- CT145 (5.06, E2): enlace de cada firma con el PDF firmado que custodia
-- Documentos. El enlace va dentro del material que autoriza la V3 (AD3-85) y
-- se escribe en la misma transacción que la firma, su auditoría y su outbox.
-- CT no lee tablas de Documentos: guarda la referencia opaca, la versión y la
-- huella (la del firmado verificado) que Documentos devolvió al custodiar.
-- registrar/consultar v2 sustituyen a las v1, que dejan de poder ejecutarse:
-- una firma no puede registrarse ya sin pasar por el material nuevo.
-- Requiere CT118 y AD3-85. Errores: los de CT118 (P1181..P1185).
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000145', 0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;

DO $pre$
DECLARE f oid;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR getdatabaseencoding() <> 'UTF8'
       OR to_regclass('vec_contratacion_temporal.firma_documento_v1') IS NULL
       OR to_regclass('vec_contratacion_temporal.firma_documento_custodia_v1') IS NOT NULL
       OR to_regprocedure('vec_contratacion_temporal.registrar_firma_documento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_contratacion_temporal.consultar_firmas_documento_v1(text,text)') IS NULL
       OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
                  WHERE n.nspname='vec_contratacion_temporal'
                    AND p.proname IN ('registrar_firma_documento_v2','consultar_firmas_documento_v2')) THEN
        RAISE EXCEPTION 'CT145: antecedentes u objetos incompatibles' USING ERRCODE='55000';
    END IF;
    f := to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_documento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
    IF f IS NULL OR NOT has_function_privilege(current_user, f, 'EXECUTE') THEN
        RAISE EXCEPTION 'CT145: AD3-85 requerido' USING ERRCODE='55000';
    END IF;
END
$pre$;

CREATE TABLE vec_contratacion_temporal.firma_documento_custodia_v1 (
    firma_ref text PRIMARY KEY,
    recibo_ref text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL,
    documento_ref text NOT NULL CHECK (documento_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    documento_version numeric(20,0) NOT NULL CHECK (documento_version BETWEEN 1 AND 9007199254740991::numeric),
    documento_huella_sha256 text NOT NULL CHECK (documento_huella_sha256 ~ '^[0-9a-f]{64}$'),
    -- Un documento custodiado pertenece a una sola firma.
    CONSTRAINT firma_documento_custodia_v1_documento_unico UNIQUE (documento_ref, documento_version),
    FOREIGN KEY (firma_ref, recibo_ref, registrada_en)
        REFERENCES vec_contratacion_temporal.firma_documento_v1 (firma_ref, recibo_ref, registrada_en)
);
COMMENT ON TABLE vec_contratacion_temporal.firma_documento_custodia_v1 IS
    'Enlace 1:1 y de solo adición entre una firma registrada y el PDF firmado que custodia Documentos (referencia opaca, versión y huella del firmado).';

DO $seguridad$
DECLARE a record;
BEGIN
    ALTER TABLE vec_contratacion_temporal.firma_documento_custodia_v1 ENABLE ROW LEVEL SECURITY;
    ALTER TABLE vec_contratacion_temporal.firma_documento_custodia_v1 FORCE ROW LEVEL SECURITY;
    CREATE POLICY propietario ON vec_contratacion_temporal.firma_documento_custodia_v1
        TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true);
    REVOKE ALL ON TABLE vec_contratacion_temporal.firma_documento_custodia_v1 FROM PUBLIC;
    FOR a IN SELECT DISTINCT x.grantee FROM pg_class c,
        LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
        WHERE c.oid='vec_contratacion_temporal.firma_documento_custodia_v1'::regclass
          AND x.grantee<>0 AND x.grantee<>c.relowner LOOP
        EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.firma_documento_custodia_v1 FROM %I',pg_get_userbyid(a.grantee));
    END LOOP;
    REVOKE ALL ON TYPE vec_contratacion_temporal.firma_documento_custodia_v1 FROM PUBLIC;
    FOR a IN SELECT DISTINCT x.grantee FROM pg_type ty,
        LATERAL aclexplode(coalesce(ty.typacl,acldefault('T',ty.typowner))) x
        WHERE ty.oid=(SELECT c.reltype FROM pg_class c WHERE c.oid='vec_contratacion_temporal.firma_documento_custodia_v1'::regclass)
          AND x.grantee<>0 AND x.grantee<>ty.typowner LOOP
        EXECUTE format('REVOKE ALL ON TYPE vec_contratacion_temporal.firma_documento_custodia_v1 FROM %I',pg_get_userbyid(a.grantee));
    END LOOP;
    CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE
        ON vec_contratacion_temporal.firma_documento_custodia_v1
        FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
END
$seguridad$;

-- registrar v2: el cuerpo de v1 con dos claves más en la solicitud
-- (DocumentoCustodiaRef y DocumentoCustodiaVersion, ambas nulas o ambas
-- presentes, solo en una firma) y la fila del enlace en la misma transacción.
CREATE FUNCTION vec_contratacion_temporal.registrar_firma_documento_v2(
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
    siguiente integer; firmado boolean; custodia boolean; recurso text; enlace record;
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
    -- jsonb se queda con la última de dos claves repetidas y la huella se
    -- calcula sobre el texto: una clave repetida se rechaza.
    IF (SELECT count(*) FROM json_each(p_solicitud::json)) <> (SELECT count(*) FROM jsonb_each(s)) THEN
        RAISE EXCEPTION 'solicitud de firma inválida' USING ERRCODE='22023';
    END IF;
    IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(s,ARRAY[
        'OrganizacionRef','ExpedienteRef','VersionExpediente','Documento','CatalogoRef','CatalogoHuella',
        'PasoRef','PasoOrden','Secuencia','Resultado','MotivoDevolucion','OriginalHuella','FirmadoHuella',
        'CertificadoHuella','FirmanteRef','PoliticaVerificacion','RevocacionEstado','SelloTiempoEstado',
        'ClaveIdempotencia','DocumentoCustodiaRef','DocumentoCustodiaVersion']) IS NOT TRUE
       OR jsonb_typeof(s->'VersionExpediente') <> 'number' OR jsonb_typeof(s->'PasoOrden') <> 'number'
       OR jsonb_typeof(s->'Secuencia') <> 'number'
       OR (s->>'VersionExpediente') !~ '^[1-9][0-9]{0,15}$' OR (s->>'PasoOrden') !~ '^([1-9]|1[0-6])$'
       OR (s->>'Secuencia') !~ '^[1-9][0-9]{0,5}$'
       OR EXISTS (SELECT 1 FROM jsonb_each(s) x WHERE x.key NOT IN ('VersionExpediente','PasoOrden','Secuencia','DocumentoCustodiaVersion')
                  AND jsonb_typeof(x.value) NOT IN ('string','null'))
       OR jsonb_typeof(s->'DocumentoCustodiaVersion') NOT IN ('number','null')
       OR (jsonb_typeof(s->'DocumentoCustodiaVersion') = 'number'
           AND ((s->>'DocumentoCustodiaVersion') !~ '^[1-9][0-9]{0,15}$'
                OR (s->>'DocumentoCustodiaVersion')::numeric > 9007199254740991))
       OR (jsonb_typeof(s->'DocumentoCustodiaRef') = 'string'
           AND (s->>'DocumentoCustodiaRef') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR s->>'Resultado' IS NULL OR s->>'Resultado' NOT IN ('firmado','devuelto')
       OR (s->>'ClaveIdempotencia') !~ '^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$' THEN
        RAISE EXCEPTION 'solicitud de firma inválida' USING ERRCODE='22023';
    END IF;
    firmado := s->>'Resultado' = 'firmado';
    custodia := jsonb_typeof(s->'DocumentoCustodiaRef') = 'string';
    -- El documento custodiado (Documentos, 5.06) es el PDF firmado: solo una
    -- firma lo lleva, con referencia y versión juntas. Su huella es la del
    -- firmado, así que no viaja aparte.
    IF custodia IS DISTINCT FROM (jsonb_typeof(s->'DocumentoCustodiaVersion') = 'number')
       OR (custodia AND NOT firmado) THEN
        RAISE EXCEPTION 'solicitud de firma incoherente' USING ERRCODE='22023';
    END IF;
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
    -- El cerrojo solo ordena las escrituras de un mismo documento: en
    -- SERIALIZABLE la instantánea ya está tomada al llegar aquí, así que no
    -- impide que dos registros simultáneos lean la misma historia. Lo que
    -- protege son las restricciones únicas de clave y de secuencia y la
    -- detección de conflictos de SERIALIZABLE: la transacción que pierde
    -- termina con 23505 (con el nombre de la restricción) o 40001, que se
    -- dejan salir sin convertir para que el adaptador recupere el recibo,
    -- informe del conflicto o reintente.
    PERFORM pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:firma_documento:'||
        (s->>'ExpedienteRef')||':'||(s->>'Documento'),0));
    SELECT * INTO previa FROM vec_contratacion_temporal.firma_documento_v1
     WHERE organizacion_ref=s->>'OrganizacionRef' AND expediente_ref=s->>'ExpedienteRef'
       AND clave_idempotencia=s->>'ClaveIdempotencia';
    IF FOUND THEN
        IF previa.solicitud_huella_sha256 IS DISTINCT FROM h THEN
            RAISE EXCEPTION 'clave de firma reutilizada con otro material' USING ERRCODE='P1181';
        END IF;
        -- Misma huella: mismo material, luego mismo enlace (o ninguno).
        SELECT * INTO enlace FROM vec_contratacion_temporal.firma_documento_custodia_v1 WHERE firma_ref=previa.firma_ref;
        RETURN jsonb_build_object('FirmaRef',previa.firma_ref,'ReciboRef',previa.recibo_ref,
            'Secuencia',previa.secuencia,'Resultado',previa.resultado,'ExpedienteVersion',previa.expediente_version,
            'ActorRef',previa.actor_ref,'PerfilRef',previa.perfil_ref,'RegistradaEn',previa.registrada_en,
            'SolicitudHuella',previa.solicitud_huella_sha256,'YaRegistrada',true,
            'DocumentoCustodiaRef',enlace.documento_ref,'DocumentoCustodiaVersion',enlace.documento_version);
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
    IF custodia THEN
        INSERT INTO vec_contratacion_temporal.firma_documento_custodia_v1 VALUES (
            firma,recibo,ahora,s->>'DocumentoCustodiaRef',(s->>'DocumentoCustodiaVersion')::numeric,s->>'FirmadoHuella');
    END IF;
    INSERT INTO vec_contratacion_temporal.firma_documento_auditoria_v1 VALUES (
        auditoria,firma,recibo,'contratacion_temporal.documento.firmar',s->>'Resultado',consumo.decision_ref,
        consumo.consumo_huella_sha256,consumo.auditoria_ref,ahora);
    INSERT INTO vec_contratacion_temporal.firma_documento_outbox_v1 VALUES (
        evento,firma,recibo,CASE WHEN firmado THEN 'contratacion_temporal.documento.firmado'
        ELSE 'contratacion_temporal.documento.devuelto' END,'pendiente',ahora);
    RETURN jsonb_build_object('FirmaRef',firma,'ReciboRef',recibo,'Secuencia',siguiente,'Resultado',s->>'Resultado',
        'ExpedienteVersion',version_actual,'ActorRef',d->>'principal_id','PerfilRef',d->>'perfil_activo_ref',
        'RegistradaEn',ahora,'SolicitudHuella',h,'YaRegistrada',false,
        'DocumentoCustodiaRef',s->'DocumentoCustodiaRef','DocumentoCustodiaVersion',s->'DocumentoCustodiaVersion');
EXCEPTION WHEN lock_not_available THEN
    RAISE EXCEPTION 'registro de firma no disponible' USING ERRCODE='P1185';
WHEN data_exception THEN
    RAISE EXCEPTION 'solicitud de firma inválida' USING ERRCODE='22023';
END
$funcion$;


-- consultar v2: la de v1 más la referencia y versión del documento custodiado
-- y la clave de idempotencia (opaca, la genera el cliente), con la que la
-- aplicación reconoce el reintento de una firma ya registrada.
CREATE FUNCTION vec_contratacion_temporal.consultar_firmas_documento_v2(
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
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') THEN
        RAISE EXCEPTION 'consulta de firmas denegada' USING ERRCODE='42501';
    END IF;
    IF p_organizacion_ref IS NULL OR p_organizacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_expediente_ref IS NULL OR p_expediente_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
        RAISE EXCEPTION 'consulta de firmas inválida' USING ERRCODE='22023';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_integral_actual a
                 JOIN vec_contratacion_temporal.expediente_version_integral v
                   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
                WHERE a.expediente_ref=p_expediente_ref
                  AND v.agregado_json->>'organizacion_ref' IS DISTINCT FROM p_organizacion_ref) THEN
        RAISE EXCEPTION 'consulta de firmas denegada' USING ERRCODE='42501';
    END IF;
    RETURN coalesce((SELECT jsonb_agg(jsonb_build_object(
        'FirmaRef',f.firma_ref,'ReciboRef',f.recibo_ref,'Documento',f.documento,'Secuencia',f.secuencia,
        'ExpedienteVersion',f.expediente_version,'CatalogoRef',f.catalogo_ref,'CatalogoHuella',f.catalogo_huella_sha256,
        'PasoRef',f.paso_ref,'PasoOrden',f.paso_orden,'Resultado',f.resultado,'ConMotivoDevolucion',f.motivo_devolucion IS NOT NULL,
        'OriginalHuella',f.original_huella_sha256,'FirmadoHuella',f.firmado_huella_sha256,
        'SelloTiempoEstado',f.sello_tiempo_estado,
        'RegistradaEn',f.registrada_en,'ClaveIdempotencia',f.clave_idempotencia,
        'DocumentoCustodiaRef',c.documento_ref,'DocumentoCustodiaVersion',c.documento_version) ORDER BY f.documento,f.secuencia)
      FROM (SELECT * FROM vec_contratacion_temporal.firma_documento_v1
             WHERE organizacion_ref=p_organizacion_ref AND expediente_ref=p_expediente_ref
             ORDER BY documento,secuencia LIMIT 1000) f
      LEFT JOIN vec_contratacion_temporal.firma_documento_custodia_v1 c ON c.firma_ref=f.firma_ref),'[]'::jsonb);
END
$funcion$;


DO $acl_funciones$
DECLARE f regprocedure; a record;
BEGIN
    FOREACH f IN ARRAY ARRAY[
        'vec_contratacion_temporal.registrar_firma_documento_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
        'vec_contratacion_temporal.consultar_firmas_documento_v2(text,text)'::regprocedure
    ] LOOP
        EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f);
        FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,
            LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
            WHERE p.oid=f AND x.grantee<>0 AND x.grantee<>p.proowner LOOP
            EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f,pg_get_userbyid(a.grantee));
        END LOOP;
        EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_contratacion_temporal_ejecutor',f);
    END LOOP;
    -- Las v1 dejan de estar al alcance de la aplicación.
    REVOKE EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_firma_documento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
        FROM vec_contratacion_temporal_ejecutor;
    REVOKE EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_firmas_documento_v1(text,text)
        FROM vec_contratacion_temporal_ejecutor;
END
$acl_funciones$;
COMMIT;
