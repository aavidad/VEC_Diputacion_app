\set ON_ERROR_STOP on
-- CA34. Metadatos opacos para AUT43; no nombres, cuentas ni otra Persona.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:000034',0));
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN
  RAISE EXCEPTION 'CA34: PARO clave=operador actual=incompatible esperado=superusuario_PG18' USING ERRCODE='55000';END IF;
 IF to_regclass('vec_contexto_actor_v1.persona_actual') IS NULL
 OR to_regclass('vec_contexto_actor_v1.perfil_actual') IS NULL
 OR to_regrole('vec_autorizacion_propietario') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.metadatos_persona_administrable_v1(text)') IS NOT NULL
 OR to_regprocedure('vec_contexto_actor_v1.metadatos_perfil_administrable_v1(text,text)') IS NOT NULL THEN
  RAISE EXCEPTION 'CA34: PARO clave=preimagen actual=incompatible esperado=CA1_AUT_sin_CA34' USING ERRCODE='55000';END IF;
END $pre$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
-- Puerto propietario 0/1: ausencia comprobada devuelve NULL; fallo propaga error.
-- Persona sólo vigente; perfiles con historia de caducidad/revocación no conceden uso.
CREATE FUNCTION vec_contexto_actor_v1.metadatos_persona_administrable_v1(p_persona text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET row_security=on AS $f$
DECLARE x record;
BEGIN
 IF vec_contexto_actor_v1.referencia_valida(p_persona,'per_') IS NOT TRUE
 OR octet_length(p_persona)>128 THEN RAISE EXCEPTION 'CA34: referencia_invalida' USING ERRCODE='22023';END IF;
 SELECT v.persona_ref,v.version,v.estado,v.vigente_desde,v.vigente_hasta INTO x FROM vec_contexto_actor_v1.persona_actual a
 JOIN vec_contexto_actor_v1.persona_versiones v USING(persona_ref,version)
 WHERE a.persona_ref=p_persona FOR SHARE OF a;
 IF NOT FOUND THEN RETURN NULL;END IF;
 IF x.estado<>'activo' OR clock_timestamp()<x.vigente_desde OR clock_timestamp()>=x.vigente_hasta THEN RETURN NULL;END IF;
 IF x.version NOT BETWEEN 1 AND 9007199254740991 THEN RAISE EXCEPTION 'CA34: metadata_no_disponible' USING ERRCODE='55000';END IF;
 RETURN jsonb_build_object('persona_ref',x.persona_ref,'version',x.version);
END $f$;
CREATE FUNCTION vec_contexto_actor_v1.metadatos_perfil_administrable_v1(p_persona text,p_perfil text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET row_security=on AS $f$
DECLARE x record;
BEGIN
 IF vec_contexto_actor_v1.referencia_valida(p_persona,'per_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(p_perfil,'prf_') IS NOT TRUE
 OR octet_length(p_persona)>128 OR octet_length(p_perfil)>128 THEN
  RAISE EXCEPTION 'CA34: referencia_invalida' USING ERRCODE='22023';END IF;
 SELECT v.perfil_ref,v.version,v.estado,v.vigente_desde,v.vigente_hasta INTO x
 FROM vec_contexto_actor_v1.perfil_actual a JOIN vec_contexto_actor_v1.perfil_versiones v USING(perfil_ref,version)
 WHERE a.perfil_ref=p_perfil AND v.persona_ref=p_persona FOR SHARE OF a;
 IF NOT FOUND THEN RETURN NULL;END IF;
 IF x.version NOT BETWEEN 1 AND 9007199254740991 THEN RAISE EXCEPTION 'CA34: metadata_no_disponible' USING ERRCODE='55000';END IF;
 RETURN jsonb_build_object('perfil_ref',x.perfil_ref,'version',x.version,'estado',x.estado,
  'vigente_desde',x.vigente_desde,'vigente_hasta',x.vigente_hasta);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.metadatos_persona_administrable_v1(text),vec_contexto_actor_v1.metadatos_perfil_administrable_v1(text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.metadatos_persona_administrable_v1(text),vec_contexto_actor_v1.metadatos_perfil_administrable_v1(text,text) TO vec_autorizacion_propietario;
DO $acl$
DECLARE f oid;
BEGIN
 FOR f IN SELECT p.oid FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_contexto_actor_v1' AND p.proname IN('metadatos_persona_administrable_v1','metadatos_perfil_administrable_v1') LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_contexto_actor_v1_propietario'::regrole AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog','TimeZone=UTC','row_security=on'])
  OR NOT has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
  OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,'vec_autorizacion_propietario'::regrole) OR a.privilege_type<>'EXECUTE' OR a.is_grantable)) THEN
   RAISE EXCEPTION 'CA34: PARO clave=ACL actual=incompatible esperado=owners_CA_AUT_EXECUTE' USING ERRCODE='55000';END IF;
 END LOOP;
END $acl$;
COMMIT;
