\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE f oid;p pg_proc%ROWTYPE;
BEGIN
 f:=to_regprocedure('vec_catalogos_configurables.listar_motivos_meritos_cartera_v1(text)');
 IF f IS NULL THEN RAISE EXCEPTION 'CC13: fachada ausente' USING ERRCODE='55000';END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner<>to_regrole('vec_catalogos_configurables_propietario')
 OR p.prosecdef IS NOT TRUE OR p.provolatile<>'v' OR p.proparallel<>'u'
 OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
 OR has_function_privilege('vec_meritos_cartera_externo',f,'EXECUTE')
 OR NOT has_schema_privilege('vec_autorizacion_propietario','vec_catalogos_configurables','USAGE')
 THEN RAISE EXCEPTION 'CC13: ACL o propietario divergente' USING ERRCODE='55000';END IF;
END $test$;
ROLLBACK;
