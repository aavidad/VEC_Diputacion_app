\set ON_ERROR_STOP on
-- AUT95: extensión prospectiva de M4, misma cuenta, perfil y fuente externa.
-- Las tres acciones publicadas conservan su significado y sus recibos.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000095',0));
DO $pre$
BEGIN
 IF current_user<>'vec_autorizacion_propietario' OR
  to_regrole('vec_documentos_subida_datos') IS NULL OR
  to_regprocedure('vec_autorizacion.acciones_meritos_cartera_v2()') IS NOT NULL OR
  to_regprocedure('vec_autorizacion.obtener_instantanea_meritos_externo_v1(text,text)') IS NULL OR
  to_regprocedure('vec_autorizacion.decision_meritos_externa_exacta_v1(jsonb,jsonb)') IS NULL OR
  to_regprocedure('vec_autorizacion.cotejar_cartera_meritos_actual_v1(jsonb,jsonb)') IS NULL OR
  to_regprocedure('vec_autorizacion.publicar_asignacion_usuarios_externo_v1(bytea,text,bigint,text,text,text,text)') IS NULL
 THEN RAISE EXCEPTION 'AUT95: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_autorizacion.acciones_meritos_cartera_v2() RETURNS text[]
LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT ARRAY[
  'meritos.cartera.comprobante.descargar_propio',
  'meritos.cartera.comprobante.subir_propio',
  'meritos.cartera.consultar_propia',
  'meritos.cartera.retirar_propio',
  'meritos.hecho.declarar']::text[]
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.acciones_meritos_cartera_v2() FROM PUBLIC;

-- La fuente M4 existente admite los tres estados exactos 2/3/5.
DO $fuente$
BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef(
  'vec_autorizacion.obtener_instantanea_meritos_externo_v1(text,text)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM
  '3843cf4b256a89f045c6e98b4b33c9a9a769b6b09b6bc6bfa232f6c0dc10f297'
 THEN RAISE EXCEPTION 'AUT95: fuente M4 instalada divergente' USING ERRCODE='55000'; END IF;
END $fuente$;
CREATE OR REPLACE FUNCTION vec_autorizacion.obtener_instantanea_meritos_externo_v1(p_principal_id text, p_perfil_activo_ref text)
 RETURNS TABLE(documento_asignacion jsonb, documento_rol jsonb, documento_control_rol jsonb, revision_catalogo text, huella_catalogo text, documentos_politicas jsonb)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
AS $function$
DECLARE rol record;
BEGIN
 IF current_user<>'vec_autorizacion_propietario'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR vec_autorizacion.login_candidato_externo_v1(
  'vec_externo_meritos_fuente_autorizacion','vec_autorizacion_fuente_meritos_externa') IS NOT TRUE
 OR vec_contexto_actor_v1.perfil_usuarios_externo_provisionado_v1(
  p_principal_id,p_perfil_activo_ref) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT78: fuente externa denegada' USING ERRCODE='42501'; END IF;
 SELECT r.rol_id,r.documento INTO rol
 FROM vec_autorizacion.asignacion_perfil_actual_externa ap
 JOIN vec_autorizacion.asignacion_perfil_externa a
  ON a.perfil_activo_ref=ap.perfil_activo_ref AND a.asignacion_ref=ap.asignacion_ref
 JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref
 WHERE ap.perfil_activo_ref=p_perfil_activo_ref AND a.principal_id=p_principal_id
 FOR SHARE OF ap,a,r;
 IF NOT FOUND OR rol.rol_id IS DISTINCT FROM 'persona_meritos_cartera'
 OR jsonb_typeof(rol.documento->'concesiones') IS DISTINCT FROM 'array'
 OR ((SELECT array_agg(x.value->>'accion' ORDER BY x.value->>'accion')
    FROM jsonb_array_elements(rol.documento->'concesiones') x) IS DISTINCT FROM
   ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio']::text[]
  AND (SELECT array_agg(x.value->>'accion' ORDER BY x.value->>'accion')
    FROM jsonb_array_elements(rol.documento->'concesiones') x) IS DISTINCT FROM
   ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio','meritos.hecho.declarar']::text[]
  AND (SELECT array_agg(x.value->>'accion' ORDER BY x.value->>'accion')
   FROM jsonb_array_elements(rol.documento->'concesiones') x) IS DISTINCT FROM
   vec_autorizacion.acciones_meritos_cartera_v2())
 THEN RAISE EXCEPTION 'AUT78: rol externo de Méritos ausente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT a.documento,r.documento,c.documento,cat.revision::text,cat.huella_sha256,
  coalesce((SELECT jsonb_agg(p.documento ORDER BY p.politica_ref)
   FROM vec_autorizacion.politica_restrictiva_actual pa
   JOIN vec_autorizacion.politica_restrictiva p
    ON p.politica_id=pa.politica_id AND p.politica_ref=pa.politica_ref),'[]'::jsonb)
 FROM vec_autorizacion.asignacion_perfil_actual_externa ap
 JOIN vec_autorizacion.asignacion_perfil_externa a
  ON a.perfil_activo_ref=ap.perfil_activo_ref AND a.asignacion_ref=ap.asignacion_ref
 JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol_actual cp
  ON cp.version_rol_ref=r.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol c
  ON c.version_rol_ref=cp.version_rol_ref AND c.revision=cp.revision
 JOIN vec_autorizacion.control_catalogo_politicas cat ON cat.control_id IS TRUE
 WHERE ap.perfil_activo_ref=p_perfil_activo_ref AND a.principal_id=p_principal_id
  AND r.rol_id='persona_meritos_cartera' AND r.documento->>'estado'='publicada'
  AND c.estado='habilitada';
END $function$;

-- M4 conserva sus casos 2/3 y consulta de ocho campos. El rol nuevo permite
-- nueve campos solo si orden y concesión coinciden de forma exacta.
DO $cartera$
BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef(
  'vec_autorizacion.cotejar_cartera_meritos_actual_v1(jsonb,jsonb)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM
  '4eeffb4bb5a1ca7977d8392400d85b1ef25dcb4dd0db2e56226e2cc4a4dc26ea'
 THEN RAISE EXCEPTION 'AUT95: cotejo M4 instalado divergente' USING ERRCODE='55000'; END IF;
END $cartera$;
CREATE OR REPLACE FUNCTION vec_autorizacion.cotejar_cartera_meritos_actual_v1(p_orden jsonb, p_vinculo jsonb)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
AS $function$
DECLARE asignacion record;rol record;control record;catalogo record;
 politicas jsonb;concesion jsonb;ahora timestamptz(6);accion text;persona text;
 campos jsonb;
BEGIN
 IF current_user<>'vec_autorizacion_propietario'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR jsonb_typeof(p_vinculo) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
   JOIN pg_roles g ON g.oid=m.roleid
   WHERE l.rolname=session_user AND l.rolname='vec_externo_meritos_cartera'
    AND l.rolcanlogin AND l.rolinherit
   AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
   AND l.rolconfig IS NULL AND g.rolname='vec_meritos_cartera_externo'
   AND NOT(g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
   AND g.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 THEN RETURN false; END IF;
 accion:=p_orden->>'accion';persona:=p_orden->>'persona_ref';
 campos:=CASE accion
  WHEN 'meritos.cartera.consultar_propia' THEN
   CASE WHEN p_orden->'campos_permitidos' IN (
    '["denominacion","disponibilidad","hecho_ref","motivos_retirada","retirada","tipo","total","version_hecho"]'::jsonb,
    '["denominacion","disponibilidad","evidencias","hecho_ref","motivos_retirada","retirada","tipo","total","version_hecho"]'::jsonb)
   THEN p_orden->'campos_permitidos' ELSE NULL::jsonb END
  WHEN 'meritos.cartera.retirar_propio' THEN '["disponibilidad","recibo"]'::jsonb
  ELSE NULL END;
 IF campos IS NULL OR p_orden->>'modulo_id' IS DISTINCT FROM 'meritos'
 OR p_orden->>'tipo_recurso' IS DISTINCT FROM
  (CASE WHEN accion='meritos.cartera.consultar_propia' THEN 'cartera_personal' ELSE 'hecho' END)
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'gestion_cartera_propia'
 OR p_orden->>'canal' IS DISTINCT FROM 'externa_personal'
 OR (accion='meritos.cartera.consultar_propia' AND
  (p_orden->>'idioma' IS NULL OR p_orden->>'idioma' NOT IN ('es','en')))
 OR p_orden->'campos_permitidos' IS DISTINCT FROM campos
 OR p_orden->>'recurso_ref' IS DISTINCT FROM
  (CASE WHEN accion='meritos.cartera.consultar_propia' THEN 'cartera:'||persona
  ELSE p_orden->>'hecho_ref' END)
 OR persona IS DISTINCT FROM p_vinculo->>'principal_id'
 OR p_orden->>'perfil_activo_ref' IS DISTINCT FROM p_vinculo->>'perfil_activo_ref'
 OR p_vinculo->>'superficie' IS DISTINCT FROM 'externa_personal'
 OR p_vinculo->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
 OR p_orden->>'emitida_en' IS NULL OR p_orden->>'valida_hasta' IS NULL
 OR jsonb_typeof(p_orden->'politicas_evaluadas') IS DISTINCT FROM 'array'
 THEN RETURN false; END IF;
 ahora:=clock_timestamp();
 IF (p_orden->>'emitida_en')::timestamptz>ahora
 OR (p_orden->>'valida_hasta')::timestamptz<=ahora
 OR (p_orden->>'valida_hasta')::timestamptz>ahora+interval '30 seconds'
 OR vec_autorizacion.revalidar_sesion_cartera_meritos_v1(
   p_vinculo,(p_orden->>'emitida_en')::timestamptz,
   (p_orden->>'valida_hasta')::timestamptz,ahora) IS NOT TRUE
 THEN RETURN false; END IF;
 SELECT x.* INTO asignacion FROM vec_autorizacion.asignacion_perfil_actual_externa ap
 JOIN vec_autorizacion.asignacion_perfil_externa x
 ON x.perfil_activo_ref=ap.perfil_activo_ref AND x.asignacion_ref=ap.asignacion_ref
 WHERE ap.perfil_activo_ref=p_orden->>'perfil_activo_ref' FOR SHARE OF ap,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO rol FROM vec_autorizacion.version_rol x
 WHERE x.version_rol_ref=asignacion.version_rol_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO control FROM vec_autorizacion.control_vigencia_version_rol_actual cp
 JOIN vec_autorizacion.control_vigencia_version_rol x
 ON x.version_rol_ref=cp.version_rol_ref AND x.revision=cp.revision
 WHERE cp.version_rol_ref=rol.version_rol_ref FOR SHARE OF cp,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT * INTO catalogo FROM vec_autorizacion.control_catalogo_politicas
 WHERE control_id IS TRUE FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object('referencia',p.politica_ref,
  'huella_sha256',p.huella_sha256) ORDER BY p.politica_ref),'[]'::jsonb)
 INTO politicas FROM vec_autorizacion.politica_restrictiva_actual pa
 JOIN vec_autorizacion.politica_restrictiva p
 ON p.politica_id=pa.politica_id AND p.politica_ref=pa.politica_ref;
 IF asignacion.principal_id IS DISTINCT FROM persona
 OR asignacion.asignacion_ref IS DISTINCT FROM p_orden->>'asignacion_ref'
 OR asignacion.version IS DISTINCT FROM (p_orden->>'asignacion_version')::numeric
 OR asignacion.huella_sha256 IS DISTINCT FROM p_orden->>'asignacion_huella_sha256'
 OR asignacion.documento->>'estado' IS DISTINCT FROM 'activa'
 OR ahora<(asignacion.documento->>'vigente_desde')::timestamptz
 OR ahora>=(asignacion.documento->>'vigente_hasta')::timestamptz
 OR rol.rol_id IS DISTINCT FROM 'persona_meritos_cartera'
 OR rol.version_rol_ref IS DISTINCT FROM p_orden->>'version_rol_ref'
 OR rol.version IS DISTINCT FROM (p_orden->>'version_rol_version')::numeric
 OR rol.huella_sha256 IS DISTINCT FROM p_orden->>'version_rol_huella_sha256'
 OR rol.documento->>'estado' IS DISTINCT FROM 'publicada'
 OR jsonb_typeof(rol.documento->'concesiones') IS DISTINCT FROM 'array'
 OR ((SELECT array_agg(e.value->>'accion' ORDER BY e.value->>'accion')
     FROM jsonb_array_elements(rol.documento->'concesiones') e) IS DISTINCT FROM
    ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio']::text[]
  AND (SELECT array_agg(e.value->>'accion' ORDER BY e.value->>'accion')
     FROM jsonb_array_elements(rol.documento->'concesiones') e) IS DISTINCT FROM
    ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio','meritos.hecho.declarar']::text[]
  AND (SELECT array_agg(e.value->>'accion' ORDER BY e.value->>'accion')
   FROM jsonb_array_elements(rol.documento->'concesiones') e) IS DISTINCT FROM
   vec_autorizacion.acciones_meritos_cartera_v2())
 OR control.version_rol_ref IS DISTINCT FROM p_orden->>'control_version_rol_ref'
 OR control.revision IS DISTINCT FROM (p_orden->>'revision_control_version_rol')::numeric
 OR control.huella_sha256 IS DISTINCT FROM p_orden->>'control_version_rol_huella_sha256'
 OR control.estado IS DISTINCT FROM 'habilitada'
 OR catalogo.revision IS DISTINCT FROM (p_orden->>'revision_catalogo_politicas')::numeric
 OR catalogo.huella_sha256 IS DISTINCT FROM p_orden->>'catalogo_politicas_huella_sha256'
 OR politicas IS DISTINCT FROM p_orden->'politicas_evaluadas'
 OR jsonb_typeof(asignacion.documento->'ambitos') IS DISTINCT FROM 'array'
 OR jsonb_array_length(asignacion.documento->'ambitos')<>1
 OR vec_autorizacion.ambitos_positivos_validos(asignacion.documento) IS NOT TRUE
 OR asignacion.documento->'ambitos' @> jsonb_build_array(
    jsonb_build_object('clave','persona_ref','valores',jsonb_build_array(persona))) IS NOT TRUE
 THEN RETURN false; END IF;
 SELECT e.value INTO concesion FROM jsonb_array_elements(rol.documento->'concesiones') e
 WHERE e.value->>'accion'=accion;
 IF NOT FOUND OR (SELECT count(*) FROM jsonb_array_elements(rol.documento->'concesiones') e
   WHERE e.value->>'accion'=accion)<>1
 OR concesion->>'modulo_id' IS DISTINCT FROM 'meritos'
 OR concesion->>'tipo_recurso' IS DISTINCT FROM p_orden->>'tipo_recurso'
 OR concesion->'finalidades' IS DISTINCT FROM '["gestion_cartera_propia"]'::jsonb
 OR concesion->'campos_permitidos' IS DISTINCT FROM campos
 OR concesion->>'garantia_minima' IS DISTINCT FROM 'alto'
 OR coalesce(concesion->'obligaciones','[]'::jsonb)
  IS DISTINCT FROM '["auditar"]'::jsonb
 THEN RETURN false; END IF;
 RETURN true;
EXCEPTION WHEN data_exception OR invalid_text_representation OR numeric_value_out_of_range THEN
 RETURN false;
END $function$;

-- El registrador V3 M4 conserva las dos operaciones anteriores y acepta
-- subir/descargar solo con el rol de cinco concesiones gobernado.
DO $decision_pre$
BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef(
  'vec_autorizacion.decision_meritos_externa_exacta_v1(jsonb,jsonb)'::regprocedure),'UTF8')),'hex')
  IS DISTINCT FROM 'c2e91ee495253426458f8b9c6923d0ac10d2e1231566dfdf6cc59b4718382499'
 THEN RAISE EXCEPTION 'AUT95: decisión M4 instalada divergente' USING ERRCODE='55000'; END IF;
END $decision_pre$;
CREATE OR REPLACE FUNCTION vec_autorizacion.decision_meritos_externa_exacta_v1(d jsonb,m jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE v jsonb; accion text; campos jsonb; obligaciones jsonb; lista text[]; rol_documento jsonb;
BEGIN
 v:=d->'vinculo_autenticacion_actor'; accion:=d->>'accion';
 campos:=CASE accion
  WHEN 'meritos.hecho.declarar' THEN '["declarante_ref","hecho","recibo","version"]'::jsonb
  WHEN 'meritos.cartera.retirar_propio' THEN '["disponibilidad","recibo"]'::jsonb
  WHEN 'meritos.cartera.comprobante.subir_propio' THEN '["contenido","estado","recibo"]'::jsonb
  WHEN 'meritos.cartera.comprobante.descargar_propio' THEN '["contenido"]'::jsonb END;
 obligaciones:=CASE WHEN accion IN ('meritos.cartera.comprobante.subir_propio',
  'meritos.cartera.comprobante.descargar_propio') THEN '[]'::jsonb ELSE '["auditar"]'::jsonb END;
 IF current_user<>'vec_autorizacion_propietario' OR
  vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE OR
  campos IS NULL OR d->>'modulo_id' IS DISTINCT FROM 'meritos' OR
  d->>'tipo_recurso' IS DISTINCT FROM 'hecho' OR
  d->>'finalidad' IS DISTINCT FROM (CASE
   WHEN accion='meritos.hecho.declarar' THEN 'declaracion_hecho_propio'
   WHEN accion='meritos.cartera.retirar_propio' THEN 'gestion_cartera_propia'
   ELSE 'custodiar_evidencia_meritos' END) OR
  (accion IN ('meritos.cartera.comprobante.subir_propio',
   'meritos.cartera.comprobante.descargar_propio') AND
   d->>'recurso_ref' !~ '^ref:[0-9a-f]{64}$') OR
  v->>'superficie' IS DISTINCT FROM 'externa_personal' OR
  v->>'cuenta_privilegiada' IS DISTINCT FROM 'false' OR
  d->>'principal_id' IS DISTINCT FROM v->>'principal_id' OR
  d->>'perfil_activo_ref' IS DISTINCT FROM v->>'perfil_activo_ref' OR
  vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(
   v->>'cuenta_ref',d->>'principal_id',d->>'perfil_activo_ref') IS NOT TRUE OR
  m#>>'{referencia,catalogo_id}' IS DISTINCT FROM
   (CASE WHEN accion='meritos.hecho.declarar' THEN 'motivos_meritos_declaracion'
    ELSE 'motivos_autorizacion_meritos_cartera' END)
 THEN RETURN false; END IF;
 SELECT (SELECT array_agg(x.value->>'accion' ORDER BY x.value->>'accion')
  FROM jsonb_array_elements(r.documento->'concesiones') x),r.documento
 INTO lista,rol_documento FROM vec_autorizacion.version_rol r
 WHERE r.version_rol_ref=d->>'version_rol_ref' AND r.rol_id='persona_meritos_cartera';
 IF lista IS NULL OR NOT (
  (accion='meritos.hecho.declarar' AND lista IN (
   ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio','meritos.hecho.declarar']::text[],
   vec_autorizacion.acciones_meritos_cartera_v2())) OR
  (accion='meritos.cartera.retirar_propio' AND lista IN (
   ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio']::text[],
   ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio','meritos.hecho.declarar']::text[],
   vec_autorizacion.acciones_meritos_cartera_v2())) OR
  (accion IN ('meritos.cartera.comprobante.subir_propio',
   'meritos.cartera.comprobante.descargar_propio') AND
   lista IS NOT DISTINCT FROM vec_autorizacion.acciones_meritos_cartera_v2() AND
   vec_autorizacion.rol_meritos_cartera_cinco_v2(rol_documento) IS TRUE))
 THEN RETURN false; END IF;
 IF d->>'concedida'='true' THEN
  RETURN d->'campos_permitidos' IS NOT DISTINCT FROM campos AND
   d->'obligaciones' IS NOT DISTINCT FROM obligaciones;
 END IF;
 RETURN d->>'concedida'='false' AND d->'campos_permitidos' <@ campos AND
  d->'obligaciones' <@ obligaciones;
EXCEPTION WHEN data_exception OR invalid_text_representation THEN RETURN false;
END $f$;

-- El publicador de asignaciones acepta la nueva versión M4 si cada una de
-- sus cinco concesiones tiene exactamente la semántica publicada.
CREATE FUNCTION vec_autorizacion.rol_meritos_cartera_cinco_v2(p_rol jsonb) RETURNS boolean
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE c jsonb; accion text; n integer:=0;
BEGIN
 IF current_user<>'vec_autorizacion_propietario' OR
  p_rol->>'rol_id' IS DISTINCT FROM 'persona_meritos_cartera' OR
  jsonb_typeof(p_rol->'concesiones') IS DISTINCT FROM 'array' OR
  jsonb_array_length(p_rol->'concesiones')<>5
 THEN RETURN false; END IF;
 FOR c IN SELECT value FROM jsonb_array_elements(p_rol->'concesiones') LOOP
  accion:=c->>'accion'; n:=n+1;
  IF jsonb_typeof(c)<>'object' OR
   (SELECT count(*) FROM jsonb_object_keys(c))<>7 OR
   c->>'modulo_id' IS DISTINCT FROM 'meritos' OR
   c->>'garantia_minima' IS DISTINCT FROM 'alto' OR
   (accion=ANY(vec_autorizacion.acciones_meritos_cartera_v2())) IS NOT TRUE OR
   c->>'tipo_recurso' IS DISTINCT FROM
    (CASE WHEN accion='meritos.cartera.consultar_propia' THEN 'cartera_personal' ELSE 'hecho' END) OR
   c->'finalidades' IS DISTINCT FROM
    (CASE WHEN accion='meritos.hecho.declarar' THEN '["declaracion_hecho_propio"]'::jsonb
     WHEN accion IN ('meritos.cartera.comprobante.subir_propio','meritos.cartera.comprobante.descargar_propio')
      THEN '["custodiar_evidencia_meritos"]'::jsonb ELSE '["gestion_cartera_propia"]'::jsonb END) OR
   c->'campos_permitidos' IS DISTINCT FROM
    (CASE accion
     WHEN 'meritos.cartera.consultar_propia' THEN
      '["denominacion","disponibilidad","evidencias","hecho_ref","motivos_retirada","retirada","tipo","total","version_hecho"]'::jsonb
     WHEN 'meritos.cartera.retirar_propio' THEN '["disponibilidad","recibo"]'::jsonb
     WHEN 'meritos.hecho.declarar' THEN '["declarante_ref","hecho","recibo","version"]'::jsonb
     WHEN 'meritos.cartera.comprobante.subir_propio' THEN '["contenido","estado","recibo"]'::jsonb
     ELSE '["contenido"]'::jsonb END) OR
   c->'obligaciones' IS DISTINCT FROM
    (CASE WHEN accion IN ('meritos.cartera.comprobante.subir_propio','meritos.cartera.comprobante.descargar_propio')
      THEN '[]'::jsonb ELSE '["auditar"]'::jsonb END)
  THEN RETURN false; END IF;
 END LOOP;
 RETURN n=5 AND ((SELECT array_agg(x.value->>'accion' ORDER BY x.value->>'accion')
  FROM jsonb_array_elements(p_rol->'concesiones') x)
   IS NOT DISTINCT FROM vec_autorizacion.acciones_meritos_cartera_v2());
EXCEPTION WHEN data_exception THEN RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.rol_meritos_cartera_cinco_v2(jsonb) FROM PUBLIC;

DO $asignacion$
BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef(
  'vec_autorizacion.publicar_asignacion_usuarios_externo_v1(bytea,text,bigint,text,text,text,text)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM
  'ed39739ff168213b1cfd896491fcbf1ea5f8dc02d93d81d56c7843bc0b2d5eb9'
 THEN RAISE EXCEPTION 'AUT95: publicador M4 instalado divergente' USING ERRCODE='55000'; END IF;
END $asignacion$;
CREATE OR REPLACE FUNCTION vec_autorizacion.publicar_asignacion_usuarios_externo_v1(p_documento_canonico bytea, p_huella_sha256 text, p_version_esperada bigint, p_huella_esperada text, p_actualizada_por text, p_acto_ref text, p_cuenta_ref text)
 RETURNS TABLE(asignacion_ref text, version bigint, huella_sha256 text)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE d jsonb; r record; anterior record; ahora timestamptz:=pg_catalog.clock_timestamp();
        id text; perfil text; principal text; rol text; nueva_version bigint; ref text;
BEGIN
 IF vec_autorizacion.publicador_usuarios_externo_interno_valido_v1() IS NOT TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_documento_canonico IS NULL OR pg_catalog.octet_length(p_documento_canonico) NOT BETWEEN 1 AND 65536
    OR p_huella_sha256 !~ '^[0-9a-f]{64}$'
    OR pg_catalog.encode(pg_catalog.sha256(p_documento_canonico),'hex') IS DISTINCT FROM p_huella_sha256
    OR p_version_esperada IS NULL OR p_version_esperada<0
    OR (p_version_esperada=0 AND p_huella_esperada IS NOT NULL)
    OR (p_version_esperada>0 AND p_huella_esperada !~ '^[0-9a-f]{64}$')
    OR vec_autorizacion.texto_positivo_valido(p_actualizada_por,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(p_acto_ref,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(p_cuenta_ref,512) IS NOT TRUE
 THEN RAISE EXCEPTION 'publicación externa rechazada' USING ERRCODE='42501'; END IF;
 BEGIN d:=pg_catalog.convert_from(p_documento_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'documento externo inválido' USING ERRCODE='22023'; END;
 id:=d->>'asignacion_id'; perfil:=d->>'perfil_activo_ref'; principal:=d->>'principal_id'; rol:=d->>'version_rol_ref';
 nueva_version:=p_version_esperada+1; ref:='asignacion:'||id||':v'||nueva_version::text;
 IF vec_autorizacion.texto_positivo_valido(id,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(perfil,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(principal,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(rol,512) IS NOT TRUE
    OR (d->>'version')::bigint IS DISTINCT FROM nueva_version
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>(11+CASE WHEN d ? 'revocada_en' THEN 1 ELSE 0 END)
    OR NOT d ?& ARRAY['asignacion_id','version','perfil_activo_ref','principal_id','version_rol_ref',
      'estado','ambitos','vigente_desde','vigente_hasta','emitida_por','emitida_en']
    OR d->>'estado' IS DISTINCT FROM 'activa'
    OR d ?| ARRAY['revocada_por','revocacion_ref']
    OR (d ? 'revocada_en' AND d->'revocada_en' IS DISTINCT FROM '"0001-01-01T00:00:00Z"'::jsonb)
    OR pg_catalog.jsonb_array_length(d->'ambitos')<>1
    OR pg_catalog.jsonb_typeof(d->'ambitos'->0) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d->'ambitos'->0))<>2
    OR NOT (d->'ambitos'->0) ?& ARRAY['clave','valores']
    OR d->'ambitos'->0->>'clave' IS DISTINCT FROM 'persona_ref'
    OR pg_catalog.jsonb_array_length(d->'ambitos'->0->'valores')<>1
    OR vec_autorizacion.texto_positivo_valido(d->'ambitos'->0->'valores'->>0,512) IS NOT TRUE
    OR d->'ambitos'->0->'valores'->>0 IS DISTINCT FROM principal
    OR d->>'emitida_por' IS DISTINCT FROM p_actualizada_por
    OR d->>'vigente_desde' IS NULL OR d->>'vigente_hasta' IS NULL
    OR (d->>'vigente_desde')::timestamptz>ahora
    OR (d->>'vigente_hasta')::timestamptz<=ahora
    OR vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(p_cuenta_ref,principal,perfil) IS NOT TRUE
 THEN RAISE EXCEPTION 'asignación externa inválida o sin provisión' USING ERRCODE='42501'; END IF;
 SELECT vr.rol_id,vr.huella_sha256,vr.documento INTO r FROM vec_autorizacion.version_rol vr
 JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=vr.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
 WHERE vr.version_rol_ref=rol AND c.estado='habilitada' FOR SHARE OF ca;
 IF NOT FOUND OR (CASE r.rol_id
    WHEN 'candidato_usuarios_propios_desarrollo' THEN
     vec_autorizacion.rol_usuarios_externo_acotado_v1(r.documento) IS NOT TRUE
    WHEN 'persona_meritos_cartera' THEN
     r.documento->>'rol_id' IS DISTINCT FROM 'persona_meritos_cartera'
     OR r.documento->>'estado' IS DISTINCT FROM 'publicada'
     OR (r.documento->'concesiones' IS DISTINCT FROM '[{"accion":"meritos.cartera.consultar_propia","modulo_id":"meritos","tipo_recurso":"cartera_personal","finalidades":["gestion_cartera_propia"],"garantia_minima":"alto","campos_permitidos":["denominacion","disponibilidad","hecho_ref","motivos_retirada","retirada","tipo","total","version_hecho"],"obligaciones":["auditar"]},{"accion":"meritos.cartera.retirar_propio","modulo_id":"meritos","tipo_recurso":"hecho","finalidades":["gestion_cartera_propia"],"garantia_minima":"alto","campos_permitidos":["disponibilidad","recibo"],"obligaciones":["auditar"]}]'::pg_catalog.jsonb
      AND r.documento->'concesiones' IS DISTINCT FROM '[{"accion":"meritos.cartera.consultar_propia","modulo_id":"meritos","tipo_recurso":"cartera_personal","finalidades":["gestion_cartera_propia"],"garantia_minima":"alto","campos_permitidos":["denominacion","disponibilidad","hecho_ref","motivos_retirada","retirada","tipo","total","version_hecho"],"obligaciones":["auditar"]},{"accion":"meritos.cartera.retirar_propio","modulo_id":"meritos","tipo_recurso":"hecho","finalidades":["gestion_cartera_propia"],"garantia_minima":"alto","campos_permitidos":["disponibilidad","recibo"],"obligaciones":["auditar"]},{"accion":"meritos.hecho.declarar","modulo_id":"meritos","tipo_recurso":"hecho","finalidades":["declaracion_hecho_propio"],"garantia_minima":"alto","campos_permitidos":["declarante_ref","hecho","recibo","version"],"obligaciones":["auditar"]}]'::pg_catalog.jsonb AND vec_autorizacion.rol_meritos_cartera_cinco_v2(r.documento) IS NOT TRUE)
    ELSE true END)
 THEN RAISE EXCEPTION 'rol externo no aprobado' USING ERRCODE='42501'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:asignacion_usuarios_externo:'||perfil,0));
 SELECT a.asignacion_ref,a.asignacion_id,a.version,a.principal_id,a.huella_sha256
 INTO anterior FROM vec_autorizacion.asignacion_perfil_actual_externa aa
 JOIN vec_autorizacion.asignacion_perfil_externa a ON a.asignacion_ref=aa.asignacion_ref
 WHERE aa.perfil_activo_ref=perfil FOR UPDATE OF aa;
 IF (NOT FOUND AND (p_version_esperada<>0 OR p_huella_esperada IS NOT NULL))
    OR (FOUND AND (anterior.version<>p_version_esperada OR anterior.huella_sha256 IS DISTINCT FROM p_huella_esperada
      OR anterior.asignacion_id IS DISTINCT FROM id OR anterior.principal_id IS DISTINCT FROM principal))
 THEN RAISE EXCEPTION 'asignación externa: CAS divergente' USING ERRCODE='40001'; END IF;
 INSERT INTO vec_autorizacion.asignacion_perfil_externa(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES(ref,id,nueva_version,perfil,principal,rol,p_huella_sha256,(d->>'emitida_en')::timestamptz,d);
 IF p_version_esperada=0 THEN
  INSERT INTO vec_autorizacion.asignacion_perfil_actual_externa(perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
  VALUES(perfil,ref,ahora,p_actualizada_por,p_acto_ref);
 ELSE
  UPDATE vec_autorizacion.asignacion_perfil_actual_externa AS puntero
   SET asignacion_ref=ref,actualizada_en=ahora,actualizada_por=p_actualizada_por,acto_ref=p_acto_ref
  WHERE puntero.perfil_activo_ref=perfil AND puntero.asignacion_ref=anterior.asignacion_ref;
  IF NOT FOUND THEN RAISE EXCEPTION 'asignación externa: puntero cambiado' USING ERRCODE='40001'; END IF;
 END IF;
 INSERT INTO vec_autorizacion.publicacion_candidato_externo_evento(
  tipo,acto_ref,objeto_ref,version,huella_previa,huella_nueva,publicada_por)
 VALUES('asignacion',p_acto_ref,ref,nueva_version,p_huella_esperada,p_huella_sha256,p_actualizada_por);
 RETURN QUERY SELECT ref,nueva_version,p_huella_sha256;
END $function$;

-- El núcleo AD273 ejecuta las decisiones del acto M4 con un LOGIN de datos
-- exclusivo. Los dos helpers M4 conservan sus cuentas anteriores y permiten
-- esa cuenta solo con un único grupo heredado y sin capacidad de SET ROLE.
DO $registro_datos$
BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef(
  'vec_autorizacion.registrar_y_revalidar_decision_meritos_externo_v3(bytea,bytea,numeric,numeric)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM
  '4d87ca659d4fcc43e6f224f702d38e6ed1ebdedc4780ba19e5a90d681ffc2c9d' OR
  encode(sha256(convert_to(pg_get_functiondef(
  'vec_autorizacion.revalidar_decision_meritos_externo_v3_viva(bytea,bytea,numeric,numeric)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM
  'd43b6897de107cfea37c23298235f63b26e51774b3eae03b974f0e1ad8e3834e'
 THEN RAISE EXCEPTION 'AUT95: registro o revalidacion M4 divergente' USING ERRCODE='55000'; END IF;
END $registro_datos$;
CREATE OR REPLACE FUNCTION vec_autorizacion.registrar_y_revalidar_decision_meritos_externo_v3(p_decision bytea, p_motivo bytea, p_persona_version numeric, p_perfil_version numeric)
 RETURNS TABLE(concedida boolean, codigo text, decision_huella_sha256 text, registrada_en timestamp with time zone, revalidada_en timestamp with time zone)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
AS $function$
DECLARE d jsonb;m jsonb;registro record;viva timestamptz;
BEGIN
 IF NOT (vec_autorizacion.login_candidato_externo_v1(
  'vec_externo_meritos_registro_autorizacion','vec_autorizacion_registro_meritos_externo') IS TRUE
  OR (session_user='vec_externo_meritos_cartera'
   AND pg_has_role(session_user,'vec_meritos_cartera_externo','MEMBER')
   AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1)
  OR (session_user='vec_externo_meritos_operacion'
   AND pg_has_role(session_user,'vec_meritos_ejecutor','MEMBER')
   AND pg_has_role(session_user,'vec_meritos_externo','MEMBER')
   AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=2
   AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
    AND m.roleid='vec_meritos_ejecutor'::regrole AND m.inherit_option
    AND NOT m.set_option AND NOT m.admin_option)
   AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
    AND m.roleid='vec_meritos_externo'::regrole AND m.inherit_option
    AND NOT m.set_option AND NOT m.admin_option))  OR (session_user='vec_externo_documentos_datos'
   AND pg_has_role(session_user,'vec_documentos_subida_datos','MEMBER')
   AND EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
    AND NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls))
   AND EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
    AND m.roleid='vec_documentos_subida_datos'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
   AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1))
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
 OR p_motivo IS NULL OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
 THEN RAISE EXCEPTION 'AUT78: registro V3 externo denegado' USING ERRCODE='42501';END IF;
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;m:=convert_from(p_motivo,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AUT78: material V3 inválido' USING ERRCODE='22023';END;
 IF vec_autorizacion.decision_contexto_actor_v3_canonica(d) IS DISTINCT FROM p_decision
 OR vec_autorizacion.motivo_contexto_actor_v3_canonico(m) IS DISTINCT FROM p_motivo
 OR vec_autorizacion.decision_meritos_externa_exacta_v1(d,m) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT78: decisión M4 divergente' USING ERRCODE='42501';END IF;
 SELECT * INTO registro FROM vec_autorizacion.registrar_decision_contexto_actor_v3_meritos_interna(
  p_decision,p_motivo,p_persona_version,p_perfil_version);
 IF NOT FOUND THEN RETURN;END IF;
 -- La fila negativa permite confirmar el asiento durable. No revalida
 -- permiso ejecutable ni produce material de capacidad para una denegación.
 IF registro.concedida IS FALSE THEN
  RETURN QUERY SELECT false,registro.codigo,registro.decision_huella_sha256,
   registro.registrada_en,NULL::timestamptz;
  RETURN;
 END IF;
 IF registro.concedida IS NOT TRUE THEN RETURN;END IF;
 viva:=vec_autorizacion.revalidar_decision_contexto_actor_v3_meritos_interna(
  p_decision,p_motivo,p_persona_version,p_perfil_version);
 IF viva IS NULL THEN RETURN;END IF;
 RETURN QUERY SELECT registro.concedida,registro.codigo,registro.decision_huella_sha256,
  registro.registrada_en,viva;
END $function$;
CREATE OR REPLACE FUNCTION vec_autorizacion.revalidar_decision_meritos_externo_v3_viva(p_decision bytea, p_motivo bytea, p_persona_version numeric, p_perfil_version numeric)
 RETURNS timestamp with time zone
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
AS $function$
DECLARE d jsonb;m jsonb;
BEGIN
 IF NOT ((session_user='vec_externo_meritos_cartera'
   AND pg_has_role(session_user,'vec_meritos_cartera_externo','MEMBER')
   AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1)
  OR (session_user='vec_externo_meritos_operacion'
   AND pg_has_role(session_user,'vec_meritos_ejecutor','MEMBER')
   AND pg_has_role(session_user,'vec_meritos_externo','MEMBER')
   AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=2
   AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
    AND m.roleid='vec_meritos_ejecutor'::regrole AND m.inherit_option
    AND NOT m.set_option AND NOT m.admin_option)
   AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
    AND m.roleid='vec_meritos_externo'::regrole AND m.inherit_option
    AND NOT m.set_option AND NOT m.admin_option))  OR (session_user='vec_externo_documentos_datos'
   AND pg_has_role(session_user,'vec_documentos_subida_datos','MEMBER')
   AND EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
    AND NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls))
   AND EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
    AND m.roleid='vec_documentos_subida_datos'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
   AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1))
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR p_decision IS NULL OR p_motivo IS NULL
 THEN RAISE EXCEPTION 'AUT78: revalidación V3 externa denegada' USING ERRCODE='42501';END IF;
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;m:=convert_from(p_motivo,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AUT78: material V3 inválido' USING ERRCODE='22023';END;
 IF vec_autorizacion.decision_contexto_actor_v3_canonica(d) IS DISTINCT FROM p_decision
 OR vec_autorizacion.motivo_contexto_actor_v3_canonico(m) IS DISTINCT FROM p_motivo
 OR vec_autorizacion.decision_meritos_externa_exacta_v1(d,m) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT78: revalidación M4 divergente' USING ERRCODE='42501';END IF;
 RETURN vec_autorizacion.revalidar_decision_contexto_actor_v3_meritos_interna(
  p_decision,p_motivo,p_persona_version,p_perfil_version);
END $function$;

-- La comprobación de la sesión activa M4 se usa tanto en el asiento como
-- en la revalidación posterior del consumo; el LOGIN de datos queda nominal.
DO $sesion_datos$
BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef(
  'vec_autorizacion.revalidar_sesion_cartera_meritos_v1(jsonb,timestamptz,timestamptz,timestamptz)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM
  '78834a038bae17e5c1c632d4840ee58bb026bebda7cb96202af68f048e203b2b'
 THEN RAISE EXCEPTION 'AUT95: sesion M4 divergente' USING ERRCODE='55000'; END IF;
END $sesion_datos$;
CREATE OR REPLACE FUNCTION vec_autorizacion.revalidar_sesion_cartera_meritos_v1(p_vinculo jsonb, p_emitida_en timestamp with time zone, p_valida_hasta timestamp with time zone, p_instante timestamp with time zone)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
AS $function$
BEGIN
 IF current_user<>'vec_autorizacion_propietario'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR jsonb_typeof(p_vinculo) IS DISTINCT FROM 'object'
 OR vec_autorizacion.vinculo_contexto_actor_v2_valido(p_vinculo) IS NOT TRUE
 OR p_vinculo->>'superficie' IS DISTINCT FROM 'externa_personal'
 OR p_vinculo->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
 OR p_vinculo->>'metodo_observado' NOT IN ('certificado','dnie')
 OR p_vinculo->>'garantia_observada' IS DISTINCT FROM 'alto'
 OR p_emitida_en IS NULL OR p_valida_hasta IS NULL OR p_instante IS NULL
 OR p_valida_hasta<=p_emitida_en OR p_instante<p_emitida_en OR p_instante>=p_valida_hasta
 OR NOT EXISTS(SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
   JOIN pg_roles g ON g.oid=m.roleid
   WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
   AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
   AND l.rolconfig IS NULL
   AND ((g.rolname='vec_meritos_cartera_externo'
     AND l.rolname='vec_externo_meritos_cartera')
    OR (g.rolname='vec_autorizacion_registro_meritos_externo'
     AND l.rolname='vec_externo_meritos_registro_autorizacion')
    OR (g.rolname='vec_meritos_ejecutor'
     AND l.rolname='vec_externo_meritos_operacion')
    OR (g.rolname='vec_documentos_subida_datos'
     AND l.rolname='vec_externo_documentos_datos'))
   AND NOT(g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
   AND g.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 OR (CASE WHEN session_user='vec_externo_meritos_operacion' THEN
      (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>2
      OR NOT EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
       AND m.roleid='vec_meritos_externo'::regrole AND m.inherit_option
       AND NOT m.set_option AND NOT m.admin_option)
      ELSE (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1 END)
 THEN RETURN false; END IF;
 IF vec_identidad_externa_v1.acreditar_sesion_externa_v1(
   p_vinculo->>'autenticacion_ref',p_vinculo->>'autenticacion_huella_sha256',
   p_vinculo->>'asercion_ref',p_vinculo->>'sesion_ref',p_vinculo->>'cuenta_ref',
   p_vinculo->>'cuenta_ordinaria_ref',(p_vinculo->>'cuenta_privilegiada')::boolean,
   p_vinculo->>'superficie',p_vinculo->>'metodo_observado',p_vinculo->>'garantia_observada',
   p_vinculo->>'politica_garantia_ref',p_vinculo->>'politica_garantia_huella_sha256',
   (p_vinculo->>'autenticacion_verificada_en')::timestamptz,
   (p_vinculo->>'sesion_emitida_en')::timestamptz,
   p_vinculo->>'control_sesion_ref',p_vinculo->>'control_sesion_revision','activa',
   p_vinculo->>'control_sesion_huella_sha256',
   (p_vinculo->>'sesion_revalidada_en')::timestamptz,
   (p_vinculo->>'sesion_valida_hasta')::timestamptz) IS NOT TRUE
 THEN RETURN false; END IF;
 RETURN p_emitida_en >= (p_vinculo->>'sesion_revalidada_en')::timestamptz
  AND p_valida_hasta <= (p_vinculo->>'sesion_valida_hasta')::timestamptz
  AND p_instante >= (p_vinculo->>'sesion_revalidada_en')::timestamptz
  AND p_instante < (p_vinculo->>'sesion_valida_hasta')::timestamptz;
EXCEPTION WHEN data_exception OR invalid_text_representation OR datetime_field_overflow THEN
 RETURN false;
END $function$;
COMMIT;
