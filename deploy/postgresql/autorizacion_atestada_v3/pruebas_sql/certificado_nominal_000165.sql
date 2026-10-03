\set ON_ERROR_STOP on
-- Puerta estructural/ACL en PostgreSQL 18 sintético tras AUT33, CA25 y AD165.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
DO $acl$
DECLARE f regprocedure:='vec_autorizacion.operar_certificado_nominal_v3(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 consumidor regprocedure:='vec_autorizacion_atestada_v3.consumir_publicacion_certificado_nominal_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF has_function_privilege('vec_contexto_actor_v1_runtime',f,'EXECUTE')
  OR has_function_privilege('vec_contexto_actor_v1_runtime',consumidor,'EXECUTE')
  OR has_function_privilege('vec_autorizacion_certificado_nominal_ejecutor',consumidor,'EXECUTE')
  OR NOT has_function_privilege('vec_autorizacion_propietario',consumidor,'EXECUTE')
  OR NOT has_function_privilege('vec_autorizacion_certificado_nominal_ejecutor',f,'EXECUTE')
  OR (SELECT count(*) FROM pg_catalog.pg_auth_members WHERE member='vec_autorizacion_certificado_nominal_ejecutor'::regrole)<>0
 THEN RAISE EXCEPTION 'AD165: ACL nominal divergente'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=consumidor
  AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f
   AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef
   AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'])
 THEN RAISE EXCEPTION 'AD165: frontera de propietario divergente'; END IF;
END $acl$;
DO $negativos$
BEGIN
 BEGIN
  PERFORM vec_autorizacion.operar_certificado_nominal_v3(NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'AD165: entrada nula aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $negativos$;
ROLLBACK;
