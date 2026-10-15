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
  AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='7b506eb45f0f1cd4e35658dff1fca1de1101b0dd5c516afc59fc20266c718393')
 OR has_function_privilege('vec_contratacion_temporal_consultor_rrhh',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AUT102: prueba de ASG o frontera EXECUTE divergente' USING ERRCODE='55000'; END IF;
END $prueba$;
ROLLBACK;
