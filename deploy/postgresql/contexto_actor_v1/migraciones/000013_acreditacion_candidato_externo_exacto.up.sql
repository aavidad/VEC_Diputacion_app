\set ON_ERROR_STOP on
-- ContextoActor 000013: AUT16 contrasta el ámbito candidato exacto contra
-- la provisión externa vigente. No publica perfiles ni concede al LOGIN web.
-- DOWN prohibido tras decisiones AUT16 y sus recibos.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
 'vec_contexto_actor_v1:migracion:candidato_externo_exacto:000013',0));
DO $preimagen$
DECLARE propietario oid:=pg_catalog.to_regrole('vec_contexto_actor_v1_propietario');
        aut oid:=pg_catalog.to_regrole('vec_autorizacion_propietario');
        helper oid:=pg_catalog.to_regprocedure('vec_contexto_actor_v1.provision_candidato_externo_vigente_v1(text,text,timestamptz)');
        legado oid:=pg_catalog.to_regprocedure('vec_contexto_actor_v1.perfil_candidato_externo_provisionado_v1(text,text)');
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.current_setting('transaction_isolation')<>'read committed'
    OR propietario IS NULL OR aut IS NULL OR helper IS NULL OR legado IS NULL
    OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_candidato_externo_v1(text,text,text)') IS NOT NULL
    OR pg_catalog.to_regclass('vec_contexto_actor_v1.candidato_externo_actual') IS NULL
    OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=helper) IS DISTINCT FROM propietario
    OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=legado) IS DISTINCT FROM propietario
    OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex')
          FROM pg_catalog.pg_proc WHERE oid=helper)
       IS DISTINCT FROM '60b8f1564637d9b4c511c250a4b0b793d65e882dc79c31247d207030370aec6d'
    OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex')
          FROM pg_catalog.pg_proc WHERE oid=legado)
       IS DISTINCT FROM '1bc5bb6dc74e6ac6951a4004e33207cc32d2bea0fd487a69a32b174c9b1ac810'
    OR NOT pg_catalog.has_function_privilege(aut,legado,'EXECUTE')
    OR pg_catalog.has_function_privilege('vec_contexto_actor_v1_candidato_externo',legado,'EXECUTE') THEN
  RAISE EXCEPTION 'ContextoActor 000013: preimagen incompatible' USING ERRCODE='55000';
 END IF;
END $preimagen$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path=pg_catalog;
CREATE FUNCTION vec_contexto_actor_v1.acreditar_candidato_externo_v1(
 p_persona_ref text,p_perfil_ref text,p_candidato_ref text
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog AS $funcion$
DECLARE provision record; vigente record; coincidencias integer; ahora timestamptz;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
    OR pg_catalog.current_setting('transaction_read_only')<>'off'
    OR vec_contexto_actor_v1.referencia_valida(p_persona_ref,'per_') IS NOT TRUE
    OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref,'prf_') IS NOT TRUE
    OR vec_contexto_actor_v1.referencia_valida(p_candidato_ref,'can_') IS NOT TRUE THEN
  RETURN false;
 END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
  'vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 ahora:=pg_catalog.clock_timestamp();
 SELECT pg_catalog.count(*) INTO coincidencias
 FROM vec_contexto_actor_v1.candidato_externo_actual a
 JOIN vec_contexto_actor_v1.candidato_externo_versiones v USING(provision_ref,version)
 WHERE v.persona_ref=p_persona_ref AND v.perfil_ref=p_perfil_ref
   AND v.estado='activo' AND ahora>=v.vigente_desde AND ahora<v.vigente_hasta;
 IF coincidencias<>1 THEN RETURN false; END IF;
 SELECT v.* INTO STRICT provision
 FROM vec_contexto_actor_v1.candidato_externo_actual a
 JOIN vec_contexto_actor_v1.candidato_externo_versiones v USING(provision_ref,version)
 WHERE v.persona_ref=p_persona_ref AND v.perfil_ref=p_perfil_ref
   AND v.estado='activo' AND ahora>=v.vigente_desde AND ahora<v.vigente_hasta;
 IF provision.candidato_ref IS DISTINCT FROM p_candidato_ref THEN RETURN false; END IF;
 -- El helper comprueba cuenta/perfil/persona, vínculo candidato único y
 -- generación MVCC FOR SHARE; una revocación confirmada después del snapshot
 -- y antes del bloqueo fuerza 40001 en vez de una respuesta positiva obsoleta.
 SELECT * INTO vigente FROM vec_contexto_actor_v1.provision_candidato_externo_vigente_v1(
  provision.cuenta_ref,p_perfil_ref,ahora);
 IF NOT FOUND THEN RETURN false; END IF;
 RETURN vigente.provision_ref IS NOT DISTINCT FROM provision.provision_ref
    AND vigente.version IS NOT DISTINCT FROM provision.version
    AND vigente.huella_sha256 IS NOT DISTINCT FROM provision.huella_sha256
    AND vigente.candidato_ref IS NOT DISTINCT FROM p_candidato_ref;
EXCEPTION WHEN no_data_found OR too_many_rows THEN
 RETURN false;
END $funcion$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.acreditar_candidato_externo_v1(text,text,text)
 FROM PUBLIC,vec_contexto_actor_v1_runtime,vec_contexto_actor_v1_candidato_externo;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_candidato_externo_v1(text,text,text)
 TO vec_autorizacion_propietario;
RESET ROLE;
DO $postimagen$
DECLARE f oid:='vec_contexto_actor_v1.acreditar_candidato_externo_v1(text,text,text)'::regprocedure;
        propietario oid:='vec_contexto_actor_v1_propietario'::regrole;
        aut oid:='vec_autorizacion_propietario'::regrole;
BEGIN
 IF (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT proconfig FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
    OR NOT pg_catalog.has_function_privilege(aut,f,'EXECUTE')
    OR pg_catalog.has_function_privilege('vec_contexto_actor_v1_runtime',f,'EXECUTE')
    OR pg_catalog.has_function_privilege('vec_contexto_actor_v1_candidato_externo',f,'EXECUTE')
    OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
       CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) acl
       WHERE p.oid=f AND (acl.grantee=0 OR acl.grantee NOT IN(propietario,aut)
            OR acl.privilege_type<>'EXECUTE' OR acl.is_grantable)) THEN
  RAISE EXCEPTION 'ContextoActor 000013: ACL incompatible' USING ERRCODE='55000';
 END IF;
END $postimagen$;
COMMIT;
