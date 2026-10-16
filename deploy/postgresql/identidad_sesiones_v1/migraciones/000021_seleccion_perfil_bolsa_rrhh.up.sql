\set ON_ERROR_STOP on
-- IS21: identidad interna revalidada para el selector privado AUT101.
-- Ninguna lista de perfiles sale al LOGIN de aplicación.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_identidad_sesiones_v1:migracion:000021',0));
DO $pre$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 OR
  NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) OR
  to_regprocedure('vec_contexto_actor_v1.perfiles_propios_cuenta_rrhh_bolsa_v1(text)') IS NULL OR
  to_regprocedure('vec_identidad_sesiones_v1.acreditar_perfiles_propios_rrhh_bolsa_v1(text,text,text,text,numeric,text,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'IS21: entorno incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_identidad_sesiones_v1_propietario'::regrole OR
  p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v' OR
  p.proacl::text IS DISTINCT FROM
   '{vec_identidad_sesiones_v1_propietario=X/vec_identidad_sesiones_v1_propietario,vec_identidad_sesiones_v1_revalidador=X/vec_identidad_sesiones_v1_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_identidad_sesiones_v1_propietario}' OR
  encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM
   'a6359500e578352c8c103ecd4771e4eda11f9a191a952601f07161f5abdc4bd1'
 THEN RAISE EXCEPTION 'IS21: revalidador divergente' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
CREATE FUNCTION vec_identidad_sesiones_v1.acreditar_perfiles_propios_rrhh_bolsa_v1(
 p_autenticacion_ref text,p_sesion_ref text,p_cuenta_confirmada text,
 p_control_ref text,p_control_revision numeric,p_control_sha256 text,p_autenticacion_sha256 text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE s record; propios jsonb; ahora timestamptz(6):=clock_timestamp();
BEGIN
 IF current_user<>'vec_identidad_sesiones_v1_propietario' OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off' OR
  coalesce(p_cuenta_confirmada ~ '^cta_[A-Za-z0-9_-]{22,128}$',false) IS NOT TRUE OR
  coalesce(p_control_ref ~ '^cse_[A-Za-z0-9_-]{22,128}$',false) IS NOT TRUE OR
  p_control_revision IS NULL OR p_control_revision NOT BETWEEN 1 AND 18446744073709551615::numeric OR
  coalesce(p_control_sha256 ~ '^[0-9a-f]{64}$',false) IS NOT TRUE OR
  coalesce(p_autenticacion_sha256 ~ '^[0-9a-f]{64}$',false) IS NOT TRUE
 THEN RAISE EXCEPTION 'IS21: sesión propia denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO s FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
  p_autenticacion_ref,p_sesion_ref);
 IF NOT FOUND OR s.cuenta_ref IS DISTINCT FROM p_cuenta_confirmada OR
  s.control_sesion_ref IS DISTINCT FROM p_control_ref OR
  s.control_sesion_revision::numeric IS DISTINCT FROM p_control_revision OR
  s.control_sesion_huella_sha256 IS DISTINCT FROM p_control_sha256 OR
  s.autenticacion_huella_sha256 IS DISTINCT FROM p_autenticacion_sha256 OR
  s.cuenta_privilegiada IS DISTINCT FROM false OR
  s.superficie IS DISTINCT FROM 'interna_corporativa' OR
  s.metodo_observado IS DISTINCT FROM 'certificado' OR
  s.garantia_observada IS DISTINCT FROM 'alto' OR
  s.sesion_valida_hasta IS NULL OR s.sesion_valida_hasta<=ahora OR
  s.autenticacion_verificada_en IS NULL OR s.autenticacion_verificada_en>ahora
 THEN RAISE EXCEPTION 'IS21: sesión propia denegada' USING ERRCODE='42501'; END IF;
 propios:=vec_contexto_actor_v1.perfiles_propios_cuenta_rrhh_bolsa_v1(s.cuenta_ref);
 IF propios->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.perfiles-propios-rrhh-bolsa.v1' OR
  coalesce((propios->>'persona_ref') ~ '^per_[A-Za-z0-9_-]{22,128}$',false) IS NOT TRUE OR
  jsonb_typeof(propios->'perfiles') IS DISTINCT FROM 'array' OR
  jsonb_array_length(propios->'perfiles')>32
 THEN RAISE EXCEPTION 'IS21: perfiles propios inválidos' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('esquema','vec.identidad-sesiones.perfiles-propios-rrhh-bolsa.v1',
  'cuenta_ref',s.cuenta_ref,'persona_ref',propios->>'persona_ref',
  'perfiles',propios->'perfiles','autenticacion_ref',s.autenticacion_ref,
  'sesion_ref',s.sesion_ref,'control_sesion_ref',s.control_sesion_ref,
  'control_sesion_revision',s.control_sesion_revision,
  'control_sesion_huella_sha256',s.control_sesion_huella_sha256);
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.acreditar_perfiles_propios_rrhh_bolsa_v1(
 text,text,text,text,numeric,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_identidad_sesiones_v1 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.acreditar_perfiles_propios_rrhh_bolsa_v1(
 text,text,text,text,numeric,text,text) TO vec_autorizacion_propietario;
RESET ROLE;
DO $post$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_identidad_sesiones_v1.acreditar_perfiles_propios_rrhh_bolsa_v1(text,text,text,text,numeric,text,text)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_identidad_sesiones_v1_propietario'::regrole OR
  p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v' OR
  p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
   'TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[] OR
  p.proacl::text IS DISTINCT FROM
   '{vec_identidad_sesiones_v1_propietario=X/vec_identidad_sesiones_v1_propietario,vec_autorizacion_propietario=X/vec_identidad_sesiones_v1_propietario}' OR
  encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM
   '8996d57acf024630c7d3666cc16d7176a7f7126cba42fc4b72a252e1f6a3ef2f'
 THEN RAISE EXCEPTION 'IS21: postimagen divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
