\set ON_ERROR_STOP on
-- AUT104: dos proyecciones exactas de la lectura propia y evolución gobernada
-- de la misma concesión, sin acciones, audiencias ni actos nuevos.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000104',0));
DO $pre$ BEGIN
 IF current_user<>'vec_autorizacion_propietario' OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR to_regprocedure('vec_autorizacion.cotejar_campos_preferencias_modalidades_aut104(jsonb,jsonb)') IS NOT NULL
  OR to_regprocedure('vec_autorizacion.estado_rol_candidato_preferencias_aut104(jsonb)') IS NOT NULL
  OR to_regprocedure('vec_autorizacion.delta_modalidades_candidato_aut104(jsonb,jsonb)') IS NOT NULL
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion.cotejar_lectura_preferencias_bolsas_actual_v1(jsonb,text,text)'::regprocedure
  AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='960df3a85600b836c2b28d26b296a329b747489d8ac99cdacc764d3316548720'
  AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef IS TRUE
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s']::text[] AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario}')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion.cotejar_lectura_preferencias_bolsas_actual_interna_v1(jsonb,text,text)'::regprocedure
  AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='2abd58ae19187875ba1bb06349a07146046636a12321ef1c5cfce6afa86292f7'
  AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef IS TRUE
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s']::text[] AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario}')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion.resolver_base_version_rol_bolsa_externa_v1()'::regprocedure
  AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='8ba8bdb4c54e07d79921f0ce0c76df3583874391964993ffab6b6caf245e448e'
  AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef IS TRUE
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on']::text[] AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_admin_version_rol_bolsa_externa_ejecutor=X/vec_autorizacion_propietario}')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion.validar_plan_version_rol_bolsa_externa_v1(jsonb,boolean)'::regprocedure
  AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='a7baee7b6e0f935e3fccc79b91ba865a6db8d04ef3f9b6928f5d3457d6d0c564'
  AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef IS TRUE
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC','row_security=on']::text[] AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario}')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion.canon_version_rol_bolsa_externa_v1(jsonb,text)'::regprocedure
  AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='cbad6f7a0d51035c8fed089de7e7bf0964ecfb728e533d437e2778993b1e8abf'
  AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef IS FALSE
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']::text[] AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario}')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion.aplicar_version_rol_bolsa_externa_v1(boolean,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
  AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='f6e8d25dd778a212f22023bceb39201ebff3171e4a73585ac05178882ad7faba'
  AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef IS TRUE
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC','lock_timeout=2s']::text[] AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario}')
 THEN RAISE EXCEPTION 'AUT104: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_autorizacion.cotejar_campos_preferencias_modalidades_aut104(p_solicitados jsonb,p_concedidos jsonb)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT coalesce(
  (p_solicitados='["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","recibo_ref","registrado_en"]'::jsonb AND p_concedidos IN ('["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","recibo_ref","registrado_en"]'::jsonb,'["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","modalidades_guardadas","modalidades_disponibles","recibo_ref","registrado_en"]'::jsonb))
  OR (p_solicitados='["version","modalidades_guardadas","modalidades_disponibles","recibo_ref","registrado_en"]'::jsonb AND p_concedidos='["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","modalidades_guardadas","modalidades_disponibles","recibo_ref","registrado_en"]'::jsonb),false)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.cotejar_campos_preferencias_modalidades_aut104(jsonb,jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.estado_rol_candidato_preferencias_aut104(p_rol jsonb)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE v integer;acciones text[];resto text[];campos jsonb;ref text;
 base constant text[]:=ARRAY['bolsa.historial_propio.consultar','bolsa.participaciones_propias.consultar','bolsa.participaciones_propias.solicitar_pausa','bolsa.participaciones_propias.solicitar_reactivacion','bolsa.participaciones_propias.responder_llamamiento','bolsa.participaciones_propias.manifestar_disposicion','bolsa.participaciones_propias.confirmar_contacto'];
 pref constant text[]:=ARRAY['bolsa.preferencias_bolsas.consultar_propias','bolsa.preferencias_bolsas.ordenar_propias'];
 inscr constant text[]:=ARRAY['bolsa.inscripcion.convocatorias.listar','bolsa.inscripcion.convocatoria.consultar','bolsa.inscripcion.propias.listar','bolsa.inscripcion.propia.consultar','bolsa.inscripcion.presentar'];
 aleg constant text[]:=ARRAY['bolsa.alegacion.propias.consultar','bolsa.inscripcion.alegacion.presentar'];
BEGIN
 IF jsonb_typeof(p_rol) IS DISTINCT FROM 'object'
 OR p_rol->>'rol_id' IS DISTINCT FROM 'candidato_bolsa_portal_historial_propio_desarrollo'
 OR jsonb_typeof(p_rol->'version') IS DISTINCT FROM 'number'
 OR p_rol->>'version' !~ '^[1-5]$'
 OR jsonb_typeof(p_rol->'concesiones') IS DISTINCT FROM 'array' THEN RETURN NULL; END IF;
 v:=(p_rol->>'version')::integer;
 SELECT array_agg(e->>'accion' ORDER BY n) INTO acciones
 FROM jsonb_array_elements(p_rol->'concesiones') WITH ORDINALITY x(e,n);
 -- Las siete concesiones de la v1 se comparan como conjunto exacto: la v1
 -- publicada empieza por participaciones_propias.consultar. Lo añadido después
 -- conserva el orden de las altas gobernadas.
 IF coalesce(cardinality(acciones),0)<7
 OR (SELECT array_agg(x ORDER BY x) FROM unnest(acciones[1:7]) x)
  IS DISTINCT FROM (SELECT array_agg(x ORDER BY x) FROM unnest(base) x) THEN RETURN NULL; END IF;
 resto:=acciones[8:];
 IF resto=ARRAY[]::text[] AND v=1 THEN RETURN 'base'; END IF;
 IF resto=inscr AND v=2 THEN RETURN 'i'; END IF;
 IF NOT (resto=pref OR resto=pref||inscr OR resto=inscr||pref
  OR resto=pref||inscr||aleg OR resto=inscr||pref||aleg) THEN RETURN NULL; END IF;
 SELECT e->'campos_permitidos' INTO campos FROM jsonb_array_elements(p_rol->'concesiones') e
 WHERE e->>'accion'='bolsa.preferencias_bolsas.consultar_propias';
 IF campos IS DISTINCT FROM '["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","recibo_ref","registrado_en"]'::jsonb AND campos IS DISTINCT FROM '["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","modalidades_guardadas","modalidades_disponibles","recibo_ref","registrado_en"]'::jsonb THEN RETURN NULL; END IF;
 ref:=CASE WHEN campos='["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","recibo_ref","registrado_en"]'::jsonb THEN '8' ELSE '10' END;
 IF resto=pref AND ((v=2) OR (v=3 AND ref='10')) THEN RETURN 'p'||ref; END IF;
 IF resto=pref||inscr AND ((v=3) OR (v=4 AND ref='10')) THEN RETURN 'pi'||ref; END IF;
 IF resto=inscr||pref AND ((v=3) OR (v=4 AND ref='10')) THEN RETURN 'ip'||ref; END IF;
 IF resto=pref||inscr||aleg AND ((v=4) OR (v=5 AND ref='10')) THEN RETURN 'pia'||ref; END IF;
 IF resto=inscr||pref||aleg AND ((v=4) OR (v=5 AND ref='10')) THEN RETURN 'ipa'||ref; END IF;
 RETURN NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.estado_rol_candidato_preferencias_aut104(jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.delta_modalidades_candidato_aut104(p_base jsonb,p_nueva jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE old_arr jsonb;new_arr jsonb;indice integer;esperado jsonb;
BEGIN
 IF coalesce(vec_autorizacion.estado_rol_candidato_preferencias_aut104(p_base) IN
  ('p8','pi8','ip8','pia8','ipa8'),false) IS NOT TRUE THEN RETURN false; END IF;
 old_arr:=p_base->'concesiones';new_arr:=p_nueva->'concesiones';
 SELECT (n-1)::integer INTO indice FROM jsonb_array_elements(old_arr) WITH ORDINALITY x(e,n)
 WHERE e->>'accion'='bolsa.preferencias_bolsas.consultar_propias';
 IF indice IS NULL OR jsonb_typeof(new_arr) IS DISTINCT FROM 'array' THEN RETURN false; END IF;
 esperado:=jsonb_set(old_arr,ARRAY[indice::text],
  jsonb_set(old_arr->indice,'{campos_permitidos}','["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","modalidades_guardadas","modalidades_disponibles","recibo_ref","registrado_en"]'::jsonb,false),false);
 RETURN coalesce(new_arr IS NOT DISTINCT FROM esperado
  AND vec_autorizacion.estado_rol_candidato_preferencias_aut104(p_nueva) IN
   ('p10','pi10','ip10','pia10','ipa10'),false);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.delta_modalidades_candidato_aut104(jsonb,jsonb) FROM PUBLIC;

-- Definición final literal de cotejar_lectura_preferencias_bolsas_actual_v1.
CREATE OR REPLACE FUNCTION vec_autorizacion.cotejar_lectura_preferencias_bolsas_actual_v1(p_orden jsonb, p_principal text, p_perfil text)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
AS $function$
DECLARE a record;r record;c record;catalogo record;politicas jsonb;ahora timestamptz(6);
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR p_orden->>'accion' IS DISTINCT FROM 'bolsa.preferencias_bolsas.consultar_propias'
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR p_orden->>'tipo_recurso' IS DISTINCT FROM 'participaciones_candidato'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'gestion_preferencias_bolsas_propias'
 OR p_orden->>'canal' IS DISTINCT FROM 'externa_personal'
 OR p_orden->>'recurso_ref' IS DISTINCT FROM 'mi-bolsa:'||(p_orden->>'candidato_ref')
 OR p_orden->>'candidato_ref' !~ '^can_[A-Za-z0-9_-]{22,128}$'
 OR vec_autorizacion.cotejar_campos_preferencias_modalidades_aut104(
  p_orden->'campos_permitidos','["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","modalidades_guardadas","modalidades_disponibles","recibo_ref","registrado_en"]'::jsonb) IS NOT TRUE
 OR jsonb_typeof(p_orden->'politicas_evaluadas') IS DISTINCT FROM 'array'
 THEN RETURN false; END IF;
 ahora:=clock_timestamp();
 IF (p_orden->>'emitida_en')::timestamptz>ahora
 OR (p_orden->>'valida_hasta')::timestamptz<=ahora
 OR (p_orden->>'valida_hasta')::timestamptz>ahora+interval '30 seconds'
 THEN RETURN false; END IF;
 SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil_actual_externa ap
 JOIN vec_autorizacion.asignacion_perfil_externa x ON x.perfil_activo_ref=ap.perfil_activo_ref
  AND x.asignacion_ref=ap.asignacion_ref
 WHERE ap.perfil_activo_ref=p_perfil FOR SHARE OF ap,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO r FROM vec_autorizacion.version_rol x
 WHERE x.version_rol_ref=a.version_rol_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO c FROM vec_autorizacion.control_vigencia_version_rol_actual cp
 JOIN vec_autorizacion.control_vigencia_version_rol x
  ON x.version_rol_ref=cp.version_rol_ref AND x.revision=cp.revision
 WHERE cp.version_rol_ref=r.version_rol_ref FOR SHARE OF cp,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT * INTO catalogo FROM vec_autorizacion.control_catalogo_politicas
 WHERE control_id IS TRUE FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT COALESCE(jsonb_agg(jsonb_build_object('referencia',p.politica_ref,
  'huella_sha256',p.huella_sha256) ORDER BY p.politica_ref),'[]'::jsonb)
 INTO politicas FROM vec_autorizacion.politica_restrictiva_actual pa
 JOIN vec_autorizacion.politica_restrictiva p
  ON p.politica_id=pa.politica_id AND p.politica_ref=pa.politica_ref;
 RETURN a.principal_id IS NOT DISTINCT FROM p_principal
  AND a.perfil_activo_ref IS NOT DISTINCT FROM p_perfil
  AND a.asignacion_ref IS NOT DISTINCT FROM p_orden->>'asignacion_ref'
  AND a.version IS NOT DISTINCT FROM (p_orden->>'asignacion_version')::numeric
  AND a.huella_sha256 IS NOT DISTINCT FROM p_orden->>'asignacion_huella_sha256'
  AND a.documento->>'estado' IS NOT DISTINCT FROM 'activa'
  AND ahora>=(a.documento->>'vigente_desde')::timestamptz
  AND ahora<(a.documento->>'vigente_hasta')::timestamptz
  AND r.rol_id IS NOT DISTINCT FROM 'candidato_bolsa_portal_historial_propio_desarrollo'
  AND r.version_rol_ref IS NOT DISTINCT FROM p_orden->>'version_rol_ref'
  AND r.version IS NOT DISTINCT FROM (p_orden->>'version_rol_version')::numeric
  AND r.huella_sha256 IS NOT DISTINCT FROM p_orden->>'version_rol_huella_sha256'
  AND r.documento->>'estado' IS NOT DISTINCT FROM 'publicada'
  AND c.version_rol_ref IS NOT DISTINCT FROM p_orden->>'control_version_rol_ref'
  AND c.revision IS NOT DISTINCT FROM (p_orden->>'revision_control_version_rol')::numeric
  AND c.huella_sha256 IS NOT DISTINCT FROM p_orden->>'control_version_rol_huella_sha256'
  AND c.estado IS NOT DISTINCT FROM 'habilitada'
  AND catalogo.revision IS NOT DISTINCT FROM (p_orden->>'revision_catalogo_politicas')::numeric
  AND catalogo.huella_sha256 IS NOT DISTINCT FROM p_orden->>'catalogo_politicas_huella_sha256'
  AND politicas IS NOT DISTINCT FROM p_orden->'politicas_evaluadas'
  AND CASE WHEN jsonb_typeof(a.documento->'ambitos')='array' THEN
   jsonb_array_length(a.documento->'ambitos')=1
   AND vec_autorizacion.ambitos_positivos_validos(a.documento) IS TRUE
   AND a.documento->'ambitos' @> jsonb_build_array(
    jsonb_build_object('clave','candidato_ref',
     'valores',jsonb_build_array(p_orden->>'candidato_ref')))
   ELSE false END
  AND (SELECT count(*) FROM jsonb_array_elements(r.documento->'concesiones') concesion
   WHERE concesion->>'accion'='bolsa.preferencias_bolsas.consultar_propias')=1
  AND EXISTS(SELECT 1 FROM jsonb_array_elements(r.documento->'concesiones') concesion
   WHERE concesion->>'accion'='bolsa.preferencias_bolsas.consultar_propias'
    AND concesion->>'modulo_id'='bolsa'
    AND concesion->>'tipo_recurso'='participaciones_candidato'
    AND concesion->'finalidades'='["gestion_preferencias_bolsas_propias"]'::jsonb
    AND vec_autorizacion.cotejar_campos_preferencias_modalidades_aut104(
     p_orden->'campos_permitidos',concesion->'campos_permitidos') IS TRUE
    AND concesion->>'garantia_minima'='alto'
    AND coalesce(concesion->'obligaciones','[]'::jsonb)
      IN ('[]'::jsonb,'["auditar"]'::jsonb));
EXCEPTION WHEN data_exception OR invalid_text_representation OR numeric_value_out_of_range THEN
 RETURN false;
END $function$;

-- Definición final literal de cotejar_lectura_preferencias_bolsas_actual_interna_v1.
CREATE OR REPLACE FUNCTION vec_autorizacion.cotejar_lectura_preferencias_bolsas_actual_interna_v1(p_orden jsonb, p_principal text, p_perfil text)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
AS $function$
DECLARE a record;r record;c record;catalogo record;politicas jsonb;ahora timestamptz(6);
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR p_orden->>'accion' IS DISTINCT FROM 'bolsa.preferencias_bolsas.consultar_propias'
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR p_orden->>'tipo_recurso' IS DISTINCT FROM 'participaciones_candidato'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'gestion_preferencias_bolsas_propias'
 OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_orden->>'recurso_ref' IS DISTINCT FROM 'mi-bolsa:'||(p_orden->>'candidato_ref')
 OR p_orden->>'candidato_ref' !~ '^can_[A-Za-z0-9_-]{22,128}$'
 OR vec_autorizacion.cotejar_campos_preferencias_modalidades_aut104(
  p_orden->'campos_permitidos','["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","modalidades_guardadas","modalidades_disponibles","recibo_ref","registrado_en"]'::jsonb) IS NOT TRUE
 OR jsonb_typeof(p_orden->'politicas_evaluadas') IS DISTINCT FROM 'array'
 THEN RETURN false; END IF;
 ahora:=clock_timestamp();
 -- Una fecha ausente da NULL y también se rechaza.
 IF ((p_orden->>'emitida_en')::timestamptz>ahora
 OR (p_orden->>'valida_hasta')::timestamptz<=ahora
 OR (p_orden->>'valida_hasta')::timestamptz>ahora+interval '30 seconds') IS NOT FALSE
 THEN RETURN false; END IF;
 SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil_actual ap
 JOIN vec_autorizacion.asignacion_perfil x ON x.perfil_activo_ref=ap.perfil_activo_ref
  AND x.asignacion_ref=ap.asignacion_ref
 WHERE ap.perfil_activo_ref=p_perfil FOR SHARE OF ap,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO r FROM vec_autorizacion.version_rol x
 WHERE x.version_rol_ref=a.version_rol_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO c FROM vec_autorizacion.control_vigencia_version_rol_actual cp
 JOIN vec_autorizacion.control_vigencia_version_rol x
  ON x.version_rol_ref=cp.version_rol_ref AND x.revision=cp.revision
 WHERE cp.version_rol_ref=r.version_rol_ref FOR SHARE OF cp,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT * INTO catalogo FROM vec_autorizacion.control_catalogo_politicas
 WHERE control_id IS TRUE FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT COALESCE(jsonb_agg(jsonb_build_object('referencia',p.politica_ref,
  'huella_sha256',p.huella_sha256) ORDER BY p.politica_ref),'[]'::jsonb)
 INTO politicas FROM vec_autorizacion.politica_restrictiva_actual pa
 JOIN vec_autorizacion.politica_restrictiva p
  ON p.politica_id=pa.politica_id AND p.politica_ref=pa.politica_ref;
 RETURN a.principal_id IS NOT DISTINCT FROM p_principal
  AND a.perfil_activo_ref IS NOT DISTINCT FROM p_perfil
  AND a.asignacion_ref IS NOT DISTINCT FROM p_orden->>'asignacion_ref'
  AND a.version IS NOT DISTINCT FROM (p_orden->>'asignacion_version')::numeric
  AND a.huella_sha256 IS NOT DISTINCT FROM p_orden->>'asignacion_huella_sha256'
  AND a.documento->>'estado' IS NOT DISTINCT FROM 'activa'
  AND ahora>=(a.documento->>'vigente_desde')::timestamptz
  AND ahora<(a.documento->>'vigente_hasta')::timestamptz
  AND r.rol_id IS NOT DISTINCT FROM 'candidato_bolsa_portal_historial_propio_desarrollo'
  AND r.version_rol_ref IS NOT DISTINCT FROM p_orden->>'version_rol_ref'
  AND r.version IS NOT DISTINCT FROM (p_orden->>'version_rol_version')::numeric
  AND r.huella_sha256 IS NOT DISTINCT FROM p_orden->>'version_rol_huella_sha256'
  AND r.documento->>'estado' IS NOT DISTINCT FROM 'publicada'
  AND c.version_rol_ref IS NOT DISTINCT FROM p_orden->>'control_version_rol_ref'
  AND c.revision IS NOT DISTINCT FROM (p_orden->>'revision_control_version_rol')::numeric
  AND c.huella_sha256 IS NOT DISTINCT FROM p_orden->>'control_version_rol_huella_sha256'
  AND c.estado IS NOT DISTINCT FROM 'habilitada'
  AND catalogo.revision IS NOT DISTINCT FROM (p_orden->>'revision_catalogo_politicas')::numeric
  AND catalogo.huella_sha256 IS NOT DISTINCT FROM p_orden->>'catalogo_politicas_huella_sha256'
  AND politicas IS NOT DISTINCT FROM p_orden->'politicas_evaluadas'
  AND CASE WHEN jsonb_typeof(a.documento->'ambitos')='array' THEN
   jsonb_array_length(a.documento->'ambitos')=1
   AND vec_autorizacion.ambitos_positivos_validos(a.documento) IS TRUE
   AND a.documento->'ambitos' @> jsonb_build_array(
    jsonb_build_object('clave','candidato_ref',
     'valores',jsonb_build_array(p_orden->>'candidato_ref')))
   ELSE false END
  AND (SELECT count(*) FROM jsonb_array_elements(r.documento->'concesiones') concesion
   WHERE concesion->>'accion'='bolsa.preferencias_bolsas.consultar_propias')=1
  AND EXISTS(SELECT 1 FROM jsonb_array_elements(r.documento->'concesiones') concesion
   WHERE concesion->>'accion'='bolsa.preferencias_bolsas.consultar_propias'
    AND concesion->>'modulo_id'='bolsa'
    AND concesion->>'tipo_recurso'='participaciones_candidato'
    AND concesion->'finalidades'='["gestion_preferencias_bolsas_propias"]'::jsonb
    AND vec_autorizacion.cotejar_campos_preferencias_modalidades_aut104(
     p_orden->'campos_permitidos',concesion->'campos_permitidos') IS TRUE
    AND concesion->>'garantia_minima'='alto'
    AND coalesce(concesion->'obligaciones','[]'::jsonb)
      IN ('[]'::jsonb,'["auditar"]'::jsonb));
EXCEPTION WHEN data_exception OR invalid_text_representation OR numeric_value_out_of_range THEN
 RETURN false;
END $function$;

-- Definición final literal de resolver_base_version_rol_bolsa_externa_v1.
CREATE OR REPLACE FUNCTION vec_autorizacion.resolver_base_version_rol_bolsa_externa_v1()
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
AS $function$
DECLARE r record;c record;
BEGIN
 IF current_user<>'vec_autorizacion_propietario'
 OR current_setting('role')<>'none'
 OR vec_autorizacion_atestada_v3.login_version_rol_bolsa_externa_valido_v1() IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT77: lectura base denegada' USING ERRCODE='42501';END IF;
 SELECT * INTO STRICT r FROM vec_autorizacion.version_rol
 WHERE rol_id='candidato_bolsa_portal_historial_propio_desarrollo'
 ORDER BY version DESC LIMIT 1 FOR SHARE;
 SELECT h.* INTO STRICT c FROM vec_autorizacion.control_vigencia_version_rol_actual p
 JOIN vec_autorizacion.control_vigencia_version_rol h USING(version_rol_ref,revision)
 WHERE p.version_rol_ref=r.version_rol_ref FOR SHARE OF p,h;
 IF r.rol_id<>'candidato_bolsa_portal_historial_propio_desarrollo'
 OR r.documento->>'estado'<>'publicada' OR c.estado<>'habilitada'
 OR r.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(
  vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
 OR c.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(
  vec_autorizacion.canon_control_rol_admin_v1(c.documento),'UTF8')),'hex')
 OR vec_autorizacion.estado_rol_candidato_preferencias_aut104(r.documento) IS NULL THEN RAISE EXCEPTION 'AUT77: base externa divergente' USING ERRCODE='40001';END IF;
 RETURN jsonb_build_object('rol',vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(r.documento),
  'control_vigencia',c.documento,'tipo_perfil','administrable');
END $function$;

-- Definición final literal de validar_plan_version_rol_bolsa_externa_v1.
CREATE OR REPLACE FUNCTION vec_autorizacion.validar_plan_version_rol_bolsa_externa_v1(p jsonb, p_primera boolean)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET "TimeZone" TO 'UTC'
 SET row_security TO 'on'
AS $function$
DECLARE catalogo jsonb;entrada jsonb;seleccion jsonb;base jsonb;nueva jsonb;
 cabeza record;rol record;control record;puntero_control record;asignacion record;
 t jsonb;original jsonb;vistas text[]:=ARRAY[]::text[];
 acciones text[]:=ARRAY['bolsa.preferencias_bolsas.consultar_propias',
  'bolsa.preferencias_bolsas.ordenar_propias'];
 tipos text[]:=ARRAY['participaciones_candidato','participaciones_candidato'];
 finalidades text[]:=ARRAY['gestion_preferencias_bolsas_propias',
  'gestion_preferencias_bolsas_propias'];
 hojas text[];normalizadas jsonb:='[]'::jsonb;concesion jsonb;ahora timestamptz;
 i integer;campos_actuales jsonb;campos_esperados jsonb;ambitos jsonb;
 limite_entradas timestamptz:='infinity'::timestamptz;
 origen_inscripcion boolean:=false;origen_alegacion boolean:=false;
 origen_modalidades boolean:=false;origen_preferencias boolean:=false;estado_base text;
BEGIN
 PERFORM vec_autorizacion.canon_version_rol_bolsa_externa_v1(p,'plan');
 IF p_primera IS NULL OR p->>'operacion' IS DISTINCT FROM 'versionar'
 OR p#>>'{definicion_nueva,rol_id}' IS DISTINCT FROM 'candidato_bolsa_portal_historial_propio_desarrollo'
 OR p#>>'{base,rol,rol_id}' IS DISTINCT FROM p#>>'{definicion_nueva,rol_id}'
 OR (p#>>'{definicion_nueva,version}')::bigint<>(p#>>'{base,rol,version}')::bigint+1
 OR p->>'version_rol_objetivo_ref' IS DISTINCT FROM
  'rol:'||(p#>>'{definicion_nueva,rol_id}')||':v'||(p#>>'{definicion_nueva,version}')
 OR p#>>'{base,tipo_perfil}' IS DISTINCT FROM 'administrable'
 OR jsonb_array_length(p->'selecciones') NOT IN (2,5)
 THEN RAISE EXCEPTION 'AUT77: version o base invalida' USING ERRCODE='22023';END IF;
 base:=p->'base';nueva:=p->'definicion_nueva';ahora:=clock_timestamp();
 estado_base:=vec_autorizacion.estado_rol_candidato_preferencias_aut104(base->'rol');
 origen_modalidades:=jsonb_array_length(p->'selecciones')=2
  AND jsonb_array_length(nueva->'concesiones')=jsonb_array_length(base#>'{rol,concesiones}')
  AND estado_base IN ('p8','pi8','ip8','pia8','ipa8');
 origen_inscripcion:=jsonb_array_length(p->'selecciones')=5
  AND estado_base IN ('base','p8','p10');
 origen_alegacion:=jsonb_array_length(p->'selecciones')=2
  AND jsonb_array_length(nueva->'concesiones')=jsonb_array_length(base#>'{rol,concesiones}')+2
  AND estado_base IN ('pi8','pi10','ip8','ip10');
 origen_preferencias:=jsonb_array_length(p->'selecciones')=2
  AND jsonb_array_length(nueva->'concesiones')=jsonb_array_length(base#>'{rol,concesiones}')+2
  AND estado_base IN ('base','i');
 IF estado_base IS NULL OR coalesce(
  origen_modalidades OR origen_inscripcion OR origen_alegacion OR origen_preferencias,false) IS NOT TRUE THEN
  RAISE EXCEPTION 'AUT104: transición de rol no admitida' USING ERRCODE='42501'; END IF;
 IF origen_inscripcion THEN
  acciones:=ARRAY['bolsa.inscripcion.convocatorias.listar','bolsa.inscripcion.convocatoria.consultar',
   'bolsa.inscripcion.propias.listar','bolsa.inscripcion.propia.consultar','bolsa.inscripcion.presentar'];
  tipos:=ARRAY['convocatoria_inscripcion','convocatoria_inscripcion','solicitud_inscripcion',
   'solicitud_inscripcion','inscripcion_convocatoria'];
  finalidades:=ARRAY['consulta_convocatoria_abierta','consulta_convocatoria_abierta',
   'consulta_inscripcion_propia','consulta_inscripcion_propia','presentar_inscripcion'];
 ELSIF origen_alegacion THEN
  acciones:=ARRAY['bolsa.alegacion.propias.consultar','bolsa.inscripcion.alegacion.presentar'];
  tipos:=ARRAY['alegaciones_propias','alegacion_bolsa'];
  finalidades:=ARRAY['consulta_alegaciones_propias','presentacion_alegacion_propia'];
 END IF;
 IF p_primera THEN
  SELECT * INTO cabeza FROM vec_autorizacion.cabeza_catalogo_acciones_admin_v1
   WHERE catalogo_ref=p->>'catalogo_ref' FOR SHARE;
  IF NOT FOUND OR cabeza.version::text IS DISTINCT FROM p->>'catalogo_version'
  OR cabeza.huella_sha256 IS DISTINCT FROM p->>'catalogo_huella_sha256'
  THEN RAISE EXCEPTION 'AUT77: cabeza de catalogo divergente' USING ERRCODE='40001';END IF;
 END IF;
 -- El catálogo gobernado acredita las dos entradas exactas.
 SELECT convert_from(x.catalogo_canon,'UTF8')::jsonb INTO STRICT catalogo
 FROM vec_autorizacion.resolver_catalogo_acciones_administracion_v2(
  p->>'catalogo_ref',(p->>'catalogo_version')::integer,p->>'catalogo_huella_sha256') x;
 IF p_primera AND (ahora<(catalogo->>'vigente_desde')::timestamptz
 OR (catalogo->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
  AND ahora>=(catalogo->>'vigente_hasta')::timestamptz))
 THEN RAISE EXCEPTION 'AUT77: catalogo no vigente' USING ERRCODE='42501';END IF;
 FOR i IN 0..cardinality(acciones)-1 LOOP
  seleccion:=p->'selecciones'->i;
  PERFORM vec_autorizacion.canon_gobierno_rol_nuevo_v1(seleccion,'seleccion');
  SELECT value INTO entrada FROM jsonb_array_elements(catalogo->'entradas')
   WHERE value->>'referencia'=seleccion->>'entrada_ref';
  IF NOT FOUND THEN RAISE EXCEPTION 'AUT77: descriptor ausente' USING ERRCODE='42501';END IF;
  concesion:=vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(
   jsonb_build_object('concesiones',jsonb_build_array(entrada->'concesion')))#>'{concesiones,0}';
  IF entrada->>'version' IS DISTINCT FROM seleccion->>'entrada_version'
  OR encode(sha256(convert_to(vec_autorizacion.canon_gobierno_rol_nuevo_v1(entrada,'entrada'),'UTF8')),'hex')
    IS DISTINCT FROM seleccion->>'entrada_huella_sha256'
  OR (origen_inscripcion AND entrada->>'fuente_ref' IS DISTINCT FROM 'fuente:bolsa:inscripcion_recorrido_20261013')
  OR (origen_inscripcion AND entrada->>'fuente_version' IS DISTINCT FROM '1')
  OR (origen_inscripcion AND entrada->>'fuente_huella_sha256' IS DISTINCT FROM '07bf028039044c91a23f09a2d5d5fe0d2af254e2fe85aa7fbbacea7c473487fe')
  OR (origen_alegacion AND entrada->>'fuente_ref' IS DISTINCT FROM 'fuente:bolsa:alegaciones_propias_v1')
  OR (origen_alegacion AND entrada->>'fuente_version' IS DISTINCT FROM '1')
  OR (origen_alegacion AND entrada->>'fuente_huella_sha256' IS DISTINCT FROM '5b9f0fc2d99f4c2cf7b08829bb56bcddd5262c5cb44cc797b48a3f1f072a1060')
  OR entrada#>>'{concesion,accion}' IS DISTINCT FROM acciones[i+1]
  OR entrada#>>'{concesion,modulo_id}' IS DISTINCT FROM 'bolsa'
  OR entrada#>>'{concesion,tipo_recurso}' IS DISTINCT FROM tipos[i+1]
  OR entrada#>'{concesion,finalidades}' IS DISTINCT FROM jsonb_build_array(finalidades[i+1])
  OR entrada#>>'{concesion,garantia_minima}' IS DISTINCT FROM 'alto'
  OR NOT (coalesce(entrada#>'{concesion,obligaciones}','[]'::jsonb)='[]'::jsonb
    OR (NOT origen_inscripcion AND i=0 AND entrada#>'{concesion,obligaciones}'='["auditar"]'::jsonb))
  OR entrada->'dimensiones_ambito' IS DISTINCT FROM '["candidato_ref"]'::jsonb
  OR entrada->>'clase_control' IS DISTINCT FROM 'ordinario'
  OR (NOT origen_modalidades AND concesion IS DISTINCT FROM nueva->'concesiones'->
    (jsonb_array_length(base#>'{rol,concesiones}')+i))
  OR (origen_modalidades AND concesion IS DISTINCT FROM nueva->'concesiones'->
    (SELECT (n-1)::integer FROM jsonb_array_elements(base#>'{rol,concesiones}') WITH ORDINALITY x(e,n)
     WHERE e->>'accion'=acciones[i+1]))
  THEN RAISE EXCEPTION 'AUT77: descriptor o concesion divergente %',acciones[i+1] USING ERRCODE='42501';END IF;
  IF origen_inscripcion THEN
   hojas:=CASE WHEN i=0 THEN ARRAY['convocatorias[].catalogo_version','convocatorias[].categorias[].categoria','convocatorias[].categorias[].categoria_ref','convocatorias[].convocatoria_ref','convocatorias[].estado_solicitud_propia','convocatorias[].impedimento_etiqueta','convocatorias[].numero_categorias','convocatorias[].plazo_fin','convocatorias[].plazo_inicio','convocatorias[].puede_iniciar','convocatorias[].requisitos[].codigo','convocatorias[].requisitos[].descripcion','convocatorias[].requisitos[].estado','convocatorias[].requisitos[].hito_cumplimiento','convocatorias[].requisitos[].hito_etiqueta','convocatorias[].requisitos[].hito_fecha','convocatorias[].requisitos[].motivo_codigo','convocatorias[].requisitos[].motivo_etiqueta','convocatorias[].requisitos[].obligatorio','convocatorias[].requisitos[].procedencia_ref','convocatorias[].requisitos_resumen','convocatorias[].solicitud_ref','convocatorias[].titulo','cursor_siguiente','total']::text[] WHEN i=1 THEN ARRAY['catalogo_version','categorias[].categoria','categorias[].categoria_ref','convocatoria_ref','estado_solicitud_propia','impedimento_etiqueta','numero_categorias','plazo_fin','plazo_inicio','puede_iniciar','requisitos[].codigo','requisitos[].descripcion','requisitos[].estado','requisitos[].hito_cumplimiento','requisitos[].hito_etiqueta','requisitos[].hito_fecha','requisitos[].motivo_codigo','requisitos[].motivo_etiqueta','requisitos[].obligatorio','requisitos[].procedencia_ref','requisitos_resumen','solicitud_ref','titulo']::text[] WHEN i=2 THEN ARRAY['cursor_siguiente','solicitudes[].bases_ref','solicitudes[].bolsa_ref','solicitudes[].catalogo_version','solicitudes[].categoria','solicitudes[].categoria_ref','solicitudes[].convocatoria_ref','solicitudes[].convocatoria_titulo','solicitudes[].decidida_en','solicitudes[].decision_ref','solicitudes[].declaracion_ref','solicitudes[].estado','solicitudes[].motivo_codigo','solicitudes[].motivo_etiqueta','solicitudes[].participacion_ref','solicitudes[].plazo_fin','solicitudes[].plazo_inicio','solicitudes[].recibo_ref','solicitudes[].registrada_en','solicitudes[].requisitos[].codigo','solicitudes[].requisitos[].descripcion','solicitudes[].requisitos[].estado','solicitudes[].requisitos[].hito_cumplimiento','solicitudes[].requisitos[].hito_etiqueta','solicitudes[].requisitos[].hito_fecha','solicitudes[].requisitos[].motivo_codigo','solicitudes[].requisitos[].motivo_etiqueta','solicitudes[].requisitos[].obligatorio','solicitudes[].requisitos[].procedencia_ref','solicitudes[].solicitud_ref','solicitudes[].version','total']::text[] WHEN i=3 THEN ARRAY['bases_ref','bolsa_ref','catalogo_version','categoria','categoria_ref','convocatoria_ref','convocatoria_titulo','decidida_en','decision_ref','declaracion_ref','estado','motivo_codigo','motivo_etiqueta','participacion_ref','plazo_fin','plazo_inicio','recibo_ref','registrada_en','requisitos[].codigo','requisitos[].descripcion','requisitos[].estado','requisitos[].hito_cumplimiento','requisitos[].hito_etiqueta','requisitos[].hito_fecha','requisitos[].motivo_codigo','requisitos[].motivo_etiqueta','requisitos[].obligatorio','requisitos[].procedencia_ref','solicitud_ref','version']::text[] WHEN i=4 THEN ARRAY[]::text[] END;
  ELSIF origen_alegacion THEN
  hojas:=CASE WHEN i=0 THEN ARRAY['alegaciones[].alegacion_ref','alegaciones[].auditoria_ref','alegaciones[].estado','alegaciones[].fundamento','alegaciones[].objeto.acto_ref','alegaciones[].objeto.participacion_ref','alegaciones[].objeto.solicitud_ref','alegaciones[].objeto.tipo','alegaciones[].objeto.version_acto','alegaciones[].presentada_en','alegaciones[].recibo_ref','cursor_siguiente','objetos[].acto_ref','objetos[].bolsa_ref','objetos[].convocatoria_ref','objetos[].estado','objetos[].ocurrido_en','objetos[].participacion_ref','objetos[].solicitud_ref','objetos[].tipo','objetos[].titulo','objetos[].version_acto','total']::text[] ELSE ARRAY[]::text[] END;
  ELSE
  hojas:=CASE WHEN i=0 AND origen_modalidades THEN ARRAY['version','orden_guardado','bolsas_actuales',
   'bolsas_nuevas','bolsas_retiradas','revision_requerida','modalidades_guardadas',
   'modalidades_disponibles','recibo_ref','registrado_en']::text[]
   WHEN i=0 THEN ARRAY['version','orden_guardado','bolsas_actuales',
   'bolsas_nuevas','bolsas_retiradas','revision_requerida','recibo_ref','registrado_en']::text[]
   ELSE ARRAY[]::text[] END;
  END IF;
  IF (origen_preferencias AND i=0 AND
    coalesce(entrada#>'{concesion,campos_permitidos}','[]'::jsonb) NOT IN
     ('["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","recibo_ref","registrado_en"]'::jsonb,'["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","modalidades_guardadas","modalidades_disponibles","recibo_ref","registrado_en"]'::jsonb))
  OR (NOT (origen_preferencias AND i=0) AND
    coalesce(entrada#>'{concesion,campos_permitidos}','[]'::jsonb) IS DISTINCT FROM to_jsonb(hojas))
  THEN RAISE EXCEPTION 'AUT77: campos fuera de contrato %',acciones[i+1] USING ERRCODE='42501';END IF;
  IF p_primera AND (ahora<(entrada->>'vigente_desde')::timestamptz
   OR (entrada->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
    AND ahora>=(entrada->>'vigente_hasta')::timestamptz))
  THEN RAISE EXCEPTION 'AUT77: descriptor caducado %',acciones[i+1] USING ERRCODE='42501';END IF;
  IF entrada->>'vigente_hasta'<>'0001-01-01T00:00:00Z' THEN
   limite_entradas:=LEAST(limite_entradas,(entrada->>'vigente_hasta')::timestamptz);
  END IF;
  normalizadas:=normalizadas||jsonb_build_array(concesion);
 END LOOP;
 IF catalogo->>'vigente_hasta'<>'0001-01-01T00:00:00Z' THEN
  limite_entradas:=LEAST(limite_entradas,(catalogo->>'vigente_hasta')::timestamptz);
 END IF;
 SELECT * INTO rol FROM vec_autorizacion.version_rol
  WHERE version_rol_ref='rol:candidato_bolsa_portal_historial_propio_desarrollo:v'||(base#>>'{rol,version}')
  FOR SHARE;
 IF NOT FOUND OR rol.rol_id IS DISTINCT FROM 'candidato_bolsa_portal_historial_propio_desarrollo'
 OR rol.documento->>'estado' IS DISTINCT FROM 'publicada'
 -- AUT94: la base v1 heredada sólo se admite con su documento exacto.
 OR (rol.version=1 AND rol.documento->>'nombre' IS DISTINCT FROM 'areaPersonal.miBolsa.rolPortal'
  AND (rol.documento->>'nombre'='Consulta y acciones propias de bolsa en desarrollo'
   AND rol.huella_sha256='11044c697a11418870dc21794a9cb570505e017090ca8d767ad316851ec34b15') IS NOT TRUE)
 OR (rol.version=1 AND rol.documento->>'publicada_por' IS DISTINCT FROM 'seguridad:desarrollo:no-autoritativa')
 OR rol.version::text IS DISTINCT FROM (base#>>'{rol,version}')
 OR rol.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(rol.documento),'UTF8')),'hex')
 OR vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(rol.documento) IS DISTINCT FROM base->'rol'
 OR (p_primera AND EXISTS(SELECT 1 FROM vec_autorizacion.version_rol z
   WHERE z.rol_id=rol.rol_id AND z.version>rol.version))
 THEN RAISE EXCEPTION 'AUT77: preimagen del rol divergente' USING ERRCODE='40001';END IF;
 SELECT x.* INTO control FROM vec_autorizacion.control_vigencia_version_rol x
 WHERE x.version_rol_ref=rol.version_rol_ref
 AND x.revision=(base#>>'{control_vigencia,revision}')::numeric FOR SHARE;
 IF NOT FOUND OR control.estado IS DISTINCT FROM 'habilitada'
 OR control.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(control.documento),'UTF8')),'hex')
 OR control.documento IS DISTINCT FROM base->'control_vigencia'
 THEN RAISE EXCEPTION 'AUT77: control o perfil catalogado divergente' USING ERRCODE='40001';END IF;
 -- El cierre nuevo retiene el puntero base hasta COMMIT. Una retirada o
 -- avance concurrente no puede intercalarse entre cotejo y efecto.
 IF p_primera THEN
  SELECT q.revision INTO puntero_control
  FROM vec_autorizacion.control_vigencia_version_rol_actual q
  WHERE q.version_rol_ref=control.version_rol_ref FOR SHARE;
  IF NOT FOUND OR puntero_control.revision IS DISTINCT FROM control.revision
  THEN RAISE EXCEPTION 'AUT77: control base ya no actual' USING ERRCODE='40001';END IF;
 END IF;
 IF nueva->>'nombre' IS DISTINCT FROM base#>>'{rol,nombre}'
 OR vec_autorizacion.concesiones_positivas_validas(nueva) IS NOT TRUE
 OR vec_autorizacion.estado_rol_candidato_preferencias_aut104(nueva) IS NULL
 OR (origen_modalidades AND vec_autorizacion.delta_modalidades_candidato_aut104(base->'rol',nueva) IS NOT TRUE)
 OR (NOT origen_modalidades AND nueva->'concesiones' IS DISTINCT FROM
   (base#>'{rol,concesiones}')||normalizadas)
 OR (NOT origen_modalidades AND EXISTS(SELECT 1 FROM jsonb_array_elements(base#>'{rol,concesiones}') x
   WHERE x->>'accion'=ANY(acciones)))
 OR (p_primera AND EXISTS(SELECT 1 FROM vec_autorizacion.version_rol z
   WHERE z.version_rol_ref=p->>'version_rol_objetivo_ref'))
 THEN RAISE EXCEPTION 'AUT77: delta de dos altas divergente' USING ERRCODE='40001';END IF;
 FOR t IN SELECT value FROM jsonb_array_elements(p->'asignaciones') WITH ORDINALITY x(value,n) ORDER BY n LOOP
  IF t->>'asignacion_ref'=ANY(vistas)
  THEN RAISE EXCEPTION 'AUT77: asignacion duplicada' USING ERRCODE='22023';END IF;
  vistas:=array_append(vistas,t->>'asignacion_ref');original:=t->'documento';
  ambitos:=original->'ambitos';
  IF t->>'asignacion_ref' IS DISTINCT FROM 'asignacion:'||(original->>'asignacion_id')||':v'||(original->>'version')
  OR t->>'huella_sha256' IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(original),'UTF8')),'hex')
  OR original->>'version_rol_ref' IS DISTINCT FROM ('rol:'||(base#>>'{rol,rol_id}')||':v'||(base#>>'{rol,version}'))
  OR original->>'estado' IS DISTINCT FROM 'activa'
  OR vec_autorizacion.ambitos_positivos_validos(original) IS NOT TRUE
  OR jsonb_typeof(ambitos) IS DISTINCT FROM 'array' OR jsonb_array_length(ambitos)<>1
  OR ambitos->0->>'clave' IS DISTINCT FROM 'candidato_ref'
  OR jsonb_typeof(ambitos#>'{0,valores}') IS DISTINCT FROM 'array'
  OR jsonb_array_length(ambitos#>'{0,valores}')<>1
  OR (ambitos#>>'{0,valores,0}' ~ '^can_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE
  OR (p_primera AND vec_contexto_actor_v1.acreditar_candidato_externo_v1(
   original->>'principal_id',original->>'perfil_activo_ref',ambitos#>>'{0,valores,0}') IS NOT TRUE)
  THEN RAISE EXCEPTION 'AUT77: asignacion o ambitos invalidos' USING ERRCODE='22023';END IF;
  SELECT x.* INTO asignacion FROM vec_autorizacion.asignacion_perfil_externa x
   WHERE x.asignacion_ref=t->>'asignacion_ref' FOR SHARE;
  IF NOT FOUND OR asignacion.huella_sha256 IS DISTINCT FROM t->>'huella_sha256'
  OR asignacion.documento IS DISTINCT FROM original
  OR asignacion.principal_id IS DISTINCT FROM original->>'principal_id'
  OR asignacion.perfil_activo_ref IS DISTINCT FROM original->>'perfil_activo_ref'
  OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol z
   WHERE z.version_rol_ref=asignacion.version_rol_ref AND z.rol_id=rol.rol_id AND z.version<=rol.version)
  THEN RAISE EXCEPTION 'AUT77: asignacion historica divergente' USING ERRCODE='40001';END IF;
  IF p_primera AND (NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual_externa q
    WHERE q.perfil_activo_ref=asignacion.perfil_activo_ref AND q.asignacion_ref=asignacion.asignacion_ref)
   OR ahora<(asignacion.documento->>'vigente_desde')::timestamptz
   OR ahora>=(asignacion.documento->>'vigente_hasta')::timestamptz
   OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_externa rev
     WHERE rev.asignacion_id=asignacion.asignacion_id AND rev.documento->>'estado'='revocada'))
  THEN RAISE EXCEPTION 'AUT77: asignacion no actual o vigente' USING ERRCODE='40001';END IF;
 END LOOP;
 RETURN jsonb_build_object('vigente_hasta',CASE WHEN isfinite(limite_entradas)
  THEN to_char(limite_entradas AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
  ELSE '0001-01-01T00:00:00Z' END);
END $function$;

DO $post$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion.cotejar_lectura_preferencias_bolsas_actual_v1(jsonb,text,text)'::regprocedure
  AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='7d02c0b1deaca015bf4545c29ea07b5494e45e241cabb51779d8cc31a105a4bb'
  AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef IS TRUE
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s']::text[] AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario}')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion.cotejar_lectura_preferencias_bolsas_actual_interna_v1(jsonb,text,text)'::regprocedure
  AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='c355d635e7bafd3b96daa1cdc895e6e3172586cbe57e6303ba2522ccbe39f88e'
  AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef IS TRUE
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s']::text[] AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario}')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion.resolver_base_version_rol_bolsa_externa_v1()'::regprocedure
  AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='7f4a6565484f2c86e4b5e5b2fd9adbd96147e158abd8ca7fb1b8b8192f482bf9'
  AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef IS TRUE
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on']::text[] AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_admin_version_rol_bolsa_externa_ejecutor=X/vec_autorizacion_propietario}')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion.validar_plan_version_rol_bolsa_externa_v1(jsonb,boolean)'::regprocedure
  AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='41b606db0dbab29b9454d63cfc850b4f18ec025720e8f80bede6a79e910de57d'
  AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef IS TRUE
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC','row_security=on']::text[] AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario}')
 THEN RAISE EXCEPTION 'AUT104: postimagen incompatible' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion.cotejar_campos_preferencias_modalidades_aut104(jsonb,jsonb)'::regprocedure
  AND p.proowner='vec_autorizacion_propietario'::regrole AND NOT p.prosecdef
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']::text[]
  AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario}')
 THEN RAISE EXCEPTION 'AUT104: helper sin ACL' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion.estado_rol_candidato_preferencias_aut104(jsonb)'::regprocedure
  AND p.proowner='vec_autorizacion_propietario'::regrole AND NOT p.prosecdef
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']::text[]
  AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario}')
 THEN RAISE EXCEPTION 'AUT104: helper sin ACL' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion.delta_modalidades_candidato_aut104(jsonb,jsonb)'::regprocedure
  AND p.proowner='vec_autorizacion_propietario'::regrole AND NOT p.prosecdef
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']::text[]
  AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario}')
 THEN RAISE EXCEPTION 'AUT104: helper sin ACL' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
