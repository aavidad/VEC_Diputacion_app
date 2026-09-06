\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000060',0));
LOCK TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh IN ACCESS EXCLUSIVE MODE;
DO $dependencias$
BEGIN
    IF (SELECT count(*) FROM pg_attribute
        WHERE attrelid='vec_contratacion_temporal.resolucion_manual_respuesta_rrhh'::regclass
          AND NOT attisdropped AND attnum>0
          AND ((attname='comando_siguiente_ref' AND atttypid='text'::regtype)
            OR (attname='comando_siguiente_json' AND atttypid='jsonb'::regtype)))<>2
       OR NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
        WHERE n.nspname='vec_autorizacion_atestada_v3'
          AND p.proname='registrar_y_consumir_continuacion_ct_v3_atestada'
          AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef) THEN
        RAISE EXCEPTION 'faltan intención CT59 o consumidor 19' USING ERRCODE='55000';
    END IF;
END
$dependencias$;
-- Solo confirmación adicional en la fila CT59: todos los bytes originales
-- siguen inmutables. No lee/escribe Bolsa ni ejecuta el siguiente candidato.
ALTER TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
    ADD COLUMN continuacion_clave uuid,
    ADD COLUMN continuacion_material text,
    ADD COLUMN continuacion_material_sha256 text,
    ADD COLUMN continuacion_recibo jsonb,
    ADD COLUMN continuacion_actor_ref text,
    ADD COLUMN continuacion_perfil_ref text,
    ADD COLUMN continuacion_consumo jsonb,
    ADD CONSTRAINT continuacion_clave_unica UNIQUE (organizacion_ref,continuacion_clave),
    ADD CONSTRAINT continuacion_confirmacion_completa CHECK ((
        (num_nonnulls(continuacion_clave,continuacion_material,continuacion_material_sha256,continuacion_recibo,
            continuacion_actor_ref,continuacion_perfil_ref,continuacion_consumo)=0)
        OR (num_nonnulls(continuacion_clave,continuacion_material,continuacion_material_sha256,continuacion_recibo,
            continuacion_actor_ref,continuacion_perfil_ref,continuacion_consumo)=7
            AND solicitud_json->>'Respuesta'='renuncia'
            AND octet_length(continuacion_material) BETWEEN 1 AND 65536
            AND continuacion_material_sha256=encode(sha256(convert_to(continuacion_material,'UTF8')),'hex')
            AND (continuacion_material::jsonb)->>'Etapa'='confirmacion'
            AND (continuacion_material::jsonb)->'Solicitud'=continuacion_recibo->'Solicitud'
            AND (continuacion_material::jsonb)->'ReciboBolsa'=continuacion_recibo->'ReciboBolsa'
            AND continuacion_recibo->'Solicitud'->>'ClaveIdempotencia'=continuacion_clave::text
            AND continuacion_recibo->'Solicitud'->>'OrganizacionRef'=organizacion_ref
            AND continuacion_recibo->'Solicitud'->>'ExpedienteRef'=expediente_ref
            AND continuacion_recibo->'Solicitud'->>'ResolucionRef'=resolucion_ref
            AND continuacion_recibo->'Solicitud'->>'IntencionRef'=comando_siguiente_json->>'intencion_ref'
            AND continuacion_recibo->>'LlamamientoAnteriorRef'=llamamiento_ref
            AND continuacion_recibo->'ReciboBolsa'->>'LlamamientoRef'<>llamamiento_ref
            AND continuacion_recibo->>'Estado'='confirmado'
            AND jsonb_typeof(continuacion_consumo)='object'
            AND continuacion_actor_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
            AND continuacion_perfil_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
    ) IS TRUE);
CREATE UNIQUE INDEX continuacion_recibo_unico ON vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
    ((continuacion_recibo->>'ReciboRef')) WHERE continuacion_clave IS NOT NULL;
COMMENT ON COLUMN vec_contratacion_temporal.resolucion_manual_respuesta_rrhh.continuacion_recibo IS
    'Confirmación local inmutable del recibo Bolsa comprobado por composición confiable; no correo ni plazo legal. Recibo e intención originales CT59 no se actualizan.';

CREATE FUNCTION vec_contratacion_temporal.proteger_historia_continuacion_ct_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog
AS $proteccion$
DECLARE v_columnas text[]:=ARRAY['continuacion_clave','continuacion_material','continuacion_material_sha256',
    'continuacion_recibo','continuacion_actor_ref','continuacion_perfil_ref','continuacion_consumo'];
BEGIN
    IF TG_OP<>'UPDATE' OR current_user<>'vec_contratacion_temporal_propietario' THEN
        RAISE EXCEPTION 'historia CT inmutable' USING ERRCODE='55000';
    END IF;
    IF OLD.continuacion_clave IS NOT NULL OR NEW.continuacion_clave IS NULL
       OR (to_jsonb(OLD)-v_columnas) IS DISTINCT FROM (to_jsonb(NEW)-v_columnas) THEN
        RAISE EXCEPTION 'solo se admite una confirmación adicional CT' USING ERRCODE='55000';
    END IF;
    RETURN NEW;
END
$proteccion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.proteger_historia_continuacion_ct_v1() FROM PUBLIC;
-- Guardar el contrato preciso del trigger sustituido, no tocar la función
-- compartida ni los triggers de otras historias.
DO $trigger$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger
        WHERE tgrelid='vec_contratacion_temporal.resolucion_manual_respuesta_rrhh'::regclass
          AND tgname='historia_inmutable' AND NOT tgisinternal AND tgenabled='O' AND tgtype=27
          AND tgfoid='vec_contratacion_temporal.rechazar_mutacion_historia_v1()'::regprocedure
          AND tgnargs=0 AND tgqual IS NULL) THEN
        RAISE EXCEPTION 'trigger CT59 incompatible' USING ERRCODE='55000';
    END IF;
END
$trigger$;
DROP TRIGGER historia_inmutable ON vec_contratacion_temporal.resolucion_manual_respuesta_rrhh;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE
    ON vec_contratacion_temporal.resolucion_manual_respuesta_rrhh FOR EACH ROW
    EXECUTE FUNCTION vec_contratacion_temporal.proteger_historia_continuacion_ct_v1();

CREATE FUNCTION vec_contratacion_temporal.continuar_llamamiento_rrhh_v1(
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
    m jsonb; s jsonb; b jsonb; d jsonb; v_campo text;
    v_hash text; v_contexto_hash text; v_consumo record;
    v_fila vec_contratacion_temporal.resolucion_manual_respuesta_rrhh%ROWTYPE;
    v_ahora timestamptz(6); v_fecha_bolsa timestamptz(6); v_resultado jsonb;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
       OR current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'off' THEN
        RAISE EXCEPTION 'continuación denegada' USING ERRCODE='P0603';
    END IF;
    IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN
        RAISE EXCEPTION 'material de continuación inválido' USING ERRCODE='P0600';
    END IF;
    BEGIN m:=p_material::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'JSON de continuación inválido' USING ERRCODE='P0600';
    END;
    IF (m->'Etapa') NOT IN ('"consulta"'::jsonb,'"confirmacion"'::jsonb) OR m->'Etapa' IS NULL
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,
           CASE WHEN m->>'Etapa'='consulta' THEN ARRAY['Etapa','Solicitud']
                ELSE ARRAY['Etapa','Solicitud','ReciboBolsa'] END) IS NOT TRUE
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m->'Solicitud',
           ARRAY['ClaveIdempotencia','OrganizacionRef','ExpedienteRef','ResolucionRef','IntencionRef']) IS NOT TRUE THEN
        RAISE EXCEPTION 'campos de continuación inválidos' USING ERRCODE='P0600';
    END IF;
    IF (SELECT count(*) FROM json_each(p_material::json))<>(CASE WHEN m->>'Etapa'='consulta' THEN 2 ELSE 3 END)
       OR (SELECT count(*) FROM json_each((p_material::json)->'Solicitud'))<>5 THEN
        RAISE EXCEPTION 'claves de continuación repetidas' USING ERRCODE='P0600';
    END IF;
    s:=m->'Solicitud'; b:=m->'ReciboBolsa';
    IF jsonb_typeof(s->'ClaveIdempotencia') IS DISTINCT FROM 'string'
       OR (s->>'ClaveIdempotencia')!~'^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
       OR s->>'ClaveIdempotencia'='00000000-0000-4000-8000-000000000000' THEN
        RAISE EXCEPTION 'clave de continuación inválida' USING ERRCODE='P0600';
    END IF;
    FOREACH v_campo IN ARRAY ARRAY['OrganizacionRef','ExpedienteRef','ResolucionRef','IntencionRef'] LOOP
        IF jsonb_typeof(s->v_campo) IS DISTINCT FROM 'string'
           OR (s->>v_campo)!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
            RAISE EXCEPTION 'referencia de continuación inválida' USING ERRCODE='P0600';
        END IF;
    END LOOP;
    IF m->>'Etapa'='confirmacion' THEN
        IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(b,ARRAY[
            'IntencionRef','TerminalOperacionRef','OperacionRef','LlamamientoRef','PropuestaRef',
            'ReciboRef','AuditoriaRef','EventoRef','RegistroSHA256','ConfirmadaEn']) IS NOT TRUE THEN
            RAISE EXCEPTION 'recibo Bolsa incompleto' USING ERRCODE='P0600';
        END IF;
        IF (SELECT count(*) FROM json_each((p_material::json)->'ReciboBolsa'))<>10 THEN
            RAISE EXCEPTION 'recibo Bolsa con claves repetidas' USING ERRCODE='P0600';
        END IF;
        FOREACH v_campo IN ARRAY ARRAY['IntencionRef','TerminalOperacionRef','OperacionRef','LlamamientoRef',
            'PropuestaRef','ReciboRef','AuditoriaRef','EventoRef'] LOOP
            IF jsonb_typeof(b->v_campo) IS DISTINCT FROM 'string'
               OR (b->>v_campo)!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
                RAISE EXCEPTION 'referencia de recibo Bolsa inválida' USING ERRCODE='P0600';
            END IF;
        END LOOP;
        IF b->'IntencionRef' IS DISTINCT FROM s->'IntencionRef'
           OR b->>'OperacionRef'=b->>'TerminalOperacionRef'
           OR jsonb_typeof(b->'RegistroSHA256') IS DISTINCT FROM 'string'
           OR (b->>'RegistroSHA256')!~'^[0-9a-f]{64}$' OR b->>'RegistroSHA256'=repeat('0',64)
           OR jsonb_typeof(b->'ConfirmadaEn') IS DISTINCT FROM 'string'
           OR (b->>'ConfirmadaEn')!~'^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$' THEN
            RAISE EXCEPTION 'recibo Bolsa divergente' USING ERRCODE='P0600';
        END IF;
        BEGIN v_fecha_bolsa:=(b->>'ConfirmadaEn')::timestamptz;
        EXCEPTION WHEN data_exception THEN
            RAISE EXCEPTION 'fecha Bolsa inválida' USING ERRCODE='P0600';
        END;
        IF NOT isfinite(v_fecha_bolsa) OR v_fecha_bolsa='0001-01-01T00:00:00Z'::timestamptz THEN
            RAISE EXCEPTION 'fecha Bolsa inválida' USING ERRCODE='P0600';
        END IF;
    END IF;
    v_hash:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
    v_contexto_hash:=encode(sha256(convert_to(
        '{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
        '"},"atributos":{"material_sha256":"'||v_hash||'"}}','UTF8')),'hex');
    IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
        RAISE EXCEPTION 'decisión de continuación inválida' USING ERRCODE='P0603';
    END IF;
    BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'decisión de continuación inválida' USING ERRCODE='P0603';
    END;
    IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.llamamiento.siguiente.continuar'
       OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'continuacion_llamamiento_ct'
       OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
       OR d->>'recurso_ref' IS DISTINCT FROM s->>'ExpedienteRef'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_hash THEN
        RAISE EXCEPTION 'permiso de continuación divergente' USING ERRCODE='P0603';
    END IF;
    BEGIN
        SELECT * INTO STRICT v_consumo
          FROM vec_autorizacion_atestada_v3.registrar_y_consumir_continuacion_ct_v3_atestada(
            p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
            p_payload,p_sobre,p_evidencia,p_raiz);
    EXCEPTION WHEN insufficient_privilege OR data_exception OR SQLSTATE 'P0603' THEN
        RAISE EXCEPTION 'consumo de continuación denegado' USING ERRCODE='P0603';
    END;
    IF v_consumo.consumo_nuevo IS NOT TRUE
       OR v_consumo.efecto_ref IS DISTINCT FROM s->>'ExpedienteRef'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_hash THEN
        RAISE EXCEPTION 'continuación requiere consumo nuevo ligado' USING ERRCODE='P0603';
    END IF;

    -- El consumo fresco precede a toda lectura, incluida recuperación.
    SELECT * INTO v_fila FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
     WHERE resolucion_ref=s->>'ResolucionRef' AND organizacion_ref=s->>'OrganizacionRef'
       AND expediente_ref=s->>'ExpedienteRef' AND estado='confirmado'
       AND solicitud_json->>'Respuesta'='renuncia' AND revision_respuesta_rrhh AND revision_plazo_rrhh
       AND politica_ref='politica:ct:revision-manual-sintetica:20260906' AND politica_version=1
       AND politica_sha256='ea41d65808044fa75b597855e81a469ed274403a521890bafa07c33ae89ec2e3'
       AND estado_plazo='vigente' AND version_resultante=3
       AND comando_siguiente_ref IS NOT NULL
       AND comando_siguiente_json->>'intencion_ref'=s->>'IntencionRef'
       AND recibo_json->>'Estado'='confirmado'
       AND recibo_json->'IntencionSiguiente'->>'Estado'='pendiente'
       AND recibo_json->'Solicitud'=solicitud_json
     FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'antecedente de continuación no disponible' USING ERRCODE='P0602';
    END IF;
    IF v_fila.continuacion_clave IS NOT NULL
       AND v_fila.continuacion_clave::text IS DISTINCT FROM s->>'ClaveIdempotencia' THEN
        RAISE EXCEPTION 'intención ya confirmada con otra clave' USING ERRCODE='P0601';
    END IF;
    IF m->>'Etapa'='consulta' THEN
        RETURN jsonb_build_object('Resolucion',v_fila.recibo_json,
            'ComandoSiguienteRef',v_fila.comando_siguiente_ref,'ComandoSiguiente',v_fila.comando_siguiente_json);
    END IF;
    IF b->>'LlamamientoRef'=v_fila.llamamiento_ref OR v_fecha_bolsa<v_fila.resuelta_en THEN
        RAISE EXCEPTION 'recibo Bolsa no corresponde a continuación' USING ERRCODE='P0602';
    END IF;
    IF v_fila.continuacion_clave IS NOT NULL THEN
        IF v_fila.continuacion_material IS DISTINCT FROM p_material
           OR v_fila.continuacion_material_sha256 IS DISTINCT FROM v_hash
           OR v_fila.continuacion_actor_ref IS DISTINCT FROM d->>'principal_id'
           OR v_fila.continuacion_perfil_ref IS DISTINCT FROM d->>'perfil_activo_ref' THEN
            RAISE EXCEPTION 'continuación confirmada divergente' USING ERRCODE='P0601';
        END IF;
        RETURN v_fila.continuacion_recibo || jsonb_build_object('Estado','replay_confirmado');
    END IF;
    v_ahora:=clock_timestamp();
    IF v_fecha_bolsa>v_ahora THEN
        RAISE EXCEPTION 'recibo Bolsa posterior al registro' USING ERRCODE='P0600';
    END IF;
    v_resultado:=jsonb_build_object('Solicitud',s,'LlamamientoAnteriorRef',v_fila.llamamiento_ref,
        'ReciboBolsa',b,'ReciboRef','recibo:'||gen_random_uuid()::text,'AuditoriaRef',v_consumo.auditoria_ref,
        'ConfirmadaEn',to_char(v_ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'Estado','confirmado');
    UPDATE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh SET
        continuacion_clave=(s->>'ClaveIdempotencia')::uuid,continuacion_material=p_material,
        continuacion_material_sha256=v_hash,continuacion_recibo=v_resultado,
        continuacion_actor_ref=d->>'principal_id',continuacion_perfil_ref=d->>'perfil_activo_ref',
        continuacion_consumo=jsonb_build_object('DecisionRef',v_consumo.decision_ref,
            'ConsumoSHA256',v_consumo.consumo_huella_sha256,
            'EvidenciaSHA256',encode(sha256(p_evidencia),'hex'))
     WHERE resolucion_ref=v_fila.resolucion_ref AND continuacion_clave IS NULL;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'continuación ya ocupada' USING ERRCODE='P0601';
    END IF;
    RETURN v_resultado;
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.continuar_llamamiento_rrhh_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC,vec_contratacion_temporal_migrador;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.continuar_llamamiento_rrhh_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_ejecutor;
COMMIT;
