\set ON_ERROR_STOP on
-- Revisión manual exclusivamente del ejercicio sintético de desarrollo.
-- «vigente» registra la declaración positiva de RRHH bajo su política exacta:
-- no calcula un plazo legal, acredita entrega ni confirma el terminal Bolsa.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000058_resolucion_manual_respuesta_rrhh',0));

DO $dependencias$
BEGIN
    IF to_regprocedure('vec_contratacion_temporal.consultar_justificante_respuesta_recibida_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
           WHERE n.nspname='vec_autorizacion_atestada_v3'
             AND p.proname='registrar_y_consumir_resolucion_manual_ct_v3_atestada'
             AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef) THEN
        RAISE EXCEPTION 'faltan consulta de justificante o consumidor de resolución manual' USING ERRCODE='55000';
    END IF;
END
$dependencias$;

-- Una sola operación local inmutable: contiene también su asiento de
-- trazabilidad, evaluación manual y recibo. No duplica la declaración CT56,
-- ni el estado vivo de comunicación/Bolsa. No hay intención de siguiente.
CREATE TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh (
    resolucion_ref text PRIMARY KEY,
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    llamamiento_ref text NOT NULL,
    comunicacion_ref text NOT NULL REFERENCES vec_contratacion_temporal.comunicacion_llamamiento_local,
    justificante_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.respuesta_recibida_rrhh,
    seleccion_clave uuid NOT NULL REFERENCES vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6,
    clave_idempotencia uuid NOT NULL,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    revision_respuesta_rrhh boolean NOT NULL CHECK (revision_respuesta_rrhh),
    revision_plazo_rrhh boolean NOT NULL CHECK (revision_plazo_rrhh),
    politica_ref text NOT NULL CHECK (politica_ref='politica:ct:revision-manual-sintetica:20260906'),
    politica_version numeric(20,0) NOT NULL CHECK (politica_version=1),
    -- SHA256 UTF8 del raw JSON politicaRevisionManualDesarrollo, sin salto final.
    politica_sha256 text NOT NULL CHECK (politica_sha256='ea41d65808044fa75b597855e81a469ed274403a521890bafa07c33ae89ec2e3'),
    evaluacion_plazo_ref text NOT NULL UNIQUE,
    estado_plazo text NOT NULL CHECK (estado_plazo='vigente'),
    version_resultante numeric(20,0) NOT NULL CHECK (version_resultante=3),
    material text NOT NULL CHECK (octet_length(material) BETWEEN 1 AND 16384),
    material_json jsonb NOT NULL CHECK (jsonb_typeof(material_json)='object' AND material_json=material::jsonb),
    material_huella_sha256 text NOT NULL CHECK (material_huella_sha256=encode(sha256(convert_to(material,'UTF8')),'hex')),
    solicitud_json jsonb NOT NULL CHECK (jsonb_typeof(solicitud_json)='object' AND solicitud_json=material_json->'Solicitud'),
    auditoria_ref text NOT NULL UNIQUE,
    decision_ref text NOT NULL UNIQUE,
    consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
    evidencia_huella_sha256 text NOT NULL CHECK (evidencia_huella_sha256 ~ '^[0-9a-f]{64}$'),
    recibo_ref text NOT NULL UNIQUE,
    recibo_json jsonb NOT NULL CHECK (jsonb_typeof(recibo_json)='object'),
    estado text NOT NULL CHECK (estado='confirmado'),
    resuelta_en timestamptz(6) NOT NULL CHECK (isfinite(resuelta_en) AND resuelta_en<>'0001-01-01T00:00:00Z'::timestamptz),
    UNIQUE (organizacion_ref,clave_idempotencia),
    UNIQUE (organizacion_ref,comunicacion_ref)
);
COMMENT ON TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh IS
    'Resolución local append-only del ejercicio sintético; revisión expresa de respuesta y plazo del ejercicio, no vigencia legal ni cierre/aceptación en Bolsa. Recibo y auditoría originales en la misma fila.';
ALTER TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario ON vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
    TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true);
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE
    ON vec_contratacion_temporal.resolucion_manual_respuesta_rrhh FOR EACH ROW
    EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
REVOKE ALL ON TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;

CREATE FUNCTION vec_contratacion_temporal.registrar_resolucion_manual_respuesta_rrhh_v1(
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
    m jsonb; s jsonb; p jsonb; d jsonb; v_referencia text;
    v_material_huella text; v_contexto_huella text; v_consumo record;
    v_declaracion vec_contratacion_temporal.respuesta_recibida_rrhh%ROWTYPE;
    v_previa vec_contratacion_temporal.resolucion_manual_respuesta_rrhh%ROWTYPE;
    v_ahora timestamptz(6); v_resolucion text; v_recibo text; v_evaluacion text;
    v_resultado jsonb;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario'
       OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
       OR current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'off' THEN
        RAISE EXCEPTION 'resolución manual denegada' USING ERRCODE='P0583';
    END IF;
    IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 16384 THEN
        RAISE EXCEPTION 'material de resolución manual inválido' USING ERRCODE='P0580';
    END IF;
    BEGIN
        m:=p_material::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'JSON de resolución manual inválido' USING ERRCODE='P0580';
    END;
    IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY['Solicitud','Politica']) IS NOT TRUE
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m->'Solicitud',ARRAY[
           'ClaveIdempotencia','OrganizacionRef','ExpedienteRef','LlamamientoRef',
           'ComunicacionRef','VersionEsperada','Respuesta','PruebaRespuestaRef',
           'RevisionRespuestaRRHH','RevisionPlazoRRHH','CriterioValidacionRef']) IS NOT TRUE
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m->'Politica',ARRAY[
           'Referencia','Version','HuellaSHA256']) IS NOT TRUE THEN
        RAISE EXCEPTION 'campos de resolución manual inválidos' USING ERRCODE='P0580';
    END IF;
    -- json.Marshal del material completo, con los once campos de Solicitud.
    -- La huella usa bytes originales, sin normalizar/reserializar el JSON.
    IF (SELECT count(*) FROM json_each(p_material::json))<>2
       OR (SELECT count(*) FROM json_each((p_material::json)->'Solicitud'))<>11
       OR (SELECT count(*) FROM json_each((p_material::json)->'Politica'))<>3 THEN
        RAISE EXCEPTION 'claves repetidas en resolución manual' USING ERRCODE='P0580';
    END IF;
    s:=m->'Solicitud'; p:=m->'Politica';
    IF jsonb_typeof(s->'ClaveIdempotencia') IS DISTINCT FROM 'string'
       OR (s->>'ClaveIdempotencia')!~'^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
       OR s->>'ClaveIdempotencia'='00000000-0000-4000-8000-000000000000'
       OR s->'Respuesta' IS DISTINCT FROM '"aceptacion"'::jsonb
       OR s->'RevisionRespuestaRRHH' IS DISTINCT FROM 'true'::jsonb
       OR s->'RevisionPlazoRRHH' IS DISTINCT FROM 'true'::jsonb THEN
        RAISE EXCEPTION 'se requiere revisión manual positiva de aceptación' USING ERRCODE='P0580';
    END IF;
    FOREACH v_referencia IN ARRAY ARRAY[
        'OrganizacionRef','ExpedienteRef','LlamamientoRef','ComunicacionRef','PruebaRespuestaRef'
    ] LOOP
        IF jsonb_typeof(s->v_referencia) IS DISTINCT FROM 'string'
           OR (s->>v_referencia)!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
            RAISE EXCEPTION 'referencia de resolución manual inválida' USING ERRCODE='P0580';
        END IF;
    END LOOP;
    IF jsonb_typeof(s->'VersionEsperada') IS DISTINCT FROM 'number'
       OR s->>'VersionEsperada' IS DISTINCT FROM '2' THEN
        RAISE EXCEPTION 'versión de resolución manual incompatible' USING ERRCODE='P0582';
    END IF;
    IF s->'CriterioValidacionRef' IS DISTINCT FROM '"politica:ct:revision-manual-sintetica:20260906"'::jsonb
       OR p->'Referencia' IS DISTINCT FROM s->'CriterioValidacionRef'
       OR jsonb_typeof(p->'Version') IS DISTINCT FROM 'number'
       OR p->>'Version' IS DISTINCT FROM '1'
       OR p->'HuellaSHA256' IS DISTINCT FROM '"ea41d65808044fa75b597855e81a469ed274403a521890bafa07c33ae89ec2e3"'::jsonb THEN
        RAISE EXCEPTION 'política de ejercicio sintético incompatible' USING ERRCODE='P0580';
    END IF;
    v_material_huella:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
    v_contexto_huella:=encode(sha256(convert_to(
        '{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
        '"},"atributos":{"material_sha256":"'||v_material_huella||'"}}','UTF8')),'hex');
    IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
        RAISE EXCEPTION 'decisión de resolución manual inválida' USING ERRCODE='P0583';
    END IF;
    BEGIN
        d:=convert_from(p_decision,'UTF8')::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'decisión de resolución manual inválida' USING ERRCODE='P0583';
    END;
    IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.llamamiento.respuesta.validacion_manual.registrar'
       OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'resolucion_manual_respuesta_ct'
       OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
       OR d->>'recurso_ref' IS DISTINCT FROM s->>'ExpedienteRef'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'autorización de resolución manual divergente' USING ERRCODE='P0583';
    END IF;
    -- Consumo nuevo ANTES de cualquier lectura o replay. El actor y perfil
    -- provienen solo de la decisión atestada; nunca del material del navegador.
    BEGIN
        SELECT * INTO STRICT v_consumo
          FROM vec_autorizacion_atestada_v3.registrar_y_consumir_resolucion_manual_ct_v3_atestada(
            p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,
            p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    EXCEPTION
        WHEN insufficient_privilege OR data_exception OR SQLSTATE 'P0583' THEN
            RAISE EXCEPTION 'consumo de resolución manual denegado' USING ERRCODE='P0583';
    END;
    IF v_consumo.consumo_nuevo IS NOT TRUE
       OR v_consumo.efecto_ref IS DISTINCT FROM s->>'ExpedienteRef'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION 'resolución manual requiere consumo nuevo ligado al material' USING ERRCODE='P0583';
    END IF;

    -- Solo tablas propias CT. La declaración, el aviso local y la selección
    -- original deben conservar todas sus coordenadas y la propuesta confirmada.
    SELECT r.* INTO v_declaracion
      FROM vec_contratacion_temporal.respuesta_recibida_rrhh r
      JOIN vec_contratacion_temporal.comunicacion_llamamiento_local c
        ON c.comunicacion_ref=r.comunicacion_ref
      JOIN vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
        ON e.clave_idempotencia=r.seleccion_clave AND e.clave_idempotencia=c.seleccion_clave
     WHERE r.justificante_ref=s->>'PruebaRespuestaRef'
       AND r.organizacion_ref=s->>'OrganizacionRef'
       AND r.expediente_ref=s->>'ExpedienteRef'
       AND r.llamamiento_ref=s->>'LlamamientoRef'
       AND r.comunicacion_ref=s->>'ComunicacionRef'
       AND r.respuesta='aceptacion' AND r.version_comunicacion=2
       AND r.estado='registrada_por_rrhh'
       AND c.organizacion_ref=r.organizacion_ref AND c.expediente_ref=r.expediente_ref
       AND c.llamamiento_ref=r.llamamiento_ref
       AND c.version_resultante=2 AND c.estado='registrada_localmente'
       AND e.situacion='confirmada'
       AND e.solicitud_json->>'organizacion_ref'=r.organizacion_ref
       AND e.solicitud_json->>'expediente_ref'=r.expediente_ref
       AND e.recibo_json->>'organizacion_ref'=r.organizacion_ref
       AND e.recibo_json->>'expediente_ref'=r.expediente_ref
       AND e.recibo_json->>'llamamiento_ref'=r.llamamiento_ref
       AND e.recibo_json->'propuesta_generada'='true'::jsonb
       AND e.recibo_json->>'recibo_ref'=c.material_json->'solicitud'->>'PruebaEntregaRef'
       AND r.recibo_json->>'JustificanteRef'=r.justificante_ref
       AND r.recibo_json->>'ReciboRef'=r.recibo_ref
       AND r.recibo_json->'Solicitud'=r.material_json
       AND r.recibo_json->>'Estado'='registrada_por_rrhh'
     FOR SHARE OF r,c,e;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'justificante o antecedentes incompatibles' USING ERRCODE='P0582';
    END IF;
    SELECT * INTO v_previa FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
     WHERE organizacion_ref=s->>'OrganizacionRef'
       AND clave_idempotencia=(s->>'ClaveIdempotencia')::uuid;
    IF FOUND THEN
        IF v_previa.actor_ref IS DISTINCT FROM d->>'principal_id'
           OR v_previa.perfil_ref IS DISTINCT FROM d->>'perfil_activo_ref' THEN
            RAISE EXCEPTION 'replay de resolución manual denegado' USING ERRCODE='P0583';
        END IF;
        IF v_previa.material IS DISTINCT FROM p_material OR v_previa.material_json IS DISTINCT FROM m THEN
            RAISE EXCEPTION 'clave de resolución manual divergente' USING ERRCODE='P0581';
        END IF;
        RETURN v_previa.recibo_json || jsonb_build_object('Estado','replay_confirmado');
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
        WHERE organizacion_ref=s->>'OrganizacionRef' AND comunicacion_ref=s->>'ComunicacionRef') THEN
        RAISE EXCEPTION 'comunicación con resolución manual previa' USING ERRCODE='P0582';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF v_ahora<v_declaracion.registrada_en THEN
        RAISE EXCEPTION 'reloj anterior al justificante registrado' USING ERRCODE='P0584';
    END IF;
    v_resolucion:='resolucion:'||gen_random_uuid()::text;
    v_recibo:='recibo:'||gen_random_uuid()::text;
    v_evaluacion:='evaluacion:'||gen_random_uuid()::text;
    v_resultado:=jsonb_build_object(
        'Solicitud',s,'Politica',p,'EvaluacionPlazoRef',v_evaluacion,'EstadoPlazo','vigente',
        'ResolucionRef',v_resolucion,'ReciboLocalRef',v_recibo,'AuditoriaRef',v_consumo.auditoria_ref,
        'IntencionSiguiente','{}'::jsonb,'VersionResultante',3,
        'ResueltaEn',to_char(v_ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'Estado','confirmado');
    INSERT INTO vec_contratacion_temporal.resolucion_manual_respuesta_rrhh (
        resolucion_ref,organizacion_ref,expediente_ref,llamamiento_ref,comunicacion_ref,
        justificante_ref,seleccion_clave,clave_idempotencia,actor_ref,perfil_ref,
        revision_respuesta_rrhh,revision_plazo_rrhh,politica_ref,politica_version,politica_sha256,
        evaluacion_plazo_ref,estado_plazo,version_resultante,material,material_json,
        material_huella_sha256,solicitud_json,auditoria_ref,decision_ref,
        consumo_huella_sha256,evidencia_huella_sha256,recibo_ref,recibo_json,estado,resuelta_en
    ) VALUES (
        v_resolucion,s->>'OrganizacionRef',s->>'ExpedienteRef',s->>'LlamamientoRef',s->>'ComunicacionRef',
        v_declaracion.justificante_ref,v_declaracion.seleccion_clave,(s->>'ClaveIdempotencia')::uuid,
        d->>'principal_id',d->>'perfil_activo_ref',true,true,p->>'Referencia',1,p->>'HuellaSHA256',
        v_evaluacion,'vigente',3,p_material,m,v_material_huella,s,v_consumo.auditoria_ref,
        v_consumo.decision_ref,v_consumo.consumo_huella_sha256,encode(sha256(p_evidencia),'hex'),
        v_recibo,v_resultado,'confirmado',v_ahora);
    RETURN v_resultado;
EXCEPTION
    WHEN unique_violation THEN
        RAISE EXCEPTION 'conflicto de resolución manual' USING ERRCODE='P0581';
    WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN
        RAISE EXCEPTION 'resolución manual no disponible' USING ERRCODE='P0584';
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_resolucion_manual_respuesta_rrhh_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_resolucion_manual_respuesta_rrhh_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_ejecutor;
COMMIT;
