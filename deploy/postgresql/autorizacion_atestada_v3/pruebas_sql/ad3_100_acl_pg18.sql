\set ON_ERROR_STOP on
DO $acl$
DECLARE nueva regprocedure:='vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_ambitos_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
        vieja regprocedure:='vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
        ct regprocedure:='vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_ambitos_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
        ct_vieja regprocedure:='vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF NOT has_function_privilege('vec_contratacion_temporal_propietario',nueva,'EXECUTE')
    OR has_function_privilege('vec_contratacion_temporal_propietario',vieja,'EXECUTE')
    OR has_function_privilege('vec_ct137_login',nueva,'EXECUTE')
    OR has_schema_privilege('vec_ct137_login','vec_autorizacion_atestada_v3','USAGE')
    OR NOT has_function_privilege('vec_ct137_login',ct,'EXECUTE')
    OR has_function_privilege('vec_ct137_login',ct_vieja,'EXECUTE')
 THEN RAISE EXCEPTION 'AD3-100/CT-137: ACL divergente'; END IF;
END $acl$;
SET ROLE vec_contratacion_temporal_propietario;
DO $negativos$
DECLARE m jsonb:=jsonb_build_object('operacion','listar','organizacion_ref','organizacion:desarrollo:dipgra',
 'clase_ambito','organizacion','ambito_ref','organizacion:desarrollo:dipgra',
 'expediente_ref','expediente:ct137-sintetico','version_observada',1,'consulta_huella_sha256',repeat('d',64));
 v jsonb; n integer:=0;
BEGIN
 FOREACH v IN ARRAY ARRAY[m-'organizacion_ref',m-'clase_ambito',m-'ambito_ref',
   jsonb_set(m,'{organizacion_ref}','"organizacion:ajena"'),
   jsonb_set(m,'{clase_ambito}','"centro"'),
   jsonb_set(m,'{ambito_ref}','"centro:ajeno"')] LOOP
  BEGIN
   PERFORM vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_ambitos_v3_atestada(
    v,convert_to('{}','UTF8'),convert_to('{}','UTF8'),''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea);
   RAISE EXCEPTION 'AD3-100 admitió ámbito inválido';
  EXCEPTION WHEN SQLSTATE '42501' THEN n:=n+1;
  END;
 END LOOP;
 IF n<>6 THEN RAISE EXCEPTION 'AD3-100: negativos incompletos'; END IF;
END $negativos$;
RESET ROLE;
