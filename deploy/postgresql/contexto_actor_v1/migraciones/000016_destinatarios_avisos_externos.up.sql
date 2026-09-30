\set ON_ERROR_STOP on
-- CTX16: fuente nominal sobre snapshots externos aprobados por huella y CAS.
-- No consulta persona, cuenta, perfiles ni vínculos de la población interna.
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:000016',0));
DO $pre$ BEGIN
 IF to_regclass('vec_contexto_actor_v1.contexto_externo_actual') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.snapshot_contexto_externo_vigente_v1(text,text,text,timestamptz)') IS NULL
 OR to_regrole('vec_usuarios_correos_externo_propietario') IS NULL
 OR to_regrole('vec_bolsa_llamamientos_propietario') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.persona_candidato_externo_avisos_v1(text)') IS NOT NULL
 THEN RAISE EXCEPTION 'CTX16: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_contexto_actor_v1.persona_candidato_externo_avisos_v1(p_candidato text)
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE s record; persona text; candidata text; n integer:=0; ahora timestamptz:=clock_timestamp();
BEGIN
 IF p_candidato IS NULL OR p_candidato !~ '^can_[A-Za-z0-9_-]{22,128}$'
 THEN RAISE EXCEPTION 'CTX16: referencia inválida' USING ERRCODE='22023'; END IF;
 -- El lock compartido conserva la versión frente a revocación concurrente.
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:externo:mutacion:v1',0));
 FOR s IN
  SELECT v.cuenta_ref,v.perfil_ref FROM vec_contexto_actor_v1.contexto_externo_actual a
  JOIN vec_contexto_actor_v1.contexto_externo_versiones v USING(provision_ref,version)
  WHERE v.familia='candidato' AND v.snapshot#>>'{vinculo_candidato,candidato_ref}'=p_candidato
 LOOP
  SELECT x.persona_ref INTO candidata
  FROM vec_contexto_actor_v1.snapshot_contexto_externo_vigente_v1(s.cuenta_ref,s.perfil_ref,'candidato',ahora) x
  WHERE x.snapshot#>>'{vinculo_candidato,candidato_ref}'=p_candidato
   AND x.snapshot#>>'{vinculo_candidato,estado}'='activo';
  IF FOUND THEN n:=n+1; persona:=candidata; END IF;
 END LOOP;
 IF n<>1 THEN RETURN NULL; END IF;
 RETURN persona;
END $f$;

CREATE FUNCTION vec_contexto_actor_v1.es_candidato_externo_avisos_v1(p_candidato text)
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT vec_contexto_actor_v1.persona_candidato_externo_avisos_v1(p_candidato) IS NOT NULL
$f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.persona_candidato_externo_avisos_v1(text),
 vec_contexto_actor_v1.es_candidato_externo_avisos_v1(text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_bolsa_llamamientos_propietario,vec_usuarios_correos_externo_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.persona_candidato_externo_avisos_v1(text) TO vec_usuarios_correos_externo_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.es_candidato_externo_avisos_v1(text) TO vec_bolsa_llamamientos_propietario;
DO $post$ DECLARE f regprocedure; BEGIN
 FOREACH f IN ARRAY ARRAY['vec_contexto_actor_v1.persona_candidato_externo_avisos_v1(text)'::regprocedure,
  'vec_contexto_actor_v1.es_candidato_externo_avisos_v1(text)'::regprocedure] LOOP
  IF (SELECT proowner FROM pg_proc WHERE oid=f)<>'vec_contexto_actor_v1_propietario'::regrole
   OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
   OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee=0)
  THEN RAISE EXCEPTION 'CTX16: ACL incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
END $post$;
COMMIT;
