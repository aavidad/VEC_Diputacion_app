\set ON_ERROR_STOP on
BEGIN READ ONLY;
DO $prueba$
DECLARE f record;t oid:=to_regclass('vec_contratacion_temporal.prueba_selectores_recuperacion_rrhh_v1');
BEGIN
 IF t IS NULL OR NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid=t
  AND c.relowner=to_regrole('vec_contratacion_temporal_propietario')
  AND c.relrowsecurity AND c.relforcerowsecurity)
  OR has_table_privilege('vec_contratacion_temporal_consultor_rrhh',t,'SELECT,INSERT,UPDATE,DELETE')
  OR (SELECT count(*) FROM pg_trigger WHERE tgrelid=t AND NOT tgisinternal)<>2
 THEN RAISE EXCEPTION 'CT229: anexo accesible o mutable' USING ERRCODE='55000'; END IF;
 FOR f IN SELECT p.proname,p.prosrc,p.proowner,p.proacl,p.prosecdef,p.proconfig
  FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_contratacion_temporal' AND p.proname IN(
   'consultar_detalle_rrhh_con_selectores_v1',
   'consultar_original_propuesta_rrhh_con_selectores_v1',
   'consultar_detalle_rrhh_con_bolsa_y_selectores_v1') LOOP
  IF f.proowner<>to_regrole('vec_contratacion_temporal_propietario') OR NOT f.prosecdef
   OR f.proacl IS DISTINCT FROM ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
    'vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario']::aclitem[]
   OR f.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC',
    'lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[]
   OR encode(sha256(convert_to(f.prosrc,'UTF8')),'hex') IS DISTINCT FROM CASE f.proname
    WHEN 'consultar_detalle_rrhh_con_selectores_v1' THEN '813cffac9b975183ad7aa4e7e2088e1cf05e96c1497bd19d9b8e17f2fb926094'
    WHEN 'consultar_original_propuesta_rrhh_con_selectores_v1' THEN '61f9f4dd81f2d350c0143aace736013573fbf28e9a6e3c0d27a24b8c02c9cfca'
    ELSE '853d51a877900a9372a9f1ca446a38dc448a24347543f8625b482a26a3a53e0f' END
  THEN RAISE EXCEPTION 'CT229: fachada, hash o ACL divergentes (%)',f.proname USING ERRCODE='55000'; END IF;
 END LOOP;
 IF (SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_contratacion_temporal' AND p.proname IN(
   'consultar_detalle_rrhh_con_selectores_v1',
   'consultar_original_propuesta_rrhh_con_selectores_v1',
   'consultar_detalle_rrhh_con_bolsa_y_selectores_v1'))<>3
 THEN RAISE EXCEPTION 'CT229: fachadas incompletas' USING ERRCODE='55000'; END IF;
END $prueba$;
ROLLBACK;
