\set ON_ERROR_STOP on
-- CA46: proyección privada de perfiles vigentes de la cuenta ya autenticada.
-- La sesión y el permiso se acreditan en IS21/AUT101; CA no concede acceso.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:000046',0));
DO $pre$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 OR
  NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) OR
  to_regrole('vec_identidad_sesiones_v1_propietario') IS NULL OR
  to_regprocedure('vec_contexto_actor_v1.perfiles_propios_cuenta_rrhh_bolsa_v1(text)') IS NOT NULL
 THEN RAISE EXCEPTION 'CA46: entorno incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_contexto_actor_v1.listar_perfiles_cuenta_v1(text,text)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_contexto_actor_v1_propietario'::regrole OR
  p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v' OR
  p.proacl::text IS DISTINCT FROM
   '{vec_contexto_actor_v1_propietario=X/vec_contexto_actor_v1_propietario}' OR
  encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM
   '178295da15209065e838527ecdc7a78750fffdafb69140994f11ae9331cba1ca'
 THEN RAISE EXCEPTION 'CA46: CA19 divergente' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_contexto_actor_v1.perfiles_propios_cuenta_rrhh_bolsa_v1(p_cuenta_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE persona text; numero_personas bigint; numero_perfiles bigint; perfiles jsonb;
BEGIN
 IF current_user<>'vec_contexto_actor_v1_propietario' OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off' OR
  coalesce(p_cuenta_ref ~ '^cta_[A-Za-z0-9_-]{22,128}$',false) IS NOT TRUE
 THEN RAISE EXCEPTION 'CA46: cuenta no disponible' USING ERRCODE='42501'; END IF;
 SELECT count(DISTINCT v.persona_ref),min(v.persona_ref)
 INTO numero_personas,persona
 FROM vec_contexto_actor_v1.vinculo_contexto_actual a
 JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v
  ON v.vinculo_ref=a.vinculo_ref AND v.version=a.version
 WHERE v.cuenta_ref=p_cuenta_ref AND v.estado='activo' AND
  v.procedencia_autoridad='autoridad_maestra_acreditada' AND
  clock_timestamp()>=v.vigente_desde AND clock_timestamp()<v.vigente_hasta;
 IF numero_personas IS DISTINCT FROM 1 OR persona IS NULL
 THEN RAISE EXCEPTION 'CA46: titularidad ausente o ambigua' USING ERRCODE='42501'; END IF;
 SELECT count(*),coalesce(jsonb_agg(jsonb_build_object(
  'perfil_ref',x.perfil_ref,'vinculo_ref',x.vinculo_ref,
  'perfil_version',x.perfil_version,'vinculo_version',x.vinculo_version)
  ORDER BY x.perfil_ref,x.vinculo_ref),'[]'::jsonb)
 INTO numero_perfiles,perfiles
 FROM vec_contexto_actor_v1.listar_perfiles_cuenta_v1(p_cuenta_ref,persona) x;
 IF numero_perfiles>32
 THEN RAISE EXCEPTION 'CA46: perfiles propios exceden límite' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('esquema','vec.contexto-actor.perfiles-propios-rrhh-bolsa.v1',
  'persona_ref',persona,'perfiles',perfiles);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.perfiles_propios_cuenta_rrhh_bolsa_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.perfiles_propios_cuenta_rrhh_bolsa_v1(text)
 TO vec_identidad_sesiones_v1_propietario;
RESET ROLE;
DO $post$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_contexto_actor_v1.perfiles_propios_cuenta_rrhh_bolsa_v1(text)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_contexto_actor_v1_propietario'::regrole OR
  p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v' OR
  p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
   'TimeZone=UTC','lock_timeout=2s','statement_timeout=10s']::text[] OR
  p.proacl::text IS DISTINCT FROM
   '{vec_contexto_actor_v1_propietario=X/vec_contexto_actor_v1_propietario,vec_identidad_sesiones_v1_propietario=X/vec_contexto_actor_v1_propietario}' OR
  encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM
   '6c92bef6b8fc37d7d6746b21f10c5184622b52f35e2fa3003c983a1ceaaa26db'
 THEN RAISE EXCEPTION 'CA46: postimagen divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
