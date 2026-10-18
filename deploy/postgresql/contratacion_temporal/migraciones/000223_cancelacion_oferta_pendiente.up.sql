\set ON_ERROR_STOP on
-- CT223: la cancelación comprueba la selección de oferta pendiente bajo el
-- mismo bloqueo de cabecera que CT220. Cuerpo explícito de CT122 instalado;
-- solo añade la guarda después de FOR UPDATE OF ac,v.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000223',0));
DO $pre$
DECLARE f oid; p pg_proc%ROWTYPE;
BEGIN
 f:=to_regprocedure('vec_contratacion_temporal.confirmar_cancelacion_expediente_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR f IS NULL
    OR to_regclass('vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.reservar_seleccion_oferta_ct220(jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT EXISTS(SELECT 1 FROM pg_class c
      WHERE c.oid='vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'::regclass
        AND c.relowner='vec_contratacion_temporal_propietario'::regrole
        AND c.relrowsecurity AND c.relforcerowsecurity)
    OR NOT EXISTS(SELECT 1 FROM pg_index x JOIN pg_class idx ON idx.oid=x.indexrelid
      WHERE x.indrelid='vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'::regclass
        AND idx.relname='ejecucion_seleccion_oferta_unica_expediente_ct220'
        AND x.indisunique AND x.indisvalid)
    OR NOT EXISTS(SELECT 1 FROM pg_attribute a
      WHERE a.attrelid='vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'::regclass
        AND a.attname IN ('tipo_seleccion','comando_oferta_json') AND NOT a.attisdropped
      GROUP BY a.attrelid HAVING count(*)=2)
 THEN RAISE EXCEPTION 'CT223: preimagen CT122 o CT220 ausente' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM
      '30af4df38af7ab6e004e5196657d67278cc8949b01b4521c8b30713309c09c35'
    OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM
      'bc2145d14e5788b7b3315db797c407ba949837fe973d07da01ea411f9b6ab9a6'
    OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR p.pronamespace IS DISTINCT FROM 'vec_contratacion_temporal'::regnamespace
    OR p.prorettype IS DISTINCT FROM 'jsonb'::regtype OR p.pronargs<>11
    OR p.prolang IS DISTINCT FROM (SELECT oid FROM pg_language WHERE lanname='plpgsql')
    OR NOT p.prosecdef OR p.provolatile<>'v'
    OR p.proconfig IS DISTINCT FROM ARRAY[
      'search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[]
    OR p.proacl::text IS DISTINCT FROM
      '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
 THEN RAISE EXCEPTION 'CT223: cuerpo, propietario o ACL CT122 divergente' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.confirmar_cancelacion_expediente_v1(p_operacion jsonb, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE
    m jsonb:=p_operacion->'material'; refs jsonb:=p_operacion->'referencias'; pol jsonb:=p_operacion->'politica'; a jsonb:=p_operacion->'autorizacion';
    e text:='vec.contratacion-temporal.resultado-cancelacion.v1';
    r vec_contratacion_temporal.cancelacion_expediente_v1%ROWTYPE;
    v_version numeric; v_instante timestamptz; v_ahora timestamptz(6); v_actual record; v_dominio jsonb; v_admision text; v_fase text; v_centro text;
    v_decision jsonb; v_consumo record; v_actuacion jsonb; v_siguiente jsonb; v_ambitos jsonb; v_atributos jsonb; v_contexto text;
    v_secuencia_actuacion numeric; v_agregado_huella text; v_prueba bytea; v_payload bytea; v_anterior text; v_secuencia numeric;
    v_recibo jsonb; v_restriccion text; v_tabla text; v_esquema text;
BEGIN
    PERFORM vec_contratacion_temporal.exigir_sesion_ct122(false);
    PERFORM vec_contratacion_temporal.validar_confirmacion_ct122(p_operacion,p_decision,p_motivo,p_persona_version,p_perfil_version);
    PERFORM vec_contratacion_temporal.validar_material_cancelacion_ct122(m);
    IF p_capacidad IS NULL OR p_contexto IS NULL OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
        RAISE EXCEPTION 'CT122: autorización incompleta' USING ERRCODE='42501';
    END IF;
    v_version:=(m->>'version_esperada')::numeric;
    v_instante:=(p_operacion->>'instante_efecto')::timestamptz;
    v_decision:=convert_from(p_decision,'UTF8')::jsonb;
    PERFORM set_config('vec.ct122.organizacion_ref',m->>'organizacion_ref',true);
    PERFORM set_config('vec.ct122.expediente_ref',m->>'expediente_ref',true);
    PERFORM set_config('vec.ct122.actor_ref',m->>'actor_ref',true);
    PERFORM set_config('vec.ct122.perfil_ref',m->>'perfil_ref',true);
    PERFORM set_config('vec.ct122.ambito_hmac',p_operacion->>'ambito_idempotencia_hmac',true);
    SELECT * INTO r FROM vec_contratacion_temporal.cancelacion_expediente_v1 WHERE ambito_hmac=p_operacion->>'ambito_idempotencia_hmac';
    IF FOUND THEN
        -- Una petición nueva recupera por la preparación; aquí solo se admite
        -- la repetición directa con la misma evidencia ya consumida.
        IF NOT vec_contratacion_temporal.misma_intencion_ct122(r,p_operacion->>'huella_peticion_hmac',m) THEN
            RETURN jsonb_build_object('esquema',e,'resultado','idempotencia_reutilizada');
        END IF;
        IF r.decision_ref IS DISTINCT FROM a->>'decision_ref' THEN
            RAISE EXCEPTION 'CT122: evidencia de repetición divergente' USING ERRCODE='42501';
        END IF;
        RETURN jsonb_build_object('esquema',e,'resultado','confirmada','recibo',r.recibo_json);
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.cancelacion_expediente_v1 WHERE expediente_ref=m->>'expediente_ref') THEN
        RETURN jsonb_build_object('esquema',e,'resultado','cancelacion_existente');
    END IF;
    SELECT v.* INTO v_actual FROM vec_contratacion_temporal.expediente_integral_actual ac
      JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version)
     WHERE ac.expediente_ref=m->>'expediente_ref' FOR UPDATE OF ac,v;
    IF EXISTS (
      SELECT 1 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 seleccion
      WHERE seleccion.tipo_seleccion='oferta'
        AND seleccion.situacion IN ('propietaria','indeterminada')
        AND seleccion.solicitud_json->>'organizacion_ref'=m->>'organizacion_ref'
        AND seleccion.solicitud_json->>'expediente_ref'=m->>'expediente_ref'
    ) THEN
      RETURN jsonb_build_object('esquema',e,'resultado','seleccion_oferta_pendiente');
    END IF;
    v_dominio:=vec_contratacion_temporal.agregado_dominio_ct122(v_actual.agregado_json,v_actual.version);
    v_admision:=vec_contratacion_temporal.admision_cancelacion_ct122(v_dominio,v_actual.version,m);
    IF v_admision<>'admitida' THEN
        RETURN jsonb_build_object('esquema',e,'resultado',v_admision);
    END IF;
    IF v_dominio IS DISTINCT FROM p_operacion->'expediente_anterior' THEN
        RETURN jsonb_build_object('esquema',e,'resultado','version_en_conflicto');
    END IF;
    v_fase:=v_dominio->>'fase_actual';
    v_secuencia_actuacion:=jsonb_array_length(v_dominio->'actuaciones')+1;
    v_actuacion:=jsonb_build_object('secuencia',v_secuencia_actuacion,'version_expediente',v_version+1,
        'accion_clave','contratacion_temporal.expediente.cancelar','actor_ref',m->>'actor_ref',
        'unidad_ref',v_dominio->'actuaciones'->-1->>'unidad_ref','recibo_ref',refs->>'recibo_ref',
        'realizada_en',p_operacion->'instante_efecto','fase_origen',v_fase,'fase_destino',v_fase,
        'estado_origen','en_curso','estado_destino','cancelado');
    IF m->>'observaciones'<>'' THEN
        v_actuacion:=v_actuacion||jsonb_build_object('observaciones',m->>'observaciones');
    END IF;
    v_siguiente:=v_dominio||jsonb_build_object('version',v_version+1,'actualizado_en',p_operacion->'instante_efecto',
        'estado_actual','cancelado','actuaciones',(v_dominio->'actuaciones')||jsonb_build_array(v_actuacion));
    IF v_instante<(v_dominio->>'actualizado_en')::timestamptz
       OR v_actuacion IS DISTINCT FROM p_operacion->'actuacion'
       OR v_siguiente IS DISTINCT FROM p_operacion->'expediente_siguiente' THEN
        RAISE EXCEPTION 'CT122: proyección de cancelación divergente' USING ERRCODE='22023';
    END IF;
    v_ambitos:=jsonb_build_object('organizacion_ref',m->>'organizacion_ref','expediente_ref',m->>'expediente_ref',
        'fase_previa',v_fase,'estado_previo','en_curso');
    IF m->>'canal'='centro' THEN
        v_centro:=v_dominio#>>'{solicitud,centro_ref}';
        v_ambitos:=v_ambitos||jsonb_build_object('centro_ref',v_centro);
    END IF;
    v_atributos:=jsonb_build_object('version_expediente',v_version::text,'canal',m->>'canal','motivo_clave',m->>'motivo_clave',
        'fases_admitidas',vec_contratacion_temporal.fases_texto_ct122(m),
        'observaciones_huella_sha256',encode(sha256(convert_to(m->>'observaciones','UTF8')),'hex'),
        'politica_ref',pol->>'definicion_ref','politica_version',pol->>'definicion_version',
        'politica_huella_sha256',pol->>'definicion_huella_sha256','ambito_idempotencia_hmac',p_operacion->>'ambito_idempotencia_hmac',
        'huella_peticion_hmac',p_operacion->>'huella_peticion_hmac');
    IF p_operacion#>'{contexto,ambitos}' IS DISTINCT FROM v_ambitos OR p_operacion#>'{contexto,atributos}' IS DISTINCT FROM v_atributos THEN
        RAISE EXCEPTION 'CT122: contexto de cancelación divergente' USING ERRCODE='42501';
    END IF;
    v_contexto:=vec_contratacion_temporal.huella_contexto_go_ct122(v_ambitos,v_atributos);
    IF a->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto
       OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto
       OR v_decision->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
       OR v_decision->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
       OR v_decision->>'recurso_ref' IS DISTINCT FROM m->>'expediente_ref'
       OR v_decision->>'accion' IS DISTINCT FROM 'contratacion_temporal.expediente.cancelar'
       OR v_decision->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'cancelacion_contratacion_temporal'
       OR v_decision->>'finalidad' IS DISTINCT FROM 'cancelar_expediente_contratacion_temporal'
       OR v_decision->>'decision_ref' IS DISTINCT FROM a->>'decision_ref' THEN
        RAISE EXCEPTION 'CT122: contexto autorizado de cancelación divergente' USING ERRCODE='42501';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF (pol->>'evaluada_en')::timestamptz>v_instante OR v_ahora<v_instante OR v_ahora>=(pol->>'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION 'CT122: vigencia de cancelación agotada' USING ERRCODE='42501';
    END IF;
    SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_cancelacion_expediente_ct_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.decision_ref IS DISTINCT FROM a->>'decision_ref' OR v_consumo.efecto_ref IS DISTINCT FROM m->>'expediente_ref'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto OR coalesce(v_consumo.auditoria_ref,'') !~ '^aud_v3_[0-9a-f]{32}$'
       OR v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'CT122: consumo de cancelación divergente' USING ERRCODE='42501';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF v_ahora<v_instante OR v_ahora>=(pol->>'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION 'CT122: vigencia final de cancelación agotada' USING ERRCODE='42501';
    END IF;
    v_agregado_huella:=encode(sha256(convert_to(v_siguiente::text,'UTF8')),'hex');
    v_prueba:=convert_to('VEC-CT-EXPEDIENTE-CANCELACION-CT122'||chr(10)||(m->>'expediente_ref')||chr(10)||(v_version+1)::text||chr(10)
        ||v_agregado_huella||chr(10)||(refs->>'reserva_ref')||chr(10)||(refs->>'recibo_ref')||chr(10)||v_consumo.decision_ref||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.expediente_version_integral(
        expediente_ref,version,agregado_json,agregado_json_huella_sha256,prueba_canonica,prueba_huella_sha256,
        flujo_ref,flujo_version,flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,registrada_en)
    VALUES(m->>'expediente_ref',v_version+1,v_siguiente,v_agregado_huella,v_prueba,encode(sha256(v_prueba),'hex'),
        v_actual.flujo_ref,v_actual.flujo_version,v_actual.flujo_huella_sha256,v_fase,'cancelado',
        'cancelacion_expediente_ct122',refs->>'reserva_ref',v_ahora);
    UPDATE vec_contratacion_temporal.expediente_integral_actual SET version=v_version+1,actualizada_en=v_ahora,operacion_ref=refs->>'reserva_ref'
     WHERE expediente_ref=m->>'expediente_ref' AND version=v_version;
    IF NOT FOUND THEN RAISE EXCEPTION 'CT122: CAS final de cancelación perdido' USING ERRCODE='40001'; END IF;
    v_prueba:=convert_to('VEC-CT-ACTUACION-CANCELACION-CT122'||chr(10)||encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex')||chr(10)
        ||(refs->>'recibo_ref')||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral(
        expediente_ref,secuencia,version_expediente,operacion_ref,recibo_ref,actuacion_json,
        actuacion_json_huella_sha256,prueba_canonica,prueba_huella_sha256,registrada_en)
    VALUES(m->>'expediente_ref',v_secuencia_actuacion,v_version+1,refs->>'reserva_ref',refs->>'recibo_ref',v_actuacion,
        encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex'),v_prueba,encode(sha256(v_prueba),'hex'),v_ahora);
    SELECT secuencia_outbox,cabeza_outbox_sha256 INTO STRICT v_secuencia,v_anterior
      FROM vec_contratacion_temporal.control_cadenas_expediente_integral WHERE control_id FOR UPDATE;
    IF v_secuencia>=9007199254740991 THEN RAISE EXCEPTION 'CT122: límite de outbox alcanzado' USING ERRCODE='22003'; END IF;
    v_secuencia:=v_secuencia+1;
    -- Evento de dominio `ct.expediente_cancelado.v1`: solo referencias opacas,
    -- claves y fechas. No hay llamamiento previo a la fiscalización: no lleva
    -- efecto para Bolsa.
    v_payload:=convert_to(jsonb_build_object('esquema','vec.contratacion-temporal.expediente-cancelado.v1','organizacion_ref',m->>'organizacion_ref',
        'expediente_ref',m->>'expediente_ref','version_resultante',v_version+1,'fase_previa',v_fase,'canal',m->>'canal',
        'motivo_clave',m->>'motivo_clave','recibo_ref',refs->>'recibo_ref','registrada_en',p_operacion->'instante_efecto')::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.outbox_expediente_integral(
        evento_ref,secuencia,operacion_ref,expediente_ref,version_expediente,tipo_evento,payload_canonico,
        payload_huella_sha256,anterior_sha256,huella_sha256,registrada_en)
    VALUES(refs->>'evento_ref',v_secuencia,refs->>'reserva_ref',m->>'expediente_ref',v_version+1,'ct.expediente_cancelado.v1',v_payload,
        encode(sha256(v_payload),'hex'),v_anterior,encode(sha256(v_anterior::bytea||v_payload),'hex'),v_ahora);
    UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral
       SET secuencia_outbox=v_secuencia,cabeza_outbox_sha256=encode(sha256(v_anterior::bytea||v_payload),'hex'),actualizada_en=v_ahora
     WHERE control_id;
    v_recibo:=jsonb_build_object('operacion','cancelar_expediente','organizacion_ref',m->>'organizacion_ref','expediente_ref',m->>'expediente_ref',
        'version_anterior',v_version,'version_resultante',v_version+1,'fase_resultante',v_fase,'estado_resultante','cancelado',
        'motivo_clave',m->>'motivo_clave','recibo_ref',refs->>'recibo_ref',
        'auditoria_ref',v_consumo.auditoria_ref,'evento_ref',refs->>'evento_ref','actor_ref',m->>'actor_ref',
        'registrada_en',p_operacion->'instante_efecto');
    INSERT INTO vec_contratacion_temporal.cancelacion_expediente_v1(
        ambito_hmac,huella_peticion_hmac,organizacion_ref,expediente_ref,version_esperada,actor_ref,perfil_ref,canal,centro_ref,motivo_clave,
        fases_admitidas,fase_previa,observaciones,estado,reserva_ref,recibo_ref,evento_ref,expediente_anterior_json,expediente_siguiente_json,
        recibo_json,decision_ref,decision_huella_sha256,consumo_huella_sha256,auditoria_ref,politica_ref,politica_version,politica_huella_sha256,
        registrada_en,confirmada_en)
    VALUES(p_operacion->>'ambito_idempotencia_hmac',p_operacion->>'huella_peticion_hmac',m->>'organizacion_ref',m->>'expediente_ref',v_version,
        m->>'actor_ref',m->>'perfil_ref',m->>'canal',v_centro,m->>'motivo_clave',
        ARRAY(SELECT x #>> '{}' FROM jsonb_array_elements(m->'fases_admitidas') WITH ORDINALITY y(x,o) ORDER BY o),v_fase,
        m->>'observaciones','confirmada',refs->>'reserva_ref',refs->>'recibo_ref',refs->>'evento_ref',v_dominio,v_siguiente,
        v_recibo,v_consumo.decision_ref,a->>'decision_huella_sha256',v_consumo.consumo_huella_sha256,v_consumo.auditoria_ref,
        pol->>'definicion_ref',(pol->>'definicion_version')::numeric,pol->>'definicion_huella_sha256',v_instante,v_ahora);
    RETURN jsonb_build_object('esquema',e,'resultado','confirmada','recibo',v_recibo);
EXCEPTION
WHEN unique_violation THEN
    GET STACKED DIAGNOSTICS v_restriccion=CONSTRAINT_NAME,v_tabla=TABLE_NAME,v_esquema=SCHEMA_NAME;
    IF v_esquema='vec_contratacion_temporal' AND v_tabla='cancelacion_expediente_v1' THEN
        RETURN jsonb_build_object('esquema',e,'resultado',CASE WHEN v_restriccion='cancelacion_expediente_v1_pkey' THEN 'idempotencia_reutilizada' ELSE 'cancelacion_existente' END);
    END IF;
    RAISE;
WHEN invalid_text_representation OR datetime_field_overflow OR numeric_value_out_of_range OR character_not_in_repertoire THEN
    RAISE EXCEPTION 'CT122: entrada de cancelación inválida' USING ERRCODE='22023';
END
$function$
;
DO $post$
DECLARE f oid; p pg_proc%ROWTYPE;
BEGIN
 f:='vec_contratacion_temporal.confirmar_cancelacion_expediente_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM
      'ec8c561770cebd31e83d2f0ee9b79861846ab78875cfcf61cf1907336c3f8545'
    OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM
      '8ef53b5355ccd0fd732e5beb11c762a8d91133080bdb8500b50da730a3801c26'
    OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR p.pronamespace IS DISTINCT FROM 'vec_contratacion_temporal'::regnamespace
    OR p.prorettype IS DISTINCT FROM 'jsonb'::regtype OR p.pronargs<>11
    OR p.prolang IS DISTINCT FROM (SELECT oid FROM pg_language WHERE lanname='plpgsql')
    OR NOT p.prosecdef OR p.provolatile<>'v'
    OR p.proconfig IS DISTINCT FROM ARRAY[
      'search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[]
    OR p.proacl::text IS DISTINCT FROM
      '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
 THEN RAISE EXCEPTION 'CT223: postimagen, search_path o permisos divergentes' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
