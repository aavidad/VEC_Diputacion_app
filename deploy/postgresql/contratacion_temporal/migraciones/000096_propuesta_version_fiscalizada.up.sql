\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000096',0));
-- Propuesta tras subsanación: conserva función CT65 y todas las filas previas.
DO $preflight$
DECLARE v_pre text; v_post text;
BEGIN
    IF to_regprocedure('vec_contratacion_temporal.registrar_propuesta_formalizacion_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR NOT EXISTS (SELECT 1 FROM pg_proc p
           WHERE p.oid=to_regprocedure('vec_contratacion_temporal.registrar_propuesta_formalizacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
             AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef
             AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='f3e681486de901b464497af49737e72487329f5014322857c8845f5f782135bf') THEN
        RAISE EXCEPTION 'propuesta anterior incompatible o migración ya instalada' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_constraintdef(oid) INTO STRICT v_pre FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.propuesta_formalizacion'::regclass
       AND conname='propuesta_formalizacion_version_previa_check' AND contype='c' AND convalidated;
    SELECT pg_get_constraintdef(oid) INTO STRICT v_post FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.propuesta_formalizacion'::regclass
       AND conname='propuesta_formalizacion_version_resultante_check' AND contype='c' AND convalidated;
    IF v_pre IS DISTINCT FROM 'CHECK ((version_previa = (6)::numeric))'
       OR v_post IS DISTINCT FROM 'CHECK ((version_resultante = (7)::numeric))' THEN
        RAISE EXCEPTION 'restricciones de propuesta incompatibles' USING ERRCODE='55000';
    END IF;
END
$preflight$;
ALTER TABLE vec_contratacion_temporal.propuesta_formalizacion
    DROP CONSTRAINT propuesta_formalizacion_version_previa_check,
    DROP CONSTRAINT propuesta_formalizacion_version_resultante_check;
ALTER TABLE vec_contratacion_temporal.propuesta_formalizacion
    ADD CONSTRAINT propuesta_formalizacion_version_previa_check CHECK (version_previa BETWEEN 6 AND 9007199254740990),
    ADD CONSTRAINT propuesta_formalizacion_version_resultante_check CHECK (version_resultante=version_previa+1);

CREATE FUNCTION vec_contratacion_temporal.registrar_propuesta_formalizacion_v2(
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
    v_n bigint; v_siguiente bigint;
    m jsonb; s jsonb; b jsonb; d jsonb; v_campo text; v_snapshot jsonb;
    v_hash text; v_contexto_hash text; v_consumo record;
    v_resolucion vec_contratacion_temporal.resolucion_manual_respuesta_rrhh%ROWTYPE;
    v_previa vec_contratacion_temporal.propuesta_formalizacion%ROWTYPE;
    v_actual vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
    v_respuesta jsonb; v_seleccion jsonb; v_resultado jsonb; v_continuacion jsonb;
    v_ahora timestamptz(6); v_fecha_bolsa timestamptz(6);
    v_propuesta text; v_recibo text; v_evento text; v_unidad text;
    v_actuacion jsonb; v_agregado jsonb; v_agregado_hash text; v_prueba bytea;
    v_secuencia numeric(20,0); v_anterior text; v_payload bytea; v_outbox_hash text;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
       OR current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'off' THEN
        RAISE EXCEPTION 'propuesta denegada' USING ERRCODE='P0613';
    END IF;
    IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN
        RAISE EXCEPTION 'material de propuesta inválido' USING ERRCODE='P0610';
    END IF;
    BEGIN m:=p_material::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'JSON de propuesta inválido' USING ERRCODE='P0610';
    END;
    IF m->'Etapa' IS NULL OR (m->'Etapa') NOT IN ('"consulta"'::jsonb,'"confirmacion"'::jsonb)
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,
           CASE WHEN m->>'Etapa'='consulta' THEN ARRAY['Etapa','Solicitud']
                ELSE ARRAY['Etapa','Solicitud','AceptacionBolsa'] END) IS NOT TRUE
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m->'Solicitud',ARRAY[
           'ClaveIdempotencia','OrganizacionRef','ExpedienteRef','LlamamientoRef',
           'ResolucionLlamamientoAceptadaRef','ReciboResolucionAceptadaRef','VersionEsperada',
           'TipoFormalizacion','Plantilla','Anexos','PoliticaFirma','PlanFirma']) IS NOT TRUE THEN
        RAISE EXCEPTION 'campos de propuesta inválidos' USING ERRCODE='P0610';
    END IF;
    IF (SELECT count(*) FROM json_each(p_material::json))<>(CASE WHEN m->>'Etapa'='consulta' THEN 2 ELSE 3 END)
       OR (SELECT count(*) FROM json_each((p_material::json)->'Solicitud'))<>12 THEN
        RAISE EXCEPTION 'claves de propuesta repetidas' USING ERRCODE='P0610';
    END IF;
    s:=m->'Solicitud'; b:=m->'AceptacionBolsa';
    IF jsonb_typeof(s->'ClaveIdempotencia') IS DISTINCT FROM 'string'
       OR (s->>'ClaveIdempotencia')!~'^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
       OR s->>'ClaveIdempotencia'='00000000-0000-4000-8000-000000000000'
       OR jsonb_typeof(s->'VersionEsperada') IS DISTINCT FROM 'number'
       OR coalesce(s->>'VersionEsperada','') !~ '^[1-9][0-9]{0,15}$'
       OR (s->'Anexos') NOT IN ('null'::jsonb,'[]'::jsonb) THEN
        RAISE EXCEPTION 'solicitud de propuesta inválida' USING ERRCODE='P0610';
    END IF;
    v_n:=(s->>'VersionEsperada')::bigint;
    IF v_n NOT BETWEEN 7 AND 9007199254740990 THEN
        RAISE EXCEPTION 'versión de propuesta inválida' USING ERRCODE='P0610';
    END IF;
    v_siguiente:=v_n+1;
    FOREACH v_campo IN ARRAY ARRAY['OrganizacionRef','ExpedienteRef','LlamamientoRef',
        'ResolucionLlamamientoAceptadaRef','ReciboResolucionAceptadaRef'] LOOP
        IF jsonb_typeof(s->v_campo) IS DISTINCT FROM 'string'
           OR (s->>v_campo)!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
            RAISE EXCEPTION 'referencia de propuesta inválida' USING ERRCODE='P0610';
        END IF;
    END LOOP;
    FOREACH v_campo IN ARRAY ARRAY['TipoFormalizacion','Plantilla','PoliticaFirma','PlanFirma'] LOOP
        v_snapshot:=s->v_campo;
        IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(v_snapshot,ARRAY['Referencia','Version','HuellaSHA256']) IS NOT TRUE THEN
            RAISE EXCEPTION 'publicación de propuesta inválida' USING ERRCODE='P0610';
        END IF;
        IF (SELECT count(*) FROM json_each(((p_material::json)->'Solicitud')->v_campo))<>3
           OR jsonb_typeof(v_snapshot->'Referencia') IS DISTINCT FROM 'string'
           OR (v_snapshot->>'Referencia')!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR v_snapshot->'Version' IS DISTINCT FROM '1'::jsonb
           OR jsonb_typeof(v_snapshot->'HuellaSHA256') IS DISTINCT FROM 'string'
           OR (v_snapshot->>'HuellaSHA256')!~'^[0-9a-f]{64}$'
           OR v_snapshot->>'HuellaSHA256'=repeat('0',64) THEN
            RAISE EXCEPTION 'compromiso de publicación inválido' USING ERRCODE='P0610';
        END IF;
    END LOOP;
    IF m->>'Etapa'='confirmacion' THEN
        IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(b,ARRAY[
            'OperacionRef','AperturaOperacionRef','LlamamientoRef','JustificanteRef',
            'EvaluacionPlazoRef','Politica','RegistroSHA256','ResueltaEn']) IS NOT TRUE
           OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(b->'Politica',ARRAY['Referencia','Version','HuellaSHA256']) IS NOT TRUE THEN
            RAISE EXCEPTION 'evidencia de aceptación incompleta' USING ERRCODE='P0610';
        END IF;
        IF (SELECT count(*) FROM json_each((p_material::json)->'AceptacionBolsa'))<>8
           OR (SELECT count(*) FROM json_each(((p_material::json)->'AceptacionBolsa')->'Politica'))<>3 THEN
            RAISE EXCEPTION 'evidencia con claves repetidas' USING ERRCODE='P0610';
        END IF;
        FOREACH v_campo IN ARRAY ARRAY['OperacionRef','AperturaOperacionRef','LlamamientoRef','JustificanteRef','EvaluacionPlazoRef'] LOOP
            IF jsonb_typeof(b->v_campo) IS DISTINCT FROM 'string'
               OR (b->>v_campo)!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
                RAISE EXCEPTION 'referencia de aceptación inválida' USING ERRCODE='P0610';
            END IF;
        END LOOP;
        IF b->>'OperacionRef'=b->>'AperturaOperacionRef'
           OR b->'LlamamientoRef' IS DISTINCT FROM s->'LlamamientoRef'
           OR jsonb_typeof(b->'RegistroSHA256') IS DISTINCT FROM 'string'
           OR (b->>'RegistroSHA256')!~'^[0-9a-f]{64}$' OR b->>'RegistroSHA256'=repeat('0',64)
           OR jsonb_typeof(b->'ResueltaEn') IS DISTINCT FROM 'string'
           OR (b->>'ResueltaEn')!~'^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$' THEN
            RAISE EXCEPTION 'evidencia de aceptación inválida' USING ERRCODE='P0610';
        END IF;
        BEGIN v_fecha_bolsa:=(b->>'ResueltaEn')::timestamptz;
        EXCEPTION WHEN data_exception THEN
            RAISE EXCEPTION 'fecha de aceptación inválida' USING ERRCODE='P0610';
        END;
        IF NOT isfinite(v_fecha_bolsa) OR v_fecha_bolsa='0001-01-01T00:00:00Z'::timestamptz THEN
            RAISE EXCEPTION 'fecha de aceptación inválida' USING ERRCODE='P0610';
        END IF;
    END IF;
    v_hash:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
    v_contexto_hash:=encode(sha256(convert_to(
        '{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
        '"},"atributos":{"material_sha256":"'||v_hash||'"}}','UTF8')),'hex');
    IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
        RAISE EXCEPTION 'decisión de propuesta inválida' USING ERRCODE='P0613';
    END IF;
    BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'decisión de propuesta inválida' USING ERRCODE='P0613';
    END;
    IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.formalizacion.propuesta.registrar'
       OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'propuesta_formalizacion_ct'
       OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
       OR d->>'recurso_ref' IS DISTINCT FROM s->>'ExpedienteRef'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_hash THEN
        RAISE EXCEPTION 'permiso de propuesta divergente' USING ERRCODE='P0613';
    END IF;
    BEGIN
        SELECT * INTO STRICT v_consumo
          FROM vec_autorizacion_atestada_v3.registrar_y_consumir_propuesta_formalizacion_ct_v3_atestada(
            p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
            p_payload,p_sobre,p_evidencia,p_raiz);
    EXCEPTION WHEN insufficient_privilege OR data_exception OR SQLSTATE 'P0613' THEN
        RAISE EXCEPTION 'consumo de propuesta denegado' USING ERRCODE='P0613';
    END;
    IF v_consumo.consumo_nuevo IS NOT TRUE
       OR v_consumo.efecto_ref IS DISTINCT FROM s->>'ExpedienteRef'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_hash THEN
        RAISE EXCEPTION 'propuesta requiere consumo nuevo ligado' USING ERRCODE='P0613';
    END IF;

    -- Primera lectura de negocio: permiso fresco ya consumido en esta tx.
    FOREACH v_campo IN ARRAY ARRAY['TipoFormalizacion','Plantilla','PoliticaFirma','PlanFirma'] LOOP
        IF NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.publicacion_propuesta_formalizacion_desarrollo p
            WHERE p.componente=v_campo AND p.referencia=s->v_campo->>'Referencia'
              AND to_jsonb(p.version)=s->v_campo->'Version' AND p.huella_sha256=s->v_campo->>'HuellaSHA256'
              AND p.huella_sha256=encode(sha256(convert_to(p.contenido,'UTF8')),'hex')) THEN
            RAISE EXCEPTION 'publicación de propuesta no acreditada' USING ERRCODE='P0610';
        END IF;
    END LOOP;
    SELECT * INTO v_resolucion FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh r
     WHERE r.resolucion_ref=s->>'ResolucionLlamamientoAceptadaRef' AND r.recibo_ref=s->>'ReciboResolucionAceptadaRef'
       AND r.organizacion_ref=s->>'OrganizacionRef' AND r.expediente_ref=s->>'ExpedienteRef'
       AND r.llamamiento_ref=s->>'LlamamientoRef' AND r.estado='confirmado'
       AND r.solicitud_json->>'Respuesta'='aceptacion' AND r.revision_respuesta_rrhh AND r.revision_plazo_rrhh
       AND r.politica_ref='politica:ct:revision-manual-sintetica:20260906' AND r.politica_version=1
       AND r.politica_sha256='ea41d65808044fa75b597855e81a469ed274403a521890bafa07c33ae89ec2e3'
       AND r.solicitud_json->>'CriterioValidacionRef'=r.politica_ref
       AND r.solicitud_json->'RevisionRespuestaRRHH'='true'::jsonb
       AND r.solicitud_json->'RevisionPlazoRRHH'='true'::jsonb
       AND r.estado_plazo='vigente' AND r.version_resultante=3
       AND r.recibo_json->>'Estado'='confirmado' AND r.recibo_json->'Solicitud'=r.solicitud_json
       AND r.recibo_json->>'ResolucionRef'=r.resolucion_ref AND r.recibo_json->>'ReciboLocalRef'=r.recibo_ref
       AND r.recibo_json->>'EvaluacionPlazoRef'=r.evaluacion_plazo_ref
       AND r.recibo_json->'Politica'=jsonb_build_object('Referencia',r.politica_ref,'Version',r.politica_version,'HuellaSHA256',r.politica_sha256)
     FOR SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'aceptación CT no acreditada' USING ERRCODE='P0615';
    END IF;
    SELECT j.recibo_json,e.recibo_json,continuacion.continuacion_recibo
      INTO v_respuesta,v_seleccion,v_continuacion
      FROM vec_contratacion_temporal.respuesta_recibida_rrhh j
      JOIN vec_contratacion_temporal.comunicacion_llamamiento_local c ON c.comunicacion_ref=j.comunicacion_ref
      JOIN vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
        ON e.clave_idempotencia=j.seleccion_clave AND e.clave_idempotencia=c.seleccion_clave
      -- CT65: confirmación original del primer sucesor, ligada al aviso exacto.
      LEFT JOIN vec_contratacion_temporal.resolucion_manual_respuesta_rrhh continuacion
        ON c.material_json->'solicitud'->>'TipoAntecedente'='continuacion_confirmada'
       AND continuacion.seleccion_clave=e.clave_idempotencia
       AND continuacion.organizacion_ref=j.organizacion_ref
       AND continuacion.expediente_ref=j.expediente_ref
       AND continuacion.llamamiento_ref=e.recibo_json->>'llamamiento_ref'
       AND continuacion.llamamiento_ref<>j.llamamiento_ref
       AND continuacion.estado='confirmado'
       AND continuacion.solicitud_json->>'Respuesta'='renuncia'
       AND continuacion.continuacion_clave IS NOT NULL
       AND continuacion.continuacion_recibo->>'Estado'='confirmado'
       AND continuacion.continuacion_recibo->>'ReciboRef'=c.material_json->'solicitud'->>'PruebaEntregaRef'
       AND continuacion.continuacion_recibo->'Solicitud'->>'OrganizacionRef'=j.organizacion_ref
       AND continuacion.continuacion_recibo->'Solicitud'->>'ExpedienteRef'=j.expediente_ref
       AND continuacion.continuacion_recibo->'Solicitud'->>'ResolucionRef'=continuacion.resolucion_ref
       AND continuacion.continuacion_recibo->'Solicitud'->>'ClaveIdempotencia'=continuacion.continuacion_clave::text
       AND continuacion.continuacion_recibo->>'LlamamientoAnteriorRef'=continuacion.llamamiento_ref
       AND continuacion.continuacion_recibo->'ReciboBolsa'->>'LlamamientoRef'=j.llamamiento_ref
     WHERE j.justificante_ref=v_resolucion.justificante_ref AND j.seleccion_clave=v_resolucion.seleccion_clave
       AND j.organizacion_ref=v_resolucion.organizacion_ref AND j.expediente_ref=v_resolucion.expediente_ref
       AND j.llamamiento_ref=v_resolucion.llamamiento_ref AND j.comunicacion_ref=v_resolucion.comunicacion_ref
       AND j.respuesta='aceptacion' AND j.version_comunicacion=2 AND j.estado='registrada_por_rrhh'
       AND c.organizacion_ref=j.organizacion_ref AND c.expediente_ref=j.expediente_ref AND c.llamamiento_ref=j.llamamiento_ref
       AND c.estado='registrada_localmente' AND c.version_resultante=2
       AND e.solicitud_json->'version_expediente'=to_jsonb(v_n)
       AND e.recibo_json->'version_expediente'=to_jsonb(v_n)
       AND e.situacion='confirmada' AND e.solicitud_json->>'organizacion_ref'=j.organizacion_ref
       AND e.solicitud_json->>'expediente_ref'=j.expediente_ref
       AND e.recibo_json->>'organizacion_ref'=j.organizacion_ref AND e.recibo_json->>'expediente_ref'=j.expediente_ref
       AND (
           (NOT ((c.material_json->'solicitud') ? 'TipoAntecedente')
            AND e.recibo_json->>'llamamiento_ref'=j.llamamiento_ref
            AND e.recibo_json->>'recibo_ref'=c.material_json->'solicitud'->>'PruebaEntregaRef')
           OR (c.material_json->'solicitud'->>'TipoAntecedente'='continuacion_confirmada'
               AND continuacion.resolucion_ref IS NOT NULL)
       )
       AND e.recibo_json->'propuesta_generada'='true'::jsonb
       AND j.recibo_json->>'JustificanteRef'=j.justificante_ref AND j.recibo_json->>'ReciboRef'=j.recibo_ref
       AND j.recibo_json->'Solicitud'=j.material_json AND j.recibo_json->>'Estado'='registrada_por_rrhh'
       AND j.registrada_en<=v_resolucion.resuelta_en
     FOR SHARE OF j,c,e;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'antecedentes de aceptación incompatibles' USING ERRCODE='P0615';
    END IF;
    IF m->>'Etapa'='consulta' THEN
        RETURN jsonb_build_object('Resolucion',v_resolucion.recibo_json,
            'Justificante',jsonb_build_object('Respuesta',v_respuesta,'Seleccion',v_seleccion)||
                CASE WHEN v_continuacion IS NULL THEN '{}'::jsonb
                     ELSE jsonb_build_object('Continuacion',v_continuacion) END,
            'SeleccionClave',v_resolucion.seleccion_clave::text);
    END IF;

    -- El terminal y SHA de Bolsa los acredita composición mediante su puerto
    -- propietario antes de emitir V3 sobre este material. SQL no lee Bolsa.
    IF b->>'AperturaOperacionRef' IS DISTINCT FROM
           (CASE WHEN v_continuacion IS NULL THEN v_seleccion->>'operacion_ref'
                ELSE v_continuacion->'ReciboBolsa'->>'OperacionRef' END)
       OR b->>'JustificanteRef' IS DISTINCT FROM v_resolucion.justificante_ref
       OR b->>'EvaluacionPlazoRef' IS DISTINCT FROM v_resolucion.evaluacion_plazo_ref
       OR b->'Politica' IS DISTINCT FROM v_resolucion.recibo_json->'Politica'
       OR v_fecha_bolsa<v_resolucion.resuelta_en OR v_fecha_bolsa>clock_timestamp() THEN
        RAISE EXCEPTION 'evidencia Bolsa desligada de aceptación' USING ERRCODE='P0615';
    END IF;
    -- Recuperación antes de exigir la versión N. Misma autoridad fresca, actor, perfil
    -- y material completo; no vuelve a tocar expediente, recibos o cadenas.
    SELECT * INTO v_previa FROM vec_contratacion_temporal.propuesta_formalizacion
     WHERE organizacion_ref=s->>'OrganizacionRef' AND clave_idempotencia=(s->>'ClaveIdempotencia')::uuid;
    IF FOUND THEN
        IF v_previa.actor_ref IS DISTINCT FROM d->>'principal_id'
           OR v_previa.perfil_ref IS DISTINCT FROM d->>'perfil_activo_ref' THEN
            RAISE EXCEPTION 'recuperación de propuesta denegada' USING ERRCODE='P0613';
        END IF;
        IF v_previa.material IS DISTINCT FROM p_material OR v_previa.material_json IS DISTINCT FROM m THEN
            RAISE EXCEPTION 'clave de propuesta divergente' USING ERRCODE='P0611';
        END IF;
        RETURN v_previa.recibo_json||jsonb_build_object('Estado','replay_confirmado');
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.propuesta_formalizacion
        WHERE resolucion_ref=v_resolucion.resolucion_ref OR
            (organizacion_ref=s->>'OrganizacionRef' AND expediente_ref=s->>'ExpedienteRef')) THEN
        RAISE EXCEPTION 'aceptación con propuesta previa' USING ERRCODE='P0611';
    END IF;
    SELECT v.* INTO v_actual FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version)
     WHERE a.expediente_ref=s->>'ExpedienteRef' FOR UPDATE OF a,v;
    IF NOT FOUND OR v_actual.version<>v_n OR v_actual.fase_clave<>'fiscalizacion' OR v_actual.estado<>'en_curso'
       OR v_actual.agregado_json->>'organizacion_ref' IS DISTINCT FROM s->>'OrganizacionRef'
       OR v_actual.agregado_json->>'referencia' IS DISTINCT FROM s->>'ExpedienteRef'
       OR v_actual.agregado_json->'version' IS DISTINCT FROM to_jsonb(v_n)
       OR v_actual.agregado_json->>'fase_actual' IS DISTINCT FROM 'fiscalizacion'
       OR v_actual.agregado_json->>'estado_actual' IS DISTINCT FROM 'en_curso'
       OR jsonb_typeof(v_actual.agregado_json->'actuaciones') IS DISTINCT FROM 'array'
       OR jsonb_array_length(v_actual.agregado_json->'actuaciones')<>v_n
       OR coalesce(v_actual.agregado_json#>>'{fiscalizacion,resultado}','') NOT IN ('favorable','favorable_con_observaciones')
       OR v_actual.agregado_json_huella_sha256<>encode(sha256(convert_to(v_actual.agregado_json::text,'UTF8')),'hex') THEN
        RAISE EXCEPTION 'versión de expediente incompatible' USING ERRCODE='P0612';
    END IF;
    v_unidad:=v_actual.agregado_json#>>'{asignacion,unidad_ref}';
    IF coalesce(v_unidad,'')!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
        RAISE EXCEPTION 'unidad de expediente no disponible' USING ERRCODE='P0614';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF v_ahora<v_fecha_bolsa OR v_ahora<v_actual.registrada_en THEN
        RAISE EXCEPTION 'reloj anterior a los antecedentes' USING ERRCODE='P0614';
    END IF;
    v_propuesta:='propuesta:'||gen_random_uuid()::text;
    v_recibo:='recibo:'||gen_random_uuid()::text;
    v_evento:='evento:'||gen_random_uuid()::text;
    v_actuacion:=jsonb_build_object(
        'secuencia',v_siguiente,'version_expediente',v_siguiente,'accion_clave','registrar_propuesta_formalizacion',
        'actor_ref',d->>'principal_id','unidad_ref',v_unidad,'recibo_ref',v_recibo,
        'realizada_en',to_char(v_ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'fase_origen',v_actual.agregado_json->'fase_actual','fase_destino','nombramiento',
        'estado_origen',v_actual.agregado_json->'estado_actual','estado_destino','en_curso');
    v_agregado:=v_actual.agregado_json||jsonb_build_object(
        'version',v_siguiente,'fase_actual','nombramiento','estado_actual','en_curso',
        'actualizado_en',to_char(v_ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'actuaciones',(v_actual.agregado_json->'actuaciones')||jsonb_build_array(v_actuacion));
    v_agregado_hash:=encode(sha256(convert_to(v_agregado::text,'UTF8')),'hex');
    v_prueba:=convert_to('VEC-CT-EXPEDIENTE-PROPUESTA-V1'||chr(10)||
        (s->>'ExpedienteRef')||chr(10)||v_siguiente::text||chr(10)||v_agregado_hash||chr(10)||
        v_propuesta||chr(10)||v_recibo||chr(10)||v_consumo.decision_ref||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.expediente_version_integral (
        expediente_ref,version,agregado_json,agregado_json_huella_sha256,prueba_canonica,prueba_huella_sha256,
        flujo_ref,flujo_version,flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,registrada_en
    ) VALUES (
        s->>'ExpedienteRef',v_siguiente,v_agregado,v_agregado_hash,v_prueba,encode(sha256(v_prueba),'hex'),
        v_actual.flujo_ref,v_actual.flujo_version,v_actual.flujo_huella_sha256,'nombramiento','en_curso',
        'propuesta_formalizacion_o6',v_propuesta,v_ahora);
    UPDATE vec_contratacion_temporal.expediente_integral_actual
       SET version=v_siguiente,actualizada_en=v_ahora,operacion_ref=v_propuesta
     WHERE expediente_ref=s->>'ExpedienteRef' AND version=v_n;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'versión de propuesta perdida' USING ERRCODE='P0612';
    END IF;
    v_prueba:=convert_to('VEC-CT-ACTUACION-PROPUESTA-V1'||chr(10)||v_actuacion::text||chr(10)||v_recibo||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral (
        expediente_ref,secuencia,version_expediente,operacion_ref,recibo_ref,actuacion_json,
        actuacion_json_huella_sha256,prueba_canonica,prueba_huella_sha256,registrada_en
    ) VALUES (
        s->>'ExpedienteRef',v_siguiente,v_siguiente,v_propuesta,v_recibo,v_actuacion,
        encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex'),v_prueba,encode(sha256(v_prueba),'hex'),v_ahora);
    SELECT secuencia_outbox,cabeza_outbox_sha256 INTO STRICT v_secuencia,v_anterior
      FROM vec_contratacion_temporal.control_cadenas_expediente_integral WHERE control_id FOR UPDATE;
    IF v_secuencia>=9007199254740991::numeric THEN
        RAISE EXCEPTION 'límite de outbox alcanzado' USING ERRCODE='P0614';
    END IF;
    v_secuencia:=v_secuencia+1;
    v_payload:=convert_to(jsonb_build_object('esquema','vec.contratacion-temporal.propuesta-formalizacion-registrada.v1',
        'expediente_ref',s->>'ExpedienteRef','version_resultante',v_siguiente,'propuesta_ref',v_propuesta,
        'resolucion_aceptada_ref',v_resolucion.resolucion_ref,'recibo_ref',v_recibo)::text,'UTF8');
    v_outbox_hash:=encode(sha256(v_anterior::bytea||v_payload),'hex');
    INSERT INTO vec_contratacion_temporal.outbox_expediente_integral (
        evento_ref,secuencia,operacion_ref,expediente_ref,version_expediente,tipo_evento,
        payload_canonico,payload_huella_sha256,anterior_sha256,huella_sha256,registrada_en
    ) VALUES (
        v_evento,v_secuencia,v_propuesta,s->>'ExpedienteRef',v_siguiente,'contratacion_temporal.propuesta_formalizacion_registrada',
        v_payload,encode(sha256(v_payload),'hex'),v_anterior,v_outbox_hash,v_ahora);
    UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral
       SET secuencia_outbox=v_secuencia,cabeza_outbox_sha256=v_outbox_hash,actualizada_en=v_ahora WHERE control_id;
    v_resultado:=jsonb_build_object('Solicitud',s,'PropuestaRef',v_propuesta,'ReciboLocalRef',v_recibo,
        'AuditoriaRef',v_consumo.auditoria_ref,'VersionResultante',v_siguiente,
        'ConfirmadaEn',to_char(v_ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'Estado','confirmado');
    INSERT INTO vec_contratacion_temporal.propuesta_formalizacion (
        propuesta_ref,organizacion_ref,expediente_ref,llamamiento_ref,resolucion_ref,recibo_aceptacion_ref,
        clave_idempotencia,version_previa,version_resultante,material,material_json,material_sha256,solicitud_json,
        aceptacion_bolsa_json,actor_ref,perfil_ref,auditoria_ref,decision_ref,consumo_huella_sha256,
        evidencia_huella_sha256,recibo_ref,recibo_json,evento_ref,confirmada_en
    ) VALUES (
        v_propuesta,s->>'OrganizacionRef',s->>'ExpedienteRef',s->>'LlamamientoRef',v_resolucion.resolucion_ref,v_resolucion.recibo_ref,
        (s->>'ClaveIdempotencia')::uuid,v_n,v_siguiente,p_material,m,v_hash,s,b,d->>'principal_id',d->>'perfil_activo_ref',
        v_consumo.auditoria_ref,v_consumo.decision_ref,v_consumo.consumo_huella_sha256,
        encode(sha256(p_evidencia),'hex'),v_recibo,v_resultado,v_evento,v_ahora);
    RETURN v_resultado;
EXCEPTION
    WHEN unique_violation THEN
        RAISE EXCEPTION 'conflicto de propuesta' USING ERRCODE='P0611';
    WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN
        RAISE EXCEPTION 'propuesta no disponible' USING ERRCODE='P0614';
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_propuesta_formalizacion_v2(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC,vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_propuesta_formalizacion_v2(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_ejecutor;
COMMIT;
