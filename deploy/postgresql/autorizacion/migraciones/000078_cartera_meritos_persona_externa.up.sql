\set ON_ERROR_STOP on
-- AUT78: cotejo vivo de cartera de Méritos para persona externa, también antes
-- de una candidatura. El certificado no crea una concesión ni una asignación.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000078',0));
DO $pre$ BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regrole('vec_meritos_cartera_externo') IS NOT NULL
 OR to_regprocedure('vec_autorizacion.cotejar_cartera_meritos_actual_v1(jsonb,jsonb)') IS NOT NULL
 OR to_regprocedure('vec_identidad_externa_v1.acreditar_sesion_externa_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)') IS NULL
 OR to_regclass('vec_autorizacion.asignacion_perfil_actual_externa') IS NULL
 THEN RAISE EXCEPTION 'AUT78: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_meritos_cartera_externo NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $db$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_meritos_cartera_externo',current_database()); END $db$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.cotejar_cartera_meritos_actual_v1(p_orden jsonb,p_vinculo jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
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
   WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
   AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
   AND l.rolconfig IS NULL AND g.rolname='vec_meritos_cartera_externo'
   AND NOT(g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
   AND g.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 THEN RETURN false; END IF;
 accion:=p_orden->>'accion';persona:=p_orden->>'persona_ref';
 campos:=CASE accion
  WHEN 'meritos.cartera.consultar_propia' THEN
   '["denominacion","disponibilidad","hecho_ref","retirada","tipo","total","version_hecho"]'::jsonb
  WHEN 'meritos.cartera.retirar_propio' THEN '["disponibilidad","recibo"]'::jsonb
  ELSE NULL END;
 IF campos IS NULL OR p_orden->>'modulo_id' IS DISTINCT FROM 'meritos'
 OR p_orden->>'tipo_recurso' IS DISTINCT FROM
  (CASE WHEN accion='meritos.cartera.consultar_propia' THEN 'cartera_personal' ELSE 'hecho' END)
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'gestion_cartera_propia'
 OR p_orden->>'canal' IS DISTINCT FROM 'externa_personal'
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
 OR vec_identidad_externa_v1.acreditar_sesion_externa_v1(
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
 OR (SELECT array_agg(e.value->>'accion' ORDER BY e.value->>'accion')
     FROM jsonb_array_elements(rol.documento->'concesiones') e) IS DISTINCT FROM
    ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio']::text[]
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
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.cotejar_cartera_meritos_actual_v1(jsonb,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.cotejar_cartera_meritos_actual_v1(jsonb,jsonb)
 TO vec_autorizacion_atestada_v3_propietario;
RESET ROLE;
COMMIT;
