\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE f oid;
BEGIN
 IF to_regrole('vec_meritos_cartera_externo') IS NULL
 OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_meritos_cartera_externo'
  AND (rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 THEN RAISE EXCEPTION 'AUT78: grupo técnico inseguro' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_autorizacion.cotejar_cartera_meritos_actual_v1(jsonb,jsonb)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_autorizacion_atestada_v3_propietario',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AUT78: ACL divergente' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_autorizacion.revalidar_sesion_cartera_meritos_v1(jsonb,timestamptz,timestamptz,timestamptz)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE')
 OR NOT has_function_privilege('vec_autorizacion_atestada_v3_propietario',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AUT78: ACL de sesión divergente' USING ERRCODE='55000'; END IF;
 IF vec_autorizacion.cotejar_cartera_meritos_actual_v1('{}'::jsonb,'{}'::jsonb) IS NOT FALSE
 THEN RAISE EXCEPTION 'AUT78: concede orden vacía' USING ERRCODE='55000'; END IF;
END $test$;
ROLLBACK;
