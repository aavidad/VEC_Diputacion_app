\set ON_ERROR_STOP on
-- Puerta estructural de AD166 en un clon desechable, tras AUT33 y AD165.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE f oid; p record; c text;
BEGIN
 f:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_publicacion_cargo_competencial_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL THEN RAISE EXCEPTION 'AD166: fachada ausente'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT p.prosecdef
  OR p.provolatile<>'v' OR p.proparallel<>'u'
  OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
  OR position('personal.cargo_competencial.publicar' IN p.prosrc)=0
  OR position('vec_personal.cargo_competencial.publicar.v1' IN p.prosrc)=0
  OR EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE a.grantee NOT IN (p.proowner,'vec_personal_propietario'::regrole))
  OR NOT has_function_privilege('vec_personal_propietario',f,'EXECUTE')
  OR has_function_privilege('vec_personal_ejecutor',f,'EXECUTE')
  OR (SELECT count(*) FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))))<>2 THEN
  RAISE EXCEPTION 'AD166: contrato o ACL divergentes'; END IF;
 SELECT pg_get_constraintdef(k.oid,true) INTO STRICT c FROM pg_constraint k
 WHERE k.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND k.conname='clave_capacidad_version_audiencia_consumo_check' AND k.convalidated;
 IF position('vec_personal.cargo_competencial.publicar.v1' IN c)=0
  OR position('vec_contexto_actor.certificado_nominal.publicar.v1' IN c)=0 THEN
  RAISE EXCEPTION 'AD166: audiencia ausente'; END IF;
END $test$;
ROLLBACK;
