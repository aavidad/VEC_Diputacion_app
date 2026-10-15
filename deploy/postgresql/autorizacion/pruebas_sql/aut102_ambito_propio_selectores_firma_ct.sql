\set ON_ERROR_STOP on
BEGIN READ ONLY;
DO $prueba$
DECLARE f oid:=to_regprocedure('vec_autorizacion.comprobar_ambito_propio_selectores_firma_ct_v1(text,text,text,text,text,text,text)');
BEGIN
 IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
  AND p.proowner=to_regrole('vec_autorizacion_propietario') AND p.prosecdef
  AND p.provolatile='v' AND p.proparallel='u'
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
  AND p.proacl=ARRAY['vec_autorizacion_propietario=X/vec_autorizacion_propietario',
   'vec_contratacion_temporal_propietario=X/vec_autorizacion_propietario']::aclitem[]
  AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='8d39eeaae8ad82700397125e1044acb45639303bdfd1fe2c025a7e443b7a3b10')
 OR has_function_privilege('vec_contratacion_temporal_consultor_rrhh',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AUT102: prueba de ASG o frontera EXECUTE divergente' USING ERRCODE='55000'; END IF;
END $prueba$;
ROLLBACK;
