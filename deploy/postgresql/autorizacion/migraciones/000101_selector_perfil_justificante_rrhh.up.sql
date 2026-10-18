\set ON_ERROR_STOP on
-- AUT101: perfil propio único con permiso documental actual, sin rol fijo.
-- CA46/IS21 permanecen privadas; solo esta salida llega a la aplicación.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000101',0));
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 OR
  NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) OR
  to_regprocedure('vec_identidad_sesiones_v1.acreditar_perfiles_propios_rrhh_bolsa_v1(text,text,text,text,numeric,text,text)') IS NULL OR
  to_regprocedure('vec_autorizacion_atestada_v3.registrar_preperfil_bolsa_rrhh_v1(jsonb)') IS NULL OR
  to_regprocedure('vec_autorizacion.seleccionar_perfil_justificante_rrhh_v1(text,text,text,text,numeric,text,text,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT101: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.seleccionar_perfil_justificante_rrhh_v1(
 p_autenticacion_ref text,p_sesion_ref text,p_cuenta_confirmada text,
 p_control_ref text,p_control_revision numeric,p_control_sha256 text,
 p_autenticacion_sha256 text,p_correlacion_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE identidad jsonb; candidato jsonb; a record; n integer:=0; perfil text;
 cabeza record; canon bytea; catalogo jsonb; entrada jsonb; entradas integer:=0; total_cabezas integer:=0;
 accion_vigente boolean:=false;
 evento jsonb; recibo record; ahora timestamptz(6):=clock_timestamp();
BEGIN
 IF current_user<>'vec_autorizacion_propietario' OR
  current_setting('role')<>'none' OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off' OR
  coalesce(p_correlacion_ref ~ '^correlacion_[0-9a-f]{32}$',false) IS NOT TRUE OR
  NOT EXISTS(SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
   WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit AND l.rolconfig IS NULL AND
    NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls) AND
    m.roleid='vec_autorizacion_fuente'::regrole AND m.inherit_option AND
    NOT m.set_option AND NOT m.admin_option AND
    (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1)
 THEN RAISE EXCEPTION 'AUT101: selector denegado' USING ERRCODE='42501'; END IF;
 identidad:=vec_identidad_sesiones_v1.acreditar_perfiles_propios_rrhh_bolsa_v1(
  p_autenticacion_ref,p_sesion_ref,p_cuenta_confirmada,p_control_ref,
  p_control_revision,p_control_sha256,p_autenticacion_sha256);
 IF identidad->>'esquema' IS DISTINCT FROM
  'vec.identidad-sesiones.perfiles-propios-rrhh-bolsa.v1' OR
  identidad->>'cuenta_ref' IS DISTINCT FROM p_cuenta_confirmada OR
  identidad->>'autenticacion_ref' IS DISTINCT FROM p_autenticacion_ref OR
  identidad->>'sesion_ref' IS DISTINCT FROM p_sesion_ref OR
  coalesce((identidad->>'persona_ref') ~ '^per_[A-Za-z0-9_-]{22,128}$',false) IS NOT TRUE OR
  jsonb_typeof(identidad->'perfiles') IS DISTINCT FROM 'array' OR
  jsonb_array_length(identidad->'perfiles')>32
 THEN RAISE EXCEPTION 'AUT101: identidad propia divergente' USING ERRCODE='42501'; END IF;
 FOR cabeza IN SELECT * FROM vec_autorizacion.cabeza_catalogo_acciones_admin_v1
  ORDER BY catalogo_ref FOR SHARE LOOP
  total_cabezas:=total_cabezas+1;
  IF total_cabezas>16 OR cabeza.version<1 OR cabeza.version>2147483647
  THEN RAISE EXCEPTION 'AUT101: catálogo no acotado' USING ERRCODE='42501'; END IF;
  SELECT r.catalogo_canon INTO STRICT canon FROM
   vec_autorizacion.resolver_catalogo_acciones_administracion_v2(
    cabeza.catalogo_ref,cabeza.version::integer,cabeza.huella_sha256) r;
  catalogo:=convert_from(canon,'UTF8')::jsonb;
  IF jsonb_typeof(catalogo->'entradas') IS DISTINCT FROM 'array' OR
   jsonb_array_length(catalogo->'entradas')>512
  THEN RAISE EXCEPTION 'AUT101: catálogo divergente' USING ERRCODE='42501'; END IF;
  FOR candidato IN SELECT value FROM jsonb_array_elements(catalogo->'entradas') LOOP
   IF candidato->>'referencia'='accion:bolsa.justificante_renuncia.descargar_rrhh' AND
    candidato->>'version'='2' AND
    (candidato->>'vigente_desde')::timestamptz<=ahora AND
    ahora<(candidato->>'vigente_hasta')::timestamptz THEN
    entradas:=entradas+1; entrada:=candidato;
   END IF;
  END LOOP;
 END LOOP;
 accion_vigente:=entradas=1 AND
  entrada->'dimensiones_ambito'='["ambito_ref","unidad_ref"]'::jsonb AND
  entrada->>'clase_control'='ordinario' AND
  entrada->>'fuente_ref' IS NOT NULL AND
  coalesce((entrada->>'fuente_huella_sha256') ~ '^[0-9a-f]{64}$',false) AND
  entrada#>>'{concesion,accion}'='bolsa.justificante_renuncia.descargar_rrhh' AND
  entrada#>>'{concesion,modulo_id}'='bolsa' AND
  entrada#>>'{concesion,tipo_recurso}'='justificante_renuncia' AND
  entrada#>'{concesion,finalidades}'='["custodiar_justificante_renuncia_bolsa"]'::jsonb AND
  entrada#>'{concesion,campos_permitidos}'='["contenido"]'::jsonb AND
  entrada#>>'{concesion,garantia_minima}'='alto';
 IF accion_vigente THEN
 FOR candidato IN SELECT value FROM jsonb_array_elements(identidad->'perfiles') LOOP
  IF coalesce((candidato->>'perfil_ref') ~ '^prf_[A-Za-z0-9_-]{22,128}$',false) IS NOT TRUE
  THEN RAISE EXCEPTION 'AUT101: perfil propio inválido' USING ERRCODE='42501'; END IF;
  SELECT x.perfil_activo_ref,x.principal_id,x.documento AS asignacion,
   r.documento AS rol,v.estado AS control_estado
  INTO a FROM vec_autorizacion.asignacion_perfil_actual ap
  JOIN vec_autorizacion.asignacion_perfil x ON x.perfil_activo_ref=ap.perfil_activo_ref
   AND x.asignacion_ref=ap.asignacion_ref
  JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=x.version_rol_ref
  JOIN vec_autorizacion.control_vigencia_version_rol_actual cp
   ON cp.version_rol_ref=r.version_rol_ref
  JOIN vec_autorizacion.control_vigencia_version_rol v
   ON v.version_rol_ref=cp.version_rol_ref AND v.revision=cp.revision
  WHERE x.perfil_activo_ref=candidato->>'perfil_ref' FOR SHARE OF ap,x,r,cp,v;
  IF NOT FOUND OR a.principal_id IS DISTINCT FROM identidad->>'persona_ref' OR
   a.asignacion->>'estado' IS DISTINCT FROM 'activa' OR
   a.rol->>'estado' IS DISTINCT FROM 'publicada' OR
   a.control_estado IS DISTINCT FROM 'habilitada' OR
   ahora<(a.asignacion->>'vigente_desde')::timestamptz OR
   ahora>=(a.asignacion->>'vigente_hasta')::timestamptz OR
   jsonb_typeof(a.rol->'concesiones') IS DISTINCT FROM 'array' OR
   jsonb_typeof(a.asignacion->'ambitos') IS DISTINCT FROM 'array' OR
   jsonb_array_length(a.asignacion->'ambitos')<>2 OR
   NOT (a.asignacion->'ambitos' @> '[{"clave":"unidad_ref"},{"clave":"ambito_ref"}]'::jsonb) OR
   EXISTS(SELECT 1 FROM jsonb_array_elements(a.asignacion->'ambitos') e
    WHERE e->>'clave' NOT IN ('unidad_ref','ambito_ref') OR
     jsonb_typeof(e->'valores') IS DISTINCT FROM 'array' OR
     jsonb_array_length(e->'valores')<>1 OR
     coalesce((e->'valores'->>0) ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$',false) IS NOT TRUE) OR
   (SELECT count(*) FROM jsonb_array_elements(a.rol->'concesiones') c WHERE
     c->>'accion'='bolsa.justificante_renuncia.descargar_rrhh' AND
     c->>'modulo_id'='bolsa' AND c->>'tipo_recurso'='justificante_renuncia' AND
     c->'finalidades' IS NOT DISTINCT FROM entrada#>'{concesion,finalidades}' AND
     c->'campos_permitidos' IS NOT DISTINCT FROM entrada#>'{concesion,campos_permitidos}' AND
     c->>'garantia_minima'='alto')<>1
  THEN CONTINUE; END IF;
  n:=n+1; perfil:=a.perfil_activo_ref;
 END LOOP;
 END IF;
 evento:=jsonb_build_object('evento_ref','evento_'||replace(gen_random_uuid()::text,'-',''),
  'actor_ref',identidad->>'persona_ref',
  'recurso_ref','cuenta:'||encode(sha256(convert_to(p_cuenta_confirmada,'UTF8')),'hex'),
  'resultado',CASE WHEN n=1 THEN 'permitido' ELSE 'denegado' END,
  'motivo_ref',CASE WHEN n=1 THEN 'perfil_unico' ELSE 'perfil_ausente_o_ambiguo' END,
  'correlacion_ref',p_correlacion_ref,
  'fuente_sha256',encode(sha256(convert_to(identidad::text,'UTF8')),'hex'));
 SELECT * INTO STRICT recibo FROM vec_autorizacion_atestada_v3.registrar_preperfil_bolsa_rrhh_v1(evento);
 IF coalesce(recibo.auditoria_ref,'')='' OR recibo.correlacion_ref IS DISTINCT FROM p_correlacion_ref
 THEN RAISE EXCEPTION 'AUT101: acuse común inválido' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('esquema','vec.autorizacion.perfil-justificante-rrhh.v1',
  'perfil_activo_ref',CASE WHEN n=1 THEN perfil ELSE '' END,
  'auditoria_ref',recibo.auditoria_ref);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.seleccionar_perfil_justificante_rrhh_v1(
 text,text,text,text,numeric,text,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_autorizacion_fuente;
GRANT EXECUTE ON FUNCTION vec_autorizacion.seleccionar_perfil_justificante_rrhh_v1(
 text,text,text,text,numeric,text,text,text) TO vec_autorizacion_fuente;
RESET ROLE;
DO $post$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_autorizacion.seleccionar_perfil_justificante_rrhh_v1(text,text,text,text,numeric,text,text,text)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole OR
  p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v' OR
  p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
   'TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[] OR
  p.proacl::text IS DISTINCT FROM
   '{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_fuente=X/vec_autorizacion_propietario}' OR
  encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM
   'bfbc2ad675d76811f3b0f86a1913d7e94d96ecf63d43085b8a473af8a749ffa5'
 THEN RAISE EXCEPTION 'AUT101: postimagen divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
