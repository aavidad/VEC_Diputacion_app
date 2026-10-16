\set ON_ERROR_STOP on
-- CT227. El ámbito autorizado de vínculo, consulta y descarte de oferta
-- es organización + unidad gestora del expediente. Centro y categoría
-- permanecen en los cánones y se cotejan con la versión del expediente.
-- CT220, CT223 y CT225 están ya instaladas; esta migración es prospectiva.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000227',0));
DO $pre$
DECLARE f oid; p pg_proc%ROWTYPE; r record;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR to_regprocedure('vec_contratacion_temporal.preparar_contexto_estado_oferta_ct227(text,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT227: preimagen o número incompatible' USING ERRCODE='55000'; END IF;
 FOR r IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.consumir_etapa_seleccion_oferta_ct220(text,jsonb,jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   '059b45d8c5118e9fb874a3eb7908deca78c4cec50d256dd58191f2acc610d404',
   '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}'),
  ('vec_contratacion_temporal.descartar_seleccion_oferta_ct220(jsonb,text,jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   '0d1db657d7359a7b09cb0f6781e2af59f2fa060292de65fc536fcfa5b14e3d8e',
   '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'),
  ('vec_contratacion_temporal.consultar_estado_oferta_expediente_ct225(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   '3f4c78128de156e3e8537f537114aeffd05bc276e1e099b5ecb3ae026ec7ec23',
   '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'),
  ('vec_contratacion_temporal.preparar_contexto_estado_oferta_ct225(text,text)',
   '89934f3ad1e74b6487ce3adb536aa9d8754950be697091f50cfd0a4d10222031',
   '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}')
 ) AS x(firma,huella,acl) LOOP
 f:=to_regprocedure(r.firma);
 IF f IS NULL THEN RAISE EXCEPTION 'CT227: función ausente %',r.firma USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM r.huella
    OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR NOT p.prosecdef OR p.proacl::text IS DISTINCT FROM r.acl
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
 THEN RAISE EXCEPTION 'CT227: preimagen, dueño, configuración o ACL divergente %',r.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;

CREATE FUNCTION vec_contratacion_temporal.preparar_contexto_estado_oferta_ct227(
 p_organizacion text,p_expediente text)
RETURNS TABLE(version_actual bigint,categoria_ref text,centro_ref text,unidad_gestora_ref text)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
BEGIN
 -- CT225 valida sesión, organización y expediente antes de exponer contexto.
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER') IS TRUE
 THEN RAISE EXCEPTION 'CT227: contexto denegado' USING ERRCODE='42501'; END IF;
 RETURN QUERY
 SELECT c.version_actual,c.categoria_ref,c.centro_ref,
        v.agregado_json#>>'{asignacion,unidad_ref}'
 FROM vec_contratacion_temporal.preparar_contexto_estado_oferta_ct225(p_organizacion,p_expediente) c
 JOIN vec_contratacion_temporal.expediente_integral_actual a
   ON a.expediente_ref=p_expediente AND a.version=c.version_actual
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
 WHERE v.agregado_json->>'organizacion_ref'=p_organizacion
   AND v.agregado_json#>>'{asignacion,unidad_ref}' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$';
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.preparar_contexto_estado_oferta_ct227(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.preparar_contexto_estado_oferta_ct227(text,text)
 TO vec_contratacion_temporal_ejecutor;
-- Desde aquí la preparación CT225 solo se usa dentro de la CT227, como propietario.
REVOKE EXECUTE ON FUNCTION vec_contratacion_temporal.preparar_contexto_estado_oferta_ct225(text,text)
 FROM vec_contratacion_temporal_ejecutor;

-- Definición final instalada y verificada de consumir_etapa_seleccion_oferta_ct220.
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.consumir_etapa_seleccion_oferta_ct220(p_etapa text, p_intencion jsonb, p_comando jsonb, p_oferta jsonb, p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS TABLE(decision_ref text, auditoria_ref text, consumo_huella_sha256 text)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE m jsonb;d jsonb;consumo record;h text;ctx text;v_unidad_gestora text;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER') IS TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_etapa NOT IN ('consultar','reservar','abrir_ventana','confirmar')
    OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 128 AND 1048576
    OR vec_contratacion_temporal.intencion_seleccion_oferta_valida_ct220(
      p_intencion,(p_intencion->>'clave_idempotencia')::uuid,p_intencion->>'huella_intencion') IS NOT TRUE
 THEN RAISE EXCEPTION 'CT220: etapa no autorizable' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'CT220: material ilegible' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(m)<>'object' OR m->>'esquema'<>'vec.ct.seleccion-oferta.etapa.v1'
    OR m->>'etapa' IS DISTINCT FROM p_etapa
    OR m->>'organizacion_ref' IS DISTINCT FROM p_intencion->>'organizacion_ref'
    OR m->>'expediente_ref' IS DISTINCT FROM p_intencion->>'expediente_ref'
    OR m->'version_expediente' IS DISTINCT FROM p_intencion->'version_expediente'
    OR m->>'clave_idempotencia' IS DISTINCT FROM p_intencion->>'clave_idempotencia'
    OR m->>'huella_intencion' IS DISTINCT FROM p_intencion->>'huella_intencion'
    OR m->>'categoria_ref' IS DISTINCT FROM p_intencion->>'categoria_ref'
    OR m->>'centro_ref' IS DISTINCT FROM p_intencion->>'centro_ref'
    OR m->'comando' IS DISTINCT FROM p_comando
    OR m->'oferta' IS DISTINCT FROM p_oferta
    OR m-ARRAY['esquema','etapa','organizacion_ref','expediente_ref',
      'version_expediente','clave_idempotencia','huella_intencion',
      'categoria_ref','centro_ref','comando','oferta']<>'{}'::jsonb
    OR (p_etapa='consultar' AND (m ? 'comando' OR m ? 'oferta'))
    OR (p_etapa<>'consultar' AND (p_comando IS NULL OR NOT (m ? 'comando')))
    OR (p_etapa<>'confirmar' AND (p_oferta IS NOT NULL OR m ? 'oferta'))
    OR (p_etapa='confirmar' AND (p_oferta IS NULL OR NOT (m ? 'oferta')))
 THEN RAISE EXCEPTION 'CT220: etapa y material divergentes' USING ERRCODE='42501'; END IF;
 SELECT v.agregado_json#>>'{asignacion,unidad_ref}' INTO v_unidad_gestora
 FROM vec_contratacion_temporal.expediente_version_integral v
 WHERE v.expediente_ref=p_intencion->>'expediente_ref'
   AND v.version=(p_intencion->>'version_expediente')::bigint
   AND v.agregado_json->>'organizacion_ref'=p_intencion->>'organizacion_ref';
 IF v_unidad_gestora IS NULL OR v_unidad_gestora !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 THEN RAISE EXCEPTION 'CT227: unidad gestora no acreditada' USING ERRCODE='42501'; END IF;
 h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 ctx:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":'||
   to_json(p_intencion->>'organizacion_ref')::text||',"unidad_ref":'||
   to_json(v_unidad_gestora)::text||
   '},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 IF d->>'principal_id' IS DISTINCT FROM p_intencion->>'actor_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM p_intencion->>'perfil_ref'
    OR d->>'accion' IS DISTINCT FROM 'contratacion_temporal.bolsa.oferta.vincular'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'vinculo_oferta_bolsa_expediente'
    OR d->>'finalidad' IS DISTINCT FROM 'tramitacion_expediente_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM p_intencion->>'expediente_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM ctx
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'CT220: decisión no ligada a etapa' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_oferta_bolsa_ct_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.efecto_ref IS DISTINCT FROM p_intencion->>'expediente_ref'
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM ctx
    OR consumo.consumo_nuevo IS NOT TRUE
 THEN RAISE EXCEPTION 'CT220: consumo V3 divergente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256;
END $function$;

-- Definición final instalada y verificada de descartar_seleccion_oferta_ct220.
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.descartar_seleccion_oferta_ct220(p_intencion jsonb, p_reserva text, p_comando jsonb, p_cierre jsonb, p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE e vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
 cab record;m jsonb;d jsonb;auth record;h text;ctx text;v_unidad_gestora text;
 historico jsonb;historico_h text;
 v_ahora timestamptz(6);v_recibo text;v_evento text;v_salida jsonb;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER') IS TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_reserva IS NULL OR p_material IS NULL
    OR octet_length(p_material) NOT BETWEEN 128 AND 1048576
    OR vec_contratacion_temporal.intencion_seleccion_oferta_valida_ct220(
      p_intencion,(p_intencion->>'clave_idempotencia')::uuid,p_intencion->>'huella_intencion') IS NOT TRUE
    OR vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(p_comando,p_intencion) IS NOT TRUE
    OR jsonb_typeof(p_cierre)<>'object' OR octet_length(p_cierre::text)>65536
 THEN RAISE EXCEPTION 'CT220: descarte no autorizable' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'CT220: material de descarte ilegible' USING ERRCODE='22023';
 END;
 IF m->>'esquema' IS DISTINCT FROM 'vec.ct.seleccion-oferta.descarte.v1'
    OR m->>'organizacion_ref' IS DISTINCT FROM p_intencion->>'organizacion_ref'
    OR m->>'expediente_ref' IS DISTINCT FROM p_intencion->>'expediente_ref'
    OR m->'version_expediente' IS DISTINCT FROM p_intencion->'version_expediente'
    OR m->>'clave_idempotencia' IS DISTINCT FROM p_intencion->>'clave_idempotencia'
    OR m->>'huella_intencion' IS DISTINCT FROM p_intencion->>'huella_intencion'
    OR m->>'actor_ref' IS DISTINCT FROM p_intencion->>'actor_ref'
    OR m->>'categoria_ref' IS DISTINCT FROM p_intencion->>'categoria_ref'
    OR m->>'centro_ref' IS DISTINCT FROM p_intencion->>'centro_ref'
    OR m->'comando' IS DISTINCT FROM p_comando
    OR m->'cierre' IS DISTINCT FROM p_cierre
    OR m-ARRAY['esquema','organizacion_ref','expediente_ref','version_expediente',
      'clave_idempotencia','huella_intencion','actor_ref','categoria_ref','centro_ref',
      'comando','cierre']<>'{}'::jsonb
    OR p_cierre->>'esquema' IS DISTINCT FROM 'vec.bolsa.oferta-ct.cierre.v1'
    OR p_cierre->>'estado' IS DISTINCT FROM 'cerrada_sin_publicar'
    OR p_cierre->>'organizacion_ref' IS DISTINCT FROM p_intencion->>'organizacion_ref'
    OR p_cierre->>'expediente_ref' IS DISTINCT FROM p_intencion->>'expediente_ref'
    OR p_cierre->>'version_expediente' IS DISTINCT FROM p_intencion->>'version_expediente'
    OR p_cierre->>'clave_ct' IS DISTINCT FROM p_intencion->>'clave_idempotencia'
    OR p_cierre->>'actor_ref' IS DISTINCT FROM p_intencion->>'actor_ref'
    OR p_cierre->>'bolsa_ref' IS DISTINCT FROM p_comando->>'bolsa_ref'
    OR p_cierre->>'clave_bolsa' IS DISTINCT FROM p_comando->>'clave_bolsa'
    OR p_cierre->'numero_plazas' IS DISTINCT FROM p_comando->'numero_plazas'
    OR p_cierre->>'comando_sha256' IS DISTINCT FROM encode(sha256(convert_to(p_comando::text,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'CT220: cierre o material divergente' USING ERRCODE='42501'; END IF;
 SELECT v.agregado_json#>>'{asignacion,unidad_ref}' INTO v_unidad_gestora
 FROM vec_contratacion_temporal.expediente_version_integral v
 WHERE v.expediente_ref=p_intencion->>'expediente_ref'
   AND v.version=(p_intencion->>'version_expediente')::bigint
   AND v.agregado_json->>'organizacion_ref'=p_intencion->>'organizacion_ref';
 IF v_unidad_gestora IS NULL OR v_unidad_gestora !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 THEN RAISE EXCEPTION 'CT227: unidad gestora no acreditada' USING ERRCODE='42501'; END IF;
 h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 ctx:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":'||
   to_json(p_intencion->>'organizacion_ref')::text||',"unidad_ref":'||
   to_json(v_unidad_gestora)::text||
   '},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 IF d->>'principal_id' IS DISTINCT FROM p_intencion->>'actor_ref'
    OR d->>'perfil_activo_ref' IS NOT DISTINCT FROM p_intencion->>'perfil_ref'
    OR d->>'accion' IS DISTINCT FROM 'contratacion_temporal.bolsa.oferta.descartar'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'seleccion_oferta_expediente'
    OR d->>'finalidad' IS DISTINCT FROM 'tramitacion_expediente_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM p_intencion->>'expediente_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM ctx
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'CT220: decisión de descarte divergente' USING ERRCODE='42501'; END IF;
 -- Igual orden que reservar y cancelar: cabecera CT → V3 → O6.
 SELECT a.version INTO cab FROM vec_contratacion_temporal.expediente_integral_actual a
 JOIN vec_contratacion_temporal.expediente_alta alta ON alta.expediente_ref=a.expediente_ref
 WHERE a.expediente_ref=p_intencion->>'expediente_ref'
   AND alta.organizacion_ref=p_intencion->>'organizacion_ref'
 FOR UPDATE OF a;
 SELECT * INTO STRICT auth FROM vec_autorizacion_atestada_v3.registrar_y_consumir_descarte_seleccion_oferta_ct_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF auth.efecto_ref IS DISTINCT FROM p_intencion->>'expediente_ref'
    OR auth.huella_efecto_sha256 IS DISTINCT FROM ctx OR auth.consumo_nuevo IS NOT TRUE
 THEN RAISE EXCEPTION 'CT220: consumo de descarte divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO e FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 WHERE clave_idempotencia=(p_intencion->>'clave_idempotencia')::uuid FOR UPDATE;
 IF cab.version IS NULL OR e.clave_idempotencia IS NULL
    OR e.tipo_seleccion<>'oferta' OR e.solicitud_json IS DISTINCT FROM p_intencion
    OR e.comando_oferta_json IS DISTINCT FROM p_comando
    OR e.reserva_ref IS DISTINCT FROM p_reserva
 THEN RAISE EXCEPTION 'CT220: reserva de descarte divergente' USING ERRCODE='23505'; END IF;
 IF e.situacion='descartada' THEN
  IF vec_contratacion_temporal.terminal_descarte_oferta_valido_ct220(e) IS NOT TRUE
     OR (SELECT cierre_json FROM vec_contratacion_temporal.descarte_seleccion_oferta_ct_v1
       WHERE clave_idempotencia=e.clave_idempotencia) IS DISTINCT FROM p_cierre
  THEN RAISE EXCEPTION 'CT220: descarte previo divergente' USING ERRCODE='23505'; END IF;
  RETURN e.recibo_json;
 END IF;
 IF e.situacion NOT IN ('propietaria','indeterminada')
    OR vec_bolsa_llamamientos.verificar_cierre_intencion_oferta_ct_v1(p_comando,p_cierre) IS NOT TRUE
 THEN RAISE EXCEPTION 'CT220: Bolsa no acredita cierre sin oferta' USING ERRCODE='42501'; END IF;
 SELECT v.agregado_json,v.agregado_json_huella_sha256 INTO historico,historico_h
 FROM vec_contratacion_temporal.expediente_version_integral v
 WHERE v.expediente_ref=p_intencion->>'expediente_ref'
   AND v.version=(p_intencion->>'version_expediente')::numeric;
 IF NOT FOUND OR historico_h IS DISTINCT FROM encode(sha256(convert_to(historico::text,'UTF8')),'hex')
    OR historico->>'fase_actual' IS DISTINCT FROM 'fiscalizacion'
    OR historico->>'estado_actual' IS DISTINCT FROM 'en_curso'
    OR historico#>>'{solicitud,numero_personas}' IS DISTINCT FROM p_comando->>'numero_plazas'
    OR historico#>>'{solicitud,centro_ref}' IS DISTINCT FROM p_intencion->>'centro_ref'
    OR historico#>>'{analisis,categoria_ref}' IS DISTINCT FROM p_intencion->>'categoria_ref'
    OR historico#>>'{asignacion,unidad_ref}' IS DISTINCT FROM p_comando#>>'{origen_ct,unidad_ref}'
 THEN RAISE EXCEPTION 'CT220: versión original de descarte divergente' USING ERRCODE='55000'; END IF;
 v_ahora:=date_trunc('microseconds',clock_timestamp());
 IF v_ahora<(p_cierre->>'cerrada_en')::timestamptz THEN
  RAISE EXCEPTION 'CT220: cierre futuro' USING ERRCODE='22023'; END IF;
 v_recibo:='recibo:ct:oferta-descarte:'||gen_random_uuid()::text;
 v_evento:='evento:ct:oferta-descarte:'||gen_random_uuid()::text;
 v_salida:=jsonb_build_object(
  'esquema','vec.contratacion-temporal.recibo-descarte-oferta.v1','estado','descartada',
  'organizacion_ref',p_intencion->>'organizacion_ref',
  'expediente_ref',p_intencion->>'expediente_ref',
  'version_expediente',(p_intencion->>'version_expediente')::bigint,
  'bolsa_ref',p_comando->>'bolsa_ref','clave_bolsa',p_comando->>'clave_bolsa',
  'recibo_cierre_ref',p_cierre->>'recibo_cierre_ref','recibo_descarte_ref',v_recibo,
  'motivo','origen_caducado',
  'descartada_en',to_char(v_ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'auditoria_ref',auth.auditoria_ref,'evento_ref',v_evento);
 INSERT INTO vec_contratacion_temporal.descarte_seleccion_oferta_ct_v1(
  clave_idempotencia,organizacion_ref,expediente_ref,version_expediente,
  bolsa_ref,clave_bolsa,actor_ref,perfil_descarte_ref,comando_sha256,
  recibo_cierre_ref,cierre_json,recibo_descarte_ref,recibo_json,
  decision_ref,auditoria_ref,consumo_huella_sha256,evento_ref,descartada_en)
 VALUES(e.clave_idempotencia,p_intencion->>'organizacion_ref',p_intencion->>'expediente_ref',
  (p_intencion->>'version_expediente')::bigint,p_comando->>'bolsa_ref',p_comando->>'clave_bolsa',
  p_intencion->>'actor_ref',d->>'perfil_activo_ref',encode(sha256(convert_to(p_comando::text,'UTF8')),'hex'),
  p_cierre->>'recibo_cierre_ref',p_cierre,v_recibo,v_salida,
  auth.decision_ref,auth.auditoria_ref,auth.consumo_huella_sha256,v_evento,v_ahora);
 INSERT INTO vec_contratacion_temporal.outbox_descarte_seleccion_oferta_ct_v1(
  evento_ref,clave_idempotencia,recibo_descarte_ref,tipo,payload,payload_sha256,registrada_en)
 VALUES(v_evento,e.clave_idempotencia,v_recibo,'ct.oferta_bolsa.preparacion_descartada.v1',v_salida,
  encode(sha256(convert_to(v_salida::text,'UTF8')),'hex'),v_ahora);
 UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 SET situacion='descartada',efecto=NULL,recibo_json=v_salida,actualizada_en=v_ahora
 WHERE clave_idempotencia=e.clave_idempotencia;
 RETURN v_salida;
END $function$;

-- Definición final instalada y verificada de consultar_estado_oferta_expediente_ct225.
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.consultar_estado_oferta_expediente_ct225(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE m jsonb;d jsonb;h text;ctx text;c record;e vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
 consumo record;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER') IS TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 128 AND 65536
 THEN RAISE EXCEPTION 'CT225: lectura no autorizable' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'CT225: material ilegible' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(m)<>'object'
    OR m->>'esquema' IS DISTINCT FROM 'vec.ct.seleccion-oferta.consulta-expediente.v1'
    OR m-ARRAY['esquema','organizacion_ref','expediente_ref','version_observada',
      'categoria_ref','centro_ref','actor_ref','perfil_ref']<>'{}'::jsonb
    OR NOT (m ?& ARRAY['esquema','organizacion_ref','expediente_ref','version_observada',
      'categoria_ref','centro_ref','actor_ref','perfil_ref'])
    OR jsonb_typeof(m->'organizacion_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(m->'expediente_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(m->'categoria_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(m->'centro_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(m->'actor_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(m->'perfil_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(m->'version_observada')<>'number'
    OR m->>'version_observada' !~ '^[1-9][0-9]{0,15}$'
    OR length(m->>'organizacion_ref') NOT BETWEEN 2 AND 160
    OR length(m->>'expediente_ref') NOT BETWEEN 2 AND 160
    OR length(m->>'categoria_ref') NOT BETWEEN 2 AND 160
    OR length(m->>'centro_ref') NOT BETWEEN 2 AND 160
    OR length(m->>'actor_ref') NOT BETWEEN 2 AND 160
    OR length(m->>'perfil_ref') NOT BETWEEN 2 AND 160
 THEN RAISE EXCEPTION 'CT225: consulta inválida' USING ERRCODE='22023'; END IF;
 -- Se bloquea la cabecera antes de leer O6; reserva, confirmación y
 -- cancelación usan el mismo orden de bloqueo.
 SELECT a.version,v.agregado_json INTO c
 FROM vec_contratacion_temporal.expediente_integral_actual a
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
 WHERE a.expediente_ref=m->>'expediente_ref'
   AND v.agregado_json->>'organizacion_ref'=m->>'organizacion_ref'
 FOR SHARE OF a,v;
 IF c.version IS NULL OR c.version::text IS DISTINCT FROM m->>'version_observada'
    OR c.agregado_json#>>'{solicitud,categoria_ref}' IS DISTINCT FROM m->>'categoria_ref'
    OR c.agregado_json#>>'{solicitud,centro_ref}' IS DISTINCT FROM m->>'centro_ref'
 THEN RAISE EXCEPTION 'CT225: contexto CT cambió' USING ERRCODE='55000'; END IF;
 IF c.agregado_json#>>'{asignacion,unidad_ref}' IS NULL
    OR c.agregado_json#>>'{asignacion,unidad_ref}' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 THEN RAISE EXCEPTION 'CT227: unidad gestora no acreditada' USING ERRCODE='42501'; END IF;
 h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 ctx:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":'||
   to_json(m->>'organizacion_ref')::text||',"unidad_ref":'||
   to_json(c.agregado_json#>>'{asignacion,unidad_ref}')::text||
   '},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 IF d->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
    OR d->>'accion' IS DISTINCT FROM 'contratacion_temporal.bolsa.oferta.vincular'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'vinculo_oferta_bolsa_expediente'
    OR d->>'finalidad' IS DISTINCT FROM 'tramitacion_expediente_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM m->>'expediente_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM ctx
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'CT225: decisión no ligada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_oferta_bolsa_ct_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.efecto_ref IS DISTINCT FROM m->>'expediente_ref'
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM ctx
    OR consumo.consumo_nuevo IS NOT TRUE
 THEN RAISE EXCEPTION 'CT225: consumo V3 divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO e FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 x
 WHERE x.tipo_seleccion='oferta'
   AND x.solicitud_json->>'organizacion_ref'=m->>'organizacion_ref'
   AND x.solicitud_json->>'expediente_ref'=m->>'expediente_ref'
 ORDER BY (x.situacion<>'descartada') DESC,x.actualizada_en DESC,x.clave_idempotencia DESC
 LIMIT 1 FOR SHARE;
 IF e.clave_idempotencia IS NULL THEN
  RETURN jsonb_build_object('situacion','ausente');
 END IF;
 IF e.solicitud_json->>'actor_ref' IS DISTINCT FROM m->>'actor_ref'
    OR e.solicitud_json->>'perfil_ref' IS DISTINCT FROM m->>'perfil_ref'
    OR e.solicitud_json->>'categoria_ref' IS DISTINCT FROM m->>'categoria_ref'
    OR e.solicitud_json->>'centro_ref' IS DISTINCT FROM m->>'centro_ref'
    OR vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(e.comando_oferta_json,e.solicitud_json) IS NOT TRUE
    OR (e.situacion='confirmada' AND
      vec_contratacion_temporal.terminal_seleccion_oferta_valido_ct220(e) IS NOT TRUE)
    OR (e.situacion='descartada' AND
      vec_contratacion_temporal.terminal_descarte_oferta_valido_ct220(e) IS NOT TRUE)
 THEN RAISE EXCEPTION 'CT225: ejecución incompatible' USING ERRCODE='55000'; END IF;
 RETURN jsonb_build_object('situacion',e.situacion,
  'intencion_original',e.solicitud_json,
  'estado',vec_contratacion_temporal.estado_seleccion_oferta_ct220(e));
END $function$;

DO $post$
DECLARE f oid; p pg_proc%ROWTYPE; r record;
BEGIN
 FOR r IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.consumir_etapa_seleccion_oferta_ct220(text,jsonb,jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','6a322faf1d1b8519ac8f5326f491589142b4b51334b1d4c9765199e3c582ef3a',
   '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}'),
  ('vec_contratacion_temporal.descartar_seleccion_oferta_ct220(jsonb,text,jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','49da40eac4a0f44a8a9b7c096446b09a74099d3bc3f908be5195a8af0b48b001',
   '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'),
  ('vec_contratacion_temporal.consultar_estado_oferta_expediente_ct225(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','1ee860093ab0ce7e71d4b304a0f0ed6f2eae31f84a4e8952260a088aea50a35f',
   '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}')
 ) AS x(firma,huella,acl) LOOP
 f:=to_regprocedure(r.firma);
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM r.huella
    OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR NOT p.prosecdef OR p.proacl::text IS DISTINCT FROM r.acl
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
 THEN RAISE EXCEPTION 'CT227: postimagen divergente %',r.firma USING ERRCODE='55000'; END IF;
 END LOOP;
 f:=to_regprocedure('vec_contratacion_temporal.preparar_contexto_estado_oferta_ct227(text,text)');
 IF f IS NULL THEN RAISE EXCEPTION 'CT227: preparación ausente' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR NOT p.prosecdef OR p.provolatile<>'s'
    OR p.proacl::text IS DISTINCT FROM
       '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
 THEN RAISE EXCEPTION 'CT227: preparación o ACL divergente' USING ERRCODE='55000'; END IF;
 IF (SELECT x.proacl::text FROM pg_proc x
     WHERE x.oid=to_regprocedure('vec_contratacion_temporal.preparar_contexto_estado_oferta_ct225(text,text)'))
    IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}'
 THEN RAISE EXCEPTION 'CT227: preparación CT225 aún ejecutable fuera del propietario' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
