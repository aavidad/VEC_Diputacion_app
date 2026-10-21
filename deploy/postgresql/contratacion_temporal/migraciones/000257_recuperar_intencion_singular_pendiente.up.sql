\set ON_ERROR_STOP on
-- CT257 / F1-141. Recupera la intención singular pendiente desde O6 bajo
-- la misma sesión, decisión V3 y auditoría que la lectura existente.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000257',0));
DO $pre$
DECLARE f oid; p record;
BEGIN
 f:=to_regprocedure('vec_contratacion_temporal.consultar_estado_oferta_expediente_ct225(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR to_regclass('vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.numero_personas_detalle_rrhh_ct225(text,text,numeric)') IS NULL
    OR f IS NULL
 THEN RAISE EXCEPTION 'CT257: dependencias ausentes' USING ERRCODE='55000'; END IF;
 SELECT pg_get_userbyid(q.proowner) AS dueno,q.prosecdef,q.proacl::text AS acl,
   q.proconfig,encode(sha256(convert_to(pg_get_functiondef(q.oid),'UTF8')),'hex') AS huella
 INTO p FROM pg_proc q WHERE q.oid=f;
 IF p.dueno<>'vec_contratacion_temporal_propietario' OR p.prosecdef IS NOT TRUE
    OR p.acl IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']
    OR p.huella IS DISTINCT FROM '82941407107240e6c27850e86c85b6d8706fe9fea8486f3e23ae207ebb4947f9'
 THEN RAISE EXCEPTION 'CT257: preimagen de lectura incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

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
 -- CT241 / F1-19: la unidad del hijo es la copiada del padre; no autoriza.
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta alta
   WHERE alta.expediente_ref=m->>'expediente_ref' AND alta.derivacion_ref IS NOT NULL) THEN
  RAISE EXCEPTION 'CT241: nombramiento derivado sin autoridad propia' USING ERRCODE='42501'; END IF;
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
 -- El expediente singular conserva su intención O6 pendiente. Se busca después
 -- del consumo V3 y se coteja con la versión actual; el lease y los tokens
 -- internos no cruzan esta lectura.
 IF vec_contratacion_temporal.numero_personas_detalle_rrhh_ct225(
   m->>'organizacion_ref',m->>'expediente_ref',c.version::numeric)=1 THEN
  SELECT * INTO e FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 x
  WHERE x.tipo_seleccion='llamamiento'
    AND x.solicitud_json->>'organizacion_ref'=m->>'organizacion_ref'
    AND x.solicitud_json->>'expediente_ref'=m->>'expediente_ref'
    AND x.situacion IN ('propietaria','indeterminada')
  ORDER BY x.actualizada_en DESC,x.clave_idempotencia DESC
  LIMIT 1 FOR SHARE;
  IF e.clave_idempotencia IS NULL THEN
   RETURN jsonb_build_object('situacion','ausente');
  END IF;
  IF e.solicitud_json->>'version_expediente' IS DISTINCT FROM m->>'version_observada'
     OR e.solicitud_json->>'clave_idempotencia' IS DISTINCT FROM e.clave_idempotencia::text
     OR e.recibo_json IS NOT NULL OR e.artefacto_canonico IS NOT NULL
     OR (e.efecto IS NULL AND (e.ventana_orden_abierta OR e.ventana_llamamiento_abierta))
     OR (e.efecto IS NOT NULL AND NOT e.ventana_orden_abierta)
     OR (e.efecto='preparar_orden' AND e.ventana_llamamiento_abierta)
     OR (e.efecto='solicitar_llamamiento' AND NOT e.ventana_llamamiento_abierta)
     OR (e.situacion='indeterminada' AND e.efecto IS NULL)
  THEN RAISE EXCEPTION 'CT257: intención singular incompatible' USING ERRCODE='55000'; END IF;
  RETURN jsonb_build_object('situacion',e.situacion,'tipo_seleccion','llamamiento',
    'intencion_original',jsonb_build_object(
      'organizacion_ref',e.solicitud_json->'organizacion_ref',
      'expediente_ref',e.solicitud_json->'expediente_ref',
      'version_expediente',e.solicitud_json->'version_expediente',
      'clave_idempotencia',e.solicitud_json->'clave_idempotencia'),
    'estado',jsonb_build_object('efecto',e.efecto,
      'ventana_abierta',e.ventana_orden_abierta OR e.ventana_llamamiento_abierta));
 END IF;
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
DECLARE f oid;p record;
BEGIN
 f:=to_regprocedure('vec_contratacion_temporal.consultar_estado_oferta_expediente_ct225(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 SELECT pg_get_userbyid(q.proowner) AS dueno,q.prosecdef,q.proacl::text AS acl,q.proconfig
 INTO p FROM pg_proc q WHERE q.oid=f;
 IF NOT FOUND OR p.dueno<>'vec_contratacion_temporal_propietario' OR p.prosecdef IS NOT TRUE
    OR p.acl IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']
 THEN RAISE EXCEPTION 'CT257: lectura instalada incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
