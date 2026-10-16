\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_identidad_sesiones_v1.acreditar_perfiles_propios_rrhh_bolsa_v1(text,text,text,text,numeric,text,text)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_identidad_sesiones_v1_propietario'::regrole OR
  NOT p.prosecdef OR EXISTS(SELECT 1 FROM aclexplode(p.proacl) WHERE grantee=0) OR
  NOT has_function_privilege('vec_autorizacion_propietario',p.oid,'EXECUTE') OR
  has_function_privilege('vec_identidad_sesiones_v1_revalidador',p.oid,'EXECUTE')
 THEN RAISE EXCEPTION 'IS21: ACL de identidad propia abierta'; END IF;
END $test$;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
DO $test$
BEGIN
 BEGIN
  PERFORM vec_identidad_sesiones_v1.acreditar_perfiles_propios_rrhh_bolsa_v1(
   'aut_'||repeat('f',32),'ses_'||repeat('f',32),
   'cta_'||repeat('f',32),'cse_'||repeat('f',32),1,repeat('a',64),repeat('b',64));
  RAISE EXCEPTION 'IS21: reveló perfiles sin sesión válida';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
END $test$;
ROLLBACK;
