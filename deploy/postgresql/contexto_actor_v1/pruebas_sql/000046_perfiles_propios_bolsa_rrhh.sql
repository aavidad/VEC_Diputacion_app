\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_contexto_actor_v1.perfiles_propios_cuenta_rrhh_bolsa_v1(text)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_contexto_actor_v1_propietario'::regrole OR
  NOT p.prosecdef OR EXISTS(SELECT 1 FROM aclexplode(p.proacl) WHERE grantee=0) OR
  NOT has_function_privilege('vec_identidad_sesiones_v1_propietario',p.oid,'EXECUTE') OR
  has_function_privilege('vec_contexto_actor_v1_runtime',p.oid,'EXECUTE')
 THEN RAISE EXCEPTION 'CA46: ACL de proyección privada abierta'; END IF;
END $test$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $test$
BEGIN
 BEGIN
  PERFORM vec_contexto_actor_v1.perfiles_propios_cuenta_rrhh_bolsa_v1('cta_'||repeat('f',32));
  RAISE EXCEPTION 'CA46: aceptó cuenta sin acreditación';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
END $test$;
ROLLBACK;
