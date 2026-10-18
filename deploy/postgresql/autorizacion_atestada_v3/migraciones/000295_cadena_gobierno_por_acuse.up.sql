\set ON_ERROR_STOP on
-- AD295 / F0-29: mismo lector, tramo sellado ligado al acuse original.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000295',0));
DO $pre$
DECLARE f regprocedure:=to_regprocedure('vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1(text)');
 p pg_proc%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999 OR f IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1(text,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD295: firma previa incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
 OR p.prosecdef IS NOT TRUE OR p.provolatile<>'s' OR p.proparallel<>'u'
 OR p.proacl::text IS DISTINCT FROM
  '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_autorizacion_atestada_v3_lector_gobierno=X/vec_autorizacion_atestada_v3_propietario}'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']
 OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM
  'da4ca1ed39fddea2801e0de5381109fd9f3de89f87447a91684cd23ed86dcca1'
 THEN RAISE EXCEPTION 'AD295: cuerpo o permisos previos divergentes' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1(p_esquema text, p_auditoria_ref text DEFAULT NULL::text)
 RETURNS jsonb
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE d jsonb;
BEGIN
 IF p_esquema IS NULL OR p_esquema !~ '^[a-z0-9][a-z0-9._-]{0,127}$'
 THEN RAISE EXCEPTION 'AD235: PARO clave=esquema actual=invalido esperado=identificador_acotado' USING ERRCODE='22023'; END IF;
 IF p_auditoria_ref IS NOT NULL AND p_auditoria_ref !~ '^aud_v3_gu_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'AD295: referencia de acuse inválida' USING ERRCODE='22023'; END IF;
 WITH corte AS (SELECT secuencia n FROM vec_autorizacion_atestada_v3.control_cadena_auditoria),
 cadena AS (
  SELECT a.*,a.secuencia posicion,NULL::jsonb eslabon,a.anterior_sha256 enlace_previo,a.huella_sha256 enlace
  FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a,corte WHERE a.secuencia<=corte.n
  UNION ALL
  SELECT a.*,e.posicion,jsonb_build_object('posicion',e.posicion,'secuencia',e.secuencia,'anterior_sha256',e.anterior_sha256,'eslabon_sha256',e.eslabon_sha256,'registrada_en',to_char(a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'sellado_en',to_char(e.sellado_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),e.anterior_sha256,e.eslabon_sha256
  FROM vec_autorizacion_atestada_v3.eslabon_auditoria_v5 e JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 a USING (secuencia)
 ), ancla AS (
  SELECT posicion FROM cadena
  WHERE auditoria_ref=p_auditoria_ref AND tipo_registro='gobierno_usuarios_admin'
 ), limites AS (
  SELECT max(posicion) FILTER (WHERE p_auditoria_ref IS NULL OR posicion<(SELECT posicion FROM ancla)) inferior,
   min(posicion) FILTER (WHERE posicion>(SELECT posicion FROM ancla)) superior
  FROM cadena WHERE tipo_registro NOT IN ('gobierno_usuarios_admin','intento_gobierno_usuarios_admin')
 ), rows AS (
  SELECT c.* FROM cadena c,limites l
  WHERE tipo_registro IN ('gobierno_usuarios_admin','intento_gobierno_usuarios_admin')
  AND posicion>COALESCE(l.inferior,0)
  AND (p_auditoria_ref IS NULL OR (EXISTS(SELECT 1 FROM ancla) AND (l.superior IS NULL OR posicion<l.superior)))
 ), rango AS(SELECT min(posicion) primero,max(posicion) ultimo,count(*) cuenta FROM rows)
 SELECT jsonb_build_object('esquema',p_esquema,'manifiesto',jsonb_build_object('cadena_id','cadena:comun:interna','primera_secuencia',ra.primero,'ultima_secuencia',ra.ultimo,'registros',ra.cuenta,'anterior_sha256',(SELECT enlace_previo FROM rows ORDER BY posicion LIMIT 1),'cabeza_sha256',(SELECT enlace FROM rows ORDER BY posicion DESC LIMIT 1)),
  'registros',(SELECT jsonb_agg(jsonb_build_object('tipo_registro',a.tipo_registro) || CASE WHEN a.eslabon IS NULL THEN '{}'::jsonb ELSE jsonb_build_object('eslabon',a.eslabon) END ||
  CASE WHEN a.tipo_registro='gobierno_usuarios_admin' THEN jsonb_build_object('gobierno_usuarios',jsonb_build_object('auditoria_ref',a.auditoria_ref,'secuencia',a.secuencia,'anterior_sha256',a.anterior_sha256,'huella_sha256',a.huella_sha256,'registrada_en',to_char(a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'evento_ref',a.evento_ref,'evento_material_sha256',a.evento_material_sha256,'operador_login',a.operador_login,'accion',a.accion,'modulo_id',a.modulo_id,'recurso_ref',a.recurso_ref,'resultado',a.resultado,'motivo_ref',a.motivo_ref,'proceso',a.proceso,'canal',a.canal,'finalidad_ref',a.finalidad_ref,'correlacion_ref',a.correlacion_ref) || jsonb_build_object('plan_sha256',a.plan_sha256,'preimagen_sha256',a.gobierno_usuarios_detalle->>'preimagen_sha256','configuracion_origen_ref',a.gobierno_usuarios_detalle->>'configuracion_origen_ref','configuracion_destino_ref',a.gobierno_usuarios_detalle->>'configuracion_destino_ref','claves_sha256',a.gobierno_usuarios_detalle->>'claves_sha256'))
  ELSE jsonb_build_object('intento_gobierno_usuarios',jsonb_build_object('auditoria_ref',a.auditoria_ref,'secuencia',a.secuencia,'anterior_sha256',a.anterior_sha256,'huella_sha256',a.huella_sha256,'registrada_en',to_char(a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'evento_ref',a.evento_ref,'evento_material_sha256',a.evento_material_sha256,'operador_login',a.operador_login,'accion',a.accion,'modulo_id',a.modulo_id,'recurso_ref',a.recurso_ref,'resultado',a.resultado,'motivo_ref',a.motivo_ref,'proceso',a.proceso,'canal',a.canal,'finalidad_ref',a.finalidad_ref,'correlacion_ref',a.correlacion_ref) || jsonb_build_object('solicitud_sha256',a.gobierno_usuarios_solicitud_sha256)) END
  ORDER BY a.posicion) FROM rows a))
 INTO d FROM rango ra;
 IF p_auditoria_ref IS NOT NULL AND (d#>>'{manifiesto,registros}')::bigint=0
 THEN RAISE EXCEPTION 'AD295: acuse no disponible en la cadena sellada' USING ERRCODE='42501'; END IF;
 RETURN d;
END $function$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1(text,text)
 TO vec_autorizacion_atestada_v3_lector_gobierno;
DROP FUNCTION vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1(text) RESTRICT;
DO $post$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1(text,text)'::regprocedure;
BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM '023b2eaa5c2283b37e2646778322a242d1f7c36d53af0114f2d905c872482f8c'
 OR (SELECT proacl::text FROM pg_proc WHERE oid=f) IS DISTINCT FROM
  '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_autorizacion_atestada_v3_lector_gobierno=X/vec_autorizacion_atestada_v3_propietario}'
 THEN RAISE EXCEPTION 'AD295: cuerpo o permisos finales divergentes' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
